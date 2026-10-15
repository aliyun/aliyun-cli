package computenestutil

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestVenvInstallAndUpgradeBoundaries(t *testing.T) {
	oldExec, oldTransport := execCommandFunc, http.DefaultTransport
	t.Cleanup(func() { execCommandFunc = oldExec; http.DefaultTransport = oldTransport })
	t.Setenv("GOCOVERDIR", t.TempDir())
	for _, scenario := range []string{"fresh", "install failure", "upgrade failure"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			c := NewContext(ctx)
			c.venvPath = filepath.Join(dir, "venv")
			c.venvPythonPath = filepath.Join(c.venvPath, "python")
			c.venvPipPath = filepath.Join(c.venvPath, "pip")
			c.checkVersionCacheFilePath = filepath.Join(dir, "cache")
			c.installed = scenario == "upgrade failure"
			if c.installed {
				if err := os.Mkdir(c.venvPath, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(c.venvPythonPath, []byte("installed"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			http.DefaultTransport = pythonTransport(func(r *http.Request) (*http.Response, error) {
				body := "wheel"
				if strings.HasSuffix(r.URL.Path, "version.txt") {
					body = "2.0"
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			var commands [][]string
			execCommandFunc = func(name string, args ...string) *exec.Cmd {
				commands = append(commands, append([]string{name}, args...))
				if name == c.venvPipPath && scenario != "fresh" {
					return exec.Command(filepath.Join(dir, "missing-command"))
				}
				return exec.Command(os.Args[0], "-test.run=^TestVenvBoundaryChild$")
			}
			err := c.EnsureVenvAndPackage()
			if scenario == "install failure" {
				if err == nil || !strings.Contains(err.Error(), "both OSS and PyPI unavailable") {
					t.Fatalf("first installation failure=%v", err)
				}
				if c.installed {
					t.Fatal("failed install marked installed")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if !c.installed {
					t.Fatal("usable package marked uninstalled")
				}
				if !fileExists(c.checkVersionCacheFilePath) || c.NeedCheckVersion() {
					t.Fatal("installation/update cache not refreshed")
				}
			}
			if len(commands) < 2 {
				t.Fatalf("venv and package commands=%v", commands)
			}
		})
	}
}
func TestVenvBoundaryChild(t *testing.T) {}

func TestEmbeddedPythonWindowsArchiveAndMissingBinary(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	for _, entry := range []string{"python/python.exe", "README"} {
		t.Run(entry, func(t *testing.T) {
			var b bytes.Buffer
			zw := zip.NewWriter(&b)
			w, err := zw.Create(entry)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte("Python runtime")); err != nil {
				t.Fatal(err)
			}
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}
			http.DefaultTransport = pythonTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(b.Bytes()))}, nil
			})
			c := NewContext(cli.NewCommandContext(io.Discard, io.Discard))
			c.osType = "windows"
			c.osArch = "amd64"
			c.embeddedPythonDir = filepath.Join(t.TempDir(), "python")
			err = c.downloadEmbeddedPython()
			if entry == "README" {
				if err == nil || !strings.Contains(err.Error(), "binary not found") {
					t.Fatalf("missing executable error=%v", err)
				}
				if fileExists(c.embeddedPythonDir) {
					t.Fatal("incomplete runtime not removed")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(c.getEmbeddedPythonPath())
				if err != nil || string(data) != "Python runtime" {
					t.Fatalf("downloaded runtime=%q,%v", data, err)
				}
			}
		})
	}
	dir := t.TempDir()
	blocked := filepath.Join(dir, "file")
	if err := os.WriteFile(blocked, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	c := NewContext(cli.NewCommandContext(io.Discard, io.Discard))
	c.embeddedPythonDir = filepath.Join(blocked, "python")
	if err := c.downloadEmbeddedPython(); err == nil {
		t.Fatal("blocked runtime directory accepted")
	}
	t.Setenv("TMPDIR", filepath.Join(blocked, "tmp"))
	t.Setenv("TMP", filepath.Join(blocked, "tmp"))
	t.Setenv("TEMP", filepath.Join(blocked, "tmp"))
	http.DefaultTransport = pythonTransport(func(r *http.Request) (*http.Response, error) {
		body := "wheel"
		if strings.HasSuffix(r.URL.Path, "version.txt") {
			body = "2.0"
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	if _, err := c.downloadWhlFromOSS(); err == nil || !strings.Contains(err.Error(), "temp dir") {
		t.Fatalf("blocked wheel temporary directory=%v", err)
	}
}

func TestPythonTarEntryFailureBoundaries(t *testing.T) {
	fixture := func(name string, truncated bool) []byte {
		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		tw := tar.NewWriter(gz)
		h := &tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0600, Size: 4}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if !truncated {
			if _, err := tw.Write([]byte("data")); err != nil {
				t.Fatal(err)
			}
			if err := tw.Close(); err != nil {
				t.Fatal(err)
			}
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	dir := t.TempDir()
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(fixture("file", false), blocked); err == nil {
		t.Fatal("blocked file parent accepted")
	}
	if err := os.Mkdir(filepath.Join(dir, "file"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(fixture("file", false), dir); err == nil {
		t.Fatal("destination directory overwritten")
	}
	if err := extractTarGz(fixture("file", true), t.TempDir()); err == nil {
		t.Fatal("truncated file accepted")
	}
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte("invalid tar")); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(b.Bytes(), t.TempDir()); err == nil {
		t.Fatal("invalid tar accepted")
	}
}

func TestHostFlagIncompleteValueIsIgnored(t *testing.T) {
	c := NewContext(nil)
	c.applyMainCliFlagsFromArgs([]string{"--profile", "default"})
	ctx, _, _ := newOriginCtx()
	c = NewContext(ctx)
	flag := &cli.Flag{Name: "profile", AssignedMode: cli.AssignedOnce, Category: "config"}
	ctx.Flags().Add(flag)
	c.applyMainCliFlagsFromArgs([]string{"--profile"})
	if flag.IsAssigned() {
		t.Fatal("missing host value consumed")
	}
	if _, err := c.getPythonVersion(filepath.Join(t.TempDir(), "missing-python")); err == nil {
		t.Fatal("missing interpreter accepted")
	}
}
