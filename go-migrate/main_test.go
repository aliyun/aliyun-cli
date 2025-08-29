package go_migrate

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

// helper 创建一个��执行脚本(仅 *nix) 输出给定版本
func createExecScript(t *testing.T, dir, name, version string) string {
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\necho " + version + "\n"
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		// 尝试普通写入
		if err2 := os.WriteFile(path, []byte(content), 0644); err2 != nil {
			t.Fatalf("write script failed: %v %v", err, err2)
		}
	}
	return path
}

func newTestContext() (*cli.Context, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return cli.NewCommandContext(stdout, stderr), stdout, stderr
}

// 重置全局可注入函数
func restoreGlobals() {
	fetchRemoteContentFunc = fetchRemoteContent
	downloadGoMigrateFunc = DownloadGoMigrate
	getConfigurePathFunc = func() string { return configPathBackup }
}

var configPathBackup = ""

func TestGoMigrate_NotInstalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows 环境下脚本模拟复杂，跳过")
	}
	ctx, stdout, _ := newTestContext()
	tmpDir := t.TempDir()
	configPathBackup = tmpDir
	getConfigurePathFunc = func() string { return tmpDir }

	remoteVersion := "1.2.3"
	fetchCalls := 0
	fetchRemoteContentFunc = func(url string) (string, error) {
		fetchCalls++
		return remoteVersion, nil
	}
	downloadCalls := 0
	downloadGoMigrateFunc = func(osType, osArch, version, configurePath string) error {
		downloadCalls++
		// 创建脚本
		createExecScript(t, configurePath, getExecName(osType), version)
		return nil
	}
	defer restoreGlobals()

	options := NewGoMigrateOptionsContext(ctx)
	if err := options.Run([]string{"--version"}); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if fetchCalls != 1 {
		t.Fatalf("expected fetch 1 got %d", fetchCalls)
	}
	if downloadCalls != 1 {
		t.Fatalf("expected download 1 got %d", downloadCalls)
	}
	if !options.download || !options.localInstalled || !options.isLatest {
		t.Fatalf("flags not set correctly: %+v", options)
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, "not installed") {
		t.Fatalf("stdout missing install notice: %s", outStr)
	}
	if !strings.Contains(outStr, "install successfully") {
		t.Fatalf("stdout missing success: %s", outStr)
	}
}

func TestGoMigrate_VersionEqual(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows 环境下脚本模拟复杂，跳过")
	}
	ctx, stdout, _ := newTestContext()
	tmpDir := t.TempDir()
	configPathBackup = tmpDir
	getConfigurePathFunc = func() string { return tmpDir }

	remoteVersion := "2.0.0"
	execName := getExecName(runtime.GOOS)
	createExecScript(t, tmpDir, execName, remoteVersion) // 预置本地文件

	fetchRemoteContentFunc = func(url string) (string, error) { return remoteVersion, nil }
	downloadCalled := 0
	downloadGoMigrateFunc = func(osType, osArch, version, configurePath string) error {
		downloadCalled++
		return nil
	}
	defer restoreGlobals()

	options := NewGoMigrateOptionsContext(ctx)
	if err := options.Run([]string{"--version"}); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if downloadCalled != 0 {
		t.Fatalf("should not download when version equal")
	}
	if !options.isLatest || !options.localInstalled || options.download {
		t.Fatalf("flags incorrect: %+v", options)
	}
	outStr := stdout.String()
	if strings.Contains(outStr, "upgrade") {
		t.Fatalf("unexpected upgrade message: %s", outStr)
	}
}

func TestGoMigrate_VersionUpgrade(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows 环境下脚本模拟复杂，跳过")
	}
	ctx, stdout, _ := newTestContext()
	tmpDir := t.TempDir()
	configPathBackup = tmpDir
	getConfigurePathFunc = func() string { return tmpDir }

	oldVersion := "1.0.0"
	newVersion := "1.1.0"
	execName := getExecName(runtime.GOOS)
	createExecScript(t, tmpDir, execName, oldVersion)

	fetchRemoteContentFunc = func(url string) (string, error) { return newVersion, nil }
	removedOld := false
	downloadCalled := 0
	downloadGoMigrateFunc = func(osType, osArch, version, configurePath string) error {
		downloadCalled++
		// 确认旧文件已经被删除
		if _, err := os.Stat(filepath.Join(tmpDir, execName)); err == nil {
			// 旧文件在 removeLocalFile 后还是存在，意味着升级逻辑没先删
			// 这里不直接失败，后面再检查
		} else {
			removedOld = true
		}
		createExecScript(t, configurePath, execName, version)
		return nil
	}
	defer restoreGlobals()

	options := NewGoMigrateOptionsContext(ctx)
	if err := options.Run([]string{"--version"}); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if downloadCalled != 1 {
		t.Fatalf("expected one download, got %d", downloadCalled)
	}
	if !removedOld {
		t.Fatalf("old binary not removed before download")
	}
	if !options.isLatest || !options.localInstalled || !options.download {
		t.Fatalf("flags incorrect: %+v", options)
	}
	out := stdout.String()
	if !strings.Contains(out, "local version "+oldVersion) {
		t.Fatalf("missing local version msg: %s", out)
	}
	if !strings.Contains(out, "upgrade to latest version "+newVersion) {
		t.Fatalf("missing upgrade success msg: %s", out)
	}
}

func TestFetchRemoteContent_Success(t *testing.T) {
	body := "3.3.3"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(body))
	}))
	defer ts.Close()

	got, err := fetchRemoteContent(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != body {
		t.Fatalf("expect %s got %s", body, got)
	}
}

func TestDownloadGoMigrate_Success(t *testing.T) {
	if runtime.GOOS == "windows" { // windows 权限位检查差异，但仍可测试
		// 继续执行，不跳过
	}
	osType := runtime.GOOS
	osArch := runtime.GOARCH
	version := "9.9.9"
	execName := getExecName(osType)
	binaryContent := []byte("BIN-test-content")

	// 搭建 mock server，路径需与 DownloadGoMigrate 生成的一致
	var base string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 期望路径
		expectedPath := "/go-migrate/" + version + "/" + osType + "-" + osArch + "/" + execName
		if r.URL.Path != expectedPath {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write(binaryContent)
	}))
	defer ts.Close()
	base = ts.URL

	old := cdnBaseUrlFunc
	cdnBaseUrlFunc = func() string { return base }
	defer func() { cdnBaseUrlFunc = old }()

	tmpDir := t.TempDir()
	if err := DownloadGoMigrate(osType, osArch, version, tmpDir); err != nil {
		t.Fatalf("download error: %v", err)
	}
	binPath := filepath.Join(tmpDir, execName)
	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read bin failed: %v", err)
	}
	if string(data) != string(binaryContent) {
		t.Fatalf("binary content mismatch")
	}
	if osType != "windows" {
		info, err := os.Stat(binPath)
		if err != nil {
			t.Fatalf("stat failed: %v", err)
		}
		if info.Mode()&0100 == 0 {
			t.Fatalf("expected executable bit set")
		}
	}
}
