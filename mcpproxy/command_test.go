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
	"os"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
)

func TestNewMCPProxyCommand(t *testing.T) {
	cmd := NewMCPProxyCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "mcp-proxy", cmd.Name)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Usage)
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
	// 保存原始环境变量
	originalHome := os.Getenv("HOME")
	originalIgnoreProfile := os.Getenv("ALIBABA_CLOUD_IGNORE_PROFILE")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
		if originalIgnoreProfile != "" {
			os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", originalIgnoreProfile)
		} else {
			os.Unsetenv("ALIBABA_CLOUD_IGNORE_PROFILE")
		}
	}()

	// 设置临时目录和忽略配置文件，确保不会使用真实账号
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")

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
	originalHome := os.Getenv("HOME")
	originalIgnoreProfile := os.Getenv("ALIBABA_CLOUD_IGNORE_PROFILE")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
		if originalIgnoreProfile != "" {
			os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", originalIgnoreProfile)
		} else {
			os.Unsetenv("ALIBABA_CLOUD_IGNORE_PROFILE")
		}
	}()

	// 设置临时目录和忽略配置文件，确保不会使用真实账号
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")

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
	// 保存原始环境变量
	originalHome := os.Getenv("HOME")
	originalIgnoreProfile := os.Getenv("ALIBABA_CLOUD_IGNORE_PROFILE")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
		if originalIgnoreProfile != "" {
			os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", originalIgnoreProfile)
		} else {
			os.Unsetenv("ALIBABA_CLOUD_IGNORE_PROFILE")
		}
	}()

	regionTypes := []string{"CN", "INTL"}
	for _, regionType := range regionTypes {
		t.Run(regionType, func(t *testing.T) {
			// 设置临时目录和忽略配置文件，确保不会使用真实账号
			tmpDir := t.TempDir()
			os.Setenv("HOME", tmpDir)
			os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")

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
