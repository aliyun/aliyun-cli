package diagnosis

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cliext/acrutil/binmgr"
)

func TestNewDiagnosisCommand(t *testing.T) {
	cmd := NewDiagnosisCommand()
	if cmd == nil {
		t.Fatalf("NewDiagnosisCommand returned nil")
	}
	if cmd.Name != "diagnosis" {
		t.Errorf("Name: got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "ACR Instance Diagnosis" {
		t.Errorf("Short en: got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "ACR 实例诊断" {
		t.Errorf("Short zh: got %s", zh)
	}
	if cmd.Usage != "acrutil diagnosis [domain] [options...]" {
		t.Errorf("Usage: got %s", cmd.Usage)
	}
	if !cmd.EnableUnknownFlag {
		t.Errorf("EnableUnknownFlag should be true")
	}
	if !cmd.KeepArgs {
		t.Errorf("KeepArgs should be true")
	}
	if !cmd.SkipDefaultHelp {
		t.Errorf("SkipDefaultHelp should be true")
	}
	if cmd.Run == nil {
		t.Errorf("Run should not be nil")
	}
}

// TestDiagnosisExecute_UserAgentEnvPassed verifies the full chain from the
// diagnosis subcommand config through binmgr.Execute, ensuring the correct
// User-Agent and compat-mode environment variables are injected into the
// subprocess.
func TestDiagnosisExecute_UserAgentEnvPassed(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, errOut)

	m := binmgr.New(diagnosisConfig, ctx)
	m.SetExecFilePathForTest("/no/op")
	m.SetExecCommandForTest(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if runtime.GOOS == "windows" {
			return exec.CommandContext(ctx, "cmd", "/c", "set")
		}
		return exec.CommandContext(ctx, "env")
	})

	if err := m.Execute(context.Background(), nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := out.String()

	// Verify ALIBABA_CLOUD_CR_DIAG is set correctly.
	expectedUA := "ALIBABA_CLOUD_CR_DIAG=aliyun-cli/" + cli.Version
	if !strings.Contains(output, expectedUA) {
		t.Errorf("expected env %q in subprocess output, got:\n%s", expectedUA, output)
	}

	// Verify ALIBABA_CLOUD_CR_DIAG_COMPAT_MODE is set correctly.
	expectedCompat := "ALIBABA_CLOUD_CR_DIAG_COMPAT_MODE=aliyun acrutil diagnosis"
	if !strings.Contains(output, expectedCompat) {
		t.Errorf("expected env %q in subprocess output, got:\n%s", expectedCompat, output)
	}
}

func TestDiagnosisConfig(t *testing.T) {
	if diagnosisConfig.Name != "diagnosis" {
		t.Errorf("Name: got %s", diagnosisConfig.Name)
	}
	if diagnosisConfig.BaseURL != "https://acr-public-asset.oss-cn-hangzhou.aliyuncs.com/cr-diagnosis/" {
		t.Errorf("BaseURL: got %s", diagnosisConfig.BaseURL)
	}
	if diagnosisConfig.EnvCompatMode != "ALIBABA_CLOUD_CR_DIAG_COMPAT_MODE" {
		t.Errorf("EnvCompatMode: got %s", diagnosisConfig.EnvCompatMode)
	}
	if diagnosisConfig.EnvCompatModeVal != "aliyun acrutil diagnosis" {
		t.Errorf("EnvCompatModeVal: got %s", diagnosisConfig.EnvCompatModeVal)
	}
	if diagnosisConfig.EnvUserAgent != "ALIBABA_CLOUD_CR_DIAG" {
		t.Errorf("EnvUserAgent: got %s", diagnosisConfig.EnvUserAgent)
	}
}

// TestDiagnosisConfig_StripFlags verifies that diagnosisConfig opts out of
// stripping the long flag names cr-diagnosis owns (--mode/--yes/--quiet) while
// still stripping genuine parent-only flags (e.g. --profile).
func TestDiagnosisConfig_StripFlags(t *testing.T) {
	for _, name := range []string{"mode", "yes", "quiet"} {
		if diagnosisConfig.StripFlags[name] {
			t.Errorf("%q must NOT be stripped (owned by cr-diagnosis)", name)
		}
	}
	for _, name := range []string{"profile", "ram-role-arn", "config-path"} {
		if !diagnosisConfig.StripFlags[name] {
			t.Errorf("%q should still be stripped (parent-only flag)", name)
		}
	}
}

// TestDiagnosisRemoveFlags_PassThroughOwnFlags verifies that flags owned by the
// cr-diagnosis CLI (--mode/--yes/--quiet) are forwarded to the child even though
// they collide with parent-CLI flag names, while a genuine parent-only flag
// (--profile) is stripped together with its value.
func TestDiagnosisRemoveFlags_PassThroughOwnFlags(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, errOut)
	m := binmgr.New(diagnosisConfig, ctx)

	args := []string{
		"example.cn-hangzhou.cr.aliyuncs.com",
		"--mode", "network",
		"--yes",
		"--quiet",
		"--profile", "default",
	}
	got, err := m.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("RemoveFlagsForMainCli: %v", err)
	}

	for _, want := range []string{"--mode", "network", "--yes", "--quiet", "example.cn-hangzhou.cr.aliyuncs.com"} {
		if !containsArg(got, want) {
			t.Errorf("expected %q to be passed through, got: %v", want, got)
		}
	}
	for _, unwanted := range []string{"--profile", "default"} {
		if containsArg(got, unwanted) {
			t.Errorf("%q should be stripped, got: %v", unwanted, got)
		}
	}
}

func containsArg(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}
