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
	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"golang.org/x/mod/semver"
)

type installerType int

const (
	installerDirect    installerType = iota // binary downloaded directly
	installerHomebrew                       // macOS / Linux Homebrew
	installerLinuxbrew                      // Linuxbrew
)

var (
	ossBaseURL    = "https://aliyuncli.alicdn.com"
	ossVersionURL = ossBaseURL + "/version"
)

var (
	httpClient           = &http.Client{Timeout: 60 * time.Second}
	stdin      io.Reader = os.Stdin

	execCommand         = exec.Command
	detectInstallerFunc = detectInstaller
	resolveExecPathFunc = resolveExecPath

	osExecutable      = os.Executable
	evalSymlinks      = filepath.EvalSymlinks
	readSignatureFile = os.ReadFile

	// publisherKeys are the keys allowed to replace this binary.
	// Tests replace the slice; production uses the keys embedded in the CLI.
	publisherKeys = publishsig.TrustedPublicKeys()

	// clientSignatureVersion selects which releases this binary will install.
	// Tests replace it.
	clientSignatureVersion = publishsig.SignatureVersion
)

func NewUpgradeCommand() *cli.Command {
	cmd := &cli.Command{
		Name: "upgrade",
		Short: i18n.T(
			"upgrade Alibaba Cloud CLI to the latest version",
			"升级阿里云CLI到最新版本"),
		Long: i18n.T(
			"Upgrade Alibaba Cloud CLI to the latest release.\n"+
				"If installed via Homebrew, delegates to 'brew upgrade'.\n"+
				"Otherwise downloads from Alibaba Cloud OSS mirror.\n"+
				"Only a release newer than this CLI and listing this CLI's publisher key generation is installed, after its publisher signature verifies.",
			"升级阿里云CLI到最新版本。\n"+
				"如果通过Homebrew安装，自动调用brew upgrade升级。\n"+
				"否则从阿里云OSS镜像源下载。\n"+
				"只会安装比当前更新、且列出本包公钥版本、并且发行方签名校验通过的版本。"),
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

func upgradeViaDirect(ctx *cli.Context, currentVersion string) error {
	w := ctx.Stdout()

	cli.Printf(w, "Checking for latest version...\n")
	pointer, err := fetchVersionFromOSS()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %s", err)
	}
	listed, catalogPresent, listErr := fetchVersionList()
	if listErr != nil {
		cli.Printf(w, "Version history is unavailable (%s); checking the latest release only.\n", listErr)
		catalogPresent = false
	}
	latest := selectUpgradeVersion(currentVersion, pointer, clientSignatureVersion, listed, catalogPresent)
	if latest == "" {
		cli.PrintfWithColor(w, cli.Green, "You are already using the latest version.\n")
		return nil
	}
	cli.Printf(w, "Latest version:  %s\n\n", latest)

	if !confirmUpgrade(ctx, currentVersion, latest) {
		cli.Printf(w, "Upgrade cancelled.\n")
		return nil
	}

	assetName, err := buildAssetName(latest)
	if err != nil {
		return err
	}
	downloadURL := ossBaseURL + "/" + assetName
	cli.Printf(w, "Downloading %s...\n  From: %s\n", assetName, downloadURL)
	extracted, cleanup, err := downloadAndExtract(w, downloadURL, assetName)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return err
	}
	if !isNewer(currentVersion, latest) {
		return fmt.Errorf("refusing to install %s because it is not newer than %s", latest, currentVersion)
	}

	execPath, err := resolveExecPathFunc()
	if err != nil {
		return err
	}
	cli.Printf(w, "Installing new version to %s ...\n", execPath)
	if err := replaceBinary(extracted, execPath); err != nil {
		return fmt.Errorf("installation failed: %s\nYou may need to run with elevated privileges (sudo)", err)
	}

	cli.PrintfWithColor(w, cli.Green,
		"\nSuccessfully upgraded Alibaba Cloud CLI from %s to %s!\n", currentVersion, latest)
	if isNewer(latest, pointer) {
		cli.Printf(w, "Installed %s, the latest release that supports publisher key generation %s. Published latest is %s. Run `aliyun upgrade` again to continue.\n",
			latest, clientSignatureVersion, strings.TrimSpace(pointer))
	}
	return nil
}

func confirmUpgrade(ctx *cli.Context, currentVersion, latestVersion string) bool {
	yesFlag := ctx.Flags().Get("yes")
	if yesFlag != nil && yesFlag.IsAssigned() {
		return true
	}
	w := ctx.Stdout()
	cli.Printf(w, "Upgrade from %s to %s? (y/N): ", currentVersion, latestVersion)
	reader := bufio.NewReader(stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func downloadAndExtract(w io.Writer, downloadURL, assetName string) (binaryPath string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp("", "aliyun-cli-upgrade-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %s", err)
	}
	cleanup = func() { os.RemoveAll(tmpDir) }

	archivePath := filepath.Join(tmpDir, assetName)
	if err := downloadFile(w, downloadURL, archivePath); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("download failed: %s", err)
	}
	sigPath := archivePath + ".minisig"
	if err := downloadFile(w, downloadURL+".minisig", sigPath); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("publisher signature download failed: %s", err)
	}
	sig, err := readSignatureFile(sigPath)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("publisher signature download failed: %s", err)
	}
	if len(sig) > 4096 {
		cleanup()
		return "", nil, fmt.Errorf("publisher signature rejected: document is too large")
	}
	if err := publishsig.VerifyFile(publisherKeys, assetName, archivePath, sig); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("publisher signature rejected: %s", err)
	}

	binaryName := "aliyun"
	if runtime.GOOS == "windows" {
		binaryName = "aliyun.exe"
	}
	extractedPath := filepath.Join(tmpDir, binaryName)
	if err := extractBinary(archivePath, extractedPath, binaryName); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("extraction failed: %s", err)
	}

	return extractedPath, cleanup, nil
}

func resolveExecPath() (string, error) {
	execPath, err := osExecutable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %s", err)
	}
	execPath, err = evalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %s", err)
	}
	return execPath, nil
}

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

const maxVersionListBytes = 256 * 1024

type publishedVersion struct {
	Version     string   `json:"version"`
	PublicKeys  []string `json:"public_keys"`
	Description string   `json:"description"`
}

// fetchVersionList reads the stable history published beside /version.
// A missing object is not an error: the caller follows /version once.
func fetchVersionList() ([]publishedVersion, bool, error) {
	resp, err := httpClient.Get(ossBaseURL + "/versions")
	if err != nil {
		return nil, false, fmt.Errorf("failed to reach OSS: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("OSS versions file returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxVersionListBytes+1))
	if err != nil {
		return nil, false, err
	}
	if len(body) > maxVersionListBytes {
		return nil, false, fmt.Errorf("OSS versions file is too large")
	}
	listed, err := parseVersionCatalog(body)
	if err != nil {
		return nil, false, err
	}
	return listed, true, nil
}

func parseVersionCatalog(body []byte) ([]publishedVersion, error) {
	var doc struct {
		Format   string             `json:"format"`
		Versions []publishedVersion `json:"versions"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("OSS versions file: %s", err)
	}
	if doc.Format != publishsig.VersionsFormat {
		return nil, fmt.Errorf("OSS versions file: unsupported format %q", doc.Format)
	}
	if doc.Versions == nil {
		return []publishedVersion{}, nil
	}
	return doc.Versions, nil
}

// selectUpgradeVersion returns the single release to install.
// It is the newest catalog entry newer than current, not newer than the
// /version pointer, whose public_keys list contains this binary's generation.
// Support for several generations is read from that list, not from the
// version's position between two other releases.
// When the catalog is absent, the pointer itself is that one release.
func selectUpgradeVersion(current, pointer, signatureVersion string, listed []publishedVersion, catalogPresent bool) string {
	pointer = strings.TrimSpace(pointer)
	if !catalogPresent {
		if pointer != "" && isNewer(current, pointer) {
			return pointer
		}
		return ""
	}
	ceiling, ok := canonicalUpgradeVersion(pointer)
	if !ok {
		return ""
	}
	if _, ok := canonicalUpgradeVersion(current); !ok {
		return ""
	}
	best := ""
	for _, entry := range listed {
		if !listsPublicKey(entry.PublicKeys, signatureVersion) {
			continue
		}
		version, ok := canonicalUpgradeVersion(entry.Version)
		if !ok || !isNewer(current, version) {
			continue
		}
		if semver.Compare("v"+version, "v"+ceiling) > 0 {
			continue
		}
		if best == "" || semver.Compare("v"+version, "v"+best) > 0 {
			best = version
		}
	}
	return best
}

func listsPublicKey(keys []string, want string) bool {
	for _, key := range keys {
		if strings.TrimSpace(key) == want {
			return true
		}
	}
	return false
}

func canonicalUpgradeVersion(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, " \t/\\") {
		return "", false
	}
	if strings.HasPrefix(raw, "v") || strings.HasPrefix(raw, "V") {
		raw = raw[1:]
	}
	if !semver.IsValid("v" + raw) {
		return "", false
	}
	core := raw
	if i := strings.IndexAny(core, "-+"); i >= 0 {
		core = core[:i]
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return "", false
	}
	for _, part := range parts {
		if part == "" {
			return "", false
		}
	}
	return raw, true
}

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

func downloadFile(w io.Writer, url, destPath string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d for %s", resp.StatusCode, url)
	}

	totalSize := resp.ContentLength

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	var downloaded int64
	var lastPct int64 = -1
	buf := make([]byte, 256*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if totalSize > 0 {
				pct := downloaded * 100 / totalSize
				if pct != lastPct {
					lastPct = pct
					cli.Printf(w, "\r  Progress: %d%% (%s / %s)", pct, formatSize(downloaded), formatSize(totalSize))
				}
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

func installerTypeFromExecPathLower(lower string) installerType {
	if strings.Contains(lower, "linuxbrew") {
		return installerLinuxbrew
	}
	if strings.Contains(lower, "homebrew") || strings.Contains(lower, "/cellar/") {
		return installerHomebrew
	}
	return installerDirect
}

func detectInstaller() installerType {
	execPath, err := osExecutable()
	if err != nil {
		return installerDirect
	}
	execPath, _ = evalSymlinks(execPath)
	return installerTypeFromExecPathLower(strings.ToLower(execPath))
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
