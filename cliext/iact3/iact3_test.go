package iact3

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func prepareConfig(t *testing.T, home string) {
	cfgDir := filepath.Join(home, ".aliyun")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	configJSON := `{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou","language":"en"}]}`
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

func TestPrepareEnv_Success(t *testing.T) {
	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)
	prepareConfig(t, home)

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.PrepareEnv(); err != nil {
		t.Fatalf("PrepareEnv err: %v", err)
	}

	if c.envMap["ALIBABA_CLOUD_REGION_ID"] != "cn-hangzhou" {
		t.Fatalf("region mismatch: %v", c.envMap["ALIBABA_CLOUD_REGION_ID"])
	}
	if c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"] != "ak" {
		t.Fatalf("ak mismatch: %v", c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"])
	}
	if c.envMap["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] != "sk" {
		t.Fatalf("sk mismatch: %v", c.envMap["ALIBABA_CLOUD_ACCESS_KEY_SECRET"])
	}
	if c.envMap["ALIBABA_CLOUD_IACT3_COMPAT_MODE"] != "aliyun" {
		t.Fatalf("compat mode mismatch: %v", c.envMap["ALIBABA_CLOUD_IACT3_COMPAT_MODE"])
	}
}

func TestRemoveFlagsForMainCli_Success(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "profile", "test")

	c := NewContext(ctx)
	args, err := c.RemoveFlagsForMainCli([]string{"aliyun", "iact3", "test", "--profile", "test"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--profile") {
		t.Fatalf("profile flag should be removed: %s", joined)
	}
	if !strings.Contains(joined, "iact3 test") {
		t.Fatalf("original args missing: %s", joined)
	}
}

func TestRemoveFlagsForMainCli_InlineValue(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	args, err := c.RemoveFlagsForMainCli([]string{"--profile=prod", "--region=cn-hangzhou", "upgrade"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(args) != 1 || args[0] != "upgrade" {
		t.Fatalf("inline-value flags not stripped: %v", args)
	}
}

func TestRemoveFlagsForMainCli_SpaceValueAndShorthand(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	// --profile is a long flag that takes a value; --mode also takes a value
	args, err := c.RemoveFlagsForMainCli([]string{"--profile", "prod", "--mode", "AK", "test", "--template", "foo"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--profile") || strings.Contains(joined, "--mode") {
		t.Fatalf("main cli flags should be stripped: %v", args)
	}
	if !strings.Contains(joined, "test") || !strings.Contains(joined, "--template") || !strings.Contains(joined, "foo") {
		t.Fatalf("user args should be preserved: %v", args)
	}
}

func TestRun_UpgradeAfterMainFlags(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) { return "0.1.13", nil }
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	installCalled := false
	oldDownload := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, dest, exe string) error {
		installCalled = true
		return os.WriteFile(exe, []byte("fake"), 0755)
	}
	defer func() { downloadAndExtractFunc = oldDownload }()

	execCalled := false
	oldExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		execCalled = true
		return exec.Command("true")
	}
	defer func() { execCommandFunc = oldExec }()

	// Simulate `aliyun iact3 --profile prod upgrade`: KeepArgs passes through
	// main CLI flags, so `upgrade` is not at args[0].
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"--profile", "prod", "upgrade"}); err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if !installCalled {
		t.Errorf("upgrade should still be dispatched even when main CLI flags precede it")
	}
	if execCalled {
		t.Errorf("subprocess should NOT be invoked on upgrade path")
	}
}

func TestCheckOsTypeAndArch(t *testing.T) {
	tests := []struct {
		goos      string
		goarch    string
		supported bool
		suffix    string
	}{
		{"linux", "amd64", true, "linux-amd64.tar.gz"},
		{"darwin", "arm64", true, "darwin-arm64.tar.gz"},
		{"linux", "arm64", false, ""},
		{"darwin", "amd64", false, ""},
		{"windows", "amd64", false, ""},
		{"windows", "arm64", false, ""},
		{"linux", "386", false, ""},
		{"freebsd", "amd64", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"_"+tt.goarch, func(t *testing.T) {
			oldGOOS := runtimeGOOSFunc
			oldGOARCH := runtimeGOARCHFunc
			runtimeGOOSFunc = func() string { return tt.goos }
			runtimeGOARCHFunc = func() string { return tt.goarch }
			defer func() {
				runtimeGOOSFunc = oldGOOS
				runtimeGOARCHFunc = oldGOARCH
			}()

			ctx, _, _ := newOriginCtx()
			c := NewContext(ctx)
			c.CheckOsTypeAndArch()
			if c.osSupport != tt.supported {
				t.Errorf("osSupport: got %v, want %v", c.osSupport, tt.supported)
			}
			if tt.supported && c.downloadPathSuffix != tt.suffix {
				t.Errorf("downloadPathSuffix: got %s, want %s", c.downloadPathSuffix, tt.suffix)
			}
		})
	}
}

func TestInitBasicInfo(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()

	if c.configPath != tmpDir {
		t.Errorf("configPath: got %s, want %s", c.configPath, tmpDir)
	}
	if c.installed {
		t.Errorf("installed should be false when binary doesn't exist")
	}
	expectedExec := filepath.Join(tmpDir, "iact3")
	if c.execFilePath != expectedExec {
		t.Errorf("execFilePath: got %s, want %s", c.execFilePath, expectedExec)
	}
}

func TestSelfUpgrade_FreshInstall(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) { return "0.1.13", nil }
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	installCalled := false
	oldDownload := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, dest, exe string) error {
		installCalled = true
		return os.WriteFile(exe, []byte("fake"), 0755)
	}
	defer func() { downloadAndExtractFunc = oldDownload }()

	ctx, out, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()

	if err := c.SelfUpgrade(); err != nil {
		t.Fatalf("SelfUpgrade err: %v", err)
	}
	if !installCalled {
		t.Errorf("Install should be called for fresh install")
	}
	if !strings.Contains(out.String(), "not installed yet") {
		t.Errorf("expected 'not installed yet' message, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "upgraded to 0.1.13 successfully") {
		t.Errorf("expected upgrade success message, got: %s", out.String())
	}
}

func TestSelfUpgrade_AlreadyLatest(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	execPath := filepath.Join(tmpDir, "iact3")
	_ = os.WriteFile(execPath, []byte("fake"), 0755)

	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) { return "0.1.13", nil }
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	oldExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "0.1.13")
	}
	defer func() { execCommandFunc = oldExec }()

	installCalled := false
	oldDownload := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, dest, exe string) error {
		installCalled = true
		return nil
	}
	defer func() { downloadAndExtractFunc = oldDownload }()

	ctx, out, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()

	if err := c.SelfUpgrade(); err != nil {
		t.Fatalf("SelfUpgrade err: %v", err)
	}
	if installCalled {
		t.Errorf("Install should NOT be called when already latest")
	}
	if !strings.Contains(out.String(), "Already up to date") {
		t.Errorf("expected 'Already up to date' message, got: %s", out.String())
	}
}

func TestSelfUpgrade_NeedsUpgrade(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	execPath := filepath.Join(tmpDir, "iact3")
	_ = os.WriteFile(execPath, []byte("fake"), 0755)

	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) { return "0.1.13", nil }
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	oldExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "0.1.10")
	}
	defer func() { execCommandFunc = oldExec }()

	installCalled := false
	oldDownload := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, dest, exe string) error {
		installCalled = true
		return nil
	}
	defer func() { downloadAndExtractFunc = oldDownload }()

	ctx, out, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()

	if err := c.SelfUpgrade(); err != nil {
		t.Fatalf("SelfUpgrade err: %v", err)
	}
	if !installCalled {
		t.Errorf("Install should be called when versions differ")
	}
	if !strings.Contains(out.String(), "Current installed iact3 version: 0.1.10") {
		t.Errorf("expected current version output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "upgraded to 0.1.13 successfully") {
		t.Errorf("expected upgrade success message, got: %s", out.String())
	}
}

func TestRun_NoAutoUpgrade(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)
	prepareConfig(t, home)

	execPath := filepath.Join(tmpDir, "iact3")
	_ = os.WriteFile(execPath, []byte("fake"), 0755)

	verCheckCalled := false
	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) {
		verCheckCalled = true
		return "0.1.13", nil
	}
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	oldExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}
	defer func() { execCommandFunc = oldExec }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"test", "--template", "foo"}); err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if verCheckCalled {
		t.Errorf("version check must NOT be triggered for non-upgrade invocations once binary is installed")
	}
}

// readCache decodes the on-disk version-check cache; helper for hint tests.
func readCache(t *testing.T, dir string) iact3VersionCache {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, versionCacheFileName))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	var c iact3VersionCache
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("unmarshal cache: %v", err)
	}
	return c
}

func writeCache(t *testing.T, dir string, c iact3VersionCache) {
	t.Helper()
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, versionCacheFileName), data, 0600); err != nil {
		t.Fatalf("write cache: %v", err)
	}
}

func TestRun_FreshInstallWritesCache(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	origHOME := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", origHOME) }()
	home := t.TempDir()
	_ = os.Setenv("HOME", home)
	prepareConfig(t, home)

	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) { return "0.1.13", nil }
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	oldDownload := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, dest, exe string) error {
		return os.WriteFile(exe, []byte("fake"), 0755)
	}
	defer func() { downloadAndExtractFunc = oldDownload }()

	oldExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return exec.Command("true") }
	defer func() { execCommandFunc = oldExec }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"test"}); err != nil {
		t.Fatalf("Run err: %v", err)
	}

	cache := readCache(t, tmpDir)
	if cache.InstalledVersion != "0.1.13" || cache.LastKnownRemote != "0.1.13" {
		t.Errorf("fresh-install cache mismatch: %+v", cache)
	}
	if cache.LastRemoteCheck == 0 {
		t.Errorf("LastRemoteCheck should be set")
	}
}

func TestSelfUpgrade_AlreadyLatestWritesCache(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	_ = os.WriteFile(filepath.Join(tmpDir, "iact3"), []byte("fake"), 0755)

	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) { return "0.1.13", nil }
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	oldExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return exec.Command("echo", "0.1.13") }
	defer func() { execCommandFunc = oldExec }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.SelfUpgrade(); err != nil {
		t.Fatalf("SelfUpgrade err: %v", err)
	}
	cache := readCache(t, tmpDir)
	if cache.InstalledVersion != "0.1.13" || cache.LastKnownRemote != "0.1.13" {
		t.Errorf("already-latest cache mismatch: %+v", cache)
	}
}

func TestSelfUpgrade_UpgradePathWritesCache(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	_ = os.WriteFile(filepath.Join(tmpDir, "iact3"), []byte("fake"), 0755)

	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) { return "0.1.13", nil }
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	oldExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return exec.Command("echo", "0.1.10") }
	defer func() { execCommandFunc = oldExec }()

	oldDownload := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, dest, exe string) error { return nil }
	defer func() { downloadAndExtractFunc = oldDownload }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if err := c.SelfUpgrade(); err != nil {
		t.Fatalf("SelfUpgrade err: %v", err)
	}
	cache := readCache(t, tmpDir)
	if cache.InstalledVersion != "0.1.13" || cache.LastKnownRemote != "0.1.13" {
		t.Errorf("upgrade-path cache mismatch: %+v", cache)
	}
}

func TestMaybeShowUpgradeHint_FreshCacheNoNetwork(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	_ = os.WriteFile(filepath.Join(tmpDir, "iact3"), []byte("fake"), 0755)

	now := time.Now()
	oldNow := timeNowFunc
	timeNowFunc = func() time.Time { return now }
	defer func() { timeNowFunc = oldNow }()
	writeCache(t, tmpDir, iact3VersionCache{
		InstalledVersion: "0.1.10",
		LastKnownRemote:  "0.1.13",
		LastRemoteCheck:  now.Unix(),
	})

	netCalled := false
	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) {
		netCalled = true
		return "", nil
	}
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	ctx, _, errOut := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.maybeShowUpgradeHint()

	if netCalled {
		t.Errorf("fresh cache must NOT trigger network call")
	}
	if !strings.Contains(errOut.String(), "newer iact3 version 0.1.13") {
		t.Errorf("expected upgrade hint, got: %q", errOut.String())
	}
	if !strings.Contains(errOut.String(), "you have 0.1.10") {
		t.Errorf("expected installed-version mention, got: %q", errOut.String())
	}
}

func TestMaybeShowUpgradeHint_StaleCacheRefreshes(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	_ = os.WriteFile(filepath.Join(tmpDir, "iact3"), []byte("fake"), 0755)

	now := time.Now()
	oldNow := timeNowFunc
	timeNowFunc = func() time.Time { return now }
	defer func() { timeNowFunc = oldNow }()
	// LastRemoteCheck is 2 days ago — well past the 24h TTL.
	writeCache(t, tmpDir, iact3VersionCache{
		InstalledVersion: "0.1.10",
		LastKnownRemote:  "0.1.10",
		LastRemoteCheck:  now.Add(-48 * time.Hour).Unix(),
	})

	netCalled := false
	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) {
		netCalled = true
		return "0.1.14", nil
	}
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	ctx, _, errOut := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.maybeShowUpgradeHint()

	if !netCalled {
		t.Errorf("stale cache must trigger network refresh")
	}
	cache := readCache(t, tmpDir)
	if cache.LastKnownRemote != "0.1.14" {
		t.Errorf("refreshed cache should record new remote, got: %+v", cache)
	}
	if cache.LastRemoteCheck != now.Unix() {
		t.Errorf("refreshed cache should bump timestamp, got: %d want %d", cache.LastRemoteCheck, now.Unix())
	}
	if !strings.Contains(errOut.String(), "newer iact3 version 0.1.14") {
		t.Errorf("expected refreshed hint, got: %q", errOut.String())
	}
}

func TestMaybeShowUpgradeHint_NoHintWhenSame(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	_ = os.WriteFile(filepath.Join(tmpDir, "iact3"), []byte("fake"), 0755)

	now := time.Now()
	oldNow := timeNowFunc
	timeNowFunc = func() time.Time { return now }
	defer func() { timeNowFunc = oldNow }()
	writeCache(t, tmpDir, iact3VersionCache{
		InstalledVersion: "0.1.13",
		LastKnownRemote:  "0.1.13",
		LastRemoteCheck:  now.Unix(),
	})

	ctx, _, errOut := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.maybeShowUpgradeHint()

	if errOut.String() != "" {
		t.Errorf("no hint expected when versions match, got: %q", errOut.String())
	}
}

func TestMaybeShowUpgradeHint_BootstrapPath(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	_ = os.WriteFile(filepath.Join(tmpDir, "iact3"), []byte("fake"), 0755)

	oldExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return exec.Command("echo", "0.1.10") }
	defer func() { execCommandFunc = oldExec }()

	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) { return "0.1.13", nil }
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	ctx, _, errOut := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.maybeShowUpgradeHint()

	cache := readCache(t, tmpDir)
	if cache.InstalledVersion != "0.1.10" || cache.LastKnownRemote != "0.1.13" {
		t.Errorf("bootstrap cache mismatch: %+v", cache)
	}
	if !strings.Contains(errOut.String(), "newer iact3 version 0.1.13") {
		t.Errorf("expected bootstrap hint, got: %q", errOut.String())
	}
}

func TestMaybeShowUpgradeHint_NotInstalledNoop(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	netCalled := false
	oldGetVer := getLatestIact3VersionFunc
	getLatestIact3VersionFunc = func() (string, error) {
		netCalled = true
		return "0.1.13", nil
	}
	defer func() { getLatestIact3VersionFunc = oldGetVer }()

	ctx, _, errOut := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo() // c.installed = false (no binary on disk)
	c.maybeShowUpgradeHint()

	if netCalled {
		t.Errorf("hint must not run when binary is absent")
	}
	if errOut.String() != "" {
		t.Errorf("no hint expected when not installed, got: %q", errOut.String())
	}
}
