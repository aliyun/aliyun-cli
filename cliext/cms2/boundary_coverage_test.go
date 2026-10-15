package cms2

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

type binaryBoundaryTransport func(*http.Request) (*http.Response, error)

func (f binaryBoundaryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type binaryBoundaryBodyError struct{}

func (binaryBoundaryBodyError) Read([]byte) (int, error) { return 0, errors.New("body failed") }
func (binaryBoundaryBodyError) Close() error             { return nil }

func TestBinaryDownloadFilesystemBoundaries(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	for _, tc := range []struct {
		name   string
		status int
		body   io.ReadCloser
		reqErr error
		dest   string
		want   string
	}{
		{name: "body read", status: 200, body: binaryBoundaryBodyError{}, want: "body failed"}, {name: "bad destination", status: 200, body: io.NopCloser(strings.NewReader("binary")), dest: "missing/binary", want: "create"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			http.DefaultTransport = binaryBoundaryTransport(func(*http.Request) (*http.Response, error) {
				if tc.reqErr != nil {
					return nil, tc.reqErr
				}
				return &http.Response{StatusCode: tc.status, Body: tc.body}, nil
			})
			dest := tc.dest
			if dest == "" {
				dest = "binary"
			}
			err := downloadFile("https://example.invalid/binary", filepath.Join(t.TempDir(), dest))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("download error=%v", err)
			}
		})
	}
}

func TestBinaryInstallAndUpdateFailureBoundaries(t *testing.T) {
	oldDownload, oldRemote, oldOS := downloadFileFunc, getLatestCms2VersionFunc, runtimeGOOSFunc
	t.Cleanup(func() { downloadFileFunc = oldDownload; getLatestCms2VersionFunc = oldRemote; runtimeGOOSFunc = oldOS })
	t.Setenv("ALIBABA_CLOUD_CMS2_EXEC_PATH", "")
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	dir := t.TempDir()
	c := NewContext(ctx)
	c.execFilePath = filepath.Join(dir, "binary")
	c.versionFilePath = filepath.Join(dir, "version")
	c.checkVersionCacheFilePath = filepath.Join(dir, "cache")
	runtimeGOOSFunc = func() string { return "linux" }
	downloadFileFunc = func(string, string) error { return errors.New("download failed") }
	getLatestCms2VersionFunc = func() (string, error) { return "", errors.New("offline") }
	if err := c.EnsureInstalledAndUpdated(); err == nil {
		t.Fatal("first installation lookup failure lost")
	}
	c.installed = true
	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("existing installation blocked by version lookup: %v", err)
	}
	if err := c.Install(); err == nil || !strings.Contains(err.Error(), "download failed") {
		t.Fatalf("download install error=%v", err)
	}
	downloadFileFunc = func(string, string) error { return nil }
	if err := c.Install(); err == nil {
		t.Fatal("download succeeded without creating executable")
	}
	runtimeGOOSFunc = func() string { return "windows" }
	c.execFilePath = filepath.Join(dir, "nonempty")
	if err := os.Mkdir(c.execFilePath, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.execFilePath, "child"), []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	downloadFileFunc = func(_ string, dest string) error { return os.WriteFile(dest, []byte("new"), 0600) }
	if err := c.Install(); err == nil {
		t.Fatal("nonempty destination directory overwritten")
	}
	c.versionFilePath = filepath.Join(dir, "missing", "version")
	if err := c.SaveLocalVersion(); err == nil {
		t.Fatal("unwritable version accepted")
	}
	c.checkVersionCacheFilePath = c.versionFilePath
	if err := c.UpdateCheckCacheTime(); err == nil {
		t.Fatal("unwritable cache accepted")
	}
	c.versionFilePath = dir
	if err := c.GetLocalVersion(); err == nil {
		t.Fatal("version directory accepted")
	}
	c.checkVersionCacheFilePath = dir
	if !c.NeedCheckVersion() {
		t.Fatal("unreadable cache not stale")
	}
}
