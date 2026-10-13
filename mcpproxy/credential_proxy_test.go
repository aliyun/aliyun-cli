// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mcpproxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	credentialsv2 "github.com/aliyun/credentials-go/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type proxyCredentialTransportFunc func(*http.Request) (*http.Response, error)

func (f proxyCredentialTransportFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func proxyTestCredential() (*credentialsv2.CredentialModel, error) {
	return &credentialsv2.CredentialModel{
		Type:            tea.String("access_key"),
		AccessKeyId:     tea.String("test-access-key"),
		AccessKeySecret: tea.String("test-access-key-secret"),
	}, nil
}

func newProxyTestCredentialAuth() (*credentialAuth, *atomic.Int32) {
	auth := newCredentialAuth(proxyTestCredential, RegionCN)
	calls := new(atomic.Int32)
	auth.client.Transport = proxyCredentialTransportFunc(func(r *http.Request) (*http.Response, error) {
		token := fmt.Sprintf("credential-token-%d", calls.Add(1))
		body := fmt.Sprintf(`{"Data":{"AccessToken":%q,"ExpiresIn":3600}}`, token)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})
	return auth, calls
}

func TestCredentialProxyForwardsRequest(t *testing.T) {
	const requestBody = `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/mcp/server%2Fid", r.URL.EscapedPath())
		assert.Equal(t, "marker=a%2Bb&tag=one&tag=two", r.URL.RawQuery)
		assert.Equal(t, "Bearer credential-token-1", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json, text/event-stream", r.Header.Get("Accept"))
		assert.Equal(t, "2025-06-18", r.Header.Get("Mcp-Protocol-Version"))
		assert.Equal(t, "session-123", r.Header.Get("Mcp-Session-Id"))
		assert.Contains(t, r.Header.Get("User-Agent"), "aliyun-cli-mcp-proxy")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, requestBody, string(body))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "session-456")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`)
	}))
	defer upstream.Close()
	auth, exchanges := newProxyTestCredentialAuth()
	proxy := NewMCPProxy(ProxyConfig{CredentialAuth: auth, UpstreamBaseURL: upstream.URL})
	t.Cleanup(proxy.upstreamClient.CloseIdleConnections)
	require.Nil(t, proxy.TokenRefresher)
	req := httptest.NewRequest(http.MethodPost, "/mcp/server%2Fid?marker=a%2Bb&tag=one&tag=two", strings.NewReader(requestBody))
	req.Header.Set("Authorization", "Bearer untrusted-client-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Protocol-Version", "2025-06-18")
	req.Header.Set("Mcp-Session-Id", "session-123")
	recorder := httptest.NewRecorder()

	proxy.ServeMCPProxyRequest(recorder, req)

	assert.Equal(t, http.StatusAccepted, recorder.Code)
	assert.Equal(t, "session-456", recorder.Header().Get("Mcp-Session-Id"))
	assert.JSONEq(t, `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`, recorder.Body.String())
	assert.EqualValues(t, 1, exchanges.Load())
	assert.EqualValues(t, 1, atomic.LoadInt64(&proxy.stats.SuccessRequests))
	assert.Zero(t, atomic.LoadInt64(&proxy.stats.ActiveRequests))
}

func TestCredentialProxyUnauthorizedRefreshesAndReplaysOnce(t *testing.T) {
	for _, body := range []string{"", `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"test"}}`} {
		for _, finalStatus := range []int{http.StatusOK, http.StatusUnauthorized, http.StatusForbidden} {
			t.Run(fmt.Sprintf("body_length_%d_status_%d", len(body), finalStatus), func(t *testing.T) {
				var requests atomic.Int32
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					call := requests.Add(1)
					assert.LessOrEqual(t, call, int32(2))
					assert.Equal(t, fmt.Sprintf("Bearer credential-token-%d", call), r.Header.Get("Authorization"))
					assert.Equal(t, "/mcp/retry?session=123", r.URL.RequestURI())
					requestBody, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					assert.Equal(t, body, string(requestBody))
					if call == 1 {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					w.WriteHeader(finalStatus)
					_, _ = io.WriteString(w, "upstream-result")
				}))
				defer upstream.Close()
				auth, exchanges := newProxyTestCredentialAuth()
				proxy := NewMCPProxy(ProxyConfig{CredentialAuth: auth, UpstreamBaseURL: upstream.URL})
				t.Cleanup(proxy.upstreamClient.CloseIdleConnections)
				recorder := httptest.NewRecorder()

				proxy.ServeMCPProxyRequest(recorder, httptest.NewRequest(http.MethodPost, "/mcp/retry?session=123", strings.NewReader(body)))

				assert.Equal(t, finalStatus, recorder.Code)
				assert.EqualValues(t, 2, requests.Load())
				assert.EqualValues(t, 2, exchanges.Load())
				if finalStatus != http.StatusUnauthorized {
					assert.Equal(t, "upstream-result", recorder.Body.String())
				}
				assert.Zero(t, atomic.LoadInt64(&proxy.stats.ActiveRequests))
			})
		}
	}
}

func TestCredentialProxyDoesNotRefreshForOtherStatusCodes(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusNotFound, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var requests atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Retry-After", "30")
				w.WriteHeader(status)
				_, _ = io.WriteString(w, "upstream-error")
			}))
			defer upstream.Close()
			auth, exchanges := newProxyTestCredentialAuth()
			proxy := NewMCPProxy(ProxyConfig{CredentialAuth: auth, UpstreamBaseURL: upstream.URL})
			t.Cleanup(proxy.upstreamClient.CloseIdleConnections)
			recorder := httptest.NewRecorder()

			proxy.ServeMCPProxyRequest(recorder, httptest.NewRequest(http.MethodGet, "/mcp/server", nil))

			assert.Equal(t, status, recorder.Code)
			assert.Equal(t, "upstream-error", recorder.Body.String())
			assert.Equal(t, "30", recorder.Header().Get("Retry-After"))
			assert.EqualValues(t, 1, requests.Load())
			assert.EqualValues(t, 1, exchanges.Load())
			assert.EqualValues(t, 1, atomic.LoadInt64(&proxy.stats.ErrorRequests))
		})
	}
}

func TestCredentialProxyConcurrentUnauthorizedRequestsShareRefresh(t *testing.T) {
	const requestCount = 8
	var oldTokenRequests, newTokenRequests atomic.Int32
	allOldRequestsStarted := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Authorization") {
		case "Bearer credential-token-1":
			if oldTokenRequests.Add(1) == requestCount {
				close(allOldRequestsStarted)
			}
			select {
			case <-allOldRequestsStarted:
			case <-r.Context().Done():
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
		case "Bearer credential-token-2":
			newTokenRequests.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected authorization: %s", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer upstream.Close()
	auth, exchanges := newProxyTestCredentialAuth()
	proxy := NewMCPProxy(ProxyConfig{CredentialAuth: auth, UpstreamBaseURL: upstream.URL})
	t.Cleanup(proxy.upstreamClient.CloseIdleConnections)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results := make(chan int, requestCount)
	for i := 0; i < requestCount; i++ {
		go func() {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/mcp/concurrent", nil).WithContext(ctx)
			proxy.ServeMCPProxyRequest(recorder, req)
			results <- recorder.Code
		}()
	}
	for i := 0; i < requestCount; i++ {
		select {
		case status := <-results:
			assert.Equal(t, http.StatusNoContent, status)
		case <-ctx.Done():
			t.Fatal("concurrent proxy requests did not complete")
		}
	}
	assert.EqualValues(t, requestCount, oldTokenRequests.Load())
	assert.EqualValues(t, requestCount, newTokenRequests.Load())
	assert.EqualValues(t, 2, exchanges.Load())
	assert.EqualValues(t, requestCount, atomic.LoadInt64(&proxy.stats.SuccessRequests))
}

func TestCredentialProxyStreamsSSEBeforeUpstreamCompletes(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer credential-token-1", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		<-release
		_, _ = io.WriteString(w, "data: second\n\n")
	}))
	defer upstream.Close()
	auth, exchanges := newProxyTestCredentialAuth()
	proxy := NewMCPProxy(ProxyConfig{CredentialAuth: auth, UpstreamBaseURL: upstream.URL})
	t.Cleanup(proxy.upstreamClient.CloseIdleConnections)
	frontend := httptest.NewServer(http.HandlerFunc(proxy.ServeMCPProxyRequest))
	defer frontend.Close()
	defer close(release)
	client := &http.Client{Timeout: 3 * time.Second}

	resp, err := client.Get(frontend.URL + "/sse/server")
	require.NoError(t, err, "the proxy must flush the first event while the upstream remains open")
	defer resp.Body.Close()
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	assert.Equal(t, "no", resp.Header.Get("X-Accel-Buffering"))
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "data: first\n", line)
	assert.EqualValues(t, 1, exchanges.Load())
}

func TestCredentialProxyCredentialErrorsDoNotExposeSecrets(t *testing.T) {
	const secret = "credential-source-secret-value"
	for _, duringRefresh := range []bool{false, true} {
		t.Run(fmt.Sprintf("refresh_%t", duringRefresh), func(t *testing.T) {
			var requests atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(http.StatusUnauthorized)
			}))
			defer upstream.Close()
			auth, exchanges := newProxyTestCredentialAuth()
			if duringRefresh {
				_, err := auth.GetToken()
				require.NoError(t, err)
			}
			auth.source = func() (*credentialsv2.CredentialModel, error) {
				return nil, errors.New(secret)
			}
			proxy := NewMCPProxy(ProxyConfig{CredentialAuth: auth, UpstreamBaseURL: upstream.URL})
			t.Cleanup(proxy.upstreamClient.CloseIdleConnections)
			var logs bytes.Buffer
			previousLogOutput := log.Writer()
			log.SetOutput(&logs)
			defer log.SetOutput(previousLogOutput)
			recorder := httptest.NewRecorder()

			proxy.ServeMCPProxyRequest(recorder, httptest.NewRequest(http.MethodPost, "/mcp/server", strings.NewReader("{}")))

			if duringRefresh {
				assert.Equal(t, http.StatusUnauthorized, recorder.Code)
				assert.EqualValues(t, 1, requests.Load())
				assert.EqualValues(t, 1, exchanges.Load())
			} else {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Zero(t, requests.Load())
				assert.Zero(t, exchanges.Load())
			}
			assert.NotContains(t, recorder.Body.String(), secret)
			assert.NotContains(t, logs.String(), secret)
			assert.EqualValues(t, 1, atomic.LoadInt64(&proxy.stats.ErrorRequests))
		})
	}
}

func TestCredentialProxyRejectsRequestsBeforeLoadingCredentials(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		allowed  []string
		blocked  []string
		stopped  bool
		wantCode int
	}{
		{name: "favicon", path: "/favicon.ico", wantCode: http.StatusNotFound},
		{name: "robots", path: "/robots.txt", wantCode: http.StatusNotFound},
		{name: "touch icon", path: "/apple-touch-icon.png", wantCode: http.StatusNotFound},
		{name: "blocked", path: "/mcp/blocked", blocked: []string{"/mcp/blocked"}, wantCode: http.StatusForbidden},
		{name: "not allowed", path: "/mcp/other", allowed: []string{"/mcp/allowed"}, wantCode: http.StatusForbidden},
		{name: "blocked overrides allowed", path: "/mcp/blocked", allowed: []string{"/mcp"}, blocked: []string{"/mcp/blocked"}, wantCode: http.StatusForbidden},
		{name: "stopped", path: "/mcp/server", stopped: true, wantCode: http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sourceCalls atomic.Int32
			auth := newCredentialAuth(func() (*credentialsv2.CredentialModel, error) {
				sourceCalls.Add(1)
				return nil, errors.New("credential source must not be consulted")
			}, RegionCN)
			proxy := NewMCPProxy(ProxyConfig{CredentialAuth: auth, AllowedServers: tt.allowed, BlockedServers: tt.blocked})
			if tt.stopped {
				require.NoError(t, proxy.Stop())
			}
			recorder := httptest.NewRecorder()

			proxy.ServeMCPProxyRequest(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantCode, recorder.Code)
			assert.Zero(t, sourceCalls.Load())
			assert.Zero(t, atomic.LoadInt64(&proxy.stats.ActiveRequests))
		})
	}
}

func TestCredentialProxyHealthAndCallback(t *testing.T) {
	auth, exchanges := newProxyTestCredentialAuth()
	_, err := auth.GetToken()
	require.NoError(t, err)
	proxy := NewMCPProxy(ProxyConfig{CredentialAuth: auth})
	require.Nil(t, proxy.TokenRefresher)
	healthRecorder := httptest.NewRecorder()

	proxy.handleHealth(healthRecorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusOK, healthRecorder.Code)
	var health map[string]any
	require.NoError(t, json.Unmarshal(healthRecorder.Body.Bytes(), &health))
	assert.Equal(t, "healthy", health["status"])
	assert.Equal(t, "credentials", health["auth_type"])
	assert.Equal(t, "valid", health["token_status"])
	assert.NotContains(t, health, "refresh_token_status")
	assert.NotContains(t, healthRecorder.Body.String(), "credential-token-1")
	assert.EqualValues(t, 1, exchanges.Load())
	callbackRecorder := httptest.NewRecorder()

	proxy.handleOAuthCallback(callbackRecorder, httptest.NewRequest(http.MethodGet, "/callback?code=unused", nil))

	assert.Equal(t, http.StatusNotFound, callbackRecorder.Code)
	assert.EqualValues(t, 1, exchanges.Load())
}

func TestCredentialProxyHealthDoesNotAcquireTokens(t *testing.T) {
	tests := []struct {
		name        string
		expiresAt   time.Time
		lastErr     error
		tokenStatus string
	}{
		{name: "not acquired", tokenStatus: "unavailable"},
		{name: "expired", expiresAt: time.Now().Add(-time.Minute), tokenStatus: "expired"},
		{name: "refresh failed", expiresAt: time.Now().Add(time.Hour), lastErr: errors.New("sensitive-refresh-error"), tokenStatus: "unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, exchanges := newProxyTestCredentialAuth()
			auth.expiresAt, auth.lastErr = tt.expiresAt, tt.lastErr
			proxy := NewMCPProxy(ProxyConfig{CredentialAuth: auth})
			recorder := httptest.NewRecorder()

			proxy.handleHealth(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

			assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
			var health map[string]any
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &health))
			assert.Equal(t, "degraded", health["status"])
			assert.Equal(t, tt.tokenStatus, health["token_status"])
			assert.NotContains(t, recorder.Body.String(), "sensitive-refresh-error")
			assert.Zero(t, exchanges.Load())
		})
	}
}

func TestCredentialProxyStartFailureAndStop(t *testing.T) {
	auth, exchanges := newProxyTestCredentialAuth()
	proxy := NewMCPProxy(ProxyConfig{Host: "127.0.0.1", Port: -1, CredentialAuth: auth})

	require.Error(t, proxy.Start())
	require.NotNil(t, proxy.Server)
	require.NoError(t, proxy.Stop())
	assert.Zero(t, exchanges.Load())
	select {
	case <-proxy.stopCh:
	default:
		t.Fatal("stop must close the proxy shutdown channel")
	}
}

func TestOAuthProxyContinuesUsingOAuthToken(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer existing-oauth-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	profile := NewMcpProfile("oauth-regression")
	profile.MCPOAuthAccessToken = "existing-oauth-token"
	profile.MCPOAuthAccessTokenExpire = time.Now().Add(time.Hour).Unix()
	proxy := NewMCPProxy(ProxyConfig{McpProfile: profile, UpstreamBaseURL: upstream.URL})
	t.Cleanup(proxy.upstreamClient.CloseIdleConnections)
	require.NotNil(t, proxy.TokenRefresher)
	require.Nil(t, proxy.CredentialAuth)
	recorder := httptest.NewRecorder()

	proxy.ServeMCPProxyRequest(recorder, httptest.NewRequest(http.MethodDelete, "/mcp/server", nil))

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.EqualValues(t, 1, atomic.LoadInt64(&proxy.stats.SuccessRequests))
}

func TestOAuthProxyUnauthorizedStillRefreshesOAuthToken(t *testing.T) {
	setTestHome(t)
	var refreshRequests, proxyRequests atomic.Int32
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshRequests.Add(1)
		assert.Equal(t, "/v1/token", r.URL.Path)
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, "existing-refresh-token", r.Form.Get("refresh_token"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"refreshed-oauth-token","refresh_token":"new-refresh-token","expires_in":3600,"token_type":"Bearer"}`)
	}))
	defer oauthServer.Close()
	const region RegionType = "oauth-proxy-refresh-test"
	EndpointMap[region] = EndpointConfig{OAuth: oauthServer.URL}
	defer delete(EndpointMap, region)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := proxyRequests.Add(1)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, `{"id":1}`, string(body))
		if call == 1 {
			assert.Equal(t, "Bearer existing-oauth-token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		assert.Equal(t, "Bearer refreshed-oauth-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	profile := NewMcpProfile("oauth-refresh-regression")
	profile.MCPOAuthAccessToken = "existing-oauth-token"
	profile.MCPOAuthAccessTokenExpire = time.Now().Add(time.Hour).Unix()
	profile.MCPOAuthRefreshToken = "existing-refresh-token"
	profile.MCPOAuthRefreshTokenExpire = time.Now().Add(24 * time.Hour).Unix()
	proxy := NewMCPProxy(ProxyConfig{McpProfile: profile, RegionType: region, UpstreamBaseURL: upstream.URL})
	t.Cleanup(proxy.upstreamClient.CloseIdleConnections)
	recorder := httptest.NewRecorder()

	proxy.ServeMCPProxyRequest(recorder, httptest.NewRequest(http.MethodPost, "/mcp/server", strings.NewReader(`{"id":1}`)))

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.EqualValues(t, 1, refreshRequests.Load())
	assert.EqualValues(t, 2, proxyRequests.Load())
	assert.Equal(t, "refreshed-oauth-token", profile.MCPOAuthAccessToken)
	assert.Equal(t, "new-refresh-token", profile.MCPOAuthRefreshToken)
}

func TestOAuthProxyUnauthorizedPreservesErrors(t *testing.T) {
	for _, tc := range []struct {
		name             string
		firstRefreshOK   bool
		wantStatus       int
		wantBody         string
		wantRefreshCalls int32
	}{
		{"refresh fails", false, http.StatusUnauthorized, "Authentication failed: oauth permanent error", 1},
		{"getting refreshed token fails", true, http.StatusInternalServerError, "Internal Server Error", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setTestHome(t)
			var refreshRequests, proxyRequests atomic.Int32
			oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				call := refreshRequests.Add(1)
				w.Header().Set("Content-Type", "application/json")
				if tc.firstRefreshOK && call == 1 {
					_, _ = io.WriteString(w, `{"access_token":"expired-token","refresh_token":"new-refresh-token","expires_in":0,"token_type":"Bearer"}`)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":"invalid_grant","error_description":"refresh token rejected"}`)
			}))
			defer oauthServer.Close()
			const region RegionType = "oauth-proxy-error-test"
			EndpointMap[region] = EndpointConfig{OAuth: oauthServer.URL}
			defer delete(EndpointMap, region)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				proxyRequests.Add(1)
				w.WriteHeader(http.StatusUnauthorized)
			}))
			defer upstream.Close()
			profile := NewMcpProfile("oauth-error-regression")
			profile.MCPOAuthAccessToken = "existing-oauth-token"
			profile.MCPOAuthAccessTokenExpire = time.Now().Add(time.Hour).Unix()
			profile.MCPOAuthRefreshToken = "existing-refresh-token"
			profile.MCPOAuthRefreshTokenExpire = time.Now().Add(24 * time.Hour).Unix()
			proxy := NewMCPProxy(ProxyConfig{McpProfile: profile, RegionType: region, UpstreamBaseURL: upstream.URL})
			t.Cleanup(proxy.upstreamClient.CloseIdleConnections)
			recorder := httptest.NewRecorder()

			proxy.ServeMCPProxyRequest(recorder, httptest.NewRequest(http.MethodGet, "/mcp/server", nil))

			assert.Equal(t, tc.wantStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), tc.wantBody)
			assert.Equal(t, tc.wantRefreshCalls, refreshRequests.Load())
			assert.EqualValues(t, 1, proxyRequests.Load())
		})
	}
}
