package computenestutil

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewComputenestCommand(t *testing.T) {
	cmd := NewComputenestCommand()
	if cmd == nil {
		t.Fatalf("NewComputenestCommand returned nil")
	}
	if cmd.Name != "computenest-cli" {
		t.Errorf("Name expected 'computenest-cli', got %s", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatalf("Short i18n text nil")
	}
	if en := cmd.Short.Get("en"); en != "Alibaba Cloud ComputeNest CLI" {
		t.Errorf("Short en expected 'Alibaba Cloud ComputeNest CLI', got %s", en)
	}
	if zh := cmd.Short.Get("zh"); zh != "阿里云计算巢CLI工具" {
		t.Errorf("Short zh expected '阿里云计算巢CLI工具', got %s", zh)
	}
	if cmd.Usage != "aliyun computenest-cli <command> [args...]" {
		t.Errorf("Usage expected 'aliyun computenest-cli <command> [args...]', got %s", cmd.Usage)
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

func TestNewComputenestCommandMetadata(t *testing.T) {
	cmd := NewComputenestCommand()
	metaMap := map[string]*cli.Metadata{}
	cmd.GetMetadata(metaMap)
	m, ok := metaMap[cmd.Name]
	if !ok {
		t.Fatalf("metadata for %s not found", cmd.Name)
	}
	if m.Name != "computenest-cli" {
		t.Errorf("metadata name expected computenest-cli, got %s", m.Name)
	}
	if m.Usage != cmd.Usage {
		t.Errorf("metadata usage mismatch")
	}
	if m.Hidden != cmd.Hidden {
		t.Errorf("metadata hidden mismatch")
	}
	if se := m.Short["en"]; se != "Alibaba Cloud ComputeNest CLI" {
		t.Errorf("metadata short en mismatch: %s", se)
	}
	if sz := m.Short["zh"]; sz != "阿里云计算巢CLI工具" {
		t.Errorf("metadata short zh mismatch: %s", sz)
	}
}

func TestComputenestCommandRunInstalledSkipNetwork(t *testing.T) {
	// 准备临时目录作为配置路径
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	t.Setenv("PATH", t.TempDir())
	fixture := NewContext(cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{}))
	if err := fixture.InitBasicInfo(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{fixture.execFilePath, fixture.venvPythonPath, fixture.getEmbeddedPythonPath()} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	originalTransport := http.DefaultTransport
	http.DefaultTransport = pythonDownloadTransport(func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected network request: %s", r.URL)
		return nil, fmt.Errorf("network forbidden")
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	originalCommand := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		if name != fixture.execFilePath || len(args) != 1 || args[0] != "--version" {
			t.Fatalf("unexpected subprocess: %s %v", name, args)
		}
		calls++
		return exec.Command(executable, "-test.run=^TestComputenestInstalledProcess$", "--", "computenest-fixture")
	}
	t.Cleanup(func() { execCommandFunc = originalCommand })

	// 创建版本缓存文件，跳过远程版本检查（占位符URL暂不可用）
	cacheFile := filepath.Join(tmpDir, ".computenest_version_check")
	if err := os.WriteFile(cacheFile, []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}

	// 创建 config.json 以便 PrepareEnv 跳过写入
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(`{"current":"default"}`), 0644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	cmd := NewComputenestCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, stderr)

	if err := cmd.Run(ctx, []string{"--version"}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || strings.TrimSpace(stdout.String()) != "dummy" {
		t.Fatalf("calls=%d stdout=%q stderr=%q", calls, stdout, stderr)
	}
}

func TestComputenestInstalledProcess(t *testing.T) {
	if os.Args[len(os.Args)-1] != "computenest-fixture" {
		return
	}
	fmt.Fprintln(os.Stdout, "dummy")
	os.Exit(0)
}
