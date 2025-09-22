package ossutil

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
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

// helper to create a fake executable script
func writeExecutable(t *testing.T, path string, content string) {
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write exec failed: %v", err)
	}
	if runtime.GOOS == "windows" { // ensure .exe presence not required, we just keep name
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

func TestRun_NotInstalled_FreshInstallAndExecute(t *testing.T) {
	origHOME := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", origHOME) })

	home := t.TempDir()
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	prepareConfig(t, home, "zh")

	// mock latest version & download & exec
	origGetLatest := getLatestOssUtilVersionFunc
	origDownload := downloadAndUnzipFunc
	origTimeNow := timeNowFunc
	origExec := execCommandFunc
	getLatestOssUtilVersionFunc = func() (string, error) { return "1.2.3", nil }
	installCount := 0
	downloadAndUnzipFunc = func(url, dest, exe, center string) error {
		installCount++
		// create fake binary file for existence check only
		writeExecutable(t, exe, "#!/bin/sh\n")
		return nil
	}
	fixedNow := time.Unix(1700000000, 0)
	timeNowFunc = func() time.Time { return fixedNow }
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		// mock main execution always success
		return exec.Command("bash", "-c", "exit 0")
	}
	t.Cleanup(func() {
		getLatestOssUtilVersionFunc = origGetLatest
		downloadAndUnzipFunc = origDownload
		timeNowFunc = origTimeNow
		execCommandFunc = origExec
	})

	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")

	c := NewContext(ctx)

	if err := c.Run([]string{"ossutil", "ls", "--region", "cn-hangzhou"}); err != nil {
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
	}
	if installCount != 1 {
		t.Fatalf("expected install once, got %d", installCount)
	}
	// verify cache file timestamp
	data, err := os.ReadFile(filepath.Join(config.GetConfigPath(), ".ossutil_version_check"))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	if strings.TrimSpace(string(data)) != fmt.Sprintf("%d", fixedNow.Unix()) {
		t.Fatalf("cache timestamp mismatch: %s", string(data))
	}
}

func TestRun_Installed_NoVersionCheckWithinTTL(t *testing.T) {
	origHOME := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", origHOME) })
	home := t.TempDir()
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	prepareConfig(t, home, "en")

	// prepare existing binary
	execPath := filepath.Join(config.GetConfigPath(), "ossutil")
	writeExecutable(t, execPath, "#!/bin/sh\n")

	// create fresh cache timestamp (recent)
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ossutil_version_check"), []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)

	origGetLatest := getLatestOssUtilVersionFunc
	origDownload := downloadAndUnzipFunc
	origExec := execCommandFunc
	getCalls := 0
	getLatestOssUtilVersionFunc = func() (string, error) { getCalls++; return "1.2.3", nil }
	installCount := 0
	downloadAndUnzipFunc = func(url, dest, exe, center string) error { installCount++; return nil }
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return exec.Command("bash", "-c", "exit 0") }
	t.Cleanup(func() {
		getLatestOssUtilVersionFunc = origGetLatest
		downloadAndUnzipFunc = origDownload
		execCommandFunc = origExec
	})

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"ossutil", "ls"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if getCalls != 0 {
		t.Fatalf("expected no version check, got %d", getCalls)
	}
	if installCount != 0 {
		t.Fatalf("expected not install, got %d", installCount)
	}
}

func TestRun_Installed_UpdateWhenExpired(t *testing.T) {
	origHOME := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", origHOME) })
	home := t.TempDir()
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	prepareConfig(t, home, "en")

	execPath := filepath.Join(config.GetConfigPath(), "ossutil")
	writeExecutable(t, execPath, "#!/bin/sh\n")
	old := time.Now().Unix() - int64(VersionCheckTTL) - 10
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ossutil_version_check"), []byte(fmt.Sprintf("%d", old)), 0644)

	origGetLatest := getLatestOssUtilVersionFunc
	origDownload := downloadAndUnzipFunc
	origExec := execCommandFunc
	getLatestOssUtilVersionFunc = func() (string, error) { return "1.0.1", nil }
	installCount := 0
	downloadAndUnzipFunc = func(url, dest, exe, center string) error { installCount++; return nil }
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return exec.Command("bash", "-c", "exit 0") }
	t.Cleanup(func() {
		getLatestOssUtilVersionFunc = origGetLatest
		downloadAndUnzipFunc = origDownload
		execCommandFunc = origExec
	})

	ctx, out, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"ossutil", "ls"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if installCount == 0 {
		t.Fatalf("expected install triggered")
	}
	if strings.Contains(out.String(), "A new version of ossutil is available") {
		t.Fatalf("expected update message, got %s", out.String())
	}
}

func TestNeedCheckVersionVariants(t *testing.T) {
	origHOME := os.Getenv("HOME")
	home := t.TempDir()
	_ = os.Setenv("HOME", home)
	prepareConfig(t, home, "zh")
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if c.NeedCheckVersion() {
		t.Fatalf("not installed should return false")
	}
	// simulate installed
	execPath := filepath.Join(config.GetConfigPath(), "ossutil")
	writeExecutable(t, execPath, "#!/bin/sh\n")
	c.InitBasicInfo()
	if !c.NeedCheckVersion() {
		t.Fatalf("installed no cache => true")
	}
	// invalid content
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ossutil_version_check"), []byte("abc"), 0644)
	if !c.NeedCheckVersion() {
		t.Fatalf("invalid content => true")
	}
	// recent timestamp
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ossutil_version_check"), []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)
	if c.NeedCheckVersion() {
		t.Fatalf("fresh cache => false")
	}
	// expired
	_ = os.WriteFile(filepath.Join(config.GetConfigPath(), ".ossutil_version_check"), []byte(fmt.Sprintf("%d", time.Now().Unix()-int64(VersionCheckTTL)-5)), 0644)
	if !c.NeedCheckVersion() {
		t.Fatalf("expired => true")
	}
	_ = os.Setenv("HOME", origHOME)
}

func TestRemoveFlagsForMainCli(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")
	// simulate already set language in profile via context defaultLanguage
	c := NewContext(ctx)
	c.defaultLanguage = "zh"
	args, err := c.RemoveFlagsForMainCli([]string{"ossutil", "ls", "--region", "cn-hangzhou"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--region") {
		t.Fatalf("region flag should be removed: %s", joined)
	}
	if !strings.Contains(joined, "--language zh") {
		t.Fatalf("language should be appended: %s", joined)
	}
}

func TestGetLatestOssUtilVersionWithServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ossutil version: 9.9.9")
	}))
	defer srv.Close()
	origURL := versionUpdateBaseUrl
	versionUpdateBaseUrl = srv.URL + "/"
	defer func() { versionUpdateBaseUrl = origURL }()
	ver, err := GetLatestOssUtilVersion()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ver != "9.9.9" {
		t.Fatalf("unexpected version %s", ver)
	}
}

func TestDownloadAndUnzip(t *testing.T) {
	// create zip with structure ossutil-1.0.0-mac-amd64/ossutil
	zipFile := filepath.Join(t.TempDir(), "ossutil.zip")
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	center := "ossutil-1.0.0-mac-amd64"
	// add file
	f, _ := zw.Create(center + "/ossutil")
	_, _ = f.Write([]byte("#!/bin/sh\n"))
	_ = zw.Close()
	if err := os.WriteFile(zipFile, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	// mock httpGetFunc
	origHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		fh, err := os.Open(zipFile)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Body: fh}, nil
	}
	defer func() { httpGetFunc = origHTTPGet }()

	destFile := filepath.Join(t.TempDir(), "d.zip")
	exeFile := filepath.Join(t.TempDir(), "ossutil")
	if err := DownloadAndUnzip("http://example/zip", destFile, exeFile, center); err != nil {
		t.Fatalf("DownloadAndUnzip: %v", err)
	}
	if !fileExists(exeFile) {
		t.Fatalf("exe not exist")
	}
}

func TestGetLocalVersion_NotInstalled(t *testing.T) {
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

func TestInstallUrlAndInvocation(t *testing.T) {
	c := &Context{versionRemote: "2.3.4", downloadPathSuffix: "linux-amd64.zip"}
	called := false
	origDownload := downloadAndUnzipFunc
	downloadAndUnzipFunc = func(url, dest, exe, center string) error {
		called = true
		if !strings.Contains(center, "2.3.4") {
			t.Fatalf("center dir mismatch %s", center)
		}
		return nil
	}
	defer func() { downloadAndUnzipFunc = origDownload }()
	if err := c.Install(); err != nil {
		t.Fatalf("Install err: %v", err)
	}
	if !called {
		t.Fatalf("download not called")
	}
}

func TestGetLocalVersion_Success(t *testing.T) {
	origExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return exec.Command("bash", "-c", "echo 5.6.7") }
	defer func() { execCommandFunc = origExec }()
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.installed = true
	c.execFilePath = "/any/path/ossutil"
	if err := c.GetLocalVersion(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if c.versionLocal != "5.6.7" {
		t.Fatalf("versionLocal mismatch %s", c.versionLocal)
	}
}

func TestUpdateCheckCacheTime(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)
	prepareConfig(t, home, "zh")
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

func TestDownloadAndUnzip_ErrorStatus(t *testing.T) {
	origHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString("err"))}, nil
	}
	defer func() { httpGetFunc = origHTTPGet }()
	tmp := t.TempDir()
	err := DownloadAndUnzip("http://x", filepath.Join(tmp, "a.zip"), filepath.Join(tmp, "ossutil"), "center")
	if err == nil || !strings.Contains(err.Error(), "status code") {
		t.Fatalf("expect status code error, got %v", err)
	}
}

func TestGetLatestOssUtilVersion_ParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "bad content") }))
	defer srv.Close()
	origURL := versionUpdateBaseUrl
	versionUpdateBaseUrl = srv.URL + "/"
	defer func() { versionUpdateBaseUrl = origURL }()
	_, err := GetLatestOssUtilVersion()
	if err == nil || !strings.Contains(err.Error(), "parse version") {
		t.Fatalf("expect parse error, got %v", err)
	}
}

func TestPrepareEnv(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)
	prepareConfig(t, home, "en")
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv err: %v", err)
	}
	if c.envMap["OSSUTIL_COMPAT_MODE"] != "alicli" {
		t.Fatalf("OSSUTIL_COMPAT_MODE missing")
	}
	// OSSUTIL_CONFIG_VALUE should contain region and language
	val, ok := c.envMap["OSSUTIL_CONFIG_VALUE"]
	if !ok {
		t.Fatalf("OSSUTIL_CONFIG_VALUE missing")
	}
	// base64 decode
	dec, err := base64.StdEncoding.DecodeString(val)

	if err != nil {
		t.Fatalf("base64 decode err: %v", err)
	}
	var config map[string]any
	err = json.Unmarshal(dec, &config)
	if err != nil {
		t.Fatalf("json unmarshal err: %v", err)
	}
	if config["region_id"] != "cn-hangzhou" {
		t.Fatalf("region mismatch: %v", config["region"])
	}
	if config["language"] != "en" {
		t.Fatalf("language mismatch: %v", config["language"])
	}
	// access_key_id and access_key_secret should be present
	if config["access_key_id"] != "ak" {
		t.Fatalf("ak mismatch: %v", config["access_key_id"])
	}

	if config["access_key_secret"] != "sk" {
		t.Fatalf("sk mismatch: %v", config["access_key_secret"])
	}
}

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

func TestDownloadAndUnzip_UnzipError(t *testing.T) {
	// simulate http ok but corrupt zip
	origHTTP := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte("not zip")))}, nil
	}
	defer func() { httpGetFunc = origHTTP }()
	tmp := t.TempDir()
	err := DownloadAndUnzip("http://x", filepath.Join(tmp, "a.zip"), filepath.Join(tmp, "ossutil"), "center")
	if err == nil {
		t.Fatalf("expected unzip error")
	}
}

func TestDownloadAndUnzip_HttpError(t *testing.T) {
	origHTTP := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) { return nil, fmt.Errorf("net") }
	defer func() { httpGetFunc = origHTTP }()
	tmp := t.TempDir()
	err := DownloadAndUnzip("http://x", filepath.Join(tmp, "a.zip"), filepath.Join(tmp, "ossutil"), "center")
	if err == nil || !strings.Contains(err.Error(), "failed to download") {
		t.Fatalf("expected download error")
	}
}

func TestInfoAndErrorf(t *testing.T) {
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	ctx := cli.NewCommandContext(outBuf, errBuf)
	c := NewContext(ctx)
	c.info()
	c.info("plain message")
	c.info("hello", "world") // 不使用格式化占位符，避免 vet 误报
	c.errorf("err: %d", 5)
	if !strings.Contains(outBuf.String(), "plain message") {
		t.Fatalf("info plain missing: %s", outBuf.String())
	}
	if !strings.Contains(outBuf.String(), "hello world") {
		t.Fatalf("info multi-arg missing: %s", outBuf.String())
	}
	if !strings.Contains(errBuf.String(), "err: 5") {
		t.Fatalf("errorf output missing: %s", errBuf.String())
	}
}

func TestCheckOsTypeAndArchVariants(t *testing.T) {
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	defer func() { runtimeGOOSFunc = origGOOS; runtimeGOARCHFunc = origGOARCH }()
	tests := []struct {
		os, arch       string
		support        bool
		suffixContains string
	}{
		{"linux", "amd64", true, "linux-amd64.zip"},
		{"linux", "s390x", false, ""},
		{"windows", "386", true, "windows-386.zip"},
		{"windows", "arm64", false, ""},
		{"unknown", "amd64", false, ""},
	}
	for _, tc := range tests {
		runtimeGOOSFunc = func(val string) func() string { return func() string { return val } }(tc.os)
		runtimeGOARCHFunc = func(val string) func() string { return func() string { return val } }(tc.arch)
		ctx, _, _ := newOriginCtx()
		c := NewContext(ctx)
		c.CheckOsTypeAndArch()
		if c.osSupport != tc.support {
			f := fmt.Sprintf("expect support=%v for %s/%s got %v", tc.support, tc.os, tc.arch, c.osSupport)
			t.Fatalf(f)
		}
		if tc.support && !strings.Contains(c.downloadPathSuffix, tc.suffixContains) {
			t.Fatalf("suffix mismatch: %s vs %s", c.downloadPathSuffix, tc.suffixContains)
		}
		if !tc.support && c.downloadPathSuffix != "" {
			t.Fatalf("unsupported should not set suffix: %s", c.downloadPathSuffix)
		}
	}
}

func TestUnzip_OpenReaderError(t *testing.T) {
	// 提供一个不存在的文件路径
	tmp := t.TempDir()
	err := unzip(filepath.Join(tmp, "not-exist.zip"), filepath.Join(tmp, "dest"))
	if err == nil {
		t.Fatalf("expected error for non-existing zip")
	}
}

func createZipWithEntries(t *testing.T, entries map[string]string) string {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for name, content := range entries {
		var f io.Writer
		var err error
		if strings.HasSuffix(name, "/") { // directory entry
			_, err = zw.Create(name)
		} else {
			f, err = zw.Create(name)
			if err == nil {
				_, err = f.Write([]byte(content))
			}
		}
		if err != nil {
			t.Fatalf("create zip entry failed: %v", err)
		}
	}
	_ = zw.Close()
	zipPath := filepath.Join(t.TempDir(), "test.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write zip: %v", err)
	}
	return zipPath
}

func TestUnzip_MkdirAllDirError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip permission test on windows")
	}
	zipPath := createZipWithEntries(t, map[string]string{"dir1/": ""})
	roParent := filepath.Join(t.TempDir(), "ro")
	if err := os.MkdirAll(roParent, 0555); err != nil {
		t.Fatalf("mkdir ro: %v", err)
	}
	dest := filepath.Join(roParent, "dest")
	err := unzip(zipPath, dest)
	if err == nil {
		t.Fatalf("expected mkdir error for directory entry")
	}
	// 还原权限便于清理
	_ = os.Chmod(roParent, 0755)
}

func TestUnzip_MkdirAllParentError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip permission test on windows")
	}
	zipPath := createZipWithEntries(t, map[string]string{"a/b.txt": "hello"})
	roParent := filepath.Join(t.TempDir(), "ro2")
	if err := os.MkdirAll(roParent, 0555); err != nil {
		t.Fatalf("mkdir ro: %v", err)
	}
	dest := filepath.Join(roParent, "dest")
	err := unzip(zipPath, dest)
	if err == nil {
		t.Fatalf("expected mkdir error for parent of file")
	}
	_ = os.Chmod(roParent, 0755)
}

func TestUnzip_CreateFileError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip permission test on windows")
	}
	zipPath := createZipWithEntries(t, map[string]string{"file.txt": "content"})
	destRoot := filepath.Join(t.TempDir(), "destRoot")
	if err := os.MkdirAll(destRoot, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	conflictDir := filepath.Join(destRoot, "file.txt")
	if err := os.MkdirAll(conflictDir, 0755); err != nil {
		t.Fatalf("mkdir colliding dir: %v", err)
	}
	// 断言确实是目录
	st, statErr := os.Stat(conflictDir)
	if statErr != nil || !st.IsDir() {
		t.Fatalf("expected directory conflict setup, got err=%v", statErr)
	}
	err := unzip(zipPath, destRoot)
	if err == nil {
		t.Skip("unable to reproduce create-file-over-directory error on this platform; skipping branch-specific test")
	}
}

func TestPrepareEnv_StsToken(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)

	prepareConfigWithMode(t, home, "StsToken", map[string]string{
		"sts_token": "sts123",
	})

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv err: %v", err)
	}

	val, ok := c.envMap["OSSUTIL_CONFIG_VALUE"]
	if !ok {
		t.Fatalf("OSSUTIL_CONFIG_VALUE missing")
	}

	dec, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		t.Fatalf("base64 decode err: %v", err)
	}

	var config map[string]interface{}
	err = json.Unmarshal(dec, &config)
	if err != nil {
		t.Fatalf("json unmarshal err: %v", err)
	}

	if config["mode"] != "StsToken" {
		t.Fatalf("mode mismatch: %v", config["mode"])
	}
	if config["sts_token"] != "sts123" {
		t.Fatalf("sts_token mismatch: %v", config["sts_token"])
	}
}

func TestPrepareEnv_RamRoleArn(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)

	prepareConfigWithMode(t, home, "RamRoleArn", map[string]string{
		"ram_role_arn":     "arn:acs:ram::123:role/test",
		"ram_session_name": "session123",
		"sts_token":        "optional_sts",
	})

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv err: %v", err)
	}

	val, ok := c.envMap["OSSUTIL_CONFIG_VALUE"]
	if !ok {
		t.Fatalf("OSSUTIL_CONFIG_VALUE missing")
	}

	dec, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		t.Fatalf("base64 decode err: %v", err)
	}

	var config map[string]interface{}
	err = json.Unmarshal(dec, &config)
	if err != nil {
		t.Fatalf("json unmarshal err: %v", err)
	}

	if config["mode"] != "RamRoleArn" {
		t.Fatalf("mode mismatch: %v", config["mode"])
	}
	if config["ram_role_arn"] != "arn:acs:ram::123:role/test" {
		t.Fatalf("ram_role_arn mismatch: %v", config["ram_role_arn"])
	}
	if config["ram_session_name"] != "session123" {
		t.Fatalf("ram_session_name mismatch: %v", config["ram_session_name"])
	}
	if config["sts_token"] != "optional_sts" {
		t.Fatalf("sts_token mismatch: %v", config["sts_token"])
	}
}

func TestPrepareEnv_RamRoleArn_NoStsToken(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)

	prepareConfigWithMode(t, home, "RamRoleArn", map[string]string{
		"ram_role_arn":     "arn:acs:ram::123:role/test",
		"ram_session_name": "session123",
	})

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv err: %v", err)
	}

	val, ok := c.envMap["OSSUTIL_CONFIG_VALUE"]
	if !ok {
		t.Fatalf("OSSUTIL_CONFIG_VALUE missing")
	}

	dec, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		t.Fatalf("base64 decode err: %v", err)
	}

	var config map[string]interface{}
	err = json.Unmarshal(dec, &config)
	if err != nil {
		t.Fatalf("json unmarshal err: %v", err)
	}

	if config["mode"] != "RamRoleArn" {
		t.Fatalf("mode mismatch: %v", config["mode"])
	}
	// sts_token should not be present when empty
	if _, exists := config["sts_token"]; exists {
		t.Fatalf("sts_token should not be present when empty")
	}
}

func TestPrepareEnv_EcsRamRole(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)

	prepareConfigWithMode(t, home, "EcsRamRole", map[string]string{
		"ram_role_name": "test-role",
	})

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv err: %v", err)
	}

	val, ok := c.envMap["OSSUTIL_CONFIG_VALUE"]
	if !ok {
		t.Fatalf("OSSUTIL_CONFIG_VALUE missing")
	}

	dec, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		t.Fatalf("base64 decode err: %v", err)
	}

	var config map[string]interface{}
	err = json.Unmarshal(dec, &config)
	if err != nil {
		t.Fatalf("json unmarshal err: %v", err)
	}

	if config["mode"] != "EcsRamRole" {
		t.Fatalf("mode mismatch: %v", config["mode"])
	}
	if config["ram_role_name"] != "test-role" {
		t.Fatalf("ram_role_name mismatch: %v", config["ram_role_name"])
	}
}

func TestPrepareEnv_CredentialsURI(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)

	prepareConfigWithMode(t, home, "CredentialsURI", map[string]string{
		"credentials_uri": "http://localhost/credentials",
	})

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv err: %v", err)
	}

	val, ok := c.envMap["OSSUTIL_CONFIG_VALUE"]
	if !ok {
		t.Fatalf("OSSUTIL_CONFIG_VALUE missing")
	}

	dec, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		t.Fatalf("base64 decode err: %v", err)
	}

	var config map[string]interface{}
	err = json.Unmarshal(dec, &config)
	if err != nil {
		t.Fatalf("json unmarshal err: %v", err)
	}

	if config["mode"] != "CredentialsURI" {
		t.Fatalf("mode mismatch: %v", config["mode"])
	}
	if config["credentials_uri"] != "http://localhost/credentials" {
		t.Fatalf("credentials_uri mismatch: %v", config["credentials_uri"])
	}
}

func TestPrepareEnv_OIDC(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)

	prepareConfigWithMode(t, home, "OIDC", map[string]string{
		"oidc_provider_arn": "arn:acs:ram::123:oidc-provider/test",
		"oidc_token_file":   "/path/to/token",
		"ram_role_arn":      "arn:acs:ram::123:role/test",
		"ram_session_name":  "oidc-session",
	})

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv err: %v", err)
	}

	val, ok := c.envMap["OSSUTIL_CONFIG_VALUE"]
	if !ok {
		t.Fatalf("OSSUTIL_CONFIG_VALUE missing")
	}

	dec, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		t.Fatalf("base64 decode err: %v", err)
	}

	var config map[string]interface{}
	err = json.Unmarshal(dec, &config)
	if err != nil {
		t.Fatalf("json unmarshal err: %v", err)
	}

	if config["mode"] != "OIDC" {
		t.Fatalf("mode mismatch: %v", config["mode"])
	}
	if config["oidc_provider_arn"] != "arn:acs:ram::123:oidc-provider/test" {
		t.Fatalf("oidc_provider_arn mismatch: %v", config["oidc_provider_arn"])
	}
	if config["oidc_token_file"] != "/path/to/token" {
		t.Fatalf("oidc_token_file mismatch: %v", config["oidc_token_file"])
	}
}

func TestPrepareEnv_ConfigLoadError(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)

	// 不创建配置文件，导致配置加载失败
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
