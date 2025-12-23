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
	"runtime"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"golang.org/x/mod/semver"
)

const (
	IndexURL = "https://aliyun-cli-pub.oss-cn-hangzhou.aliyuncs.com/plugins/plugin-index.json" // 默认索引地址
)

type Manager struct {
	rootDir  string
	indexURL string // For testing: allows overriding IndexURL
}

func getHomePath() string {
	// Check environment variables first (for testing and user overrides)
	if runtime.GOOS == "windows" {
		// Windows: check USERPROFILE first, then HOMEDRIVE+HOMEPATH, then HOME
		if home := os.Getenv("USERPROFILE"); home != "" {
			return home
		}
		if home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH"); home != "" {
			return home
		}
		if home := os.Getenv("HOME"); home != "" {
			return home
		}
	} else {
		if home := os.Getenv("HOME"); home != "" {
			return home
		}
	}

	// Fallback to os.UserHomeDir() if no environment variable is set
	home, err := os.UserHomeDir()
	if err != nil {
		return "" // Return empty string if all methods fail
	}
	return home
}

func NewManager() (*Manager, error) {
	home := getHomePath()
	if home == "" {
		return nil, fmt.Errorf("home directory not found")
	}
	rootDir := filepath.Join(home, ".aliyun", "plugins")
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, err
	}
	return &Manager{rootDir: rootDir}, nil
}

func (m *Manager) GetIndex() (*Index, error) {
	indexURL := IndexURL
	if m.indexURL != "" {
		indexURL = m.indexURL
	}
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := client.Get(indexURL)
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

	return false
}

func isDevVersion(version string) bool {
	if strings.HasPrefix(version, "0.0.") || version == "0.0.1" {
		return true
	}

	lowerVersion := strings.ToLower(version)
	return strings.Contains(lowerVersion, "-dev") || strings.Contains(lowerVersion, "dev")
}

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

func (m *Manager) findPluginInIndex(pluginName string) (*PluginInfo, error) {
	index, err := m.GetIndex()
	if err != nil {
		return nil, err
	}

	for _, p := range index.Plugins {
		if matchPluginName(p.Name, pluginName) {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("plugin %s not found", pluginName)
}

func (m *Manager) validateVersionAndPlatform(ctx *cli.Context, targetPlugin *PluginInfo, version, actualPluginName string) (*PlatformInfo, error) {
	verInfo, ok := targetPlugin.Versions[version]
	if !ok {
		return nil, fmt.Errorf("version %s not found for plugin %s", version, actualPluginName)
	}

	// Check minimum CLI version requirement
	if verInfo.Metadata != nil && verInfo.Metadata.MinCliVersion != "" {
		currentVersion := cli.GetVersion()

		if isDevVersion(currentVersion) {
			cli.Printf(ctx.Stdout(),
				"Skipping version check, plugin requires CLI version %s or higher\n",
				verInfo.Metadata.MinCliVersion)
		} else if compareVersion(currentVersion, verInfo.Metadata.MinCliVersion) < 0 {
			return nil, fmt.Errorf(
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
		return nil, fmt.Errorf("plugin %s version %s not supported on %s", actualPluginName, version, platform)
	}

	return &platInfo, nil
}

func downloadFile(url, dest string) error {
	client := &http.Client{
		Timeout: 300 * time.Second,
	}
	resp, err := client.Get(url)
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
		// Reject paths starting with / or \ for cross-platform security
		if strings.HasPrefix(header.Name, "/") || strings.HasPrefix(header.Name, "\\") {
			return fmt.Errorf("illegal path starting with separator in archive: %s", header.Name)
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
		// Reject paths starting with / or \ for cross-platform security
		if strings.HasPrefix(f.Name, "/") || strings.HasPrefix(f.Name, "\\") {
			return fmt.Errorf("illegal path starting with separator in archive: %s", f.Name)
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

func (m *Manager) downloadAndVerifyPlugin(ctx *cli.Context, platInfo *PlatformInfo, actualPluginName, version string) (string, error) {
	cli.Printf(ctx.Stdout(), "Downloading %s %s...\n", actualPluginName, version)

	tmpDir, err := os.MkdirTemp("", "aliyun-plugin-")
	if err != nil {
		return "", err
	}

	archivePath := filepath.Join(tmpDir, "plugin.archive")
	if err := downloadFile(platInfo.URL, archivePath); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	actualChecksum, err := calculateSHA256(archivePath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if actualChecksum != platInfo.Checksum {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf(
			"checksum verification failed for plugin %s version %s\n"+
				"Expected: %s\n"+
				"Actual:   %s\n"+
				"The downloaded file may be corrupted or tampered with",
			actualPluginName, version, platInfo.Checksum, actualChecksum,
		)
	}
	return archivePath, nil
}

func (m *Manager) extractPlugin(archivePath, extractDir, downloadURL string) error {
	if err := os.RemoveAll(extractDir); err != nil {
		return err
	}
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}

	if strings.HasSuffix(downloadURL, ".zip") {
		return unzip(archivePath, extractDir)
	}
	return untar(archivePath, extractDir)
}

func (m *Manager) loadAndValidatePluginManifest(extractDir, expectedName string) (*PluginManifest, error) {
	pluginManifestPath := filepath.Join(extractDir, "manifest.json")
	pluginManifestFile, err := os.Open(pluginManifestPath)
	if err != nil {
		return nil, fmt.Errorf("invalid plugin package: manifest.json not found")
	}
	defer pluginManifestFile.Close()

	var pManifest PluginManifest
	if err := json.NewDecoder(pluginManifestFile).Decode(&pManifest); err != nil {
		return nil, fmt.Errorf("invalid plugin manifest: %w", err)
	}

	if pManifest.Name != expectedName {
		return nil, fmt.Errorf("plugin manifest name %s does not match expected name %s", pManifest.Name, expectedName)
	}

	return &pManifest, nil
}

func (m *Manager) savePluginToManifest(actualPluginName, version, extractDir string, pManifest *PluginManifest) error {
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

	return m.saveLocalManifest(localManifest)
}

func (m *Manager) installPlugin(ctx *cli.Context, targetPlugin *PluginInfo, version string) error {
	actualPluginName := targetPlugin.Name

	if version == "" {
		version = targetPlugin.LatestVersion
	}

	platInfo, err := m.validateVersionAndPlatform(ctx, targetPlugin, version, actualPluginName)
	if err != nil {
		return err
	}

	archivePath, err := m.downloadAndVerifyPlugin(ctx, platInfo, actualPluginName, version)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(archivePath))

	extractDir := filepath.Join(m.rootDir, actualPluginName)
	if err := m.extractPlugin(archivePath, extractDir, platInfo.URL); err != nil {
		return err
	}

	pManifest, err := m.loadAndValidatePluginManifest(extractDir, actualPluginName)
	if err != nil {
		return err
	}

	if err := m.savePluginToManifest(actualPluginName, version, extractDir, pManifest); err != nil {
		return err
	}

	cli.Printf(ctx.Stdout(), "Plugin %s %s installed successfully!\n", actualPluginName, version)
	return nil
}

func (m *Manager) Install(ctx *cli.Context, pluginName, version string) error {
	targetPlugin, err := m.findPluginInIndex(pluginName)
	if err != nil {
		return err
	}

	return m.installPlugin(ctx, targetPlugin, version)
}

func (m *Manager) Upgrade(ctx *cli.Context, pluginName string) error {
	actualPluginName, localPlugin, err := m.findLocalPlugin(pluginName)
	if err != nil {
		return err
	}

	targetPlugin, err := m.findPluginInIndex(pluginName)
	if err != nil {
		return fmt.Errorf("plugin %s not found in repository", pluginName)
	}

	if compareVersion(localPlugin.Version, targetPlugin.LatestVersion) >= 0 {
		cli.Printf(ctx.Stdout(), "Plugin %s is already up to date (version %s).\n", actualPluginName, localPlugin.Version)
		return nil
	}

	cli.Printf(ctx.Stdout(), "Upgrading plugin %s from %s to %s...\n", actualPluginName, localPlugin.Version, targetPlugin.LatestVersion)
	return m.installPlugin(ctx, targetPlugin, targetPlugin.LatestVersion)
}

func (m *Manager) Uninstall(ctx *cli.Context, pluginName string) error {
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

	cli.Printf(ctx.Stdout(), "Plugin %s uninstalled.\n", actualPluginName)
	return nil
}

func (m *Manager) UpdateAll(ctx *cli.Context) error {
	localManifest, err := m.GetLocalManifest()
	if err != nil {
		return fmt.Errorf("failed to get local manifest: %w", err)
	}

	if len(localManifest.Plugins) == 0 {
		cli.Printf(ctx.Stdout(), "No plugins installed.\n")
		return nil
	}

	index, err := m.GetIndex()
	if err != nil {
		return fmt.Errorf("failed to get plugin index: %w", err)
	}

	cli.Printf(ctx.Stdout(), "Checking for updates for %d installed plugin(s)...\n", len(localManifest.Plugins))

	var updated, upToDate, failed int
	for pluginName, localPlugin := range localManifest.Plugins {
		var targetPlugin *PluginInfo
		for _, p := range index.Plugins {
			if p.Name == pluginName {
				targetPlugin = &p
				break
			}
		}

		if targetPlugin == nil {
			cli.Printf(ctx.Stdout(), "Skipping %s (not found in repository)\n", pluginName)
			continue
		}

		if compareVersion(localPlugin.Version, targetPlugin.LatestVersion) >= 0 {
			cli.Printf(ctx.Stdout(), "Skipping %s (already up to date: %s)\n", pluginName, localPlugin.Version)
			upToDate++
			continue
		}

		cli.Printf(ctx.Stdout(), "Updating %s from %s to %s...\n", pluginName, localPlugin.Version, targetPlugin.LatestVersion)

		if err := m.installPlugin(ctx, targetPlugin, targetPlugin.LatestVersion); err != nil {
			cli.Printf(ctx.Stdout(), "Failed to update %s: %v\n", pluginName, err)
			failed++
			continue
		}

		updated++
	}

	// if upToDate > 0 {
	// 	cli.Printf(ctx.Stdout(), "Up to date: %d\n", upToDate)
	// }

	if updated == 0 {
		cli.Printf(ctx.Stdout(), "All plugins are up to date.\n")
	} else if updated > 0 {
		cli.Printf(ctx.Stdout(), "Updated: %d\n", updated)
	}

	if failed > 0 {
		cli.Printf(ctx.Stdout(), "Failed: %d\n", failed)
		return fmt.Errorf("%d plugin(s) failed to update", failed)
	}

	return nil
}

func (m *Manager) InstallAll(ctx *cli.Context) error {
	index, err := m.GetIndex()
	if err != nil {
		return fmt.Errorf("failed to get plugin index: %w", err)
	}

	localManifest, err := m.GetLocalManifest()
	if err != nil {
		return fmt.Errorf("failed to get local manifest: %w", err)
	}

	cli.Printf(ctx.Stdout(), "Found %d plugins in index\n", len(index.Plugins))

	var installed, skipped, failed int
	for _, plugin := range index.Plugins {
		pluginName := plugin.Name

		if _, exists := localManifest.Plugins[pluginName]; exists {
			cli.Printf(ctx.Stdout(), "Skipping %s (already installed)\n", pluginName)
			skipped++
			continue
		}

		cli.Printf(ctx.Stdout(), "Installing %s...\n", pluginName)

		if err := m.installPlugin(ctx, &plugin, ""); err != nil {
			cli.Printf(ctx.Stdout(), "Failed to install %s: %v\n", pluginName, err)
			failed++
			continue
		}

		installed++
	}

	if installed > 0 {
		cli.Printf(ctx.Stdout(), "Installed: %d\n", installed)
	}
	if skipped > 0 {
		cli.Printf(ctx.Stdout(), "Skipped: %d\n", skipped)
	}
	if failed > 0 {
		cli.Printf(ctx.Stdout(), "Failed: %d\n", failed)
		return fmt.Errorf("%d plugin(s) failed to install", failed)
	}

	return nil
}
