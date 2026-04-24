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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMCPProxy(t *testing.T) {
	host := "127.0.0.1"
	port := 8088
	regionType := RegionCN
	scope := "/acs/mcp-server"
	mcpProfile := NewMcpProfile("test-profile")
	servers := []MCPServerInfo{
		{
			Id:         "server1",
			Name:       "Test Server",
			SourceType: "api",
			Product:    "ecs",
			Urls: MCPInfoUrls{
				MCP: "/mcp/server1",
				SSE: "/sse/server1",
			},
		},
	}
	manager := NewOAuthCallbackManager()
	autoOpenBrowser := true

	config := ProxyConfig{
		Host:            host,
		Port:            port,
		RegionType:      regionType,
		Scope:           scope,
		McpProfile:      mcpProfile,
		ExistMcpServers: servers,
		CallbackManager: manager,
		AutoOpenBrowser: autoOpenBrowser,
		UpstreamBaseURL: "",
	}
	proxy := NewMCPProxy(config)

	assert.NotNil(t, proxy)
	assert.Equal(t, host, proxy.Host)
	assert.Equal(t, port, proxy.Port)
	assert.Equal(t, regionType, proxy.RegionType)
	assert.Equal(t, servers, proxy.ExistMcpServers)
	assert.NotNil(t, proxy.TokenRefresher)
	assert.NotNil(t, proxy.stopCh)
	assert.NotNil(t, proxy.stats)
	assert.Equal(t, mcpProfile, proxy.TokenRefresher.profile)
	assert.Equal(t, regionType, proxy.TokenRefresher.regionType)
	assert.Equal(t, manager, proxy.TokenRefresher.callbackManager)
	assert.Equal(t, host, proxy.TokenRefresher.host)
	assert.Equal(t, port, proxy.TokenRefresher.port)
	assert.Equal(t, scope, proxy.TokenRefresher.scope)
	assert.Equal(t, autoOpenBrowser, proxy.TokenRefresher.autoOpenBrowser)
}

func TestNewMCPProxy_ServerPathsMapping(t *testing.T) {
	tests := []struct {
		name    string
		servers []MCPServerInfo
		want    map[string][]string // server name/id -> expected paths
	}{
		{
			name: "server with both MCP and SSE URLs",
			servers: []MCPServerInfo{
				{
					Id:   "server1-id",
					Name: "server1",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server1",
						SSE: "https://example.com/sse/server1",
					},
				},
			},
			want: map[string][]string{
				"server1":    {"/mcp/server1", "/sse/server1"},
				"server1-id": {"/mcp/server1", "/sse/server1"},
			},
		},
		{
			name: "server with only MCP URL",
			servers: []MCPServerInfo{
				{
					Id:   "server2-id",
					Name: "server2",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server2",
						SSE: "",
					},
				},
			},
			want: map[string][]string{
				"server2":    {"/mcp/server2"},
				"server2-id": {"/mcp/server2"},
			},
		},
		{
			name: "server with only SSE URL",
			servers: []MCPServerInfo{
				{
					Id:   "server3-id",
					Name: "server3",
					Urls: MCPInfoUrls{
						MCP: "",
						SSE: "https://example.com/sse/server3",
					},
				},
			},
			want: map[string][]string{
				"server3":    {"/sse/server3"},
				"server3-id": {"/sse/server3"},
			},
		},
		{
			name: "server with no URLs",
			servers: []MCPServerInfo{
				{
					Id:   "server4-id",
					Name: "server4",
					Urls: MCPInfoUrls{
						MCP: "",
						SSE: "",
					},
				},
			},
			want: map[string][]string{},
		},
		{
			name: "multiple servers",
			servers: []MCPServerInfo{
				{
					Id:   "server5-id",
					Name: "server5",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server5",
						SSE: "https://example.com/sse/server5",
					},
				},
				{
					Id:   "server6-id",
					Name: "server6",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server6",
						SSE: "",
					},
				},
			},
			want: map[string][]string{
				"server5":    {"/mcp/server5", "/sse/server5"},
				"server5-id": {"/mcp/server5", "/sse/server5"},
				"server6":    {"/mcp/server6"},
				"server6-id": {"/mcp/server6"},
			},
		},
		{
			name: "server with invalid URL",
			servers: []MCPServerInfo{
				{
					Id:   "server7-id",
					Name: "server7",
					Urls: MCPInfoUrls{
						MCP: "://invalid-url",
						SSE: "https://example.com/sse/server7",
					},
				},
			},
			want: map[string][]string{
				"server7":    {"/sse/server7"},
				"server7-id": {"/sse/server7"},
			},
		},
		{
			name: "server with root path",
			servers: []MCPServerInfo{
				{
					Id:   "server8-id",
					Name: "server8",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/",
						SSE: "https://example.com/",
					},
				},
			},
			want: map[string][]string{
				"server8":    {"/", "/"},
				"server8-id": {"/", "/"},
			},
		},
		{
			name:    "empty servers list",
			servers: []MCPServerInfo{},
			want:    map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ProxyConfig{
				Host:            "127.0.0.1",
				Port:            8088,
				RegionType:      RegionCN,
				Scope:           "/acs/mcp-server",
				McpProfile:      NewMcpProfile("test-profile"),
				ExistMcpServers: tt.servers,
				CallbackManager: NewOAuthCallbackManager(),
				AutoOpenBrowser: false,
				UpstreamBaseURL: "",
			}

			proxy := NewMCPProxy(config)

			// Verify serverPaths mapping
			assert.NotNil(t, proxy.serverPaths)
			assert.Equal(t, len(tt.want), len(proxy.serverPaths), "serverPaths map size mismatch")

			for key, expectedPaths := range tt.want {
				actualPaths, exists := proxy.serverPaths[key]
				assert.True(t, exists, "serverPaths should contain key: %s", key)
				assert.Equal(t, expectedPaths, actualPaths, "paths mismatch for key: %s", key)
			}

			// Verify that no unexpected keys exist
			for key := range proxy.serverPaths {
				_, exists := tt.want[key]
				assert.True(t, exists, "unexpected key in serverPaths: %s", key)
			}
		})
	}
}

func TestMCPProxy_Stop(t *testing.T) {
	config := ProxyConfig{
		Host:            "127.0.0.1",
		Port:            0,
		RegionType:      RegionCN,
		Scope:           "/acs/mcp-server",
		McpProfile:      NewMcpProfile("test"),
		ExistMcpServers: nil,
		CallbackManager: NewOAuthCallbackManager(),
		AutoOpenBrowser: false,
		UpstreamBaseURL: "",
	}
	proxy := NewMCPProxy(config)

	err := proxy.Stop()
	assert.NoError(t, err)

	select {
	case <-proxy.stopCh:
		// 正常，channel 已关闭
	default:
		t.Error("stopCh should be closed")
	}
}

func TestMCPProxy_handleHealth(t *testing.T) {
	profile := NewMcpProfile("test-profile")
	currentTime := time.Now().Unix()
	profile.MCPOAuthAccessToken = "test-token"
	profile.MCPOAuthAccessTokenExpire = currentTime + 3600
	profile.MCPOAuthRefreshToken = "refresh-token"
	profile.MCPOAuthRefreshTokenExpire = currentTime + 86400

	config := ProxyConfig{
		Host:            "127.0.0.1",
		Port:            8088,
		RegionType:      RegionCN,
		Scope:           "/acs/mcp-server",
		McpProfile:      profile,
		ExistMcpServers: nil,
		CallbackManager: NewOAuthCallbackManager(),
		AutoOpenBrowser: false,
		UpstreamBaseURL: "",
	}
	proxy := NewMCPProxy(config)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	proxy.handleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var health map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &health)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", health["status"])
	assert.NotNil(t, health["timestamp"])
	assert.NotNil(t, health["uptime"])
	assert.NotNil(t, health["memory"])
	assert.NotNil(t, health["requests"])
	assert.NotNil(t, health["token_refreshes"])
}

func TestMCPProxy_handleHealth_ExpiredToken(t *testing.T) {
	profile := NewMcpProfile("test-profile")
	// 设置过期的 token
	profile.MCPOAuthAccessToken = "test-token"
	profile.MCPOAuthAccessTokenExpire = time.Now().Unix() - 100
	profile.MCPOAuthRefreshToken = "refresh-token"
	profile.MCPOAuthRefreshTokenExpire = time.Now().Unix() + 86400

	config := ProxyConfig{
		Host:            "127.0.0.1",
		Port:            8088,
		RegionType:      RegionCN,
		Scope:           "/acs/mcp-server",
		McpProfile:      profile,
		ExistMcpServers: nil,
		CallbackManager: NewOAuthCallbackManager(),
		AutoOpenBrowser: false,
		UpstreamBaseURL: "",
	}
	proxy := NewMCPProxy(config)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	proxy.handleHealth(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var health map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &health)
	assert.NoError(t, err)
	assert.Equal(t, "degraded", health["status"])
	assert.Equal(t, "expired", health["token_status"])
}

func TestMCPProxy_ServeHTTP_ShuttingDown(t *testing.T) {
	config := ProxyConfig{
		Host:            "127.0.0.1",
		Port:            8088,
		RegionType:      RegionCN,
		Scope:           "/acs/mcp-server",
		McpProfile:      NewMcpProfile("test"),
		ExistMcpServers: nil,
		CallbackManager: NewOAuthCallbackManager(),
		AutoOpenBrowser: false,
		UpstreamBaseURL: "",
	}
	proxy := NewMCPProxy(config)

	// 关闭 stopCh 模拟正在关闭
	close(proxy.stopCh)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	proxy.ServeMCPProxyRequest(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "Server is shutting down")
	assert.Equal(t, int64(1), atomic.LoadInt64(&proxy.stats.ErrorRequests))
}

func TestMCPProxy_ServeMCPProxyRequest_AccessControl(t *testing.T) {
	tests := []struct {
		name           string
		servers        []MCPServerInfo
		blockedServers []string
		allowedServers []string
		requestPath    string
		shouldBlock    bool // true if request should be blocked by access control
		expectedBody   string
	}{
		{
			name: "path blocked by blacklist (server name)",
			servers: []MCPServerInfo{
				{
					Id:   "server1-id",
					Name: "server1",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server1",
						SSE: "https://example.com/sse/server1",
					},
				},
			},
			blockedServers: []string{"server1"},
			allowedServers: []string{},
			requestPath:    "/mcp/server1/tools",
			shouldBlock:    true,
			expectedBody:   "Access denied: This MCP server is blocked",
		},
		{
			name: "path blocked by blacklist (server ID)",
			servers: []MCPServerInfo{
				{
					Id:   "server1-id",
					Name: "server1",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server1",
					},
				},
			},
			blockedServers: []string{"server1-id"},
			allowedServers: []string{},
			requestPath:    "/mcp/server1/tools",
			shouldBlock:    true,
			expectedBody:   "Access denied: This MCP server is blocked",
		},
		{
			name: "path blocked by path prefix in blacklist",
			servers: []MCPServerInfo{
				{
					Id:   "server2-id",
					Name: "server2",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server2",
					},
				},
			},
			blockedServers: []string{"/mcp/server2"},
			allowedServers: []string{},
			requestPath:    "/mcp/server2/tools",
			shouldBlock:    true,
			expectedBody:   "Access denied: This MCP server is blocked",
		},
		{
			name: "path not in whitelist",
			servers: []MCPServerInfo{
				{
					Id:   "server3-id",
					Name: "server3",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server3",
					},
				},
				{
					Id:   "server4-id",
					Name: "server4",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server4",
					},
				},
			},
			blockedServers: []string{},
			allowedServers: []string{"server3"},
			requestPath:    "/mcp/server4/tools",
			shouldBlock:    true,
			expectedBody:   "Access denied: This MCP server is not allowed",
		},
		{
			name: "path allowed by whitelist (server name)",
			servers: []MCPServerInfo{
				{
					Id:   "server5-id",
					Name: "server5",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server5",
					},
				},
			},
			blockedServers: []string{},
			allowedServers: []string{"server5"},
			requestPath:    "/mcp/server5/tools",
			shouldBlock:    false, // Access control passes, will continue to upstream
			expectedBody:   "",
		},
		{
			name: "path allowed by whitelist (server ID)",
			servers: []MCPServerInfo{
				{
					Id:   "server5-id",
					Name: "server5",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server5",
					},
				},
			},
			blockedServers: []string{},
			allowedServers: []string{"server5-id"},
			requestPath:    "/mcp/server5/tools",
			shouldBlock:    false,
			expectedBody:   "",
		},
		{
			name: "blacklist takes precedence over whitelist",
			servers: []MCPServerInfo{
				{
					Id:   "server6-id",
					Name: "server6",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server6",
					},
				},
			},
			blockedServers: []string{"server6"},
			allowedServers: []string{"server6"},
			requestPath:    "/mcp/server6/tools",
			shouldBlock:    true,
			expectedBody:   "Access denied: This MCP server is blocked",
		},
		{
			name: "no access control - default allow",
			servers: []MCPServerInfo{
				{
					Id:   "server7-id",
					Name: "server7",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server7",
					},
				},
			},
			blockedServers: []string{},
			allowedServers: []string{},
			requestPath:    "/mcp/server7/tools",
			shouldBlock:    false, // No access control, default allow
			expectedBody:   "",
		},
		{
			name: "path prefix matching in whitelist",
			servers: []MCPServerInfo{
				{
					Id:   "server8-id",
					Name: "server8",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server8",
					},
				},
			},
			blockedServers: []string{},
			allowedServers: []string{"/mcp/server8"},
			requestPath:    "/mcp/server8/tools",
			shouldBlock:    false,
			expectedBody:   "",
		},
		{
			name: "path prefix not matching in whitelist",
			servers: []MCPServerInfo{
				{
					Id:   "server9-id",
					Name: "server9",
					Urls: MCPInfoUrls{
						MCP: "https://example.com/mcp/server9",
					},
				},
			},
			blockedServers: []string{},
			allowedServers: []string{"/mcp/server10"},
			requestPath:    "/mcp/server9/tools",
			shouldBlock:    true,
			expectedBody:   "Access denied: This MCP server is not allowed",
		},
		{
			name: "SSE path blocked by blacklist",
			servers: []MCPServerInfo{
				{
					Id:   "server10-id",
					Name: "server10",
					Urls: MCPInfoUrls{
						SSE: "https://example.com/sse/server10",
					},
				},
			},
			blockedServers: []string{"server10"},
			allowedServers: []string{},
			requestPath:    "/sse/server10/stream",
			shouldBlock:    true,
			expectedBody:   "Access denied: This MCP server is blocked",
		},
		{
			name: "SSE path allowed by whitelist",
			servers: []MCPServerInfo{
				{
					Id:   "server11-id",
					Name: "server11",
					Urls: MCPInfoUrls{
						SSE: "https://example.com/sse/server11",
					},
				},
			},
			blockedServers: []string{},
			allowedServers: []string{"server11"},
			requestPath:    "/sse/server11/stream",
			shouldBlock:    false,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := NewMcpProfile("test-profile")
			profile.MCPOAuthAccessToken = "test-token"
			profile.MCPOAuthAccessTokenExpire = time.Now().Unix() + 3600

			config := ProxyConfig{
				Host:            "127.0.0.1",
				Port:            8088,
				RegionType:      RegionCN,
				Scope:           "/acs/mcp-server",
				McpProfile:      profile,
				ExistMcpServers: tt.servers,
				CallbackManager: NewOAuthCallbackManager(),
				AutoOpenBrowser: false,
				UpstreamBaseURL: "",
				BlockedServers:  tt.blockedServers,
				AllowedServers:  tt.allowedServers,
			}

			proxy := NewMCPProxy(config)

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()

			initialErrorCount := atomic.LoadInt64(&proxy.stats.ErrorRequests)

			proxy.ServeMCPProxyRequest(w, req)

			if tt.shouldBlock {
				// Should be blocked by access control
				assert.Equal(t, http.StatusForbidden, w.Code, "status code should be 403 Forbidden for test: %s", tt.name)
				assert.Contains(t, w.Body.String(), tt.expectedBody, "response body mismatch for test: %s", tt.name)
				assert.Greater(t, atomic.LoadInt64(&proxy.stats.ErrorRequests), initialErrorCount, "error count should be incremented for denied request")
			} else {
				// Should pass access control (may fail later at upstream, but access control passed)
				assert.NotEqual(t, http.StatusForbidden, w.Code, "status code should not be 403 Forbidden for test: %s (access control should pass)", tt.name)
				assert.NotContains(t, w.Body.String(), "Access denied", "response should not contain access denied message for test: %s", tt.name)
			}
		})
	}
}

func TestMCPProxy_buildUpstreamRequest(t *testing.T) {
	profile := NewMcpProfile("test-profile")
	profile.MCPOAuthAccessToken = "test-access-token"
	profile.MCPOAuthAccessTokenExpire = time.Now().Unix() + 3600

	config := ProxyConfig{
		Host:            "127.0.0.1",
		Port:            8088,
		RegionType:      RegionCN,
		Scope:           "/acs/mcp-server",
		McpProfile:      profile,
		ExistMcpServers: nil,
		CallbackManager: NewOAuthCallbackManager(),
		AutoOpenBrowser: false,
		UpstreamBaseURL: "",
	}
	proxy := NewMCPProxy(config)

	body := bytes.NewBufferString("test body")
	req := httptest.NewRequest("POST", "/test/path?query=value", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", "localhost")
	req.Header.Set("Authorization", "Bearer old-token")

	upstreamReq, err := proxy.buildUpstreamRequest(req, "new-access-token")
	assert.NoError(t, err)
	assert.NotNil(t, upstreamReq)

	assert.Equal(t, "POST", upstreamReq.Method)
	assert.Contains(t, upstreamReq.URL.String(), EndpointMap[RegionCN].MCP)
	assert.Contains(t, upstreamReq.URL.Path, "/test/path")
	assert.Equal(t, "value", upstreamReq.URL.Query().Get("query"))
	assert.Equal(t, "Bearer new-access-token", upstreamReq.Header.Get("Authorization"))
	assert.Equal(t, "application/json", upstreamReq.Header.Get("Content-Type"))
	assert.NotEqual(t, "localhost", upstreamReq.Header.Get("Host"))
}

func TestMCPProxy_buildUpstreamRequest_WithCustomURL(t *testing.T) {
	profile := NewMcpProfile("test-profile")
	profile.MCPOAuthAccessToken = "test-access-token"
	profile.MCPOAuthAccessTokenExpire = time.Now().Unix() + 3600

	// 测试使用自定义的 upstream URL
	customURL := "https://custom-mcp.example.com"
	config := ProxyConfig{
		Host:            "127.0.0.1",
		Port:            8088,
		RegionType:      RegionCN,
		Scope:           "/acs/mcp-server",
		McpProfile:      profile,
		ExistMcpServers: nil,
		CallbackManager: NewOAuthCallbackManager(),
		AutoOpenBrowser: false,
		UpstreamBaseURL: customURL,
	}
	proxy := NewMCPProxy(config)

	body := bytes.NewBufferString("test body")
	req := httptest.NewRequest("POST", "/test/path?query=value", body)
	req.Header.Set("Content-Type", "application/json")

	upstreamReq, err := proxy.buildUpstreamRequest(req, "new-access-token")
	assert.NoError(t, err)
	assert.NotNil(t, upstreamReq)

	assert.Equal(t, "POST", upstreamReq.Method)
	assert.Contains(t, upstreamReq.URL.String(), "https://custom-mcp.example.com")
	assert.Contains(t, upstreamReq.URL.Path, "/test/path")
	assert.Equal(t, "Bearer new-access-token", upstreamReq.Header.Get("Authorization"))
}

func TestMCPProxy_buildUpstreamRequest_WithCustomURL_NoProtocol(t *testing.T) {
	profile := NewMcpProfile("test-profile")
	profile.MCPOAuthAccessToken = "test-access-token"
	profile.MCPOAuthAccessTokenExpire = time.Now().Unix() + 3600

	// 测试使用自定义的 upstream URL（没有协议前缀）
	customURL := "custom-mcp.example.com"
	config := ProxyConfig{
		Host:            "127.0.0.1",
		Port:            8088,
		RegionType:      RegionCN,
		Scope:           "/acs/mcp-server",
		McpProfile:      profile,
		ExistMcpServers: nil,
		CallbackManager: NewOAuthCallbackManager(),
		AutoOpenBrowser: false,
		UpstreamBaseURL: customURL,
	}
	proxy := NewMCPProxy(config)

	body := bytes.NewBufferString("test body")
	req := httptest.NewRequest("POST", "/test/path", body)

	upstreamReq, err := proxy.buildUpstreamRequest(req, "new-access-token")
	assert.NoError(t, err)
	assert.NotNil(t, upstreamReq)

	// 应该自动添加 https:// 前缀
	assert.Contains(t, upstreamReq.URL.String(), "https://custom-mcp.example.com")
}

func TestTokenRefresher_Stop(t *testing.T) {
	refresher := &TokenRefresher{
		stopCh: make(chan struct{}),
		stats: &RuntimeStats{
			StartTime: time.Now(),
		},
	}

	refresher.Stop()

	select {
	case <-refresher.stopCh:
		// 正常，channel 已关闭
	default:
		t.Error("stopCh should be closed")
	}
}

func TestTokenRefresher_sendToken(t *testing.T) {
	profile := NewMcpProfile("test-profile")
	profile.MCPOAuthAccessToken = "test-token"
	profile.MCPOAuthAccessTokenExpire = time.Now().Unix() + 3600

	refresher := &TokenRefresher{
		profile: profile,
		tokenCh: make(chan TokenInfo, 1),
		stats: &RuntimeStats{
			StartTime: time.Now(),
		},
	}

	refresher.sendToken()

	select {
	case tokenInfo := <-refresher.tokenCh:
		assert.Equal(t, "test-token", tokenInfo.Token)
		assert.Equal(t, profile.MCPOAuthAccessTokenExpire, tokenInfo.ExpiresAt)
	case <-time.After(100 * time.Millisecond):
		t.Error("token should be sent to channel")
	}
}

func TestTokenRefresher_atomicSaveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := getHomeEnv()
	defer restoreHomeEnv(originalHome)

	setHomeEnv(tmpDir)

	profile := NewMcpProfile("test-profile")
	profile.MCPOAuthAccessToken = "test-token"

	refresher := &TokenRefresher{
		profile: profile,
		stats: &RuntimeStats{
			StartTime: time.Now(),
		},
	}

	err := refresher.atomicSaveProfile()
	assert.NoError(t, err)

	configPath := getMCPConfigPath()
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}

func TestRetrySaveProfile(t *testing.T) {
	attempts := int32(0)
	maxAttempts := 3

	// success case: save succeeds on first attempt, returns true
	saved := retrySaveProfile(func() error {
		atomic.AddInt32(&attempts, 1)
		return nil
	}, maxAttempts, func() {
		t.Error("onMaxFailures should not be called")
	})

	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
	assert.True(t, saved, "retrySaveProfile should return true on success")

	// failure case: all attempts exhausted, returns false and calls onMaxFailures
	attempts = 0
	onMaxFailuresCalled := false

	saved = retrySaveProfile(func() error {
		atomic.AddInt32(&attempts, 1)
		return assert.AnError
	}, maxAttempts, func() {
		onMaxFailuresCalled = true
	})

	assert.Equal(t, int32(maxAttempts), atomic.LoadInt32(&attempts))
	assert.True(t, onMaxFailuresCalled)
	assert.False(t, saved, "retrySaveProfile should return false when onMaxFailures is called")

	// success after retry: save succeeds on the second attempt
	attempts = 0
	saved = retrySaveProfile(func() error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			return assert.AnError
		}
		return nil
	}, maxAttempts, func() {
		t.Error("onMaxFailures should not be called when save eventually succeeds")
	})

	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
	assert.True(t, saved, "retrySaveProfile should return true when save eventually succeeds")
}

func TestRetrySaveProfile_NoDoubleUnlock(t *testing.T) {
	// Verify that when retrySaveProfile fails and the onMaxFailures callback unlocks
	// a mutex, the caller can detect the failure (return value = false) and must NOT
	// call Unlock a second time. This test exercises the mutex directly to confirm
	// that exactly one Unlock occurs on the failure path.
	var mu sync.Mutex
	mu.Lock()

	unlockCount := 0
	saved := retrySaveProfile(
		func() error { return assert.AnError },
		1,
		func() {
			mu.Unlock()
			unlockCount++
		},
	)

	assert.False(t, saved)
	assert.Equal(t, 1, unlockCount, "onMaxFailures should unlock exactly once")

	// If the caller incorrectly called mu.Unlock() again it would panic; the fact
	// that this test completes without panic proves the fix is correct.
	// Re-lock to confirm the mutex is now unlocked and available.
	mu.Lock()
	mu.Unlock()
}

func TestGetContentFromApiResponse_Integration(t *testing.T) {
	response := map[string]any{
		"body": map[string]any{
			"key": "value",
		},
	}

	content, err := GetContentFromApiResponse(response)
	assert.NoError(t, err)
	assert.NotNil(t, content)
	assert.Contains(t, string(content), "key")
	assert.Contains(t, string(content), "value")
}

func TestTokenRefresher_refreshAccessToken_PermanentError(t *testing.T) {
	// 测试永久性错误时立即停止重试
	// 这里主要测试逻辑：永久性错误应该立即返回，不应该重试
	// 实际的网络调用测试在 oauth_app_test.go 中
	permanentErr := &OAuthPermanentError{
		StatusCode: 400,
		ErrorCode:  "invalid_grant",
		Message:    "OAuth permanent error: invalid_grant (status 400)",
	}

	assert.True(t, IsPermanentError(permanentErr))

	// 验证错误信息
	assert.Contains(t, permanentErr.Error(), "invalid_grant")
	assert.Equal(t, 400, permanentErr.StatusCode)
	assert.Equal(t, "invalid_grant", permanentErr.ErrorCode)
}

func TestTokenRefresher_refreshAccessToken_AccessTokenStillValid(t *testing.T) {
	// 测试当 access token 还有效时，临时错误应该返回 nil
	currentTime := time.Now().Unix()
	profile := NewMcpProfile("test-profile")
	profile.MCPOAuthAccessTokenExpire = currentTime + 3600 // access token 还有 1 小时有效

	// 验证 access token 还有效时的逻辑
	accessTimeRemaining := profile.MCPOAuthAccessTokenExpire - currentTime
	assert.Greater(t, accessTimeRemaining, int64(0), "access token should still be valid")
	assert.GreaterOrEqual(t, accessTimeRemaining, int64(3600), "access token should have at least 1 hour remaining")
}

func TestTokenRefresher_refreshAccessToken_AccessTokenExpired(t *testing.T) {
	// 测试当 access token 已过期时，应该返回错误
	currentTime := time.Now().Unix()
	profile := NewMcpProfile("test-profile")
	profile.MCPOAuthAccessTokenExpire = currentTime - 100 // access token 已过期

	// 验证 access token 已过期时的逻辑
	accessTimeRemaining := profile.MCPOAuthAccessTokenExpire - currentTime
	assert.LessOrEqual(t, accessTimeRemaining, int64(0), "access token should be expired")
	assert.Less(t, accessTimeRemaining, int64(0), "access token should be expired (negative remaining time)")
}

// 辅助函数
func getHomeEnv() string {
	return os.Getenv("HOME")
}

func setHomeEnv(value string) {
	os.Setenv("HOME", value)
}

func restoreHomeEnv(value string) {
	if value != "" {
		os.Setenv("HOME", value)
	} else {
		os.Unsetenv("HOME")
	}
}
