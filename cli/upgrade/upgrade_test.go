package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Version helpers
// ---------------------------------------------------------------------------

func TestEnsureVPrefix(t *testing.T) {
	assert.Equal(t, "v3.0.1", ensureVPrefix("3.0.1"))
	assert.Equal(t, "v3.0.1", ensureVPrefix("v3.0.1"))
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current  string
		latest   string
		expected bool
	}{
		{"3.0.1", "3.0.2", true},
		{"3.0.2", "3.0.2", false},
		{"3.0.3", "3.0.2", false},
		{"3.0.0", "3.1.0", true},
		{"2.9.9", "3.0.0", true},
		{"3.0.0-beta", "3.0.0", true},
		{"3.0.0", "3.0.0-beta", false},
		{"3.0.0-alpha", "3.0.0-beta", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			assert.Equal(t, tt.expected, isNewer(tt.current, tt.latest))
		})
	}
}

func TestFormatSize(t *testing.T) {
	assert.Equal(t, "0 B", formatSize(0))
	assert.Equal(t, "512 B", formatSize(512))
	assert.Equal(t, "1.0 KB", formatSize(1024))
	assert.Equal(t, "1.5 KB", formatSize(1536))
	assert.Equal(t, "1.0 MB", formatSize(1024*1024))
	assert.Equal(t, "10.5 MB", formatSize(11010048))
}

// ---------------------------------------------------------------------------
// Installer detection
// ---------------------------------------------------------------------------

func TestDetectInstaller_Default(t *testing.T) {
	result := detectInstaller()
	// In a test environment the binary is in a temp dir, not Homebrew
	assert.Equal(t, installerDirect, result)
}

// ---------------------------------------------------------------------------
// Brew upgrade (mock exec)
// ---------------------------------------------------------------------------

func TestUpgradeViaBrew_Success(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	var calls [][]string
	execCommand = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, append([]string{name}, args...))
		return exec.Command("echo", "ok")
	}

	ctx := newTestContext()
	err := upgradeViaBrew(ctx)
	assert.NoError(t, err)

	assert.Len(t, calls, 2)
	assert.Equal(t, []string{"brew", "update"}, calls[0])
	assert.Equal(t, []string{"brew", "upgrade", "aliyun-cli"}, calls[1])
}

func TestUpgradeViaBrew_UpdateFails(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	ctx := newTestContext()
	err := upgradeViaBrew(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "brew update failed")
}

func TestUpgradeViaBrew_UpgradeFails(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	callCount := 0
	execCommand = func(name string, args ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("echo", "ok") // brew update succeeds
		}
		return exec.Command("false") // brew upgrade fails
	}

	ctx := newTestContext()
	err := upgradeViaBrew(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "brew upgrade failed")
}

func TestDoUpgrade_DelegatesToBrew(t *testing.T) {
	origDetect := detectInstallerFunc
	origExec := execCommand
	defer func() {
		detectInstallerFunc = origDetect
		execCommand = origExec
	}()

	detectInstallerFunc = func() installerType { return installerHomebrew }

	var calls [][]string
	execCommand = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, append([]string{name}, args...))
		return exec.Command("echo", "ok")
	}

	ctx := newTestContext()
	err := doUpgrade(ctx)
	assert.NoError(t, err)
	assert.Len(t, calls, 2)
	assert.Equal(t, "brew", calls[0][0])
}

func TestDoUpgrade_DirectWhenNotBrew(t *testing.T) {
	origDetect := detectInstallerFunc
	origExec := execCommand
	origStdin := stdin
	defer func() {
		detectInstallerFunc = origDetect
		execCommand = origExec
		stdin = origStdin
	}()

	detectInstallerFunc = func() installerType { return installerDirect }

	brewCalled := false
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "brew" {
			brewCalled = true
		}
		return exec.Command("echo", "ok")
	}

	// Provide "n" to the confirmation prompt so it cancels gracefully
	stdin = strings.NewReader("n\n")

	ctx := newTestContext()
	_ = doUpgrade(ctx)
	assert.False(t, brewCalled, "brew should not be called for direct installer")
}

// ---------------------------------------------------------------------------
// upgradeViaDirect end-to-end
// ---------------------------------------------------------------------------

func TestUpgradeViaDirect_FullFlow(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho upgraded\n")
	archive := createTarGzInMemory(t, "aliyun", binaryContent)

	mux := http.NewServeMux()
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("99.0.0\n"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", archive.Len()))
		w.Write(archive.Bytes())
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	origClient := httpClient
	origOSSBase := ossBaseURL
	origOSSVer := ossVersionURL
	origStdin := stdin
	origResolve := resolveExecPathFunc
	defer func() {
		httpClient = origClient
		ossBaseURL = origOSSBase
		ossVersionURL = origOSSVer
		stdin = origStdin
		resolveExecPathFunc = origResolve
	}()

	httpClient = server.Client()
	ossBaseURL = server.URL
	ossVersionURL = server.URL + "/version"
	stdin = strings.NewReader("y\n")

	targetBinary := filepath.Join(t.TempDir(), "aliyun")
	os.WriteFile(targetBinary, []byte("old"), 0755)
	resolveExecPathFunc = func() (string, error) { return targetBinary, nil }

	ctx := newTestContext()
	err := upgradeViaDirect(ctx, "3.0.0")
	assert.NoError(t, err)

	got, err := os.ReadFile(targetBinary)
	assert.NoError(t, err)
	assert.Equal(t, binaryContent, got, "binary should be replaced with new content")
}

func TestUpgradeViaDirect_AlreadyLatest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("3.0.0\n"))
	}))
	defer server.Close()

	origClient := httpClient
	origOSSBase := ossBaseURL
	origOSSVer := ossVersionURL
	defer func() {
		httpClient = origClient
		ossBaseURL = origOSSBase
		ossVersionURL = origOSSVer
	}()

	httpClient = server.Client()
	ossBaseURL = server.URL
	ossVersionURL = server.URL + "/version"

	ctx := newTestContext()
	var stdout bytes.Buffer
	ctx = cli.NewCommandContext(&stdout, &bytes.Buffer{})
	ctx.EnterCommand(NewUpgradeCommand())

	err := upgradeViaDirect(ctx, "3.0.0")
	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), "already using the latest version")
}

func TestUpgradeViaDirect_UserCancels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("99.0.0\n"))
	}))
	defer server.Close()

	origClient := httpClient
	origOSSBase := ossBaseURL
	origOSSVer := ossVersionURL
	origStdin := stdin
	defer func() {
		httpClient = origClient
		ossBaseURL = origOSSBase
		ossVersionURL = origOSSVer
		stdin = origStdin
	}()

	httpClient = server.Client()
	ossBaseURL = server.URL
	ossVersionURL = server.URL + "/version"
	stdin = strings.NewReader("n\n")

	ctx := newTestContext()
	var stdout bytes.Buffer
	ctx = cli.NewCommandContext(&stdout, &bytes.Buffer{})
	ctx.EnterCommand(NewUpgradeCommand())

	err := upgradeViaDirect(ctx, "3.0.0")
	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), "Upgrade cancelled")
}

// ---------------------------------------------------------------------------
// confirmUpgrade
// ---------------------------------------------------------------------------

func TestConfirmUpgrade_YesFlag(t *testing.T) {
	ctx := newTestContext()
	ctx.Flags().Get("yes").SetAssigned(true)

	assert.True(t, confirmUpgrade(ctx, "3.0.0", "3.1.0"))
}

func TestConfirmUpgrade_StdinYes(t *testing.T) {
	origStdin := stdin
	defer func() { stdin = origStdin }()
	stdin = strings.NewReader("y\n")

	ctx := newTestContext()
	assert.True(t, confirmUpgrade(ctx, "3.0.0", "3.1.0"))
}

func TestConfirmUpgrade_StdinNo(t *testing.T) {
	origStdin := stdin
	defer func() { stdin = origStdin }()
	stdin = strings.NewReader("n\n")

	ctx := newTestContext()
	assert.False(t, confirmUpgrade(ctx, "3.0.0", "3.1.0"))
}

func TestConfirmUpgrade_StdinEmpty(t *testing.T) {
	origStdin := stdin
	defer func() { stdin = origStdin }()
	stdin = strings.NewReader("\n")

	ctx := newTestContext()
	assert.False(t, confirmUpgrade(ctx, "3.0.0", "3.1.0"))
}

// ---------------------------------------------------------------------------
// downloadAndExtract
// ---------------------------------------------------------------------------

func TestDownloadAndExtract_Success(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho upgraded\n")
	archiveBuf := createTarGzInMemory(t, "aliyun", binaryContent)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", archiveBuf.Len()))
		w.Write(archiveBuf.Bytes())
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	var out bytes.Buffer
	binaryPath, cleanup, err := downloadAndExtract(&out, server.URL+"/aliyun-cli-linux-3.4.0-amd64.tgz", "aliyun-cli-linux-3.4.0-amd64.tgz")
	assert.NoError(t, err)
	assert.NotNil(t, cleanup)
	defer cleanup()

	got, err := os.ReadFile(binaryPath)
	assert.NoError(t, err)
	assert.Equal(t, binaryContent, got)
}

func TestDownloadAndExtract_DownloadFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	var out bytes.Buffer
	_, cleanup, err := downloadAndExtract(&out, server.URL+"/missing.tgz", "missing.tgz")
	assert.Error(t, err)
	assert.Nil(t, cleanup)
	assert.Contains(t, err.Error(), "download failed")
}

func TestDownloadAndExtract_ExtractFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not a valid archive"))
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	var out bytes.Buffer
	_, cleanup, err := downloadAndExtract(&out, server.URL+"/bad.tgz", "bad.tgz")
	assert.Error(t, err)
	assert.Nil(t, cleanup)
	assert.Contains(t, err.Error(), "extraction failed")
}

// ---------------------------------------------------------------------------
// resolveExecPath
// ---------------------------------------------------------------------------

func TestResolveExecPath(t *testing.T) {
	p, err := resolveExecPath()
	assert.NoError(t, err)
	assert.NotEmpty(t, p)

	info, err := os.Stat(p)
	assert.NoError(t, err)
	assert.False(t, info.IsDir())
}

// ---------------------------------------------------------------------------
// Build asset name (OSS path)
// ---------------------------------------------------------------------------

func TestBuildAssetName(t *testing.T) {
	name, err := buildAssetName("3.3.4")
	assert.NoError(t, err)

	switch runtime.GOOS {
	case "darwin":
		assert.Equal(t, "aliyun-cli-macosx-3.3.4-"+runtime.GOARCH+".tgz", name)
	case "linux":
		assert.Equal(t, "aliyun-cli-linux-3.3.4-"+runtime.GOARCH+".tgz", name)
	case "windows":
		assert.Equal(t, "aliyun-cli-windows-3.3.4-"+runtime.GOARCH+".zip", name)
	}
}

func TestBuildAssetName_VersionWithPrerelease(t *testing.T) {
	name, err := buildAssetName("3.4.0-beta.1")
	assert.NoError(t, err)
	assert.Contains(t, name, "3.4.0-beta.1")
}

// ---------------------------------------------------------------------------
// OSS version fetch
// ---------------------------------------------------------------------------

func TestFetchVersionFromOSS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("3.3.4\n"))
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	resp, err := httpClient.Get(server.URL)
	assert.NoError(t, err)
	defer resp.Body.Close()

	buf := make([]byte, 64)
	n, _ := resp.Body.Read(buf)
	version := strings.TrimSpace(string(buf[:n]))
	assert.Equal(t, "3.3.4", version)
}

// ---------------------------------------------------------------------------
// OSS resolution
// ---------------------------------------------------------------------------

func TestResolveFromOSS_BuildsCorrectURL(t *testing.T) {
	version := "3.3.4"
	assetName, err := buildAssetName(version)
	assert.NoError(t, err)

	expectedURL := ossBaseURL + "/" + assetName
	assert.True(t, strings.HasPrefix(expectedURL, ossBaseURL+"/aliyun-cli-"))
	assert.Contains(t, expectedURL, version)
}

// ---------------------------------------------------------------------------
// Extract: tar.gz
// ---------------------------------------------------------------------------

func TestExtractFromTarGz(t *testing.T) {
	tmpDir := t.TempDir()

	binaryContent := []byte("#!/bin/sh\necho hello\n")
	archivePath := filepath.Join(tmpDir, "test.tgz")
	createTestTarGz(t, archivePath, "aliyun", binaryContent)

	destPath := filepath.Join(tmpDir, "aliyun")
	err := extractFromTarGz(archivePath, destPath, "aliyun")
	assert.NoError(t, err)

	got, err := os.ReadFile(destPath)
	assert.NoError(t, err)
	assert.Equal(t, binaryContent, got)

	if runtime.GOOS != "windows" {
		info, _ := os.Stat(destPath)
		assert.True(t, info.Mode()&0100 != 0, "binary should be executable")
	}
}

func TestExtractFromTarGz_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tgz")
	createTestTarGz(t, archivePath, "other-binary", []byte("content"))

	err := extractFromTarGz(archivePath, filepath.Join(tmpDir, "aliyun"), "aliyun")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in archive")
}

// ---------------------------------------------------------------------------
// Extract: zip
// ---------------------------------------------------------------------------

func TestExtractFromZip(t *testing.T) {
	tmpDir := t.TempDir()

	binaryContent := []byte("MZ fake exe content")
	archivePath := filepath.Join(tmpDir, "test.zip")
	createTestZip(t, archivePath, "aliyun.exe", binaryContent)

	destPath := filepath.Join(tmpDir, "aliyun.exe")
	err := extractFromZip(archivePath, destPath, "aliyun.exe")
	assert.NoError(t, err)

	got, _ := os.ReadFile(destPath)
	assert.Equal(t, binaryContent, got)
}

func TestExtractFromZip_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")
	createTestZip(t, archivePath, "other.exe", []byte("content"))

	err := extractFromZip(archivePath, filepath.Join(tmpDir, "aliyun.exe"), "aliyun.exe")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in archive")
}

func TestExtractBinary_UnsupportedFormat(t *testing.T) {
	err := extractBinary("/tmp/test.rar", "/tmp/out", "aliyun")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported archive format")
}

// ---------------------------------------------------------------------------
// Binary replacement
// ---------------------------------------------------------------------------

func TestReplaceBinary(t *testing.T) {
	tmpDir := t.TempDir()

	oldPath := filepath.Join(tmpDir, "aliyun")
	os.WriteFile(oldPath, []byte("old binary"), 0755)

	newPath := filepath.Join(tmpDir, "aliyun.new")
	newContent := []byte("new binary")
	os.WriteFile(newPath, newContent, 0755)

	err := replaceBinary(newPath, oldPath)
	assert.NoError(t, err)

	got, _ := os.ReadFile(oldPath)
	assert.Equal(t, newContent, got)
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	content := []byte("test content")
	srcPath := filepath.Join(tmpDir, "src")
	dstPath := filepath.Join(tmpDir, "dst")

	os.WriteFile(srcPath, content, 0644)

	err := copyFile(srcPath, dstPath, 0755)
	assert.NoError(t, err)

	got, _ := os.ReadFile(dstPath)
	assert.Equal(t, content, got)

	if runtime.GOOS != "windows" {
		info, _ := os.Stat(dstPath)
		assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
	}
}

// ---------------------------------------------------------------------------
// Download
// ---------------------------------------------------------------------------

func TestDownloadFile(t *testing.T) {
	content := bytes.Repeat([]byte("x"), 1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1024")
		w.Write(content)
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "download.tgz")

	var buf bytes.Buffer
	err := downloadFile(&buf, server.URL+"/test.tgz", destPath)
	assert.NoError(t, err)

	got, _ := os.ReadFile(destPath)
	assert.Equal(t, content, got)
	assert.Contains(t, buf.String(), "Download complete")
}

// ---------------------------------------------------------------------------
// Command struct
// ---------------------------------------------------------------------------

func TestNewUpgradeCommand(t *testing.T) {
	cmd := NewUpgradeCommand()
	assert.Equal(t, "upgrade", cmd.Name)
	assert.NotNil(t, cmd.Short)
	assert.NotNil(t, cmd.Long)
	assert.NotNil(t, cmd.Run)

	yesFlag := cmd.Flags().Get("yes")
	assert.NotNil(t, yesFlag)
	assert.Equal(t, 'y', rune(yesFlag.Shorthand))
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestContext() *cli.Context {
	var stdout, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &stderr)
	cmd := NewUpgradeCommand()
	ctx.EnterCommand(cmd)
	return ctx
}

func createTestTarGz(t *testing.T, archivePath, fileName string, content []byte) {
	t.Helper()
	f, err := os.Create(archivePath)
	assert.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	tw.WriteHeader(&tar.Header{
		Name: fileName, Size: int64(len(content)), Mode: 0755, Typeflag: tar.TypeReg,
	})
	tw.Write(content)
}

func createTestZip(t *testing.T, archivePath, fileName string, content []byte) {
	t.Helper()
	f, err := os.Create(archivePath)
	assert.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	w, _ := zw.Create(fileName)
	w.Write(content)
}

func createTarGzInMemory(t *testing.T, fileName string, content []byte) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	tw.WriteHeader(&tar.Header{
		Name: fileName, Size: int64(len(content)), Mode: 0755, Typeflag: tar.TypeReg,
	})
	tw.Write(content)
	tw.Close()
	gw.Close()
	return &buf
}
