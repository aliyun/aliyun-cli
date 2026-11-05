package dgate

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func oldDgateInstallation(t *testing.T, c *Context) {
	t.Helper()
	if err := os.WriteFile(c.execFilePath, []byte("old executable"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.versionFilePath, []byte("1.2.2"), 0644); err != nil {
		t.Fatal(err)
	}
	c.installed, c.versionLocal, c.versionRemote = true, "1.2.2", "1.2.3"
}

func TestVersionRecordFailurePreservesOldExecutable(t *testing.T) {
	c, _ := signedDgateContext(t)
	oldDgateInstallation(t, c)
	signedDgateRelease(t, c, "1.2.3", []byte("new executable"))
	previous := downloadBinaryFunc
	t.Cleanup(func() { downloadBinaryFunc = previous })
	downloadBinaryFunc = func(url, dest string) error {
		if err := previous(url, dest); err != nil {
			return err
		}
		if err := os.Remove(c.versionFilePath); err != nil {
			return err
		}
		return os.Mkdir(c.versionFilePath, 0755)
	}
	if err := c.install(); err == nil {
		t.Fatal("version record failure must abort installation")
	}
	data, err := os.ReadFile(c.execFilePath)
	if err != nil || string(data) != "old executable" {
		t.Fatalf("old executable was changed: %q, %v", data, err)
	}
	if c.versionLocal != "1.2.2" {
		t.Fatal("failed installation updated in-memory version")
	}
}

func TestReadOnlyVersionRecordReplacedAtomically(t *testing.T) {
	c, _ := signedDgateContext(t)
	oldDgateInstallation(t, c)
	if err := os.Chmod(c.versionFilePath, 0444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(c.versionFilePath, 0600) })
	signedDgateRelease(t, c, "1.2.3", []byte("new executable"))
	err := c.install()
	expectedFiles := map[string]string{c.execFilePath: "new executable", c.versionFilePath: "1.2.3"}
	if runtime.GOOS == "windows" {
		if err == nil {
			t.Fatal("read-only Windows version record must reject replacement")
		}
		expectedFiles = map[string]string{c.execFilePath: "old executable", c.versionFilePath: "1.2.2"}
	} else if err != nil {
		t.Fatal(err)
	}
	for name, expected := range expectedFiles {
		data, err := os.ReadFile(name)
		if err != nil || string(data) != expected {
			t.Fatalf("inconsistent installation: %s = %q, %v", name, data, err)
		}
	}
}

func TestExecutableFailureRollsBackVersionRecord(t *testing.T) {
	for _, installed := range []bool{false, true} {
		t.Run(map[bool]string{false: "first install", true: "upgrade"}[installed], func(t *testing.T) {
			c, _ := signedDgateContext(t)
			if installed {
				oldDgateInstallation(t, c)
			}
			c.versionRemote = "1.2.3"
			signedDgateRelease(t, c, "1.2.3", []byte("new executable"))
			c.rename = func(string, string) error { return os.ErrPermission }
			if err := c.install(); err == nil {
				t.Fatal("executable replacement failure must be returned")
			}
			data, err := os.ReadFile(c.versionFilePath)
			if installed {
				if err != nil || string(data) != "1.2.2" {
					t.Fatalf("old version record not restored: %q, %v", data, err)
				}
				data, err = os.ReadFile(c.execFilePath)
				if err != nil || string(data) != "old executable" {
					t.Fatalf("old binary not preserved: %q, %v", data, err)
				}
			} else if !os.IsNotExist(err) {
				t.Fatalf("failed first install left a version record: %v", err)
			}
			files, err := filepath.Glob(filepath.Join(c.configPath, ".dgate-version-*"))
			if err != nil || len(files) != 0 {
				t.Fatalf("version temporary files leaked: %v, %v", files, err)
			}
		})
	}
}

func TestManifestExpiryDuringDownloadPreservesOldExecutable(t *testing.T) {
	c, _ := signedDgateContext(t)
	oldDgateInstallation(t, c)
	f, _ := signedDgateRelease(t, c, "1.2.3", []byte("new executable"))
	now := timeNowFunc()
	c.assets.Now = func() time.Time { return now }
	previous := downloadBinaryFunc
	t.Cleanup(func() { downloadBinaryFunc = previous })
	downloadBinaryFunc = func(url, dest string) error {
		err := previous(url, dest)
		now = f.manifest.ExpiresAt
		return err
	}
	if err := c.install(); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired authorization installed: %v", err)
	}
	data, err := os.ReadFile(c.execFilePath)
	if err != nil || !bytes.Equal(data, []byte("old executable")) {
		t.Fatalf("expired new binary replaced old binary: %q, %v", data, err)
	}
}
