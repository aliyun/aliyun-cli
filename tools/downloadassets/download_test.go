package downloadassets

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestDownloadSuccessAndRerunReplacesChecksum(t *testing.T) {
	var seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		name := strings.TrimPrefix(r.URL.Path, "/")
		if !strings.Contains(name, "1.2.3") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("asset-" + filepath.Base(name)))
	}))
	defer srv.Close()

	dir := t.TempDir()
	stale := filepath.Join(dir, checksumName)
	if err := os.WriteFile(stale, []byte("oldhash  leftover\noldhash  leftover\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opt := Options{
		Version: "1.2.3",
		BaseURL: srv.URL,
		Token:   "test-token",
		WorkDir: dir,
	}
	if err := Download(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if seenAuth != "Bearer test-token" {
		t.Fatalf("auth header = %q", seenAuth)
	}

	first := mustRead(t, stale)
	names := AssetNames("1.2.3")
	assertFreshChecksum(t, dir, names, first)

	if err := Download(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	second := mustRead(t, stale)
	if first != second {
		t.Fatalf("rerun changed checksum\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if strings.Contains(second, "leftover") {
		t.Fatalf("rerun kept stale checksum lines:\n%s", second)
	}
	if strings.Count(second, names[0]) != 1 {
		t.Fatalf("rerun duplicated %s:\n%s", names[0], second)
	}
}

func TestDownloadHTTP404DoesNotCommitChecksum(t *testing.T) {
	names := AssetNames("1.2.3")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, names[2]) {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	stale := []byte("stale  keep-me\n")
	if err := os.WriteFile(filepath.Join(dir, checksumName), stale, 0o644); err != nil {
		t.Fatal(err)
	}

	err := Download(context.Background(), Options{Version: "1.2.3", BaseURL: srv.URL, WorkDir: dir})
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("expected 404, got %v", err)
	}
	if got := mustRead(t, filepath.Join(dir, checksumName)); got != string(stale) {
		t.Fatalf("failure replaced checksum:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(dir, names[0])); !os.IsNotExist(err) {
		t.Fatal("partial asset was committed after 404")
	}
}

func TestDownloadTimeoutDoesNotCommitChecksum(t *testing.T) {
	dir := t.TempDir()
	err := Download(context.Background(), Options{
		Version: "1.2.3",
		BaseURL: "http://download-assets.invalid",
		WorkDir: dir,
		Timeout: 30 * time.Millisecond,
		Client: &http.Client{
			Timeout: 30 * time.Millisecond,
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				<-r.Context().Done()
				return nil, r.Context().Err()
			}),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected timeout, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, checksumName)); !os.IsNotExist(statErr) {
		t.Fatal("timeout wrote checksum")
	}
}

func TestDownloadEmptyAssetIsMissing(t *testing.T) {
	names := AssetNames("1.2.3")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, names[1]) {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := Download(context.Background(), Options{Version: "1.2.3", BaseURL: srv.URL, WorkDir: dir})
	if err == nil || !strings.Contains(err.Error(), "missing or empty") {
		t.Fatalf("expected empty asset failure, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, checksumName)); !os.IsNotExist(statErr) {
		t.Fatal("empty asset wrote checksum")
	}
}

func TestDownloadChecksumFailureDoesNotCommit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := Download(context.Background(), Options{
		Version: "1.2.3",
		BaseURL: srv.URL,
		WorkDir: dir,
		Summer: func(path string) (string, error) {
			return "", errors.New("shasum failed")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum failure, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, checksumName)); !os.IsNotExist(statErr) {
		t.Fatal("checksum tool failure wrote SHASUMS256.txt")
	}
}

func TestDownloadDefaultsAndDirectFailures(t *testing.T) {
	dir := t.TempDir()
	var gotURL string
	err := Download(nil, Options{
		Version: "1.2.3",
		WorkDir: dir,
		Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotURL = r.URL.String()
			return nil, errors.New("stopped")
		})},
	})
	if err == nil || !strings.Contains(err.Error(), "stopped") {
		t.Fatalf("expected default URL failure, got %v", err)
	}
	if !strings.Contains(gotURL, "https://github.com/aliyun/aliyun-cli/releases/download/v1.2.3/") {
		t.Fatalf("default URL = %s", gotURL)
	}
	if _, statErr := os.Stat(filepath.Join(dir, checksumName)); !os.IsNotExist(statErr) {
		t.Fatal("default URL failure wrote checksum")
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	err = Download(nil, Options{
		Version: " 1.2.3 ",
		Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("stopped")
		})},
	})
	if err == nil || !strings.Contains(err.Error(), "stopped") {
		t.Fatalf("expected empty workdir failure, got %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}

	if err := downloadFile(context.Background(), http.DefaultClient, "://bad", "", filepath.Join(dir, "x"), time.Second); err == nil {
		t.Fatal("expected invalid URL")
	}
	dest := filepath.Join(dir, "exists.tgz")
	if err := os.WriteFile(dest, []byte("already"), 0o644); err != nil {
		t.Fatal(err)
	}
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("body")), Header: make(http.Header)}
	err = downloadFile(context.Background(), &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return resp, nil
	})}, "http://example.invalid/a.tgz", "", dest, time.Second)
	if err == nil {
		t.Fatal("expected existing dest to fail")
	}

	err = downloadFile(context.Background(), &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(errReader{}), Header: make(http.Header)}, nil
	})}, "http://example.invalid/a.tgz", "", filepath.Join(dir, "copy-fail.tgz"), time.Second)
	if err == nil || !strings.Contains(err.Error(), "read failed") {
		t.Fatalf("expected copy failure, got %v", err)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestDownloadRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []Options{
		{Version: ""},
		{Version: "../1.2.3", WorkDir: dir},
		{Version: "1.2.3/evil", WorkDir: dir},
		{Version: `1.2.3\evil`, WorkDir: dir},
		{Version: "1.2.3", WorkDir: filepath.Join(dir, "missing")},
		{Version: "1.2.3", WorkDir: file},
	}
	for _, opt := range cases {
		if err := Download(context.Background(), opt); err == nil {
			t.Fatalf("expected error for %+v", opt)
		}
	}
}

func TestChecksumHelpers(t *testing.T) {
	if _, err := checksumLine("abc", "a.tgz"); err == nil {
		t.Fatal("expected invalid digest")
	}
	if _, err := checksumLine(strings.Repeat("AB", 32), "a.tgz"); err != nil {
		t.Fatal(err)
	}
	if _, err := checksumLine(strings.Repeat("ab", 32), "bad\nname"); err == nil {
		t.Fatal("expected newline in name to fail")
	}
	if _, err := checksumLine(strings.Repeat("ab", 32), ""); err == nil {
		t.Fatal("expected invalid name")
	}
	names := AssetNames("1.2.3")
	if err := verifyChecksum("", names); err == nil {
		t.Fatal("expected empty checksum to fail")
	}
	line, err := checksumLine(strings.Repeat("ab", 32), names[0])
	if err != nil {
		t.Fatal(err)
	}
	body := line + "\n" + line + "\n"
	if err := verifyChecksum(body, []string{names[0], names[0]}); err == nil {
		t.Fatal("expected duplicate checksum to fail")
	}
	if err := verifyChecksum("not-a-digest  "+names[0]+"\n", []string{names[0]}); err == nil {
		t.Fatal("expected malformed line to fail")
	}

	missing := filepath.Join(t.TempDir(), "gone.tgz")
	if err := ensureAsset(missing); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected missing asset, got %v", err)
	}
	dir := t.TempDir()
	if err := ensureAsset(dir); err == nil {
		t.Fatal("expected directory to be rejected")
	}
	empty := filepath.Join(t.TempDir(), "empty.tgz")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureAsset(empty); err == nil {
		t.Fatal("expected empty file to be rejected")
	}
	if _, err := sha256File(missing); err == nil {
		t.Fatal("expected sha256 of missing file to fail")
	}
	sum, err := sha256File(empty)
	if err != nil || len(sum) != 64 {
		t.Fatalf("empty file sum = %q err=%v", sum, err)
	}
}

func TestDownloadInvalidDigestDoesNotCommit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := Download(context.Background(), Options{
		Version: "1.2.3",
		BaseURL: srv.URL,
		WorkDir: dir,
		Summer:  func(path string) (string, error) { return "nope", nil },
	})
	if err == nil || !strings.Contains(err.Error(), "invalid digest") {
		t.Fatalf("expected invalid digest, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, checksumName)); !os.IsNotExist(statErr) {
		t.Fatal("invalid digest committed checksum")
	}
}

func assertFreshChecksum(t *testing.T, dir string, names []string, body string) {
	t.Helper()
	if err := verifyChecksum(body, names); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			t.Fatalf("asset %s missing after success: %v", name, err)
		}
		if strings.Count(body, name) != 1 {
			t.Fatalf("checksum listing for %s is not unique:\n%s", name, body)
		}
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
