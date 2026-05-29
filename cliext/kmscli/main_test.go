package kmscli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewKmscliCommand(t *testing.T) {
	cmd := NewKmscliCommand()
	if cmd == nil {
		t.Fatalf("NewKmscliCommand returned nil")
	}
	if cmd.Name != "kmscli" {
		t.Errorf("Name expected 'kmscli', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "AlibabaCloud KMS CLI" {
		t.Errorf("Short en expected 'AlibabaCloud KMS CLI', got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "KMS CLI工具" {
		t.Errorf("Short zh expected 'KMS CLI工具', got %s", zh)
	}
	expectedUsage := "aliyun kmscli secret getsecret <secretName>\naliyun kmscli openclaw getsecret"
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

func TestNewKmscliCommandMetadata(t *testing.T) {
	cmd := NewKmscliCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "kmscli" {
		t.Errorf("metadata name expected kmscli, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != "AlibabaCloud KMS CLI" {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != "KMS CLI工具" {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}

func TestKmscliCommandRunInstalledSkipNetwork(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	oldGetVersion := getLatestKmscliVersionFunc
	getConfigurePathFunc = func() string { return tmpDir }
	getLatestKmscliVersionFunc = func() (string, error) { return "v0.1.0", nil }
	defer func() {
		getConfigurePathFunc = oldGet
		getLatestKmscliVersionFunc = oldGetVersion
	}()

	execPath := filepath.Join(tmpDir, "kmscli")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	if err := os.WriteFile(execPath, []byte("dummy"), 0755); err != nil {
		t.Fatalf("write fake exec: %v", err)
	}
	cacheFile := filepath.Join(tmpDir, ".kmscli_version_check")
	if err := os.WriteFile(cacheFile, []byte("0"), 0644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	versionFile := filepath.Join(tmpDir, ".kmscli_version")
	if err := os.WriteFile(versionFile, []byte("v0.1.0"), 0644); err != nil {
		t.Fatalf("write version: %v", err)
	}

	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")
	defer os.Unsetenv("ALIBABA_CLOUD_IGNORE_PROFILE")

	cmd := NewKmscliCommand()
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
