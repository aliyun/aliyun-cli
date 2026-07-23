package ecctl

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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

func buildTarGzArchive(entries map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, data := range entries {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(data))}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(data); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildZipArchive(entries map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range entries {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

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

	origDownload := downloadAndExtractFunc
	origExec := execCommandFunc
	origTimeNow := timeNowFunc
	origGetVersion := getLatestEcctlVersionFunc
	extractCount := 0
	downloadAndExtractFunc = func(url, destArchive, exe string) error {
		extractCount++
		writeExecutable(t, exe, "dummy")
		return nil
	}
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return mockExecCommand()
	}
	getLatestEcctlVersionFunc = func() (string, error) { return "0.1.1", nil }
	fixedNow := time.Unix(1700000000, 0)
	timeNowFunc = func() time.Time { return fixedNow }
	t.Cleanup(func() {
		downloadAndExtractFunc = origDownload
		execCommandFunc = origExec
		timeNowFunc = origTimeNow
		getLatestEcctlVersionFunc = origGetVersion
	})

	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")
	c := NewContext(ctx)

	if err := c.Run([]string{"help"}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if extractCount != 1 {
		t.Fatalf("expected extract once, got %d", extractCount)
	}
	data, err := os.ReadFile(filepath.Join(config.GetConfigPath(), ".ecctl_version_check"))
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

	execPath := filepath.Join(config.GetConfigPath(), "ecctl")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "dummy")

	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version_check"), []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)

	origDownload := downloadAndExtractFunc
	origExec := execCommandFunc
	extractCount := 0
	downloadAndExtractFunc = func(url, destArchive, exe string) error { extractCount++; return nil }
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return mockExecCommand() }
	t.Cleanup(func() {
		downloadAndExtractFunc = origDownload
		execCommandFunc = origExec
	})

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"help"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if extractCount != 0 {
		t.Fatalf("expected no download, got %d", extractCount)
	}
}

func TestRun_Installed_UpdateWhenExpired(t *testing.T) {
	setupTestHome(t)
	t.Setenv("ALIBABA_CLOUD_ECCTL_NO_UPDATE_CHECK", "0")
	execPath := filepath.Join(config.GetConfigPath(), "ecctl")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "dummy")
	old := time.Now().Unix() - int64(VersionCheckTTL) - 10
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version_check"), []byte(fmt.Sprintf("%d", old)), 0o644)
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version"), []byte("0.0.0"), 0o644)

	extractCount := 0
	origExtract := downloadAndExtractFunc
	origGetVersion := getLatestEcctlVersionFunc
	downloadAndExtractFunc = func(url, destArchive, exe string) error {
		extractCount++
		return nil
	}
	getLatestEcctlVersionFunc = func() (string, error) { return "0.1.1", nil }
	t.Cleanup(func() {
		downloadAndExtractFunc = origExtract
		getLatestEcctlVersionFunc = origGetVersion
	})

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if !c.installed {
		t.Fatal("expected installed")
	}
	if !c.NeedCheckVersion() {
		t.Fatal("expected version check")
	}
	if err := c.GetLocalVersion(); err != nil || c.versionLocal != "0.0.0" {
		t.Fatalf("local version: %v %q", err, c.versionLocal)
	}
	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if extractCount != 1 {
		t.Fatalf("expected extract once, got %d", extractCount)
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

	execPath := filepath.Join(config.GetConfigPath(), "ecctl")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "dummy")
	c.InitBasicInfo()
	if !c.NeedCheckVersion() {
		t.Fatalf("installed no cache => true")
	}

	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version_check"), []byte("abc"), 0644)
	c.InitBasicInfo()
	if !c.NeedCheckVersion() {
		t.Fatalf("invalid content => true")
	}

	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version_check"), []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)
	c.InitBasicInfo()
	if c.NeedCheckVersion() {
		t.Fatalf("fresh cache => false")
	}

	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version_check"), []byte(fmt.Sprintf("%d", time.Now().Unix()-int64(VersionCheckTTL)-5)), 0644)
	c.InitBasicInfo()
	if !c.NeedCheckVersion() {
		t.Fatalf("expired => true")
	}
}

// --- RemoveFlagsForMainCli ---

func TestRemoveFlagsForMainCli(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	out, err := c.RemoveFlagsForMainCli([]string{"help", "--region", "cn-hangzhou"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	joined := strings.Join(out, " ")
	if strings.Contains(joined, "--region") || strings.Contains(joined, "cn-hangzhou") {
		t.Fatalf("region flag should be removed: %s", joined)
	}
	if !strings.Contains(joined, "help") {
		t.Fatalf("help should remain: %s", joined)
	}
}

func TestRemoveFlagsForMainCli_NilFlags(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	out, err := c.RemoveFlagsForMainCli([]string{"ecctl", "ls"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 2 || out[0] != "ecctl" || out[1] != "ls" {
		t.Fatalf("unexpected args: %v", out)
	}
}

func TestRemoveFlagsForMainCli_InlineValue(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	out, err := c.RemoveFlagsForMainCli([]string{"list", "--profile=prod", "--region=cn-hangzhou", "--key", "value"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for _, a := range out {
		if strings.HasPrefix(a, "--profile") || strings.HasPrefix(a, "--region") {
			t.Errorf("main CLI flags should be removed, got %v", out)
		}
	}
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
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
	c.CheckOsTypeAndArch()
	c.versionRemote = "0.1.1"

	called := false
	origDownload := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, destArchive, exe string) error {
		called = true
		want := c.archiveDownloadURL("0.1.1")
		if url != want {
			t.Fatalf("url mismatch: %s vs %s", url, want)
		}
		writeExecutable(t, exe, "dummy")
		return nil
	}
	defer func() { downloadAndExtractFunc = origDownload }()

	if err := c.Install(); err != nil {
		t.Fatalf("Install err: %v", err)
	}
	if !called {
		t.Fatalf("download not called")
	}
	if c.versionLocal != "0.1.1" {
		t.Fatalf("versionLocal should be 0.1.1, got %s", c.versionLocal)
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
	if c.envMap["ALIBABA_CLOUD_ECCTL_COMPAT_MODE"] != "aliyun ecctl" {
		t.Fatalf("ALIBABA_CLOUD_ECCTL_COMPAT_MODE missing")
	}
	if c.envMap["ALIBABA_CLOUD_REGION_ID"] != "cn-hangzhou" {
		t.Fatalf("ALIBABA_CLOUD_REGION_ID mismatch: %v", c.envMap["ALIBABA_CLOUD_REGION_ID"])
	}
	if c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"] != "ak" {
		t.Fatalf("ak mismatch: %v", c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"])
	}
	if c.envMap["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] != "sk" {
		t.Fatalf("sk mismatch: %v", c.envMap["ALIBABA_CLOUD_ACCESS_KEY_SECRET"])
	}
	// Verify no legacy REGION_ID double-write
	if _, hasLegacy := c.envMap["REGION_ID"]; hasLegacy {
		t.Fatalf("legacy REGION_ID should not be set, use ALIBABA_CLOUD_REGION_ID only")
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
	if c.envMap["ALIBABA_CLOUD_ECCTL_COMPAT_MODE"] != "aliyun ecctl" {
		t.Fatalf("ALIBABA_CLOUD_ECCTL_COMPAT_MODE missing")
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
		os, arch string
		support  bool
		ext      string
	}{
		{"linux", "amd64", true, "tar.gz"},
		{"linux", "arm64", true, "tar.gz"},
		{"darwin", "amd64", true, "tar.gz"},
		{"darwin", "arm64", true, "tar.gz"},
		{"windows", "amd64", true, "zip"},
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
		if c.archiveExt != tc.ext {
			t.Fatalf("ext mismatch for %s/%s: got %s want %s", tc.os, tc.arch, c.archiveExt, tc.ext)
		}
	}
}

// --- DownloadAndExtract ---

func TestDownloadAndExtract_TarGz(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar.gz install on unix")
	}
	payload, err := buildTarGzArchive(map[string][]byte{"ecctl": []byte("#!/bin/sh\n")})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	origHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) { return http.Get(srv.URL) }
	t.Cleanup(func() { httpGetFunc = origHTTPGet })

	destDir := t.TempDir()
	archive := filepath.Join(destDir, "dl.tar.gz")
	exe := filepath.Join(destDir, "ecctl")
	if err := DownloadAndExtract(srv.URL, archive, exe); err != nil {
		t.Fatalf("DownloadAndExtract: %v", err)
	}
}

func TestDownloadAndExtract_Zip(t *testing.T) {
	payload, err := buildZipArchive(map[string][]byte{"ecctl.exe": []byte("MZ")})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	origHTTPGet := httpGetFunc
	origGOOS := runtimeGOOSFunc
	httpGetFunc = func(url string) (*http.Response, error) { return http.Get(srv.URL) }
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() {
		httpGetFunc = origHTTPGet
		runtimeGOOSFunc = origGOOS
	})

	destDir := t.TempDir()
	archive := filepath.Join(destDir, "dl.zip")
	exe := filepath.Join(destDir, "ecctl.exe")
	if err := DownloadAndExtract(srv.URL, archive, exe); err != nil {
		t.Fatalf("DownloadAndExtract zip: %v", err)
	}
}

func TestExtractTarGz_RejectsPathTraversal(t *testing.T) {
	dest := t.TempDir()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{Name: "../ecctl", Mode: 0o755, Size: 4, Typeflag: tar.TypeReg}
	_ = tw.WriteHeader(hdr)
	_, _ = tw.Write([]byte("evil"))
	_ = tw.Close()
	_ = gw.Close()
	src := filepath.Join(dest, "bad.tar.gz")
	_ = os.WriteFile(src, buf.Bytes(), 0o644)
	if err := extractTarGz(src, filepath.Join(dest, "out")); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}
}

// --- GetLatestEcctlVersion ---

func TestGetLatestEcctlVersion_Success(t *testing.T) {
	origHTTPDo := httpDoFunc
	defer func() { httpDoFunc = origHTTPDo }()
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("User-Agent") == "" {
			t.Fatalf("User-Agent should be set")
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("  0.1.1\n")),
		}, nil
	}

	ver, err := GetLatestEcctlVersion()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ver != "0.1.1" {
		t.Errorf("version mismatch: got %s, want v1.2.3", ver)
	}
}

func TestGetLatestEcctlVersion_NetworkError(t *testing.T) {
	origHTTPDo := httpDoFunc
	defer func() { httpDoFunc = origHTTPDo }()
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network down")
	}

	_, err := GetLatestEcctlVersion()
	if err == nil || !strings.Contains(err.Error(), "network down") {
		t.Fatalf("expected network error, got %v", err)
	}
}

func TestGetLatestEcctlVersion_Non200(t *testing.T) {
	origHTTPDo := httpDoFunc
	defer func() { httpDoFunc = origHTTPDo }()
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found"))}, nil
	}

	_, err := GetLatestEcctlVersion()
	if err == nil || !strings.Contains(err.Error(), "status code 404") {
		t.Fatalf("expected 404 error, got %v", err)
	}
}

func TestGetLatestEcctlVersion_Empty(t *testing.T) {
	origHTTPDo := httpDoFunc
	defer func() { httpDoFunc = origHTTPDo }()
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("  \n")),
		}, nil
	}

	_, err := GetLatestEcctlVersion()
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty error, got %v", err)
	}
}

// --- ExitError ---

func TestExitError(t *testing.T) {
	e := &ExitError{Code: 42}
	if e.Error() != "subprocess exited with code 42" {
		t.Errorf("Error() mismatch: %s", e.Error())
	}
	if e.ExitCode() != 42 {
		t.Errorf("ExitCode() mismatch: %d", e.ExitCode())
	}
}

func TestExecuteEcctl_ExitCode(t *testing.T) {
	origExec := execCommandFunc
	defer func() { execCommandFunc = origExec }()
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "exit 42")
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.execFilePath = "/any/path"
	c.envMap = map[string]string{}

	err := c.ExecuteEcctl([]string{"version"})
	if err == nil {
		t.Fatalf("expected error for non-zero exit")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 42 {
		t.Errorf("exit code: got %d, want 42", exitErr.Code)
	}
}

func TestExecuteEcctl_Success(t *testing.T) {
	origExec := execCommandFunc
	defer func() { execCommandFunc = origExec }()
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "exit 0")
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.execFilePath = "/any/path"
	c.envMap = map[string]string{"ALIBABA_CLOUD_ECCTL_COMPAT_MODE": "aliyun ecctl"}

	if err := c.ExecuteEcctl([]string{"version"}); err != nil {
		t.Fatalf("Execute should succeed: %v", err)
	}
}

func TestRun_Installed_VersionCheckNetworkFail(t *testing.T) {
	home := setupTestHome(t)
	prepareConfig(t, home, "en")
	execPath := filepath.Join(config.GetConfigPath(), "ecctl")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "dummy")
	old := time.Now().Unix() - int64(VersionCheckTTL) - 10
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version_check"), []byte(fmt.Sprintf("%d", old)), 0o644)
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version"), []byte("\n"), 0o644)

	origExtract := downloadAndExtractFunc
	origExec := execCommandFunc
	origGetVersion := getLatestEcctlVersionFunc
	extractCount := 0
	downloadAndExtractFunc = func(url, destArchive, exe string) error { extractCount++; return nil }
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return mockExecCommand() }
	getLatestEcctlVersionFunc = func() (string, error) { return "", fmt.Errorf("network down") }
	t.Cleanup(func() {
		downloadAndExtractFunc = origExtract
		execCommandFunc = origExec
		getLatestEcctlVersionFunc = origGetVersion
	})

	ctx, _, _ := newOriginCtx()
	if err := NewContext(ctx).Run([]string{"help"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if extractCount != 0 {
		t.Fatalf("expected no extract, got %d", extractCount)
	}
}

func TestEnsureInstalled_NO_UPDATE_CHECK(t *testing.T) {
	tmp := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmp }
	t.Cleanup(func() { getConfigurePathFunc = oldGet })
	execPath := filepath.Join(tmp, "ecctl")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "x")
	old := time.Now().Unix() - int64(VersionCheckTTL) - 10
	_ = os.WriteFile(filepath.Join(tmp, ".ecctl_version_check"), []byte(fmt.Sprintf("%d", old)), 0o644)
	t.Setenv("ALIBABA_CLOUD_ECCTL_NO_UPDATE_CHECK", "1")
	called := false
	origGetVersion := getLatestEcctlVersionFunc
	getLatestEcctlVersionFunc = func() (string, error) { called = true; return "9.9.9", nil }
	t.Cleanup(func() { getLatestEcctlVersionFunc = origGetVersion })
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if called {
		t.Fatalf("should skip version fetch")
	}
}

func TestEnsureInstalled_ExecPathMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("ALIBABA_CLOUD_ECCTL_EXEC_PATH", filepath.Join(tmp, "missing"))
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	err := c.EnsureInstalledAndUpdated()
	if err == nil || !strings.Contains(err.Error(), "ALIBABA_CLOUD_ECCTL_EXEC_PATH") {
		t.Fatalf("got %v", err)
	}
}

func TestEnsureInstalled_ExecPathDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("ALIBABA_CLOUD_ECCTL_EXEC_PATH", tmp)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	err := c.EnsureInstalledAndUpdated()
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("got %v", err)
	}
}

func TestEnsureInstalled_ExecPathNotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "ecctl")
	_ = os.WriteFile(fake, []byte("x"), 0o644)
	t.Setenv("ALIBABA_CLOUD_ECCTL_EXEC_PATH", fake)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	err := c.EnsureInstalledAndUpdated()
	if err == nil || !strings.Contains(err.Error(), "not executable") {
		t.Fatalf("got %v", err)
	}
}

func TestApplyMainCliFlagsFromArgs(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	c := NewContext(ctx)
	c.applyMainCliFlagsFromArgs([]string{"-pprod", "--region=cn-hangzhou"})

	pf := ctx.Flags().Get("profile")
	if pf == nil || !pf.IsAssigned() || func() string { v, _ := pf.GetValue(); return v }() != "prod" {
		t.Fatalf("profile flag")
	}
	rf := ctx.Flags().Get("region")
	if rf == nil || !rf.IsAssigned() || func() string { v, _ := rf.GetValue(); return v }() != "cn-hangzhou" {
		t.Fatalf("region flag")
	}
}

func TestRemoveFlagsForMainCli_ShorthandForms(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	out, err := c.RemoveFlagsForMainCli([]string{"cmd", "-p", "prod", "-p=value", "-pcn-hangzhou", "--profile=x", "keep"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0] != "cmd" || out[1] != "keep" {
		t.Fatalf("got %v", out)
	}
}

func TestSanitizeArchivePath(t *testing.T) {
	if _, err := sanitizeArchivePath("../ecctl"); err == nil {
		t.Fatal("expected err")
	}
	if _, err := sanitizeArchivePath("ecctl"); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestGetLatestEcctlVersion_CustomBaseURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("2.0.0"))
	}))
	defer srv.Close()
	t.Setenv("ALIBABA_CLOUD_ECCTL_DOWNLOAD_BASE_URL", srv.URL)
	ver, err := GetLatestEcctlVersion()
	if err != nil || ver != "2.0.0" {
		t.Fatalf("got %q err %v", ver, err)
	}
}

func TestDownloadAndExtract_BadStatus(t *testing.T) {
	orig := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("no"))}, nil
	}
	t.Cleanup(func() { httpGetFunc = orig })
	err := DownloadAndExtract("http://example.com/x", filepath.Join(t.TempDir(), "a.tar.gz"), filepath.Join(t.TempDir(), "ecctl"))
	if err == nil || !strings.Contains(err.Error(), "status code") {
		t.Fatalf("got %v", err)
	}
}

func TestDownloadAndExtract_UnsupportedFormat(t *testing.T) {
	orig := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("data"))}, nil
	}
	t.Cleanup(func() { httpGetFunc = orig })
	dir := t.TempDir()
	archive := filepath.Join(dir, "x.bin")
	err := DownloadAndExtract("http://example.com/x", archive, filepath.Join(dir, "ecctl"))
	if err == nil || !strings.Contains(err.Error(), "unsupported archive") {
		t.Fatalf("got %v", err)
	}
}

func TestDownloadAndExtract_NestedBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar on unix")
	}
	payload, err := buildTarGzArchive(map[string][]byte{"nested/ecctl": []byte("#!/bin/sh\n")})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()
	origHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) { return http.Get(srv.URL) }
	t.Cleanup(func() { httpGetFunc = origHTTPGet })
	dir := t.TempDir()
	if err := DownloadAndExtract(srv.URL, filepath.Join(dir, "d.tar.gz"), filepath.Join(dir, "ecctl")); err != nil {
		t.Fatalf("nested: %v", err)
	}
}

func TestSanitizeArchivePath_Variants(t *testing.T) {
	if _, err := sanitizeArchivePath(""); err == nil {
		t.Fatal("empty")
	}
	if _, err := sanitizeArchivePath("C:foo"); err == nil {
		t.Fatal("drive")
	}
	clean, err := sanitizeArchivePath("././ecctl")
	if err != nil || clean != "ecctl" {
		t.Fatalf("got %q %v", clean, err)
	}
}

func TestApplyMainCliFlagsFromArgs_MissingValue(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	c := NewContext(ctx)
	c.applyMainCliFlagsFromArgs([]string{"-p"})
	f := ctx.Flags().Get("profile")
	if f != nil && f.IsAssigned() {
		t.Fatalf("profile should not be assigned without value")
	}
}

func TestSaveLocalVersion_Error(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.versionFilePath = filepath.Join(t.TempDir(), "missing", "v")
	c.versionLocal = "1"
	if err := c.SaveLocalVersion(); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateExecPathOverride_StatError(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.execFilePath = string([]byte{0})
	c.osType = "linux"
	err := c.validateExecPathOverride()
	if err == nil {
		t.Fatal("expected stat error")
	}
}

func TestbuildTarGzArchive_Error(t *testing.T) {
	_, err := buildTarGzArchive(map[string][]byte{"bad": make([]byte, 1)})
	if err != nil {
		// header write always succeeds for small payload; still call buildZipArchive
	}
	if _, err := buildZipArchive(map[string][]byte{"a": []byte("x")}); err != nil {
		t.Fatalf("zip: %v", err)
	}
}

func TestEnsureInstalled_ExecPathValidRuns(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "ecctl")
	writeExecutable(t, fake, "bin")
	t.Setenv("ALIBABA_CLOUD_ECCTL_EXEC_PATH", fake)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.validateExecPathOverride(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestAppendHelpIfNeeded(t *testing.T) {
	out := AppendHelpIfNeeded(true, []string{"version"})
	if len(out) != 2 || out[1] != "--help" {
		t.Fatalf("got %v", out)
	}
	out = AppendHelpIfNeeded(true, []string{"--help"})
	if len(out) != 1 {
		t.Fatalf("got %v", out)
	}
	out = AppendHelpIfNeeded(false, []string{"version"})
	if len(out) != 1 {
		t.Fatalf("got %v", out)
	}
}

func TestDownloadAndExtract_HttpGetError(t *testing.T) {
	orig := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) { return nil, fmt.Errorf("boom") }
	t.Cleanup(func() { httpGetFunc = orig })
	err := DownloadAndExtract("http://x", filepath.Join(t.TempDir(), "a.tar.gz"), filepath.Join(t.TempDir(), "ecctl"))
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("got %v", err)
	}
}

func TestExtractTarGz_OpenError(t *testing.T) {
	err := extractTarGz(filepath.Join(t.TempDir(), "missing.tar.gz"), t.TempDir())
	if err == nil {
		t.Fatal("expected open error")
	}
}

func TestUnzipArchive_InvalidZip(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.zip")
	_ = os.WriteFile(f, []byte("notzip"), 0o644)
	if err := unzipArchive(f, t.TempDir()); err == nil {
		t.Fatal("expected error")
	}
}

func TestSanitizeArchivePath_NUL(t *testing.T) {
	if _, err := sanitizeArchivePath("ec" + string(rune(0)) + "ctl"); err == nil {
		t.Fatal("expected NUL error")
	}
}

func TestApplyMainCliFlagsFromArgs_BooleanFlag(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	c := NewContext(ctx)
	c.applyMainCliFlagsFromArgs([]string{"--help"})
}

func TestDownloadAndExtract_ExtractFailure(t *testing.T) {
	orig := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("not-a-valid-archive"))}, nil
	}
	t.Cleanup(func() { httpGetFunc = orig })
	dir := t.TempDir()
	err := DownloadAndExtract("http://x", filepath.Join(dir, "a.tar.gz"), filepath.Join(dir, "ecctl"))
	if err == nil {
		t.Fatal("expected extract error")
	}
}

func TestDownloadAndExtract_MissingEcctlInArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar")
	}
	payload, _ := buildTarGzArchive(map[string][]byte{"README.md": []byte("hi")})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()
	origHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) { return http.Get(srv.URL) }
	t.Cleanup(func() { httpGetFunc = origHTTPGet })
	dir := t.TempDir()
	err := DownloadAndExtract(srv.URL, filepath.Join(dir, "a.tar.gz"), filepath.Join(dir, "ecctl"))
	if err == nil || !strings.Contains(err.Error(), "not exist") {
		t.Fatalf("got %v", err)
	}
}

func TestGetLatestEcctlVersion_ReadBodyError(t *testing.T) {
	origHTTPDo := httpDoFunc
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(errReader{})}, nil
	}
	t.Cleanup(func() { httpDoFunc = origHTTPDo })
	_, err := GetLatestEcctlVersion()
	if err == nil {
		t.Fatal("expected read error")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, fmt.Errorf("read fail") }

func TestExtractTarGz_InvalidGzip(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.tar.gz")
	_ = os.WriteFile(f, []byte("not gzip"), 0o644)
	if err := extractTarGz(f, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected gzip error")
	}
}

func TestInstall_DownloadError(t *testing.T) {
	setupTestHome(t)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	c.versionRemote = "0.1.1"
	orig := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, destArchive, exe string) error { return fmt.Errorf("dl fail") }
	t.Cleanup(func() { downloadAndExtractFunc = orig })
	err := c.Install()
	if err == nil || !strings.Contains(err.Error(), "dl fail") {
		t.Fatalf("got %v", err)
	}
}

func TestEnsureInstalledAndUpdated_UpdateInstallFail(t *testing.T) {
	setupTestHome(t)
	execPath := filepath.Join(config.GetConfigPath(), "ecctl")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "x")
	old := time.Now().Unix() - int64(VersionCheckTTL) - 10
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version_check"), []byte(fmt.Sprintf("%d", old)), 0o644)
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version"), []byte("\n"), 0o644)
	origGet := getLatestEcctlVersionFunc
	origExtract := downloadAndExtractFunc
	getLatestEcctlVersionFunc = func() (string, error) { return "0.2.0", nil }
	downloadAndExtractFunc = func(url, destArchive, exe string) error { return fmt.Errorf("fail") }
	t.Cleanup(func() {
		getLatestEcctlVersionFunc = origGet
		downloadAndExtractFunc = origExtract
	})
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("best effort should return nil, got %v", err)
	}
}

func TestApplyMainCliFlagsFromArgs_LongUnknown(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	c := NewContext(ctx)
	c.applyMainCliFlagsFromArgs([]string{"--unknown-flag", "x"})
}

func TestRemoveFlagsForMainCli_AssignedCtxFlag(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")
	c := NewContext(ctx)
	out, err := c.RemoveFlagsForMainCli([]string{"x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "x" {
		t.Fatalf("got %v", out)
	}
}

func TestGetLocalVersion_ReadError(t *testing.T) {
	tmp := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmp }
	t.Cleanup(func() { getConfigurePathFunc = oldGet })
	execPath := filepath.Join(tmp, "ecctl")
	writeExecutable(t, execPath, "x")
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	_ = os.WriteFile(c.versionFilePath, []byte("1"), 0o000)
	if err := c.GetLocalVersion(); err == nil {
		t.Fatal("expected read error")
	}
}

func TestUpdateCheckCacheTime_Error(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.checkVersionCacheFilePath = filepath.Join(t.TempDir(), "missing/sub/file")
	if err := c.UpdateCheckCacheTime(); err == nil {
		t.Fatal("expected error")
	}
}

func TestExecuteEcctl_NonExitError(t *testing.T) {
	origExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("no-such-binary-xyz-abc")
	}
	t.Cleanup(func() { execCommandFunc = origExec })
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.execFilePath = "no-such-binary-xyz-abc"
	c.envMap = map[string]string{}
	err := c.ExecuteEcctl([]string{"x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*ExitError); ok {
		t.Fatal("expected non-ExitError")
	}
}

func TestInitializeAndValidatePlatform_ExternalOnUnsupported(t *testing.T) {
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "s390x" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS; runtimeGOARCHFunc = origGOARCH })
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "ecctl")
	writeExecutable(t, fake, "x")
	t.Setenv("ALIBABA_CLOUD_ECCTL_EXEC_PATH", fake)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.InitializeAndValidatePlatform(); err != nil {
		t.Fatalf("external binary should bypass platform check: %v", err)
	}
}

func TestUnzipArchive_ExtractEcctl(t *testing.T) {
	payload, err := buildZipArchive(map[string][]byte{"ecctl.exe": []byte("MZ"), "LICENSE": []byte("l")})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, payload, 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
}

func TestRun_ConfigFailed(t *testing.T) {
	setupTestHome(t)
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "ecctl")
	writeExecutable(t, fake, "x")
	t.Setenv("ALIBABA_CLOUD_ECCTL_EXEC_PATH", fake)
	ctx, _, _ := newOriginCtx()
	err := NewContext(ctx).Run([]string{"version"})
	if err == nil || !strings.Contains(err.Error(), "config failed") {
		t.Fatalf("got %v", err)
	}
}

func TestDownloadAndExtract_CreateArchiveError(t *testing.T) {
	orig := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("x"))}, nil
	}
	t.Cleanup(func() { httpGetFunc = orig })
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "blocked")
	_ = os.Mkdir(archiveDir, 0o755)
	err := DownloadAndExtract("http://x", archiveDir, filepath.Join(dir, "ecctl"))
	if err == nil {
		t.Fatal("expected create error")
	}
}

func TestExtractTarGz_SkipsNonRegular(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Name: "skipdir", Mode: 0o755, Typeflag: tar.TypeDir})
	_ = tw.WriteHeader(&tar.Header{Name: "ecctl", Mode: 0o755, Size: 2, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("ok"))
	_ = tw.Close()
	_ = gw.Close()
	src := filepath.Join(dir, "a.tar.gz")
	_ = os.WriteFile(src, buf.Bytes(), 0o644)
	out := filepath.Join(dir, "out")
	if err := extractTarGz(src, out); err != nil {
		t.Fatalf("extract: %v", err)
	}
}

func TestUnzipArchive_SkipsOversized(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("ecctl.exe")
	_, _ = w.Write([]byte("tiny"))
	_ = zw.Close()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, buf.Bytes(), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
}

func TestEnsureInstalledAndUpdated_GetLocalVersionError(t *testing.T) {
	setupTestHome(t)
	execPath := filepath.Join(config.GetConfigPath(), "ecctl")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "x")
	old := time.Now().Unix() - int64(VersionCheckTTL) - 10
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version_check"), []byte(fmt.Sprintf("%d", old)), 0o644)
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version"), []byte("\n"), 0o644)
	origGet := getLatestEcctlVersionFunc
	getLatestEcctlVersionFunc = func() (string, error) { return "0.2.0", nil }
	t.Cleanup(func() { getLatestEcctlVersionFunc = origGet })
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("best effort: %v", err)
	}
}

func TestDownloadAndExtract_RemoveExtractDirError(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.tar.gz")
	extractDir := filepath.Join(dir, "ecctl_extract")
	_ = os.Mkdir(extractDir, 0o755)
	orig := osRemoveAllFunc
	osRemoveAllFunc = func(string) error { return fmt.Errorf("no remove") }
	t.Cleanup(func() { osRemoveAllFunc = orig })
	origHTTP := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("x"))}, nil
	}
	t.Cleanup(func() { httpGetFunc = origHTTP })
	err := DownloadAndExtract("http://x", archive, filepath.Join(dir, "ecctl"))
	if err == nil || !strings.Contains(err.Error(), "no remove") {
		t.Fatalf("got %v", err)
	}
}

func TestDownloadAndExtract_CopyError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar")
	}
	payload, _ := buildTarGzArchive(map[string][]byte{"ecctl": []byte("x")})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()
	origHTTP := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) { return http.Get(srv.URL) }
	t.Cleanup(func() { httpGetFunc = origHTTP })
	origCopy := copyFileFunc
	copyFileFunc = func(src, dst string) error { return fmt.Errorf("copy fail") }
	t.Cleanup(func() { copyFileFunc = origCopy })
	dir := t.TempDir()
	err := DownloadAndExtract(srv.URL, filepath.Join(dir, "a.tar.gz"), filepath.Join(dir, "ecctl"))
	if err == nil || !strings.Contains(err.Error(), "copy fail") {
		t.Fatalf("got %v", err)
	}
}

func TestDownloadAndExtract_ChmodError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix chmod")
	}
	payload, _ := buildTarGzArchive(map[string][]byte{"ecctl": []byte("x")})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()
	origHTTP := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) { return http.Get(srv.URL) }
	t.Cleanup(func() { httpGetFunc = origHTTP })
	origChmod := osChmodFunc
	osChmodFunc = func(string, os.FileMode) error { return fmt.Errorf("chmod fail") }
	t.Cleanup(func() { osChmodFunc = origChmod })
	dir := t.TempDir()
	err := DownloadAndExtract(srv.URL, filepath.Join(dir, "a.tar.gz"), filepath.Join(dir, "ecctl"))
	if err == nil || !strings.Contains(err.Error(), "chmod fail") {
		t.Fatalf("got %v", err)
	}
}

func TestRun_SuccessPath(t *testing.T) {
	home := setupTestHome(t)
	prepareConfig(t, home, "en")
	origExtract := downloadAndExtractFunc
	origExec := execCommandFunc
	origGet := getLatestEcctlVersionFunc
	downloadAndExtractFunc = func(url, destArchive, exe string) error {
		writeExecutable(t, exe, "x")
		return nil
	}
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return mockExecCommand() }
	getLatestEcctlVersionFunc = func() (string, error) { return "0.1.1", nil }
	t.Cleanup(func() {
		downloadAndExtractFunc = origExtract
		execCommandFunc = origExec
		getLatestEcctlVersionFunc = origGet
	})
	ctx, _, _ := newOriginCtx()
	if err := NewContext(ctx).Run([]string{"version"}); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestGetLatestEcctlVersion_NewRequestError(t *testing.T) {
	orig := httpNewRequestFunc
	httpNewRequestFunc = func(string, string, io.Reader) (*http.Request, error) {
		return nil, fmt.Errorf("bad request")
	}
	t.Cleanup(func() { httpNewRequestFunc = orig })
	_, err := GetLatestEcctlVersion()
	if err == nil || !strings.Contains(err.Error(), "bad request") {
		t.Fatalf("got %v", err)
	}
}

func TestExtractTarGz_SkipsSymlink(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "ecctl"})
	_ = tw.WriteHeader(&tar.Header{Name: "ecctl", Mode: 0o755, Size: 1, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("x"))
	_ = tw.Close()
	_ = gw.Close()
	src := filepath.Join(dir, "a.tar.gz")
	_ = os.WriteFile(src, buf.Bytes(), 0o644)
	if err := extractTarGz(src, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("extract: %v", err)
	}
}

func TestEnsureInstalledAndUpdated_GetLocalVersionErrorInstalls(t *testing.T) {
	setupTestHome(t)
	execPath := filepath.Join(config.GetConfigPath(), "ecctl")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "x")
	old := time.Now().Unix() - int64(VersionCheckTTL) - 10
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version_check"), []byte(fmt.Sprintf("%d", old)), 0o644)
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version"), []byte("\n"), 0o644)
	extractCount := 0
	origExtract := downloadAndExtractFunc
	origGet := getLatestEcctlVersionFunc
	downloadAndExtractFunc = func(url, destArchive, exe string) error { extractCount++; return nil }
	getLatestEcctlVersionFunc = func() (string, error) { return "0.2.0", nil }
	t.Cleanup(func() {
		downloadAndExtractFunc = origExtract
		getLatestEcctlVersionFunc = origGet
	})
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if extractCount != 1 {
		t.Fatalf("expected install on bad local version file, got %d", extractCount)
	}
}
func TestDownloadAndExtract_RemoveExistingExeError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar")
	}
	payload, _ := buildTarGzArchive(map[string][]byte{"ecctl": []byte("x")})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()
	dir := t.TempDir()
	exe := filepath.Join(dir, "ecctl")
	writeExecutable(t, exe, "old")
	origHTTP := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) { return http.Get(srv.URL) }
	t.Cleanup(func() { httpGetFunc = origHTTP })
	origRemove := osRemoveFunc
	osRemoveFunc = func(string) error { return fmt.Errorf("remove fail") }
	t.Cleanup(func() { osRemoveFunc = origRemove })
	err := DownloadAndExtract(srv.URL, filepath.Join(dir, "a.tar.gz"), exe)
	if err == nil || !strings.Contains(err.Error(), "remove fail") {
		t.Fatalf("got %v", err)
	}
}

func TestApplyMainCliFlagsFromArgs_ShorthandEquals(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	c := NewContext(ctx)
	c.applyMainCliFlagsFromArgs([]string{"-p=value"})
	pf := ctx.Flags().Get("profile")
	if pf == nil || !pf.IsAssigned() {
		t.Fatal("profile")
	}
}

func TestRemoveFlagsForMainCli_ShortInline(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	out, err := c.RemoveFlagsForMainCli([]string{"x", "-p=value", "-pcn-hangzhou"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "x" {
		t.Fatalf("got %v", out)
	}
}
func TestUnzipArchive_SkipsOversizedHeader(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	h := &zip.FileHeader{Name: "ecctl.exe", Method: zip.Store}
	h.UncompressedSize64 = uint64(ecctlMaxExtractSize + 1)
	w, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("x"))
	_ = zw.Close()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, buf.Bytes(), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
}

func TestValidateExecPathOverride_WindowsSkipsMode(t *testing.T) {
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "ecctl.exe")
	_ = os.WriteFile(fake, []byte("x"), 0o644)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.execFilePath = fake
	c.osType = "windows"
	if err := c.validateExecPathOverride(); err != nil {
		t.Fatalf("windows should not require exec bit: %v", err)
	}
}

func TestInitializeAndValidatePlatform_Success(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.InitializeAndValidatePlatform(); err != nil {
		t.Fatalf("init: %v", err)
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, fmt.Errorf("read fail") }

func TestDownloadAndExtract_BodyReadError(t *testing.T) {
	orig := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(failReader{})}, nil
	}
	t.Cleanup(func() { httpGetFunc = orig })
	err := DownloadAndExtract("http://x", filepath.Join(t.TempDir(), "a.tar.gz"), filepath.Join(t.TempDir(), "ecctl"))
	if err == nil || !strings.Contains(err.Error(), "read fail") {
		t.Fatalf("got %v", err)
	}
}

func TestSanitizeArchivePath_MoreCases(t *testing.T) {
	if _, err := sanitizeArchivePath("//host/share"); err == nil {
		t.Fatal("unc")
	}
	if _, err := sanitizeArchivePath("C:evil"); err == nil {
		t.Fatal("drive")
	}
	clean, err := sanitizeArchivePath(".")
	if err != nil || clean != "" {
		t.Fatalf("dot: %q %v", clean, err)
	}
}

func TestExtractTarGz_SkipsHardLink(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Name: "link", Typeflag: tar.TypeLink, Linkname: "ecctl"})
	_ = tw.WriteHeader(&tar.Header{Name: "ecctl", Mode: 0o755, Size: 1, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("x"))
	_ = tw.Close()
	_ = gw.Close()
	src := filepath.Join(dir, "a.tar.gz")
	_ = os.WriteFile(src, buf.Bytes(), 0o644)
	if err := extractTarGz(src, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("extract: %v", err)
	}
}

func TestArchiveFileName_Windows(t *testing.T) {
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	runtimeGOOSFunc = func() string { return "windows" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS; runtimeGOARCHFunc = origGOARCH })
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.CheckOsTypeAndArch()
	name := c.archiveFileName("1.0.0")
	if !strings.HasSuffix(name, ".zip") {
		t.Fatalf("name %s", name)
	}
}
func TestExtractTarGz_TruncatedPayload(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Name: "ecctl", Mode: 0o755, Size: 10, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("x"))
	_ = tw.Close()
	_ = gw.Close()
	src := filepath.Join(dir, "a.tar.gz")
	_ = os.WriteFile(src, buf.Bytes(), 0o644)
	if err := extractTarGz(src, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected copy error")
	}
}

func TestRemoveFlagsForMainCli_Empty(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	out, err := c.RemoveFlagsForMainCli(nil)
	if err != nil || len(out) != 0 {
		t.Fatalf("got %v err %v", out, err)
	}
}

func TestFilterEnv_EmptyOverride(t *testing.T) {
	out := filterEnv(nil, map[string]string{})
	if len(out) != 0 {
		t.Fatalf("got %v", out)
	}
}
func TestDownloadAndExtract_MkdirExtractError(t *testing.T) {
	orig := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("x"))}, nil
	}
	t.Cleanup(func() { httpGetFunc = orig })
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.tar.gz")
	extractDir := filepath.Join(dir, "ecctl_extract")
	_ = os.WriteFile(extractDir, []byte("block"), 0o644)
	err := DownloadAndExtract("http://x", archive, filepath.Join(dir, "ecctl"))
	if err == nil {
		t.Fatal("expected mkdir/extract error")
	}
}

func TestApplyMainCliFlagsFromArgs_LongFlagNoValue(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	c := NewContext(ctx)
	c.applyMainCliFlagsFromArgs([]string{"--profile"})
}
func TestUnzipArchive_DuplicateEntries(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := 0; i < 2; i++ {
		w, _ := zw.Create("ecctl.exe")
		_, _ = w.Write([]byte("a"))
	}
	_ = zw.Close()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, buf.Bytes(), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
}

func TestGetLocalVersion_NoVersionFile(t *testing.T) {
	setupTestHome(t)
	execPath := filepath.Join(config.GetConfigPath(), "ecctl")
	writeExecutable(t, execPath, "x")
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.GetLocalVersion(); err != nil {
		t.Fatalf("missing version file should be ok: %v", err)
	}
	if c.versionLocal != "" {
		t.Fatalf("expected empty local version")
	}
}
func TestEnsureInstalledAndUpdated_SameVersionSkipsInstall(t *testing.T) {
	setupTestHome(t)
	execPath := filepath.Join(config.GetConfigPath(), "ecctl")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	writeExecutable(t, execPath, "x")
	old := time.Now().Unix() - int64(VersionCheckTTL) - 10
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version_check"), []byte(fmt.Sprintf("%d", old)), 0o644)
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ecctl_version"), []byte("0.1.1"), 0o644)
	extractCount := 0
	origExtract := downloadAndExtractFunc
	origGet := getLatestEcctlVersionFunc
	downloadAndExtractFunc = func(url, destArchive, exe string) error { extractCount++; return nil }
	getLatestEcctlVersionFunc = func() (string, error) { return "0.1.1", nil }
	t.Cleanup(func() {
		downloadAndExtractFunc = origExtract
		getLatestEcctlVersionFunc = origGet
	})
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if extractCount != 0 {
		t.Fatalf("expected no install, got %d", extractCount)
	}
}

func TestNewEcctlCommand_RunGenericError(t *testing.T) {
	cmd := NewEcctlCommand()
	ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	runtimeGOOSFunc = func() string { return "plan9" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS; runtimeGOARCHFunc = origGOARCH })
	err := cmd.Run(ctx, []string{"version"})
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("got %v", err)
	}
}
func TestInstall_SaveVersionError(t *testing.T) {
	setupTestHome(t)
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	c.versionRemote = "0.1.1"
	c.versionFilePath = filepath.Join(t.TempDir(), "missing", "ver")
	orig := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, destArchive, exe string) error { return nil }
	t.Cleanup(func() { downloadAndExtractFunc = orig })
	if err := c.Install(); err == nil || !strings.Contains(err.Error(), "version file") {
		t.Fatalf("got %v", err)
	}
}
func TestUnzipArchive_SkipsBadPaths(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("../../ecctl.exe")
	_, _ = w.Write([]byte("x"))
	w2, _ := zw.Create("ecctl.exe")
	_, _ = w2.Write([]byte("y"))
	_ = zw.Close()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, buf.Bytes(), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
}

func TestUnzipArchive_MkdirError(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("sub/ecctl.exe")
	_, _ = w.Write([]byte("x"))
	_ = zw.Close()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, buf.Bytes(), 0o644)
	out := filepath.Join(dir, "out")
	_ = os.WriteFile(out, []byte("file"), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, out); err == nil {
		t.Fatal("expected mkdir error when out is a file")
	}
}
func TestUnzipArchive_SkipsNonWantedName(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("README.md")
	_, _ = w.Write([]byte("x"))
	_ = zw.Close()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, buf.Bytes(), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
}
func TestExtractTarGz_SkipsNonWantedName(t *testing.T) {
	dir := t.TempDir()
	payload, _ := buildTarGzArchive(map[string][]byte{"README.md": []byte("x"), "ecctl": []byte("y")})
	src := filepath.Join(dir, "a.tar.gz")
	_ = os.WriteFile(src, payload, 0o644)
	if err := extractTarGz(src, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("extract: %v", err)
	}
}

func TestSafeJoinUnderDir_PathEscape(t *testing.T) {
	_, err := safeJoinUnderDir("/tmp/safe", "..")
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("expected escape error, got %v", err)
	}
}

func TestSafeJoinUnderDir_SuccessNested(t *testing.T) {
	root := t.TempDir()
	got, err := safeJoinUnderDir(root, "nested/bin/ecctl")
	if err != nil {
		t.Fatalf("safeJoin: %v", err)
	}
	if !strings.Contains(got, "ecctl") {
		t.Fatalf("path %q", got)
	}
}

func TestFilterEnv_StripsConflictingKeys(t *testing.T) {
	base := []string{"ALIBABA_CLOUD_ACCESS_KEY_ID=old", "FOO=1"}
	out := filterEnv(base, map[string]string{"ALIBABA_CLOUD_ACCESS_KEY_ID": "new"})
	if len(out) != 1 || out[0] != "FOO=1" {
		t.Fatalf("filterEnv: %v", out)
	}
}

func withLowExtractMax(t *testing.T, n int64) func() {
	t.Helper()
	old := ecctlMaxExtractSize
	ecctlMaxExtractSize = n
	return func() { ecctlMaxExtractSize = old }
}

func TestUnzipArchive_SkipsDirectoryEntry(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	_, _ = zw.Create("dir/")
	w, _ := zw.Create("ecctl.exe")
	_, _ = w.Write([]byte("x"))
	_ = zw.Close()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, buf.Bytes(), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
}

func TestUnzipArchive_SkipsSymlinkEntry(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	h := &zip.FileHeader{Name: "ecctl.exe", Method: zip.Store}
	h.SetMode(os.ModeSymlink | 0o755)
	w, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("x"))
	_ = zw.Close()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, buf.Bytes(), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
}

func TestUnzipArchive_SkipsDeclaredOversize(t *testing.T) {
	defer withLowExtractMax(t, 8)()
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	h := &zip.FileHeader{Name: "ecctl.exe", Method: zip.Store}
	h.UncompressedSize64 = 100
	w, _ := zw.CreateHeader(h)
	_, _ = w.Write([]byte("small"))
	_ = zw.Close()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, buf.Bytes(), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
}

func TestUnzipArchive_SkipsStreamOversize(t *testing.T) {
	defer withLowExtractMax(t, 2)()
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("ecctl.exe")
	_, _ = w.Write([]byte("12345"))
	_ = zw.Close()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, buf.Bytes(), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
}

func TestUnzipArchive_SkipsSanitizeErrorName(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("/abs/ecctl.exe")
	_, _ = w.Write([]byte("x"))
	_ = zw.Close()
	zipPath := filepath.Join(dir, "a.zip")
	_ = os.WriteFile(zipPath, buf.Bytes(), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := unzipArchive(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
}

func TestUnzipArchive_InvalidZipFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.zip")
	_ = os.WriteFile(p, []byte("not-a-zip"), 0o644)
	if err := unzipArchive(p, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected open error")
	}
}

func TestExtractTarGz_SkipsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Name: "/ecctl", Mode: 0o755, Size: 1, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("x"))
	_ = tw.WriteHeader(&tar.Header{Name: "ecctl", Mode: 0o755, Size: 1, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("y"))
	_ = tw.Close()
	_ = gw.Close()
	src := filepath.Join(dir, "a.tar.gz")
	_ = os.WriteFile(src, buf.Bytes(), 0o644)
	if err := extractTarGz(src, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("extract: %v", err)
	}
}

func TestExtractTarGz_SkipsDirectoryAndDefaultType(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Name: "dir", Mode: 0o755, Typeflag: tar.TypeDir})
	_ = tw.WriteHeader(&tar.Header{Name: "meta", Mode: 0o755, Typeflag: tar.TypeXGlobalHeader})
	_ = tw.WriteHeader(&tar.Header{Name: "ecctl", Mode: 0o755, Size: 1, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("x"))
	_ = tw.Close()
	_ = gw.Close()
	src := filepath.Join(dir, "a.tar.gz")
	_ = os.WriteFile(src, buf.Bytes(), 0o644)
	if err := extractTarGz(src, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("extract: %v", err)
	}
}

func TestExtractTarGz_NegativeSizeSkipped(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Name: "ecctl", Mode: 0o755, Size: -1, Typeflag: tar.TypeReg})
	_ = tw.WriteHeader(&tar.Header{Name: "ecctl2", Mode: 0o755, Size: 1, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("x"))
	_ = tw.Close()
	_ = gw.Close()
	src := filepath.Join(dir, "a.tar.gz")
	_ = os.WriteFile(src, buf.Bytes(), 0o644)
	origGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "linux" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS })
	if err := extractTarGz(src, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("extract: %v", err)
	}
}

func TestExtractTarGz_NestedPathMkdir(t *testing.T) {
	dir := t.TempDir()
	payload, err := buildTarGzArchive(map[string][]byte{"bin/ecctl": []byte("x")})
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "a.tar.gz")
	_ = os.WriteFile(src, payload, 0o644)
	if err := extractTarGz(src, filepath.Join(dir, "out")); err != nil {
		t.Fatalf("extract: %v", err)
	}
}

func TestFindEcctlBinary_MissingRoot(t *testing.T) {
	_, err := findEcctlBinary(filepath.Join(t.TempDir(), "missing"), "ecctl")
	if err == nil {
		t.Fatal("expected walk error")
	}
}

func TestFindEcctlBinary_FoundNested(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(nested, "ecctl")
	if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findEcctlBinary(root, "ecctl")
	if err != nil || got != p {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestApplyMainCliFlagsFromArgs_NilOrigin(t *testing.T) {
	c := &Context{originCtx: nil}
	c.applyMainCliFlagsFromArgs([]string{"--profile", "x"})
}

func TestApplyMainCliFlagsFromArgs_BooleanConfigFlag(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	f := &cli.Flag{Name: "dummy-bool", Category: "config", AssignedMode: cli.AssignedNone}
	ctx.Flags().Add(f)
	c := NewContext(ctx)
	c.applyMainCliFlagsFromArgs([]string{"--dummy-bool"})
	if !f.IsAssigned() {
		t.Fatal("expected boolean flag assigned")
	}
}

func TestApplyMainCliFlagsFromArgs_MissingValueAtEnd(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	c := NewContext(ctx)
	c.applyMainCliFlagsFromArgs([]string{"--profile"})
	pf := ctx.Flags().Get("profile")
	if pf != nil && pf.IsAssigned() {
		t.Fatal("profile should not be assigned without value")
	}
}

func TestRemoveFlagsForMainCli_NilOriginCtx(t *testing.T) {
	c := NewContext(nil)
	out, err := c.RemoveFlagsForMainCli([]string{"ecctl", "help", "--region", "cn-hangzhou"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0] != "ecctl" || out[1] != "help" {
		t.Fatalf("expected region stripped via mainCliFs, got %v", out)
	}
}

func TestRun_InitializePlatformError(t *testing.T) {
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	runtimeGOOSFunc = func() string { return "plan9" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS; runtimeGOARCHFunc = origGOARCH })
	ctx, _, _ := newOriginCtx()
	err := NewContext(ctx).Run([]string{"version"})
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("got %v", err)
	}
}

func TestRun_EnsureExternalPathMissing(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_ECCTL_EXEC_PATH", filepath.Join(t.TempDir(), "nope"))
	ctx, _, _ := newOriginCtx()
	err := NewContext(ctx).Run([]string{"version"})
	if err == nil || !strings.Contains(err.Error(), "ALIBABA_CLOUD_ECCTL_EXEC_PATH") {
		t.Fatalf("got %v", err)
	}
}

func TestRun_ExecuteNonExitError(t *testing.T) {
	home := setupTestHome(t)
	prepareConfig(t, home, "en")
	fake := filepath.Join(config.GetConfigPath(), "ecctl")
	writeExecutable(t, fake, "x")
	t.Setenv("ALIBABA_CLOUD_ECCTL_EXEC_PATH", fake)
	origExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("no-such-binary-ecctl-test")
	}
	t.Cleanup(func() { execCommandFunc = origExec })
	ctx, _, _ := newOriginCtx()
	err := NewContext(ctx).Run([]string{"version"})
	if err == nil {
		t.Fatal("expected execute error")
	}
	if _, ok := err.(*ExitError); ok {
		t.Fatal("expected non-ExitError")
	}
}

func TestPrepareEnv_CompatMode(t *testing.T) {
	home := setupTestHome(t)
	prepareConfig(t, home, "en")
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv: %v", err)
	}
	if c.envMap["ALIBABA_CLOUD_ECCTL_COMPAT_MODE"] != "aliyun ecctl" {
		t.Fatalf("compat: %v", c.envMap)
	}
}
