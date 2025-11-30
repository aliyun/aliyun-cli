// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mcpproxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewOAuthCallbackManager(t *testing.T) {
	manager := NewOAuthCallbackManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.pendingAuth)
	assert.NotNil(t, manager.errorCh)
	assert.False(t, manager.isWaiting)
}

func TestOAuthCallbackManager_StartWaiting(t *testing.T) {
	manager := NewOAuthCallbackManager()

	manager.StartWaiting()
	assert.True(t, manager.IsWaiting())

	// 清空channels
	select {
	case <-manager.pendingAuth:
	default:
	}
	select {
	case <-manager.errorCh:
	default:
	}
}

func TestOAuthCallbackManager_StopWaiting(t *testing.T) {
	manager := NewOAuthCallbackManager()
	manager.StartWaiting()
	assert.True(t, manager.IsWaiting())

	manager.StopWaiting()
	assert.False(t, manager.IsWaiting())
}

func TestOAuthCallbackManager_HandleCallback(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*OAuthCallbackManager)
		code          string
		err           error
		expectHandled bool
	}{
		{
			name: "not waiting",
			setup: func(m *OAuthCallbackManager) {
				// 不调用 StartWaiting
			},
			code:          "test-code",
			err:           nil,
			expectHandled: false,
		},
		{
			name: "waiting with code",
			setup: func(m *OAuthCallbackManager) {
				m.StartWaiting()
			},
			code:          "test-code",
			err:           nil,
			expectHandled: true,
		},
		{
			name: "waiting with error",
			setup: func(m *OAuthCallbackManager) {
				m.StartWaiting()
			},
			code:          "",
			err:           assert.AnError,
			expectHandled: true,
		},
		{
			name: "waiting with empty code",
			setup: func(m *OAuthCallbackManager) {
				m.StartWaiting()
			},
			code:          "",
			err:           nil,
			expectHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewOAuthCallbackManager()
			tt.setup(manager)

			handled := manager.HandleCallback(tt.code, tt.err)
			assert.Equal(t, tt.expectHandled, handled)
		})
	}
}

func TestOAuthCallbackManager_WaitForCode(t *testing.T) {
	t.Run("receive code", func(t *testing.T) {
		manager := NewOAuthCallbackManager()
		manager.StartWaiting()

		go func() {
			time.Sleep(10 * time.Millisecond)
			manager.HandleCallback("test-code", nil)
		}()

		code, err := manager.WaitForCode()
		assert.NoError(t, err)
		assert.Equal(t, "test-code", code)
	})

	t.Run("receive error", func(t *testing.T) {
		manager := NewOAuthCallbackManager()
		manager.StartWaiting()

		testErr := assert.AnError
		go func() {
			time.Sleep(10 * time.Millisecond)
			manager.HandleCallback("", testErr)
		}()

		code, err := manager.WaitForCode()
		assert.Error(t, err)
		assert.Empty(t, code)
	})

	t.Run("timeout", func(t *testing.T) {
		// 临时缩短超时时间用于测试
		originalTimeout := oauthTimeout
		oauthTimeout = 100 * time.Millisecond
		defer func() {
			oauthTimeout = originalTimeout
		}()

		manager := NewOAuthCallbackManager()
		manager.StartWaiting()

		code, err := manager.WaitForCode()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
		assert.Empty(t, code)
	})
}

func TestGenerateCodeVerifier(t *testing.T) {
	verifier, err := generateCodeVerifier()
	assert.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.GreaterOrEqual(t, len(verifier), 32) // base64编码后应该至少有32个字符

	// 测试多次生成应该不同
	verifier2, err := generateCodeVerifier()
	assert.NoError(t, err)
	assert.NotEqual(t, verifier, verifier2)
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test-verifier-string-for-pkce"
	challenge := generateCodeChallenge(verifier)

	assert.NotEmpty(t, challenge)
	assert.NotEqual(t, verifier, challenge)

	// 相同verifier应该生成相同challenge
	challenge2 := generateCodeChallenge(verifier)
	assert.Equal(t, challenge, challenge2)

	// 不同verifier应该生成不同challenge
	challenge3 := generateCodeChallenge("different-verifier")
	assert.NotEqual(t, challenge, challenge3)
}

func TestBuildRedirectUri(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "localhost",
			host:     "127.0.0.1",
			port:     8088,
			expected: "http://127.0.0.1:8088/callback",
		},
		{
			name:     "all interfaces",
			host:     "0.0.0.0",
			port:     9000,
			expected: "http://0.0.0.0:9000/callback",
		},
		{
			name:     "custom host",
			host:     "example.com",
			port:     443,
			expected: "http://example.com:443/callback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRedirectUri(tt.host, tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildOAuthURL(t *testing.T) {
	host := "127.0.0.1"
	port := 8088
	codeChallenge := "test-challenge"
	scope := "/acs/mcp-server"

	// 测试CN区域
	urlCN := buildOAuthURL("test-app-id", RegionCN, host, port, codeChallenge, scope)
	assert.Contains(t, urlCN, EndpointMap[RegionCN].SignIn)
	assert.Contains(t, urlCN, "test-app-id")
	assert.Contains(t, urlCN, "test-challenge")
	assert.Contains(t, urlCN, "code_challenge_method=S256")
	assert.Contains(t, urlCN, "response_type=code")

	// 测试INTL区域
	urlINTL := buildOAuthURL("test-app-id", RegionINTL, host, port, codeChallenge, scope)
	assert.Contains(t, urlINTL, EndpointMap[RegionINTL].SignIn)

	// 验证URL可以解析
	parsedURL, err := url.Parse(urlCN)
	assert.NoError(t, err)
	assert.Equal(t, "https", parsedURL.Scheme)
	assert.Equal(t, EndpointMap[RegionCN].SignIn, parsedURL.Scheme+"://"+parsedURL.Host)

	query := parsedURL.Query()
	assert.Equal(t, "test-app-id", query.Get("client_id"))
	assert.Equal(t, "code", query.Get("response_type"))
	assert.Equal(t, codeChallenge, query.Get("code_challenge"))
	assert.Equal(t, "S256", query.Get("code_challenge_method"))
}

func TestHandleOAuthCallbackRequest(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		code          string
		showCode      bool
		expectHandled bool
		expectStatus  int
	}{
		{
			name:          "valid callback with code",
			path:          "/callback",
			code:          "test-code-123",
			showCode:      false,
			expectHandled: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "valid callback with code (show code)",
			path:          "/callback",
			code:          "test-code-456",
			showCode:      true,
			expectHandled: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "callback without code",
			path:          "/callback",
			code:          "",
			showCode:      false,
			expectHandled: true,
			expectStatus:  http.StatusBadRequest,
		},
		{
			name:          "wrong path",
			path:          "/other",
			code:          "test-code",
			showCode:      false,
			expectHandled: false,
			expectStatus:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewOAuthCallbackManager()
			manager.StartWaiting()

			req := httptest.NewRequest("GET", tt.path+"?code="+tt.code, nil)
			w := httptest.NewRecorder()

			handled := handleOAuthCallbackRequest(w, req, manager.HandleCallback, tt.showCode)
			assert.Equal(t, tt.expectHandled, handled)
			assert.Equal(t, tt.expectStatus, w.Code)

			if tt.expectHandled && tt.code != "" {
				select {
				case code := <-manager.pendingAuth:
					assert.Equal(t, tt.code, code)
				case <-time.After(100 * time.Millisecond):
					t.Error("code not received in channel")
				}
			}
		})
	}
}

func TestOAuthRefresh(t *testing.T) {
	t.Run("successful refresh", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/v1/token", r.URL.Path)
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"access_token": "new-access-token",
				"refresh_token": "new-refresh-token",
				"expires_in": 3600,
				"refresh_expires_in": 86400,
				"token_type": "Bearer"
			}`))
		}))
		defer server.Close()

		data := url.Values{}
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", "old-refresh-token")

		tokenResp, err := oauthRefresh(server.URL, data)
		assert.NoError(t, err)
		assert.NotNil(t, tokenResp)
		assert.Equal(t, "new-access-token", tokenResp.AccessToken)
		assert.Equal(t, "new-refresh-token", tokenResp.RefreshToken)
		assert.Equal(t, int64(3600), tokenResp.ExpiresIn)
		assert.Equal(t, int64(86400), tokenResp.RefreshExpiresIn)
		assert.Equal(t, "Bearer", tokenResp.TokenType)
	})

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{
				"error": "invalid_grant",
				"error_description": "Refresh token expired"
			}`))
		}))
		defer server.Close()

		data := url.Values{}
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", "expired-token")

		tokenResp, err := oauthRefresh(server.URL, data)
		assert.Error(t, err)
		assert.Nil(t, tokenResp)
	})

	t.Run("non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		data := url.Values{}
		tokenResp, err := oauthRefresh(server.URL, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
		assert.Nil(t, tokenResp)
	})
}

func TestExchangeCodeForTokenWithPKCE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		assert.NoError(t, err)
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.Equal(t, "test-code", r.Form.Get("code"))
		assert.Equal(t, "test-app-id", r.Form.Get("client_id"))
		assert.Equal(t, "http://127.0.0.1:8088/callback", r.Form.Get("redirect_uri"))
		assert.Equal(t, "test-verifier", r.Form.Get("code_verifier"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"access_token": "access-token",
			"refresh_token": "refresh-token",
			"expires_in": 3600,
			"token_type": "Bearer"
		}`))
	}))
	defer server.Close()

	tokenResult, err := exchangeCodeForTokenWithPKCE("test-app-id", "test-code", "test-verifier", "http://127.0.0.1:8088/callback", server.URL)
	assert.NoError(t, err)
	assert.Equal(t, "access-token", tokenResult.AccessToken)
	assert.Equal(t, "refresh-token", tokenResult.RefreshToken)
	assert.NotZero(t, tokenResult.AccessTokenExpire)
}

func TestRegionTypeConstants(t *testing.T) {
	assert.Equal(t, RegionType("CN"), RegionCN)
	assert.Equal(t, RegionType("INTL"), RegionINTL)
	assert.Equal(t, "default-mcp", DefaultMcpProfileName)
}

func TestEndpointMap(t *testing.T) {
	assert.NotNil(t, EndpointMap[RegionCN])
	assert.NotNil(t, EndpointMap[RegionINTL])

	assert.NotEmpty(t, EndpointMap[RegionCN].SignIn)
	assert.NotEmpty(t, EndpointMap[RegionCN].OAuth)
	assert.NotEmpty(t, EndpointMap[RegionCN].IMS)
	assert.NotEmpty(t, EndpointMap[RegionCN].MCP)

	assert.NotEmpty(t, EndpointMap[RegionINTL].SignIn)
	assert.NotEmpty(t, EndpointMap[RegionINTL].OAuth)
	assert.NotEmpty(t, EndpointMap[RegionINTL].IMS)
	assert.NotEmpty(t, EndpointMap[RegionINTL].MCP)

	assert.True(t, strings.HasPrefix(EndpointMap[RegionCN].SignIn, "https://"))
	assert.True(t, strings.HasPrefix(EndpointMap[RegionCN].OAuth, "https://"))
	assert.True(t, strings.HasPrefix(EndpointMap[RegionINTL].SignIn, "https://"))
	assert.True(t, strings.HasPrefix(EndpointMap[RegionINTL].OAuth, "https://"))
}
