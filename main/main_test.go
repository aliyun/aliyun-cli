package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// setupBrokenConfig materializes one of the broken configuration states and
// returns the config file path. A6 (no rewrite of a corrupt file) is asserted by
// callers that compare digests before/after.
func setupBrokenConfig(t *testing.T, failure string) (configPath string) {
	t.Helper()
	clearAgentDetectionEnv(t)
	t.Setenv(sysmock.EnvMockEnabled, "false")
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	t.Setenv("USERPROFILE", home)
	configDir := filepath.Join(home, ".aliyun")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath = filepath.Join(configDir, "config.json")
	switch failure {
	case "config is directory":
		if err := os.Mkdir(configPath, 0o755); err != nil {
			t.Fatal(err)
		}
	case "missing current profile":
		data := `{"current":"missing","profiles":[{"name":"work","mode":"AK","access_key_id":"ak","access_key_secret":"TOPSECRET_SHOULD_NOT_LEAK","region_id":"cn-hangzhou"}]}`
		if err := os.WriteFile(configPath, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	default: // invalid JSON
		if err := os.WriteFile(configPath, []byte(`{invalid JSON`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return configPath
}

func runMain(t *testing.T, args ...string) (stdout, stderr bytes.Buffer, exitCodes []int) {
	t.Helper()
	resetMainHooks(t, &stdout, &stderr, func(code int) { exitCodes = append(exitCodes, code) })
	Main(args)
	return stdout, stderr, exitCodes
}

// disableExitCode keeps cli.Exit (used by Command.processError) from terminating
// the test binary; the rendered error output is asserted instead, and real exit
// codes are covered by the built-binary smoke test.
func disableExitCode(t *testing.T) {
	t.Helper()
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
}

func hasNonZero(codes []int) bool {
	for _, code := range codes {
		if code != 0 {
			return true
		}
	}
	return false
}

func fileDigest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// Basic commands must remain usable even when the configuration cannot be
// parsed or does not select an existing profile (A1).
func TestMainConfigLoadFailureKeepsBasicCommandsUsable(t *testing.T) {
	for _, failure := range []string{"invalid JSON", "config is directory", "missing current profile"} {
		t.Run(failure, func(t *testing.T) {
			setupBrokenConfig(t, failure)
			for _, args := range [][]string{
				{"version"},
				{"--help"},
				{"help"},
				{"ecs", "--help"},
			} {
				stdout, stderr, exitCodes := runMain(t, args...)
				if hasNonZero(exitCodes) {
					t.Fatalf("%v: exit codes = %v, want success (stderr=%q)", args, exitCodes, stderr.String())
				}
				if stderr.Len() != 0 {
					t.Fatalf("%v: stderr = %q, want empty", args, stderr.String())
				}
				if stdout.Len() == 0 {
					t.Fatalf("%v: stdout is empty, want version/help output", args)
				}
			}
		})
	}
}

// In the "current points to a missing profile" state (S3) configure must be able
// to repair the selection instead of being blocked at startup (A2).
func TestMainConfigLoadFailureAllowsConfigureRepair(t *testing.T) {
	configPath := setupBrokenConfig(t, "missing current profile")

	stdout, stderr, exitCodes := runMain(t, "configure", "switch", "--profile", "work")
	if hasNonZero(exitCodes) {
		t.Fatalf("configure switch failed: exit codes = %v, stderr = %q", exitCodes, stderr.String())
	}
	if !strings.Contains(stdout.String(), "work") {
		t.Fatalf("stdout = %q, want confirmation mentioning work", stdout.String())
	}

	repaired, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(repaired), `"current": "work"`) && !strings.Contains(string(repaired), `"current":"work"`) {
		t.Fatalf("config current not repaired to work:\n%s", repaired)
	}

	stdout, stderr, exitCodes = runMain(t, "configure", "list")
	if hasNonZero(exitCodes) {
		t.Fatalf("configure list after repair failed: %v, stderr = %q", exitCodes, stderr.String())
	}
	if !strings.Contains(stdout.String(), "work") {
		t.Fatalf("configure list = %q, want work profile", stdout.String())
	}
}

// When the file is unparseable (S4) configure and its subcommands must still
// execute and report an actionable, path-bearing error (A2/A4) instead of being
// blocked before routing.
func TestMainConfigLoadFailureConfigureReportsActionableError(t *testing.T) {
	for _, failure := range []string{"invalid JSON", "config is directory"} {
		t.Run(failure, func(t *testing.T) {
			configPath := setupBrokenConfig(t, failure)
			disableExitCode(t)
			before := ""
			if failure != "config is directory" {
				before = fileDigest(t, configPath)
			}

			for _, args := range [][]string{
				{"configure", "list"},
				{"configure"},
			} {
				stdout, stderr, _ := runMain(t, args...)
				if stdout.Len() != 0 {
					t.Fatalf("%v: stdout = %q, want empty", args, stdout.String())
				}
				msg := stderr.String()
				if !strings.Contains(msg, "load configure failed") &&
					!strings.Contains(msg, "init config failed") &&
					!strings.Contains(msg, "parsing config file") &&
					!strings.Contains(msg, "reading config from") {
					t.Fatalf("%v: stderr = %q, want actionable config error", args, msg)
				}
			}

			// A6: a corrupt/unreadable config must not be rewritten.
			if failure != "config is directory" {
				if after := fileDigest(t, configPath); after != before {
					t.Fatalf("config.json mutated by configure: %s -> %s", before, after)
				}
			}
		})
	}
}

// Real API calls must still fail with a configuration error (never silently run
// unauthenticated) and must leave a corrupt file untouched (A5/A6).
func TestMainConfigLoadFailureStillBlocksAPICalls(t *testing.T) {
	for _, failure := range []string{"invalid JSON", "config is directory", "missing current profile"} {
		t.Run(failure, func(t *testing.T) {
			configPath := setupBrokenConfig(t, failure)
			disableExitCode(t)
			before := ""
			if failure != "config is directory" {
				before = fileDigest(t, configPath)
			}

			stdout, stderr, _ := runMain(t, "ecs", "DescribeInstances", "--version", "2014-05-26")
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty for config failure", stdout.String())
			}
			msg := stderr.String()
			if failure == "missing current profile" {
				if !strings.Contains(msg, "unknown profile missing") {
					t.Fatalf("stderr = %q, want unknown profile error", msg)
				}
			} else if !strings.Contains(msg, "init config failed") && !strings.Contains(msg, "parsing config file") && !strings.Contains(msg, "reading config from") {
				t.Fatalf("stderr = %q, want config load error", msg)
			}

			if failure != "config is directory" {
				if after := fileDigest(t, configPath); after != before {
					t.Fatalf("config.json mutated by API call: %s -> %s", before, after)
				}
			}
		})
	}
}

// AI mode must still return the established structured profile-configuration
// error for config failures without echoing configuration values.
func TestMainConfigLoadFailureAIModeStructuredError(t *testing.T) {
	setupBrokenConfig(t, "missing current profile")
	t.Setenv("ALIBABA_CLOUD_CLI_AI_MODE", "1")
	disableExitCode(t)

	stdout, stderr, _ := runMain(t, "ecs", "DescribeInstances", "--version", "2014-05-26")
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	var envelope cli.AgentErrorEnvelope
	if err := json.Unmarshal(stderr.Bytes(), &envelope); err != nil {
		t.Fatalf("AI error is not JSON: %v\nstderr = %q", err, stderr.String())
	}
	if envelope.SchemaVersion != "v1" {
		t.Fatalf("schemaVersion = %q, want v1 (%+v)", envelope.SchemaVersion, envelope)
	}
	if envelope.Message != "profile configuration failed" {
		t.Fatalf("message = %q, want profile configuration failed (%+v)", envelope.Message, envelope)
	}
	if envelope.ErrorCode != "ConfigurationError" {
		t.Fatalf("error_code = %q, want ConfigurationError (%+v)", envelope.ErrorCode, envelope)
	}
	if envelope.Recovery.Action != "check_configuration" {
		t.Fatalf("recovery.action = %q, want check_configuration (%+v)", envelope.Recovery.Action, envelope)
	}
	if envelope.Recovery.Command != "aliyun configure" {
		t.Fatalf("recovery.command = %q, want 'aliyun configure' (%+v)", envelope.Recovery.Command, envelope)
	}
	if strings.Contains(stderr.String(), "TOPSECRET_SHOULD_NOT_LEAK") {
		t.Fatalf("AI error leaked credential value: %q", stderr.String())
	}
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
			// Assert host-owned localized Help text, not mutable API metadata prose.
			want:    "inspect this API's structure: request parameters (default) or response schema",
			notWant: "查看该 API 的结构：request 参数（默认）或 response 响应结构",
		},
		{
			name: "kebab OpenAPI",
			args: []string{
				"fc", "create-alias", "--api-version", "2023-03-30",
				"--help", "--cli-output", "json", "--no-cli-ai-mode", "--language", "en",
			},
			want:    "inspect request parameters or the response schema",
			notWant: "查看 request 请求参数或 response 响应结构",
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
		"CODEX_SHELL", "CODEX_SANDBOX", "QODER_AGENT", "QODER_CLI", "QODERCN_CLI", "AGENT",
		"WORKBUDDY_APP_NAME", "TRAE_BRAND_NAME", "HERMES_AGENT",
		"ALIBABA_CLOUD_CLI_AI_MODE", "ALIBABA_CLOUD_CLI_AGENT_INTEGRATION", "NO_COLOR",
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

// Root help exercises the public query/output path without credentials or API
// requests. Keep these cases in both normal and packed build test runs.
func TestMainQueryNumericRegressions(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "distinct large integers",
			query: "`9007199254740992` == `9007199254740993`",
			want:  "false",
		},
		{
			name:  "empty average is null",
			query: "avg(`[]`)",
			want:  "null",
		},
		{
			name:  "empty average does not equal a number",
			query: "avg(`[]`) == `100`",
			want:  "false",
		},
		{
			name:  "empty average equals null",
			query: "avg(`[]`) == `null`",
			want:  "true",
		},
		{
			name:  "empty average does not select a record",
			query: "`[{\"id\":\"empty\",\"values\":[]},{\"id\":\"match\",\"values\":[100]},{\"id\":\"other\",\"values\":[200]}]`[?avg(values) == `100`].id",
			want:  `["match"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAgentDetectionEnv(t)
			t.Setenv(sysmock.EnvMockEnabled, "false")
			t.Setenv("GENERATE_METADATA", "")
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("HOMEDRIVE", "")
			t.Setenv("HOMEPATH", "")
			t.Setenv("USERPROFILE", home)

			var stdout, stderr bytes.Buffer
			resetMainHooks(t, &stdout, &stderr, func(code int) {
				t.Fatalf("unexpected exit(%d)", code)
			})
			Main([]string{"--help", "--cli-output", "json", "--cli-query", tt.query})

			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			var compact bytes.Buffer
			if err := json.Compact(&compact, stdout.Bytes()); err != nil {
				t.Fatalf("query output is not JSON: %v; stdout = %q", err, stdout.String())
			}
			if got := compact.String(); got != tt.want {
				t.Fatalf("query %q: got %s, want %s", tt.query, got, tt.want)
			}
		})
	}
}
