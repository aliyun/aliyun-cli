package dgate

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRemoveParentFlagsKeepsDgateFlags(t *testing.T) {
	got := removeParentFlags([]string{
		"exec",
		"--profile", "demo",
		"--profile=other",
		"--mode", "AK",
		"--config-path", "/tmp/aliyun",
		"--region", "cn-hangzhou",
		"--region=cn-shanghai",
		"-o", "json",
		"--output", "text",
		"--pager",
		"--cli-ai-mode",
		"-p", "demo",
		"--access-key-id", "id",
		"--access-key-secret=secret",
	})
	want := []string{"exec", "--region", "cn-hangzhou", "--region=cn-shanghai", "-o", "json", "--output", "text"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}

func TestPrepareEnvIsTypeC(t *testing.T) {
	c := NewContext(nil)
	c.prepareEnv()
	if c.envMap[compatModeEnv] != compatModeValue {
		t.Fatalf("compat = %#v", c.envMap)
	}
	for k := range c.envMap {
		if strings.Contains(strings.ToLower(k), "access_key") || strings.Contains(strings.ToLower(k), "secret") {
			t.Fatalf("type C injected credential env %s", k)
		}
	}
	if len(c.envMap) != 1 {
		t.Fatalf("env = %#v", c.envMap)
	}
}

func TestValidReleaseVersion(t *testing.T) {
	if !validReleaseVersion("0.6.22") || !validReleaseVersion("v1.2.3") {
		t.Fatal("semver should pass")
	}
	for _, bad := range []string{"", "../0.6.22", "0.6.22/evil", "latest", "1.2"} {
		if validReleaseVersion(bad) {
			t.Fatalf("accepted %q", bad)
		}
	}
}

func TestDownloadAndChecksum(t *testing.T) {
	payload := []byte("#!/bin/sh\necho dgate\n")
	sum := sha256.Sum256(payload)
	mux := http.NewServeMux()
	mux.HandleFunc("/dgate/prd/latest.txt", func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, "aliyun-cli/") {
			t.Errorf("user agent = %q", ua)
		}
		_, _ = w.Write([]byte("0.6.22\n"))
	})
	mux.HandleFunc("/dgate/prd/0.6.22/dgate_darwin_arm64", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	mux.HandleFunc("/dgate/prd/0.6.22/dgate_darwin_arm64.sha256", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(hex.EncodeToString(sum[:]) + "  dgate_darwin_arm64\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	t.Setenv(execPathEnv, "")
	t.Setenv(downloadBaseEnv, srv.URL+"/dgate/prd")
	prevPath := getConfigurePathFunc
	getConfigurePathFunc = func() string { return dir }
	defer func() { getConfigurePathFunc = prevPath }()
	prevOS := runtimeGOOSFunc
	prevArch := runtimeGOARCHFunc
	runtimeGOOSFunc = func() string { return "darwin" }
	runtimeGOARCHFunc = func() string { return "arm64" }
	defer func() {
		runtimeGOOSFunc = prevOS
		runtimeGOARCHFunc = prevArch
	}()

	c := NewContext(nil)
	if err := c.initialize(); err != nil {
		t.Fatal(err)
	}
	if err := c.ensureInstalled(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "dgate"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("binary = %q", got)
	}
	ver, err := os.ReadFile(filepath.Join(dir, ".dgate_version"))
	if err != nil || string(ver) != "0.6.22" {
		t.Fatalf("version file = %q err=%v", ver, err)
	}
}

func TestExternalBinarySkipsDownload(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-dgate")
	if err := os.WriteFile(bin, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(execPathEnv, bin)
	called := false
	prev := getRemoteVersionFunc
	getRemoteVersionFunc = func(string) (string, error) {
		called = true
		return "", os.ErrNotExist
	}
	defer func() { getRemoteVersionFunc = prev }()

	c := NewContext(nil)
	if err := c.initialize(); err != nil {
		t.Fatal(err)
	}
	if err := c.ensureInstalled(); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("external binary should skip version check")
	}
}

func TestUpdateFailureDoesNotBlock(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(execPathEnv, "")
	if err := os.WriteFile(filepath.Join(dir, "dgate"), []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".dgate_version"), []byte("0.6.1"), 0644); err != nil {
		t.Fatal(err)
	}
	prevPath := getConfigurePathFunc
	getConfigurePathFunc = func() string { return dir }
	defer func() { getConfigurePathFunc = prevPath }()
	prevVer := getRemoteVersionFunc
	getRemoteVersionFunc = func(string) (string, error) { return "", os.ErrDeadlineExceeded }
	defer func() { getRemoteVersionFunc = prevVer }()
	prevNow := timeNowFunc
	timeNowFunc = func() time.Time { return time.Unix(2_000_000_000, 0) }
	defer func() { timeNowFunc = prevNow }()

	c := NewContext(nil)
	if err := c.initialize(); err != nil {
		t.Fatal(err)
	}
	if err := c.ensureInstalled(); err != nil {
		t.Fatal(err)
	}
}
