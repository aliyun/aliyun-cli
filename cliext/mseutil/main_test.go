package mseutil

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewMseutilCommand(t *testing.T) {
	cmd := NewMseutilCommand()
	if cmd == nil {
		t.Fatalf("NewMseutilCommand returned nil")
	}
	if cmd.Name != "mseutil" {
		t.Errorf("Name expected 'mseutil', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba Cloud MSE utility for diagnosing Nacos/ZooKeeper instances" {
		t.Errorf("Short en mismatch: %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "阿里云 MSE 诊断工具（Nacos/ZooKeeper）" {
		t.Errorf("Short zh mismatch: %s", zh)
	}
	if cmd.Usage != "aliyun mseutil <command> [args...]" {
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

func TestNewMseutilCommandMetadata(t *testing.T) {
	cmd := NewMseutilCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "mseutil" {
		t.Errorf("metadata name expected mseutil, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
}

func TestMseutilCommandRunInstalledSkipNetwork(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	oldGetVersion := getRemoteBinaryVersionFunc
	getConfigurePathFunc = func() string { return tmpDir }
	getRemoteBinaryVersionFunc = func(url string) (string, error) { return "etag-v1", nil }
	defer func() {
		getConfigurePathFunc = oldGet
		getRemoteBinaryVersionFunc = oldGetVersion
	}()

	execPath := filepath.Join(tmpDir, "mseutil")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	if err := os.WriteFile(execPath, []byte("dummy"), 0755); err != nil {
		t.Fatalf("write fake exec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".mseutil_version_check"), []byte("0"), 0644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".mseutil_version"), []byte("etag-v1"), 0644); err != nil {
		t.Fatalf("write version: %v", err)
	}

	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")
	defer os.Unsetenv("ALIBABA_CLOUD_IGNORE_PROFILE")

	cmd := NewMseutilCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, stderr)

	err := cmd.Run(ctx, []string{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	errStr := err.Error()
	if !bytes.Contains([]byte(errStr), []byte("profile default is not configure yet")) &&
		!bytes.Contains([]byte(errStr), []byte("can't get credential")) &&
		!bytes.Contains([]byte(errStr), []byte("config failed")) {
		t.Errorf("unexpected error: %v", err)
	}
}
