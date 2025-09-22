package go_migrate

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

// 简单测试 NewGoMigrateCommand 基础属性与执行流程（不做文本内容比较）
func TestNewGoMigrateCommand_Basic(t *testing.T) {
	if runtime.GOOS == "windows" { // 简化：暂不在 Windows 下执行（需要真正 exe）
		t.Skip("skip on windows")
	}

	cmd := NewGoMigrateCommand()
	if cmd == nil {
		t.Fatal("command is nil")
	}
	if cmd.Name != "go-migrate" {
		t.Fatalf("unexpected name: %s", cmd.Name)
	}
	if !cmd.EnableUnknownFlag || !cmd.KeepArgs || !cmd.SkipDefaultHelp {
		t.Fatalf("flags not set correctly: %+v", cmd)
	}
	if cmd.Run == nil {
		t.Fatalf("Run should not be nil")
	}

	// stub 全局函数，避免真实网络和下载
	oldFetch := fetchRemoteContentFunc
	oldDownload := downloadGoMigrateFunc
	oldGetConf := getConfigurePathFunc
	oldCdn := cdnBaseUrlFunc
	defer func() {
		fetchRemoteContentFunc = oldFetch
		downloadGoMigrateFunc = oldDownload
		getConfigurePathFunc = oldGetConf
		cdnBaseUrlFunc = oldCdn
	}()

	fetchRemoteContentFunc = func(url string) (string, error) { return "1.0.0", nil }

	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	cdnBaseUrlFunc = func() string { return "http://example.invalid" }

	downloadGoMigrateFunc = func(osType, osArch, version, configurePath string) error {
		execName := getExecName(osType)
		p := filepath.Join(configurePath, execName)
		script := "#!/bin/sh\necho " + version + "\n"
		return os.WriteFile(p, []byte(script), 0o755)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, stderr)

	// 直接调用 Run，不关心输出文本，只验证不会报错且文件已创建
	if err := cmd.Run(ctx, []string{"--version"}); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	execName := getExecName(runtime.GOOS)
	if _, err := os.Stat(filepath.Join(tmpDir, execName)); err != nil {
		t.Fatalf("expected binary file created: %v", err)
	}
}
