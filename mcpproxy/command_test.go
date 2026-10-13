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
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMCPProxyCommand(t *testing.T) {
	cmd := NewMCPProxyCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "mcp-proxy", cmd.Name)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.Equal(t, "mcp-proxy [--port PORT] [--host HOST] [--region-type REGION_TYPE] [--upstream-url URL] [--oauth-app-name NAME] [--profile NAME]", cmd.Usage)
	assert.Equal(t, "aliyun utils mcp-proxy --region-type CN --port 8088", cmd.Sample)
	assert.NotNil(t, cmd.Run)

	// 检查标志
	flags := cmd.Flags()
	assert.NotNil(t, flags)

	portFlag := flags.Get("port")
	assert.NotNil(t, portFlag)
	assert.Equal(t, "8088", portFlag.DefaultValue)

	hostFlag := flags.Get("host")
	assert.NotNil(t, hostFlag)
	assert.Equal(t, "127.0.0.1", hostFlag.DefaultValue)

	regionFlag := flags.Get("region-type")
	assert.NotNil(t, regionFlag)
	assert.Equal(t, "CN", regionFlag.DefaultValue)

	scopeFlag := flags.Get("scope")
	assert.NotNil(t, scopeFlag)
	assert.Equal(t, "/acs/mcp-server", scopeFlag.DefaultValue)
}

func TestGetContentFromApiResponse(t *testing.T) {
	tests := []struct {
		name     string
		response map[string]any
		wantErr  bool
		validate func(t *testing.T, result []byte)
	}{
		{
			name: "string body",
			response: map[string]any{
				"body": "test string",
			},
			wantErr: false,
			validate: func(t *testing.T, result []byte) {
				assert.Equal(t, "test string", string(result))
			},
		},
		{
			name: "map body",
			response: map[string]any{
				"body": map[string]any{
					"key": "value",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "key")
				assert.Contains(t, string(result), "value")
			},
		},
		{
			name: "array body",
			response: map[string]any{
				"body": []any{"item1", "item2"},
			},
			wantErr: false,
			validate: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "item1")
				assert.Contains(t, string(result), "item2")
			},
		},
		{
			name: "byte slice body",
			response: map[string]any{
				"body": []byte("test bytes"),
			},
			wantErr: false,
			validate: func(t *testing.T, result []byte) {
				assert.Equal(t, "test bytes", string(result))
			},
		},
		{
			name: "int body",
			response: map[string]any{
				"body": 123,
			},
			wantErr: false,
			validate: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "123")
			},
		},
		{
			name: "nil body",
			response: map[string]any{
				"body": nil,
			},
			wantErr: true,
			validate: func(t *testing.T, result []byte) {
				assert.Nil(t, result)
			},
		},
		{
			name: "no body key",
			response: map[string]any{
				"other": "value",
			},
			wantErr: true,
			validate: func(t *testing.T, result []byte) {
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetContentFromApiResponse(tt.response)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestRunMCPProxy_InvalidPort(t *testing.T) {
	// 设置临时目录和忽略配置文件，确保不会使用真实账号
	setTestHome(t)
	t.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")

	ctx := cli.NewCommandContext(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	portFlag := &cli.Flag{
		Name:         "port",
		DefaultValue: "invalid",
	}
	portFlag.SetAssigned(true)
	portFlag.SetValue("invalid")
	ctx.Flags().Add(portFlag)
	ctx.Flags().Add(&cli.Flag{
		Name:         "host",
		DefaultValue: "127.0.0.1",
	})
	ctx.Flags().Add(&cli.Flag{
		Name:         "region-type",
		DefaultValue: "CN",
	})

	err := runMCPProxy(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port")
}

func TestRunMCPProxy_InvalidRegionType(t *testing.T) {
	// 设置临时目录和忽略配置文件，确保不会使用真实账号
	setTestHome(t)
	t.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")

	ctx := cli.NewCommandContext(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	portFlag := &cli.Flag{
		Name:         "port",
		DefaultValue: "8088",
	}
	portFlag.SetAssigned(true)
	portFlag.SetValue("8088")
	ctx.Flags().Add(portFlag)
	ctx.Flags().Add(&cli.Flag{
		Name:         "host",
		DefaultValue: "127.0.0.1",
	})
	regionFlag := &cli.Flag{
		Name:         "region-type",
		DefaultValue: "INVALID",
	}
	regionFlag.SetAssigned(true)
	regionFlag.SetValue("INVALID")
	ctx.Flags().Add(regionFlag)

	err := runMCPProxy(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid region type")
}

func TestRunMCPProxy_ValidRegionTypes(t *testing.T) {
	regionTypes := []string{"CN", "INTL"}
	for _, regionType := range regionTypes {
		t.Run(regionType, func(t *testing.T) {
			// 设置临时目录和忽略配置文件，确保不会使用真实账号
			setTestHome(t)
			t.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")

			ctx := cli.NewCommandContext(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
			portFlag := &cli.Flag{
				Name:         "port",
				DefaultValue: "8088",
			}
			portFlag.SetAssigned(true)
			portFlag.SetValue("8088")
			ctx.Flags().Add(portFlag)
			ctx.Flags().Add(&cli.Flag{
				Name:         "host",
				DefaultValue: "127.0.0.1",
			})
			regionFlag := &cli.Flag{
				Name:         "region-type",
				DefaultValue: regionType,
			}
			regionFlag.SetAssigned(true)
			regionFlag.SetValue(regionType)
			ctx.Flags().Add(regionFlag)

			err := runMCPProxy(ctx)
			// 预期会因为缺少profile或配置而失败
			assert.Error(t, err)
			// 验证错误不是 "invalid region type"，说明 region type 解析成功
			assert.NotContains(t, err.Error(), "invalid region type")
			assert.Contains(t, err.Error(), "failed to load profile")
		})
	}
}

func setMCPCommandTestFlag(ctx *cli.Context, name, value string) {
	flag := ctx.Flags().Get(name)
	if flag == nil {
		flag = &cli.Flag{Name: name}
		ctx.Flags().Add(flag)
	}
	flag.SetAssigned(true)
	flag.SetValue(value)
}

func forbidMCPCommandTestNetwork(t *testing.T) {
	t.Helper()
	previousTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	http.DefaultTransport = credentialRoundTripper(func(req *http.Request) (*http.Response, error) {
		t.Errorf("invalid authentication options must not make a request to %s", req.URL.Host)
		return credentialResponse(http.StatusForbidden, `{"Code":"Forbidden"}`), nil
	})
}

func TestUsesCLICredentials(t *testing.T) {
	for _, tc := range []struct {
		name       string
		flags      map[string]string
		profileEnv string
		ignoreEnv  string
		want       bool
	}{
		{name: "unchanged default"},
		{name: "explicit profile uses credentials", flags: map[string]string{"profile": "existing"}, want: true},
		{name: "empty explicit profile requires validation", flags: map[string]string{"profile": ""}, want: true},
		{name: "no browser preserves OAuth", flags: map[string]string{"no-browser": ""}},
		{name: "environment profile preserves OAuth", profileEnv: "environment"},
		{name: "ignored profile environment preserves OAuth", profileEnv: "environment", ignoreEnv: "ALIBABA_CLOUD_IGNORE_PROFILE"},
		{name: "legacy ignored profile environment preserves OAuth", profileEnv: "environment", ignoreEnv: "ALIBABACLOUD_IGNORE_PROFILE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ALIBABA_CLOUD_PROFILE", tc.profileEnv)
			if tc.ignoreEnv != "" {
				t.Setenv(tc.ignoreEnv, "TRUE")
			}
			ctx := cli.NewCommandContext(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
			for name, value := range tc.flags {
				setMCPCommandTestFlag(ctx, name, value)
			}
			require.Equal(t, tc.want, usesCLICredentials(ctx))
		})
	}
}

func TestConfigureMCPCredentialAuthentication(t *testing.T) {
	for _, mode := range []config.AuthenticateMode{config.AK, config.StsToken, config.OAuth} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := newMCPCredentialTestContext(t)
			ctx.SetInConfigureMode(false)
			profile := mcpTestAKProfile("selected", "selected-key")
			profile.Mode = mode
			profile.StsToken = "selected-sts"
			profile.StsExpiration = time.Now().Add(time.Hour).Unix()
			profile.OAuthSiteType = "CN"
			writeMCPCredentialTestConfig(t, ctx, mcpTestAKProfile("default", "wrong-key"),
				mcpTestAKProfile("environment", "environment-key"), profile)
			t.Setenv("ALIBABA_CLOUD_PROFILE", "environment")
			setMCPCommandTestFlag(ctx, "profile", "selected")
			setMCPCommandTestFlag(ctx, "no-browser", "")
			oldTransport := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = oldTransport })
			calls := 0
			http.DefaultTransport = credentialRoundTripper(func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "ramoauth.aliyuncs.com", req.URL.Host)
				require.Equal(t, "GenerateAccessToken", req.URL.Query().Get("Action"))
				require.Equal(t, "selected-key", req.URL.Query().Get("AccessKeyId"))
				if mode == config.StsToken || mode == config.OAuth {
					require.Equal(t, "selected-sts", req.URL.Query().Get("SecurityToken"))
				} else {
					require.Empty(t, req.URL.Query().Get("SecurityToken"))
				}
				return credentialResponse(200, `{"Data":{"AccessToken":"mcp-test-token","ExpiresIn":3600}}`), nil
			})
			proxyConfig := ProxyConfig{RegionType: RegionCN}
			require.NoError(t, configureMCPAuthentication(ctx, &proxyConfig))
			require.False(t, ctx.InConfigureMode(), "MCP must preserve the caller's credential flag policy")
			require.NotNil(t, proxyConfig.CredentialAuth)
			require.Nil(t, proxyConfig.McpProfile)
			token, err := proxyConfig.CredentialAuth.GetToken()
			require.NoError(t, err)
			require.Equal(t, "mcp-test-token", token)
			require.Equal(t, 1, calls)
			_, err = os.Stat(filepath.Join(config.GetConfigPath(), ".mcpproxy_config"))
			require.True(t, os.IsNotExist(err), "credential authentication must not create an MCP OAuth profile")
		})
	}
}

func TestConfigureMCPAuthenticationRejectsUnsupportedOptions(t *testing.T) {
	for _, tc := range []struct {
		name, option, message string
		mode                  config.AuthenticateMode
	}{
		{name: "BearerToken", mode: config.BearerToken, message: "not supported"},
		{name: "Anonymous", mode: config.Anonymous, message: "not supported"},
		{name: "unknown", mode: "unknown", message: "invalid mode"},
		{name: "OAuth app name", mode: config.AK, option: "oauth-app-name", message: "omit --profile"},
		{name: "OAuth scope", mode: config.AK, option: "scope", message: "omit --profile"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := newMCPCredentialTestContext(t)
			forbidMCPCommandTestNetwork(t)
			profile := mcpTestAKProfile("selected", "key")
			profile.Mode = tc.mode
			profile.BearerTokenValue = "test-bearer"
			writeMCPCredentialTestConfig(t, ctx, mcpTestAKProfile("default", "fallback-key"), profile)
			setMCPCommandTestFlag(ctx, "profile", "selected")
			if tc.option != "" {
				setMCPCommandTestFlag(ctx, tc.option, "custom")
			}
			err := configureMCPAuthentication(ctx, &ProxyConfig{RegionType: RegionCN})
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestConfigureMCPAuthenticationRejectsInvalidProfileWithoutFallback(t *testing.T) {
	for _, profileName := range []string{"", " \t ", "missing"} {
		t.Run(profileName, func(t *testing.T) {
			ctx := newMCPCredentialTestContext(t)
			forbidMCPCommandTestNetwork(t)
			writeMCPCredentialTestConfig(t, ctx, mcpTestAKProfile("default", "default-key"),
				mcpTestAKProfile("environment", "environment-key"))
			t.Setenv("ALIBABA_CLOUD_PROFILE", "environment")
			setMCPCommandTestFlag(ctx, "profile", profileName)
			proxyConfig := ProxyConfig{RegionType: RegionCN}

			err := configureMCPAuthentication(ctx, &proxyConfig)

			if profileName == "missing" {
				require.ErrorContains(t, err, "unknown profile missing")
			} else {
				require.ErrorContains(t, err, "--profile requires a non-empty profile name")
			}
			require.Nil(t, proxyConfig.CredentialAuth)
			require.Nil(t, proxyConfig.McpProfile)
		})
	}
}

func TestConfigureMCPAuthenticationIgnoresCredentialEnvironment(t *testing.T) {
	for _, variable := range []string{"ALIBABA_CLOUD_IGNORE_PROFILE", "ALIBABACLOUD_IGNORE_PROFILE"} {
		t.Run(variable, func(t *testing.T) {
			ctx := newMCPCredentialTestContext(t)
			writeMCPCredentialTestConfig(t, ctx, mcpTestAKProfile("default", "default-key"),
				mcpTestAKProfile("environment", "environment-profile-key"), mcpTestAKProfile("selected", "selected-key"))
			setMCPCommandTestFlag(ctx, "profile", "selected")
			t.Setenv(variable, "TRUE")
			t.Setenv("ALIBABA_CLOUD_PROFILE", "environment")
			t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", "environment-key")
			t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "environment-secret")
			previousTransport := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = previousTransport })
			calls := 0
			http.DefaultTransport = credentialRoundTripper(func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "GenerateAccessToken", req.URL.Query().Get("Action"))
				require.Equal(t, "selected-key", req.URL.Query().Get("AccessKeyId"))
				return credentialResponse(200, `{"Data":{"AccessToken":"mcp-test-token","ExpiresIn":3600}}`), nil
			})
			proxyConfig := ProxyConfig{RegionType: RegionCN}

			err := configureMCPAuthentication(ctx, &proxyConfig)

			require.NoError(t, err)
			require.NotNil(t, proxyConfig.CredentialAuth)
			require.Nil(t, proxyConfig.McpProfile)
			require.Equal(t, 1, calls)
		})
	}
}

func TestConfigureMCPAuthenticationDistinguishesProfilesWithSameMode(t *testing.T) {
	ctx := newMCPCredentialTestContext(t)
	writeMCPCredentialTestConfig(t, ctx, mcpTestAKProfile("first", "first-key"), mcpTestAKProfile("second", "second-key"))
	t.Setenv("ALIBABA_CLOUD_PROFILE", "first")
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	var accessKeys []string
	http.DefaultTransport = credentialRoundTripper(func(req *http.Request) (*http.Response, error) {
		accessKeys = append(accessKeys, req.URL.Query().Get("AccessKeyId"))
		return credentialResponse(200, `{"Data":{"AccessToken":"mcp-test-token","ExpiresIn":3600}}`), nil
	})
	for _, name := range []string{"first", "second"} {
		setMCPCommandTestFlag(ctx, "profile", name)
		require.NoError(t, configureMCPAuthentication(ctx, &ProxyConfig{RegionType: RegionCN}))
	}
	require.Equal(t, []string{"first-key", "second-key"}, accessKeys)
}

func TestMCPCredentialAuthenticationPinsSelectedProfile(t *testing.T) {
	ctx := newMCPCredentialTestContext(t)
	first := mcpTestAKProfile("first", "first-key")
	second := mcpTestAKProfile("second", "second-key")
	writeMCPCredentialTestConfig(t, ctx, first, second)
	setMCPCommandTestFlag(ctx, "profile", "first")
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	var accessKeys []string
	http.DefaultTransport = credentialRoundTripper(func(req *http.Request) (*http.Response, error) {
		accessKeys = append(accessKeys, req.URL.Query().Get("AccessKeyId"))
		return credentialResponse(200, `{"Data":{"AccessToken":"mcp-test-token","ExpiresIn":3600}}`), nil
	})
	proxyConfig := ProxyConfig{RegionType: RegionCN}
	require.NoError(t, configureMCPAuthentication(ctx, &proxyConfig))
	first.AccessKeyId = "first-rotated-key"
	writeMCPCredentialTestConfig(t, ctx, second, first)
	_, err := proxyConfig.CredentialAuth.RefreshToken("mcp-test-token")
	require.NoError(t, err)
	require.Equal(t, []string{"first-key", "first-rotated-key"}, accessKeys)
	writeMCPCredentialTestConfig(t, ctx, second)
	_, err = proxyConfig.CredentialAuth.RefreshToken("mcp-test-token")
	require.Error(t, err, "a deleted selected profile must not fall back to the new default")
	require.Equal(t, []string{"first-key", "first-rotated-key"}, accessKeys)
}
