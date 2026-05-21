package maxc

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Function vars for HTTP — patched by tests with httptest server URLs.
// See cliext/cms2/cms2.go for the canonical pattern; we copy it because the
// installer must work offline in CI (zero real network access in unit tests).
var (
	httpGetFunc = http.Get
	httpDoFunc  = func(req *http.Request) (*http.Response, error) {
		return (&http.Client{Timeout: 30 * time.Second}).Do(req)
	}
	timeNowFunc          = func() time.Time { return time.Now() }
	getLatestVersionFunc = func(c *Context) (string, error) { return c.GetLatestVersion() }
)

// effectiveBaseURL returns the env override (ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL)
// if set, otherwise the package-level downloadBaseURL constant. Spec § 4: the
// override replaces the bucket root entirely — every subsequent URL (latest
// pointer, tarballs, sha files) is built from this single base.
func (c *Context) effectiveBaseURL() string {
	if u := os.Getenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return downloadBaseURL
}

func (c *Context) tarballURL(version string) string {
	return fmt.Sprintf("%s/%s/%s/maxc.tar.gz", c.effectiveBaseURL(), version, c.platformKey)
}

func (c *Context) tarballShaURL(version string) string {
	return c.tarballURL(version) + ".sha256"
}

func (c *Context) latestVersionURL() string {
	return c.effectiveBaseURL() + "/versions/latest"
}

// GetLatestVersion fetches the public `versions/latest` pointer and returns
// its trimmed content. Wrapped by getLatestVersionFunc so tests can stub it
// without needing httptest for the cache-only paths.
func (c *Context) GetLatestVersion() (string, error) {
	resp, err := httpGetFunc(c.latestVersionURL())
	if err != nil {
		return "", fmt.Errorf("GET latest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GET latest: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(body))
	if v == "" {
		return "", fmt.Errorf("empty latest pointer at %s", c.latestVersionURL())
	}
	return v, nil
}

// EnsureInstalledAndUpdated is the gatekeeper called by Run before exec.
// Honors two env shortcuts (per spec § 5):
//   - ALIBABA_CLOUD_MAXC_EXEC_PATH: caller is BYO-binary, do nothing
//   - ALIBABA_CLOUD_MAXC_NO_UPDATE_CHECK=1: skip the network round-trip
//
// Otherwise: install if absent, else hit the latest pointer at most once per
// TTL window; a failed remote check is logged-and-ignored when something is
// already installed (don't break offline users).
func (c *Context) EnsureInstalledAndUpdated() error {
	if os.Getenv("ALIBABA_CLOUD_MAXC_EXEC_PATH") != "" {
		return nil
	}
	if os.Getenv("ALIBABA_CLOUD_MAXC_NO_UPDATE_CHECK") == "1" {
		return nil
	}

	if !c.installed {
		latest, err := getLatestVersionFunc(c)
		if err != nil {
			return fmt.Errorf("resolve latest maxc version: %w", err)
		}
		return c.downloadAndInstall(latest)
	}

	if !c.cacheStale() {
		return nil
	}
	latest, err := getLatestVersionFunc(c)
	if err != nil {
		// Soft-fail: leave the user on whatever they've already got and try
		// again next TTL window. Stderr so it surfaces in `aliyun maxc -v`.
		fmt.Fprintf(os.Stderr, "maxc: update check failed: %v\n", err)
		_ = c.touchCache()
		return nil
	}
	_ = c.touchCache()
	if latest == c.readLocalVersion() {
		return nil
	}
	return c.downloadAndInstall(latest)
}

// downloadAndInstall is the orchestrator: GET tarball + .sha256, verify, extract
// into a staging dir, atomically swap with installDir, persist .version.
// On any failure before the swap, installDir is untouched.
func (c *Context) downloadAndInstall(version string) error {
	parent := filepath.Dir(c.installDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}

	tarPath := filepath.Join(parent, fmt.Sprintf(".maxc-dl-%d.tar.gz", timeNowFunc().UnixNano()))
	defer os.Remove(tarPath)

	if err := httpGetToFile(c.tarballURL(version), tarPath); err != nil {
		return fmt.Errorf("download tarball: %w", err)
	}

	expectedSha, err := fetchExpectedSha(c.tarballShaURL(version))
	if err != nil {
		return fmt.Errorf("fetch sha256: %w", err)
	}
	actualSha, err := computeFileSha256(tarPath)
	if err != nil {
		return fmt.Errorf("compute sha256: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(actualSha), strings.TrimSpace(expectedSha)) {
		return fmt.Errorf("sha256 mismatch: expected=%s actual=%s", expectedSha, actualSha)
	}

	c.versionRemote = version
	return c.installFromTarball(tarPath)
}

// installFromTarball extracts tarPath into a staging dir, then atomically
// swaps it into installDir. Old install (if any) is preserved as
// installDir.old.<ts> until the swap succeeds, then RemoveAll'd. On rename
// failure the old dir is restored.
func (c *Context) installFromTarball(tarPath string) error {
	parent := filepath.Dir(c.installDir)
	ts := timeNowFunc().UnixNano()
	stagingDir := filepath.Join(parent, fmt.Sprintf(".maxc-staging-%d", ts))
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return fmt.Errorf("mkdir staging: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	if err := extractTarGz(tarPath, stagingDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	// Tarball contract (spec § 3): top-level directory is `maxc/`.
	extractedRoot := filepath.Join(stagingDir, "maxc")
	if _, err := os.Stat(extractedRoot); err != nil {
		return fmt.Errorf("tarball missing top-level maxc/ directory: %w", err)
	}

	var oldDir string
	if fileExists(c.installDir) {
		oldDir = fmt.Sprintf("%s.old.%d", c.installDir, ts)
		if err := os.Rename(c.installDir, oldDir); err != nil {
			return fmt.Errorf("rename old install aside: %w", err)
		}
	}

	if err := os.Rename(extractedRoot, c.installDir); err != nil {
		// Best-effort rollback so a failed upgrade doesn't leave the user
		// with no maxc at all.
		if oldDir != "" {
			_ = os.Rename(oldDir, c.installDir)
		}
		return fmt.Errorf("rename staging into place: %w", err)
	}

	if oldDir != "" {
		_ = os.RemoveAll(oldDir)
	}

	c.installed = true
	c.versionLocal = c.versionRemote
	return c.SaveLocalVersion()
}

// SaveLocalVersion writes c.versionRemote (which is now installed) into
// .version so subsequent runs can compare against the latest pointer without
// invoking maxc itself.
func (c *Context) SaveLocalVersion() error {
	return os.WriteFile(c.versionFilePath, []byte(c.versionRemote), 0o644)
}

// httpGetToFile performs GET url, writes body to dest. Non-2xx → error.
func httpGetToFile(url, dest string) error {
	resp, err := httpGetFunc(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		return fmt.Errorf("write %s: %w", dest, err)
	}
	return out.Close()
}

// fetchExpectedSha downloads a .sha256 file and returns its content trimmed.
// The maxc release pipeline (scripts/build_release.sh) writes a single hex
// digest per file: `shasum -a 256 maxc.tar.gz | awk '{print $1}'`.
func fetchExpectedSha(url string) (string, error) {
	resp, err := httpGetFunc(url)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(string(body))
	if sha == "" {
		return "", fmt.Errorf("empty sha256 from %s", url)
	}
	return sha, nil
}

func computeFileSha256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// extractTarGz extracts src (a .tar.gz) into destDir. Rejects tar entries
// with absolute paths or `..` traversal. Supports regular files, dirs, and
// symlinks (PyInstaller onedir bundles on macOS contain framework symlinks).
// File modes are preserved so the maxc binary stays executable.
func extractTarGz(src, destDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gunzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("tar next: %w", err)
		}

		cleaned := filepath.Clean(hdr.Name)
		if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") ||
			strings.Contains(cleaned, string(filepath.Separator)+".."+string(filepath.Separator)) {
			return fmt.Errorf("tar entry has unsafe path: %s", hdr.Name)
		}
		target := filepath.Join(destDir, cleaned)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)|0o700); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			// Best-effort: if the link already exists (e.g. partial earlier
			// extract), remove it before creating.
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		default:
			// Skip hardlinks, fifos, char/block devices — PyInstaller output
			// has none of these and supporting them is a security footgun.
		}
	}
}
