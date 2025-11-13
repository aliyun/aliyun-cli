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

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/util"
)

type McpProfile struct {
	Name                         string `json:"name"`
	MCPOAuthAccessToken          string `json:"mcp_oauth_access_token,omitempty"`
	MCPOAuthRefreshToken         string `json:"mcp_oauth_refresh_token,omitempty"`
	MCPOAuthAccessTokenExpire    int64  `json:"mcp_oauth_access_token_expire,omitempty"`
	MCPOAuthRefreshTokenValidity int    `json:"mcp_oauth_refresh_token_validity,omitempty"`
	MCPOAuthRefreshTokenExpire   int64  `json:"mcp_oauth_refresh_token_expire,omitempty"`
	MCPOAuthSiteType             string `json:"mcp_oauth_site_type,omitempty"` // CN or INTL
	MCPOAuthAppId                string `json:"mcp_oauth_app_id,omitempty"`
}

func getMCPConfigPath() string {
	return config.GetConfigPath() + "/.mcpproxy_config"
}

func NewMcpProfile(name string) *McpProfile {
	return &McpProfile{Name: name}
}

func NewMcpProfileFromBytes(bytes []byte) (profile *McpProfile, err error) {
	profile = NewMcpProfile(DefaultMcpProfileName)
	err = json.Unmarshal(bytes, profile)
	return
}

func saveMcpProfile(profile *McpProfile) error {
	mcpConfigPath := getMCPConfigPath()
	tempFile := mcpConfigPath + ".tmp"

	bytes, err := json.MarshalIndent(profile, "", "\t")
	if err != nil {
		return err
	}

	if err = os.WriteFile(tempFile, bytes, 0600); err != nil {
		return err
	}

	return os.Rename(tempFile, mcpConfigPath)
}

func getOrCreateMCPProfile(ctx *cli.Context, region RegionType, host string, port int, noBrowser bool) (*McpProfile, error) {
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile: %w", err)
	}
	mcpConfigPath := getMCPConfigPath()
	if bytes, err := os.ReadFile(mcpConfigPath); err == nil {
		if mcpProfile, err := NewMcpProfileFromBytes(bytes); err == nil {
			log.Println("MCP Profile loaded from file", mcpProfile.Name, "app id", mcpProfile.MCPOAuthAppId)
			err = findExistingMCPOauthApplicationById(ctx, profile, mcpProfile, region)
			if err == nil {
				return mcpProfile, nil
			} else {
				log.Println("Failed to find existing OAuth application", err.Error())
			}
		}
	}

	app, err := getOrCreateMCPOAuthApplication(ctx, profile, region, host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create OAuth application: %w", err)
	}

	cli.Printf(ctx.Stdout(), "Setting up MCPOAuth profile '%s'...\n", DefaultMcpProfileName)

	mcpProfile := NewMcpProfile(DefaultMcpProfileName)
	mcpProfile.MCPOAuthSiteType = string(region)
	mcpProfile.MCPOAuthAppId = app.ApplicationId
	// 刷新 token 接口不返回 refresh token 有效期，所以直接在这里设置
	currentTime := util.GetCurrentUnixTime()
	mcpProfile.MCPOAuthRefreshTokenValidity = app.RefreshTokenValidity
	mcpProfile.MCPOAuthRefreshTokenExpire = currentTime + int64(app.RefreshTokenValidity)

	// noBrowser=true 表示禁用自动打开浏览器，autoOpenBrowser=false
	// noBrowser=false 表示启用自动打开浏览器，autoOpenBrowser=true
	autoOpenBrowser := !noBrowser
	if err = startMCPOAuthFlow(ctx, mcpProfile, region, host, port, autoOpenBrowser); err != nil {
		return nil, fmt.Errorf("OAuth login failed: %w", err)
	}

	if err = saveMcpProfile(mcpProfile); err != nil {
		return nil, fmt.Errorf("failed to save mcp profile: %w", err)
	}

	cli.Printf(ctx.Stdout(), "MCP Profile '%s' configured successfully!\n", mcpProfile.Name)

	return mcpProfile, nil
}
