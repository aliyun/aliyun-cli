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
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/assert"
)

func TestNewMcpProfile(t *testing.T) {
	name := "test-profile"
	profile := NewMcpProfile(name)

	assert.NotNil(t, profile)
	assert.Equal(t, name, profile.Name)
	assert.Empty(t, profile.MCPOAuthAccessToken)
	assert.Empty(t, profile.MCPOAuthRefreshToken)
	assert.Zero(t, profile.MCPOAuthAccessTokenExpire)
	assert.Zero(t, profile.MCPOAuthRefreshTokenValidity)
	assert.Zero(t, profile.MCPOAuthRefreshTokenExpire)
	assert.Empty(t, profile.MCPOAuthSiteType)
	assert.Empty(t, profile.MCPOAuthAppId)
}

func TestNewMcpProfileFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		jsonBytes []byte
		wantErr   bool
		validate  func(t *testing.T, profile *McpProfile)
	}{
		{
			name: "valid profile",
			jsonBytes: []byte(`{
				"name": "test-profile",
				"mcp_oauth_access_token": "test-access-token",
				"mcp_oauth_refresh_token": "test-refresh-token",
				"mcp_oauth_access_token_expire": 1234567890,
				"mcp_oauth_refresh_token_validity": 31536000,
				"mcp_oauth_refresh_token_expire": 1234567890,
				"mcp_oauth_site_type": "CN",
				"mcp_oauth_app_id": "test-app-id"
			}`),
			wantErr: false,
			validate: func(t *testing.T, profile *McpProfile) {
				assert.Equal(t, "test-profile", profile.Name)
				assert.Equal(t, "test-access-token", profile.MCPOAuthAccessToken)
				assert.Equal(t, "test-refresh-token", profile.MCPOAuthRefreshToken)
				assert.Equal(t, int64(1234567890), profile.MCPOAuthAccessTokenExpire)
				assert.Equal(t, 31536000, profile.MCPOAuthRefreshTokenValidity)
				assert.Equal(t, int64(1234567890), profile.MCPOAuthRefreshTokenExpire)
				assert.Equal(t, "CN", profile.MCPOAuthSiteType)
				assert.Equal(t, "test-app-id", profile.MCPOAuthAppId)
			},
		},
		{
			name: "minimal profile",
			jsonBytes: []byte(`{
				"name": "minimal-profile"
			}`),
			wantErr: false,
			validate: func(t *testing.T, profile *McpProfile) {
				assert.Equal(t, "minimal-profile", profile.Name)
			},
		},
		{
			name:      "invalid json",
			jsonBytes: []byte(`{invalid json}`),
			wantErr:   true,
			validate: func(t *testing.T, profile *McpProfile) {
				// JSON 解析失败时，profile 应该为 nil
				assert.Nil(t, profile)
			},
		},
		{
			name:      "empty bytes",
			jsonBytes: []byte{},
			wantErr:   true,
			validate: func(t *testing.T, profile *McpProfile) {
				// JSON 解析失败时，profile 应该为 nil
				assert.Nil(t, profile)
			},
		},
		{
			name: "empty name remains empty",
			jsonBytes: []byte(`{
				"mcp_oauth_access_token": "test-token"
			}`),
			wantErr: false,
			validate: func(t *testing.T, profile *McpProfile) {
				assert.NotNil(t, profile)
				assert.Empty(t, profile.Name)
				assert.Equal(t, "test-token", profile.MCPOAuthAccessToken)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := NewMcpProfileFromBytes(tt.jsonBytes)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, profile)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, profile)
			}
			if tt.validate != nil {
				tt.validate(t, profile)
			}
		})
	}
}

func TestSaveMcpProfile(t *testing.T) {
	tmpDir := t.TempDir()

	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	// 设置 HOME 环境变量指向临时目录，这样 GetConfigPath() 会返回 tmpDir/.aliyun
	os.Setenv("HOME", tmpDir)

	configPath := getMCPConfigPath()
	testConfigPath := filepath.Join(tmpDir, ".aliyun", ".mcpproxy_config")
	assert.Equal(t, testConfigPath, configPath)

	profile := NewMcpProfile("test-profile")
	profile.MCPOAuthAccessToken = "test-token"
	profile.MCPOAuthAppId = "test-app-id"
	profile.MCPOAuthSiteType = "CN"

	err := saveMcpProfile(profile)
	assert.NoError(t, err)

	// 验证文件是否存在
	_, err = os.Stat(testConfigPath)
	assert.NoError(t, err)

	// 读取并验证内容
	bytes, err := os.ReadFile(testConfigPath)
	assert.NoError(t, err)

	var loadedProfile McpProfile
	err = json.Unmarshal(bytes, &loadedProfile)
	assert.NoError(t, err)
	assert.Equal(t, profile.Name, loadedProfile.Name)
	assert.Equal(t, profile.MCPOAuthAccessToken, loadedProfile.MCPOAuthAccessToken)
	assert.Equal(t, profile.MCPOAuthAppId, loadedProfile.MCPOAuthAppId)
	assert.Equal(t, profile.MCPOAuthSiteType, loadedProfile.MCPOAuthSiteType)
}

func TestMcpProfileRegionType(t *testing.T) {
	t.Run("region type is saved and loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()

		os.Setenv("HOME", tmpDir)

		profile := NewMcpProfile("test-profile")
		profile.MCPOAuthSiteType = string(RegionCN)
		profile.MCPOAuthAppId = "test-app-id"

		err := saveMcpProfile(profile)
		assert.NoError(t, err)

		configPath := getMCPConfigPath()
		bytes, err := os.ReadFile(configPath)
		assert.NoError(t, err)

		var loadedProfile McpProfile
		err = json.Unmarshal(bytes, &loadedProfile)
		assert.NoError(t, err)
		assert.Equal(t, string(RegionCN), loadedProfile.MCPOAuthSiteType)
	})

	t.Run("region type comparison", func(t *testing.T) {
		tests := []struct {
			name            string
			savedRegion     string
			requestedRegion RegionType
			shouldMatch     bool
		}{
			{
				name:            "CN matches CN",
				savedRegion:     "CN",
				requestedRegion: RegionCN,
				shouldMatch:     true,
			},
			{
				name:            "INTL matches INTL",
				savedRegion:     "INTL",
				requestedRegion: RegionINTL,
				shouldMatch:     true,
			},
			{
				name:            "CN does not match INTL",
				savedRegion:     "CN",
				requestedRegion: RegionINTL,
				shouldMatch:     false,
			},
			{
				name:            "INTL does not match CN",
				savedRegion:     "INTL",
				requestedRegion: RegionCN,
				shouldMatch:     false,
			},
			{
				name:            "empty does not match",
				savedRegion:     "",
				requestedRegion: RegionCN,
				shouldMatch:     false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				profile := &McpProfile{
					MCPOAuthSiteType: tt.savedRegion,
				}

				// 模拟对比逻辑：region type 必须存在且匹配
				matches := profile.MCPOAuthSiteType != "" && profile.MCPOAuthSiteType == string(tt.requestedRegion)
				assert.Equal(t, tt.shouldMatch, matches)
			})
		}
	})
}

func TestMcpProfileJSONSerialization(t *testing.T) {
	profile := &McpProfile{
		Name:                         "test-profile",
		MCPOAuthSiteType:             "CN",
		MCPOAuthAppId:                "app-id",
		MCPOAuthAppName:              "app-name",
		MCPOAuthAccessToken:          "access-token",
		MCPOAuthRefreshToken:         "refresh-token",
		MCPOAuthAccessTokenExpire:    1234567890,
		MCPOAuthAccessTokenValidity:  10800,
		MCPOAuthRefreshTokenValidity: 31536000,
		MCPOAuthRefreshTokenExpire:   1234567890,
	}

	jsonBytes, err := json.Marshal(profile)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	var loadedProfile McpProfile
	err = json.Unmarshal(jsonBytes, &loadedProfile)
	assert.NoError(t, err)
	assert.Equal(t, profile.Name, loadedProfile.Name)
	assert.Equal(t, profile.MCPOAuthAppName, loadedProfile.MCPOAuthAppName)
	assert.Equal(t, profile.MCPOAuthAccessTokenValidity, loadedProfile.MCPOAuthAccessTokenValidity)
	assert.Equal(t, profile.MCPOAuthAccessToken, loadedProfile.MCPOAuthAccessToken)
	assert.Equal(t, profile.MCPOAuthRefreshToken, loadedProfile.MCPOAuthRefreshToken)
	assert.Equal(t, profile.MCPOAuthAccessTokenExpire, loadedProfile.MCPOAuthAccessTokenExpire)
	assert.Equal(t, profile.MCPOAuthRefreshTokenValidity, loadedProfile.MCPOAuthRefreshTokenValidity)
	assert.Equal(t, profile.MCPOAuthRefreshTokenExpire, loadedProfile.MCPOAuthRefreshTokenExpire)
	assert.Equal(t, profile.MCPOAuthSiteType, loadedProfile.MCPOAuthSiteType)
	assert.Equal(t, profile.MCPOAuthAppId, loadedProfile.MCPOAuthAppId)
}

func TestLoadExistingMCPProfile(t *testing.T) {
	t.Run("config file not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()
		os.Setenv("HOME", tmpDir)

		ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
		profile := config.NewProfile("test")
		opts := ProxyConfig{
			RegionType: RegionCN,
			Host:       "127.0.0.1",
			Port:       8088,
			Scope:      "/acs/mcp-server",
		}

		result := loadExistingMCPProfile(ctx, profile, opts, "default-mcp")
		assert.Nil(t, result, "should return nil when config file does not exist")
	})

	t.Run("invalid json in config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()
		os.Setenv("HOME", tmpDir)

		// 创建无效的配置文件
		configPath := getMCPConfigPath()
		err := os.MkdirAll(filepath.Dir(configPath), 0755)
		assert.NoError(t, err)
		err = os.WriteFile(configPath, []byte("{invalid json}"), 0644)
		assert.NoError(t, err)

		ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
		profile := config.NewProfile("test")
		opts := ProxyConfig{
			RegionType: RegionCN,
			Host:       "127.0.0.1",
			Port:       8088,
			Scope:      "/acs/mcp-server",
		}

		result := loadExistingMCPProfile(ctx, profile, opts, "default-mcp")
		assert.Nil(t, result, "should return nil when config file has invalid JSON")
	})

	t.Run("region type mismatch", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()
		os.Setenv("HOME", tmpDir)

		// 创建配置文件，region type 为 CN
		profile := NewMcpProfile("test-profile")
		profile.MCPOAuthSiteType = "CN"
		profile.MCPOAuthAppName = "default-mcp"
		profile.MCPOAuthAppId = "test-app-id"
		profile.MCPOAuthRefreshToken = "test-refresh-token"
		profile.MCPOAuthRefreshTokenExpire = time.Now().Unix() + 3600
		err := saveMcpProfile(profile)
		assert.NoError(t, err)

		ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
		configProfile := config.NewProfile("test")
		opts := ProxyConfig{
			RegionType: RegionINTL, // 请求 INTL，但保存的是 CN
			Host:       "127.0.0.1",
			Port:       8088,
			Scope:      "/acs/mcp-server",
		}

		result := loadExistingMCPProfile(ctx, configProfile, opts, "default-mcp")
		assert.Nil(t, result, "should return nil when region type mismatch")
	})

	t.Run("app name mismatch", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()
		os.Setenv("HOME", tmpDir)

		// 创建配置文件，app name 为 "default-mcp"
		profile := NewMcpProfile("test-profile")
		profile.MCPOAuthSiteType = "CN"
		profile.MCPOAuthAppName = "default-mcp"
		profile.MCPOAuthAppId = "test-app-id"
		profile.MCPOAuthRefreshToken = "test-refresh-token"
		profile.MCPOAuthRefreshTokenExpire = time.Now().Unix() + 3600
		err := saveMcpProfile(profile)
		assert.NoError(t, err)

		ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
		configProfile := config.NewProfile("test")
		opts := ProxyConfig{
			RegionType:   RegionCN,
			Host:         "127.0.0.1",
			Port:         8088,
			Scope:        "/acs/mcp-server",
			OAuthAppName: "different-app", // 请求不同的 app name
		}

		result := loadExistingMCPProfile(ctx, configProfile, opts, "different-app")
		assert.Nil(t, result, "should return nil when app name mismatch")
	})

	t.Run("empty AppId", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()
		os.Setenv("HOME", tmpDir)

		// 创建配置文件，AppId 为空
		profile := NewMcpProfile("test-profile")
		profile.MCPOAuthSiteType = "CN"
		profile.MCPOAuthAppName = "default-mcp"
		profile.MCPOAuthAppId = "" // 空的 AppId
		profile.MCPOAuthRefreshToken = "test-refresh-token"
		profile.MCPOAuthRefreshTokenExpire = time.Now().Unix() + 3600
		err := saveMcpProfile(profile)
		assert.NoError(t, err)

		ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
		configProfile := config.NewProfile("test")
		opts := ProxyConfig{
			RegionType: RegionCN,
			Host:       "127.0.0.1",
			Port:       8088,
			Scope:      "/acs/mcp-server",
		}

		result := loadExistingMCPProfile(ctx, configProfile, opts, "default-mcp")
		assert.Nil(t, result, "should return nil when AppId is empty")
	})

	t.Run("empty RefreshToken", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()
		os.Setenv("HOME", tmpDir)

		// 创建配置文件，RefreshToken 为空
		profile := NewMcpProfile("test-profile")
		profile.MCPOAuthSiteType = "CN"
		profile.MCPOAuthAppName = "default-mcp"
		profile.MCPOAuthAppId = "test-app-id"
		profile.MCPOAuthRefreshToken = "" // 空的 RefreshToken
		profile.MCPOAuthRefreshTokenExpire = time.Now().Unix() + 3600
		err := saveMcpProfile(profile)
		assert.NoError(t, err)

		ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
		configProfile := config.NewProfile("test")
		opts := ProxyConfig{
			RegionType: RegionCN,
			Host:       "127.0.0.1",
			Port:       8088,
			Scope:      "/acs/mcp-server",
		}

		result := loadExistingMCPProfile(ctx, configProfile, opts, "default-mcp")
		assert.Nil(t, result, "should return nil when RefreshToken is empty")
	})

	t.Run("expired RefreshToken", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()
		os.Setenv("HOME", tmpDir)

		// 创建配置文件，RefreshToken 已过期
		profile := NewMcpProfile("test-profile")
		profile.MCPOAuthSiteType = "CN"
		profile.MCPOAuthAppName = "default-mcp"
		profile.MCPOAuthAppId = "test-app-id"
		profile.MCPOAuthRefreshToken = "test-refresh-token"
		profile.MCPOAuthRefreshTokenExpire = time.Now().Unix() - 100 // 已过期
		err := saveMcpProfile(profile)
		assert.NoError(t, err)

		ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
		configProfile := config.NewProfile("test")
		opts := ProxyConfig{
			RegionType: RegionCN,
			Host:       "127.0.0.1",
			Port:       8088,
			Scope:      "/acs/mcp-server",
		}

		result := loadExistingMCPProfile(ctx, configProfile, opts, "default-mcp")
		assert.Nil(t, result, "should return nil when RefreshToken is expired")
	})

	t.Run("valid profile but missing required fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()
		os.Setenv("HOME", tmpDir)

		// 创建最小化的配置文件（缺少必要字段）
		profile := NewMcpProfile("test-profile")
		profile.MCPOAuthSiteType = "CN"
		profile.MCPOAuthAppName = "default-mcp"
		// 缺少 AppId 和 RefreshToken
		err := saveMcpProfile(profile)
		assert.NoError(t, err)

		ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
		configProfile := config.NewProfile("test")
		opts := ProxyConfig{
			RegionType: RegionCN,
			Host:       "127.0.0.1",
			Port:       8088,
			Scope:      "/acs/mcp-server",
		}

		result := loadExistingMCPProfile(ctx, configProfile, opts, "default-mcp")
		assert.Nil(t, result, "should return nil when required fields are missing")
	})
}

func TestGetOrCreateMCPProfile_FindOAuthApplicationLogic(t *testing.T) {

	t.Run("validateOAuthApplication with nil app", func(t *testing.T) {
		err := validateOAuthApplication(nil, "/acs/mcp-server", "127.0.0.1", 8088)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "OAuth application is nil")
	})

	t.Run("validateOAuthApplication with wrong AppType", func(t *testing.T) {
		app := &OAuthApplication{
			AppName:              "test-app",
			ApplicationId:        "app-id",
			AppType:              "WebApp", // 错误的类型
			Scopes:               []string{"/acs/mcp-server"},
			RedirectUris:         []string{"http://127.0.0.1:8088/callback"},
			AccessTokenValidity:  10800,
			RefreshTokenValidity: 31536000,
		}

		err := validateOAuthApplication(app, "/acs/mcp-server", "127.0.0.1", 8088)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be 'NativeApp'")
		assert.Contains(t, err.Error(), "WebApp")
	})

	t.Run("validateOAuthApplication with missing scope", func(t *testing.T) {
		app := &OAuthApplication{
			AppName:              "test-app",
			ApplicationId:        "app-id",
			AppType:              "NativeApp",
			Scopes:               []string{"/other/scope"}, // 缺少 required scope
			RedirectUris:         []string{"http://127.0.0.1:8088/callback"},
			AccessTokenValidity:  10800,
			RefreshTokenValidity: 31536000,
		}

		err := validateOAuthApplication(app, "/acs/mcp-server", "127.0.0.1", 8088)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not have required scope")
		assert.Contains(t, err.Error(), "/acs/mcp-server")
	})

	t.Run("validateOAuthApplication with wrong redirect URI", func(t *testing.T) {
		app := &OAuthApplication{
			AppName:              "test-app",
			ApplicationId:        "app-id",
			AppType:              "NativeApp",
			Scopes:               []string{"/acs/mcp-server"},
			RedirectUris:         []string{"http://127.0.0.1:9999/callback"}, // 错误的端口
			AccessTokenValidity:  10800,
			RefreshTokenValidity: 31536000,
		}

		err := validateOAuthApplication(app, "/acs/mcp-server", "127.0.0.1", 8088)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not have required redirect URI")
		assert.Contains(t, err.Error(), "http://127.0.0.1:8088/callback")
	})

	t.Run("validateOAuthApplication with valid app", func(t *testing.T) {
		app := &OAuthApplication{
			AppName:              "test-app",
			ApplicationId:        "app-id",
			AppType:              "NativeApp",
			Scopes:               []string{"/acs/mcp-server"},
			RedirectUris:         []string{"http://127.0.0.1:8088/callback"},
			AccessTokenValidity:  10800,
			RefreshTokenValidity: 31536000,
		}

		err := validateOAuthApplication(app, "/acs/mcp-server", "127.0.0.1", 8088)
		assert.NoError(t, err)
	})

	t.Run("validateOAuthApplication with multiple scopes", func(t *testing.T) {
		app := &OAuthApplication{
			AppName:              "test-app",
			ApplicationId:        "app-id",
			AppType:              "NativeApp",
			Scopes:               []string{"/other/scope", "/acs/mcp-server", "/another/scope"},
			RedirectUris:         []string{"http://127.0.0.1:8088/callback"},
			AccessTokenValidity:  10800,
			RefreshTokenValidity: 31536000,
		}

		err := validateOAuthApplication(app, "/acs/mcp-server", "127.0.0.1", 8088)
		assert.NoError(t, err, "should pass validation when required scope is present among multiple scopes")
	})

	t.Run("validateOAuthApplication with multiple redirect URIs", func(t *testing.T) {
		app := &OAuthApplication{
			AppName:              "test-app",
			ApplicationId:        "app-id",
			AppType:              "NativeApp",
			Scopes:               []string{"/acs/mcp-server"},
			RedirectUris:         []string{"http://0.0.0.0:8088/callback", "http://127.0.0.1:8088/callback"},
			AccessTokenValidity:  10800,
			RefreshTokenValidity: 31536000,
		}

		err := validateOAuthApplication(app, "/acs/mcp-server", "127.0.0.1", 8088)
		assert.NoError(t, err, "should pass validation when required redirect URI is present among multiple URIs")
	})

	t.Run("validateOAuthApplication error message format", func(t *testing.T) {
		app := &OAuthApplication{
			AppName:              "test-app",
			ApplicationId:        "app-id",
			AppType:              "WebApp",
			Scopes:               []string{"/other/scope"},
			RedirectUris:         []string{"http://127.0.0.1:9999/callback"},
			AccessTokenValidity:  10800,
			RefreshTokenValidity: 31536000,
		}

		err := validateOAuthApplication(app, "/acs/mcp-server", "127.0.0.1", 8088)
		assert.Error(t, err)
		wrappedErr := fmt.Errorf("OAuth application validation failed: %w", err)
		assert.Contains(t, wrappedErr.Error(), "validation failed")
		assert.Contains(t, wrappedErr.Error(), "must be 'NativeApp'")
	})
}
