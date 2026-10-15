package rostran

import (
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

type rostranTransport func(*http.Request) (*http.Response, error)

func (f rostranTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRostranPathAndFilesystemBoundaries(t *testing.T) {
	for _, name := range []string{"", "a\x00b", "/root/file", `C:\root\file`, "../file"} {
		if _, err := sanitizeArchivePath(name); err == nil {
			t.Errorf("unsafe path %q accepted", name)
		}
	}
	if got, err := sanitizeArchivePath("././folder/file"); err != nil || got != "folder/file" {
		t.Fatalf("clean path=%q,%v", got, err)
	}
	if got, err := sanitizeArchivePath("."); err != nil || got != "" {
		t.Fatalf("dot path=%q,%v", got, err)
	}
	for _, target := range []string{".", "../outside", "/outside"} {
		if _, err := sanitizeSymlinkTarget("root/link", target); err == nil {
			t.Errorf("unsafe link %q accepted", target)
		}
	}
	if isSafeSymlink("root/link", "../../outside") {
		t.Fatal("escaping symlink accepted")
	}
	for value, want := range map[string]string{"": "", ".": "", "root": "root", "root/file": "root"} {
		if got := firstPathComponent(value); got != want {
			t.Errorf("first component %q=%q", value, got)
		}
	}
	if _, err := archiveLocalPath("../file"); err == nil {
		t.Fatal("nonlocal path accepted")
	}
	if _, err := safeJoinLocalUnderDir(t.TempDir(), "../outside"); err == nil {
		t.Fatal("join escaped destination")
	}
	dir := t.TempDir()
	blocking := filepath.Join(dir, "block")
	if err := os.WriteFile(blocking, []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"missing archive", func() error { return extractTarGz(filepath.Join(dir, "missing"), dir) }},
		{"invalid archive", func() error { return extractTarGz(blocking, dir) }},
		{"missing directory", func() error { return copyDir(filepath.Join(dir, "missing"), filepath.Join(dir, "out")) }},
		{"missing source", func() error { return copyFileWithMode(filepath.Join(dir, "missing"), filepath.Join(dir, "out"), 0600) }},
		{"blocked destination", func() error { return copyFileWithMode(blocking, filepath.Join(blocking, "out"), 0600) }},
		{"destination is directory", func() error { return copyFileWithMode(blocking, dir, 0600) }},
		{"blocked move parent", func() error { return moveDir(dir, filepath.Join(blocking, "out")) }},
		{"missing move source", func() error { return moveDir(filepath.Join(dir, "missing"), filepath.Join(dir, "out")) }},
		{"blocked install parent", func() error { return replaceInstallDir(dir, filepath.Join(blocking, "out")) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err == nil {
				t.Fatal("filesystem error lost")
			}
		})
	}
	if _, err := packageRootDir(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing package root accepted")
	}
	if _, err := packageRootDir(t.TempDir()); err == nil {
		t.Fatal("empty package root accepted")
	}
	root, err := packageRootDir(dir)
	if err != nil || root != dir {
		t.Fatalf("flat package root=%q,%v", root, err)
	}
}

func TestRostranUpgradeAndVersionFetch(t *testing.T) {
	oldLatest, oldExec, oldTransport := getLatestRostranVersionFunc, execCommandFunc, http.DefaultTransport
	t.Cleanup(func() {
		getLatestRostranVersionFunc = oldLatest
		execCommandFunc = oldExec
		http.DefaultTransport = oldTransport
	})
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	c := NewContext(ctx)
	c.configPath = t.TempDir()
	c.installed = true
	execCommandFunc = func(string, ...string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRostranVersionChild$")
		cmd.Env = append(os.Environ(), "CLI_TEST_ROSTRAN_VERSION=2.0")
		return cmd
	}
	getLatestRostranVersionFunc = func() (string, error) { return "2.0", nil }
	if err := c.SelfUpgrade(); err != nil {
		t.Fatal(err)
	}
	if c.versionLocal != "2.0" {
		t.Fatalf("version=%q", c.versionLocal)
	}
	getLatestRostranVersionFunc = func() (string, error) { return "", errors.New("offline") }
	if err := c.SelfUpgrade(); err == nil {
		t.Fatal("lookup failure lost")
	}
	execCommandFunc = func(string, ...string) *exec.Cmd { return exec.Command(filepath.Join(t.TempDir(), "missing")) }
	if err := c.GetLocalVersion(); err == nil {
		t.Fatal("version process failure lost")
	}
	c.installed = false
	if err := c.GetLocalVersion(); err == nil {
		t.Fatal("uninstalled version accepted")
	}
	for _, tc := range []struct {
		status  int
		body    string
		wantErr bool
	}{{200, " 2.0 ", false}, {200, " ", true}, {404, "missing", true}} {
		http.DefaultTransport = rostranTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
		})
		version, err := GetLatestRostranVersion()
		if (err != nil) != tc.wantErr || (!tc.wantErr && version != "2.0") {
			t.Fatalf("version=%q,%v", version, err)
		}
	}
	http.DefaultTransport = rostranTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })
	if _, err := GetLatestRostranVersion(); err == nil {
		t.Fatal("network error lost")
	}
	if _, err := downloadHTTPGet("://invalid"); err == nil {
		t.Fatal("invalid URL accepted")
	}
}
func TestRostranVersionChild(t *testing.T) {
	if version := os.Getenv("CLI_TEST_ROSTRAN_VERSION"); version != "" {
		_, _ = io.WriteString(os.Stdout, version)
		os.Exit(0)
	}
}
