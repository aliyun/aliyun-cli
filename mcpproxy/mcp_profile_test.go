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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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
		MCPOAuthAccessToken:          "access-token",
		MCPOAuthRefreshToken:         "refresh-token",
		MCPOAuthAccessTokenExpire:    1234567890,
		MCPOAuthRefreshTokenValidity: 31536000,
		MCPOAuthRefreshTokenExpire:   1234567890,
		MCPOAuthSiteType:             "CN",
		MCPOAuthAppId:                "app-id",
	}

	jsonBytes, err := json.Marshal(profile)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	var loadedProfile McpProfile
	err = json.Unmarshal(jsonBytes, &loadedProfile)
	assert.NoError(t, err)
	assert.Equal(t, profile.Name, loadedProfile.Name)
	assert.Equal(t, profile.MCPOAuthAccessToken, loadedProfile.MCPOAuthAccessToken)
	assert.Equal(t, profile.MCPOAuthRefreshToken, loadedProfile.MCPOAuthRefreshToken)
	assert.Equal(t, profile.MCPOAuthAccessTokenExpire, loadedProfile.MCPOAuthAccessTokenExpire)
	assert.Equal(t, profile.MCPOAuthRefreshTokenValidity, loadedProfile.MCPOAuthRefreshTokenValidity)
	assert.Equal(t, profile.MCPOAuthRefreshTokenExpire, loadedProfile.MCPOAuthRefreshTokenExpire)
	assert.Equal(t, profile.MCPOAuthSiteType, loadedProfile.MCPOAuthSiteType)
	assert.Equal(t, profile.MCPOAuthAppId, loadedProfile.MCPOAuthAppId)
}
