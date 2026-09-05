package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
)

//go:embed testdata/dump
var testDumpFS embed.FS

func TestDumpFilesCopiesEmbeddedTree(t *testing.T) {
	output := t.TempDir()
	dumpFiles(testDumpFS, "./testdata/dump", output)

	for _, name := range []string{"testdata/dump/root.txt", "testdata/dump/nested/child.txt"} {
		want, err := testDumpFS.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(output, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s = %q, want embedded bytes %q", name, got, want)
		}
	}
}

func TestDumpFilesMissingPathReturnsWithoutWriting(t *testing.T) {
	output := t.TempDir()
	dumpFiles(testDumpFS, "missing", output)
	entries, err := os.ReadDir(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unexpected output entries: %v", entries)
	}
}

func TestMainWithNoArgs(t *testing.T) {
	clearAgentDetectionEnv(t)
	t.Setenv("HOME", t.TempDir())
	resetMainHooks(t, nil, nil, nil)

	Main([]string{})
}

func TestMainExplicitLanguageOverridesProfileForCoreAndOpenAPIHelp(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		notWant string
	}{
		{
			name:    "core",
			args:    []string{"--language", "en", "version", "--help"},
			want:    "print current version",
			notWant: "打印当前版本号",
		},
		{
			name: "PascalCase OpenAPI",
			args: []string{
				"ecs", "DescribeInstances", "--version", "2014-05-26",
				"--help", "--cli-output", "json", "--no-cli-ai-mode", "--language", "en",
			},
			want:    "Queries the list of instances",
			notWant: "查询一台或多台实例的详细信息",
		},
		{
			name: "kebab OpenAPI",
			args: []string{
				"fc", "create-alias", "--api-version", "2023-03-30",
				"--help", "--cli-output", "json", "--no-cli-ai-mode", "--language", "en",
			},
			want:    "Creates an alias",
			notWant: "创建别名",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearAgentDetectionEnv(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			configDir := filepath.Join(home, ".aliyun")
			if err := os.MkdirAll(configDir, 0o755); err != nil {
				t.Fatal(err)
			}
			profile := []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou","language":"zh"}]}`)
			if err := os.WriteFile(filepath.Join(configDir, "config.json"), profile, 0o600); err != nil {
				t.Fatal(err)
			}

			var stdout, stderr bytes.Buffer
			resetMainHooks(t, &stdout, &stderr, nil)
			Main(test.args)

			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if !bytes.Contains(stdout.Bytes(), []byte(test.want)) {
				t.Fatalf("stdout does not contain explicit English output %q:\n%s", test.want, stdout.String())
			}
			if bytes.Contains(stdout.Bytes(), []byte(test.notWant)) {
				t.Fatalf("stdout contains profile-selected Chinese output %q:\n%s", test.notWant, stdout.String())
			}
		})
	}
}

func TestEffectiveLanguagePriority(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		profile string
		want    string
	}{
		{name: "separate explicit value", args: []string{"--language", "en"}, profile: "zh", want: "en"},
		{name: "equals explicit value", args: []string{"--language=zh"}, profile: "en", want: "zh"},
		{name: "explicit value is case insensitive", args: []string{"--language=EN"}, profile: "zh", want: "en"},
		{name: "unsupported explicit value uses English", args: []string{"--language", "fr"}, profile: "zh", want: "en"},
		{name: "profile fallback", profile: "zh", want: "zh"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := effectiveLanguage(test.args, test.profile); got != test.want {
				t.Fatalf("effectiveLanguage(%v, %q) = %q, want %q", test.args, test.profile, got, test.want)
			}
		})
	}
}

func clearAgentDetectionEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"CURSOR_AGENT", "CLAUDECODE", "CLAUDE_CODE", "GEMINI_CLI",
		"AUGMENT_AGENT", "OPENCODE", "OPENCODE_CLIENT", "CLINE_ACTIVE",
		"CODEX_SHELL", "CODEX_SANDBOX", "QODER_AGENT", "QODER_CLI", "AGENT",
		"ALIBABA_CLOUD_CLI_AI_MODE", "NO_COLOR",
	} {
		t.Setenv(key, "")
	}
}

func TestNewCommandContextDetectsAgentOnce(t *testing.T) {
	clearAgentDetectionEnv(t)

	ctx := newCommandContext(io.Discard, io.Discard)
	if ctx.IsAgent() {
		t.Fatalf("unexpected detected agent %q", ctx.AgentName())
	}

	t.Setenv("CURSOR_AGENT", "1")
	ctx = newCommandContext(io.Discard, io.Discard)
	if !ctx.IsAgent() || ctx.AgentName() != "cursor" {
		t.Fatalf("agent state = %v, %q", ctx.IsAgent(), ctx.AgentName())
	}

	// Detection is a startup snapshot; later environment mutations do not
	// silently change the context used by the active command.
	t.Setenv("CURSOR_AGENT", "")
	if !ctx.IsAgent() || ctx.AgentName() != "cursor" {
		t.Fatalf("agent snapshot changed = %v, %q", ctx.IsAgent(), ctx.AgentName())
	}
}

func TestMainMachineHelpJSON(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantHelpLevel string
		wantStyle     string
	}{
		{name: "root", args: []string{"--help", "--cli-output", "json", "--no-cli-ai-mode"}, wantHelpLevel: "root"},
		{name: "utils", args: []string{"utils", "--help", "--cli-output", "json", "--no-cli-ai-mode"}, wantHelpLevel: "utility"},
		{name: "utility leaf", args: []string{"utils", "list-supported-pricing-apis", "--help", "--cli-output", "json", "--no-cli-ai-mode"}, wantHelpLevel: "utility"},
		{name: "product", args: []string{"ecs", "--help", "--cli-output", "json", "--no-cli-ai-mode"}, wantHelpLevel: "product"},
		{name: "camel API", args: []string{"ecs", "DescribeInstances", "--version", "2014-05-26", "--help", "--cli-output", "json", "--no-cli-ai-mode"}, wantHelpLevel: "api", wantStyle: "camel"},
		{name: "kebab API", args: []string{"ecs", "describe-instances", "--api-version", "2014-05-26", "--help", "--cli-output", "json", "--no-cli-ai-mode"}, wantHelpLevel: "api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAgentDetectionEnv(t)
			t.Setenv("HOME", t.TempDir())
			var stdout, stderr bytes.Buffer
			resetMainHooks(t, &stdout, &stderr, nil)

			Main(tt.args)

			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			var document struct {
				SchemaVersion      string `json:"schemaVersion"`
				HelpLevel          string `json:"helpLevel"`
				ActiveParameterSet string `json:"activeParameterSet"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
				t.Fatalf("machine help is not JSON: %v\n%s", err, stdout.String())
			}
			if document.SchemaVersion != "v1" || document.HelpLevel != tt.wantHelpLevel {
				t.Fatalf("document = %#v, want schema v1 help level %s", document, tt.wantHelpLevel)
			}
			if document.ActiveParameterSet != tt.wantStyle {
				t.Fatalf("activeParameterSet = %q, want %q", document.ActiveParameterSet, tt.wantStyle)
			}
		})
	}
}

func TestMainDefaultHelpHidesPricingDiscoveryUtility(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "root text", args: []string{"--help", "--no-cli-ai-mode", "--language", "en"}},
		{name: "utils text", args: []string{"utils", "--help", "--no-cli-ai-mode", "--language", "en"}},
		{name: "root machine", args: []string{"--help", "--cli-output", "json", "--no-cli-ai-mode"}},
		{name: "utils machine", args: []string{"utils", "--help", "--cli-output", "json", "--no-cli-ai-mode"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAgentDetectionEnv(t)
			t.Setenv("HOME", t.TempDir())
			var stdout, stderr bytes.Buffer
			resetMainHooks(t, &stdout, &stderr, nil)

			Main(tt.args)

			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if bytes.Contains(stdout.Bytes(), []byte("list-supported-pricing-apis")) {
				t.Fatalf("default Help exposes hidden pricing utility:\n%s", stdout.String())
			}
		})
	}
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

func TestRootCommandRegistersRostran(t *testing.T) {
	var stdout bytes.Buffer
	rootCmd := newRootCommand(config.NewProfile("default"), &stdout)

	if rootCmd.GetSubCommand("rostran") == nil {
		t.Fatalf("rostran subcommand is not registered")
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
