package cms2

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func TestNewCms2Command(t *testing.T) {
	cmd := NewCms2Command()
	if cmd == nil {
		t.Fatalf("NewCms2Command returned nil")
	}
	if cmd.Name != "cms2" {
		t.Errorf("Name expected 'cms2', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba Cloud CloudMonitor (CMS) CLI — manage monitoring integrations, Prometheus, alert rules, and PromQL." {
		t.Errorf("Short en mismatch: %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "阿里云云监控 CLI — 管理监控集成、Prometheus 实例、告警规则和 PromQL 查询。" {
		t.Errorf("Short zh mismatch: %s", zh)
	}
	if cmd.Usage != "aliyun cms2 <command> [args...] [options...]" {
		t.Errorf("Usage mismatch: %s", cmd.Usage)
	}
	if cmd.Hidden {
		t.Errorf("Hidden expected false")
	}
	if !cmd.EnableUnknownFlag {
		t.Errorf("EnableUnknownFlag expected true")
	}
	if !cmd.KeepArgs {
		t.Errorf("KeepArgs expected true")
	}
	if !cmd.SkipDefaultHelp {
		t.Errorf("SkipDefaultHelp expected true")
	}
	if cmd.Run == nil {
		t.Errorf("Run function should not be nil")
	}
}

func TestNewCms2Command_RunSuppressesExitError(t *testing.T) {
	saveAndRestore(t)
	cli.DisableExitCode()
	defer cli.EnableExitCode()

	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }

	writeExec(t, filepath.Join(tmpDir, "aliyuncms2"))
	_ = os.WriteFile(filepath.Join(tmpDir, ".cms2_version_check"),
		[]byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)

	configPath := filepath.Join(t.TempDir(), "config.json")
	_ = os.WriteFile(configPath, []byte(`{
		"current":"default",
		"profiles":[{"name":"default","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou"}]
	}`), 0644)

	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "exit 1")
	}

	ctx, stdout, _ := newTestCtx()
	config.AddFlags(ctx.Flags())
	cpFlag := ctx.Flags().Get(config.ConfigurePathFlagName)
	cpFlag.SetAssigned(true)
	cpFlag.SetValue(configPath)

	cmd := NewCms2Command()
	err := cmd.Run(ctx, []string{"version"})

	if err != nil {
		t.Fatalf("Run should return nil for ExitError (not propagate to framework), got: %v", err)
	}
	if stdout.Len() > 0 {
		t.Errorf("no ANSI error text should appear on stdout, got: %q", stdout.String())
	}
}
