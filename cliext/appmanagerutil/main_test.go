package appmanagerutil

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewAppManagerCommand(t *testing.T) {
	cmd := NewAppManagerCommand()
	if cmd == nil {
		t.Fatalf("NewAppManagerCommand returned nil")
	}
	if cmd.Name != "appmanager" {
		t.Errorf("Name expected 'appmanager', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba Cloud AppManager CLI" {
		t.Errorf("Short en expected 'Alibaba Cloud AppManager CLI', got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "阿里云应用管理CLI工具" {
		t.Errorf("Short zh expected '阿里云应用管理CLI工具', got %s", zh)
	}
	if cmd.Usage != "aliyun appmanager <command> [args...]" {
		t.Errorf("Usage expected 'aliyun appmanager <command> [args...]', got %s", cmd.Usage)
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

func TestNewAppManagerCommandMetadata(t *testing.T) {
	cmd := NewAppManagerCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "appmanager" {
		t.Errorf("metadata name expected appmanager, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != "Alibaba Cloud AppManager CLI" {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != "阿里云应用管理CLI工具" {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}

// Re-execute only this helper instead of requiring Python or an AppManager binary.
func TestAppManagerVersionProcess(t *testing.T) {
	if os.Getenv("ALIYUN_TEST_APPMANAGER_VERSION") != "1" {
		return
	}
	fmt.Fprintln(os.Stdout, "appmanager offline-test")
	os.Exit(0)
}

func TestAppManagerCommandRunInstalledSkipNetwork(t *testing.T) {
	home := t.TempDir()
	prepareConfig(t, home, "en")
	tmpDir := filepath.Join(home, ".aliyun")
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	t.Cleanup(func() { getConfigurePathFunc = oldGet })
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ALIYUN_TEST_APPMANAGER_VERSION", "1")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, stderr)
	installed := NewContext(ctx)
	if err := installed.InitBasicInfo(); err != nil {
		t.Fatal(err)
	}
	// Populate the real platform-specific layout. These placeholders are never
	// executed; missing Python or entry points must not trigger an installation.
	for _, path := range []string{installed.getEmbeddedPythonPath(), installed.venvPythonPath, installed.execFilePath} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// A fresh cache must skip upgrades as well as first-time installation.
	if err := os.WriteFile(installed.checkVersionCacheFilePath, []byte(fmt.Sprintf("%d", timeNowFunc().Unix())), 0644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}

	originalClient := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = originalClient })
	http.DefaultClient = &http.Client{Transport: pythonDownloadTransport(func(r *http.Request) (*http.Response, error) {
		t.Errorf("installed AppManager attempted a network request: %s", r.URL)
		return nil, fmt.Errorf("network disabled in installed AppManager test")
	})}

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	processContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	originalCommand := execCommandFunc
	t.Cleanup(func() { execCommandFunc = originalCommand })
	calls := 0
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		calls++
		if name != installed.execFilePath || !reflect.DeepEqual(args, []string{"--version"}) {
			t.Fatalf("unexpected process (Python/pip must not run): %q %v", name, args)
		}
		return exec.CommandContext(processContext, executable, "-test.run=^TestAppManagerVersionProcess$")
	}

	cmd := NewAppManagerCommand()
	if err := cmd.Run(ctx, []string{"--version"}); err != nil {
		t.Fatalf("Run failed: %v; stderr: %s", err, stderr.String())
	}
	if calls != 1 {
		t.Fatalf("process calls = %d, want exactly one AppManager invocation", calls)
	}
	if got := stdout.String(); got != "appmanager offline-test\n" {
		t.Fatalf("version output = %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}
