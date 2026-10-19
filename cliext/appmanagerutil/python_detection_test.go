package appmanagerutil

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type pythonDownloadTransport func(*http.Request) (*http.Response, error)

func (f pythonDownloadTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Run the test binary as the version probe so no installed Python is needed.
func TestPythonVersionProbeProcess(t *testing.T) {
	if os.Getenv("ALIYUN_TEST_PYTHON_PROBE") != "1" {
		return
	}
	fmt.Fprintln(os.Stdout, os.Getenv("ALIYUN_TEST_PYTHON_VERSION"))
	os.Exit(0)
}

func TestEnsurePythonAvailable(t *testing.T) {
	for _, tt := range []struct {
		name, python3, python, selected string
		download, failDownload          bool
	}{
		{name: "supported python3", python3: "Python 3.12.1", selected: "python3"},
		{name: "minimum version", python3: "Python 3.10.0", selected: "python3"},
		{name: "fall back to python", python3: "Python 3.9.6", python: "Python 3.11.1", selected: "python"},
		{name: "invalid version", python3: "invalid", python: "Python 3.12.1", selected: "python"},
		{name: "old Python downloads runtime", python3: "Python 3.9.6", download: true},
		{name: "missing Python downloads runtime", download: true},
		{name: "download failure", download: true, failDownload: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("PATH", dir)
			versions := map[string]string{}
			paths := map[string]string{}
			for name, version := range map[string]string{"python3": tt.python3, "python": tt.python} {
				if version == "" {
					continue
				}
				filename := name
				if runtime.GOOS == "windows" {
					filename += ".exe"
				}
				path := filepath.Join(dir, filename)
				if err := os.WriteFile(path, []byte("fixture"), 0755); err != nil {
					t.Fatal(err)
				}
				versions[path], paths[name] = version, path
			}
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			originalCommand := execCommandFunc
			execCommandFunc = func(name string, args ...string) *exec.Cmd {
				version, ok := versions[name]
				if !ok || len(args) != 1 || args[0] != "--version" {
					t.Fatalf("unexpected Python probe: %q %v", name, args)
				}
				cmd := exec.Command(executable, "-test.run=^TestPythonVersionProbeProcess$")
				cmd.Env = append(os.Environ(), "ALIYUN_TEST_PYTHON_PROBE=1", "ALIYUN_TEST_PYTHON_VERSION="+version)
				return cmd
			}
			t.Cleanup(func() { execCommandFunc = originalCommand })
			ctx, _, _ := newOriginCtx()
			c := NewContext(ctx)
			c.osType, c.osArch = "linux", "amd64"
			c.embeddedPythonDir = filepath.Join(dir, "embedded")
			downloads := 0
			originalTransport := http.DefaultTransport
			http.DefaultTransport = pythonDownloadTransport(func(r *http.Request) (*http.Response, error) {
				downloads++
				if !tt.download {
					t.Fatalf("unexpected download: %s", r.URL)
				}
				if r.Method != http.MethodGet || r.URL.String() != c.getEmbeddedPythonDownloadURL() {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
				}
				if tt.failDownload {
					return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("unavailable")), Header: make(http.Header)}, nil
				}
				var archive bytes.Buffer
				gz := gzip.NewWriter(&archive)
				tw := tar.NewWriter(gz)
				body := []byte("test Python runtime")
				if err := tw.WriteHeader(&tar.Header{Name: "python/bin/python3", Mode: 0755, Size: int64(len(body))}); err != nil {
					t.Fatal(err)
				}
				if _, err := tw.Write(body); err != nil {
					t.Fatal(err)
				}
				if err := tw.Close(); err != nil {
					t.Fatal(err)
				}
				if err := gz.Close(); err != nil {
					t.Fatal(err)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(archive.Bytes())), Header: make(http.Header)}, nil
			})
			t.Cleanup(func() { http.DefaultTransport = originalTransport })
			err = c.EnsurePythonAvailable()
			expectedDownloads := 0
			if tt.download {
				expectedDownloads = 1
			}
			if downloads != expectedDownloads {
				t.Fatalf("downloads = %d, want %d", downloads, expectedDownloads)
			}
			if tt.failDownload {
				if err == nil || !strings.Contains(err.Error(), "HTTP 503") {
					t.Fatalf("expected download error, got %v", err)
				}
				if c.pythonPath != "" {
					t.Fatalf("failed download selected Python: %q", c.pythonPath)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			expected := paths[tt.selected]
			if tt.download {
				expected = c.getEmbeddedPythonPath()
				body, err := os.ReadFile(expected)
				if err != nil || string(body) != "test Python runtime" {
					t.Fatalf("downloaded runtime = %q, err = %v", body, err)
				}
			}
			if c.pythonPath != expected {
				t.Fatalf("pythonPath = %q, want %q", c.pythonPath, expected)
			}
		})
	}
}
