package computenestutil

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

type pythonTransport func(*http.Request) (*http.Response, error)

func (f pythonTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type pythonBodyError struct{}

func (pythonBodyError) Read([]byte) (int, error) { return 0, errors.New("body read failed") }
func (pythonBodyError) Close() error             { return nil }

func TestWheelDownloadFailureBoundaries(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	for _, tc := range []struct {
		name        string
		failRequest int
		status      int
		bodyError   bool
		empty       bool
	}{
		{name: "version unavailable", failRequest: 1, status: 404}, {name: "version network", failRequest: 1, status: 0}, {name: "version read", failRequest: 1, status: 200, bodyError: true}, {name: "version empty", failRequest: 1, status: 200, empty: true}, {name: "wheel unavailable", failRequest: 2, status: 404}, {name: "wheel network", failRequest: 2, status: 0}, {name: "wheel read", failRequest: 2, status: 200, bodyError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			http.DefaultTransport = pythonTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				if requests != tc.failRequest {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("2.0"))}, nil
				}
				if tc.status == 0 {
					return nil, errors.New("offline")
				}
				var body io.ReadCloser = io.NopCloser(strings.NewReader("content"))
				if tc.bodyError {
					body = pythonBodyError{}
				}
				if tc.empty {
					body = io.NopCloser(strings.NewReader(" \n"))
				}
				return &http.Response{StatusCode: tc.status, Body: body}, nil
			})
			if path, err := c.downloadWhlFromOSS(); err == nil {
				t.Fatalf("download succeeded unexpectedly: %q", path)
			}
		})
	}
}

func TestPythonArchiveFilesystemFailures(t *testing.T) {
	dir := t.TempDir()
	block := filepath.Join(dir, "block")
	if err := os.WriteFile(block, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "directory/", Typeflag: tar.TypeDir, Mode: 0755}); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: "directory/file", Typeflag: tar.TypeReg, Mode: 0600, Size: 4}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("data")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(archive.Bytes(), block); err == nil {
		t.Fatal("tar destination file accepted")
	}
	if err := extractTarGz([]byte("invalid"), dir); err == nil {
		t.Fatal("invalid gzip accepted")
	}
	var zipArchive bytes.Buffer
	zw := zip.NewWriter(&zipArchive)
	f, err := zw.Create("directory/file")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("data")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := extractZip(zipArchive.Bytes(), block); err == nil {
		t.Fatal("zip destination file accepted")
	}
	if err := extractZip([]byte("invalid"), dir); err == nil {
		t.Fatal("invalid zip accepted")
	}
	if _, err := sanitizeArchivePath(dir, "../outside"); err == nil {
		t.Fatal("archive escaped root")
	}
}

func TestEmbeddedPythonDownloadFailures(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.osType = "linux"
	c.osArch = "amd64"
	for _, tc := range []struct {
		name       string
		status     int
		body       io.ReadCloser
		requestErr error
	}{
		{name: "network", requestErr: errors.New("offline")}, {name: "status", status: 404, body: io.NopCloser(strings.NewReader("missing"))}, {name: "read", status: 200, body: pythonBodyError{}}, {name: "archive", status: 200, body: io.NopCloser(strings.NewReader("invalid"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c.embeddedPythonDir = filepath.Join(t.TempDir(), "python")
			http.DefaultTransport = pythonTransport(func(*http.Request) (*http.Response, error) {
				if tc.requestErr != nil {
					return nil, tc.requestErr
				}
				return &http.Response{StatusCode: tc.status, Body: tc.body}, nil
			})
			if err := c.downloadEmbeddedPython(); err == nil {
				t.Fatal("download failure lost")
			}
		})
	}
}

func TestPythonHostFlagsAndVenvFailure(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	ctx.Flags().Add(&cli.Flag{Name: "feature", Category: "config", AssignedMode: cli.AssignedNone})
	c := NewContext(ctx)
	c.applyMainCliFlagsFromArgs([]string{"run", "--unknown", "--profile=work", "--endpoint=endpoint", "--feature", "--region", "cn-hangzhou"})
	if got, _ := ctx.Flags().GetValue("profile"); got != "work" {
		t.Fatalf("profile=%q", got)
	}
	if got, _ := ctx.Flags().GetValue("endpoint"); got != "endpoint" {
		t.Fatalf("endpoint=%q", got)
	}
	if !ctx.Flags().Get("feature").IsAssigned() {
		t.Fatal("boolean host flag not assigned")
	}
	got, err := c.RemoveFlagsForMainCli([]string{"run", "-p", "work", "--region=cn-hangzhou", "--feature", "--provider", "value"})
	if err != nil || !reflect.DeepEqual(got, []string{"run", "--provider", "value"}) {
		t.Fatalf("child flags=%v,%v", got, err)
	}
	oldExec := execCommandFunc
	t.Cleanup(func() { execCommandFunc = oldExec })
	execCommandFunc = func(string, ...string) *exec.Cmd { return exec.Command(filepath.Join(t.TempDir(), "missing")) }
	c.venvPythonPath = filepath.Join(t.TempDir(), "missing-venv")
	if err := c.EnsureVenvAndPackage(); err == nil {
		t.Fatal("venv creation failure lost")
	}
}
