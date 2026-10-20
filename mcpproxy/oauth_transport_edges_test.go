package mcpproxy

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type oauthBrokenBody struct{ err error }

func (b oauthBrokenBody) Read([]byte) (int, error) { return 0, b.err }
func (oauthBrokenBody) Close() error               { return nil }

func TestOAuthRefreshTransportAndMalformedSuccessErrors(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	cause := errors.New("connection interrupted")
	for _, tc := range []struct {
		name, body, want      string
		transportErr, bodyErr error
	}{
		{name: "transport", transportErr: cause, want: "http request"},
		{name: "body", bodyErr: cause, want: "failed to read response body"},
		{name: "JSON", body: "{", want: "failed to parse response JSON"},
		{name: "OAuth error", body: `{"error":"invalid_scope","error_description":"scope not granted"}`, want: "oauth error: invalid_scope: scope not granted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			http.DefaultTransport = proxyRoundTripper(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "/v1/token", req.URL.Path)
				if tc.transportErr != nil {
					return nil, tc.transportErr
				}
				var body io.ReadCloser = io.NopCloser(strings.NewReader(tc.body))
				if tc.bodyErr != nil {
					body = oauthBrokenBody{tc.bodyErr}
				}
				return &http.Response{StatusCode: 200, Body: body, Header: make(http.Header)}, nil
			})
			_, err := oauthRefresh("https://oauth.example.invalid", url.Values{"refresh_token": {"refresh"}})
			require.ErrorContains(t, err, tc.want)
			if tc.transportErr != nil || tc.bodyErr != nil {
				require.ErrorIs(t, err, cause)
			}
		})
	}
	_, err := oauthRefresh("://invalid", nil)
	require.ErrorContains(t, err, "failed to create request")
}

func TestCallbackManagerDiscardsStaleAndDuplicateDeliveries(t *testing.T) {
	manager := NewOAuthCallbackManager()
	manager.pendingAuth <- "stale"
	manager.errorCh <- errors.New("old error")
	manager.StartWaiting()
	require.Empty(t, manager.pendingAuth)
	require.Empty(t, manager.errorCh)
	require.True(t, manager.HandleCallback("fresh", nil))
	require.True(t, manager.HandleCallback("duplicate", nil))
	code, err := manager.WaitForCode()
	require.NoError(t, err)
	require.Equal(t, "fresh", code)
	cause := errors.New("authorization denied")
	require.True(t, manager.HandleCallback("", cause))
	require.True(t, manager.HandleCallback("", errors.New("duplicate")))
	_, err = manager.WaitForCode()
	require.ErrorIs(t, err, cause)
	manager.StopWaiting()
	require.False(t, manager.HandleCallback("late", nil))
}

type callbackFailureWriter struct {
	header http.Header
	writes int
}

func (w *callbackFailureWriter) Header() http.Header { return w.header }
func (w *callbackFailureWriter) WriteHeader(int)     {}
func (w *callbackFailureWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, errors.New("client disconnected")
}

func TestCallbackPageHandlesDisconnectedClient(t *testing.T) {
	for _, tc := range []struct {
		query  string
		manual bool
	}{{"", false}, {"?code=authorization", true}, {"?code=authorization", false}} {
		writer := &callbackFailureWriter{header: make(http.Header)}
		require.True(t, handleOAuthCallbackRequest(writer, httptest.NewRequest("GET", "/callback"+tc.query, nil), func(string, error) bool { return true }, tc.manual))
		require.GreaterOrEqual(t, writer.writes, 2)
	}
}

func TestUpstreamRequestAndBlocklistBoundaries(t *testing.T) {
	proxy := testProxy(t)
	require.False(t, proxy.isPathBlocked("/mcp/allowed"))
	proxy.BlockedServers = []string{"/mcp/blocked", "missing-server"}
	require.False(t, proxy.isPathBlocked("/mcp/allowed"))
	require.True(t, proxy.isPathBlocked("/mcp/blocked/child"))
	proxy.UpstreamBaseURL = "upstream.example.invalid"
	req := &http.Request{Method: "GET", URL: &url.URL{}, Header: make(http.Header)}
	forwarded, err := proxy.buildUpstreamRequest(req, "token")
	require.NoError(t, err)
	require.Equal(t, "/", forwarded.URL.Path)
	req.Method = "invalid method"
	_, err = proxy.buildUpstreamRequest(req, "token")
	require.Error(t, err)
	data, err := readMCPRequestBody(httptest.NewRecorder(), &http.Request{}, 1024)
	require.NoError(t, err)
	require.Nil(t, data)
	response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": nil}, Body: io.NopCloser(strings.NewReader("data: ready\n\n"))}
	out := httptest.NewRecorder()
	proxy.handleSSE(out, response)
	require.Equal(t, "text/event-stream", out.Header().Get("Content-Type"))
	require.Contains(t, out.Body.String(), "ready")
}

func TestInteractiveInputRejectsRegularAndClosedFiles(t *testing.T) {
	old := os.Stdin
	t.Cleanup(func() { os.Stdin = old })
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	require.NoError(t, err)
	os.Stdin = file
	require.False(t, isInteractiveInput())
	require.NoError(t, file.Close())
	require.False(t, isInteractiveInput())
}
