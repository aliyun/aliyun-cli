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

// Function vars stubbed by tests with httptest server URLs so the installer
// runs offline in CI.
var (
	httpGetFunc = http.Get
	httpDoFunc  = func(req *http.Request) (*http.Response, error) {
		return (&http.Client{Timeout: 30 * time.Second}).Do(req)
	}
	timeNowFunc          = func() time.Time { return time.Now() }
	getLatestVersionFunc = func(c *Context) (string, error) { return c.GetLatestVersion() }
)

// effectiveBaseURL returns the override URL from
// ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL (with trailing slash trimmed) when
// set, else downloadBaseURL. The override replaces the bucket root entirely
// — every subsequent URL (latest pointer, tarballs, sha files) is built
// from this single base.
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

// GetLatestVersion fetches the public versions/latest pointer and returns
// its trimmed content.
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

// EnsureInstalledAndUpdated installs maxc if missing, otherwise checks for
// a newer version at most once per VersionCheckTTL window. Two env shortcuts
// suppress all network traffic:
//   - ALIBABA_CLOUD_MAXC_EXEC_PATH: caller is BYO-binary, skip everything
//   - ALIBABA_CLOUD_MAXC_NO_UPDATE_CHECK=1: keep whatever is installed
//
// A failed remote check is logged-and-ignored when something is already
// installed so offline users don't get blocked.
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
		// again next TTL window.
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

// downloadAndInstall fetches the tarball and its .sha256, verifies, then
// hands off to installFromTarball. On any failure before the swap, installDir
// is untouched.
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

// installFromTarball extracts into a staging dir, then atomically renames it
// into installDir. Any prior install is moved aside first and only deleted
// after the swap succeeds; on rename failure it is restored.
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

// fetchExpectedSha downloads a .sha256 sidecar file and returns its trimmed
// content (a single hex digest).
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

// extractTarGz extracts src (a .tar.gz) into destDir. Both the entry's path
// and any symlink target must resolve to a path under destDir; symlinks are
// created only after every regular file/dir is in place, so a malicious tar
// can't replace a real directory with a symlink mid-extract and trick a
// later MkdirAll/OpenFile into walking outside destDir.
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

	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("abs destDir: %w", err)
	}

	type pendingSymlink struct{ target, linkname string }
	var symlinks []pendingSymlink

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar next: %w", err)
		}

		target := filepath.Join(absDest, hdr.Name)
		if !pathInside(absDest, target) {
			return fmt.Errorf("tar entry escapes destDir: %s", hdr.Name)
		}

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
			// Resolve the link target relative to the symlink's parent dir
			// (absolute targets always escape staging), then ensure the
			// resolved path stays inside destDir.
			resolved := hdr.Linkname
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(filepath.Dir(target), resolved)
			}
			if !pathInside(absDest, resolved) {
				return fmt.Errorf("symlink %q escapes destDir: -> %s", hdr.Name, hdr.Linkname)
			}
			symlinks = append(symlinks, pendingSymlink{target: target, linkname: hdr.Linkname})
		default:
			// Skip hardlinks, fifos, char/block devices.
		}
	}

	for _, s := range symlinks {
		if err := os.MkdirAll(filepath.Dir(s.target), 0o755); err != nil {
			return err
		}
		_ = os.Remove(s.target)
		if err := os.Symlink(s.linkname, s.target); err != nil {
			return err
		}
	}
	return nil
}

// pathInside reports whether p resolves to absRoot itself or somewhere
// underneath it. absRoot must already be absolute.
func pathInside(absRoot, p string) bool {
	abs, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
