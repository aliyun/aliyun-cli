// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"golang.org/x/mod/semver"
)

type installerType int

const (
	installerDirect    installerType = iota // binary downloaded directly
	installerHomebrew                       // macOS / Linux Homebrew
	installerLinuxbrew                      // Linuxbrew
)

const (
	ossBaseURL       = "https://aliyun-cli.oss-cn-hangzhou.aliyuncs.com"
	ossVersionURL    = ossBaseURL + "/version"
	githubReleaseURL = "https://api.github.com/repos/aliyun/aliyun-cli/releases/latest"
)

var (
	httpClient     = &http.Client{Timeout: 60 * time.Second}
	stdin          io.Reader = os.Stdin
	execCommand            = exec.Command // mockable for tests
	detectInstallerFunc    = detectInstaller
)

// upgradeSource holds the resolved version and download info.
type upgradeSource struct {
	latestVersion string
	downloadURL   string
	assetName     string
	assetSize     int64 // 0 if unknown (e.g. OSS path without Content-Length)
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func NewUpgradeCommand() *cli.Command {
	cmd := &cli.Command{
		Name: "upgrade",
		Short: i18n.T(
			"upgrade Alibaba Cloud CLI to the latest version",
			"升级阿里云CLI到最新版本"),
		Long: i18n.T(
			"Upgrade Alibaba Cloud CLI to the latest release.\n"+
				"If installed via Homebrew, delegates to 'brew upgrade'.\n"+
				"Otherwise downloads from Alibaba Cloud OSS mirror.",
			"升级阿里云CLI到最新版本。\n"+
				"如果通过Homebrew安装，自动调用brew upgrade升级。\n"+
				"否则从阿里云OSS镜像源下载。"),
		Usage: "upgrade [--yes]",
		Run: func(ctx *cli.Context, args []string) error {
			return doUpgrade(ctx)
		},
	}

	cmd.Flags().Add(&cli.Flag{
		Name:      "yes",
		Shorthand: 'y',
		Short: i18n.T(
			"skip confirmation prompt",
			"跳过确认提示"),
		AssignedMode: cli.AssignedNone,
	})

	return cmd
}

func doUpgrade(ctx *cli.Context) error {
	w := ctx.Stdout()

	installer := detectInstallerFunc()

	currentVersion := cli.GetVersion()
	cli.Printf(w, "Current version: %s\n", currentVersion)

	switch installer {
	case installerHomebrew, installerLinuxbrew:
		return upgradeViaBrew(ctx)
	default:
		return upgradeViaDirect(ctx, currentVersion)
	}
}

// upgradeViaBrew delegates the upgrade to Homebrew / Linuxbrew.
func upgradeViaBrew(ctx *cli.Context) error {
	w := ctx.Stdout()
	cli.Printf(w, "Detected Homebrew installation, delegating to brew...\n\n")

	cli.Printf(w, "==> brew update\n")
	update := execCommand("brew", "update")
	update.Stdout = w
	update.Stderr = ctx.Stderr()
	if err := update.Run(); err != nil {
		return fmt.Errorf("brew update failed: %s", err)
	}

	cli.Printf(w, "\n==> brew upgrade aliyun-cli\n")
	upgrade := execCommand("brew", "upgrade", "aliyun-cli")
	upgrade.Stdout = w
	upgrade.Stderr = ctx.Stderr()
	if err := upgrade.Run(); err != nil {
		return fmt.Errorf("brew upgrade failed: %s", err)
	}

	cli.PrintfWithColor(w, cli.Green, "\nHomebrew upgrade complete!\n")
	return nil
}

// upgradeViaDirect downloads the binary from OSS and replaces it in-place.
func upgradeViaDirect(ctx *cli.Context, currentVersion string) error {
	w := ctx.Stdout()

	cli.Printf(w, "Checking for latest version...\n")

	source, err := resolveUpgradeSource()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %s", err)
	}

	cli.Printf(w, "Latest version:  %s\n\n", source.latestVersion)

	if !isNewer(currentVersion, source.latestVersion) {
		cli.PrintfWithColor(w, cli.Green, "You are already using the latest version.\n")
		return nil
	}

	yesFlag := ctx.Flags().Get("yes")
	if yesFlag == nil || !yesFlag.IsAssigned() {
		cli.Printf(w, "Upgrade from %s to %s? (y/N): ", currentVersion, source.latestVersion)
		reader := bufio.NewReader(stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			cli.Printf(w, "Upgrade cancelled.\n")
			return nil
		}
	}

	sizeHint := ""
	if source.assetSize > 0 {
		sizeHint = " (" + formatSize(source.assetSize) + ")"
	}
	cli.Printf(w, "Downloading %s%s...\n", source.assetName, sizeHint)
	cli.Printf(w, "  From: %s\n", source.downloadURL)

	tmpDir, err := os.MkdirTemp("", "aliyun-cli-upgrade-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, source.assetName)
	err = downloadFile(w, source.downloadURL, archivePath, source.assetSize)
	if err != nil {
		return fmt.Errorf("download failed: %s", err)
	}

	binaryName := "aliyun"
	if runtime.GOOS == "windows" {
		binaryName = "aliyun.exe"
	}
	extractedPath := filepath.Join(tmpDir, binaryName)
	err = extractBinary(archivePath, extractedPath, binaryName)
	if err != nil {
		return fmt.Errorf("extraction failed: %s", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %s", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %s", err)
	}

	cli.Printf(w, "Installing new version to %s ...\n", execPath)
	err = replaceBinary(extractedPath, execPath)
	if err != nil {
		return fmt.Errorf("installation failed: %s\nYou may need to run with elevated privileges (sudo)", err)
	}

	cli.PrintfWithColor(w, cli.Green,
		"\nSuccessfully upgraded Alibaba Cloud CLI from %s to %s!\n", currentVersion, source.latestVersion)
	return nil
}

// ---------------------------------------------------------------------------
// Version resolution: OSS primary, GitHub fallback
// ---------------------------------------------------------------------------

func resolveUpgradeSource() (*upgradeSource, error) {
	source, ossErr := resolveFromOSS()
	if ossErr == nil {
		return source, nil
	}

	source, ghErr := resolveFromGitHub()
	if ghErr == nil {
		return source, nil
	}

	return nil, fmt.Errorf("OSS: %s; GitHub: %s", ossErr, ghErr)
}

// resolveFromOSS fetches the latest version from the OSS version file and
// constructs a deterministic download URL based on OS/arch.
func resolveFromOSS() (*upgradeSource, error) {
	version, err := fetchVersionFromOSS()
	if err != nil {
		return nil, err
	}

	assetName, err := buildAssetName(version)
	if err != nil {
		return nil, err
	}

	return &upgradeSource{
		latestVersion: version,
		downloadURL:   ossBaseURL + "/" + assetName,
		assetName:     assetName,
	}, nil
}

// resolveFromGitHub queries the GitHub Releases API and still uses the OSS
// mirror for the actual download (the asset name is identical on both).
func resolveFromGitHub() (*upgradeSource, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		return nil, err
	}

	latestVersion := normalizeVersion(release.TagName)

	asset, err := findMatchingAsset(release.Assets)
	if err != nil {
		return nil, err
	}

	return &upgradeSource{
		latestVersion: latestVersion,
		downloadURL:   ossBaseURL + "/" + asset.Name,
		assetName:     asset.Name,
		assetSize:     asset.Size,
	}, nil
}

// ---------------------------------------------------------------------------
// OSS helpers
// ---------------------------------------------------------------------------

func fetchVersionFromOSS() (string, error) {
	resp, err := httpClient.Get(ossVersionURL)
	if err != nil {
		return "", fmt.Errorf("failed to reach OSS: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OSS version file returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}

	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", fmt.Errorf("empty version from OSS")
	}
	return version, nil
}

// buildAssetName constructs the expected release archive name for the current
// platform, following the naming convention in local_build.sh:
//
//	aliyun-cli-{os}-{version}-{arch}.{tgz|zip}
func buildAssetName(version string) (string, error) {
	archName := runtime.GOARCH

	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("aliyun-cli-macosx-%s-%s.tgz", version, archName), nil
	case "linux":
		return fmt.Sprintf("aliyun-cli-linux-%s-%s.tgz", version, archName), nil
	case "windows":
		return fmt.Sprintf("aliyun-cli-windows-%s-%s.zip", version, archName), nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// ---------------------------------------------------------------------------
// GitHub helpers
// ---------------------------------------------------------------------------

func fetchLatestRelease() (*githubRelease, error) {
	req, err := http.NewRequest("GET", githubReleaseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "aliyun-cli/"+cli.GetVersion())

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %s", err)
	}
	if release.TagName == "" {
		return nil, fmt.Errorf("no release tag found")
	}

	return &release, nil
}

func findMatchingAsset(assets []githubAsset) (*githubAsset, error) {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	osKeywords := map[string][]string{
		"darwin":  {"macosx", "darwin"},
		"linux":   {"linux"},
		"windows": {"windows"},
	}

	keywords, ok := osKeywords[osName]
	if !ok {
		return nil, fmt.Errorf("unsupported operating system: %s", osName)
	}

	for _, keyword := range keywords {
		for i := range assets {
			name := strings.ToLower(assets[i].Name)
			if strings.Contains(name, keyword) && strings.Contains(name, archName) {
				return &assets[i], nil
			}
		}
	}

	if osName == "darwin" {
		for i := range assets {
			name := strings.ToLower(assets[i].Name)
			for _, keyword := range keywords {
				if strings.Contains(name, keyword) && strings.Contains(name, "universal") {
					return &assets[i], nil
				}
			}
		}
	}

	return nil, fmt.Errorf(
		"no compatible binary found for %s/%s.\nAvailable assets: %s",
		osName, archName, listAssetNames(assets))
}

// ---------------------------------------------------------------------------
// Download, extract, replace
// ---------------------------------------------------------------------------

func downloadFile(w io.Writer, url, destPath string, totalSize int64) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d for %s", resp.StatusCode, url)
	}

	if totalSize == 0 && resp.ContentLength > 0 {
		totalSize = resp.ContentLength
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if totalSize > 0 {
				pct := float64(downloaded) / float64(totalSize) * 100
				cli.Printf(w, "\r  Progress: %.1f%% (%s / %s)", pct, formatSize(downloaded), formatSize(totalSize))
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	cli.Printf(w, "\r  Download complete.                                  \n")
	return nil
}

func extractBinary(archivePath, destPath, binaryName string) error {
	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".tgz") || strings.HasSuffix(lower, ".tar.gz") {
		return extractFromTarGz(archivePath, destPath, binaryName)
	}
	if strings.HasSuffix(lower, ".zip") {
		return extractFromZip(archivePath, destPath, binaryName)
	}
	return fmt.Errorf("unsupported archive format: %s", filepath.Base(archivePath))
}

func extractFromTarGz(archivePath, destPath, binaryName string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if filepath.Base(header.Name) == binaryName && header.Typeflag == tar.TypeReg {
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			closeErr := out.Close()
			if err != nil {
				return err
			}
			return closeErr
		}
	}

	return fmt.Errorf("binary %q not found in archive", binaryName)
}

func extractFromZip(archivePath, destPath, binaryName string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return err
			}

			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				rc.Close()
				return err
			}

			_, err = io.Copy(out, rc)
			rc.Close()
			closeErr := out.Close()
			if err != nil {
				return err
			}
			return closeErr
		}
	}

	return fmt.Errorf("binary %q not found in archive", binaryName)
}

func replaceBinary(newPath, currentPath string) error {
	info, err := os.Stat(currentPath)
	if err != nil {
		return fmt.Errorf("failed to stat current binary: %s", err)
	}

	dir := filepath.Dir(currentPath)

	if runtime.GOOS == "windows" {
		oldPath := currentPath + ".old"
		os.Remove(oldPath)

		if err := os.Rename(currentPath, oldPath); err != nil {
			return fmt.Errorf("failed to move current binary: %s", err)
		}
		if err := copyFile(newPath, currentPath, info.Mode()); err != nil {
			os.Rename(oldPath, currentPath)
			return err
		}
		os.Remove(oldPath)
	} else {
		tmpPath := filepath.Join(dir, ".aliyun.upgrade.tmp")
		if err := copyFile(newPath, tmpPath, info.Mode()); err != nil {
			os.Remove(tmpPath)
			return err
		}
		if err := os.Rename(tmpPath, currentPath); err != nil {
			os.Remove(tmpPath)
			return err
		}
	}

	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func detectInstaller() installerType {
	execPath, err := os.Executable()
	if err != nil {
		return installerDirect
	}
	execPath, _ = filepath.EvalSymlinks(execPath)
	lower := strings.ToLower(execPath)

	if strings.Contains(lower, "linuxbrew") {
		return installerLinuxbrew
	}
	if strings.Contains(lower, "homebrew") || strings.Contains(lower, "/cellar/") {
		return installerHomebrew
	}
	return installerDirect
}

func normalizeVersion(tag string) string {
	return strings.TrimPrefix(tag, "v")
}

func ensureVPrefix(version string) string {
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

func isNewer(current, latest string) bool {
	cv := ensureVPrefix(current)
	lv := ensureVPrefix(latest)

	if !semver.IsValid(cv) || !semver.IsValid(lv) {
		return current != latest
	}
	return semver.Compare(cv, lv) < 0
}

func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func listAssetNames(assets []githubAsset) string {
	names := make([]string, len(assets))
	for i, a := range assets {
		names[i] = a.Name
	}
	return strings.Join(names, ", ")
}
