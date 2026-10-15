package binmgr

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type boundaryTransport func(*http.Request) (*http.Response, error)

func (f boundaryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type failureBody struct{}

func (failureBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (failureBody) Close() error             { return nil }

type infiniteBody struct{}

func (infiniteBody) Read(p []byte) (int, error) { clear(p); return len(p), nil }
func (infiniteBody) Close() error               { return nil }

func TestAtomicDownloadPreservesInstalledBinaryOnFailure(t *testing.T) {
	old := httpClient
	t.Cleanup(func() { httpClient = old })
	for _, tc := range []struct {
		name      string
		body      io.ReadCloser
		directory bool
		want      string
	}{
		{name: "read failure", body: failureBody{}, want: "read failed"},
		{name: "oversized", body: infiniteBody{}, want: "exceeds limit"},
		{name: "rename failure", body: io.NopCloser(strings.NewReader("new")), directory: true, want: "rename"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			exe := filepath.Join(dir, "binary")
			if tc.directory {
				if err := os.Mkdir(exe, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(exe, "child"), []byte("old"), 0600); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(exe, []byte("old"), 0700); err != nil {
				t.Fatal(err)
			}
			httpClient = &http.Client{Transport: boundaryTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: tc.body}, nil
			})}
			err := DownloadBinary(context.Background(), "https://example.invalid/binary", exe)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("download error=%v", err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Fatalf("temporary files leaked: %v, %v", entries, err)
			}
			if !tc.directory {
				data, err := os.ReadFile(exe)
				if err != nil || string(data) != "old" {
					t.Fatalf("installed binary changed: %q,%v", data, err)
				}
			}
		})
	}
	if err := DownloadBinary(context.Background(), "://invalid", filepath.Join(t.TempDir(), "binary")); err == nil {
		t.Fatal("invalid request accepted")
	}
	httpClient = &http.Client{Transport: boundaryTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("binary"))}, nil
	})}
	if err := DownloadBinary(context.Background(), "https://example.invalid/binary", filepath.Join(t.TempDir(), "missing", "binary")); err == nil {
		t.Fatal("missing destination parent accepted")
	}
}

func TestManagerFlagAndVersionBoundaries(t *testing.T) {
	got, err := removeFlagsForMainCli([]string{"list", "-p", "default", "--read-timeout", "20", "--retry-timeout=10", "--provider"}, map[string]bool{"profile": true, "read-timeout": true}, map[string]bool{"p": true})
	if err != nil || !reflect.DeepEqual(got, []string{"list", "--provider"}) {
		t.Fatalf("flags=%v,%v", got, err)
	}
	if _, _, ok := parseVersionParts("invalid-1234567"); ok {
		t.Fatal("invalid semantic version accepted")
	}
	old := httpDoFunc
	t.Cleanup(func() { httpDoFunc = old })
	httpDoFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: failureBody{}}, nil
	}
	if _, err := GetLatestVersion(context.Background(), "https://example.invalid/"); err == nil {
		t.Fatal("version body failure lost")
	}
	if _, err := GetLatestVersion(context.Background(), "://invalid"); err == nil {
		t.Fatal("invalid version URL accepted")
	}
}
