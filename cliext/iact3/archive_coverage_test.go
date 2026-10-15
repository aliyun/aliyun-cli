package iact3

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func archiveBinaryName() string { return "iact3" }

func makeBinaryTar(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, h := range []tar.Header{
		{Name: "../outside", Typeflag: tar.TypeReg, Size: 3},
		{Name: ".", Typeflag: tar.TypeDir},
		{Name: "folder/", Typeflag: tar.TypeDir},
		{Name: "link", Linkname: "../outside", Typeflag: tar.TypeSymlink},
		{Name: "README", Typeflag: tar.TypeReg, Size: 3},
		{Name: archiveBinaryName(), Typeflag: tar.TypeReg, Size: 3},
		{Name: "other/" + archiveBinaryName(), Typeflag: tar.TypeReg, Size: 3},
		{Name: "device", Typeflag: tar.TypeChar},
	} {
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			if _, err := tw.Write([]byte("bin")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestArchivePathsAndExtraction(t *testing.T) {
	for _, name := range []string{"", "a\x00b", "/tmp/iact3", `\\server\iact3`, `C:\iact3`, "../iact3", "foo/../../iact3"} {
		if _, err := sanitizeArchivePath(name); err == nil {
			t.Errorf("unsafe path %q accepted", name)
		}
	}
	for name, want := range map[string]string{`././folder\iact3`: "folder/iact3", ".": "", "folder/../iact3": "iact3"} {
		got, err := sanitizeArchivePath(name)
		if err != nil || got != want {
			t.Errorf("normalize %q = %q, %v", name, got, err)
		}
	}
	dest := t.TempDir()
	if _, err := safeJoinUnderDir(dest, "../outside"); err == nil {
		t.Fatal("path escaped destination")
	}
	if got, err := safeJoinUnderDir(dest, "folder/iact3"); err != nil || got != filepath.Join(dest, "folder", "iact3") {
		t.Fatalf("safe path = %q, %v", got, err)
	}
	src := filepath.Join(t.TempDir(), "binary.tar.gz")
	if err := os.WriteFile(src, makeBinaryTar(t), 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(src, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, archiveBinaryName()))
	if err != nil || string(got) != "bin" {
		t.Fatalf("binary = %q, %v", got, err)
	}
	for _, name := range []string{"outside", "README", "link"} {
		if fileExists(filepath.Join(dest, name)) {
			t.Errorf("unwanted entry extracted: %s", name)
		}
	}
	if err := extractTarGz(filepath.Join(dest, "missing"), dest); err == nil {
		t.Fatal("missing archive accepted")
	}
	if err := os.WriteFile(src, []byte("invalid gzip"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(src, dest); err == nil {
		t.Fatal("invalid gzip accepted")
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte("invalid tar"))
	_ = gz.Close()
	if err := os.WriteFile(src, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(src, dest); err == nil {
		t.Fatal("invalid tar accepted")
	}
}

type archiveReadError struct{}

func (archiveReadError) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (archiveReadError) Close() error             { return nil }

func TestDownloadArchiveFailurePaths(t *testing.T) {
	old := httpGetFunc
	t.Cleanup(func() { httpGetFunc = old })
	for _, tc := range []struct {
		name           string
		status         int
		body           io.ReadCloser
		requestErr     error
		destName, want string
	}{
		{name: "network", requestErr: errors.New("offline"), want: "offline"},
		{name: "status", status: 404, body: io.NopCloser(strings.NewReader("")), want: "404"},
		{name: "create", status: 200, body: io.NopCloser(strings.NewReader("")), destName: "missing/archive.tar.gz", want: "create"},
		{name: "read", status: 200, body: archiveReadError{}, want: "write"},
		{name: "invalid archive", status: 200, body: io.NopCloser(strings.NewReader("bad")), want: "extract"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			httpGetFunc = func(string) (*http.Response, error) {
				if tc.requestErr != nil {
					return nil, tc.requestErr
				}
				return &http.Response{StatusCode: tc.status, Body: tc.body}, nil
			}
			name := tc.destName
			if name == "" {
				name = "archive.tar.gz"
			}
			dir := t.TempDir()
			err := DownloadAndExtract("https://example.invalid/archive", filepath.Join(dir, name), filepath.Join(dir, archiveBinaryName()))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestDownloadArchiveInstallsBinary(t *testing.T) {
	old := httpGetFunc
	t.Cleanup(func() { httpGetFunc = old })
	data := makeBinaryTar(t)
	httpGetFunc = func(string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data))}, nil
	}
	dir := t.TempDir()
	dest := filepath.Join(dir, "archive.tar.gz")
	exe := filepath.Join(dir, archiveBinaryName())
	if err := os.WriteFile(exe, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := DownloadAndExtract("https://example.invalid/archive", dest, exe); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(exe)
	if err != nil || string(got) != "bin" {
		t.Fatalf("installed binary = %q, %v", got, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(exe)
		if err != nil || info.Mode()&0111 == 0 {
			t.Fatalf("binary not executable: %v, %v", info, err)
		}
	}
	if fileExists(dest) {
		t.Fatal("download archive was not removed")
	}
}
