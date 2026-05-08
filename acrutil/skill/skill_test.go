package skill

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func prepareConfig(t *testing.T, home string, language string) {
	cfgDir := filepath.Join(home, ".aliyun")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	configJSON := fmt.Sprintf(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou","language":"%s"}]}`, language)
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func prepareConfigWithMode(t *testing.T, home string, mode string, extraFields map[string]string) {
	cfgDir := filepath.Join(home, ".aliyun")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}

	fields := fmt.Sprintf(`"name":"default","mode":"%s","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou","language":"en"`, mode)
	for k, v := range extraFields {
		fields += fmt.Sprintf(`,"%s":"%s"`, k, v)
	}
	configJSON := fmt.Sprintf(`{"current":"default","profiles":[{%s}]}`, fields)
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func newOriginCtx() (*cli.Context, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, errOut)
	return ctx, out, errOut
}

func addConfigFlag(ctx *cli.Context, name string, value string) {
	f := &cli.Flag{Name: name, AssignedMode: cli.AssignedOnce, Category: "config"}
	f.SetAssigned(true)
	f.SetValue(value)
	ctx.Flags().Add(f)
}

func TestNewSkillContext(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewSkillContext(ctx)
	if c == nil {
		t.Fatalf("NewSkillContext returned nil")
	}
	if c.originCtx != ctx {
		t.Errorf("originCtx mismatch")
	}
}

func TestCheckOsTypeAndArch(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewSkillContext(ctx)

	tests := []struct {
		osType, osArch string
		wantSupport    bool
	}{
		{"linux", "amd64", true},
		{"darwin", "arm64", true},
		{"linux", "arm64", false},
		{"darwin", "amd64", false},
		{"windows", "amd64", false},
		{"freebsd", "amd64", false},
		{"linux", "ppc64le", false},
		{"linux", "386", false},
		{"linux", "arm", false},
		{"windows", "386", false},
	}

	oldGOOS := runtimeGOOSFunc
	oldGOARCH := runtimeGOARCHFunc
	defer func() {
		runtimeGOOSFunc = oldGOOS
		runtimeGOARCHFunc = oldGOARCH
	}()

	for _, tt := range tests {
		runtimeGOOSFunc = func() string { return tt.osType }
		runtimeGOARCHFunc = func() string { return tt.osArch }
		c.CheckOsTypeAndArch()
		if c.osSupport != tt.wantSupport {
			t.Errorf("os=%s arch=%s support expected %v, got %v", tt.osType, tt.osArch, tt.wantSupport, c.osSupport)
		}
	}
}

func TestInitBasicInfo(t *testing.T) {
	tmpDir := t.TempDir()
	old := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = old }()

	ctx, _, _ := newOriginCtx()
	c := NewSkillContext(ctx)
	c.InitBasicInfo()

	if c.configPath != tmpDir {
		t.Errorf("configPath expected %s, got %s", tmpDir, c.configPath)
	}
	if c.installed {
		t.Errorf("should not be installed initially")
	}

	execPath := filepath.Join(tmpDir, "acr-skill")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	if err := os.WriteFile(execPath, []byte("fake"), 0755); err != nil {
		t.Fatalf("write fake exec: %v", err)
	}

	c2 := NewSkillContext(ctx)
	c2.InitBasicInfo()
	if !c2.installed {
		t.Errorf("should be installed now")
	}
}

func TestRun_NotInstalled_FreshInstallAndExecute(t *testing.T) {
	origHOME := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", origHOME) })

	home := t.TempDir()
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	prepareConfig(t, home, "zh")

	origDownload := downloadBinaryFunc
	origExec := execCommandFunc
	installCount := 0
	downloadBinaryFunc = func(url, exe string) error {
		installCount++
		if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatalf("write fake exec: %v", err)
		}
		return nil
	}
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		// 跨平台的 mock 实现，不依赖 bash
		if runtime.GOOS == "windows" {
			return exec.Command("cmd", "/c", "exit 0")
		}
		return exec.Command("true")
	}
	t.Cleanup(func() {
		downloadBinaryFunc = origDownload
		execCommandFunc = origExec
	})

	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")

	c := NewSkillContext(ctx)

	if err := c.Run([]string{"acr-skill", "validate", "--region", "cn-hangzhou"}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if installCount != 1 {
		t.Fatalf("expected install once, got %d", installCount)
	}
}

func TestRun_Installed_SkipDownload(t *testing.T) {
	origHOME := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", origHOME) })
	home := t.TempDir()
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	prepareConfig(t, home, "en")

	execPath := filepath.Join(config.GetConfigPath(), "acr-skill")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write exec: %v", err)
	}

	origDownload := downloadBinaryFunc
	origExec := execCommandFunc
	installCount := 0
	downloadBinaryFunc = func(url, exe string) error { installCount++; return nil }
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return exec.Command("true") }
	t.Cleanup(func() {
		downloadBinaryFunc = origDownload
		execCommandFunc = origExec
	})

	ctx, _, _ := newOriginCtx()
	c := NewSkillContext(ctx)
	if err := c.Run([]string{"acr-skill", "list"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if installCount != 0 {
		t.Fatalf("expected no download when already installed, got %d", installCount)
	}
}

func TestGetDownloadURL(t *testing.T) {
	tests := []struct {
		name        string
		platform    string
		expectError bool
		expectedURL string
	}{
		{
			name:        "valid platform linux-amd64",
			platform:    "linux-amd64",
			expectError: false,
			expectedURL: "https://acr-public-asset.oss-cn-hangzhou.aliyuncs.com/acr-skill/acr-skill-linux-amd64",
		},
		{
			name:        "valid platform darwin-arm64",
			platform:    "darwin-arm64",
			expectError: false,
			expectedURL: "https://acr-public-asset.oss-cn-hangzhou.aliyuncs.com/acr-skill/acr-skill-darwin-arm64",
		},
		{
			name:        "unsupported platform linux-arm64",
			platform:    "linux-arm64",
			expectError: true,
		},
		{
			name:        "unsupported platform darwin-amd64",
			platform:    "darwin-amd64",
			expectError: true,
		},
		{
			name:        "unsupported platform windows-amd64",
			platform:    "windows-amd64",
			expectError: true,
		},
		{
			name:        "unsupported platform freebsd-amd64",
			platform:    "freebsd-amd64",
			expectError: true,
		},
		{
			name:        "invalid platform",
			platform:    "unknown-platform",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := getDownloadURL(tt.platform)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if url != tt.expectedURL {
					t.Errorf("Expected URL '%s', got '%s'", tt.expectedURL, url)
				}
			}
		})
	}
}

func TestDownloadBinary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake binary content"))
	}))
	defer server.Close()

	oldHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return http.Get(server.URL)
	}
	defer func() { httpGetFunc = oldHTTPGet }()

	// 临时替换 httpClient 为测试服务器
	oldClient := httpClient
	httpClient = server.Client()
	httpClient.Transport = server.Client().Transport
	defer func() { httpClient = oldClient }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "acr-skill")

	err := DownloadBinary(server.URL, execPath)
	if err != nil {
		t.Fatalf("DownloadBinary failed: %v", err)
	}

	if !fileExists(execPath) {
		t.Errorf("exec file not created")
	}

	content, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("read exec: %v", err)
	}
	if string(content) != "fake binary content" {
		t.Errorf("exec content mismatch: got %q", string(content))
	}
}

func TestDownloadBinary_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	oldClient := httpClient
	httpClient = server.Client()
	httpClient.Transport = server.Client().Transport
	defer func() { httpClient = oldClient }()

	tmp := t.TempDir()
	err := DownloadBinary(server.URL, filepath.Join(tmp, "acr-skill"))
	if err == nil || !strings.Contains(err.Error(), "status code") {
		t.Fatalf("expect status code error, got %v", err)
	}
}

func TestDownloadBinary_HttpError(t *testing.T) {
	// 使用一个无效的 URL 来触发网络错误
	tmp := t.TempDir()
	err := DownloadBinary("http://invalid-host-that-does-not-exist-12345.com", filepath.Join(tmp, "acr-skill"))
	if err == nil || !strings.Contains(err.Error(), "failed to download") {
		t.Fatalf("expected download error")
	}
}

func TestDownloadBinary_OverwriteExisting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("new binary"))
	}))
	defer server.Close()

	oldHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return http.Get(server.URL)
	}
	defer func() { httpGetFunc = oldHTTPGet }()

	// 临时替换 httpClient 为测试服务器
	oldClient := httpClient
	httpClient = server.Client()
	httpClient.Transport = server.Client().Transport
	defer func() { httpClient = oldClient }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "acr-skill")
	// 先创建一个旧文件
	if err := os.WriteFile(execPath, []byte("old binary"), 0755); err != nil {
		t.Fatalf("write old exec: %v", err)
	}

	err := DownloadBinary(server.URL, execPath)
	if err != nil {
		t.Fatalf("DownloadBinary failed: %v", err)
	}

	content, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("read exec: %v", err)
	}
	if string(content) != "new binary" {
		t.Errorf("exec content should be updated, got %q", string(content))
	}
}

func TestCopySystemEnv_AK(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)
	prepareConfig(t, home, "en")

	ctx, _, _ := newOriginCtx()
	c := NewSkillContext(ctx)
	c.InitBasicInfo()

	// CopySystemEnv 不再包含凭证，只返回系统环境变量
	envMap, err := c.CopySystemEnv()
	if err != nil {
		t.Fatalf("CopySystemEnv err: %v", err)
	}
	// 验证不包含 REGISTRY_USERNAME/PASSWORD
	if _, exists := envMap["REGISTRY_USERNAME"]; exists {
		t.Fatalf("REGISTRY_USERNAME should not be in envMap")
	}
	if _, exists := envMap["REGISTRY_PASSWORD"]; exists {
		t.Fatalf("REGISTRY_PASSWORD should not be in envMap")
	}
}

func TestCopySystemEnv_StsToken(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)
	prepareConfigWithMode(t, home, "StsToken", map[string]string{
		"sts_token": "sts123",
	})

	ctx, _, _ := newOriginCtx()
	c := NewSkillContext(ctx)
	c.InitBasicInfo()

	// CopySystemEnv 不再包含凭证
	envMap, err := c.CopySystemEnv()
	if err != nil {
		t.Fatalf("CopySystemEnv err: %v", err)
	}
	if _, exists := envMap["REGISTRY_USERNAME"]; exists {
		t.Fatalf("REGISTRY_USERNAME should not be in envMap")
	}
	if _, exists := envMap["REGISTRY_PASSWORD"]; exists {
		t.Fatalf("REGISTRY_PASSWORD should not be in envMap")
	}
}

func TestCopySystemEnv_RamRoleArn(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)
	prepareConfigWithMode(t, home, "RamRoleArn", map[string]string{
		"ram_role_arn":     "arn:acs:ram::123:role/test",
		"ram_session_name": "session123",
	})

	ctx, _, _ := newOriginCtx()
	c := NewSkillContext(ctx)
	c.InitBasicInfo()

	// CopySystemEnv 不再包含凭证
	envMap, err := c.CopySystemEnv()
	if err != nil {
		t.Fatalf("CopySystemEnv err: %v", err)
	}
	if _, exists := envMap["REGISTRY_USERNAME"]; exists {
		t.Fatalf("REGISTRY_USERNAME should not be in envMap")
	}
	if _, exists := envMap["REGISTRY_PASSWORD"]; exists {
		t.Fatalf("REGISTRY_PASSWORD should not be in envMap")
	}
}

func TestCopySystemEnv_ConfigLoadError(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)

	ctx, _, _ := newOriginCtx()
	c := NewSkillContext(ctx)
	c.InitBasicInfo()

	// CopySystemEnv 不再加载配置，不会报错
	envMap, err := c.CopySystemEnv()
	if err != nil {
		t.Fatalf("CopySystemEnv should not error: %v", err)
	}
	if envMap == nil {
		t.Fatalf("envMap should not be nil")
	}
}

func TestRemoveFlagsForMainCli(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")
	addConfigFlag(ctx, "profile", "test")

	c := NewSkillContext(ctx)
	args := []string{"validate", "--region", "cn-hangzhou", "--profile", "test", "-d", "./my-skill"}
	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("RemoveFlagsForMainCli failed: %v", err)
	}

	for _, arg := range newArgs {
		if arg == "--region" || arg == "--profile" {
			t.Errorf("config flag should be removed: %s", arg)
		}
	}

	hasDir := false
	for _, arg := range newArgs {
		if arg == "-d" {
			hasDir = true
		}
	}
	if !hasDir {
		t.Errorf("non-config flag -d should remain")
	}
}

func TestFileExists(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	if fileExists(tmpFile) {
		t.Errorf("file should not exist")
	}

	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if !fileExists(tmpFile) {
		t.Errorf("file should exist")
	}
}

func TestInstallUrlAndInvocation(t *testing.T) {
	c := &SkillContext{
		downloadPathSuffix: "linux-amd64",
	}
	called := false
	origDownload := downloadBinaryFunc
	downloadBinaryFunc = func(url, exe string) error {
		called = true
		if !strings.Contains(url, "acr-skill-linux-amd64") {
			t.Fatalf("url should contain platform suffix, got %s", url)
		}
		return nil
	}
	defer func() { downloadBinaryFunc = origDownload }()
	if err := c.Install(); err != nil {
		t.Fatalf("Install err: %v", err)
	}
	if !called {
		t.Fatalf("download not called")
	}
	if !c.installed {
		t.Fatalf("installed should be true after Install()")
	}
}

func TestExecuteAcrSkill(t *testing.T) {
	origExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "exit 0")
	}
	defer func() { execCommandFunc = origExec }()

	ctx, _, _ := newOriginCtx()
	c := NewSkillContext(ctx)
	c.execFilePath = "/any/path/acr-skill"

	envMap := map[string]string{
		"REGISTRY_USERNAME": "test_ak",
		"REGISTRY_PASSWORD": "test_sk",
	}
	err := c.ExecuteAcrSkill([]string{"validate", "-d", "./my-skill"}, envMap)
	if err != nil {
		t.Fatalf("ExecuteAcrSkill failed: %v", err)
	}
}

func TestExecuteAcrSkill_Failure(t *testing.T) {
	origExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		// 跨平台的 mock 实现，模拟失败
		if runtime.GOOS == "windows" {
			return exec.Command("cmd", "/c", "exit 1")
		}
		return exec.Command("false")
	}
	defer func() { execCommandFunc = origExec }()

	ctx, _, _ := newOriginCtx()
	c := NewSkillContext(ctx)
	c.execFilePath = "/any/path/acr-skill"

	envMap := map[string]string{}
	err := c.ExecuteAcrSkill([]string{"validate"}, envMap)
	if err == nil {
		t.Fatalf("expected execution error")
	}
	if !strings.Contains(err.Error(), "failed to execute") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
