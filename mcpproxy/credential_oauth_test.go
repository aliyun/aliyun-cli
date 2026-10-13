package mcpproxy

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/require"
)

func TestOAuthProfileExchangesSTSCredentialsForMCPToken(t *testing.T) {
	for _, tc := range []struct {
		region             RegionType
		oauthHost, mcpHost string
	}{
		{RegionCN, "oauth.aliyun.com", "ramoauth.aliyuncs.com"},
		{RegionINTL, "oauth.alibabacloud.com", "ramoauth.alibabacloudcs.com"},
	} {
		t.Run(string(tc.region), func(t *testing.T) {
			ctx := newMCPCredentialTestContext(t)
			profile := mcpTestAKProfile("selected", "expired-key")
			profile.Mode = config.OAuth
			profile.OAuthSiteType = string(tc.region)
			profile.OAuthAccessToken = "cli-oauth-token"
			profile.OAuthAccessTokenExpire = time.Now().Add(time.Hour).Unix()
			writeMCPCredentialTestConfig(t, ctx, profile)
			setMCPCommandTestFlag(ctx, "profile", profile.Name)
			unrelatedPath := filepath.Join(t.TempDir(), "unrelated-config.json")
			setMCPCommandTestFlag(ctx, "config-path", unrelatedPath)

			oldTransport := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = oldTransport })
			var requests []string
			tokenCalls := 0
			http.DefaultTransport = credentialRoundTripper(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "POST", req.Method)
				switch req.URL.Host {
				case tc.oauthHost:
					requests = append(requests, "exchange-sts")
					require.Equal(t, "/v1/exchange", req.URL.Path)
					require.Equal(t, "Bearer cli-oauth-token", req.Header.Get("Authorization"))
					return credentialResponse(200, fmt.Sprintf(`{"accessKeyId":"oauth-sts-key","accessKeySecret":"oauth-sts-secret","securityToken":"oauth-sts-token","expiration":%q}`,
						time.Now().Add(time.Hour).UTC().Format(time.RFC3339))), nil
				case tc.mcpHost:
					requests = append(requests, "generate-mcp-token")
					tokenCalls++
					query := req.URL.Query()
					require.Equal(t, "GenerateAccessToken", query.Get("Action"))
					require.Equal(t, "oauth-sts-key", query.Get("AccessKeyId"))
					require.Equal(t, "oauth-sts-token", query.Get("SecurityToken"))
					require.NotEmpty(t, query.Get("Signature"))
					require.NotContains(t, req.URL.String(), "cli-oauth-token")
					return credentialResponse(200, fmt.Sprintf(`{"Data":{"AccessToken":"mcp-token-%d","ExpiresIn":3600}}`, tokenCalls)), nil
				default:
					t.Fatalf("unexpected credential request to %s", req.URL.Host)
					return nil, fmt.Errorf("unexpected credential endpoint")
				}
			})

			proxyConfig := ProxyConfig{RegionType: tc.region}
			require.NoError(t, configureMCPAuthentication(ctx, &proxyConfig))
			token, err := proxyConfig.CredentialAuth.GetToken()
			require.NoError(t, err)
			require.Equal(t, "mcp-token-1", token)
			require.Equal(t, []string{"exchange-sts", "generate-mcp-token"}, requests)

			saved, err := loadCredentialProfile(ctx)
			require.NoError(t, err)
			require.Equal(t, config.OAuth, saved.Mode)
			require.Equal(t, "oauth-sts-key", saved.AccessKeyId)
			require.Equal(t, "oauth-sts-token", saved.StsToken)
			_, err = os.Stat(unrelatedPath)
			require.True(t, os.IsNotExist(err))
			token, err = proxyConfig.CredentialAuth.RefreshToken(token)
			require.NoError(t, err)
			require.Equal(t, "mcp-token-2", token)
			require.Equal(t, []string{"exchange-sts", "generate-mcp-token", "generate-mcp-token"}, requests)
		})
	}
}
