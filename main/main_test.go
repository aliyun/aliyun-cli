package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
)

func TestMainWithNoArgs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetMainHooks(t, nil, nil, nil)

	Main([]string{})
}

func TestMainInterceptsMockBeforeLoadingProfile(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	if err := sysmock.Save(mockPath, []sysmock.Record{{
		Name:     "regions",
		Cmd:      "ecs DescribeRegions",
		ExitCode: 17,
		Stdout:   "mock regions\n",
		Times:    0,
	}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Setenv(sysmock.EnvMockEnabled, "true")
	t.Setenv(sysmock.EnvMockPath, mockPath)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "missing-home"))

	var stdout, stderr bytes.Buffer
	var exitCode int
	var exitCalled bool
	resetMainHooks(t, &stdout, &stderr, func(code int) {
		exitCalled = true
		exitCode = code
	})

	Main([]string{"ecs", "DescribeRegions"})

	if stdout.String() != "mock regions\n" {
		t.Fatalf("stdout = %q, want mock output", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !exitCalled {
		t.Fatalf("exit hook was not called")
	}
	if exitCode != 17 {
		t.Fatalf("exit code = %d, want 17", exitCode)
	}
}

func TestMainMockCommandBypassesEarlyIntercept(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	if err := os.WriteFile(mockPath, []byte("{invalid json"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv(sysmock.EnvMockEnabled, "true")
	t.Setenv(sysmock.EnvMockPath, mockPath)
	t.Setenv("HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	var exitCalled bool
	resetMainHooks(t, &stdout, &stderr, func(code int) {
		exitCalled = true
	})

	Main([]string{"mock", "path"})

	if stdout.String() != mockPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), mockPath+"\n")
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if exitCalled {
		t.Fatalf("exit hook was called, want management command to run normally")
	}
}

func TestMainMockCommandWithLeadingFlagsBypassesEarlyIntercept(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	if err := os.WriteFile(mockPath, []byte("{invalid json"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv(sysmock.EnvMockEnabled, "true")
	t.Setenv(sysmock.EnvMockPath, mockPath)
	t.Setenv("HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	var exitCalled bool
	resetMainHooks(t, &stdout, &stderr, func(code int) {
		exitCalled = true
	})

	Main([]string{"--profile", "default", "mock", "path"})

	if stdout.String() != mockPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), mockPath+"\n")
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if exitCalled {
		t.Fatalf("exit hook was called, want management command to run normally")
	}
}

func TestRootCommandRegistersMock(t *testing.T) {
	var stdout bytes.Buffer
	rootCmd := newRootCommand(config.NewProfile("default"), &stdout)

	if rootCmd.GetSubCommand("mock") == nil {
		t.Fatalf("mock subcommand is not registered")
	}
}

func TestMainMockDisabledFallsThrough(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	if err := sysmock.Save(mockPath, []sysmock.Record{{
		Name:     "disabled",
		Cmd:      "ecs DescribeRegions",
		ExitCode: 17,
		Stdout:   "mock regions\n",
		Times:    0,
	}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Setenv(sysmock.EnvMockEnabled, "false")
	t.Setenv(sysmock.EnvMockPath, mockPath)
	t.Setenv("HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	var exitCalled bool
	resetMainHooks(t, &stdout, &stderr, func(code int) {
		exitCalled = true
	})

	Main([]string{})

	if bytes.Contains(stdout.Bytes(), []byte("mock regions")) {
		t.Fatalf("stdout = %q, want normal flow without mock output", stdout.String())
	}
	if bytes.Contains(stderr.Bytes(), []byte("mock regions")) {
		t.Fatalf("stderr = %q, want normal flow without mock output", stderr.String())
	}
	if exitCalled {
		t.Fatalf("exit hook was called, want normal flow")
	}
}

func resetMainHooks(t *testing.T, stdout, stderr *bytes.Buffer, exitHook func(int)) {
	t.Helper()

	oldStdoutWriter := newStdoutWriter
	oldStderrWriter := newStderrWriter
	oldExit := exit
	t.Cleanup(func() {
		newStdoutWriter = oldStdoutWriter
		newStderrWriter = oldStderrWriter
		exit = oldExit
	})

	if stdout == nil {
		newStdoutWriter = cli.DefaultStdoutWriter
	} else {
		newStdoutWriter = func() io.Writer {
			return stdout
		}
	}
	if stderr == nil {
		newStderrWriter = cli.DefaultStderrWriter
	} else {
		newStderrWriter = func() io.Writer {
			return stderr
		}
	}
	if exitHook == nil {
		exit = func(int) {}
	} else {
		exit = exitHook
	}
}

func TestParseInSecure(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "Insecure flag present",
			args:     []string{"--insecure"},
			expected: true,
		},
		{
			name:     "Insecure flag with value",
			args:     []string{"--insecure", "true"},
			expected: true,
		},
		{
			name:     "Insecure flag absent",
			args:     []string{"--secure"},
			expected: false,
		},
		{
			name:     "Empty args",
			args:     []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := ParseInSecure(tt.args)
			if result != tt.expected {
				t.Errorf("ParseInSecure(%v) = %v; want %v", tt.args, result, tt.expected)
			}
		})
	}
}
