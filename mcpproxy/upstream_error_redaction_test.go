package mcpproxy

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	fakeUpstreamQuerySecret    = "FAKE_TOKEN_20261103"
	fakeUpstreamUserinfoSecret = "userinfo-secret-20261103"
	fakeUpstreamFragmentSecret = "test-frag"
)

func fakeUpstreamTarget() string {
	return "https://" + fakeUpstreamUserinfoSecret + "@upstream.example.com/mcp/secret-server" +
		"?access_token=" + fakeUpstreamQuerySecret + "#" + fakeUpstreamFragmentSecret
}

func requireNoUpstreamSecrets(t *testing.T, label, value string) {
	t.Helper()
	for _, secret := range []string{fakeUpstreamQuerySecret, fakeUpstreamUserinfoSecret, fakeUpstreamFragmentSecret} {
		require.NotContains(t, value, secret, "%s leaked %q", label, secret)
	}
	require.NotContains(t, value, "access_token", "%s leaked the query parameter name", label)
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previous) })
	return &logs
}

func newFailingUpstreamProxy(t *testing.T, retryAfterRefresh bool) (*MCPProxy, *int) {
	t.Helper()
	proxy := testProxy(t)
	proxy.UpstreamBaseURL = "https://upstream.example.com"
	if retryAfterRefresh {
		oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`)
		})
	}
	attempts := new(int)
	proxy.upstreamClient = &http.Client{Transport: proxyRoundTripper(func(req *http.Request) (*http.Response, error) {
		*attempts = *attempts + 1
		if retryAfterRefresh && *attempts == 1 {
			return &http.Response{StatusCode: http.StatusUnauthorized, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unauthorized"))}, nil
		}
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connect: connection refused")}
	})}
	return proxy, attempts
}

func TestProxyUpstreamFailureDoesNotEchoSecrets(t *testing.T) {
	for _, tc := range []struct {
		name              string
		retryAfterRefresh bool
		phase             string
	}{
		{name: "initial send failure", phase: "initial"},
		{name: "retry after token refresh failure", retryAfterRefresh: true, phase: "retry_after_refresh"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proxy, attempts := newFailingUpstreamProxy(t, tc.retryAfterRefresh)
			logs := captureLogs(t)

			w := httptest.NewRecorder()
			proxy.ServeMCPProxyRequest(w, httptest.NewRequest(http.MethodPost, fakeUpstreamTarget(), strings.NewReader(`{"request":"body"}`)))

			if tc.retryAfterRefresh {
				require.Equal(t, 2, *attempts)
			} else {
				require.Equal(t, 1, *attempts)
			}
			require.Equal(t, http.StatusBadGateway, w.Code, "transport failures must stay a 502")
			require.Equal(t, int64(1), proxy.stats.ErrorRequests)
			require.Zero(t, proxy.stats.ActiveRequests)

			body := w.Body.String()
			requireNoUpstreamSecrets(t, "response body", body)
			requireNoUpstreamSecrets(t, "proxy log", logs.String())

			require.Contains(t, body, "phase="+tc.phase)
			require.Contains(t, body, "category=network")
			require.Contains(t, logs.String(), "phase="+tc.phase)
			require.Contains(t, logs.String(), "category=network")
			require.Contains(t, logs.String(), "upstream.example.com/mcp/secret-server",
				"the log keeps the sanitized destination for diagnostics")
		})
	}
}

func TestProxySuccessPathStillForwardsAfterRedaction(t *testing.T) {
	proxy := testProxy(t)
	proxy.UpstreamBaseURL = "https://upstream.example.com"
	proxy.upstreamClient = &http.Client{Transport: proxyRoundTripper(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer access", req.Header.Get("Authorization"))
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"result":"ok"}`))}, nil
	})}
	w := httptest.NewRecorder()
	proxy.ServeMCPProxyRequest(w, httptest.NewRequest(http.MethodPost, "/mcp/test?access_token="+fakeUpstreamQuerySecret, strings.NewReader(`{}`)))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, `{"result":"ok"}`, w.Body.String())
	require.Equal(t, int64(1), proxy.stats.SuccessRequests)
}
