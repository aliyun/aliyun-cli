package maxc

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- test helpers ---------------------------------------------------------

// tarEntry is a single file/dir/symlink to embed in a generated tarball.
// A blank Linkname means a regular file with `Body` contents; non-blank
// Linkname means a symlink to that target (Body ignored).
type tarEntry struct {
	Name     string
	Body     string
	Mode     int64
	Linkname string // non-empty → emit a tar.TypeSymlink entry
}

// buildTarGz returns a gzipped tar archive containing the given entries.
// Top-level directory is created implicitly by file paths like "maxc/maxc".
func buildTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for _, e := range entries {
		hdr := &tar.Header{
			Name: e.Name,
			Mode: e.Mode,
			Size: int64(len(e.Body)),
		}
		switch {
		case e.Linkname != "":
			hdr.Typeflag = tar.TypeSymlink
			hdr.Linkname = e.Linkname
			hdr.Size = 0
		case strings.HasSuffix(e.Name, "/"):
			hdr.Typeflag = tar.TypeDir
			hdr.Size = 0
		default:
			hdr.Typeflag = tar.TypeReg
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("tar WriteHeader: %v", err)
		}
		if hdr.Typeflag == tar.TypeReg && len(e.Body) > 0 {
			if _, err := tw.Write([]byte(e.Body)); err != nil {
				t.Fatalf("tar Write: %v", err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar Close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gz Close: %v", err)
	}
	return buf.Bytes()
}

// sha256Hex returns the hex digest of b.
func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// patchEnvForFakeOSS installs httptest server that serves maxc-cli/{ver}/{plat}/
// objects from `objects` (key = path-after-base). Patches downloadBaseURL and
// the runtime/configure func vars to point at this fake. Returns a teardown.
type fakeOSS struct {
	srv       *httptest.Server
	requested []string // observability for tests that want to assert URL shape
}

func newFakeOSS(t *testing.T, objects map[string][]byte) *fakeOSS {
	t.Helper()
	f := &fakeOSS{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requested = append(f.requested, r.URL.Path)
		body, ok := objects[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// withCtx returns a Context configured to point at the fake OSS, with a
// per-test installDir under t.TempDir(). Patches the package-level GOOS/GOARCH
// vars to a known platform so tests don't depend on the host.
func withCtx(t *testing.T, platform string) *Context {
	t.Helper()
	parts := strings.SplitN(platform, "-", 2)
	if len(parts) != 2 {
		t.Fatalf("bad platform %q", platform)
	}

	origGOOS, origGOARCH := runtimeGOOSFunc, runtimeGOARCHFunc
	origCfg := getConfigurePathFunc
	t.Cleanup(func() {
		runtimeGOOSFunc = origGOOS
		runtimeGOARCHFunc = origGOARCH
		getConfigurePathFunc = origCfg
	})

	tmp := t.TempDir()
	runtimeGOOSFunc = func() string { return parts[0] }
	runtimeGOARCHFunc = func() string { return parts[1] }
	getConfigurePathFunc = func() string { return tmp }

	c := &Context{}
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	return c
}

// --- URL / env override tests --------------------------------------------

func TestEffectiveBaseURL_EnvOverride(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", "https://override.example.com/staging/")
	c := &Context{}
	got := c.effectiveBaseURL()
	if got != "https://override.example.com/staging" {
		t.Errorf("effectiveBaseURL = %q, want trailing slash trimmed", got)
	}
}

func TestEffectiveBaseURL_DefaultWhenEnvEmpty(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", "")
	c := &Context{}
	if got := c.effectiveBaseURL(); got != downloadBaseURL {
		t.Errorf("effectiveBaseURL = %q, want package default %q", got, downloadBaseURL)
	}
}

func TestTarballURL_Format(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", "https://b.example.com")
	c := &Context{platformKey: "darwin-arm64"}

	want := "https://b.example.com/0.3.0/darwin-arm64/maxc.tar.gz"
	if got := c.tarballURL("0.3.0"); got != want {
		t.Errorf("tarballURL = %q, want %q", got, want)
	}
	if got := c.tarballShaURL("0.3.0"); got != want+".sha256" {
		t.Errorf("tarballShaURL = %q, want %q", got, want+".sha256")
	}
	if got := c.latestVersionURL(); got != "https://b.example.com/versions/latest" {
		t.Errorf("latestVersionURL = %q", got)
	}
}

// --- download + verify + install -----------------------------------------

func TestDownloadAndInstall_HappyPath(t *testing.T) {
	tar := buildTarGz(t, []tarEntry{
		{Name: "maxc/", Mode: 0o755},
		{Name: "maxc/maxc", Body: "#!/bin/sh\necho fake\n", Mode: 0o755},
		{Name: "maxc/_internal/libpython.so", Body: "elf data", Mode: 0o644},
	})
	sha := sha256Hex(tar)

	oss := newFakeOSS(t, map[string][]byte{
		"0.3.0/linux-amd64/maxc.tar.gz":        tar,
		"0.3.0/linux-amd64/maxc.tar.gz.sha256": []byte(sha + "\n"),
	})
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", oss.srv.URL)

	c := withCtx(t, "linux-amd64")
	if err := c.downloadAndInstall("0.3.0"); err != nil {
		t.Fatalf("downloadAndInstall: %v", err)
	}

	if !fileExists(c.execFilePath) {
		t.Errorf("execFilePath %s should exist after install", c.execFilePath)
	}
	if !fileExists(filepath.Join(c.installDir, "_internal", "libpython.so")) {
		t.Errorf("nested file missing — extract was shallow")
	}

	// Execute bit preserved.
	fi, err := os.Stat(c.execFilePath)
	if err != nil {
		t.Fatalf("stat exec: %v", err)
	}
	if fi.Mode().Perm()&0o100 == 0 {
		t.Errorf("execFilePath mode = %v, expected execute bit set", fi.Mode())
	}

	// .version persisted so subsequent runs can skip the latest-pointer GET.
	verBytes, err := os.ReadFile(c.versionFilePath)
	if err != nil {
		t.Fatalf("read .version: %v", err)
	}
	if string(verBytes) != "0.3.0" {
		t.Errorf(".version = %q, want 0.3.0", string(verBytes))
	}
}

func TestDownloadAndInstall_RejectsBadSha(t *testing.T) {
	tar := buildTarGz(t, []tarEntry{
		{Name: "maxc/", Mode: 0o755},
		{Name: "maxc/maxc", Body: "real", Mode: 0o755},
	})

	oss := newFakeOSS(t, map[string][]byte{
		"0.3.0/linux-amd64/maxc.tar.gz":        tar,
		"0.3.0/linux-amd64/maxc.tar.gz.sha256": []byte("deadbeef" + strings.Repeat("0", 56)),
	})
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", oss.srv.URL)

	c := withCtx(t, "linux-amd64")
	err := c.downloadAndInstall("0.3.0")
	if err == nil {
		t.Fatal("expected sha256 mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Errorf("error = %v, want 'sha256 mismatch'", err)
	}
	// installDir must be untouched.
	if fileExists(c.installDir) {
		t.Errorf("installDir %s should not exist after bad-sha rejection", c.installDir)
	}
}

func TestDownloadAndInstall_PreservesOldOnNetworkFail(t *testing.T) {
	// Pre-populate installDir as if an older version was installed.
	c := withCtx(t, "linux-amd64")
	if err := os.MkdirAll(c.installDir, 0o755); err != nil {
		t.Fatalf("seed installDir: %v", err)
	}
	oldExec := c.execFilePath
	if err := os.WriteFile(oldExec, []byte("OLD"), 0o755); err != nil {
		t.Fatalf("seed exec: %v", err)
	}

	// Fake OSS that 404s — download will fail before any rename happens.
	oss := newFakeOSS(t, map[string][]byte{})
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", oss.srv.URL)

	err := c.downloadAndInstall("9.9.9")
	if err == nil {
		t.Fatal("expected download error, got nil")
	}

	// Old binary must still be there with original contents.
	data, readErr := os.ReadFile(oldExec)
	if readErr != nil {
		t.Fatalf("old exec missing: %v", readErr)
	}
	if string(data) != "OLD" {
		t.Errorf("old exec contents = %q, want OLD", string(data))
	}
}

// --- extractTarGz security & feature tests --------------------------------

func TestExtractTarGz_RejectsTraversalPath(t *testing.T) {
	tar := buildTarGz(t, []tarEntry{
		{Name: "../escape", Body: "x", Mode: 0o644},
	})
	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "evil.tar.gz")
	if err := os.WriteFile(tarPath, tar, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(tmp, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	err := extractTarGz(tarPath, dest)
	if err == nil {
		t.Fatal("expected unsafe-path error, got nil")
	}
	if !strings.Contains(err.Error(), "unsafe path") {
		t.Errorf("error = %v, want 'unsafe path'", err)
	}
}

func TestExtractTarGz_PreservesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks on windows need admin privileges in CI")
	}
	tar := buildTarGz(t, []tarEntry{
		{Name: "pkg/", Mode: 0o755},
		{Name: "pkg/real.txt", Body: "hello", Mode: 0o644},
		{Name: "pkg/link.txt", Linkname: "real.txt"},
	})
	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "t.tar.gz")
	if err := os.WriteFile(tarPath, tar, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(tmp, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := extractTarGz(tarPath, dest); err != nil {
		t.Fatalf("extract: %v", err)
	}

	linkPath := filepath.Join(dest, "pkg", "link.txt")
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat link: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link.txt is not a symlink (mode=%v)", fi.Mode())
	}
	// And dereferencing the symlink should yield "hello".
	body, err := os.ReadFile(linkPath)
	if err != nil {
		t.Fatalf("readfile via symlink: %v", err)
	}
	if string(body) != "hello" {
		t.Errorf("symlink dereferenced = %q, want 'hello'", string(body))
	}
}

// --- regression guard: make sure httptest interplay isn't broken ----------

func TestHttpGetToFile_PropagatesNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out")
	err := httpGetToFile(srv.URL+"/anything", dest)
	if err == nil {
		t.Fatal("expected non-2xx error, got nil")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("error = %v, want 'status 500'", err)
	}
}

// Compile-time sanity: import "fmt" / "io" actually get used so go vet stays
// quiet even if other test files churn.
var _ = fmt.Sprintf
var _ io.Reader = (*bytes.Buffer)(nil)
