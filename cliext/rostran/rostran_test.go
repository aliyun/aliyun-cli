package rostran

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	if c.envMap["ALIBABA_CLOUD_ROSTRAN_COMPAT_MODE"] != "aliyun" {
		t.Fatalf("compat mode mismatch: %v", c.envMap["ALIBABA_CLOUD_ROSTRAN_COMPAT_MODE"])
	}
}

func TestRemoveFlagsForMainCli_Success(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "profile", "test")

	c := NewContext(ctx)
	args, err := c.RemoveFlagsForMainCli([]string{"aliyun", "rostran", "convert", "--profile", "test"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--profile") {
		t.Fatalf("profile flag should be removed: %s", joined)
	}
	if !strings.Contains(joined, "rostran convert") {
		t.Fatalf("original args missing: %s", joined)
	}
}

func TestRemoveFlagsForMainCli_PreservesStandaloneVersionFlag(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	args, err := c.RemoveFlagsForMainCli([]string{"--version"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(args) != 1 || args[0] != "--version" {
		t.Fatalf("standalone --version should be forwarded to rostran, got %v", args)
	}
}

func TestRun_UpgradeAfterMainFlags(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	oldGetVer := getLatestRostranVersionFunc
	getLatestRostranVersionFunc = func() (string, error) { return "0.1.0", nil }
	defer func() { getLatestRostranVersionFunc = oldGetVer }()

	installCalled := false
	oldDownload := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, dest, installDir string) error {
		installCalled = true
		execPath := filepath.Join(installDir, "rostran")
		if err := os.MkdirAll(filepath.Dir(execPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(execPath, []byte("fake"), 0755)
	}
	defer func() { downloadAndExtractFunc = oldDownload }()

	execCalled := false
	oldExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		execCalled = true
		return exec.Command("true")
	}
	defer func() { execCommandFunc = oldExec }()

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
		t.Errorf("installed should be false when install dir doesn't exist")
	}
	expectedInstallDir := filepath.Join(tmpDir, "rostran")
	if c.installDirPath != expectedInstallDir {
		t.Errorf("installDirPath: got %s, want %s", c.installDirPath, expectedInstallDir)
	}
}

func TestInitBasicInfoDetectsExecutableInsideInstallDir(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "rostran", "rostran")
	if err := os.MkdirAll(filepath.Dir(execPath), 0755); err != nil {
		t.Fatalf("mkdir exec dir: %v", err)
	}
	if err := os.WriteFile(execPath, []byte("fake"), 0755); err != nil {
		t.Fatalf("write exec: %v", err)
	}

	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()

	if !c.installed {
		t.Errorf("installed should be true when rostran executable exists inside install dir")
	}
	if c.execFilePath != execPath {
		t.Errorf("execFilePath: got %s, want %s", c.execFilePath, execPath)
	}
}

func TestDownloadAndExtract_PreservesPackageDirectoryContents(t *testing.T) {
	archiveBytes := buildTarGz(t, map[string]string{
		"rostran/rostran":             "#!/bin/sh\necho rostran\n",
		"rostran/_internal/rules.txt": "keep me",
	})

	oldHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(archiveBytes)),
		}, nil
	}
	defer func() { httpGetFunc = oldHTTPGet }()

	tmpDir := t.TempDir()
	destFile := filepath.Join(tmpDir, "rostran_download.tar.gz")
	installDir := filepath.Join(tmpDir, "rostran")

	if err := DownloadAndExtract("https://example.com/rostran.tar.gz", destFile, installDir); err != nil {
		t.Fatalf("DownloadAndExtract err: %v", err)
	}

	if _, err := os.Stat(filepath.Join(installDir, "rostran")); err != nil {
		t.Fatalf("expected executable from preserved directory: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(installDir, "_internal", "rules.txt")); err != nil || string(data) != "keep me" {
		t.Fatalf("expected support file to be preserved, data=%q err=%v", string(data), err)
	}
	if _, err := os.Stat(destFile); !os.IsNotExist(err) {
		t.Fatalf("download archive should be removed after install, err=%v", err)
	}
}

func TestDownloadAndExtract_RejectsArchivePathTraversal(t *testing.T) {
	archiveBytes := buildTarGz(t, map[string]string{
		"../escape":       "bad",
		"rostran/rostran": "ok",
	})

	oldHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(archiveBytes)),
		}, nil
	}
	defer func() { httpGetFunc = oldHTTPGet }()

	tmpDir := t.TempDir()
	if err := DownloadAndExtract("https://example.com/rostran.tar.gz", filepath.Join(tmpDir, "rostran.tar.gz"), filepath.Join(tmpDir, "rostran")); err != nil {
		t.Fatalf("DownloadAndExtract err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "escape")); !os.IsNotExist(err) {
		t.Fatalf("path traversal entry should not be written outside install dir, err=%v", err)
	}
}

func TestDownloadAndExtract_KeepsExistingInstallWhenStagingFails(t *testing.T) {
	archiveBytes := buildTarGz(t, map[string]string{
		"rostran/rostran": "#!/bin/sh\necho rostran\n",
	})

	oldHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(archiveBytes)),
		}, nil
	}
	defer func() { httpGetFunc = oldHTTPGet }()

	oldMoveDir := moveDirFunc
	moveDirFunc = func(src, dst string) error {
		return errors.New("disk full")
	}
	defer func() { moveDirFunc = oldMoveDir }()

	tmpDir := t.TempDir()
	installDir := filepath.Join(tmpDir, "rostran")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}
	oldMarker := filepath.Join(installDir, "old-version")
	if err := os.WriteFile(oldMarker, []byte("old"), 0600); err != nil {
		t.Fatalf("write old marker: %v", err)
	}

	err := DownloadAndExtract("https://example.com/rostran.tar.gz", filepath.Join(tmpDir, "rostran.tar.gz"), installDir)
	if err == nil {
		t.Fatal("DownloadAndExtract should fail when staging new install dir fails")
	}

	data, readErr := os.ReadFile(oldMarker)
	if readErr != nil || string(data) != "old" {
		t.Fatalf("old install should remain intact, data=%q err=%v", string(data), readErr)
	}
}

func TestDownloadAndExtract_PreservesSafeSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires extra privileges on Windows")
	}

	archiveBytes := buildTarGzWithSymlinks(t, map[string]string{
		"rostran/rostran": "#!/bin/sh\necho rostran\n",
		"rostran/_internal/Python.framework/Versions/3.11/Python": "python",
	}, map[string]string{
		"rostran/_internal/Python": "Python.framework/Versions/3.11/Python",
		"rostran/_internal/bad":    "../../escape",
	})

	oldHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(archiveBytes)),
		}, nil
	}
	defer func() { httpGetFunc = oldHTTPGet }()

	tmpDir := t.TempDir()
	installDir := filepath.Join(tmpDir, "rostran")
	if err := DownloadAndExtract("https://example.com/rostran.tar.gz", filepath.Join(tmpDir, "rostran.tar.gz"), installDir); err != nil {
		t.Fatalf("DownloadAndExtract err: %v", err)
	}

	linkPath := filepath.Join(installDir, "_internal", "Python")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat safe symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink, mode=%v", linkPath, info.Mode())
	}
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("readlink safe symlink: %v", err)
	}
	if target != "Python.framework/Versions/3.11/Python" {
		t.Fatalf("symlink target = %q", target)
	}
	if _, err := os.Lstat(filepath.Join(installDir, "_internal", "bad")); !os.IsNotExist(err) {
		t.Fatalf("escaping symlink should not be created, err=%v", err)
	}
}

func TestDownloadAndExtract_DelaysSymlinkCreationUntilAfterFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires extra privileges on Windows")
	}

	archiveBytes := buildTarGzInOrder(t, []tarTestEntry{
		{name: "rostran/link", symlink: true, target: "_internal"},
		{name: "rostran/link/through-link.txt", body: "through link"},
		{name: "rostran/_internal/real.txt", body: "real"},
		{name: "rostran/rostran", body: "#!/bin/sh\necho rostran\n"},
	})

	oldHTTPGet := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(archiveBytes)),
		}, nil
	}
	defer func() { httpGetFunc = oldHTTPGet }()

	tmpDir := t.TempDir()
	installDir := filepath.Join(tmpDir, "rostran")
	if err := DownloadAndExtract("https://example.com/rostran.tar.gz", filepath.Join(tmpDir, "rostran.tar.gz"), installDir); err != nil {
		t.Fatalf("DownloadAndExtract err: %v", err)
	}

	if _, err := os.Stat(filepath.Join(installDir, "_internal", "through-link.txt")); !os.IsNotExist(err) {
		t.Fatalf("file must not be written through delayed symlink target, err=%v", err)
	}
	data, err := os.ReadFile(filepath.Join(installDir, "link", "through-link.txt"))
	if err != nil || string(data) != "through link" {
		t.Fatalf("expected file under real directory, data=%q err=%v", string(data), err)
	}
	info, err := os.Lstat(filepath.Join(installDir, "link"))
	if err != nil {
		t.Fatalf("lstat link path: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("link path should remain a directory when archive writes files below it")
	}
}

func TestCopyDirPreservesSafeSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires extra privileges on Windows")
	}

	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dst := filepath.Join(tmpDir, "dst")
	targetFile := filepath.Join(src, "_internal", "Python.framework", "Versions", "3.11", "Python")
	if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
		t.Fatalf("mkdir target file dir: %v", err)
	}
	if err := os.WriteFile(targetFile, []byte("python"), 0755); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	linkDir := filepath.Join(src, "_internal")
	if err := os.Symlink("Python.framework/Versions/3.11/Python", filepath.Join(linkDir, "Python")); err != nil {
		t.Fatalf("create source symlink: %v", err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir err: %v", err)
	}

	linkPath := filepath.Join(dst, "_internal", "Python")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat copied symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected copied link to remain a symlink, mode=%v", info.Mode())
	}
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("read copied symlink: %v", err)
	}
	if target != "Python.framework/Versions/3.11/Python" {
		t.Fatalf("copied symlink target = %q", target)
	}
}

func TestInstallSetsExecPathInsidePreservedDir(t *testing.T) {
	tmpDir := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	oldDownload := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, dest, installDir string) error {
		execPath := filepath.Join(installDir, "rostran")
		if err := os.MkdirAll(filepath.Dir(execPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(execPath, []byte("fake"), 0644); err != nil {
			return err
		}
		if !strings.Contains(url, "/alibabacloud-ros-tool-transformer/0.1.0/rostran-0.1.0-darwin-arm64.tar.gz") {
			t.Fatalf("unexpected download url: %s", url)
		}
		return nil
	}
	defer func() { downloadAndExtractFunc = oldDownload }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.osSupport = true
	c.osType = "darwin"
	c.osArch = "arm64"
	c.downloadPathSuffix = "darwin-arm64.tar.gz"
	c.versionRemote = "0.1.0"

	if err := c.Install(); err != nil {
		t.Fatalf("Install err: %v", err)
	}

	expectedExec := filepath.Join(tmpDir, "rostran", "rostran")
	if c.execFilePath != expectedExec {
		t.Fatalf("execFilePath: got %s, want %s", c.execFilePath, expectedExec)
	}
	info, err := os.Stat(expectedExec)
	if err != nil {
		t.Fatalf("stat exec: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Fatalf("expected executable bit on %s, mode=%v", expectedExec, info.Mode())
	}
}

func readCache(t *testing.T, dir string) rostranVersionCache {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, versionCacheFileName))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	var c rostranVersionCache
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("unmarshal cache: %v", err)
	}
	return c
}

func writeCache(t *testing.T, dir string, c rostranVersionCache) {
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

	oldGetVer := getLatestRostranVersionFunc
	getLatestRostranVersionFunc = func() (string, error) { return "0.1.0", nil }
	defer func() { getLatestRostranVersionFunc = oldGetVer }()

	oldDownload := downloadAndExtractFunc
	downloadAndExtractFunc = func(url, dest, installDir string) error {
		execPath := filepath.Join(installDir, "rostran")
		if err := os.MkdirAll(filepath.Dir(execPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(execPath, []byte("fake"), 0755)
	}
	defer func() { downloadAndExtractFunc = oldDownload }()

	oldExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return exec.Command("true") }
	defer func() { execCommandFunc = oldExec }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"convert"}); err != nil {
		t.Fatalf("Run err: %v", err)
	}

	cache := readCache(t, tmpDir)
	if cache.InstalledVersion != "0.1.0" || cache.LastKnownRemote != "0.1.0" {
		t.Errorf("fresh-install cache mismatch: %+v", cache)
	}
	if cache.LastRemoteCheck == 0 {
		t.Errorf("LastRemoteCheck should be set")
	}
}

func TestMaybeShowUpgradeHint_FreshCacheNoNetwork(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "rostran", "rostran")
	if err := os.MkdirAll(filepath.Dir(execPath), 0755); err != nil {
		t.Fatalf("mkdir exec dir: %v", err)
	}
	_ = os.WriteFile(execPath, []byte("fake"), 0755)

	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = oldGet }()

	now := time.Now()
	oldNow := timeNowFunc
	timeNowFunc = func() time.Time { return now }
	defer func() { timeNowFunc = oldNow }()
	writeCache(t, tmpDir, rostranVersionCache{
		InstalledVersion: "0.1.0",
		LastKnownRemote:  "0.1.1",
		LastRemoteCheck:  now.Unix(),
	})

	netCalled := false
	oldGetVer := getLatestRostranVersionFunc
	getLatestRostranVersionFunc = func() (string, error) {
		netCalled = true
		return "", nil
	}
	defer func() { getLatestRostranVersionFunc = oldGetVer }()

	ctx, _, errOut := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.maybeShowUpgradeHint()

	if netCalled {
		t.Errorf("fresh cache must NOT trigger network call")
	}
	if !strings.Contains(errOut.String(), "newer rostran version 0.1.1") {
		t.Errorf("expected upgrade hint, got: %q", errOut.String())
	}
}

func buildTarGz(t *testing.T, files map[string]string) []byte {
	return buildTarGzWithSymlinks(t, files, nil)
}

func buildTarGzWithSymlinks(t *testing.T, files map[string]string, symlinks map[string]string) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)
	for name, body := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(body)),
		}
		if filepath.Base(name) == "rostran" {
			hdr.Mode = 0755
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatalf("write tar entry: %v", err)
		}
	}
	for name, target := range symlinks {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0777,
			Typeflag: tar.TypeSymlink,
			Linkname: target,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar symlink header: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}

type tarTestEntry struct {
	name    string
	body    string
	symlink bool
	target  string
}

func buildTarGzInOrder(t *testing.T, entries []tarTestEntry) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)
	for _, entry := range entries {
		if entry.symlink {
			hdr := &tar.Header{
				Name:     entry.name,
				Mode:     0777,
				Typeflag: tar.TypeSymlink,
				Linkname: entry.target,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatalf("write tar symlink header: %v", err)
			}
			continue
		}
		hdr := &tar.Header{
			Name: entry.name,
			Mode: 0600,
			Size: int64(len(entry.body)),
		}
		if filepath.Base(entry.name) == "rostran" {
			hdr.Mode = 0755
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(entry.body)); err != nil {
			t.Fatalf("write tar entry: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}
