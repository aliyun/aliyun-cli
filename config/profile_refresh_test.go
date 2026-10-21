package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cloudsso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialRefreshPersistsToSelectedConfig(t *testing.T) {
	for _, mode := range []AuthenticateMode{OAuth, CloudSSO} {
		for _, pathType := range []string{"default", "custom"} {
			t.Run(string(mode)+"/"+pathType, func(t *testing.T) {
				originalHome, originalLoad, originalSave := hookGetHomePath, hookLoadOrCreateConfiguration, saveConfigurationFunc
				originalExchange, originalCloudRefresh := exchangeFromOAuthFunc, tryRefreshStsTokenFunc
				originalOAuthRefresh, originalHTTPClient := oauthTryRefreshOauthToken, utilNewHttpClient
				originalOAuthURLs, originalTime := oauthBaseUrlMap, utilGetCurrentUnixTime
				t.Cleanup(func() {
					hookGetHomePath, hookLoadOrCreateConfiguration, saveConfigurationFunc = originalHome, originalLoad, originalSave
					exchangeFromOAuthFunc, tryRefreshStsTokenFunc = originalExchange, originalCloudRefresh
					oauthTryRefreshOauthToken, utilNewHttpClient = originalOAuthRefresh, originalHTTPClient
					oauthBaseUrlMap, utilGetCurrentUnixTime = originalOAuthURLs, originalTime
				})
				home := t.TempDir()
				hookGetHomePath = func(func() string) func() string { return func() string { return home } }
				hookLoadOrCreateConfiguration = func(fn func(string) (*Configuration, error)) func(string) (*Configuration, error) { return fn }
				saveConfigurationFunc = SaveConfigurationWithContext
				exchangeFromOAuthFunc = exchangeFromOAuthWithCheckpoint
				tryRefreshStsTokenFunc = cloudsso.TryRefreshStsToken
				oauthTryRefreshOauthToken = tryRefreshOauthToken
				utilGetCurrentUnixTime = func() int64 { return time.Now().Unix() }
				t.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "")
				t.Setenv("ALIBABACLOUD_IGNORE_PROFILE", "")
				t.Setenv("HTTPS_PROXY", "")
				t.Setenv("https_proxy", "")
				refreshCalls, exchangeCalls := 0, 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/v1/token":
						assert.NoError(t, r.ParseForm())
						assert.Equal(t, fmt.Sprintf("refresh-%d", refreshCalls), r.Form.Get("refresh_token"))
						refreshCalls++
						fmt.Fprintf(w, `{"access_token":"access-%d","refresh_token":"refresh-%d","expires_in":3600}`, refreshCalls, refreshCalls)
					case "/v1/exchange":
						exchangeCalls++
						assert.Equal(t, fmt.Sprintf("Bearer access-%d", exchangeCalls), r.Header.Get("Authorization"))
						fmt.Fprintf(w, `{"accessKeyId":"key-%d","accessKeySecret":"secret-%d","securityToken":"token-%d","expiration":%q}`,
							exchangeCalls, exchangeCalls, exchangeCalls, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
					case "/cloud-credentials":
						refreshCalls++
						assert.Equal(t, "Bearer cloud-access", r.Header.Get("Authorization"))
						fmt.Fprintf(w, `{"CloudCredential":{"AccessKeyId":"key-%d","AccessKeySecret":"secret-%d","SecurityToken":"token-%d","Expiration":%q}}`,
							refreshCalls, refreshCalls, refreshCalls, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
					default:
						t.Errorf("unexpected credential path %s", r.URL.Path)
						http.Error(w, "unexpected path", http.StatusBadRequest)
					}
				}))
				defer server.Close()
				utilNewHttpClient = server.Client
				oauthBaseUrlMap = map[string]string{"CN": server.URL}
				ctx := newCtx()
				defaultPath := filepath.Join(GetConfigPath(), configFile)
				selectedPath := defaultPath
				var defaultBytes []byte
				if pathType == "custom" {
					selectedPath = filepath.Join(t.TempDir(), "selected-config.json")
					ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
					ConfigurePathFlag(ctx.Flags()).SetValue(selectedPath)
					defaultProfile := Profile{
						Name: "shared-name", Mode: AK, RegionId: "cn-hangzhou", AccessKeyId: "default-key", AccessKeySecret: "default-secret",
					}
					require.NoError(t, SaveConfigurationWithContext(newCtx(), &Configuration{CurrentProfile: "shared-name", Profiles: []Profile{defaultProfile}}))
					var err error
					defaultBytes, err = os.ReadFile(defaultPath)
					require.NoError(t, err)
				}
				ProfileFlag(ctx.Flags()).SetAssigned(true)
				ProfileFlag(ctx.Flags()).SetValue("shared-name")
				selectedProfile := Profile{
					Name: "shared-name", Mode: mode, RegionId: "cn-hangzhou", Endpoint: "stored-endpoint",
					AccessKeyId: "expired-key", AccessKeySecret: "expired-secret", StsToken: "expired-token",
					StsExpiration: time.Now().Add(-time.Hour).Unix(),
					OAuthSiteType: "CN", OAuthAccessToken: "expired-access", OAuthRefreshToken: "refresh-0",
					OAuthAccessTokenExpire: time.Now().Add(-time.Hour).Unix(),
					CloudSSOSignInUrl:      server.URL, CloudSSOAccountId: "account", CloudSSOAccessConfig: "access-config",
					CloudSSOAccessTokenExpire: time.Now().Add(time.Hour).Unix(), AccessToken: "cloud-access",
				}
				selectedConfiguration := &Configuration{CurrentProfile: "shared-name", Profiles: []Profile{selectedProfile}, MetaPath: "stored-metadata"}
				require.NoError(t, SaveConfigurationWithContext(ctx, selectedConfiguration))
				for call := 1; call <= 2; call++ {
					profile, err := LoadProfileWithContext(ctx)
					require.NoError(t, err)
					profile.Endpoint = "one-off-endpoint"
					profile.RegionId = "cn-beijing"
					credential, err := profile.GetCredential(ctx, nil)
					require.NoError(t, err)
					model, err := credential.GetCredential()
					require.NoError(t, err)
					require.Equal(t, fmt.Sprintf("key-%d", call), tea.StringValue(model.AccessKeyId))
					saved, err := LoadConfigurationFromFile(selectedPath)
					require.NoError(t, err)
					require.Equal(t, "stored-metadata", saved.MetaPath)
					stored, ok := saved.GetProfile("shared-name")
					require.True(t, ok)
					require.Equal(t, fmt.Sprintf("key-%d", call), stored.AccessKeyId)
					require.Equal(t, fmt.Sprintf("token-%d", call), stored.StsToken)
					require.Greater(t, stored.StsExpiration, time.Now().Unix())
					require.Equal(t, "stored-endpoint", stored.Endpoint)
					require.Equal(t, "cn-hangzhou", stored.RegionId)
					require.Equal(t, mode, stored.Mode)
					if mode == OAuth {
						require.Equal(t, fmt.Sprintf("refresh-%d", call), stored.OAuthRefreshToken)
						require.Equal(t, fmt.Sprintf("access-%d", call), stored.OAuthAccessToken)
						require.Greater(t, stored.OAuthAccessTokenExpire, time.Now().Unix())
					}
					if pathType == "custom" {
						unchangedDefault, err := os.ReadFile(defaultPath)
						require.NoError(t, err)
						require.Equal(t, defaultBytes, unchangedDefault)
					}
					if call == 1 {
						stored.StsExpiration = time.Now().Add(-time.Hour).Unix()
						stored.OAuthAccessTokenExpire = time.Now().Add(-time.Hour).Unix()
						saved.PutProfile(stored)
						require.NoError(t, SaveConfigurationWithContext(ctx, saved))
					}
				}
				require.Equal(t, 2, refreshCalls)
				if mode == OAuth {
					require.Equal(t, 2, exchangeCalls)
				}
			})
		}
	}
}
