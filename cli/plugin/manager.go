package plugin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"golang.org/x/mod/semver"
)

const (
	IndexURL = "https://aliyun-cli-pub.oss-cn-hangzhou.aliyuncs.com/plugins/plugin-index.json" // 默认索引地址
)

type Manager struct {
	rootDir string
}

func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	rootDir := filepath.Join(home, ".aliyun", "plugins")
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, err
	}
	return &Manager{rootDir: rootDir}, nil
}

func (m *Manager) GetIndex() (*Index, error) {
	resp, err := http.Get(IndexURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plugin index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch plugin index: status %d", resp.StatusCode)
	}

	var index Index
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("failed to decode plugin index: %w", err)
	}
	return &index, nil
}

func matchPluginName(pluginName, userInput string) bool {
	// Case-insensitive exact match
	if strings.EqualFold(pluginName, userInput) {
		return true
	}

	// short name match: "fc" or "FC" -> "aliyun-cli-fc"
	if strings.EqualFold(pluginName, "aliyun-cli-"+userInput) {
		return true
	}

	// alternative prefix: "aliyun-fc" or "ALIYUN-FC" -> "aliyun-cli-fc"
	alternativeName := strings.Replace(strings.ToLower(userInput), "aliyun-", "aliyun-cli-", 1)
	if strings.EqualFold(pluginName, alternativeName) {
		return true
	}

	return false
}

func isDevVersion(version string) bool {
	if strings.HasPrefix(version, "0.0.") || version == "0.0.1" {
		return true
	}

	lowerVersion := strings.ToLower(version)
	return strings.Contains(lowerVersion, "-dev") || strings.Contains(lowerVersion, "dev")
}

// compareVersion compares two semantic version strings using the official semver library
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if v1 == v2
func compareVersion(v1, v2 string) int {
	// Ensure versions have 'v' prefix for semver library
	if !strings.HasPrefix(v1, "v") {
		v1 = "v" + v1
	}
	if !strings.HasPrefix(v2, "v") {
		v2 = "v" + v2
	}

	return semver.Compare(v1, v2)
}

func (m *Manager) findLocalPlugin(userInput string) (string, *LocalPlugin, error) {
	localManifest, err := m.GetLocalManifest()
	if err != nil {
		return "", nil, err
	}

	for pluginName, plugin := range localManifest.Plugins {
		if matchPluginName(pluginName, userInput) {
			return pluginName, &plugin, nil
		}
	}

	return "", nil, fmt.Errorf("plugin %s not installed", userInput)
}

func (m *Manager) GetLocalManifest() (*LocalManifest, error) {
	path := filepath.Join(m.rootDir, "manifest.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &LocalManifest{Plugins: make(map[string]LocalPlugin)}, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var manifest LocalManifest
	if err := json.NewDecoder(f).Decode(&manifest); err != nil {
		return nil, err
	}
	if manifest.Plugins == nil {
		manifest.Plugins = make(map[string]LocalPlugin)
	}
	return &manifest, nil
}

func (m *Manager) saveLocalManifest(manifest *LocalManifest) error {
	path := filepath.Join(m.rootDir, "manifest.json")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(manifest)
}

func (m *Manager) Install(pluginName, version string) error {
	index, err := m.GetIndex()
	if err != nil {
		return err
	}

	var targetPlugin *PluginInfo
	for _, p := range index.Plugins {
		if matchPluginName(p.Name, pluginName) {
			targetPlugin = &p
			break
		}
	}
	if targetPlugin == nil {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	actualPluginName := targetPlugin.Name

	if version == "" {
		version = targetPlugin.LatestVersion
	}

	verInfo, ok := targetPlugin.Versions[version]
	if !ok {
		return fmt.Errorf("version %s not found for plugin %s", version, actualPluginName)
	}

	// Check minimum CLI version requirement from plugin-index.json metadata
	if verInfo.Metadata != nil && verInfo.Metadata.MinCliVersion != "" {
		currentVersion := cli.GetVersion()

		// Skip version check in development mode
		if isDevVersion(currentVersion) {
			cli.Printf(cli.DefaultStdoutWriter(),
				"Running in development mode (version: %s), skipping version check\n"+
					"   Plugin requires CLI version %s or higher\n",
				currentVersion, verInfo.Metadata.MinCliVersion)
		} else if compareVersion(currentVersion, verInfo.Metadata.MinCliVersion) < 0 {
			return fmt.Errorf(
				"plugin %s version %s requires CLI version %s or higher, but you have %s\n"+
					"Please upgrade the CLI by running: brew upgrade aliyun-cli\n"+
					"Or download the latest version from: https://github.com/aliyun/aliyun-cli/releases",
				actualPluginName,
				version,
				verInfo.Metadata.MinCliVersion,
				currentVersion,
			)
		}
	}

	platform := GetCurrentPlatform()
	platInfo, ok := verInfo.Platforms[platform]
	if !ok {
		return fmt.Errorf("plugin %s version %s not supported on %s", actualPluginName, version, platform)
	}

	downloadURL := platInfo.URL

	// 下载并解压
	cli.Printf(cli.DefaultStdoutWriter(), "Downloading %s %s...\n", actualPluginName, version)

	tmpDir, err := os.MkdirTemp("", "aliyun-plugin-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "plugin.archive")
	if err := downloadFile(downloadURL, archivePath); err != nil {
		return err
	}

	actualChecksum, err := calculateSHA256(archivePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	expectedChecksum := platInfo.Checksum
	if actualChecksum != expectedChecksum {
		return fmt.Errorf(
			"checksum verification failed for plugin %s version %s\n"+
				"Expected: %s\n"+
				"Actual:   %s\n"+
				"The downloaded file may be corrupted or tampered with",
			actualPluginName, version, expectedChecksum, actualChecksum,
		)
	}

	extractDir := filepath.Join(m.rootDir, actualPluginName)
	if err := os.RemoveAll(extractDir); err != nil {
		return err
	}
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}

	if strings.HasSuffix(downloadURL, ".zip") {
		if err := unzip(archivePath, extractDir); err != nil {
			return err
		}
	} else {
		if err := untar(archivePath, extractDir); err != nil {
			return err
		}
	}

	pluginManifestPath := filepath.Join(extractDir, "manifest.json")
	pluginManifestFile, err := os.Open(pluginManifestPath)
	if err != nil {
		return fmt.Errorf("invalid plugin package: manifest.json not found")
	}
	defer pluginManifestFile.Close()

	var pManifest PluginManifest
	if err := json.NewDecoder(pluginManifestFile).Decode(&pManifest); err != nil {
		return fmt.Errorf("invalid plugin manifest: %w", err)
	}

	localManifest, err := m.GetLocalManifest()
	if err != nil {
		return err
	}

	localManifest.Plugins[actualPluginName] = LocalPlugin{
		Name:        actualPluginName,
		Version:     version,
		Path:        extractDir,
		Command:     pManifest.Command,
		Description: pManifest.ShortDescription,
	}

	if err := m.saveLocalManifest(localManifest); err != nil {
		return err
	}

	cli.Printf(cli.DefaultStdoutWriter(), "Plugin %s %s installed successfully!\n", actualPluginName, version)
	return nil
}

func (m *Manager) Upgrade(pluginName string) error {
	actualPluginName, localPlugin, err := m.findLocalPlugin(pluginName)
	if err != nil {
		return err
	}

	index, err := m.GetIndex()
	if err != nil {
		return err
	}

	var targetPlugin *PluginInfo
	for _, p := range index.Plugins {
		if matchPluginName(p.Name, pluginName) {
			targetPlugin = &p
			break
		}
	}
	if targetPlugin == nil {
		return fmt.Errorf("plugin %s not found in repository", pluginName)
	}

	if localPlugin.Version == targetPlugin.LatestVersion {
		cli.Printf(cli.DefaultStdoutWriter(), "Plugin %s is already up to date (version %s).\n", actualPluginName, localPlugin.Version)
		return nil
	}

	cli.Printf(cli.DefaultStdoutWriter(), "Upgrading plugin %s from %s to %s...\n", actualPluginName, localPlugin.Version, targetPlugin.LatestVersion)
	return m.Install(actualPluginName, targetPlugin.LatestVersion)
}

func (m *Manager) Uninstall(pluginName string) error {
	actualPluginName, plugin, err := m.findLocalPlugin(pluginName)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(plugin.Path); err != nil {
		return fmt.Errorf("failed to remove plugin files: %w", err)
	}

	localManifest, err := m.GetLocalManifest()
	if err != nil {
		return err
	}

	delete(localManifest.Plugins, actualPluginName)
	if err := m.saveLocalManifest(localManifest); err != nil {
		return err
	}

	cli.Printf(cli.DefaultStdoutWriter(), "Plugin %s uninstalled.\n", actualPluginName)
	return nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func untar(src, dest string) error {
	f, err := os.Open(src)
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

		// check if the path is absolute or contains suspicious patterns
		if filepath.IsAbs(header.Name) {
			return fmt.Errorf("illegal absolute path in archive: %s", header.Name)
		}
		if strings.Contains(header.Name, "..") {
			return fmt.Errorf("illegal path with '..' in archive: %s", header.Name)
		}

		target := filepath.Join(dest, header.Name)
		target = filepath.Clean(target)

		// Double-check: ensure the target path is within the destination directory
		destPath := filepath.Clean(dest) + string(os.PathSeparator)
		if !strings.HasPrefix(target, destPath) {
			return fmt.Errorf("illegal file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// check if the path is absolute or contains suspicious patterns
		if filepath.IsAbs(f.Name) {
			return fmt.Errorf("illegal absolute path in archive: %s", f.Name)
		}
		if strings.Contains(f.Name, "..") {
			return fmt.Errorf("illegal path with '..' in archive: %s", f.Name)
		}

		fpath := filepath.Join(dest, f.Name)
		fpath = filepath.Clean(fpath)

		// Double-check: ensure the target path is within the destination directory
		destPath := filepath.Clean(dest) + string(os.PathSeparator)
		if !strings.HasPrefix(fpath, destPath) {
			return fmt.Errorf("illegal file path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (m *Manager) InstallAll() error {
	index, err := m.GetIndex()
	if err != nil {
		return fmt.Errorf("failed to get plugin index: %w", err)
	}

	localManifest, err := m.GetLocalManifest()
	if err != nil {
		return fmt.Errorf("failed to get local manifest: %w", err)
	}

	cli.Printf(cli.DefaultStdoutWriter(), "Found %d plugins in index\n", len(index.Plugins))

	var installed, skipped, failed int
	for _, plugin := range index.Plugins {
		pluginName := plugin.Name

		if _, exists := localManifest.Plugins[pluginName]; exists {
			cli.Printf(cli.DefaultStdoutWriter(), "Skipping %s (already installed)\n", pluginName)
			skipped++
			continue
		}

		cli.Printf(cli.DefaultStdoutWriter(), "Installing %s...\n", pluginName)

		if err := m.Install(pluginName, ""); err != nil {
			cli.Printf(cli.DefaultStdoutWriter(), "Failed to install %s: %v\n", pluginName, err)
			failed++
			continue
		}

		installed++
	}

	cli.Printf(cli.DefaultStdoutWriter(), "\n=== Summary ===\n")
	cli.Printf(cli.DefaultStdoutWriter(), "Installed: %d\n", installed)
	cli.Printf(cli.DefaultStdoutWriter(), "Skipped: %d\n", skipped)
	if failed > 0 {
		cli.Printf(cli.DefaultStdoutWriter(), "Failed: %d\n", failed)
		return fmt.Errorf("%d plugin(s) failed to install", failed)
	}

	return nil
}
