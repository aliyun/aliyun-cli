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

	proxy.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "Server is shutting down")
	assert.Equal(t, int64(1), atomic.LoadInt64(&proxy.stats.ErrorRequests))
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
	assert.Contains(t, upstreamReq.URL.String(), "custom-mcp.example.com")
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

	retrySaveProfile(func() error {
		atomic.AddInt32(&attempts, 1)
		return nil
	}, maxAttempts, func() {
		t.Error("onMaxFailures should not be called")
	})

	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))

	attempts = 0
	onMaxFailuresCalled := false

	retrySaveProfile(func() error {
		atomic.AddInt32(&attempts, 1)
		return assert.AnError
	}, maxAttempts, func() {
		onMaxFailuresCalled = true
	})

	assert.Equal(t, int32(maxAttempts), atomic.LoadInt32(&attempts))
	assert.True(t, onMaxFailuresCalled)
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
