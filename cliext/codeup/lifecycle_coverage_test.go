package codeup

import (
	"archive/zip"
	"bytes"
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
)

type codeupTransport func(*http.Request) (*http.Response, error)

func (f codeupTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type codeupReadError struct{}

func (codeupReadError) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (codeupReadError) Close() error             { return nil }

func codeupZip(t *testing.T, binary string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, name := range []string{"folder/", "README", "folder/" + binary} {
		f, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(name, "/") {
			if _, err := f.Write([]byte("binary")); err != nil {
				t.Fatal(err)
			}
		}
	}
	h := &zip.FileHeader{Name: "link"}
	h.SetMode(os.ModeSymlink | 0755)
	if _, err := z.CreateHeader(h); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestCodeupDownloadAndExtractResponses(t *testing.T) {
	oldClient, oldOS := httpClient, runtimeGOOSFunc
	t.Cleanup(func() { httpClient = oldClient; runtimeGOOSFunc = oldOS })
	runtimeGOOSFunc = func() string { return "linux" }
	for _, tc := range []struct {
		name   string
		status int
		data   []byte
		body   io.ReadCloser
		reqErr error
		dest   string
		want   string
	}{
		{name: "success", status: 200, data: codeupZip(t, "codeup-cli")}, {name: "network", reqErr: errors.New("offline"), want: "offline"}, {name: "status", status: 403, want: "403"}, {name: "invalid zip", status: 200, data: []byte("invalid"), want: "failed to open zip"}, {name: "binary absent", status: 200, data: codeupZip(t, "other"), want: "binary not found"}, {name: "bad destination", status: 200, data: codeupZip(t, "codeup-cli"), dest: "missing/file.zip", want: "failed to create file"}, {name: "body failure", status: 200, body: codeupReadError{}, want: "read failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			httpClient = &http.Client{Transport: codeupTransport(func(r *http.Request) (*http.Response, error) {
				if tc.reqErr != nil {
					return nil, tc.reqErr
				}
				body := tc.body
				if body == nil {
					body = io.NopCloser(bytes.NewReader(tc.data))
				}
				return &http.Response{StatusCode: tc.status, Body: body}, nil
			})}
			dir := t.TempDir()
			dest := tc.dest
			if dest == "" {
				dest = "download.zip"
			}
			dest = filepath.Join(dir, dest)
			exe := filepath.Join(dir, "codeup-cli")
			if err := os.WriteFile(exe, []byte("old"), 0700); err != nil {
				t.Fatal(err)
			}
			err := DownloadAndExtract("https://example.invalid/codeup.zip", dest, exe)
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error=%v,want=%q", err, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(exe)
			if err != nil || string(got) != "binary" {
				t.Fatalf("installed=%q,%v", got, err)
			}
			if fileExists(dest) {
				t.Fatal("temporary archive not removed")
			}
		})
	}
	if err := DownloadAndExtract("://invalid", "unused", "unused"); err == nil {
		t.Fatal("invalid URL accepted")
	}
	runtimeGOOSFunc = func() string { return "windows" }
	dir := t.TempDir()
	src := filepath.Join(dir, "windows.zip")
	if err := os.WriteFile(src, codeupZip(t, "codeup-cli.exe"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractZip(src, filepath.Join(dir, "codeup-cli.exe")); err != nil {
		t.Fatal(err)
	}
}

func TestCodeupRunAndFailurePaths(t *testing.T) {
	oldPath, oldOS, oldArch, oldDownload, oldExec := getConfigurePathFunc, runtimeGOOSFunc, runtimeGOARCHFunc, downloadAndExtractFunc, execCommandFunc
	t.Cleanup(func() {
		getConfigurePathFunc = oldPath
		runtimeGOOSFunc = oldOS
		runtimeGOARCHFunc = oldArch
		downloadAndExtractFunc = oldDownload
		execCommandFunc = oldExec
	})
	dir := t.TempDir()
	t.Setenv("TMPDIR", t.TempDir())
	getConfigurePathFunc = func() string { return dir }
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	var child *exec.Cmd
	var childArgs []string
	downloads := 0
	downloadAndExtractFunc = func(url, dest, exe string) error {
		downloads++
		if !strings.HasSuffix(url, "Linux-64.zip") {
			t.Errorf("download=%s", url)
		}
		return os.WriteFile(exe, []byte("binary"), 0700)
	}
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		childArgs = append([]string(nil), args...)
		child = exec.Command(os.Args[0], "-test.run=^TestCodeupChildSuccess$")
		return child
	}
	c := NewContext(cli.NewCommandContext(io.Discard, io.Discard))
	if err := NewCodeupCliCommand().Run(c.originCtx, []string{"import", "-p", "default", "--region", "cn-hangzhou"}); err != nil {
		t.Fatal(err)
	}
	if downloads != 1 || !reflect.DeepEqual(childArgs, []string{"import", "--region", "cn-hangzhou"}) {
		t.Fatalf("downloads=%d,args=%v", downloads, childArgs)
	}
	if !strings.Contains(strings.Join(child.Env, "\n"), "ALIBABA_CLOUD_CODEUP_INTEGRATION_MODE=aliyun codeup-cli") {
		t.Fatal("integration environment missing")
	}
	if err := c.Run([]string{"status"}); err != nil {
		t.Fatal(err)
	}
	if downloads != 1 {
		t.Fatal("installed binary downloaded again")
	}
	execCommandFunc = func(string, ...string) *exec.Cmd { return exec.Command(filepath.Join(dir, "missing")) }
	if err := c.ExecuteCodeupCli(nil); err == nil {
		t.Fatal("missing executable accepted")
	}
	if err := os.Remove(filepath.Join(dir, "codeup-cli")); err != nil {
		t.Fatal(err)
	}
	downloadAndExtractFunc = func(string, string, string) error { return errors.New("download failed") }
	if err := c.Run(nil); err == nil || !strings.Contains(err.Error(), "download failed") {
		t.Fatalf("install failure=%v", err)
	}
	runtimeGOOSFunc = func() string { return "freebsd" }
	if err := c.Run(nil); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("platform failure=%v", err)
	}
}
func TestCodeupChildSuccess(t *testing.T) {}
