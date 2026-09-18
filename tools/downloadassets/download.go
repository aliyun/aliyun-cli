package downloadassets

import (
	"context"
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

const (
	checksumName     = "SHASUMS256.txt"
	defaultTimeout   = 10 * time.Minute
	githubReleaseURL = "https://github.com/aliyun/aliyun-cli/releases/download/v"
)

// AssetNames is the release set finish_release.sh promotes. Order is stable
// so a regenerated checksum file does not reshuffle on rerun.
func AssetNames(version string) []string {
	return []string{
		fmt.Sprintf("aliyun-cli-macosx-%s-amd64.tgz", version),
		fmt.Sprintf("aliyun-cli-macosx-%s-arm64.tgz", version),
		fmt.Sprintf("aliyun-cli-%s.pkg", version),
		fmt.Sprintf("aliyun-cli-macosx-%s-universal.tgz", version),
		fmt.Sprintf("aliyun-cli-linux-%s-amd64.tgz", version),
		fmt.Sprintf("aliyun-cli-linux-%s-arm64.tgz", version),
		fmt.Sprintf("aliyun-cli-windows-%s-amd64.zip", version),
	}
}

// Options controls a download. Tests inject Client and Summer so HTTP and
// checksum failures never depend on the network or the shasum binary.
type Options struct {
	Version string
	BaseURL string
	Token   string
	WorkDir string
	Client  *http.Client
	Summer  func(path string) (string, error)
	Timeout time.Duration
}

// Download fetches every expected asset into a temp directory, writes a new
// checksum file there, and replaces the workdir copies only after the set is
// complete. A failure leaves any existing SHASUMS256.txt untouched.
func Download(ctx context.Context, opt Options) error {
	version, err := cleanVersion(opt.Version)
	if err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	workDir := opt.WorkDir
	if workDir == "" {
		workDir = "."
	}
	info, err := os.Stat(workDir)
	if err != nil {
		return fmt.Errorf("download assets: workdir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("download assets: workdir is not a directory: %s", workDir)
	}

	names := AssetNames(version)
	base := strings.TrimRight(opt.BaseURL, "/")
	if base == "" {
		base = githubReleaseURL + version
	}
	timeout := opt.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	client := opt.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	summer := opt.Summer
	if summer == nil {
		summer = sha256File
	}

	staging, err := os.MkdirTemp(workDir, ".download-assets-")
	if err != nil {
		return fmt.Errorf("download assets: temp dir: %w", err)
	}
	defer os.RemoveAll(staging)

	sums := make([]string, 0, len(names))
	for _, name := range names {
		dest := filepath.Join(staging, name)
		assetURL := base + "/" + name
		if err := downloadFile(ctx, client, assetURL, opt.Token, dest, timeout); err != nil {
			return fmt.Errorf("download assets: %s: %w", name, err)
		}
		if err := ensureAsset(dest); err != nil {
			return fmt.Errorf("download assets: %s: %w", name, err)
		}
		digest, err := summer(dest)
		if err != nil {
			return fmt.Errorf("download assets: checksum %s: %w", name, err)
		}
		line, err := checksumLine(digest, name)
		if err != nil {
			return fmt.Errorf("download assets: checksum %s: %w", name, err)
		}
		sums = append(sums, line)
	}

	sumPath := filepath.Join(staging, checksumName)
	body := strings.Join(sums, "\n") + "\n"
	if err := os.WriteFile(sumPath, []byte(body), 0o644); err != nil {
		return fmt.Errorf("download assets: write checksum: %w", err)
	}
	if err := verifyChecksum(body, names); err != nil {
		return err
	}

	if err := commitAssets(staging, workDir, names); err != nil {
		return err
	}
	return nil
}

func cleanVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return "", fmt.Errorf("download assets: version is required")
	}
	if strings.Contains(version, "/") || strings.Contains(version, "\\") || strings.Contains(version, "..") {
		return "", fmt.Errorf("download assets: invalid version %q", version)
	}
	return version, nil
}

func downloadFile(ctx context.Context, client *http.Client, assetURL, token, dest string, timeout time.Duration) error {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, assetURL, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n == 0 {
		return fmt.Errorf("missing or empty asset")
	}
	return nil
}

func ensureAsset(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing asset")
		}
		return err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("missing or empty asset")
	}
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func checksumLine(digest, name string) (string, error) {
	digest = strings.TrimSpace(digest)
	if len(digest) != 64 || !isHex(digest) {
		return "", fmt.Errorf("invalid digest %q", digest)
	}
	if name == "" || strings.ContainsAny(name, "\r\n") {
		return "", fmt.Errorf("invalid asset name %q", name)
	}
	// Two spaces matches `shasum -a 256`, so `shasum -c SHASUMS256.txt` works.
	return digest + "  " + name, nil
}

func verifyChecksum(body string, names []string) error {
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	if len(lines) != len(names) {
		return fmt.Errorf("download assets: checksum count %d != %d", len(lines), len(names))
	}
	seen := make(map[string]struct{}, len(names))
	for i, line := range lines {
		hash, name, ok := strings.Cut(line, "  ")
		if !ok || len(hash) != 64 || !isHex(hash) || name != names[i] {
			return fmt.Errorf("download assets: checksum line %d does not match %s", i+1, names[i])
		}
		if _, dup := seen[name]; dup {
			return fmt.Errorf("download assets: duplicate checksum for %s", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func commitAssets(staging, workDir string, names []string) error {
	for _, name := range names {
		if err := replaceFile(filepath.Join(staging, name), filepath.Join(workDir, name)); err != nil {
			return fmt.Errorf("download assets: commit %s: %w", name, err)
		}
	}
	if err := replaceFile(filepath.Join(staging, checksumName), filepath.Join(workDir, checksumName)); err != nil {
		return fmt.Errorf("download assets: commit checksum: %w", err)
	}
	return nil
}

func replaceFile(src, dst string) error {
	// Staging lives inside the workdir, so rename stays on one filesystem
	// and replaces any previous asset without appending.
	return os.Rename(src, dst)
}

func isHex(s string) bool {
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
