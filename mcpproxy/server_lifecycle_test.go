package mcpproxy

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func testProxy(t *testing.T) *MCPProxy {
	t.Helper()
	setTestHome(t)
	now := time.Now().Unix()
	return NewMCPProxy(ProxyConfig{
		Host: "127.0.0.1", RegionType: RegionCN,
		CallbackManager: NewOAuthCallbackManager(),
		McpProfile:      &McpProfile{Name: "test", MCPOAuthAppId: "app", MCPOAuthAccessToken: "access", MCPOAuthRefreshToken: "refresh", MCPOAuthAccessTokenExpire: now + 3600, MCPOAuthRefreshTokenExpire: now + 86400},
	})
}

func oauthServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	previous := EndpointMap[RegionCN]
	current := previous
	current.OAuth = server.URL
	EndpointMap[RegionCN] = current
	t.Cleanup(func() { EndpointMap[RegionCN] = previous })
}

func TestRefreshAccessTokenPersistsAndPublishes(t *testing.T) {
	proxy := testProxy(t)
	r := proxy.TokenRefresher
	before := time.Now().Unix()
	oauthServer(t, func(w http.ResponseWriter, req *http.Request) {
		require.NoError(t, req.ParseForm())
		require.Equal(t, "refresh_token", req.Form.Get("grant_type"))
		require.Equal(t, "refresh", req.Form.Get("refresh_token"))
		require.Equal(t, "app", req.Form.Get("client_id"))
		_, _ = io.WriteString(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":1000}`)
	})
	require.NoError(t, r.ForceRefresh())
	require.False(t, r.refreshing)
	require.Equal(t, int64(1), r.stats.TokenRefreshes)
	require.GreaterOrEqual(t, r.stats.LastTokenRefresh, before)
	require.Equal(t, "new-access", r.profile.MCPOAuthAccessToken)
	require.GreaterOrEqual(t, r.profile.MCPOAuthAccessTokenExpire, before+1000)
	data, err := os.ReadFile(getMCPConfigPath())
	require.NoError(t, err)
	stored, err := NewMcpProfileFromBytes(data)
	require.NoError(t, err)
	require.Equal(t, r.profile, stored)
	token := <-r.tokenCh
	require.Equal(t, "new-access", token.Token)
	require.Equal(t, r.profile.MCPOAuthAccessTokenExpire, token.ExpiresAt)
}

func TestRefreshAccessTokenFailurePolicies(t *testing.T) {
	for _, tc := range []struct {
		name               string
		status             int
		body               string
		expired, wantError bool
		attempts           int
	}{
		{name: "permanent", status: 400, body: `{"error":"invalid_grant"}`, wantError: true, attempts: 1},
		{name: "temporary with valid token", status: 503, body: "unavailable", attempts: 3},
		{name: "temporary with expired token", status: 503, body: "unavailable", expired: true, wantError: true, attempts: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proxy := testProxy(t)
			r := proxy.TokenRefresher
			if tc.expired {
				r.profile.MCPOAuthAccessTokenExpire = time.Now().Unix() - 1
			}
			attempts := 0
			oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
				attempts++
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			err := r.refreshAccessToken()
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.attempts, attempts)
			require.False(t, r.refreshing)
			require.Equal(t, int64(1), r.stats.TokenRefreshErrors)
			require.Equal(t, "access", r.profile.MCPOAuthAccessToken)
			require.Empty(t, r.tokenCh)
		})
	}
}

func TestRefreshAccessTokenSaveFailureIsFatal(t *testing.T) {
	r := testProxy(t).TokenRefresher
	require.NoError(t, os.MkdirAll(getMCPConfigPath()+".tmp", 0700))
	oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":"new","refresh_token":"new-refresh","expires_in":1000}`)
	})
	require.ErrorContains(t, r.refreshAccessToken(), "failed to save refreshed tokens")
	require.ErrorContains(t, <-r.fatalErrCh, "failed to save refreshed tokens")
	require.False(t, r.refreshing)
}

func TestRefreshWaitersAndAlreadyInProgress(t *testing.T) {
	r := testProxy(t).TokenRefresher
	r.refreshing = true
	require.NoError(t, r.refreshAccessToken())
	r.reauthorizing = true
	require.NoError(t, r.reauthorizeWithProxy())
	r.refreshing = false
	r.reauthorizing = false
	require.NoError(t, r.waitForRefresh(r.profile.MCPOAuthAccessTokenExpire-1))
	require.NoError(t, r.waitForReauthorization(r.profile.MCPOAuthRefreshTokenExpire-1))
	require.ErrorContains(t, r.waitForRefresh(r.profile.MCPOAuthAccessTokenExpire), "timeout")
}

func TestRefresherStartStopAndHealthCheck(t *testing.T) {
	r := testProxy(t).TokenRefresher
	done := make(chan struct{})
	go func() { r.Start(); close(done) }()
	select {
	case token := <-r.tokenCh:
		require.Equal(t, "access", token.Token)
	case <-time.After(time.Second):
		t.Fatal("initial token was not published")
	}
	r.Stop()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("refresher did not stop")
	}
	r.stats.HealthCheckCounter = 119
	r.checkAndRefresh()
	require.Equal(t, int64(120), r.stats.HealthCheckCounter)
	require.NotZero(t, r.stats.LastHealthCheckTime)
	r.tokenCh <- TokenInfo{Token: "stale"}
	r.sendToken()
	require.Equal(t, "access", (<-r.tokenCh).Token)
}

func TestCheckAndRefreshReportsPermanentError(t *testing.T) {
	r := testProxy(t).TokenRefresher
	r.profile.MCPOAuthAccessTokenExpire = time.Now().Unix() + 1
	oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		_, _ = io.WriteString(w, `{"error":"invalid_client"}`)
	})
	r.checkAndRefresh()
	require.ErrorContains(t, <-r.fatalErrCh, "refresh access token failed")
}

func TestDeleteInvalidMCPConfigAndFatalErrorBuffer(t *testing.T) {
	r := testProxy(t).TokenRefresher
	for _, profile := range []*McpProfile{
		{Name: "test", MCPOAuthAccessToken: "short", MCPOAuthRefreshToken: "long-refresh-secret", MCPOAuthAccessTokenExpire: 10, MCPOAuthRefreshTokenExpire: 20},
		{Name: "test", MCPOAuthAccessToken: "long-access-secret"},
	} {
		require.NoError(t, saveMcpProfile(profile))
		deleteMcpConfigFile()
		_, err := os.Stat(getMCPConfigPath())
		require.True(t, os.IsNotExist(err))
	}
	require.NoError(t, os.WriteFile(getMCPConfigPath(), []byte("invalid"), 0600))
	deleteMcpConfigFile()
	require.NoError(t, os.Mkdir(getMCPConfigPath(), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(getMCPConfigPath(), "child"), nil, 0600))
	deleteMcpConfigFile()
	err := errors.New("first failure")
	r.reportFatalError(err)
	r.reportFatalError(errors.New("second failure"))
	require.Same(t, err, <-r.fatalErrCh)
}

func TestMCPProxyCallbackAndStartFailure(t *testing.T) {
	proxy := testProxy(t)
	proxy.TokenRefresher.callbackManager.StartWaiting()
	w := httptest.NewRecorder()
	proxy.handleOAuthCallback(w, httptest.NewRequest("GET", "/callback?code=authorized", nil))
	require.Equal(t, 200, w.Code)
	require.Equal(t, "authorized", <-proxy.TokenRefresher.callbackManager.pendingAuth)
	proxy.Host = "invalid:host"
	require.ErrorContains(t, proxy.Start(), "proxy server failed")
	require.NoError(t, proxy.Stop())
}

type nonFlushingWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *nonFlushingWriter) Header() http.Header         { return w.header }
func (w *nonFlushingWriter) WriteHeader(status int)      { w.status = status }
func (w *nonFlushingWriter) Write(p []byte) (int, error) { return w.body.Write(p) }

type failingFlusher struct{ *nonFlushingWriter }

func (*failingFlusher) Flush()                    {}
func (*failingFlusher) Write([]byte) (int, error) { return 0, errors.New("client disconnected") }

func TestSSEForwardingAndDisconnect(t *testing.T) {
	proxy := testProxy(t)
	response := func() *http.Response {
		return &http.Response{StatusCode: 202, Header: http.Header{"Content-Length": {"999"}, "X-Upstream": {"yes"}}, Body: io.NopCloser(strings.NewReader("data: one\n\ndata: two\n\n"))}
	}
	w := httptest.NewRecorder()
	proxy.handleSSE(w, response())
	require.Equal(t, 202, w.Code)
	require.Equal(t, "data: one\n\ndata: two\n\n", w.Body.String())
	require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	require.Empty(t, w.Header().Get("Content-Length"))
	require.Equal(t, "yes", w.Header().Get("X-Upstream"))
	require.True(t, w.Flushed)
	plain := &nonFlushingWriter{header: make(http.Header)}
	proxy.handleSSE(plain, response())
	require.Equal(t, 500, plain.status)
	proxy.handleSSE(&failingFlusher{&nonFlushingWriter{header: make(http.Header)}}, response())
	close(proxy.stopCh)
	w = httptest.NewRecorder()
	proxy.handleSSE(w, response())
	require.Empty(t, w.Body.String())
}

func TestHTTPForwardingReadFailureAndShutdown(t *testing.T) {
	proxy := testProxy(t)
	w := httptest.NewRecorder()
	proxy.handleHTTP(w, &http.Response{Body: io.NopCloser(iotest.ErrReader(errors.New("broken body")))})
	require.Equal(t, 500, w.Code)
	close(proxy.stopCh)
	w = httptest.NewRecorder()
	proxy.handleHTTP(w, &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("not forwarded"))})
	require.Empty(t, w.Body.String())
}

func TestServerAccessRulesAndPrintedStatus(t *testing.T) {
	server := MCPServerInfo{Id: "server-id", Name: "server-name", Urls: MCPInfoUrls{MCP: "https://upstream/mcp/demo", SSE: "https://upstream/sse/demo"}}
	proxy := testProxy(t)
	proxy.ExistMcpServers = []MCPServerInfo{server, {Id: "other", Name: "other", Urls: MCPInfoUrls{MCP: ":bad", SSE: ":bad"}}}
	proxy.serverPaths = map[string][]string{server.Id: {"/mcp/demo", "/sse/demo"}}
	for _, rule := range []string{"server-name", "server-id", "/mcp", "/sse"} {
		proxy.BlockedServers = []string{rule}
		proxy.AllowedServers = []string{rule}
		require.True(t, proxy.isServerBlocked(server), rule)
		require.True(t, proxy.isServerAllowed(server), rule)
	}
	proxy.BlockedServers = []string{"missing", "/other"}
	proxy.AllowedServers = []string{"missing", "/other"}
	require.False(t, proxy.isServerBlocked(server))
	require.False(t, proxy.isServerAllowed(server))
	out := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, &bytes.Buffer{})
	printProxyInfo(ctx, proxy)
	require.Contains(t, out.String(), "server-name (not in whitelist)")
	proxy.BlockedServers = []string{"server-id"}
	out.Reset()
	printProxyInfo(ctx, proxy)
	require.Contains(t, out.String(), "server-name (blocked)")
	proxy.BlockedServers, proxy.AllowedServers = nil, nil
	out.Reset()
	printProxyInfo(ctx, proxy)
	require.Contains(t, out.String(), "All servers are allowed")
	require.Contains(t, out.String(), "http://127.0.0.1:0/mcp/demo")
}

func TestSaveMCPProfileFilesystemFailures(t *testing.T) {
	t.Run("config parent is a file", func(t *testing.T) {
		home := setTestHome(t)
		require.NoError(t, os.WriteFile(filepath.Join(home, ".aliyun"), nil, 0600))
		require.ErrorContains(t, saveMcpProfile(&McpProfile{Name: "test"}), "failed to create config directory")
	})
	t.Run("destination is a directory", func(t *testing.T) {
		setTestHome(t)
		require.NoError(t, os.MkdirAll(getMCPConfigPath(), 0700))
		require.ErrorContains(t, saveMcpProfile(&McpProfile{Name: "test"}), "failed to rename")
		_, err := os.Stat(getMCPConfigPath() + ".tmp")
		require.True(t, os.IsNotExist(err))
	})
}

func TestHealthReportsRefreshTokenExpirationAndLastRefresh(t *testing.T) {
	proxy := testProxy(t)
	proxy.TokenRefresher.profile.MCPOAuthRefreshTokenExpire = time.Now().Unix() - 1
	proxy.stats.LastTokenRefresh = time.Now().Unix()
	w := httptest.NewRecorder()
	proxy.handleHealth(w, httptest.NewRequest("GET", "/health", nil))
	require.Equal(t, 503, w.Code)
	require.Contains(t, w.Body.String(), `"refresh_token_status":"expired"`)
	require.Contains(t, w.Body.String(), `"last_refresh"`)
}
