package binmgr

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

// Test config that mirrors the acr-skill subcommand wiring.
func testConfig() Config {
	return Config{
		Name:             "test-bin",
		BaseURL:          "https://example.invalid/test-bin/",
		EnvCompatMode:    "TEST_BIN_COMPAT_MODE",
		EnvCompatModeVal: "aliyun acrutil test",
		EnvUserAgent:     "TEST_BIN_USER_AGENT",
		PlatformPaths: map[string]struct{}{
			"linux-amd64":  {},
			"darwin-arm64": {},
		},
		StripFlags: map[string]bool{
			"profile":     true,
			"mode":        true,
			"config-path": true,
		},
	}
}

func newOriginCtx() (*cli.Context, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, errOut)
	return ctx, out, errOut
}

func prepareConfigHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".aliyun")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	cfgJSON := `{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou","language":"en"}]}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfgJSON), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func addConfigFlag(ctx *cli.Context, name string, value string) {
	f := &cli.Flag{Name: name, AssignedMode: cli.AssignedOnce, Category: "config"}
	f.SetAssigned(true)
	f.SetValue(value)
	ctx.Flags().Add(f)
}

func TestNew(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	if m == nil {
		t.Fatal("New returned nil")
	}
	if m.cfg.Name != "test-bin" {
		t.Errorf("cfg.Name not propagated: %s", m.cfg.Name)
	}
	if m.originCtx != ctx {
		t.Errorf("originCtx not propagated")
	}
}

func TestInitBasicInfo(t *testing.T) {
	tmpDir := t.TempDir()

	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.getConfigurePathFn = func() string { return tmpDir }
	m.InitBasicInfo()

	if m.configPath != tmpDir {
		t.Errorf("configPath: got %s", m.configPath)
	}
	if m.execFilePath != filepath.Join(tmpDir, "test-bin") {
		t.Errorf("execFilePath: got %s", m.execFilePath)
	}
	if m.versionFilePath != filepath.Join(tmpDir, ".test-bin_version") {
		t.Errorf("versionFilePath: got %s", m.versionFilePath)
	}
	if m.checkVersionCacheFilePath != filepath.Join(tmpDir, ".test-bin_version_check") {
		t.Errorf("checkVersionCacheFilePath: got %s", m.checkVersionCacheFilePath)
	}
	if m.installed {
		t.Errorf("should not be installed initially")
	}

	if err := os.WriteFile(m.execFilePath, []byte("fake"), 0755); err != nil {
		t.Fatalf("write fake exec: %v", err)
	}
	m2 := New(testConfig(), ctx)
	m2.getConfigurePathFn = func() string { return tmpDir }
	m2.InitBasicInfo()
	if !m2.installed {
		t.Errorf("should be installed when exec file present")
	}
}

func TestCheckOsTypeAndArch(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)

	tests := []struct {
		osType, osArch string
		want           bool
	}{
		{"linux", "amd64", true},
		{"darwin", "arm64", true},
		{"linux", "arm64", false},
		{"darwin", "amd64", false},
		{"windows", "amd64", false},
	}
	for _, tt := range tests {
		m.runtimeGOOSFn = func() string { return tt.osType }
		m.runtimeGOARCHFn = func() string { return tt.osArch }
		m.osSupport = false
		m.CheckOsTypeAndArch()
		if m.osSupport != tt.want {
			t.Errorf("%s-%s: want support=%v, got %v", tt.osType, tt.osArch, tt.want, m.osSupport)
		}
	}
}

func TestGetDownloadURL(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	cases := []struct {
		name      string
		platform  string
		version   string
		expectErr bool
		expectURL string
	}{
		{"linux-amd64 ok", "linux-amd64", "v1.2.0", false,
			"https://example.invalid/test-bin/v1.2.0/test-bin-linux-amd64"},
		{"darwin-arm64 ok", "darwin-arm64", "v1.0.0", false,
			"https://example.invalid/test-bin/v1.0.0/test-bin-darwin-arm64"},
		{"unsupported platform", "linux-arm64", "v1.0.0", true, ""},
		{"empty version rejected", "linux-amd64", "", true, ""},
		{"unknown platform", "weird-thing", "v1.0.0", true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := New(testConfig(), ctx)
			m.downloadPathSuffix = tc.platform
			m.versionRemote = tc.version
			got, err := m.GetDownloadURL()
			if tc.expectErr {
				if err == nil {
					t.Errorf("want error, got url=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expectURL {
				t.Errorf("url: want %q, got %q", tc.expectURL, got)
			}
		})
	}
}

func TestDownloadBinary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake binary content"))
	}))
	defer server.Close()

	oldClient := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = oldClient })

	tmpDir := t.TempDir()
	exe := filepath.Join(tmpDir, "test-bin")

	if err := DownloadBinary(context.Background(), server.URL, exe); err != nil {
		t.Fatalf("DownloadBinary: %v", err)
	}
	if !fileExists(exe) {
		t.Fatalf("exec not created")
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "fake binary content" {
		t.Errorf("content: got %q", string(got))
	}
}

func TestDownloadBinary_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	oldClient := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = oldClient })

	tmp := t.TempDir()
	err := DownloadBinary(context.Background(), server.URL, filepath.Join(tmp, "test-bin"))
	if err == nil || !strings.Contains(err.Error(), "status code") {
		t.Fatalf("want status code error, got %v", err)
	}
}

func TestDownloadBinary_HttpError(t *testing.T) {
	tmp := t.TempDir()
	err := DownloadBinary(context.Background(),
		"http://invalid-host-that-does-not-exist-12345.com",
		filepath.Join(tmp, "test-bin"))
	if err == nil || !strings.Contains(err.Error(), "failed to download") {
		t.Fatalf("want download error, got %v", err)
	}
}

func TestDownloadBinary_OverwriteExisting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("new binary"))
	}))
	defer server.Close()

	oldClient := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = oldClient })

	tmpDir := t.TempDir()
	exe := filepath.Join(tmpDir, "test-bin")
	if err := os.WriteFile(exe, []byte("old binary"), 0755); err != nil {
		t.Fatalf("write old: %v", err)
	}
	if err := DownloadBinary(context.Background(), server.URL, exe); err != nil {
		t.Fatalf("DownloadBinary: %v", err)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "new binary" {
		t.Errorf("not overwritten: %q", string(got))
	}
}

func TestDownloadBinary_ContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	oldClient := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = oldClient })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmp := t.TempDir()
	err := DownloadBinary(ctx, server.URL, filepath.Join(tmp, "test-bin"))
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestRemoveFlagsForMainCli(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "profile", "test")
	m := New(testConfig(), ctx)
	args := []string{"publish", "--profile", "default", "-d", "./my"}
	out, err := m.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for _, a := range out {
		if a == "--profile" {
			t.Errorf("--profile should be stripped: %v", out)
		}
	}
	hasD := false
	for _, a := range out {
		if a == "-d" {
			hasD = true
		}
	}
	if !hasD {
		t.Errorf("-d should be retained: %v", out)
	}
}

func TestRemoveFlagsForMainCli_StripFlags(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "profile", "test")
	addConfigFlag(ctx, "mode", "AK")
	addConfigFlag(ctx, "region", "cn-hangzhou")
	m := New(testConfig(), ctx)
	args := []string{
		"--profile", "test",
		"--mode", "AK",
		"-d", "./my",
		"--version",
	}
	out, err := m.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for _, a := range out {
		if a == "--profile" || a == "--mode" {
			t.Errorf("flag should be stripped: %s", a)
		}
	}
	want := []string{"-d", "--version"}
	for _, w := range want {
		found := false
		for _, a := range out {
			if a == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("flag should remain: %s in %v", w, out)
		}
	}
}

func TestRemoveFlagsForMainCli_InlineValue(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "profile", "test")
	m := New(testConfig(), ctx)
	args := []string{"publish", "--profile=test", "-d", "./my"}
	out, err := m.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for _, a := range out {
		if strings.HasPrefix(a, "--profile") {
			t.Errorf("inline flag should be stripped: %s", a)
		}
	}
	if len(out) < 2 || out[0] != "publish" {
		t.Errorf("expected first arg publish, got %v", out)
	}
}

func TestFileExists(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "f")
	if fileExists(p) {
		t.Errorf("should not exist")
	}
	if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if !fileExists(p) {
		t.Errorf("should exist")
	}
}

func TestSaveLocalVersion(t *testing.T) {
	tmp := t.TempDir()
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.versionFilePath = filepath.Join(tmp, ".test-bin_version")
	m.versionLocal = "v1.5.0"
	if err := m.SaveLocalVersion(); err != nil {
		t.Fatalf("SaveLocalVersion: %v", err)
	}
	got, err := os.ReadFile(m.versionFilePath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "v1.5.0\n" {
		t.Errorf("content: %q", string(got))
	}
}

func TestGetLocalVersion(t *testing.T) {
	tmp := t.TempDir()
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.versionFilePath = filepath.Join(tmp, ".test-bin_version")
	m.execFilePath = filepath.Join(tmp, "test-bin")
	m.installed = true

	// File missing → no error, empty version.
	if err := m.GetLocalVersion(); err != nil {
		t.Errorf("missing file should be nil error, got %v", err)
	}
	if m.versionLocal != "" {
		t.Errorf("expected empty, got %q", m.versionLocal)
	}

	// File present.
	if err := os.WriteFile(m.versionFilePath, []byte("v1.3.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := m.GetLocalVersion(); err != nil {
		t.Fatalf("read: %v", err)
	}
	if m.versionLocal != "v1.3.0" {
		t.Errorf("expected v1.3.0, got %q", m.versionLocal)
	}

	// Empty file → error, resets versionLocal.
	m.versionLocal = "leftover"
	if err := os.WriteFile(m.versionFilePath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := m.GetLocalVersion(); err == nil {
		t.Errorf("expected error for empty file")
	}
	if m.versionLocal != "" {
		t.Errorf("versionLocal should be cleared on empty file, got %q", m.versionLocal)
	}

	// Not installed → error.
	m.installed = false
	m.versionLocal = "leftover"
	if err := m.GetLocalVersion(); err == nil {
		t.Errorf("expected error when not installed")
	}
	if m.versionLocal != "" {
		t.Errorf("versionLocal should be cleared when not installed, got %q", m.versionLocal)
	}
}

func TestUpdateCheckCacheTime(t *testing.T) {
	tmp := t.TempDir()
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.checkVersionCacheFilePath = filepath.Join(tmp, ".test-bin_version_check")
	m.timeNowFn = func() time.Time { return time.Unix(1234567890, 0) }

	if err := m.UpdateCheckCacheTime(); err != nil {
		t.Fatalf("err: %v", err)
	}
	got, err := os.ReadFile(m.checkVersionCacheFilePath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "1234567890" {
		t.Errorf("content: %q", string(got))
	}
}

func TestNeedCheckVersion(t *testing.T) {
	tmp := t.TempDir()
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.checkVersionCacheFilePath = filepath.Join(tmp, ".test-bin_version_check")
	m.installed = false

	// not installed → false
	if m.NeedCheckVersion() {
		t.Errorf("not installed should return false")
	}
	m.installed = true

	// no cache file → true
	if !m.NeedCheckVersion() {
		t.Errorf("missing cache should return true")
	}

	now := time.Now().Unix()
	mustWrite := func(content string) {
		if err := os.WriteFile(m.checkVersionCacheFilePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// recent → false
	mustWrite(fmt.Sprintf("%d", now))
	if m.NeedCheckVersion() {
		t.Errorf("recent cache should return false")
	}

	// expired → true
	mustWrite(fmt.Sprintf("%d", now-int64(VersionCheckTTL.Seconds())-100))
	if !m.NeedCheckVersion() {
		t.Errorf("expired cache should return true")
	}

	// invalid → true
	mustWrite("not-a-number")
	if !m.NeedCheckVersion() {
		t.Errorf("invalid cache should return true")
	}

	// partial parse (Sscanf would have accepted "123abc" silently;
	// strconv.ParseInt does not, so we want a re-check)
	mustWrite("123abc")
	if !m.NeedCheckVersion() {
		t.Errorf("partial-parse content should return true")
	}

	// negative timestamp → true
	mustWrite("-1")
	if !m.NeedCheckVersion() {
		t.Errorf("negative timestamp should return true")
	}

	// future timestamp → true (clock skew protection)
	mustWrite("2000000000") // year 2033
	m.timeNowFn = func() time.Time { return time.Unix(1700000000, 0) }
	if !m.NeedCheckVersion() {
		t.Errorf("future cache timestamp should force re-check (clock-skew protection)")
	}

	// mock-time exactly past TTL boundary
	mustWrite("1000")
	m.timeNowFn = func() time.Time { return time.Unix(1000+int64(VersionCheckTTL.Seconds())+1, 0) }
	if !m.NeedCheckVersion() {
		t.Errorf("past TTL should return true")
	}
	m.timeNowFn = func() time.Time { return time.Unix(1000+int64(VersionCheckTTL.Seconds())-1, 0) }
	if m.NeedCheckVersion() {
		t.Errorf("within TTL should return false")
	}
}

func TestGetLatestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/latest/version.txt") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, "aliyun-cli/") {
			t.Errorf("User-Agent: got %q", ua)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("v1.2.0\n"))
	}))
	defer server.Close()

	oldDo := httpDoFunc
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		newReq, _ := http.NewRequestWithContext(req.Context(), "GET", server.URL+req.URL.Path, nil)
		newReq.Header = req.Header
		return server.Client().Do(newReq)
	}
	t.Cleanup(func() { httpDoFunc = oldDo })

	v, err := GetLatestVersion(context.Background(), "https://example.invalid/test-bin/")
	if err != nil {
		t.Fatalf("GetLatestVersion: %v", err)
	}
	if v != "v1.2.0" {
		t.Errorf("version: got %q", v)
	}
}

func TestGetLatestVersion_ErrorStatus(t *testing.T) {
	oldDo := httpDoFunc
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, nil
	}
	t.Cleanup(func() { httpDoFunc = oldDo })

	if _, err := GetLatestVersion(context.Background(), "https://example.invalid/test-bin/"); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetLatestVersion_EmptyBody(t *testing.T) {
	oldDo := httpDoFunc
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		}, nil
	}
	t.Cleanup(func() { httpDoFunc = oldDo })
	if _, err := GetLatestVersion(context.Background(), "https://example.invalid/test-bin/"); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestInstallUrlAndInvocation(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.downloadPathSuffix = "linux-amd64"
	m.execFilePath = filepath.Join(tmpDir, "test-bin")
	m.versionFilePath = filepath.Join(tmpDir, ".test-bin_version")
	m.versionRemote = "v1.2.0"

	called := false
	m.downloadBinaryFn = func(ctx context.Context, url, exe string) error {
		called = true
		if !strings.Contains(url, "v1.2.0/test-bin-linux-amd64") {
			t.Errorf("url: got %s", url)
		}
		return os.WriteFile(exe, []byte("x"), 0755)
	}

	if err := m.Install(context.Background()); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !called {
		t.Error("download not called")
	}
	if !m.installed {
		t.Error("installed should be true")
	}
	if m.versionLocal != "v1.2.0" {
		t.Errorf("versionLocal: got %q", m.versionLocal)
	}
	data, err := os.ReadFile(m.versionFilePath)
	if err != nil {
		t.Fatalf("read version file: %v", err)
	}
	if string(data) != "v1.2.0\n" {
		t.Errorf("version file content: got %q", string(data))
	}
}

func TestInstall_EmptyVersionRejected(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.downloadPathSuffix = "linux-amd64"
	m.execFilePath = filepath.Join(tmpDir, "test-bin")
	m.versionRemote = "" // <-- the guarded case

	called := false
	m.downloadBinaryFn = func(ctx context.Context, url, exe string) error {
		called = true
		return nil
	}

	err := m.Install(context.Background())
	if err == nil {
		t.Fatal("expected error for empty versionRemote")
	}
	if called {
		t.Error("downloadBinaryFn should not be called when version is empty")
	}
}

func TestExecute(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.execFilePath = "/no/op"
	m.execCommandFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if runtime.GOOS == "windows" {
			return exec.CommandContext(ctx, "cmd", "/c", "exit 0")
		}
		return exec.CommandContext(ctx, "true")
	}
	if err := m.Execute(context.Background(), nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestExecute_Failure(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.execFilePath = "/no/op"
	m.execCommandFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if runtime.GOOS == "windows" {
			return exec.CommandContext(ctx, "cmd", "/c", "exit 1")
		}
		return exec.CommandContext(ctx, "false")
	}
	if err := m.Execute(context.Background(), nil); err == nil {
		t.Fatal("expected failure")
	}
}

func TestEnsureInstalledAndUpdated_NotInstalled(t *testing.T) {
	tmp := t.TempDir()
	ctx, _, stderr := newOriginCtx()
	m := New(testConfig(), ctx)
	m.downloadPathSuffix = "linux-amd64"
	m.execFilePath = filepath.Join(tmp, "test-bin")
	m.versionFilePath = filepath.Join(tmp, ".test-bin_version")
	m.checkVersionCacheFilePath = filepath.Join(tmp, ".test-bin_version_check")
	m.installed = false

	m.downloadBinaryFn = func(ctx context.Context, url, exe string) error {
		if !strings.Contains(url, "v2.0.0/test-bin-linux-amd64") {
			t.Errorf("url: got %s", url)
		}
		return os.WriteFile(exe, []byte("bin"), 0755)
	}
	m.getLatestVersionFn = func(ctx context.Context, base string) (string, error) { return "v2.0.0", nil }

	if err := m.EnsureInstalledAndUpdated(context.Background()); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !m.installed {
		t.Error("installed should be true")
	}
	if m.versionLocal != "v2.0.0" {
		t.Errorf("versionLocal: %q", m.versionLocal)
	}
	if !fileExists(m.checkVersionCacheFilePath) {
		t.Error("cache time file should exist")
	}
	if stderr.Len() != 0 {
		t.Errorf("unexpected stderr: %s", stderr.String())
	}
}

func TestEnsureInstalledAndUpdated_NotInstalled_GetLatestFails(t *testing.T) {
	tmp := t.TempDir()
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.downloadPathSuffix = "linux-amd64"
	m.execFilePath = filepath.Join(tmp, "test-bin")
	m.versionFilePath = filepath.Join(tmp, ".test-bin_version")
	m.checkVersionCacheFilePath = filepath.Join(tmp, ".test-bin_version_check")
	m.installed = false

	m.getLatestVersionFn = func(ctx context.Context, base string) (string, error) {
		return "", fmt.Errorf("network down")
	}

	err := m.EnsureInstalledAndUpdated(context.Background())
	if err == nil {
		t.Fatal("expected error when not installed and download fails")
	}
	if !strings.Contains(err.Error(), "auto-download failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEnsureInstalledAndUpdated_NoUpdate(t *testing.T) {
	tmp := t.TempDir()
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.downloadPathSuffix = "linux-amd64"
	m.execFilePath = filepath.Join(tmp, "test-bin")
	m.versionFilePath = filepath.Join(tmp, ".test-bin_version")
	m.checkVersionCacheFilePath = filepath.Join(tmp, ".test-bin_version_check")
	m.installed = true

	if err := os.WriteFile(m.versionFilePath, []byte("v1.0.0"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.checkVersionCacheFilePath, []byte("1000"), 0644); err != nil {
		t.Fatal(err)
	}

	downloads := 0
	m.downloadBinaryFn = func(ctx context.Context, url, exe string) error { downloads++; return nil }
	m.getLatestVersionFn = func(ctx context.Context, base string) (string, error) { return "v1.0.0", nil }
	m.timeNowFn = func() time.Time { return time.Unix(999999999, 0) }

	if err := m.EnsureInstalledAndUpdated(context.Background()); err != nil {
		t.Fatalf("err: %v", err)
	}
	if downloads != 0 {
		t.Errorf("no download expected, got %d", downloads)
	}
}

func TestEnsureInstalledAndUpdated_WithUpdate(t *testing.T) {
	tmp := t.TempDir()
	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.downloadPathSuffix = "linux-amd64"
	m.execFilePath = filepath.Join(tmp, "test-bin")
	m.versionFilePath = filepath.Join(tmp, ".test-bin_version")
	m.checkVersionCacheFilePath = filepath.Join(tmp, ".test-bin_version_check")
	m.installed = true

	if err := os.WriteFile(m.versionFilePath, []byte("v1.0.0"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.checkVersionCacheFilePath, []byte("1000"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.execFilePath, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}

	downloads := 0
	m.downloadBinaryFn = func(ctx context.Context, url, exe string) error {
		downloads++
		if !strings.Contains(url, "v2.0.0/test-bin-linux-amd64") {
			t.Errorf("url: got %s", url)
		}
		return os.WriteFile(exe, []byte("new"), 0755)
	}
	m.getLatestVersionFn = func(ctx context.Context, base string) (string, error) { return "v2.0.0", nil }
	m.timeNowFn = func() time.Time { return time.Unix(999999999, 0) }

	if err := m.EnsureInstalledAndUpdated(context.Background()); err != nil {
		t.Fatalf("err: %v", err)
	}
	if downloads != 1 {
		t.Errorf("expected 1 download, got %d", downloads)
	}
	if m.versionLocal != "v2.0.0" {
		t.Errorf("versionLocal: %q", m.versionLocal)
	}
}

// Regression test for the original thundering-herd bug: when the version
// check or update download fails, the cache timestamp MUST still be stamped
// so the next CLI invocation respects the TTL instead of re-attempting
// every time.
func TestEnsureInstalledAndUpdated_StampsCacheOnGetLatestFail(t *testing.T) {
	tmp := t.TempDir()
	ctx, _, stderr := newOriginCtx()
	m := New(testConfig(), ctx)
	m.downloadPathSuffix = "linux-amd64"
	m.execFilePath = filepath.Join(tmp, "test-bin")
	m.versionFilePath = filepath.Join(tmp, ".test-bin_version")
	m.checkVersionCacheFilePath = filepath.Join(tmp, ".test-bin_version_check")
	m.installed = true

	if err := os.WriteFile(m.checkVersionCacheFilePath, []byte("1000"), 0644); err != nil {
		t.Fatal(err)
	}

	m.getLatestVersionFn = func(ctx context.Context, base string) (string, error) {
		return "", fmt.Errorf("network error")
	}
	m.timeNowFn = func() time.Time { return time.Unix(999999999, 0) }

	if err := m.EnsureInstalledAndUpdated(context.Background()); err != nil {
		t.Fatalf("err: %v", err)
	}
	// User must see a warning about the version-check failure.
	if !strings.Contains(stderr.String(), "failed to check for") {
		t.Errorf("expected stderr warning, got: %s", stderr.String())
	}
	// Cache file should now hold the new timestamp, not 1000.
	data, err := os.ReadFile(m.checkVersionCacheFilePath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) == "1000" {
		t.Errorf("cache time should have been advanced even though GetLatestVersion failed (otherwise: thundering herd on every invocation)")
	}
}

// Same regression for the Install-failure path of the update branch: the
// user must see a warning AND the cache must be stamped (so the failure is
// throttled by the TTL).
func TestEnsureInstalledAndUpdated_StampsCacheOnInstallFail(t *testing.T) {
	tmp := t.TempDir()
	ctx, _, stderr := newOriginCtx()
	m := New(testConfig(), ctx)
	m.downloadPathSuffix = "linux-amd64"
	m.execFilePath = filepath.Join(tmp, "test-bin")
	m.versionFilePath = filepath.Join(tmp, ".test-bin_version")
	m.checkVersionCacheFilePath = filepath.Join(tmp, ".test-bin_version_check")
	m.installed = true

	if err := os.WriteFile(m.versionFilePath, []byte("v1.0.0"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.checkVersionCacheFilePath, []byte("1000"), 0644); err != nil {
		t.Fatal(err)
	}

	m.downloadBinaryFn = func(ctx context.Context, url, exe string) error {
		return fmt.Errorf("download failed")
	}
	m.getLatestVersionFn = func(ctx context.Context, base string) (string, error) { return "v2.0.0", nil }
	m.timeNowFn = func() time.Time { return time.Unix(999999999, 0) }

	if err := m.EnsureInstalledAndUpdated(context.Background()); err != nil {
		t.Fatalf("update-path Install failure should not propagate, got: %v", err)
	}
	if !strings.Contains(stderr.String(), "failed to update") {
		t.Errorf("expected update-failure warning, got: %s", stderr.String())
	}
	data, err := os.ReadFile(m.checkVersionCacheFilePath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) == "1000" {
		t.Errorf("cache time should have been advanced even though Install failed (otherwise: thundering herd on every invocation)")
	}
}

// Run() integration test: not installed, fresh install and exec succeed.
func TestRun_NotInstalled_FreshInstallAndExecute(t *testing.T) {
	prepareConfigHome(t)

	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	installCount := 0
	m.downloadBinaryFn = func(ctx context.Context, url, exe string) error {
		installCount++
		if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755); err != nil {
			return err
		}
		return nil
	}
	m.execCommandFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if runtime.GOOS == "windows" {
			return exec.CommandContext(ctx, "cmd", "/c", "exit 0")
		}
		return exec.CommandContext(ctx, "true")
	}
	m.getLatestVersionFn = func(ctx context.Context, base string) (string, error) { return "v1.0.0", nil }

	if err := m.Run(nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if installCount != 1 {
		t.Errorf("expected 1 install, got %d", installCount)
	}
}

// Run() integration test: installed, recent cache stamp → no download.
func TestRun_Installed_SkipDownload(t *testing.T) {
	prepareConfigHome(t)
	cfgDir := config.GetConfigPath()
	execFile := filepath.Join(cfgDir, "test-bin")
	if err := os.WriteFile(execFile, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write exec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ".test-bin_version_check"),
		[]byte(fmt.Sprintf("%d", time.Now().Unix())), 0644); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	ctx, _, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	installCount := 0
	m.downloadBinaryFn = func(ctx context.Context, url, exe string) error { installCount++; return nil }
	m.execCommandFn = mockOKExec
	m.getLatestVersionFn = func(ctx context.Context, base string) (string, error) { return "v1.0.0", nil }

	if err := m.Run(nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if installCount != 0 {
		t.Errorf("expected no download when installed and cache fresh, got %d", installCount)
	}
}

func mockOKExec(ctx context.Context, name string, args ...string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/c", "exit 0")
	}
	return exec.CommandContext(ctx, "true")
}

// mockEnvPrintExec returns a command that prints its environment to stdout.
// When cmd.Env is set by Execute, the subprocess sees only those variables,
// allowing us to verify the injected env vars via the captured output.
func mockEnvPrintExec(ctx context.Context, name string, args ...string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/c", "set")
	}
	return exec.CommandContext(ctx, "env")
}

func TestExecute_InjectsEnvVars(t *testing.T) {
	ctx, out, _ := newOriginCtx()
	m := New(testConfig(), ctx)
	m.execFilePath = "/no/op"
	m.execCommandFn = mockEnvPrintExec
	if err := m.Execute(context.Background(), nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := out.String()
	cfg := testConfig()

	// Verify EnvUserAgent is injected with the correct value.
	expectedUA := cfg.EnvUserAgent + "=aliyun-cli/" + cli.Version
	if !strings.Contains(output, expectedUA) {
		t.Errorf("expected env %s in subprocess output, got:\n%s", expectedUA, output)
	}

	// Verify EnvCompatMode is injected with the correct value.
	expectedCompat := cfg.EnvCompatMode + "=" + cfg.EnvCompatModeVal
	if !strings.Contains(output, expectedCompat) {
		t.Errorf("expected env %s in subprocess output, got:\n%s", expectedCompat, output)
	}
}

// TestExecute_CmdEnvContainsUserAgent verifies cmd.Env is populated with the
// correct User-Agent and compat-mode variables without relying on subprocess
// output. This avoids any platform-specific behavior (env/set commands).
func TestExecute_CmdEnvContainsUserAgent(t *testing.T) {
	var capturedCmd *exec.Cmd

	ctx, _, _ := newOriginCtx()
	cfg := testConfig()
	m := New(cfg, ctx)
	m.execFilePath = "/no/op"
	m.execCommandFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if runtime.GOOS == "windows" {
			capturedCmd = exec.CommandContext(ctx, "cmd", "/c", "exit 0")
		} else {
			capturedCmd = exec.CommandContext(ctx, "true")
		}
		return capturedCmd
	}
	if err := m.Execute(context.Background(), nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if capturedCmd == nil {
		t.Fatal("execCommandFn was not called")
	}

	// After Execute returns, cmd.Env has been set. Verify it contains the
	// expected env vars by scanning the slice directly.
	expectedUA := cfg.EnvUserAgent + "=aliyun-cli/" + cli.Version
	expectedCompat := cfg.EnvCompatMode + "=" + cfg.EnvCompatModeVal

	var foundUA, foundCompat bool
	for _, e := range capturedCmd.Env {
		if e == expectedUA {
			foundUA = true
		}
		if e == expectedCompat {
			foundCompat = true
		}
	}
	if !foundUA {
		t.Errorf("cmd.Env missing %q, got:\n%v", expectedUA, capturedCmd.Env)
	}
	if !foundCompat {
		t.Errorf("cmd.Env missing %q, got:\n%v", expectedCompat, capturedCmd.Env)
	}
}

// TestExecute_CmdEnvNoDuplicates verifies that when a conflicting env var
// already exists in the process environment, Execute filters it out and
// replaces it with the override — resulting in exactly one occurrence.
func TestExecute_CmdEnvNoDuplicates(t *testing.T) {
	cfg := testConfig()
	t.Setenv(cfg.EnvUserAgent, "stale-value")

	var capturedCmd *exec.Cmd

	ctx, _, _ := newOriginCtx()
	m := New(cfg, ctx)
	m.execFilePath = "/no/op"
	m.execCommandFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if runtime.GOOS == "windows" {
			capturedCmd = exec.CommandContext(ctx, "cmd", "/c", "exit 0")
		} else {
			capturedCmd = exec.CommandContext(ctx, "true")
		}
		return capturedCmd
	}
	if err := m.Execute(context.Background(), nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	expectedUA := cfg.EnvUserAgent + "=aliyun-cli/" + cli.Version
	count := 0
	for _, e := range capturedCmd.Env {
		if strings.HasPrefix(e, cfg.EnvUserAgent+"=") {
			count++
			if e != expectedUA {
				t.Errorf("unexpected value %q, want %q", e, expectedUA)
			}
		}
	}
	if count != 1 {
		t.Errorf("expected %s exactly once in cmd.Env, got %d occurrences", cfg.EnvUserAgent, count)
	}
}

func TestExecute_FiltersDuplicateEnvVars(t *testing.T) {
	cfg := testConfig()

	// Set a conflicting env var in the process environment that should be
	// filtered out by filterEnv and replaced by the override value.
	t.Setenv(cfg.EnvUserAgent, "old-value-should-be-overwritten")

	ctx, out, _ := newOriginCtx()
	m := New(cfg, ctx)
	m.execFilePath = "/no/op"
	m.execCommandFn = mockEnvPrintExec
	if err := m.Execute(context.Background(), nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := out.String()

	// The new override value should be present.
	expectedUA := cfg.EnvUserAgent + "=aliyun-cli/" + cli.Version
	if !strings.Contains(output, expectedUA) {
		t.Errorf("expected env %s in subprocess output, got:\n%s", expectedUA, output)
	}

	// The old conflicting value should NOT be present.
	oldUA := cfg.EnvUserAgent + "=old-value-should-be-overwritten"
	if strings.Contains(output, oldUA) {
		t.Errorf("old env %s should not appear in subprocess output, got:\n%s", oldUA, output)
	}

	// The variable should appear exactly once (no duplicates).
	count := strings.Count(output, cfg.EnvUserAgent+"=")
	if count != 1 {
		t.Errorf("expected %s= to appear exactly once, got %d occurrences", cfg.EnvUserAgent, count)
	}
}

func TestParseVersionParts(t *testing.T) {
	tests := []struct {
		input  string
		ver    string
		commit string
		valid  bool
	}{
		{"v0.1.0-a1b2c3d", "v0.1.0", "a1b2c3d", true},
		{"v1.2.3-abcdef1234567", "v1.2.3", "abcdef1234567", true},
		{"v0.1.0-ABCDEF1", "v0.1.0", "ABCDEF1", true},
		{"abc1234", "", "abc1234", false},              // 无 semver 前缀
		{"v0.1.0", "", "v0.1.0", false},                  // 无 commit 后缀
		{"v0.1.0-short", "", "v0.1.0-short", false},       // commit 太短（不是 hex）
		{"v0.1.0-xyz1234", "", "v0.1.0-xyz1234", false},   // 非 hex 字符
		{"", "", "", false},
		{"-a1b2c3d", "", "-a1b2c3d", false},                // 空 semver 部分
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ver, commit, valid := parseVersionParts(tt.input)
			assert.Equal(t, tt.ver, ver)
			assert.Equal(t, tt.commit, commit)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

func TestNeedsUpdate(t *testing.T) {
	tests := []struct {
		name   string
		local  string
		remote string
		expect bool
	}{
		{"same version", "v0.1.0-abc1234", "v0.1.0-abc1234", false},
		{"upgrade", "v0.1.0-abc1234", "v0.2.0-def5678", true},
		{"downgrade blocked", "v0.2.0-def5678", "v0.1.0-abc1234", false},
		{"same ver diff commit", "v0.1.0-abc1234", "v0.1.0-def5678", true},
		{"empty local", "", "v0.1.0-abc1234", true},
		{"old format both invalid different", "abcdef1", "1234567", true},
		{"mixed format old to new", "abcdef1", "v0.1.0-abc1234", true},
		{"mixed format new to old", "v0.1.0-abc1234", "abcdef1", true},
		{"major upgrade", "v1.0.0-abc1234", "v2.0.0-def5678", true},
		{"major downgrade blocked", "v2.0.0-abc1234", "v1.0.0-def5678", false},
		{"patch upgrade", "v0.1.0-abc1234", "v0.1.1-def5678", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsUpdate(tt.local, tt.remote)
			assert.Equal(t, tt.expect, result)
		})
	}
}
