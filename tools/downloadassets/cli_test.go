package downloadassets

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteUsageAndTimeout(t *testing.T) {
	var stderr bytes.Buffer
	if code := Execute(nil, &bytes.Buffer{}, &stderr); code == 0 || !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("usage code=%d stderr=%s", code, stderr.String())
	}
	stderr.Reset()
	if code := Execute([]string{"--", ""}, &bytes.Buffer{}, &stderr); code == 0 {
		t.Fatal("expected empty version to fail")
	}

	t.Setenv("DOWNLOAD_ASSETS_TIMEOUT", "not-a-duration")
	stderr.Reset()
	if code := Execute([]string{"1.2.3"}, &bytes.Buffer{}, &stderr); code == 0 || !strings.Contains(stderr.String(), "invalid timeout") {
		t.Fatalf("timeout code=%d stderr=%s", code, stderr.String())
	}
}

func TestExecuteSuccessAndHTTPFailure(t *testing.T) {
	names := AssetNames("9.9.9")
	fail := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			http.Error(w, "nope", http.StatusGatewayTimeout)
			return
		}
		_, _ = w.Write([]byte("body-" + filepath.Base(r.URL.Path)))
	}))
	defer srv.Close()

	dir := t.TempDir()
	t.Setenv("DOWNLOAD_ASSETS_WORKDIR", dir)
	t.Setenv("DOWNLOAD_ASSETS_BASE_URL", srv.URL)
	t.Setenv("DOWNLOAD_ASSETS_TIMEOUT", "5s")
	t.Setenv("GITHUB_TOKEN", "cli-token")

	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"--", "9.9.9"}, &stdout, &stderr); code != 0 {
		t.Fatalf("success code=%d stderr=%s", code, stderr.String())
	}
	if err := verifyChecksum(stdout.String(), names); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != mustRead(t, filepath.Join(dir, checksumName)) {
		t.Fatal("stdout does not match committed checksum")
	}
	before := mustRead(t, filepath.Join(dir, checksumName))

	fail = true
	stderr.Reset()
	var failed bytes.Buffer
	if code := Execute([]string{"9.9.9"}, &failed, &stderr); code == 0 || !strings.Contains(stderr.String(), "HTTP 504") {
		t.Fatalf("expected HTTP failure, code=%d stderr=%s", code, stderr.String())
	}
	if got := mustRead(t, filepath.Join(dir, checksumName)); got != before {
		t.Fatal("failed run replaced the committed checksum")
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestExecuteStdoutFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	t.Setenv("DOWNLOAD_ASSETS_WORKDIR", dir)
	t.Setenv("DOWNLOAD_ASSETS_BASE_URL", srv.URL)
	t.Setenv("GITHUB_TOKEN", "")

	var stderr bytes.Buffer
	if code := Execute([]string{"1.2.3"}, failWriter{}, &stderr); code == 0 || !strings.Contains(stderr.String(), "write failed") {
		t.Fatalf("expected stdout failure, code=%d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, checksumName)); err != nil {
		t.Fatal("checksum should already be committed before printing")
	}
}

func TestParseArgs(t *testing.T) {
	version, err := parseArgs([]string{"--", "3.5.1"})
	if err != nil || version != "3.5.1" {
		t.Fatalf("parse = %q %v", version, err)
	}
	if _, err := parseArgs([]string{"a", "b"}); err == nil {
		t.Fatal("expected extra args to fail")
	}
}
