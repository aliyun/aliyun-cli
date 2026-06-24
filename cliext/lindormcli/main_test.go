package lindormcli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewLindormCliCommand(t *testing.T) {
	cmd := NewLindormCliCommand()
	if cmd == nil {
		t.Fatalf("NewLindormCliCommand returned nil")
	}
	if cmd.Name != "lindorm" {
		t.Errorf("Name expected 'lindorm', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "AlibabaCloud Lindorm Open API CLI" {
		t.Errorf("Short en expected 'AlibabaCloud Lindorm Open API CLI', got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "Lindorm Open API CLI工具" {
		t.Errorf("Short zh expected 'Lindorm Open API CLI工具', got %s", zh)
	}
	expectedUsage := "aliyun lindorm <command> [options]"
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

func TestNewLindormCliCommandMetadata(t *testing.T) {
	cmd := NewLindormCliCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "lindorm" {
		t.Errorf("metadata name expected lindorm, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != "AlibabaCloud Lindorm Open API CLI" {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != "Lindorm Open API CLI工具" {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}

func TestLindormCliCommandRunInstalledSkipNetwork(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	oldGetVersion := getLatestLindormCliVersionFunc
	getConfigurePathFunc = func() string { return tmpDir }
	getLatestLindormCliVersionFunc = func() (string, error) { return "v0.1.0", nil }
	defer func() {
		getConfigurePathFunc = oldGet
		getLatestLindormCliVersionFunc = oldGetVersion
	}()

	execPath := filepath.Join(tmpDir, "lindorm-open-api-cli")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	if err := os.WriteFile(execPath, []byte("dummy"), 0755); err != nil {
		t.Fatalf("write fake exec: %v", err)
	}
	cacheFile := filepath.Join(tmpDir, ".lindormcli_version_check")
	if err := os.WriteFile(cacheFile, []byte("0"), 0644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	versionFile := filepath.Join(tmpDir, ".lindormcli_version")
	if err := os.WriteFile(versionFile, []byte("v0.1.0"), 0644); err != nil {
		t.Fatalf("write version: %v", err)
	}

	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")
	defer os.Unsetenv("ALIBABA_CLOUD_IGNORE_PROFILE")

	cmd := NewLindormCliCommand()
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
