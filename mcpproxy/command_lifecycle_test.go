package mcpproxy

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMCPCommandListenFailure(t *testing.T) {
	if oauthSDKSubprocess(t) {
		return
	}
	ctx, _ := testCLIProfile(t)
	installAPIHandler(t, func(*http.Request) (*http.Response, error) {
		return sdkJSONResponse(`{"apiMcpServers":[{"id":"test","name":"test","urls":{"mcp":"https://upstream.invalid/mcp/test"}}]}`), nil
	})
	opts := ProxyConfig{Host: "127.0.0.1", Port: -1, RegionType: RegionCN, McpProfile: &McpProfile{
		MCPOAuthAccessToken: "access", MCPOAuthAccessTokenExpire: time.Now().Add(time.Hour).Unix(),
		MCPOAuthRefreshToken: "refresh", MCPOAuthRefreshTokenExpire: time.Now().Add(24 * time.Hour).Unix(),
	}}
	require.ErrorContains(t, startMCPProxy(ctx, opts), "proxy server failed")
}
