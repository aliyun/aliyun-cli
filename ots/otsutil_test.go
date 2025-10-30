package ots

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

// helper to create a fake executable script
func writeExecutable(t *testing.T, path string, content string) {
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write exec failed: %v", err)
	}
	if runtime.GOOS == "windows" {
		// skip
	}
}

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

// prepareConfigWithMode creates a config file with specific authentication mode
func prepareConfigWithMode(t *testing.T, home string, mode string, extraFields map[string]string) {
	cfgDir := filepath.Join(home, ".aliyun")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}

	profile := map[string]interface{}{
		"name":              "default",
		"mode":              mode,
		"access_key_id":     "ak",
		"access_key_secret": "sk",
		"region_id":         "cn-hangzhou",
		"language":          "en",
	}

	// Add extra fields with proper type conversion for numeric fields
	for k, v := range extraFields {
		switch k {
		case "retry_timeout", "connect_timeout", "retry_count":
			// Convert string to int for numeric fields
			if intVal, err := strconv.Atoi(v); err == nil {
				profile[k] = intVal
			} else {
				profile[k] = v
			}
		default:
			profile[k] = v
		}
	}

	config := map[string]interface{}{
		"current":  "default",
		"profiles": []interface{}{profile},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), configJSON, 0644); err != nil {
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

func TestNewContext(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if c == nil {
		t.Fatalf("NewContext returned nil")
	}
	if c.originCtx != ctx {
		t.Errorf("originCtx mismatch")
	}
}

func TestCheckOsTypeAndArch(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)

	tests := []struct {
		osType, osArch string
		wantSupport    bool
	}{
		{"linux", "amd64", true},
		{"linux", "arm64", true},
		{"darwin", "amd64", true},
		{"darwin", "arm64", true},
		{"windows", "amd64", true},
		{"freebsd", "amd64", false},
		{"linux", "ppc64le", false},
		{"linux", "386", false},   // 官方不支持
		{"linux", "arm", false},   // 官方不支持
		{"windows", "386", false}, // 官方不支持
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
	c := NewContext(ctx)
	c.InitBasicInfo()

	if c.configPath != tmpDir {
		t.Errorf("configPath expected %s, got %s", tmpDir, c.configPath)
	}
	if c.installed {
		t.Errorf("should not be installed initially")
	}

	// create exec file
	execPath := filepath.Join(tmpDir, "ts")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	if err := os.WriteFile(execPath, []byte("fake"), 0755); err != nil {
		t.Fatalf("write fake exec: %v", err)
	}

	c2 := NewContext(ctx)
	c2.InitBasicInfo()
	if !c2.installed {
		t.Errorf("should be installed now")
	}
}

func TestDownloadAndUnzip(t *testing.T) {
	// create a zip file in memory
	zipBuf := &bytes.Buffer{}
	zw := zip.NewWriter(zipBuf)
	execName := "ts"
	if runtime.GOOS == "windows" {
		execName = "ts.exe"
	}
	// Tablestore CLI 的 ZIP 包内文件直接在根目录，没有中间目录
	innerPath := execName
	fw, err := zw.Create(innerPath)
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	fw.Write([]byte("fake binary content"))
	zw.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(zipBuf.Bytes())
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destFile := filepath.Join(tmpDir, "test.zip")
	execPath := filepath.Join(tmpDir, "ts")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}

	// 传递空字符串表示没有中间目录
	err = DownloadAndUnzip(server.URL, destFile, execPath, "")
	if err != nil {
		t.Fatalf("DownloadAndUnzip failed: %v", err)
	}

	if !fileExists(execPath) {
		t.Errorf("exec file not created")
	}

	content, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("read exec: %v", err)
	}
	if string(content) != "fake binary content" {
		t.Errorf("exec content mismatch")
	}
}

func TestEncodeMapBase64(t *testing.T) {
	m := map[string]any{
		"mode":              "AK",
		"access_key_id":     "test_ak",
		"access_key_secret": "test_sk",
	}
	encoded, err := EncodeMapBase64(m)
	if err != nil {
		t.Fatalf("EncodeMapBase64 failed: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(decoded, &result); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if result["mode"] != "AK" {
		t.Errorf("mode mismatch")
	}
	if result["access_key_id"] != "test_ak" {
		t.Errorf("access_key_id mismatch")
	}
}

func TestPrepareEnv(t *testing.T) {
	tmpHome := t.TempDir()
	prepareConfig(t, tmpHome, "en")

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	oldConfigPath := getConfigurePathFunc
	getConfigurePathFunc = func() string { return filepath.Join(tmpHome, ".aliyun") }
	defer func() { getConfigurePathFunc = oldConfigPath }()

	profile, err := config.LoadOrCreateDefaultProfile()
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}

	// 切换工作目录到一个可控的临时目录，便于校验写入位置
	wd := t.TempDir()
	oldWd, _ := os.Getwd()
	_ = os.Chdir(wd)
	defer func() { _ = os.Chdir(oldWd) }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.configPath = filepath.Join(tmpHome, ".aliyun")
	err = c.PrepareEnv()
	if err != nil {
		t.Fatalf("PrepareEnv failed: %v", err)
	}

	// 检查配置文件是否创建在当前工作目录
	configPath := filepath.Join(wd, ".tablestore_config")
	if !fileExists(configPath) {
		t.Errorf("config file %s not created", configPath)
	}

	// 读取并验证配置文件内容
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var tsConfig map[string]interface{}
	if err := json.Unmarshal(configData, &tsConfig); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if tsConfig["AccessKeyId"] != profile.AccessKeyId {
		t.Errorf("AccessKeyId mismatch in config file")
	}
	if tsConfig["AccessKeySecret"] != profile.AccessKeySecret {
		t.Errorf("AccessKeySecret mismatch in config file")
	}
}

func TestRemoveFlagsForMainCli(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")
	addConfigFlag(ctx, "profile", "test")

	c := NewContext(ctx)
	args := []string{"list", "--region", "cn-hangzhou", "--profile", "test", "--key", "value"}
	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("RemoveFlagsForMainCli failed: %v", err)
	}

	// config category flags should be removed
	for _, arg := range newArgs {
		if arg == "--region" || arg == "--profile" {
			t.Errorf("config flag should be removed: %s", arg)
		}
	}

	// non-config flags should remain
	hasKey := false
	for _, arg := range newArgs {
		if arg == "--key" {
			hasKey = true
		}
	}
	if !hasKey {
		t.Errorf("non-config flag --key should remain")
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

func TestUnzip(t *testing.T) {
	tmpDir := t.TempDir()
	zipFile := filepath.Join(tmpDir, "test.zip")
	destDir := filepath.Join(tmpDir, "unzip")

	// create zip
	f, err := os.Create(zipFile)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	fw, err := zw.Create("test/file.txt")
	if err != nil {
		t.Fatalf("zip create file: %v", err)
	}
	fw.Write([]byte("test content"))
	zw.Close()
	f.Close()

	err = unzip(zipFile, destDir)
	if err != nil {
		t.Fatalf("unzip failed: %v", err)
	}

	extractedFile := filepath.Join(destDir, "test/file.txt")
	if !fileExists(extractedFile) {
		t.Errorf("extracted file not found")
	}

	content, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("content mismatch")
	}
}

func TestInfo(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	mockCtx := cli.NewCommandContext(&stdout, &stderr)

	c := &Context{
		originCtx: mockCtx,
	}

	// Test with format string
	format := "Hello %s\n"
	c.info(format, "World")
	if stdout.String() != "Hello World\n" {
		t.Errorf("Expected 'Hello World\\n', got '%s'", stdout.String())
	}

	// Test with println
	stdout.Reset()
	c.info("Test")
	if stdout.String() != "Test\n" {
		t.Errorf("Expected 'Test\\n', got '%s'", stdout.String())
	}
}

func TestErrorf(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	mockCtx := cli.NewCommandContext(&stdout, &stderr)

	c := &Context{
		originCtx: mockCtx,
	}

	c.errorf("Error: %s\n", "test error")
	if stderr.String() != "Error: test error\n" {
		t.Errorf("Expected 'Error: test error\\n', got '%s'", stderr.String())
	}
}

func TestNeedCheckVersion(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		installed      bool
		cacheExists    bool
		cacheContent   string
		currentTime    int64
		expectedResult bool
	}{
		{
			name:           "not installed",
			installed:      false,
			expectedResult: false,
		},
		{
			name:           "cache file not exist",
			installed:      true,
			cacheExists:    false,
			expectedResult: true,
		},
		{
			name:           "cache file invalid",
			installed:      true,
			cacheExists:    true,
			cacheContent:   "invalid",
			expectedResult: true,
		},
		{
			name:           "need check - expired",
			installed:      true,
			cacheExists:    true,
			cacheContent:   "1000000",
			currentTime:    2000000,
			expectedResult: true,
		},
		{
			name:           "no need check - fresh",
			installed:      true,
			cacheExists:    true,
			cacheContent:   fmt.Sprintf("%d", time.Now().Unix()),
			currentTime:    time.Now().Unix(),
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheFile := filepath.Join(tmpDir, ".tsutil_version_check_"+tt.name)

			c := &Context{
				installed:                 tt.installed,
				checkVersionCacheFilePath: cacheFile,
			}

			if tt.cacheExists {
				err := os.WriteFile(cacheFile, []byte(tt.cacheContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write cache file: %v", err)
				}
			}

			if tt.currentTime > 0 {
				oldTimeNowFunc := timeNowFunc
				timeNowFunc = func() time.Time {
					return time.Unix(tt.currentTime, 0)
				}
				defer func() { timeNowFunc = oldTimeNowFunc }()
			}

			result := c.NeedCheckVersion()
			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestGetLocalVersion(t *testing.T) {
	tests := []struct {
		name        string
		installed   bool
		expectError bool
	}{
		{
			name:        "not installed",
			installed:   false,
			expectError: true,
		},
		{
			name:        "installed",
			installed:   true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Context{
				installed: tt.installed,
			}

			err := c.GetLocalVersion()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if c.versionLocal != currentVersion {
					t.Errorf("Expected version %s, got %s", currentVersion, c.versionLocal)
				}
			}
		})
	}
}

func TestUpdateCheckCacheTime(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".tsutil_version_check")

	fixedTime := time.Unix(1234567890, 0)
	oldTimeNowFunc := timeNowFunc
	timeNowFunc = func() time.Time {
		return fixedTime
	}
	defer func() { timeNowFunc = oldTimeNowFunc }()

	c := &Context{
		checkVersionCacheFilePath: cacheFile,
	}

	err := c.UpdateCheckCacheTime()
	if err != nil {
		t.Fatalf("UpdateCheckCacheTime failed: %v", err)
	}

	// Verify cache file content
	content, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	expectedContent := fmt.Sprintf("%d", fixedTime.Unix())
	if string(content) != expectedContent {
		t.Errorf("Expected cache content '%s', got '%s'", expectedContent, string(content))
	}
}
