package mcpproxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/require"
)

type proxyRoundTripper func(*http.Request) (*http.Response, error)

func (f proxyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestProxyForwardsResponsesAndCountsRequests(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body string
		status                  int
	}{
		{name: "JSON", contentType: "application/json", body: `{"result":"ok"}`, status: 201},
		{name: "SSE", contentType: "text/event-stream", body: "data: ready\n\n", status: 200},
		{name: "HTTP error", contentType: "text/plain", body: "upstream failure", status: 503},
		{name: "SSE error", contentType: "text/event-stream", body: "data: unavailable\n\n", status: 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proxy := testProxy(t)
			proxy.upstreamClient = &http.Client{Transport: proxyRoundTripper(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "Bearer access", req.Header.Get("Authorization"))
				require.Equal(t, "/mcp/test", req.URL.Path)
				require.Equal(t, "value", req.URL.Query().Get("query"))
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Content-Type": {tc.contentType}, "X-Upstream": {"yes"}}, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}
			req := httptest.NewRequest("GET", "/mcp/test?query=value", nil)
			req.Body = nil
			w := httptest.NewRecorder()
			proxy.ServeMCPProxyRequest(w, req)
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, tc.body, w.Body.String())
			require.Equal(t, "yes", w.Header().Get("X-Upstream"))
			require.Equal(t, int64(1), proxy.stats.TotalRequests)
			require.Zero(t, proxy.stats.ActiveRequests)
			if tc.status < 400 {
				require.Equal(t, int64(1), proxy.stats.SuccessRequests)
			} else {
				require.Equal(t, int64(1), proxy.stats.ErrorRequests)
			}
		})
	}
}

func TestProxyRetriesUnauthorizedWithFreshTokenAndBody(t *testing.T) {
	for _, tc := range []struct {
		name           string
		retryStatus    int
		retryError     bool
		expiredRefresh bool
	}{
		{name: "success", retryStatus: 200},
		{name: "still unauthorized", retryStatus: 401},
		{name: "retry network failure", retryError: true},
		{name: "reauthorize expired refresh token", retryStatus: 200, expiredRefresh: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proxy := testProxy(t)
			if tc.expiredRefresh {
				proxy.TokenRefresher.profile.MCPOAuthRefreshTokenExpire = time.Now().Add(-time.Hour).Unix()
				proxy.TokenRefresher.autoOpenBrowser = true
				callbackAuthorization(t, proxy.TokenRefresher.callbackManager, "authorization-code")
			}
			oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`)
			})
			attempts := 0
			proxy.upstreamClient = &http.Client{Transport: proxyRoundTripper(func(req *http.Request) (*http.Response, error) {
				attempts++
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.Equal(t, `{"request":"body"}`, string(body))
				status := 401
				if attempts == 1 {
					require.Equal(t, "Bearer access", req.Header.Get("Authorization"))
				} else {
					require.Equal(t, "Bearer new-access", req.Header.Get("Authorization"))
					if tc.retryError {
						return nil, errors.New("upstream offline")
					}
					status = tc.retryStatus
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("response"))}, nil
			})}
			w := httptest.NewRecorder()
			proxy.ServeMCPProxyRequest(w, httptest.NewRequest("POST", "/mcp/test", bytes.NewBufferString(`{"request":"body"}`)))
			require.Equal(t, 2, attempts)
			if tc.retryError {
				require.Equal(t, 502, w.Code)
			} else {
				require.Equal(t, tc.retryStatus, w.Code)
			}
			if tc.retryStatus == 200 {
				require.Equal(t, "response", w.Body.String())
			}
			require.Equal(t, int64(1), proxy.stats.TokenRefreshes)
		})
	}
}

func TestProxyRejectsExpiredTokensWhenRefreshFails(t *testing.T) {
	for _, initiallyExpired := range []bool{true, false} {
		t.Run(fmt.Sprint(initiallyExpired), func(t *testing.T) {
			proxy := testProxy(t)
			if initiallyExpired {
				proxy.TokenRefresher.profile.MCPOAuthAccessTokenExpire = time.Now().Add(-time.Hour).Unix()
			}
			refreshes := 0
			oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
				refreshes++
				if !initiallyExpired && refreshes == 1 {
					_, _ = io.WriteString(w, `{"access_token":"expired","refresh_token":"refresh","expires_in":0}`)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
			})
			upstreamCalls := 0
			proxy.upstreamClient = &http.Client{Transport: proxyRoundTripper(func(*http.Request) (*http.Response, error) {
				upstreamCalls++
				return &http.Response{StatusCode: 401, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unauthorized"))}, nil
			})}
			response := httptest.NewRecorder()
			proxy.ServeMCPProxyRequest(response, httptest.NewRequest("GET", "/mcp/test", nil))
			require.Equal(t, http.StatusInternalServerError, response.Code)
			require.Equal(t, int64(1), proxy.stats.ErrorRequests)
			if initiallyExpired {
				require.Zero(t, upstreamCalls)
			} else {
				require.Equal(t, 1, upstreamCalls)
			}
		})
	}
}

func TestProxyRejectsUnreadableBodyAndUpstreamFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		setup  func(*MCPProxy, *http.Request)
		status int
	}{
		{name: "body read", status: 400, setup: func(_ *MCPProxy, req *http.Request) {
			req.Body = io.NopCloser(iotest.ErrReader(errors.New("request read failed")))
		}},
		{name: "bad upstream URL", status: 502, setup: func(proxy *MCPProxy, _ *http.Request) { proxy.UpstreamBaseURL = "https://bad%host" }},
		{name: "transport", status: 502, setup: func(proxy *MCPProxy, _ *http.Request) {
			proxy.upstreamClient = &http.Client{Transport: proxyRoundTripper(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proxy := testProxy(t)
			req := httptest.NewRequest("POST", "/mcp/test", strings.NewReader("body"))
			tc.setup(proxy, req)
			w := httptest.NewRecorder()
			proxy.ServeMCPProxyRequest(w, req)
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, int64(1), proxy.stats.ErrorRequests)
			require.Zero(t, proxy.stats.ActiveRequests)
		})
	}
	proxy := testProxy(t)
	for _, path := range []string{"/favicon.ico", "/robots.txt", "/apple-touch-icon.png"} {
		w := httptest.NewRecorder()
		proxy.ServeMCPProxyRequest(w, httptest.NewRequest("GET", path, nil))
		require.Equal(t, 404, w.Code)
	}
}

func TestProxyUnauthorizedRefreshFailure(t *testing.T) {
	proxy := testProxy(t)
	oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
	})
	proxy.upstreamClient = &http.Client{Transport: proxyRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 401, Body: io.NopCloser(strings.NewReader("unauthorized"))}, nil
	})}
	w := httptest.NewRecorder()
	proxy.ServeMCPProxyRequest(w, httptest.NewRequest("GET", "/mcp/test", nil))
	require.Equal(t, 401, w.Code)
	require.Contains(t, w.Body.String(), "Authentication failed")
	require.Equal(t, int64(1), proxy.stats.ErrorRequests)
}

func TestProxyExpiredTokenRefresh(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[fail], func(t *testing.T) {
			proxy := testProxy(t)
			proxy.TokenRefresher.profile.MCPOAuthAccessTokenExpire = time.Now().Unix() - 1
			oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
				if fail {
					w.WriteHeader(400)
					_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
					return
				}
				_, _ = io.WriteString(w, `{"access_token":"fresh","refresh_token":"refresh","expires_in":3600}`)
			})
			token, err := proxy.getMCPAccessToken()
			if fail {
				require.ErrorContains(t, err, "failed to refresh access token")
			} else {
				require.NoError(t, err)
				require.Equal(t, "fresh", token)
			}
		})
	}
}

func TestProxyCustomDefaultTransport(t *testing.T) {
	previous := http.DefaultTransport
	transport := proxyRoundTripper(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previous })
	client := newMCPUpstreamHTTPClient()
	require.IsType(t, transport, client.Transport)
	proxy := testProxy(t)
	proxy.upstreamClient = nil
	w := httptest.NewRecorder()
	proxy.ServeMCPProxyRequest(w, httptest.NewRequest("GET", "/mcp/test", nil))
	require.Equal(t, 502, w.Code)
}

func TestProxyUpstreamFailuresDoNotExposeSecrets(t *testing.T) {
	const sensitiveURL = "https://FAKE_USER_20261103:FAKE_PASSWORD_20261103@localhost/mcp/test?access_token=FAKE_TOKEN_20261103#FAKE_FRAGMENT_20261103"
	for _, retry := range []bool{false, true} {
		stage := "initial"
		if retry {
			stage = "retry_after_refresh"
		}
		for _, tc := range []struct {
			name, requestURL, errorURL, category, buildURL string
			cause                                          error
		}{
			{name: "wrapped URL", requestURL: sensitiveURL, errorURL: sensitiveURL, category: "transport", cause: errors.New("inner FAKE_ERROR_20261103")},
			{name: "encoded URL", requestURL: strings.ReplaceAll(sensitiveURL, "FAKE_", "%46AKE_"), errorURL: strings.ReplaceAll(sensitiveURL, "FAKE_", "%46AKE_"), category: "transport", cause: errors.New("inner FAKE_ERROR_20261103")},
			{name: "wrapped timeout", requestURL: sensitiveURL, errorURL: sensitiveURL, category: "timeout", cause: fmt.Errorf("inner FAKE_ERROR_20261103: %w", context.DeadlineExceeded)},
			{name: "opaque error URL", requestURL: sensitiveURL, errorURL: "https:FAKE_TOKEN_20261103?access_token=FAKE_ERROR_20261103", category: "transport", cause: errors.New("inner FAKE_ERROR_20261103")},
			{name: "build URL", requestURL: sensitiveURL, category: "build_request", buildURL: "https://bad%host?access_token=FAKE_TOKEN_20261103#FAKE_FRAGMENT_20261103"},
		} {
			t.Run(stage+"/"+tc.name, func(t *testing.T) {
				proxy := testProxy(t)
				oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
					_, _ = io.WriteString(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`)
				})
				var logs bytes.Buffer
				previous := log.Writer()
				log.SetOutput(&logs)
				t.Cleanup(func() { log.SetOutput(previous) })
				if !retry && tc.buildURL != "" {
					proxy.UpstreamBaseURL = tc.buildURL
				}
				attempts := 0
				proxy.upstreamClient = &http.Client{Transport: proxyRoundTripper(func(req *http.Request) (*http.Response, error) {
					attempts++
					if retry && attempts == 1 {
						require.Equal(t, "Bearer access", req.Header.Get("Authorization"))
						if tc.buildURL != "" {
							proxy.UpstreamBaseURL = tc.buildURL
						}
						return &http.Response{StatusCode: 401, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unauthorized"))}, nil
					}
					if retry {
						require.Equal(t, "Bearer new-access", req.Header.Get("Authorization"))
					}
					inner := &url.Error{Op: "Get", URL: tc.errorURL, Err: tc.cause}
					return nil, fmt.Errorf("outer FAKE_ERROR_20261103: %w", fmt.Errorf("middle FAKE_ERROR_20261103: %w", inner))
				})}
				w := httptest.NewRecorder()
				proxy.ServeMCPProxyRequest(w, httptest.NewRequest("GET", tc.requestURL, nil))
				require.Equal(t, http.StatusBadGateway, w.Code)
				for name, output := range map[string]string{"body": w.Body.String(), "logs": logs.String()} {
					for _, secret := range []string{"FAKE_", "%46AKE_", "access_token=", sensitiveURL, "failed to send request:", "failed to build upstream request:"} {
						if strings.Contains(output, secret) {
							t.Errorf("%s exposes sensitive error data: %q", name, output)
							break
						}
					}
				}
				require.Equal(t, "Upstream request failed ("+tc.category+")\n", w.Body.String())
				require.Contains(t, logs.String(), "stage="+stage)
				require.Contains(t, logs.String(), "category="+tc.category)
				wantAttempts := 1
				if retry {
					wantAttempts++
					require.Equal(t, int64(1), proxy.stats.TokenRefreshes)
				} else {
					require.Zero(t, proxy.stats.TokenRefreshes)
				}
				if tc.buildURL != "" {
					wantAttempts--
					require.Contains(t, logs.String(), "url=<unknown>")
				} else {
					require.Contains(t, logs.String(), "method=GET url=https://"+EndpointMap[RegionCN].MCP+"/mcp/test")
				}
				require.Equal(t, wantAttempts, attempts)
				require.Equal(t, int64(1), proxy.stats.TotalRequests)
				require.Equal(t, int64(1), proxy.stats.ErrorRequests)
				require.Zero(t, proxy.stats.SuccessRequests)
				require.Zero(t, proxy.stats.ActiveRequests)
			})
		}
	}
}
