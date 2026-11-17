package computenestutil

import (
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

// TestHelperPipProcess re-execs the test binary as a failing pip: it writes the
// configured text to stderr and exits non-zero, mimicking a real pip failure such
// as a forbidden '--user' install inside a virtualenv. It is guarded by an env var
// so it stays a no-op when executed as a regular test.
func TestHelperPipProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PIP") != "1" {
		return
	}
	if s := os.Getenv("HELPER_PIP_STDERR"); s != "" {
		_, _ = os.Stderr.WriteString(s)
	}
	os.Exit(1)
}

// TestPipErrorDetail covers the helper that decides which failure reason to surface.
func TestPipErrorDetail(t *testing.T) {
	for _, tt := range []struct {
		name   string
		stderr string
		err    error
		want   string
	}{
		{name: "prefers real stderr", stderr: "ERROR: Can not perform a '--user' install.\n", err: fmt.Errorf("exit status 1"), want: "ERROR: Can not perform a '--user' install."},
		{name: "trims surrounding whitespace", stderr: "  \nERROR: boom\n ", err: fmt.Errorf("exit status 1"), want: "ERROR: boom"},
		{name: "falls back to exec error when stderr empty", stderr: "  \n ", err: fmt.Errorf("exit status 1"), want: "exit status 1"},
		{name: "unknown when nothing available", stderr: "", err: nil, want: "unknown error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := pipErrorDetail(tt.stderr, tt.err); got != tt.want {
				t.Errorf("pipErrorDetail(%q, %v) = %q, want %q", tt.stderr, tt.err, got, tt.want)
			}
		})
	}
}

// TestPipInstallSurfacesRealPipError is the regression test for #1409: when both
// OSS and PyPI installs fail, the real pip stderr must be surfaced instead of the
// previous generic "both OSS and PyPI unavailable" message that masked the root cause.
func TestPipInstallSurfacesRealPipError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	const realPipErr = "ERROR: Can not perform a '--user' install. User site-packages are not visible in this virtualenv."

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	// failingPip makes every pip invocation exit non-zero after printing realPipErr.
	failingPip := func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command(exe, "-test.run=^TestHelperPipProcess$")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PIP=1", "HELPER_PIP_STDERR="+realPipErr)
		return cmd
	}

	origExec := execCommandFunc
	origTransport := http.DefaultTransport
	execCommandFunc = failingPip
	t.Cleanup(func() {
		execCommandFunc = origExec
		http.DefaultTransport = origTransport
	})

	newCtx := func() *Context {
		ctx, _, _ := newOriginCtx()
		c := NewContext(ctx)
		c.venvPipPath = filepath.Join(t.TempDir(), "pip")
		return c
	}
	// offline makes downloadWhlFromOSS fail deterministically without hitting network.
	offline := func() {
		http.DefaultTransport = pythonDownloadTransport(func(r *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("simulated offline")
		})
	}

	t.Run("surfaces pypi stderr when oss download fails", func(t *testing.T) {
		offline()
		err := newCtx().pipInstall(false)
		if err == nil {
			t.Fatalf("expected pipInstall to fail")
		}
		if !strings.Contains(err.Error(), realPipErr) {
			t.Errorf("real pip stderr should be surfaced, got: %v", err)
		}
		if strings.Contains(err.Error(), "both OSS and PyPI unavailable") {
			t.Errorf("misleading generic message should be removed, got: %v", err)
		}
	})

	t.Run("surfaces oss stderr when oss download succeeds", func(t *testing.T) {
		http.DefaultTransport = pythonDownloadTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("1.1.2")), Header: make(http.Header)}, nil
		})
		err := newCtx().pipInstall(false)
		if err == nil {
			t.Fatalf("expected pipInstall to fail")
		}
		if !strings.Contains(err.Error(), realPipErr) {
			t.Errorf("real pip stderr should be surfaced, got: %v", err)
		}
	})

	t.Run("upgrade failure stays non-fatal", func(t *testing.T) {
		offline()
		if err := newCtx().pipInstall(true); err != nil {
			t.Errorf("upgrade failure should be non-fatal, got: %v", err)
		}
	})
}
