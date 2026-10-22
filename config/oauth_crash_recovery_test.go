package config

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOAuthRotationSurvivesProcessDeathDuringSTSExchange(t *testing.T) {
	var refreshes, exchanges atomic.Int32
	exchanging := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/token":
			if refreshes.Add(1) != 1 {
				w.WriteHeader(400)
				fmt.Fprint(w, `{"error":"invalid_grant"}`)
				return
			}
			fmt.Fprint(w, `{"access_token":"access-new","refresh_token":"refresh-new","expires_in":3600}`)
		case "/v1/exchange":
			if exchanges.Add(1) == 1 {
				close(exchanging)
				<-r.Context().Done()
				return
			}
			if r.Header.Get("Authorization") != "Bearer access-new" {
				w.WriteHeader(401)
				return
			}
			fmt.Fprintf(w, `{"accessKeyId":"renewed-key","accessKeySecret":"renewed-secret","securityToken":"renewed-token","expiration":%q}`, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	ctx := newCtx()
	ConfigurePathFlag(ctx.Flags()).SetValue(path)
	ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	p := Profile{Name: "shared", Mode: OAuth, RegionId: "cn-beijing", OAuthSiteType: "CN", OAuthRefreshToken: "refresh-old", OAuthAccessTokenExpire: 1, StsExpiration: 1}
	require.NoError(t, SaveConfigurationWithContext(ctx, &Configuration{CurrentProfile: "shared", Profiles: []Profile{p}}))
	env := append(os.Environ(), "CLI_OAUTH_TEST_CHILD=1", "CLI_OAUTH_TEST_CONFIG="+path, "CLI_OAUTH_TEST_SERVER="+server.URL, "HOME="+dir, "ALIBABA_CLOUD_IGNORE_PROFILE=", "ALIBABACLOUD_IGNORE_PROFILE=")
	child := exec.Command(os.Args[0], "-test.run=^TestOAuthRefreshProcess$", "-test.timeout=20s")
	child.Env = env
	child.Stdin = bytes.NewBufferString("go\n")
	stdout, e := child.StdoutPipe()
	require.NoError(t, e)
	var stderr bytes.Buffer
	child.Stderr = &stderr
	require.NoError(t, child.Start())
	// Cleanup before the server can wait on an abandoned handler on assertion failure.
	defer func() { _ = child.Process.Kill() }()
	line, e := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, e)
	require.Equal(t, "READY\n", line)
	select {
	case <-exchanging:
	case <-time.After(10 * time.Second):
		t.Fatal("STS exchange never started")
	}
	savedConfig, readErr := LoadConfigurationFromFile(path)
	require.NoError(t, child.Process.Kill())
	_ = child.Wait()
	require.NoError(t, readErr)
	saved, ok := savedConfig.GetProfile("shared")
	require.True(t, ok)
	require.Equal(t, "refresh-new", saved.OAuthRefreshToken, "rotation must be durable before STS request")
	restart := exec.Command(os.Args[0], "-test.run=^TestOAuthRefreshProcess$", "-test.timeout=20s")
	restart.Env = env
	restart.Stdin = bytes.NewBufferString("go\n")
	out, e := restart.CombinedOutput()
	require.NoError(t, e, string(out))
	require.Equal(t, int32(1), refreshes.Load())
	require.Equal(t, int32(2), exchanges.Load())
}
