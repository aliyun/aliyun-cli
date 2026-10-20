package mcpproxy

import (
	"context"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPProxyCommandLifecycle(t *testing.T) {
	for _, scenario := range []string{"credentials listen failure", "oauth listen failure", "oauth fatal error"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			args := []string{"-test.run=^TestMCPProxyCommandLifecycleHelper$", "-test.timeout=12s"}
			if testing.CoverMode() != "" {
				if coverage := flag.Lookup("test.gocoverdir"); coverage != nil && coverage.Value.String() != "" {
					args = append(args, "-test.gocoverdir="+coverage.Value.String())
				}
			}
			// Isolate the SDK's TLS trust roots and the command's process-wide signal handlers.
			command := exec.CommandContext(ctx, os.Args[0], args...)
			command.Env = append(os.Environ(), "MCP_COMMAND_LIFECYCLE_HELPER="+scenario,
				"GODEBUG="+os.Getenv("GODEBUG")+",x509usefallbackroots=1")
			output, err := command.CombinedOutput()
			require.NoError(t, err, "%s", output)
		})
	}
}

func TestMCPProxyCommandLifecycleHelper(t *testing.T) {
	scenario := os.Getenv("MCP_COMMAND_LIFECYCLE_HELPER")
	if scenario == "" {
		return
	}
	ctx := newMCPCredentialTestContext(t)
	t.Setenv("ALIBABA_CLOUD_VENDOR", "test-vendor")
	t.Setenv("ALIBABA_CLOUD_USER_AGENT", "test-source/1")
	writeMCPCredentialTestConfig(t, ctx, mcpTestAKProfile("default", "test-key"))
	command := NewMCPProxyCommand()
	ctx.EnterCommand(command)
	setMCPCommandTestFlag(ctx, "no-browser", "")
	port := -1
	if scenario == "oauth fatal error" {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		port = listener.Addr().(*net.TCPAddr).Port
		require.NoError(t, listener.Close())
	}
	setMCPCommandTestFlag(ctx, "port", strconv.Itoa(port))
	var appCalls, listCalls, exchangeCalls, refreshCalls, upstreamCalls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Header.Get("x-acs-action") == "GetApplication":
			appCalls.Add(1)
			assert.Equal(t, "test-app-id", r.URL.Query().Get("AppId"))
			fmt.Fprintf(w, `{"Application":{"AppId":"test-app-id","AppName":%q,"AppType":"NativeApp","RedirectUris":{"RedirectUri":[%q]},"DelegatedScope":{"PredefinedScopes":{"PredefinedScope":[{"Name":"/acs/mcp-server"}]}},"AccessTokenValidity":3600,"RefreshTokenValidity":86400}}`, MCPOAuthAppName, buildRedirectUri("127.0.0.1", port))
		case r.URL.Path == "/apimcpservers":
			listCalls.Add(1)
			assert.Contains(t, r.UserAgent(), util.GetAliyunCliUserAgent()+"/aliyun-cli-mcp-proxy")
			_, _ = io.WriteString(w, `{"apiMcpServers":[{"id":"server","name":"server","urls":{"mcp":"https://example.invalid/mcp/server"}}]}`)
		case r.URL.Path == "/v1/token":
			refreshCalls.Add(1)
			assert.NoError(t, r.ParseForm())
			assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
			_, _ = io.WriteString(w, `{"access_token":"refreshed-token","refresh_token":"refreshed-refresh-token","expires_in":3600}`)
		case r.URL.Path == "/mcp/server":
			upstreamCalls.Add(1)
			assert.Equal(t, "Bearer existing-token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	x509.SetFallbackRoots(roots)
	originalEndpoint := EndpointMap[RegionCN]
	t.Cleanup(func() { EndpointMap[RegionCN] = originalEndpoint })
	endpoint := originalEndpoint
	endpoint.IMS = strings.TrimPrefix(server.URL, "https://")
	endpoint.MCP = endpoint.IMS
	endpoint.OAuth = server.URL
	EndpointMap[RegionCN] = endpoint
	setMCPCommandTestFlag(ctx, "upstream-url", server.URL)
	if scenario == "credentials listen failure" {
		setMCPCommandTestFlag(ctx, "skip-mcp-oauth", "")
		previousTransport := http.DefaultTransport
		t.Cleanup(func() { http.DefaultTransport = previousTransport })
		http.DefaultTransport = credentialRoundTripper(func(r *http.Request) (*http.Response, error) {
			exchangeCalls.Add(1)
			require.Equal(t, "ramoauth.aliyuncs.com", r.URL.Host)
			require.Equal(t, "GenerateAccessToken", r.URL.Query().Get("Action"))
			return credentialResponse(http.StatusOK, `{"Data":{"AccessToken":"mcp-token","ExpiresIn":3600}}`), nil
		})
	} else {
		profile := NewMcpProfile("default")
		profile.MCPOAuthAppName = MCPOAuthAppName
		profile.MCPOAuthAppId = "test-app-id"
		profile.MCPOAuthSiteType = "CN"
		profile.MCPOAuthAccessToken = "existing-token"
		profile.MCPOAuthAccessTokenExpire = time.Now().Add(time.Hour).Unix()
		profile.MCPOAuthRefreshToken = "existing-refresh-token"
		profile.MCPOAuthRefreshTokenExpire = time.Now().Add(24 * time.Hour).Unix()
		require.NoError(t, saveMcpProfile(profile))
	}

	if scenario != "oauth fatal error" {
		err := command.Run(ctx, nil)
		require.ErrorContains(t, err, "proxy server failed")
		if scenario == "credentials listen failure" {
			require.EqualValues(t, 1, exchangeCalls.Load())
			require.Zero(t, appCalls.Load())
		} else {
			require.EqualValues(t, 1, appCalls.Load())
			require.Zero(t, exchangeCalls.Load())
		}
	} else {
		stopped := make(chan error, 1)
		go func() { stopped <- command.Run(ctx, nil) }()
		client := &http.Client{Timeout: 2 * time.Second}
		defer client.CloseIdleConnections()
		baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
		require.Eventually(t, func() bool {
			response, err := client.Get(baseURL + "/health")
			if err != nil {
				return false
			}
			defer response.Body.Close()
			return response.StatusCode == http.StatusOK
		}, 3*time.Second, 10*time.Millisecond)
		// The refreshed token cannot be persisted, so the refresher must stop the command.
		require.NoError(t, os.Mkdir(getMCPConfigPath()+".tmp", 0700))
		response, err := client.Get(baseURL + "/mcp/server")
		require.NoError(t, err)
		_, err = io.Copy(io.Discard, response.Body)
		require.NoError(t, response.Body.Close())
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, response.StatusCode)
		select {
		case err := <-stopped:
			require.ErrorContains(t, err, "critical: failed to save refreshed tokens after 3 attempts")
		case <-time.After(3 * time.Second):
			t.Fatal("command did not stop after a fatal token refresh error")
		}
		require.EqualValues(t, 1, refreshCalls.Load())
		require.EqualValues(t, 1, upstreamCalls.Load())
		_, err = os.Stat(getMCPConfigPath())
		require.True(t, os.IsNotExist(err))
	}
	require.EqualValues(t, 1, listCalls.Load())
}

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
