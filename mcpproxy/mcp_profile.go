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
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/util"
)

type McpProfile struct {
	Name                         string `json:"name"`
	MCPOAuthAppName              string `json:"mcp_oauth_app_name,omitempty"`
	MCPOAuthAppId                string `json:"mcp_oauth_app_id,omitempty"`
	MCPOAuthSiteType             string `json:"mcp_oauth_site_type,omitempty"` // CN or INTL
	MCPOAuthAccessToken          string `json:"mcp_oauth_access_token,omitempty"`
	MCPOAuthRefreshToken         string `json:"mcp_oauth_refresh_token,omitempty"`
	MCPOAuthAccessTokenValidity  int    `json:"mcp_oauth_access_token_validity,omitempty"`
	MCPOAuthAccessTokenExpire    int64  `json:"mcp_oauth_access_token_expire,omitempty"`
	MCPOAuthRefreshTokenValidity int    `json:"mcp_oauth_refresh_token_validity,omitempty"`
	MCPOAuthRefreshTokenExpire   int64  `json:"mcp_oauth_refresh_token_expire,omitempty"`
}

func getMCPConfigPath() string {
	return config.GetConfigPath() + "/.mcpproxy_config"
}

func NewMcpProfile(name string) *McpProfile {
	return &McpProfile{Name: name}
}

func NewMcpProfileFromBytes(bytes []byte) (profile *McpProfile, err error) {
	profile = &McpProfile{}
	err = json.Unmarshal(bytes, profile)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func saveMcpProfile(profile *McpProfile) error {
	mcpConfigPath := getMCPConfigPath()
	dir := filepath.Dir(mcpConfigPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %q: %w", dir, err)
	}

	// log.Printf("saveMcpProfile: Before marshaling - RefreshToken length=%d, RefreshToken empty=%v",
	// 	len(profile.MCPOAuthRefreshToken), profile.MCPOAuthRefreshToken == "")

	tempFile := mcpConfigPath + ".tmp"

	bytes, err := json.MarshalIndent(profile, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	// jsonStr := string(bytes)
	// hasRefreshToken := strings.Contains(jsonStr, "mcp_oauth_refresh_token")
	// log.Printf("saveMcpProfile: After marshaling - JSON contains 'mcp_oauth_refresh_token'=%v, JSON length=%d",
	// 	hasRefreshToken, len(jsonStr))

	if err := os.WriteFile(tempFile, bytes, 0600); err != nil {
		return fmt.Errorf("failed to write temp file %q: %w", tempFile, err)
	}

	// 原子性地重命名临时文件为目标文件， 避免因各种系统异常直接损坏原文件
	if err := os.Rename(tempFile, mcpConfigPath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp file to %q: %w", mcpConfigPath, err)
	}

	log.Printf("saveMcpProfile: Successfully saved MCP profile")
	return nil
}

// loadExistingMCPProfile 加载并验证已有的 MCP profile，如果有效则返回，避免重复拉起 OAuth
func loadExistingMCPProfile(ctx *cli.Context, profile config.Profile, opts ProxyConfig, desiredAppName string) *McpProfile {
	mcpConfigPath := getMCPConfigPath()
	bytes, err := os.ReadFile(mcpConfigPath)
	if err != nil {
		return nil
	}
	mcpProfile, err := NewMcpProfileFromBytes(bytes)
	if err != nil {
		return nil
	}

	if mcpProfile.MCPOAuthSiteType != string(opts.RegionType) {
		log.Printf("Region type mismatch: saved=%s, requested=%s, ignoring local profile", mcpProfile.MCPOAuthSiteType, string(opts.RegionType))
		return nil
	}

	if mcpProfile.MCPOAuthAppName != desiredAppName {
		log.Printf("App name mismatch: saved=%s, requested=%s, ignoring local profile", mcpProfile.MCPOAuthAppName, desiredAppName)
		return nil
	}

	if mcpProfile.MCPOAuthAppId == "" {
		log.Printf("MCP profile with AppId is empty, ignoring local profile")
		return nil
	}

	if mcpProfile.MCPOAuthRefreshToken == "" {
		log.Printf("MCP profile with RefreshToken is empty, ignoring local profile")
		return nil
	}

	if mcpProfile.MCPOAuthRefreshTokenExpire <= util.GetCurrentUnixTime() {
		log.Printf("MCP profile with RefreshTokenExpire is expired, ignoring local profile")
		return nil
	}

	app, err := findOAuthApplicationById(ctx, profile, mcpProfile.MCPOAuthAppId, opts.RegionType)
	if err != nil {
		log.Printf("Failed to reuse existing MCP profile (app: %s): %v, ignoring local profile", mcpProfile.MCPOAuthAppName, err)
		return nil
	}
	if app == nil {
		log.Printf("OAuth application with AppId '%s' not found, ignoring local profile", mcpProfile.MCPOAuthAppId)
		return nil
	}

	if err := validateOAuthApplication(app, opts.Scope, opts.Host, opts.Port); err != nil {
		log.Printf("Reused existing MCP profile validation failed: %v, ignoring local profile", err)
		return nil
	}

	// 根据远端 app 信息更新 mcp profile 中的相关字段，其他字段（如 token）保持不变
	mcpProfile.MCPOAuthAppName = app.AppName
	mcpProfile.MCPOAuthAppId = app.ApplicationId
	mcpProfile.MCPOAuthAccessTokenValidity = app.AccessTokenValidity
	mcpProfile.MCPOAuthRefreshTokenValidity = app.RefreshTokenValidity

	log.Printf("Reused existing MCP profile with app '%s' (AppId: %s)", app.AppName, app.ApplicationId)

	return mcpProfile
}

func getOrCreateMCPProfile(ctx *cli.Context, opts ProxyConfig) (*McpProfile, error) {
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile: %w", err)
	}

	// 如果未显式指定 app name，则使用默认的 MCPOAuthAppName，便于复用本地 profile
	desiredAppName := opts.OAuthAppName
	if desiredAppName == "" {
		desiredAppName = MCPOAuthAppName
	}

	existingMcpProfile := loadExistingMCPProfile(ctx, profile, opts, desiredAppName)
	if existingMcpProfile != nil {
		// mcpprofile might change, save it again to ensure the latest state is saved
		if err := saveMcpProfile(existingMcpProfile); err != nil {
			return nil, fmt.Errorf("failed to save mcp profile: %w", err)
		}
		return existingMcpProfile, nil
	}

	app, err := findOAuthApplicationByName(ctx, profile, opts.RegionType, desiredAppName)
	if err != nil {
		return nil, fmt.Errorf("failed to find OAuth application '%s': %w", desiredAppName, err)
	}

	if app == nil {
		if opts.OAuthAppName != "" {
			// if user provide app name, but not found, return error
			return nil, fmt.Errorf("OAuth application '%s' not found", opts.OAuthAppName)
		}
		cli.Printf(ctx.Stdout(), "Creating new default MCP profile '%s'...\n", DefaultMcpProfileName)
		app, err = createDefaultMCPOauthApplication(ctx, profile, opts.RegionType, opts.Host, opts.Port, opts.Scope)
		if err != nil {
			return nil, fmt.Errorf("failed to create default OAuth application: %w", err)
		}
		cli.Printf(ctx.Stdout(), "Created new default OAuth application '%s' (AppId: %s)\n", app.AppName, app.ApplicationId)
	} else {
		cli.Printf(ctx.Stdout(), "Using existing OAuth application '%s' (AppId: %s)\n", app.AppName, app.ApplicationId)
	}

	if err := validateOAuthApplication(app, opts.Scope, opts.Host, opts.Port); err != nil {
		return nil, fmt.Errorf("OAuth application validation failed: %w", err)
	}
	validatedApp := app

	cli.Printf(ctx.Stdout(), "Setting up MCPOAuth profile '%s'...\n", DefaultMcpProfileName)
	mcpProfile := NewMcpProfile(DefaultMcpProfileName)
	mcpProfile.MCPOAuthSiteType = string(opts.RegionType)
	mcpProfile.MCPOAuthAppId = validatedApp.ApplicationId
	mcpProfile.MCPOAuthAppName = validatedApp.AppName
	// 刷新 token 接口不返回 refresh token 有效期，所以直接在这里设置
	currentTime := util.GetCurrentUnixTime()
	mcpProfile.MCPOAuthAccessTokenValidity = validatedApp.AccessTokenValidity
	mcpProfile.MCPOAuthRefreshTokenValidity = validatedApp.RefreshTokenValidity

	// noBrowser=true 表示禁用自动打开浏览器，autoOpenBrowser=false
	// noBrowser=false 表示启用自动打开浏览器，autoOpenBrowser=true
	tokenResult, err := startMCPOAuthFlow(ctx, mcpProfile.MCPOAuthAppId, opts.RegionType, opts.Host, opts.Port, opts.AutoOpenBrowser, opts.Scope)
	if err != nil {
		return nil, fmt.Errorf("OAuth login failed: %w", err)
	}

	log.Printf("OAuth flow completed: AccessToken length=%d, RefreshToken length=%d, AccessTokenExpire=%d",
		len(tokenResult.AccessToken), len(tokenResult.RefreshToken), tokenResult.AccessTokenExpire)
	if tokenResult.RefreshToken == "" {
		return nil, fmt.Errorf("OAuth flow returned empty RefreshToken (Region=%s, AppId=%s). "+
			"Please delete this application and let the system create a new NativeApp, or manually create a NativeApp",
			opts.RegionType, mcpProfile.MCPOAuthAppId)
	}

	mcpProfile.MCPOAuthAccessToken = tokenResult.AccessToken
	mcpProfile.MCPOAuthRefreshToken = tokenResult.RefreshToken
	mcpProfile.MCPOAuthAccessTokenExpire = tokenResult.AccessTokenExpire
	// refresh token will be updated each time latest access token is refreshed,
	// however the validity and expiration time is the same as the original when finishing oauth flow
	mcpProfile.MCPOAuthRefreshTokenExpire = currentTime + int64(validatedApp.RefreshTokenValidity)

	if err = saveMcpProfile(mcpProfile); err != nil {
		return nil, fmt.Errorf("failed to save mcp profile: %w", err)
	}

	cli.Printf(ctx.Stdout(), "MCP Profile '%s' configured for oauth app '%s' successfully!\n", mcpProfile.Name, mcpProfile.MCPOAuthAppName)

	return mcpProfile, nil
}
