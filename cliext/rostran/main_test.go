package rostran

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewRostranCommand(t *testing.T) {
	cmd := NewRostranCommand()
	if cmd == nil {
		t.Fatalf("NewRostranCommand returned nil")
	}
	if cmd.Name != "rostran" {
		t.Errorf("Name expected 'rostran', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "ROS Transform Tool" {
		t.Errorf("Short en expected 'ROS Transform Tool', got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "ROS 模板转换工具" {
		t.Errorf("Short zh expected 'ROS 模板转换工具', got %s", zh)
	}
	expectedUsage := "aliyun rostran <command> [flags]\n  aliyun rostran upgrade"
	if cmd.Usage != expectedUsage {
		t.Errorf("Usage expected %q, got %q", expectedUsage, cmd.Usage)
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

func TestNewRostranCommandMetadata(t *testing.T) {
	cmd := NewRostranCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "rostran" {
		t.Errorf("metadata name expected rostran, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != "ROS Transform Tool" {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != "ROS 模板转换工具" {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}

func TestRostranCommandRunInstalledSkipNetwork(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	execPath := filepath.Join(tmpDir, "rostran", "rostran")
	if err := os.MkdirAll(filepath.Dir(execPath), 0755); err != nil {
		t.Fatalf("mkdir fake exec dir: %v", err)
	}
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\necho dummy\n"), 0755); err != nil {
		t.Fatalf("write fake exec: %v", err)
	}

	cacheFile := filepath.Join(tmpDir, ".rostran_version_check")
	if err := os.WriteFile(cacheFile, []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(`{"current":"default"}`), 0644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	cmd := NewRostranCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, stderr)

	err := cmd.Run(ctx, []string{"--version"})
	if err != nil {
		t.Logf("Run returned error (may be expected on some platforms): %v", err)
	}
}
