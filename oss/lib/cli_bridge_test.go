package lib

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
)

func TestCliBridge(t *testing.T) {
	NewCommandBridge(configCommand.command)
}

func TestParseAndGetEndpoint(t *testing.T) {
	type args struct {
		ctx  *cli.Context
		args []string
	}
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	context := cli.NewCommandContext(w, stderr)
	flag := cli.Flag{
		Name: "endpoint",
	}
	flag.SetValue("oss-cn-hangzhou.aliyuncs.com")
	context.Flags().Add(&flag)

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Valid endpoint from args",
			args: args{
				ctx:  new(cli.Context),
				args: []string{"--endpoint", "oss-cn-shenzhen.aliyuncs.com"},
			},
			want:    "oss-cn-shenzhen.aliyuncs.com",
			wantErr: assert.NoError,
		},
		{
			name: "Valid region from args",
			args: args{
				ctx:  new(cli.Context),
				args: []string{"--region", "cn-shenzhen"},
			},
			want:    "oss-cn-shenzhen.aliyuncs.com",
			wantErr: assert.NoError,
		},
		{
			name: "Fetch endpoint flag from context",
			args: args{
				ctx: context,
			},
			want:    "oss-cn-hangzhou.aliyuncs.com",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAndGetEndpoint(tt.args.ctx, tt.args.args)
			if !tt.wantErr(t, err, fmt.Sprintf("ParseAndGetEndpoint(%v, %v)", tt.args.ctx, tt.args.args)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ParseAndGetEndpoint(%v, %v)", tt.args.ctx, tt.args.args)
		})
	}
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
