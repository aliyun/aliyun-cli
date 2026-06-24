package agentbay

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

func TestNewAgentBayCommand(t *testing.T) {
	cmd := NewAgentBayCommand()
	if cmd == nil {
		t.Fatalf("NewAgentBayCommand returned nil")
	}
	if cmd.Name != "agentbay" {
		t.Errorf("Name expected 'agentbay', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "The AgentBay command-line interface (CLI) is a tool for the AgentBay service. This topic describes how to use the CLI to create, build, activate, and manage AgentBay custom images." {
		t.Errorf("Short en mismatch: %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "AgentBay CLI 是 AgentBay 服务的命令行工具，用于管理镜像、API Key、Skill 和认证。" {
		t.Errorf("Short zh mismatch: %s", zh)
	}
	if cmd.Usage != "aliyun agentbay [command] [args...] [options...]" {
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

func TestNewAgentBayCommandMetadata(t *testing.T) {
	cmd := NewAgentBayCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "agentbay" {
		t.Errorf("metadata name expected agentbay, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != cmd.Short.Get("en") {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != cmd.Short.Get("zh") {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}

func TestNewAgentBayCommandRun(t *testing.T) {
	tmpDir := t.TempDir()
	origConfig := getConfigurePathFunc
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	origLoadProfile := loadAgentBayProfileFunc
	origExec := execCommandFunc
	t.Cleanup(func() {
		getConfigurePathFunc = origConfig
		runtimeGOOSFunc = origGOOS
		runtimeGOARCHFunc = origGOARCH
		loadAgentBayProfileFunc = origLoadProfile
		execCommandFunc = origExec
	})

	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	loadAgentBayProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.AK, AccessKeyId: "ak", AccessKeySecret: "sk"}, nil
	}
	writeExecutable(t, filepath.Join(tmpDir, "agentbay"), "#!/bin/sh\n")
	if err := os.WriteFile(filepath.Join(tmpDir, ".agentbay_version_check"), []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644); err != nil {
		t.Fatalf("write version cache: %v", err)
	}
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "exit 0")
	}

	ctx, _, _ := newOriginCtx()
	cmd := NewAgentBayCommand()
	if err := cmd.Run(ctx, []string{"session", "list"}); err != nil {
		t.Fatalf("command run: %v", err)
	}
}
