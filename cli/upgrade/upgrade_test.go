package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
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

func TestIsNewer_InvalidSemverStringFallback(t *testing.T) {
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"custom-build-1", "custom-build-2", true},
		{"custom-build-1", "custom-build-1", false},
		{"same-opaque", "same-opaque", false},
		{"3.0.0", "dev-channel", true},
		{"dev", "3.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			assert.Equal(t, tt.want, isNewer(tt.current, tt.latest),
				"non-semver path should use raw string inequality")
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

func TestInstallerTypeFromExecPathLower(t *testing.T) {
	tests := []struct {
		path string
		want installerType
	}{
		{"/opt/homebrew/opt/aliyun-cli/bin/aliyun", installerHomebrew},
		{"/usr/local/homebrew/bin/aliyun", installerHomebrew},
		{"/usr/local/Cellar/aliyun-cli/3.0.0/bin/aliyun", installerHomebrew},
		{"/home/linuxbrew/.linuxbrew/bin/aliyun", installerLinuxbrew},
		{"/tmp/aliyun", installerDirect},
		{"/linuxbrew/prefix/homebrew/name", installerLinuxbrew},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			lower := strings.ToLower(tt.path)
			assert.Equal(t, tt.want, installerTypeFromExecPathLower(lower))
		})
	}
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
	assetName, binaryName, archive := createUpgradeArchiveInMemory(t, "99.0.0", binaryContent)
	priv := useTestPublisher(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("99.0.0\n"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body := archive.Bytes()
		if strings.HasSuffix(r.URL.Path, ".minisig") {
			doc, err := publishsig.Sign(priv, assetName, bytes.NewReader(body))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			body = doc
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.Write(body)
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

	targetBinary := filepath.Join(t.TempDir(), binaryName)
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

func TestFetchVersionList(t *testing.T) {
	catalog := versionCatalogBody(t,
		publishedVersion{Version: "3.5.1", PublicKeys: []string{"1"}, Description: "publisher key generation 1"},
		publishedVersion{Version: "3.5.2", PublicKeys: []string{"1"}, Description: "publisher key generation 1"},
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/missing/versions"):
			http.NotFound(w, r)
		case strings.HasSuffix(r.URL.Path, "/bad/versions"):
			w.WriteHeader(http.StatusBadGateway)
		default:
			w.Write(catalog)
		}
	}))
	defer server.Close()
	origClient, origBase := httpClient, ossBaseURL
	defer func() {
		httpClient, ossBaseURL = origClient, origBase
	}()
	httpClient = server.Client()

	ossBaseURL = server.URL
	listed, present, err := fetchVersionList()
	assert.NoError(t, err)
	assert.True(t, present)
	assert.Equal(t, "3.5.1", listed[0].Version)
	assert.Equal(t, []string{"1"}, listed[1].PublicKeys)

	ossBaseURL = server.URL + "/missing"
	listed, present, err = fetchVersionList()
	assert.NoError(t, err)
	assert.False(t, present)
	assert.Nil(t, listed)

	ossBaseURL = server.URL + "/bad"
	_, _, err = fetchVersionList()
	assert.ErrorContains(t, err, "status 502")

	_, err = parseVersionCatalog([]byte(`{"format":"other"}`))
	assert.ErrorContains(t, err, "unsupported format")
}

func TestUpgradeViaDirect_HistoryUnavailableFallsBackToPointer(t *testing.T) {
	assetName, binaryName, archive := createUpgradeArchiveInMemory(t, "4.0.0", []byte("from-pointer"))
	priv := useTestPublisher(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/versions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("4.0.0\n"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body := archive.Bytes()
		if strings.HasSuffix(r.URL.Path, ".minisig") {
			doc, err := publishsig.Sign(priv, assetName, bytes.NewReader(body))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			body = doc
		}
		w.Write(body)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	origClient, origBase, origVer, origStdin, origResolve := httpClient, ossBaseURL, ossVersionURL, stdin, resolveExecPathFunc
	defer func() {
		httpClient, ossBaseURL, ossVersionURL, stdin, resolveExecPathFunc = origClient, origBase, origVer, origStdin, origResolve
	}()
	httpClient = server.Client()
	ossBaseURL = server.URL
	ossVersionURL = server.URL + "/version"
	stdin = strings.NewReader("y\n")
	target := filepath.Join(t.TempDir(), binaryName)
	os.WriteFile(target, []byte("old"), 0755)
	resolveExecPathFunc = func() (string, error) { return target, nil }

	var stdout bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &bytes.Buffer{})
	ctx.EnterCommand(NewUpgradeCommand())
	err := upgradeViaDirect(ctx, "3.0.0")
	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), "Version history is unavailable")
	got, err := os.ReadFile(target)
	assert.NoError(t, err)
	assert.Equal(t, []byte("from-pointer"), got)
}

func TestUpgradeViaDirect_RejectsEveryCandidate(t *testing.T) {
	_, _, archive := createUpgradeArchiveInMemory(t, "3.2.0", []byte("nope"))
	mux := http.NewServeMux()
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("3.2.0\n"))
	})
	mux.HandleFunc("/versions", func(w http.ResponseWriter, r *http.Request) {
		w.Write(versionCatalogBody(t,
			publishedVersion{Version: "3.1.0", PublicKeys: []string{"1"}},
			publishedVersion{Version: "3.2.0", PublicKeys: []string{"1"}},
		))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body := archive.Bytes()
		if strings.HasSuffix(r.URL.Path, ".minisig") {
			body = []byte("not-a-signature")
		}
		w.Write(body)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	origClient, origBase, origVer, origStdin := httpClient, ossBaseURL, ossVersionURL, stdin
	defer func() {
		httpClient, ossBaseURL, ossVersionURL, stdin = origClient, origBase, origVer, origStdin
	}()
	httpClient = server.Client()
	ossBaseURL = server.URL
	ossVersionURL = server.URL + "/version"
	stdin = strings.NewReader("y\n")

	var stdout bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &bytes.Buffer{})
	ctx.EnterCommand(NewUpgradeCommand())
	err := upgradeViaDirect(ctx, "3.0.0")
	assert.ErrorContains(t, err, "publisher signature rejected")
	assert.NotContains(t, stdout.String(), "Skipped")
	assert.Contains(t, stdout.String(), "3.2.0")
	assert.NotContains(t, stdout.String(), "3.1.0")
}

func TestSelectUpgradeVersion(t *testing.T) {
	listed := []publishedVersion{
		{Version: "1.0.0", PublicKeys: []string{"1"}},
		{Version: "3.1.0", PublicKeys: []string{"1"}},
		{Version: "3.1.5", PublicKeys: []string{"2"}},
		{Version: "3.1.8", PublicKeys: []string{"1", "2"}},
		{Version: "3.2.0", PublicKeys: []string{"2"}},
		{Version: "9.9.9", PublicKeys: []string{"1"}},
		{Version: "not-a-version", PublicKeys: []string{"1"}},
	}
	assert.Equal(t, "3.1.8", selectUpgradeVersion("3.0.0", "3.2.0", "1", listed, true))
	assert.Equal(t, "3.2.0", selectUpgradeVersion("3.0.0", "3.2.0", "2", listed, true))
	assert.Equal(t, "", selectUpgradeVersion("3.2.0", "3.1.0", "1", []publishedVersion{
		{Version: "1.0.0", PublicKeys: []string{"1"}},
		{Version: "3.1.0", PublicKeys: []string{"1"}},
		{Version: "9.9.9", PublicKeys: []string{"1"}},
	}, true))
	assert.Equal(t, "3.2.0", selectUpgradeVersion("3.0.0", "3.2.0", "1", nil, false))
	assert.Equal(t, "", selectUpgradeVersion("3.2.0", "3.2.0", "1", nil, false))
	assert.Equal(t, "", selectUpgradeVersion("3.0.0", "latest", "1", listed, true))
	assert.Equal(t, "", selectUpgradeVersion("dev", "3.4.0", "1", listed, true))
	_, ok := canonicalUpgradeVersion("3.5")
	assert.False(t, ok)
	_, ok = canonicalUpgradeVersion("3.5.1/evil")
	assert.False(t, ok)
}

func TestUpgradeViaDirect_InstallsNewestVerifiedAndSkipsOlderSigned(t *testing.T) {
	latestName, binaryName, latestArchive := createUpgradeArchiveInMemory(t, "3.2.0", []byte("latest-bytes"))
	midName, _, midArchive := createUpgradeArchiveInMemory(t, "3.1.0", []byte("mid-bytes"))
	oldName, _, oldArchive := createUpgradeArchiveInMemory(t, "1.0.0", []byte("old-bytes"))
	priv := useTestPublisher(t)
	_, other, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)

	seen := map[string]int{}
	mux := http.NewServeMux()
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("3.2.0\n"))
	})
	mux.HandleFunc("/versions", func(w http.ResponseWriter, r *http.Request) {
		w.Write(versionCatalogBody(t,
			publishedVersion{Version: "1.0.0", PublicKeys: []string{"1"}},
			publishedVersion{Version: "3.1.0", PublicKeys: []string{"1"}},
			publishedVersion{Version: "3.2.0", PublicKeys: []string{"2"}},
		))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path]++
		var body []byte
		var signWith ed25519.PrivateKey
		var asset string
		switch {
		case strings.Contains(r.URL.Path, latestName):
			body, signWith, asset = latestArchive.Bytes(), other, latestName
		case strings.Contains(r.URL.Path, midName):
			body, signWith, asset = midArchive.Bytes(), priv, midName
		case strings.Contains(r.URL.Path, oldName):
			body, signWith, asset = oldArchive.Bytes(), priv, oldName
		default:
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, ".minisig") {
			doc, err := publishsig.Sign(signWith, asset, bytes.NewReader(body))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			body = doc
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.Write(body)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	origClient, origBase, origVer, origStdin, origResolve := httpClient, ossBaseURL, ossVersionURL, stdin, resolveExecPathFunc
	defer func() {
		httpClient, ossBaseURL, ossVersionURL, stdin, resolveExecPathFunc = origClient, origBase, origVer, origStdin, origResolve
	}()
	httpClient = server.Client()
	ossBaseURL = server.URL
	ossVersionURL = server.URL + "/version"
	stdin = strings.NewReader("y\n")
	target := filepath.Join(t.TempDir(), binaryName)
	os.WriteFile(target, []byte("current"), 0755)
	resolveExecPathFunc = func() (string, error) { return target, nil }

	var stdout bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &bytes.Buffer{})
	ctx.EnterCommand(NewUpgradeCommand())
	err = upgradeViaDirect(ctx, "3.0.0")
	assert.NoError(t, err)
	got, err := os.ReadFile(target)
	assert.NoError(t, err)
	assert.Equal(t, []byte("mid-bytes"), got)
	assert.Contains(t, stdout.String(), "publisher key generation "+publishsig.SignatureVersion)
	assert.Contains(t, stdout.String(), "Run `aliyun upgrade` again")
	assert.Zero(t, seen["/"+oldName])
	assert.Zero(t, seen["/"+oldName+".minisig"])
	assert.Zero(t, seen["/"+latestName], "a newer release with another signature version is not downloaded")
	assert.Zero(t, seen["/"+latestName+".minisig"])
}

func TestUpgradeViaDirect_DoesNotInstallOlderSignedRelease(t *testing.T) {
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		switch r.URL.Path {
		case "/version":
			w.Write([]byte("3.1.0\n"))
		case "/versions":
			w.Write(versionCatalogBody(t,
				publishedVersion{Version: "1.0.0", PublicKeys: []string{"1"}},
				publishedVersion{Version: "3.1.0", PublicKeys: []string{"1"}},
				publishedVersion{Version: "9.9.9", PublicKeys: []string{"2"}},
			))
		default:
			t.Errorf("unexpected download %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	origClient, origBase, origVer := httpClient, ossBaseURL, ossVersionURL
	defer func() {
		httpClient, ossBaseURL, ossVersionURL = origClient, origBase, origVer
	}()
	httpClient = server.Client()
	ossBaseURL = server.URL
	ossVersionURL = server.URL + "/version"

	var stdout bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &bytes.Buffer{})
	ctx.EnterCommand(NewUpgradeCommand())
	err := upgradeViaDirect(ctx, "3.2.0")
	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), "already using the latest version")
	for _, path := range requested {
		assert.NotContains(t, path, "aliyun-cli-")
	}
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
	assetName, _, archiveBuf := createUpgradeArchiveInMemory(t, "3.4.0", binaryContent)
	priv := useTestPublisher(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := archiveBuf.Bytes()
		if strings.HasSuffix(r.URL.Path, ".minisig") {
			doc, err := publishsig.Sign(priv, assetName, bytes.NewReader(body))
			if err != nil {
				t.Errorf("sign: %v", err)
				return
			}
			body = doc
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.Write(body)
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	var out bytes.Buffer
	binaryPath, cleanup, err := downloadAndExtract(&out, server.URL+"/"+assetName, assetName)
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
	priv := useTestPublisher(t)
	body := []byte("not a valid archive")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := body
		if strings.HasSuffix(r.URL.Path, ".minisig") {
			doc, err := publishsig.Sign(priv, "bad.tgz", bytes.NewReader(body))
			if err != nil {
				t.Errorf("sign: %v", err)
				return
			}
			payload = doc
		}
		w.Write(payload)
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

func TestDownloadAndExtract_RejectsBadSignature(t *testing.T) {
	useTestPublisher(t)
	binaryContent := []byte("#!/bin/sh\necho upgraded\n")
	assetName, _, archiveBuf := createUpgradeArchiveInMemory(t, "3.4.0", binaryContent)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".minisig") {
			w.Write([]byte("aliyun-cli-publish-signature v1\nasset: " + assetName + "\nkey: aaaaaaaaaaaaaaaa\nsignature: AAAA\n"))
			return
		}
		w.Write(archiveBuf.Bytes())
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	var out bytes.Buffer
	_, cleanup, err := downloadAndExtract(&out, server.URL+"/"+assetName, assetName)
	assert.Error(t, err)
	assert.Nil(t, cleanup)
	assert.Contains(t, err.Error(), "publisher signature rejected")
}

func TestDownloadAndExtract_RejectsSignatureForOtherAsset(t *testing.T) {
	priv := useTestPublisher(t)
	binaryContent := []byte("#!/bin/sh\necho upgraded\n")
	assetName, _, archiveBuf := createUpgradeArchiveInMemory(t, "3.4.0", binaryContent)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".minisig") {
			doc, err := publishsig.Sign(priv, "other-platform.tgz", bytes.NewReader(archiveBuf.Bytes()))
			if err != nil {
				t.Errorf("sign: %v", err)
				return
			}
			w.Write(doc)
			return
		}
		w.Write(archiveBuf.Bytes())
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	var out bytes.Buffer
	_, cleanup, err := downloadAndExtract(&out, server.URL+"/"+assetName, assetName)
	assert.Error(t, err)
	assert.Nil(t, cleanup)
	assert.Contains(t, err.Error(), "publisher signature rejected")
}

func TestDownloadAndExtract_SignatureDownloadFails(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho upgraded\n")
	assetName, _, archiveBuf := createUpgradeArchiveInMemory(t, "3.4.0", binaryContent)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".minisig") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(archiveBuf.Bytes())
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	var out bytes.Buffer
	_, cleanup, err := downloadAndExtract(&out, server.URL+"/"+assetName, assetName)
	assert.Error(t, err)
	assert.Nil(t, cleanup)
	assert.Contains(t, err.Error(), "publisher signature download failed")
}

func TestDownloadAndExtract_RejectsOversizedSignature(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho upgraded\n")
	assetName, _, archiveBuf := createUpgradeArchiveInMemory(t, "3.4.0", binaryContent)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".minisig") {
			w.Write(bytes.Repeat([]byte("a"), 4097))
			return
		}
		w.Write(archiveBuf.Bytes())
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	var out bytes.Buffer
	_, cleanup, err := downloadAndExtract(&out, server.URL+"/"+assetName, assetName)
	assert.Error(t, err)
	assert.Nil(t, cleanup)
	assert.Contains(t, err.Error(), "document is too large")
}

func TestDownloadAndExtract_SignatureReadFails(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho upgraded\n")
	assetName, _, archiveBuf := createUpgradeArchiveInMemory(t, "3.4.0", binaryContent)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := archiveBuf.Bytes()
		if strings.HasSuffix(r.URL.Path, ".minisig") {
			body = []byte("sig")
		}
		w.Write(body)
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()
	origRead := readSignatureFile
	readSignatureFile = func(string) ([]byte, error) { return nil, fmt.Errorf("unreadable") }
	defer func() { readSignatureFile = origRead }()

	var out bytes.Buffer
	_, cleanup, err := downloadAndExtract(&out, server.URL+"/"+assetName, assetName)
	assert.Error(t, err)
	assert.Nil(t, cleanup)
	assert.Contains(t, err.Error(), "publisher signature download failed")
}

func useTestPublisher(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	prev := publisherKeys
	publisherKeys = []ed25519.PublicKey{pub}
	t.Cleanup(func() { publisherKeys = prev })
	return priv
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

func TestResolveExecPath_ExecutableError(t *testing.T) {
	orig := osExecutable
	defer func() { osExecutable = orig }()
	osExecutable = func() (string, error) { return "", fmt.Errorf("no executable") }

	_, err := resolveExecPath()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get executable path")
}

func TestResolveExecPath_EvalSymlinksError(t *testing.T) {
	origExe, origEval := osExecutable, evalSymlinks
	defer func() {
		osExecutable = origExe
		evalSymlinks = origEval
	}()
	osExecutable = func() (string, error) { return "/fake/aliyun", nil }
	evalSymlinks = func(string) (string, error) { return "", fmt.Errorf("symlink error") }

	_, err := resolveExecPath()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve executable path")
}

func TestDetectInstaller_ExecutableError(t *testing.T) {
	orig := osExecutable
	defer func() { osExecutable = orig }()
	osExecutable = func() (string, error) { return "", fmt.Errorf("no executable") }

	assert.Equal(t, installerDirect, detectInstaller())
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

func TestFetchVersionFromOSS_HTTPGetFails(t *testing.T) {
	origClient, origURL := httpClient, ossVersionURL
	defer func() {
		httpClient = origClient
		ossVersionURL = origURL
	}()

	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("connection refused")
	})}
	ossVersionURL = "http://127.0.0.1:9/version"

	_, err := fetchVersionFromOSS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reach OSS")
}

func TestFetchVersionFromOSS_StatusNotOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	origClient, origURL := httpClient, ossVersionURL
	defer func() {
		httpClient = origClient
		ossVersionURL = origURL
	}()
	httpClient = server.Client()
	ossVersionURL = server.URL + "/version"

	_, err := fetchVersionFromOSS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestFetchVersionFromOSS_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	origClient, origURL := httpClient, ossVersionURL
	defer func() {
		httpClient = origClient
		ossVersionURL = origURL
	}()
	httpClient = server.Client()
	ossVersionURL = server.URL + "/version"

	_, err := fetchVersionFromOSS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty version from OSS")
}

func TestFetchVersionFromOSS_ReadBodyFails(t *testing.T) {
	origClient, origURL := httpClient, ossVersionURL
	defer func() {
		httpClient = origClient
		ossVersionURL = origURL
	}()

	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(failingReader{}),
			Request:    &http.Request{},
		}, nil
	})}
	ossVersionURL = "http://unused.example/version"

	_, err := fetchVersionFromOSS()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated body read failure")
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

func TestReplaceBinary_CurrentMissing(t *testing.T) {
	tmpDir := t.TempDir()
	current := filepath.Join(tmpDir, "missing-binary")
	newPath := filepath.Join(tmpDir, "new")
	os.WriteFile(newPath, []byte("x"), 0644)

	err := replaceBinary(newPath, current)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stat current binary")
}

func TestReplaceBinary_NewMissing(t *testing.T) {
	tmpDir := t.TempDir()
	current := filepath.Join(tmpDir, "aliyun")
	os.WriteFile(current, []byte("old"), 0755)
	newPath := filepath.Join(tmpDir, "does-not-exist")

	err := replaceBinary(newPath, current)
	assert.Error(t, err)

	got, errRead := os.ReadFile(current)
	assert.NoError(t, errRead)
	assert.Equal(t, []byte("old"), got, "current binary should be unchanged on failure")
}

func TestReplaceBinary_Unix_NoStaleTempFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows replace path uses .aliyun.upgrade.tmp")
	}
	tmpDir := t.TempDir()
	current := filepath.Join(tmpDir, "aliyun")
	newPath := filepath.Join(tmpDir, "aliyun.new")
	want := []byte("replaced")
	os.WriteFile(current, []byte("before"), 0755)
	os.WriteFile(newPath, want, 0755)

	tmpArtifact := filepath.Join(tmpDir, ".aliyun.upgrade.tmp")
	assert.NoError(t, replaceBinary(newPath, current))

	_, err := os.Stat(tmpArtifact)
	assert.True(t, os.IsNotExist(err), "temp swap file should be removed after success")

	got, _ := os.ReadFile(current)
	assert.Equal(t, want, got)
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

func TestDownloadFile_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "missing.tgz")
	var buf bytes.Buffer

	err := downloadFile(&buf, server.URL+"/missing.tgz", destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestDownloadFile_CreateFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("x"))
	}))
	defer server.Close()

	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	tmpDir := t.TempDir()
	destPath := tmpDir

	var buf bytes.Buffer
	err := downloadFile(&buf, server.URL+"/file.tgz", destPath)
	assert.Error(t, err)
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

	origDetect := detectInstallerFunc
	origExec := execCommand
	detectInstallerFunc = func() installerType { return installerHomebrew }
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}
	t.Cleanup(func() {
		detectInstallerFunc = origDetect
		execCommand = origExec
	})
	err := cmd.Run(newTestContext(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "brew update failed")
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func versionCatalogBody(t *testing.T, versions ...publishedVersion) []byte {
	t.Helper()
	body, err := json.Marshal(struct {
		Format   string             `json:"format"`
		Versions []publishedVersion `json:"versions"`
	}{Format: publishsig.VersionsFormat, Versions: versions})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

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

func createZipInMemory(t *testing.T, fileName string, content []byte) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf
}

func createUpgradeArchiveInMemory(t *testing.T, version string, content []byte) (assetName, binaryName string, archive *bytes.Buffer) {
	t.Helper()
	assetName, err := buildAssetName(version)
	if err != nil {
		t.Fatal(err)
	}
	binaryName = "aliyun"
	if runtime.GOOS == "windows" {
		binaryName = "aliyun.exe"
		return assetName, binaryName, createZipInMemory(t, binaryName, content)
	}
	return assetName, binaryName, createTarGzInMemory(t, binaryName, content)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, fmt.Errorf("simulated body read failure") }
