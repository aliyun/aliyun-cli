package config

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Each helper reads the expired profile BEFORE the parent releases the barrier.
// This reproduces stale readers without relying on scheduler timing or sleeps.
func TestOAuthRefreshProcess(t *testing.T) {
	if os.Getenv("CLI_OAUTH_TEST_CHILD") != "1" {
		return
	}
	ctx := newCtx()
	ConfigurePathFlag(ctx.Flags()).SetValue(os.Getenv("CLI_OAUTH_TEST_CONFIG"))
	ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	name := os.Getenv("CLI_OAUTH_TEST_PROFILE")
	if name == "" {
		name = "shared"
	}
	ProfileFlag(ctx.Flags()).SetValue(name)
	ProfileFlag(ctx.Flags()).SetAssigned(true)
	oauthBaseUrlMap = map[string]string{"CN": os.Getenv("CLI_OAUTH_TEST_SERVER")}
	utilNewHttpClient = func() *http.Client { return &http.Client{Timeout: 5 * time.Second} }
	p, err := LoadProfileWithContext(ctx)
	require.NoError(t, err)
	fmt.Println("READY")
	_, err = bufio.NewReader(os.Stdin).ReadString('\n')
	require.NoError(t, err)
	cred, err := p.GetCredential(ctx, nil)
	require.NoError(t, err)
	model, err := cred.GetCredential()
	require.NoError(t, err)
	require.Equal(t, "renewed-key", *model.AccessKeyId)
	require.Equal(t, "renewed-token", *model.SecurityToken)
}

func TestOAuthRefreshMultipleProcesses(t *testing.T) {
	for _, profiles := range []int{1, 2} {
		for _, nearExpiry := range []bool{false, true} {
			t.Run(fmt.Sprintf("profiles=%d/nearExpiry=%t", profiles, nearExpiry), func(t *testing.T) { testOAuthRefreshMultipleProcesses(t, profiles, nearExpiry) })
		}
	}
}

func testOAuthRefreshMultipleProcesses(t *testing.T, profiles int, nearExpiry bool) {
	var mu sync.Mutex
	refreshes, exchanges := 0, 0
	used := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/token":
			_ = r.ParseForm()
			token := r.Form.Get("refresh_token")
			if used[token] || (token != "refresh-old" && token != "refresh-other") {
				w.WriteHeader(400)
				fmt.Fprint(w, `{"error":"invalid_grant","error_description":"refresh token already consumed"}`)
				return
			}
			refreshes++
			used[token] = true
			fmt.Fprint(w, `{"access_token":"access-new","refresh_token":"refresh-new","expires_in":3600}`)
		case "/v1/exchange":
			exchanges++
			if r.Header.Get("Authorization") != "Bearer access-new" {
				w.WriteHeader(401)
				return
			}
			fmt.Fprintf(w, `{"accessKeyId":"renewed-key","accessKeySecret":"renewed-secret","securityToken":"renewed-token","expiration":%q}`, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "selected.json")
	ctx := newCtx()
	ConfigurePathFlag(ctx.Flags()).SetValue(path)
	ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	p := Profile{Name: "shared", Mode: OAuth, RegionId: "cn-beijing", Endpoint: "stored-endpoint", OAuthSiteType: "CN", OAuthAccessToken: "access-old", OAuthRefreshToken: "refresh-old", OAuthAccessTokenExpire: 1, StsExpiration: 1}
	if nearExpiry {
		p.StsExpiration = time.Now().Unix() + 30
		p.OAuthAccessTokenExpire = time.Now().Unix() + 30
	}
	require.NoError(t, SaveConfigurationWithContext(ctx, &Configuration{CurrentProfile: "shared", Profiles: []Profile{p, {Name: "unrelated", Mode: AK, AccessKeyId: "untouched"}}}))
	alias := filepath.Join(dir, "alias.json")
	hasAlias := os.Symlink(path, alias) == nil
	if profiles == 2 {
		conf, err := LoadConfigurationFromFile(path)
		require.NoError(t, err)
		other := p
		other.Name = "other"
		other.OAuthRefreshToken = "refresh-other"
		conf.PutProfile(other)
		require.NoError(t, SaveConfigurationWithContext(ctx, conf))
	}
	type child struct {
		cmd    *exec.Cmd
		input  io.WriteCloser
		output *bufio.Reader
		stderr bytes.Buffer
	}
	children := make([]*child, 6)
	for i := range children {
		c := &child{cmd: exec.Command(os.Args[0], "-test.run=^TestOAuthRefreshProcess$", "-test.timeout=20s")}
		c.cmd.Env = append(os.Environ(), "CLI_OAUTH_TEST_CHILD=1", "CLI_OAUTH_TEST_CONFIG="+path, "CLI_OAUTH_TEST_SERVER="+server.URL, "HOME="+dir, "ALIBABA_CLOUD_IGNORE_PROFILE=", "ALIBABACLOUD_IGNORE_PROFILE=")
		if hasAlias && i%2 == 1 {
			c.cmd.Env = append(c.cmd.Env, "CLI_OAUTH_TEST_CONFIG="+alias)
		}
		if profiles == 2 && i%2 == 1 {
			c.cmd.Env = append(c.cmd.Env, "CLI_OAUTH_TEST_PROFILE=other")
		}
		var err error
		c.input, err = c.cmd.StdinPipe()
		require.NoError(t, err)
		out, err := c.cmd.StdoutPipe()
		require.NoError(t, err)
		c.output = bufio.NewReader(out)
		c.cmd.Stderr = &c.stderr
		require.NoError(t, c.cmd.Start())
		t.Cleanup(func() { _ = c.cmd.Process.Kill() })
		children[i] = c
	}
	for _, c := range children {
		line, err := c.output.ReadString('\n')
		require.NoError(t, err)
		require.Equal(t, "READY\n", line)
	}
	for _, c := range children {
		_, err := io.WriteString(c.input, "go\n")
		require.NoError(t, err)
		_ = c.input.Close()
	}
	for _, c := range children {
		out, _ := io.ReadAll(c.output)
		err := c.cmd.Wait()
		if err != nil {
			t.Errorf("child failed: %v\n%s%s", err, out, c.stderr.String())
		}
	}
	mu.Lock()
	require.Equal(t, profiles, refreshes)
	require.Equal(t, profiles, exchanges)
	mu.Unlock()
	conf, err := LoadConfigurationFromFile(path)
	require.NoError(t, err)
	stored, ok := conf.GetProfile("shared")
	require.True(t, ok)
	require.Equal(t, "refresh-new", stored.OAuthRefreshToken)
	require.Equal(t, "stored-endpoint", stored.Endpoint)
	require.Equal(t, "untouched", conf.Profiles[1].AccessKeyId)
	if profiles == 2 {
		other, ok := conf.GetProfile("other")
		require.True(t, ok)
		require.Equal(t, "renewed-key", other.AccessKeyId)
		require.Equal(t, "refresh-new", other.OAuthRefreshToken)
	}
	if hasAlias {
		info, err := os.Lstat(alias)
		require.NoError(t, err)
		require.NotZero(t, info.Mode()&os.ModeSymlink)
	}
	// A fresh process must use the persisted generation without a second refresh.
	cmd := exec.Command(os.Args[0], "-test.run=^TestOAuthRefreshProcess$", "-test.timeout=20s")
	cmd.Env = children[0].cmd.Env
	cmd.Stdin = bytes.NewBufferString("go\n")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, profiles, refreshes)
	require.Equal(t, profiles, exchanges)
}
