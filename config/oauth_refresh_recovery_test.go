package config

import (
	"bufio"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestOAuthRotationSurvivesExchangeFailure(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		t.Run(fmt.Sprintf("malformed=%t", malformed), func(t *testing.T) { testOAuthRotationSurvivesExchangeFailure(t, malformed) })
	}
}
func testOAuthRotationSurvivesExchangeFailure(t *testing.T, malformed bool) {
	oldLoad, oldSave, oldExchange, oldRefresh := hookLoadOrCreateConfiguration, saveConfigurationFunc, exchangeFromOAuthFunc, oauthTryRefreshOauthToken
	t.Cleanup(func() {
		hookLoadOrCreateConfiguration, saveConfigurationFunc, exchangeFromOAuthFunc, oauthTryRefreshOauthToken = oldLoad, oldSave, oldExchange, oldRefresh
	})
	hookLoadOrCreateConfiguration = func(fn func(string) (*Configuration, error)) func(string) (*Configuration, error) { return fn }
	saveConfigurationFunc = SaveConfigurationWithContext
	exchangeFromOAuthFunc = exchangeFromOAuthWithCheckpoint
	oauthTryRefreshOauthToken = tryRefreshOauthToken
	oldURLs, oldClient := oauthBaseUrlMap, utilNewHttpClient
	t.Cleanup(func() { oauthBaseUrlMap, utilNewHttpClient = oldURLs, oldClient })
	refreshes, exchanges := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/token":
			refreshes++
			fmt.Fprint(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`)
		case "/v1/exchange":
			exchanges++
			if exchanges == 1 {
				if malformed {
					fmt.Fprint(w, `{}`)
					return
				}
				w.WriteHeader(503)
				fmt.Fprint(w, `{"message":"temporary failure"}`)
				return
			}
			fmt.Fprintf(w, `{"accessKeyId":"key","accessKeySecret":"secret","securityToken":"token","expiration":%q}`, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
		}
	}))
	defer server.Close()
	oauthBaseUrlMap = map[string]string{"CN": server.URL}
	utilNewHttpClient = server.Client
	ctx := newCtx()
	path := filepath.Join(t.TempDir(), "config.json")
	ConfigurePathFlag(ctx.Flags()).SetValue(path)
	ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	p := Profile{Name: "test", Mode: OAuth, OAuthSiteType: "CN", OAuthRefreshToken: "old-refresh", OAuthAccessTokenExpire: 1}
	require.NoError(t, SaveConfigurationWithContext(ctx, &Configuration{Profiles: []Profile{p}}))
	_, err := p.GetCredential(ctx, nil)
	require.Error(t, err)
	fresh, err := LoadProfile(path, "test")
	require.NoError(t, err)
	require.Equal(t, "new-refresh", fresh.OAuthRefreshToken)
	require.Equal(t, "new-access", fresh.OAuthAccessToken)
	_, err = fresh.GetCredential(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 1, refreshes)
	require.Equal(t, 2, exchanges)
}

func TestOAuthLockOwnerProcess(t *testing.T) {
	path := os.Getenv("CLI_OAUTH_TEST_LOCK")
	if path == "" {
		return
	}
	unlock, err := lockOAuthConfiguration(path)
	require.NoError(t, err)
	defer unlock()
	fmt.Println("LOCKED")
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestOAuthLockReleasedAfterProcessDeath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cmd := exec.Command(os.Args[0], "-test.run=^TestOAuthLockOwnerProcess$", "-test.timeout=15s")
	cmd.Env = append(os.Environ(), "CLI_OAUTH_TEST_LOCK="+path)
	in, err := cmd.StdinPipe()
	require.NoError(t, err)
	defer in.Close()
	out, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	line, err := bufio.NewReader(out).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "LOCKED\n", line)
	f, err := os.OpenFile(path+".oauth.lock", os.O_RDWR, 0600)
	require.NoError(t, err)
	acquired, err := tryOAuthFileLock(f)
	require.NoError(t, err)
	require.False(t, acquired)
	f.Close()
	require.NoError(t, cmd.Process.Kill())
	_ = cmd.Wait()
	unlock, err := lockOAuthConfiguration(path)
	require.NoError(t, err)
	unlock()
	_, err = os.Stat(path + ".oauth.lock")
	require.NoError(t, err)
}
