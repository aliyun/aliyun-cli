package maxc

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/config"
)

type maxcBodyError struct{}

func (maxcBodyError) Read([]byte) (int, error) { return 0, errors.New("body failed") }
func (maxcBodyError) Close() error             { return nil }

type maxcTransport func(*http.Request) (*http.Response, error)

func (f maxcTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMaxcDownloadAndChecksumFailures(t *testing.T) {
	old := httpGetFunc
	t.Cleanup(func() { httpGetFunc = old })
	for _, tc := range []struct {
		name   string
		status int
		body   io.ReadCloser
		reqErr error
	}{
		{name: "network", reqErr: errors.New("offline")}, {name: "status", status: 503, body: io.NopCloser(strings.NewReader(""))}, {name: "body", status: 200, body: maxcBodyError{}}, {name: "empty checksum", status: 200, body: io.NopCloser(strings.NewReader(" "))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			httpGetFunc = func(string) (*http.Response, error) {
				if tc.reqErr != nil {
					return nil, tc.reqErr
				}
				return &http.Response{StatusCode: tc.status, Body: tc.body}, nil
			}
			if _, err := fetchExpectedSha("https://example.invalid/sha"); err == nil {
				t.Fatal("invalid checksum response accepted")
			}
		})
	}
	for _, tc := range []struct {
		name, dest string
		body       io.ReadCloser
	}{{"create", "missing/file", io.NopCloser(strings.NewReader("data"))}, {"read", "file", maxcBodyError{}}} {
		t.Run(tc.name, func(t *testing.T) {
			httpGetFunc = func(string) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: tc.body}, nil }
			if err := httpGetToFile("https://example.invalid/binary", filepath.Join(t.TempDir(), tc.dest)); err == nil {
				t.Fatal("download failure lost")
			}
		})
	}
	if _, err := computeFileSha256(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing checksum file accepted")
	}
	if _, err := computeFileSha256(t.TempDir()); err == nil {
		t.Fatal("directory checksum accepted")
	}
	c := withCtx(t, "linux-amd64")
	c.installDir = filepath.Join(t.TempDir(), "install")
	if err := c.installFromTarball(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing tarball accepted")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "data")
	if err := os.WriteFile(file, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(file, dir); err == nil {
		t.Fatal("invalid gzip accepted")
	}
	c.installDir = filepath.Join(file, "install")
	if err := c.installFromTarball(file); err == nil {
		t.Fatal("blocked staging accepted")
	}
	c.installDir = filepath.Join(file, "install")
	if err := c.downloadAndInstall("2.0"); err == nil {
		t.Fatal("blocked install parent accepted")
	}
}

func TestMaxcTemporaryCredentials(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	t.Setenv(config.EnvDisableExternalProcess, "")
	http.DefaultTransport = maxcTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"Code":"Success","AccessKeyId":"id","AccessKeySecret":"secret","SecurityToken":"token"}`))}, nil
	})
	id, secret, token, err := extractCredentials(newOriginCtx(), config.Profile{Mode: config.CredentialsURI, CredentialsURI: "https://credentials.example.invalid/temporary"})
	if err != nil || id != "id" || secret != "secret" || token != "token" {
		t.Fatalf("credentials=(%q,%q,%q),%v", id, secret, token, err)
	}
	if _, _, _, err := extractCredentials(newOriginCtx(), config.Profile{Mode: config.External}); err == nil {
		t.Fatal("invalid external credentials accepted")
	}
	e := &ExitError{Code: 7}
	if e.ExitCode() != 7 || !strings.Contains(e.Error(), "7") {
		t.Fatalf("exit error=%v", e)
	}
}
