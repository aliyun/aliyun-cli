package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCliBridge(t *testing.T) {
	NewCommandBridge(configCommand.command)
}

func TestBuildOssEndpoint(t *testing.T) {
	assert.Equal(t, "oss-cn-hangzhou.aliyuncs.com", buildOssEndpoint("cn-hangzhou", ""))
	assert.Equal(t, "oss-cn-hangzhou-internal.aliyuncs.com", buildOssEndpoint("cn-hangzhou", "vpc"))
	assert.Equal(t, "oss-cn-beijing-internal.aliyuncs.com", buildOssEndpoint("cn-beijing", "VPC"))
}

func TestStripCliOnlyFlagsFromArgs(t *testing.T) {
	t.Run("strips profile long and value", func(t *testing.T) {
		got := stripCliOnlyFlagsFromArgs([]string{
			"oss://bucket/obj", "/tmp/out", "--profile", "mac16@oyj",
		})
		assert.Equal(t, []string{"oss://bucket/obj", "/tmp/out"}, got)
	})

	t.Run("strips profile equals form", func(t *testing.T) {
		got := stripCliOnlyFlagsFromArgs([]string{
			"oss://bucket/obj", "--profile=other", "--recursive",
		})
		assert.Equal(t, []string{"oss://bucket/obj", "--recursive"}, got)
	})

	t.Run("strips profile shorthand -p", func(t *testing.T) {
		got := stripCliOnlyFlagsFromArgs([]string{
			"src", "dst", "-p", "myprof", "--force",
		})
		assert.Equal(t, []string{"src", "dst", "--force"}, got)
	})

	t.Run("keeps shared OSS options like region and endpoint", func(t *testing.T) {
		got := stripCliOnlyFlagsFromArgs([]string{
			"ls", "oss://b",
			"--region", "cn-hangzhou",
			"--endpoint", "oss-cn-hangzhou.aliyuncs.com",
			"--profile", "x",
			"--access-key-id", "ak",
		})
		assert.Equal(t, []string{
			"ls", "oss://b",
			"--region", "cn-hangzhou",
			"--endpoint", "oss-cn-hangzhou.aliyuncs.com",
			"--access-key-id", "ak",
		}, got)
	})

	t.Run("strips other CLI-only config flags", func(t *testing.T) {
		got := stripCliOnlyFlagsFromArgs([]string{
			"cp", "a", "b",
			"--config-path", "/tmp/cfg.json",
			"--sts-endpoint", "sts.aliyuncs.com",
			"--source-profile", "base",
			"--recursive",
		})
		assert.Equal(t, []string{"cp", "a", "b", "--recursive"}, got)
	})

	t.Run("profile without value does not swallow next flag", func(t *testing.T) {
		got := stripCliOnlyFlagsFromArgs([]string{"ls", "--profile", "--recursive"})
		assert.Equal(t, []string{"ls", "--recursive"}, got)
	})

	t.Run("shorthand without value does not swallow next flag", func(t *testing.T) {
		got := stripCliOnlyFlagsFromArgs([]string{"ls", "-p", "--force"})
		assert.Equal(t, []string{"ls", "--force"}, got)
	})

	t.Run("consumes value that starts with single dash", func(t *testing.T) {
		got := stripCliOnlyFlagsFromArgs([]string{
			"cp", "a", "b", "--profile", "-myprof", "--recursive",
		})
		assert.Equal(t, []string{"cp", "a", "b", "--recursive"}, got)
	})

	t.Run("consumes config-path value that looks like a dash path", func(t *testing.T) {
		got := stripCliOnlyFlagsFromArgs([]string{
			"ls", "oss://b", "--config-path", "-tmp/cfg", "--force",
		})
		assert.Equal(t, []string{"ls", "oss://b", "--force"}, got)
	})

	t.Run("shorthand consumes dash-prefixed profile value", func(t *testing.T) {
		got := stripCliOnlyFlagsFromArgs([]string{
			"src", "dst", "-p", "-myprof", "--force",
		})
		assert.Equal(t, []string{"src", "dst", "--force"}, got)
	})

	t.Run("strips AssignedNone CLI-only flag without consuming next", func(t *testing.T) {
		got := stripCliOnlyFlagsFromArgs([]string{
			"ls", "oss://b", "--skip-secure-verify", "--recursive",
		})
		assert.Equal(t, []string{"ls", "oss://b", "--recursive"}, got)
	})

	t.Run("empty args", func(t *testing.T) {
		assert.Empty(t, stripCliOnlyFlagsFromArgs(nil))
		assert.Empty(t, stripCliOnlyFlagsFromArgs([]string{}))
	})
}

func TestOssKnownOptionNames_IncludesEndpointAndRegion(t *testing.T) {
	known := ossKnownOptionNames()
	_, hasEndpoint := known["--endpoint"]
	_, hasRegion := known["--region"]
	_, hasProfile := known["--profile"]
	assert.True(t, hasEndpoint)
	assert.True(t, hasRegion)
	assert.False(t, hasProfile)
}

func newEndpointTestContext(t *testing.T) *cli.Context {
	t.Helper()
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	config.AddFlags(ctx.Flags())
	ctx.SetInConfigureMode(true)
	return ctx
}

func clearEndpointTestEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"ALIBABA_CLOUD_IGNORE_PROFILE",
		"ALIBABACLOUD_IGNORE_PROFILE",
		"ALIBABA_CLOUD_PROFILE",
		"ALIBABACLOUD_PROFILE",
		"ALICLOUD_PROFILE",
		"ALIBABA_CLOUD_REGION_ID",
		"ALIBABACLOUD_REGION_ID",
		"ALICLOUD_REGION_ID",
		"REGION_ID",
		"REGION",
		"ALIBABA_CLOUD_ENDPOINT",
		"ALIBABACLOUD_ENDPOINT",
		"ALICLOUD_ENDPOINT",
		"ENDPOINT",
		"ALIBABA_CLOUD_ENDPOINT_TYPE",
		"ALIBABACLOUD_ENDPOINT_TYPE",
		"ALICLOUD_ENDPOINT_TYPE",
		"ENDPOINT_TYPE",
	} {
		t.Setenv(key, "")
	}
}

func writeEndpointTestConfig(t *testing.T, endpoint, endpointType, region string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := fmt.Sprintf(`{
  "current": "default",
  "profiles": [{
    "name": "default",
    "mode": "AK",
    "access_key_id": "test-ak",
    "access_key_secret": "test-sk",
    "region_id": %q,
    "endpoint": %q,
    "endpoint_type": %q,
    "output_format": "json",
    "language": "en"
  }]
}`, region, endpoint, endpointType)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestParseAndGetEndpoint(t *testing.T) {
	clearEndpointTestEnv(t)

	t.Run("endpoint from args wins", func(t *testing.T) {
		ctx := newEndpointTestContext(t)
		path := writeEndpointTestConfig(t, "oss-cn-hangzhou.aliyuncs.com", "vpc", "cn-hangzhou")
		config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
		config.ConfigurePathFlag(ctx.Flags()).SetValue(path)

		got, err := ParseAndGetEndpoint(ctx, []string{"--endpoint", "oss-cn-shenzhen.aliyuncs.com"})
		require.NoError(t, err)
		assert.Equal(t, "oss-cn-shenzhen.aliyuncs.com", got)
	})

	t.Run("endpoint from flag wins over profile", func(t *testing.T) {
		ctx := newEndpointTestContext(t)
		path := writeEndpointTestConfig(t, "oss-cn-hangzhou.aliyuncs.com", "", "cn-hangzhou")
		config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
		config.ConfigurePathFlag(ctx.Flags()).SetValue(path)
		config.EndpointFlag(ctx.Flags()).SetAssigned(true)
		config.EndpointFlag(ctx.Flags()).SetValue("oss-cn-beijing.aliyuncs.com")

		got, err := ParseAndGetEndpoint(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, "oss-cn-beijing.aliyuncs.com", got)
	})

	t.Run("profile endpoint used when no explicit endpoint", func(t *testing.T) {
		ctx := newEndpointTestContext(t)
		path := writeEndpointTestConfig(t, "oss-cn-shanghai.aliyuncs.com", "vpc", "cn-hangzhou")
		config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
		config.ConfigurePathFlag(ctx.Flags()).SetValue(path)

		got, err := ParseAndGetEndpoint(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, "oss-cn-shanghai.aliyuncs.com", got)
	})

	t.Run("endpoint-type vpc builds internal endpoint", func(t *testing.T) {
		ctx := newEndpointTestContext(t)
		path := writeEndpointTestConfig(t, "", "vpc", "cn-hangzhou")
		config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
		config.ConfigurePathFlag(ctx.Flags()).SetValue(path)

		got, err := ParseAndGetEndpoint(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, "oss-cn-hangzhou-internal.aliyuncs.com", got)
	})

	t.Run("region from args with vpc endpoint-type", func(t *testing.T) {
		ctx := newEndpointTestContext(t)
		path := writeEndpointTestConfig(t, "", "vpc", "cn-hangzhou")
		config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
		config.ConfigurePathFlag(ctx.Flags()).SetValue(path)

		got, err := ParseAndGetEndpoint(ctx, []string{"--region", "cn-shenzhen"})
		require.NoError(t, err)
		assert.Equal(t, "oss-cn-shenzhen-internal.aliyuncs.com", got)
	})

	t.Run("region from args without vpc is public", func(t *testing.T) {
		ctx := newEndpointTestContext(t)
		path := writeEndpointTestConfig(t, "", "", "cn-hangzhou")
		config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
		config.ConfigurePathFlag(ctx.Flags()).SetValue(path)

		got, err := ParseAndGetEndpoint(ctx, []string{"--region", "cn-shenzhen"})
		require.NoError(t, err)
		assert.Equal(t, "oss-cn-shenzhen.aliyuncs.com", got)
	})

	t.Run("endpoint-type flag vpc builds internal endpoint", func(t *testing.T) {
		ctx := newEndpointTestContext(t)
		path := writeEndpointTestConfig(t, "", "", "cn-beijing")
		config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
		config.ConfigurePathFlag(ctx.Flags()).SetValue(path)
		config.EndpointTypeFlag(ctx.Flags()).SetAssigned(true)
		config.EndpointTypeFlag(ctx.Flags()).SetValue("vpc")

		got, err := ParseAndGetEndpoint(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, "oss-cn-beijing-internal.aliyuncs.com", got)
	})

	t.Run("whitespace region from args returns error", func(t *testing.T) {
		ctx := newEndpointTestContext(t)
		path := writeEndpointTestConfig(t, "", "", "cn-hangzhou")
		config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
		config.ConfigurePathFlag(ctx.Flags()).SetValue(path)

		_, err := ParseAndGetEndpoint(ctx, []string{"--region", "  "})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing region")
	})

	t.Run("invalid region from args returns error", func(t *testing.T) {
		ctx := newEndpointTestContext(t)
		path := writeEndpointTestConfig(t, "", "", "cn-hangzhou")
		config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
		config.ConfigurePathFlag(ctx.Flags()).SetValue(path)

		_, err := ParseAndGetEndpoint(ctx, []string{"--region", "bad region!"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid region")
	})
}

func TestParseAndRunCommandFromCli(t *testing.T) {
	type args struct {
		ctx  *cli.Context
		args []string
	}

	// 保存原始的 parseAndRunCommandImpl 函数
	originalParseAndRunCommand := parseAndRunCommandImpl

	tests := []struct {
		name            string
		args            args
		wantErr         bool
		setup           func()       // 用于设置测试环境
		cleanup         func()       // 用于清理测试环境
		mockParseAndRun func() error // mock 命令执行
	}{
		{
			name: "Valid context with basic args",
			args: args{
				ctx:  createMockContextWithCmd("ls"),
				args: []string{"ls", "oss://test-bucket"},
			},
			wantErr: false,
			setup:   func() { setupMockProfile() },
			cleanup: func() { cleanupMockProfile() },
			mockParseAndRun: func() error {
				return nil
			},
		},
		{
			name: "Parser read error with unknown flag",
			args: args{
				ctx:  createContextWithInvalidFlags(),
				args: []string{"--unknown-flag", "value"},
			},
			wantErr: true, // 解析错误会失败
		},
		{
			name: "With proxy host",
			args: args{
				ctx:  createMockContextWithProxyFlags("cp"),
				args: []string{"cp", "src.txt", "oss://test-bucket/dest.txt", "--proxy-host", "proxy.example.com"},
			},
			wantErr: false,
			setup:   func() { setupMockProfile() },
			cleanup: func() { cleanupMockProfile() },
			mockParseAndRun: func() error {
				return nil
			},
		},
		{
			name: "With endpoint parsing",
			args: args{
				ctx:  createMockContextWithEndpointFlags("ls"),
				args: []string{"ls", "oss://test-bucket", "--endpoint", "oss-cn-beijing.aliyuncs.com"},
			},
			wantErr: false,
			setup:   func() { setupMockProfile() },
			cleanup: func() { cleanupMockProfile() },
			mockParseAndRun: func() error {
				return nil
			},
		},
		{
			name: "Force use HTTP",
			args: args{
				ctx:  createMockContextWithEndpointFlagsInsecure("ls"),
				args: []string{"ls", "oss://test-bucket", "--endpoint", "oss-cn-hangzhou.aliyuncs.com"},
			},
			wantErr: false,
			setup:   func() { setupMockProfile() },
			cleanup: func() { cleanupMockProfile() },
			mockParseAndRun: func() error {
				return nil
			},
		},
		{
			name: "Multiple_flag_values",
			args: args{
				ctx:  createMockContextWithMultipleFlags("cp"),
				args: []string{"cp", "src.txt", "oss://test-bucket/dest.txt", "--include", "*.txt", "--recursive"},
			},
			wantErr: false,
			setup:   func() { setupMockProfile() },
			cleanup: func() { cleanupMockProfile() },
			mockParseAndRun: func() error {
				return nil
			},
		},
		{
			name: "method and item set",
			args: args{
				ctx:  createMockBucketCname(),
				args: []string{"bucket-cname", "--method", "put", "--item", "certificate", "oss://ysg-cdntest", "./cert.xml"},
			},
			wantErr: false,
			setup:   func() { setupMockProfile() },
			cleanup: func() { cleanupMockProfile() },
			mockParseAndRun: func() error {
				return nil
			},
		},
		{
			name: "Parse and run command error",
			args: args{
				ctx:  createMockContextWithCmd("ls"),
				args: []string{"ls", "oss://test-bucket"},
			},
			wantErr: true,
			setup:   func() { setupMockProfile() },
			cleanup: func() { cleanupMockProfile() },
			mockParseAndRun: func() error {
				return fmt.Errorf("mock command execution error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			// 设置 mock 函数
			if tt.mockParseAndRun != nil {
				parseAndRunCommandImpl = tt.mockParseAndRun
			}

			// 在测试结束后恢复原始函数
			defer func() {
				parseAndRunCommandImpl = originalParseAndRunCommand
			}()

			// 保存原始的 os.Args
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			err := ParseAndRunCommandFromCli(tt.args.ctx, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAndRunCommandFromCli() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseAndRunCommandFromCli_ProfileFlagStrippedAndApplied(t *testing.T) {
	clearEndpointTestEnv(t)
	originalParseAndRunCommand := parseAndRunCommandImpl
	defer func() { parseAndRunCommandImpl = originalParseAndRunCommand }()
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
  "current": "default",
  "profiles": [
    {
      "name": "default",
      "mode": "AK",
      "access_key_id": "default-ak",
      "access_key_secret": "default-sk",
      "region_id": "cn-hangzhou",
      "output_format": "json",
      "language": "en"
    },
    {
      "name": "mac16@oyj",
      "mode": "AK",
      "access_key_id": "profile-ak",
      "access_key_secret": "profile-sk",
      "region_id": "cn-beijing",
      "output_format": "json",
      "language": "en"
    }
  ]
}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	ctx := newEndpointTestContext(t)
	cmd := &cli.Command{Name: "cp"}
	ctx.SetCommand(cmd)
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(path)

	var capturedArgs []string
	parseAndRunCommandImpl = func() error {
		capturedArgs = append([]string{}, os.Args...)
		return nil
	}

	err := ParseAndRunCommandFromCli(ctx, []string{
		"oss://oyj-test-role/file.vsix",
		"/tmp/Downloads",
		"--profile", "mac16@oyj",
	})
	require.NoError(t, err)

	joined := strings.Join(capturedArgs, " ")
	assert.NotContains(t, joined, "--profile")
	assert.NotContains(t, joined, "mac16@oyj")
	assert.Contains(t, joined, "--access-key-id")
	assert.Contains(t, joined, "profile-ak")
	assert.Contains(t, joined, "profile-sk")
	assert.Contains(t, joined, "oss://oyj-test-role/file.vsix")
	assert.Contains(t, joined, "/tmp/Downloads")
}

func TestParseAndRunCommandFromCli_ProfileEqualsForm(t *testing.T) {
	clearEndpointTestEnv(t)
	originalParseAndRunCommand := parseAndRunCommandImpl
	defer func() { parseAndRunCommandImpl = originalParseAndRunCommand }()
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
  "current": "default",
  "profiles": [
    {
      "name": "default",
      "mode": "AK",
      "access_key_id": "default-ak",
      "access_key_secret": "default-sk",
      "region_id": "cn-hangzhou",
      "output_format": "json",
      "language": "en"
    },
    {
      "name": "alt",
      "mode": "AK",
      "access_key_id": "alt-ak",
      "access_key_secret": "alt-sk",
      "region_id": "cn-shanghai",
      "output_format": "json",
      "language": "en"
    }
  ]
}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	ctx := newEndpointTestContext(t)
	ctx.SetCommand(&cli.Command{Name: "ls"})
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(path)

	var capturedArgs []string
	parseAndRunCommandImpl = func() error {
		capturedArgs = append([]string{}, os.Args...)
		return nil
	}

	err := ParseAndRunCommandFromCli(ctx, []string{"oss://bucket", "--profile=alt"})
	require.NoError(t, err)

	joined := strings.Join(capturedArgs, " ")
	assert.NotContains(t, joined, "--profile")
	assert.NotContains(t, joined, "--profile=alt")
	assert.Contains(t, joined, "alt-ak")
	assert.Contains(t, joined, "oss://bucket")
}

// Helper functions for creating mock contexts and profiles
func createMockContext() *cli.Context {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)

	// Add mock command
	cmd := &cli.Command{Name: "cp"}
	ctx.SetCommand(cmd)

	return ctx
}

func createMockContextWithCmd(cmdName string) *cli.Context {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)

	cmd := &cli.Command{Name: cmdName}
	ctx.SetCommand(cmd)

	return ctx
}

func createMockBucketCname() *cli.Context {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)

	// Mock bucket cname command
	cmd := &cli.Command{Name: "bucket-cname"}
	ctx.SetCommand(cmd)

	// Add flags for the command
	flag := cli.Flag{Name: "method"}
	flag.SetValue("put")
	ctx.Flags().Add(&flag)

	flag = cli.Flag{Name: "item"}
	flag.SetValue("certificate")
	ctx.Flags().Add(&flag)

	return ctx
}

func createContextWithInvalidFlags() *cli.Context {
	// Create context that will cause parser errors
	ctx := createMockContext()
	// Add flags that would cause parsing issues
	return ctx
}

func createMockContextWithProxyFlags(cmdName string) *cli.Context {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)

	cmd := &cli.Command{Name: cmdName}
	ctx.SetCommand(cmd)

	// Add proxy-host flag
	flag := cli.Flag{Name: "proxy-host"}
	ctx.Flags().Add(&flag)

	return ctx
}

func createMockContextWithEndpointFlags(cmdName string) *cli.Context {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)

	cmd := &cli.Command{Name: cmdName}
	ctx.SetCommand(cmd)

	// Add endpoint flag
	flag := cli.Flag{Name: "endpoint"}
	ctx.Flags().Add(&flag)

	return ctx
}

func createMockContextWithEndpointFlagsInsecure(cmdName string) *cli.Context {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)

	cmd := &cli.Command{Name: cmdName}
	ctx.SetCommand(cmd)

	// Add endpoint flag
	flag := cli.Flag{Name: "endpoint"}
	ctx.Flags().Add(&flag)
	ctx.SetInsecure(true)

	return ctx
}

func createMockContextWithMultipleFlags(cmdName string) *cli.Context {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)

	cmd := &cli.Command{Name: cmdName}
	ctx.SetCommand(cmd)

	// Add include flag
	flag := cli.Flag{Name: "include"}
	ctx.Flags().Add(&flag)

	// Add recursive flag
	flag2 := cli.Flag{Name: "recursive"}
	ctx.Flags().Add(&flag2)

	return ctx
}

func setupMockProfile() {
	// Setup mock profile and credential configuration
	// 注入有效的 AccessKeyId/AccessKeySecret
}

func cleanupMockProfile() {
	// Cleanup test environment
}
