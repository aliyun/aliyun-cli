package maxc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withGetLatestStub swaps getLatestVersionFunc for the test duration.
func withGetLatestStub(t *testing.T, fn func(*Context) (string, error)) {
	t.Helper()
	orig := getLatestVersionFunc
	t.Cleanup(func() { getLatestVersionFunc = orig })
	getLatestVersionFunc = fn
}

// stageTarball seeds the fake OSS map so downloadAndInstall succeeds.
func stageTarball(t *testing.T, c *Context, version string) {
	t.Helper()
	tar := buildTarGz(t, []tarEntry{
		{Name: "maxc/", Mode: 0o755},
		{Name: "maxc/maxc", Body: "#!/bin/sh\necho fake\n", Mode: 0o755},
	})
	sha := sha256Hex(tar)
	oss := newFakeOSS(t, map[string][]byte{
		version + "/" + c.platformKey + "/maxc.tar.gz":        tar,
		version + "/" + c.platformKey + "/maxc.tar.gz.sha256": []byte(sha),
	})
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", oss.srv.URL)
}

func TestEnsureInstalledAndUpdated_NotInstalled_DownloadsLatest(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	stageTarball(t, c, "0.4.0")
	withGetLatestStub(t, func(*Context) (string, error) { return "0.4.0", nil })

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("EnsureInstalledAndUpdated: %v", err)
	}
	if !fileExists(c.execFilePath) {
		t.Errorf("expected exec at %s after first install", c.execFilePath)
	}
	if got := c.readLocalVersion(); got != "0.4.0" {
		t.Errorf(".version = %q, want 0.4.0", got)
	}
}

func TestEnsureInstalledAndUpdated_FreshCache_Skips(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	if err := os.MkdirAll(c.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.execFilePath, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	c.installed = true
	// Touch the cache sentinel to "now" — cacheStale must return false.
	if err := c.touchCache(); err != nil {
		t.Fatal(err)
	}

	called := 0
	withGetLatestStub(t, func(*Context) (string, error) {
		called++
		return "9.9.9", nil
	})

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("EnsureInstalledAndUpdated: %v", err)
	}
	if called != 0 {
		t.Errorf("getLatestVersionFunc called %d times — fresh cache must skip remote", called)
	}
	// Old binary untouched.
	if b, _ := os.ReadFile(c.execFilePath); string(b) != "OLD" {
		t.Errorf("old exec was modified: %q", string(b))
	}
}

func TestEnsureInstalledAndUpdated_StaleCache_ChecksRemote(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	if err := os.MkdirAll(c.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.execFilePath, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.versionFilePath, []byte("0.3.0"), 0o644); err != nil {
		t.Fatal(err)
	}
	c.installed = true
	// Backdate cache sentinel past TTL.
	if err := os.MkdirAll(filepath.Dir(c.versionCachePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.versionCachePath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	ancient := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(c.versionCachePath, ancient, ancient); err != nil {
		t.Fatal(err)
	}

	stageTarball(t, c, "0.4.0")
	withGetLatestStub(t, func(*Context) (string, error) { return "0.4.0", nil })

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("EnsureInstalledAndUpdated: %v", err)
	}
	if got := c.readLocalVersion(); got != "0.4.0" {
		t.Errorf("expected upgrade to 0.4.0, got %q", got)
	}
}

func TestEnsureInstalledAndUpdated_RemoteCheckFails_KeepsOldVersion(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	if err := os.MkdirAll(c.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.execFilePath, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	c.installed = true
	// Stale cache.
	if err := os.MkdirAll(filepath.Dir(c.versionCachePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.versionCachePath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	ancient := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(c.versionCachePath, ancient, ancient)

	withGetLatestStub(t, func(*Context) (string, error) {
		return "", errors.New("network down")
	})

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Errorf("expected soft-fail (nil), got %v", err)
	}
	if b, _ := os.ReadFile(c.execFilePath); string(b) != "OLD" {
		t.Errorf("old exec was modified after soft-fail")
	}
	// Cache should be touched so we don't hammer the server next call.
	fi, err := os.Stat(c.versionCachePath)
	if err != nil {
		t.Fatalf("cache file missing: %v", err)
	}
	if time.Since(fi.ModTime()) > time.Minute {
		t.Errorf("cache mtime not refreshed; mtime=%v", fi.ModTime())
	}
}

func TestEnsureInstalledAndUpdated_NoUpdateCheckEnv_Skips(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	t.Setenv("ALIBABA_CLOUD_MAXC_NO_UPDATE_CHECK", "1")

	called := 0
	withGetLatestStub(t, func(*Context) (string, error) {
		called++
		return "9.9.9", nil
	})

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("EnsureInstalledAndUpdated: %v", err)
	}
	if called != 0 {
		t.Errorf("NO_UPDATE_CHECK=1 must short-circuit; called=%d", called)
	}
	if fileExists(c.installDir) {
		t.Errorf("nothing should be downloaded when update check is disabled")
	}
}

func TestEnsureInstalledAndUpdated_ExecPathEnv_SkipsEverything(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	t.Setenv("ALIBABA_CLOUD_MAXC_EXEC_PATH", "/opt/maxc/maxc")

	withGetLatestStub(t, func(*Context) (string, error) {
		t.Fatal("must not be called when EXEC_PATH is set")
		return "", nil
	})

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("EnsureInstalledAndUpdated: %v", err)
	}
}

// Quick sanity: readLocalVersion trims trailing whitespace.
func TestReadLocalVersion_Trim(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	if err := os.MkdirAll(c.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.versionFilePath, []byte("0.5.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := c.readLocalVersion(); got != "0.5.0" {
		t.Errorf("readLocalVersion = %q, want 0.5.0", got)
	}
	// Missing file → "".
	if err := os.Remove(c.versionFilePath); err != nil {
		t.Fatal(err)
	}
	if got := c.readLocalVersion(); got != "" {
		t.Errorf("missing .version = %q, want empty", got)
	}
	_ = strings.TrimSpace
}
