package kmscli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	os.Exit(0)
}

func setupTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	origHomeDrive := os.Getenv("HOMEDRIVE")
	origHomePath := os.Getenv("HOMEPATH")
	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
		os.Setenv("HOMEDRIVE", origHomeDrive)
		os.Setenv("HOMEPATH", origHomePath)
	})
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)
	os.Setenv("HOMEDRIVE", "")
	os.Setenv("HOMEPATH", "")
	return home
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
	for k, v := range extraFields {
		switch k {
		case "retry_timeout", "connect_timeout", "retry_count":
			if intVal, err := strconv.Atoi(v); err == nil {
				profile[k] = intVal
			} else {
				profile[k] = v
			}
		default:
			profile[k] = v
		}
	}
	configMap := map[string]interface{}{
		"current":  "default",
		"profiles": []interface{}{profile},
	}
	configJSON, err := json.Marshal(configMap)
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

func writeExecutable(t *testing.T, path string, content string) {
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write exec failed: %v", err)
	}
}

func mockExecCommand() *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

// --- Run integration tests ---

func TestRun_NotInstalled_FreshInstallAndExecute(t *testing.T) {
	home := setupTestHome(t)
	prepareConfig(t, home, "zh")

	origDownload := downloadBinaryFunc
	origExec := execCommandFunc
	origTimeNow := timeNowFunc
	origGetVersion := getLatestKmscliVersionFunc
	downloadCount := 0
	downloadBinaryFunc = func(url, dest string) error {
		downloadCount++
		writeExecutable(t, dest, "dummy")
		return nil
	}
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return mockExecCommand()
	}
	getLatestKmscliVersionFunc = func() (string, error) { return "v0.1.0", nil }
	fixedNow := time.Unix(1700000000, 0)
	timeNowFunc = func() time.Time { return fixedNow }
	t.Cleanup(func() {
		downloadBinaryFunc = origDownload
		execCommandFunc = origExec
		timeNowFunc = origTimeNow
		getLatestKmscliVersionFunc = origGetVersion
	})

	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")
	c := NewContext(ctx)

	if err := c.Run([]string{"kmscli", "secret", "getsecret", "test"}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if downloadCount != 1 {
		t.Fatalf("expected download once, got %d", downloadCount)
	}
	data, err := os.ReadFile(filepath.Join(config.GetConfigPath(), ".kmscli_version_check"))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	if strings.TrimSpace(string(data)) != fmt.Sprintf("%d", fixedNow.Unix()) {
		t.Fatalf("cache timestamp mismatch: %s", string(data))
	}
}

func TestRun_Installed_NoVersionCheckWithinTTL(t *testing.T) {
	home := setupTestHome(t)
	prepareConfig(t, home, "en")

	execPath := filepath.Join(config.GetConfigPath(), "kmscli")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "dummy")

	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".kmscli_version_check"), []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)

	origDownload := downloadBinaryFunc
	origExec := execCommandFunc
	downloadCount := 0
	downloadBinaryFunc = func(url, dest string) error { downloadCount++; return nil }
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return mockExecCommand() }
	t.Cleanup(func() {
		downloadBinaryFunc = origDownload
		execCommandFunc = origExec
	})

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"kmscli", "secret", "getsecret", "test"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if downloadCount != 0 {
		t.Fatalf("expected no download, got %d", downloadCount)
	}
}

func TestRun_Installed_UpdateWhenExpired(t *testing.T) {
	home := setupTestHome(t)
	prepareConfig(t, home, "en")

	execPath := filepath.Join(config.GetConfigPath(), "kmscli")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "dummy")

	old := time.Now().Unix() - int64(VersionCheckTTL) - 10
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".kmscli_version_check"), []byte(fmt.Sprintf("%d", old)), 0644)
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".kmscli_version"), []byte("v0.0.0"), 0644)

	origDownload := downloadBinaryFunc
	origExec := execCommandFunc
	origGetVersion := getLatestKmscliVersionFunc
	downloadCount := 0
	downloadBinaryFunc = func(url, dest string) error {
		downloadCount++
		writeExecutable(t, dest, "dummy")
		return nil
	}
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return mockExecCommand() }
	getLatestKmscliVersionFunc = func() (string, error) { return "v0.1.0", nil }
	t.Cleanup(func() {
		downloadBinaryFunc = origDownload
		execCommandFunc = origExec
		getLatestKmscliVersionFunc = origGetVersion
	})

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"kmscli", "secret", "getsecret", "test"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if downloadCount == 0 {
		t.Fatalf("expected download triggered")
	}
}

// --- NeedCheckVersion ---

func TestNeedCheckVersionVariants(t *testing.T) {
	home := setupTestHome(t)
	prepareConfig(t, home, "zh")
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if c.NeedCheckVersion() {
		t.Fatalf("not installed should return false")
	}

	execPath := filepath.Join(config.GetConfigPath(), "kmscli")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "dummy")
	c.InitBasicInfo()
	if !c.NeedCheckVersion() {
		t.Fatalf("installed no cache => true")
	}

	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".kmscli_version_check"), []byte("abc"), 0644)
	c.InitBasicInfo()
	if !c.NeedCheckVersion() {
		t.Fatalf("invalid content => true")
	}

	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".kmscli_version_check"), []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)
	c.InitBasicInfo()
	if c.NeedCheckVersion() {
		t.Fatalf("fresh cache => false")
	}

	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".kmscli_version_check"), []byte(fmt.Sprintf("%d", time.Now().Unix()-int64(VersionCheckTTL)-5)), 0644)
	c.InitBasicInfo()
	if !c.NeedCheckVersion() {
		t.Fatalf("expired => true")
	}
}

// --- RemoveFlagsForMainCli ---

func TestRemoveFlagsForMainCli(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")
	c := NewContext(ctx)
	args, err := c.RemoveFlagsForMainCli([]string{"kmscli", "secret", "getsecret", "--region", "cn-hangzhou"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--region") {
		t.Fatalf("region flag should be removed: %s", joined)
	}
	if !strings.Contains(joined, "secret") {
		t.Fatalf("secret should remain: %s", joined)
	}
}

func TestRemoveFlagsForMainCli_NilFlags(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	args, err := c.RemoveFlagsForMainCli([]string{"kmscli", "ls"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(args) != 2 || args[0] != "kmscli" || args[1] != "ls" {
		t.Fatalf("unexpected args: %v", args)
	}
}

// --- GetLocalVersion ---

func TestGetLocalVersion_NotInstalled(t *testing.T) {
	setupTestHome(t)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if c.installed {
		c.installed = false
	}
	err := c.GetLocalVersion()
	if err == nil {
		t.Fatalf("expected error when not installed")
	}
	if c.versionLocal != "" {
		t.Fatalf("versionLocal should be empty")
	}
}

func TestGetLocalVersion_Success(t *testing.T) {
	setupTestHome(t)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.installed = true
	_ = os.WriteFile(c.versionFilePath, []byte("v0.0.1\n"), 0644)
	if err := c.GetLocalVersion(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if c.versionLocal != "v0.0.1" {
		t.Fatalf("versionLocal mismatch %s", c.versionLocal)
	}
}

func TestGetLocalVersion_EmptyFile(t *testing.T) {
	setupTestHome(t)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.installed = true
	_ = os.WriteFile(c.versionFilePath, []byte("\n"), 0644)
	if err := c.GetLocalVersion(); err == nil {
		t.Fatalf("expected error for empty version file")
	}
}

// --- Install ---

func TestInstallUrlAndInvocation(t *testing.T) {
	setupTestHome(t)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.versionRemote = "v0.0.1"
	c.binaryName = "kmscli-linux-amd64"

	called := false
	origDownload := downloadBinaryFunc
	downloadBinaryFunc = func(url, dest string) error {
		called = true
		expectedURL := ossDownloadBase + "kmscli-linux-amd64"
		if url != expectedURL {
			t.Fatalf("url mismatch: %s vs %s", url, expectedURL)
		}
		writeExecutable(t, dest, "dummy")
		return nil
	}
	defer func() { downloadBinaryFunc = origDownload }()

	if err := c.Install(); err != nil {
		t.Fatalf("Install err: %v", err)
	}
	if !called {
		t.Fatalf("download not called")
	}
	if c.versionLocal != "v0.0.1" {
		t.Fatalf("versionLocal should be v0.0.1, got %s", c.versionLocal)
	}
	if !c.installed {
		t.Fatalf("installed should be true")
	}
}

// --- UpdateCheckCacheTime ---

func TestUpdateCheckCacheTime(t *testing.T) {
	setupTestHome(t)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	origNow := timeNowFunc
	fixed := time.Unix(1800000000, 0)
	timeNowFunc = func() time.Time { return fixed }
	defer func() { timeNowFunc = origNow }()

	if err := c.UpdateCheckCacheTime(); err != nil {
		t.Fatalf("err: %v", err)
	}
	b, _ := os.ReadFile(c.checkVersionCacheFilePath)
	if strings.TrimSpace(string(b)) != fmt.Sprintf("%d", fixed.Unix()) {
		t.Fatalf("cache mismatch %s", string(b))
	}
}

// --- FileExists ---

func TestFileExists(t *testing.T) {
	f := filepath.Join(t.TempDir(), "x.txt")
	if fileExists(f) {
		t.Fatalf("should false before create")
	}
	_ = os.WriteFile(f, []byte("1"), 0644)
	if !fileExists(f) {
		t.Fatalf("should true after create")
	}
}

// --- PrepareEnv ---

func TestPrepareEnv(t *testing.T) {
	home := setupTestHome(t)
	prepareConfig(t, home, "en")
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv err: %v", err)
	}
	if c.envMap["KMSCLI_NAME"] != "aliyun kmscli" {
		t.Fatalf("KMSCLI_NAME missing")
	}
	if c.envMap["REGION_ID"] != "cn-hangzhou" {
		t.Fatalf("REGION_ID mismatch: %v", c.envMap["REGION_ID"])
	}
	if c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"] != "ak" {
		t.Fatalf("ak mismatch: %v", c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"])
	}
	if c.envMap["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] != "sk" {
		t.Fatalf("sk mismatch: %v", c.envMap["ALIBABA_CLOUD_ACCESS_KEY_SECRET"])
	}
}

func TestPrepareEnv_StsToken(t *testing.T) {
	home := setupTestHome(t)
	prepareConfigWithMode(t, home, "StsToken", map[string]string{
		"sts_token": "sts123",
	})
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv err: %v", err)
	}
	if c.envMap["ALIBABA_CLOUD_SECURITY_TOKEN"] != "sts123" {
		t.Fatalf("security token mismatch: %v", c.envMap["ALIBABA_CLOUD_SECURITY_TOKEN"])
	}
	if c.envMap["KMSCLI_NAME"] != "aliyun kmscli" {
		t.Fatalf("KMSCLI_COMPAT_MODE missing")
	}
}

func TestPrepareEnv_ConfigLoadError(t *testing.T) {
	setupTestHome(t)
	// 不创建配置文件
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	err := c.PrepareEnv()
	if err == nil {
		t.Fatalf("expected config load error")
	}
	if !strings.Contains(err.Error(), "config failed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// --- CheckOsTypeAndArch ---

func TestCheckOsTypeAndArchVariants(t *testing.T) {
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	defer func() { runtimeGOOSFunc = origGOOS; runtimeGOARCHFunc = origGOARCH }()

	tests := []struct {
		os, arch   string
		support    bool
		binaryName string
	}{
		{"linux", "amd64", true, "kmscli-linux-amd64"},
		{"linux", "arm64", true, "kmscli-linux-arm64"},
		{"darwin", "amd64", true, "kmscli-darwin-amd64"},
		{"darwin", "arm64", true, "kmscli-darwin-arm64"},
		{"windows", "amd64", true, "kmscli-windows-amd64.exe"},
		{"linux", "s390x", false, ""},
		{"unknown", "amd64", false, ""},
	}
	for _, tc := range tests {
		runtimeGOOSFunc = func(val string) func() string { return func() string { return val } }(tc.os)
		runtimeGOARCHFunc = func(val string) func() string { return func() string { return val } }(tc.arch)
		ctx, _, _ := newOriginCtx()
		c := NewContext(ctx)
		c.CheckOsTypeAndArch()
		if c.osSupport != tc.support {
			t.Fatalf("expect support=%v for %s/%s got %v", tc.support, tc.os, tc.arch, c.osSupport)
		}
		if c.binaryName != tc.binaryName {
			t.Fatalf("binaryName mismatch for %s/%s: got %s want %s", tc.os, tc.arch, c.binaryName, tc.binaryName)
		}
	}
}

// --- downloadBinary ---

func TestDownloadBinary_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("binary content"))
	}))
	defer srv.Close()

	tmpFile := filepath.Join(t.TempDir(), "test.bin")
	err := downloadBinary(srv.URL+"/binary", tmpFile)
	if err != nil {
		t.Fatalf("downloadBinary err: %v", err)
	}
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read file err: %v", err)
	}
	if string(data) != "binary content" {
		t.Fatalf("content mismatch: %s", string(data))
	}
}

func TestDownloadBinary_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tmpFile := filepath.Join(t.TempDir(), "test.bin")
	err := downloadBinary(srv.URL+"/binary", tmpFile)
	if err == nil || !strings.Contains(err.Error(), "status code") {
		t.Fatalf("expected status code error, got %v", err)
	}
}

func TestDownloadBinary_HttpError(t *testing.T) {
	err := downloadBinary("http://invalid.invalid.invalid", filepath.Join(t.TempDir(), "test.bin"))
	if err == nil {
		t.Fatalf("expected error")
	}
}

// --- GetLatestKmscliVersion ---

func TestGetLatestKmscliVersion_Success(t *testing.T) {
	origHTTPDo := httpDoFunc
	defer func() { httpDoFunc = origHTTPDo }()
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("User-Agent") == "" {
			t.Fatalf("User-Agent should be set")
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("  v1.2.3\n")),
		}, nil
	}

	ver, err := GetLatestKmscliVersion()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ver != "v1.2.3" {
		t.Errorf("version mismatch: got %s, want v1.2.3", ver)
	}
}

func TestGetLatestKmscliVersion_NetworkError(t *testing.T) {
	origHTTPDo := httpDoFunc
	defer func() { httpDoFunc = origHTTPDo }()
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network down")
	}

	_, err := GetLatestKmscliVersion()
	if err == nil || !strings.Contains(err.Error(), "network down") {
		t.Fatalf("expected network error, got %v", err)
	}
}

func TestGetLatestKmscliVersion_Non200(t *testing.T) {
	origHTTPDo := httpDoFunc
	defer func() { httpDoFunc = origHTTPDo }()
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found"))}, nil
	}

	_, err := GetLatestKmscliVersion()
	if err == nil || !strings.Contains(err.Error(), "status code 404") {
		t.Fatalf("expected 404 error, got %v", err)
	}
}

func TestGetLatestKmscliVersion_Empty(t *testing.T) {
	origHTTPDo := httpDoFunc
	defer func() { httpDoFunc = origHTTPDo }()
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("  \n")),
		}, nil
	}

	_, err := GetLatestKmscliVersion()
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty error, got %v", err)
	}
}
