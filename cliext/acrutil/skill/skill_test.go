package skill

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

func TestNewSkillCommand(t *testing.T) {
	cmd := NewSkillCommand()
	if cmd == nil {
		t.Fatalf("NewSkillCommand returned nil")
	}
	if cmd.Name != "skill" {
		t.Errorf("Name: got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "ACR Skill Management" {
		t.Errorf("Short en: got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "ACR Skill管理" {
		t.Errorf("Short zh: got %s", zh)
	}
	if cmd.Usage != "acrutil skill <command> [args...]" {
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

// TestSkillExecute_UserAgentEnvPassed verifies the full chain from the skill
// subcommand config through binmgr.Execute, ensuring the correct User-Agent
// and compat-mode environment variables are injected into the subprocess.
func TestSkillExecute_UserAgentEnvPassed(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, errOut)

	m := binmgr.New(skillConfig, ctx)
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

	// Verify ALIBABA_CLOUD_ACR_SKILL is set correctly.
	expectedUA := "ALIBABA_CLOUD_ACR_SKILL=aliyun-cli/" + cli.Version
	if !strings.Contains(output, expectedUA) {
		t.Errorf("expected env %q in subprocess output, got:\n%s", expectedUA, output)
	}

	// Verify ALIBABA_CLOUD_ACR_SKILL_COMPAT_MODE is set correctly.
	expectedCompat := "ALIBABA_CLOUD_ACR_SKILL_COMPAT_MODE=aliyun acrutil skill"
	if !strings.Contains(output, expectedCompat) {
		t.Errorf("expected env %q in subprocess output, got:\n%s", expectedCompat, output)
	}
}

func TestSkillConfig(t *testing.T) {
	if skillConfig.Name != "acr-skill" {
		t.Errorf("Name: got %s", skillConfig.Name)
	}
	if skillConfig.BaseURL != "https://acr-public-asset.oss-cn-hangzhou.aliyuncs.com/acr-skill/" {
		t.Errorf("BaseURL: got %s", skillConfig.BaseURL)
	}
	if skillConfig.EnvCompatMode != "ALIBABA_CLOUD_ACR_SKILL_COMPAT_MODE" {
		t.Errorf("EnvCompatMode: got %s", skillConfig.EnvCompatMode)
	}
	if skillConfig.EnvCompatModeVal != "aliyun acrutil skill" {
		t.Errorf("EnvCompatModeVal: got %s", skillConfig.EnvCompatModeVal)
	}
	if skillConfig.EnvUserAgent != "ALIBABA_CLOUD_ACR_SKILL" {
		t.Errorf("EnvUserAgent: got %s", skillConfig.EnvUserAgent)
	}
}
