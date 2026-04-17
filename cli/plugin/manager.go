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
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/pluginsettings"
	"golang.org/x/mod/semver"
)

const (
	IndexURL        = "https://aliyuncli.alicdn.com/plugins/plugin_pkg_index.json"    // 默认索引地址
	CommandIndexURL = "https://aliyuncli.alicdn.com/plugins/plugin_search_index.json" // 命令倒排索引地址
	EnvPluginsDir   = "ALIBABA_CLOUD_CLI_PLUGINS_DIR"
	EnvNoCache      = "ALIBABA_CLOUD_CLI_PLUGIN_NO_CACHE"

	indexCacheFile         = "plugin_pkg_index_cache.json"
	commandCacheFile       = "plugin_search_index_cache.json"
	cacheTTL               = 1 * time.Hour
	fetchTimeout           = 10 * time.Second
	pluginArchiveDLTimeout = 5 * time.Minute
)

type ErrPluginNotFound struct {
	PluginName string
}

func (e *ErrPluginNotFound) Error() string {
	return fmt.Sprintf("plugin %s not found in local manifest (not installed)", e.PluginName)
}

type Manager struct {
	rootDir         string
	sourceBase      string // from plugin-settings.json / env; empty = use built-in index URLs
	indexURL        string // For testing: allows overriding resolved package index URL
	commandIndexURL string // For testing: allows overriding resolved command index URL
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
	rootDir := os.Getenv(EnvPluginsDir)
	if rootDir == "" {
		home := getHomePath()
		if home == "" {
			return nil, fmt.Errorf("home directory not found")
		}
		rootDir = filepath.Join(home, ".aliyun", "plugins")
	}
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, err
	}
	sysDir := filepath.Join(getHomePath(), ".aliyun")
	settings, err := pluginsettings.Load(sysDir)
	if err != nil {
		settings = pluginsettings.Default()
	}
	base := pluginsettings.EffectiveSourceBase(settings)
	return &Manager{rootDir: rootDir, sourceBase: base}, nil
}

func (m *Manager) ApplySourceBaseOverride(raw string) error {
	v := strings.TrimSpace(raw)
	if v == "" {
		return fmt.Errorf("source-base must not be empty")
	}
	lower := strings.ToLower(v)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return fmt.Errorf("source-base must start with http:// or https://")
	}
	m.sourceBase = strings.TrimRight(v, "/")
	return nil
}

func (m *Manager) resolvedPkgIndexURL() string {
	if m.indexURL != "" {
		return m.indexURL
	}
	if b := strings.TrimRight(strings.TrimSpace(m.sourceBase), "/"); b != "" {
		return b + "/plugin_pkg_index.json"
	}
	return IndexURL
}

func (m *Manager) resolvedCommandIndexURL() string {
	if m.commandIndexURL != "" {
		return m.commandIndexURL
	}
	if b := strings.TrimRight(strings.TrimSpace(m.sourceBase), "/"); b != "" {
		return b + "/plugin_search_index.json"
	}
	return CommandIndexURL
}

// common layout: .../pkgs/{name}/{version}/{basename}.
func (m *Manager) resolvePackageDownloadURL(origURL, pluginName, version string) string {
	if strings.TrimSpace(m.sourceBase) == "" {
		return origURL
	}
	u, err := url.Parse(origURL)
	if err != nil {
		return origURL
	}
	baseName := path.Base(u.Path)
	if baseName == "" || baseName == "." || baseName == "/" {
		return origURL
	}
	b := strings.TrimRight(strings.TrimSpace(m.sourceBase), "/")
	return fmt.Sprintf("%s/pkgs/%s/%s/%s", b, pluginName, version, baseName)
}

func (m *Manager) readCache(cacheFile string, ttl time.Duration, result interface{}) (hit bool, staleAvailable bool) {
	info, err := os.Stat(cacheFile)
	if err != nil {
		return false, false
	}
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return false, false
	}
	if err := json.Unmarshal(data, result); err != nil {
		return false, false
	}
	if time.Since(info.ModTime()) <= ttl {
		return true, true
	}
	return false, true
}

func (m *Manager) writeCache(cacheFile string, data []byte) {
	_ = os.WriteFile(cacheFile, data, 0644)
}

func (m *Manager) fetchRemote(url string) ([]byte, error) {
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func noCacheEnabled() bool {
	v := os.Getenv(EnvNoCache)
	return v == "true" || v == "1"
}

func (m *Manager) fetchWithCache(url, cacheFile string, result interface{}) error {
	if noCacheEnabled() {
		data, err := m.fetchRemote(url)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, result)
	}

	hit, staleAvailable := m.readCache(cacheFile, cacheTTL, result)
	if hit {
		return nil
	}

	data, fetchErr := m.fetchRemote(url)
	if fetchErr == nil {
		if decErr := json.Unmarshal(data, result); decErr == nil {
			m.writeCache(cacheFile, data)
			return nil
		} else {
			fetchErr = fmt.Errorf("failed to decode: %w", decErr)
		}
	}

	// Remote failed or decode failed — use stale cache if available
	if staleAvailable {
		return nil
	}
	return fetchErr
}

func (m *Manager) GetIndex() (*Index, error) {
	indexURL := m.resolvedPkgIndexURL()
	cacheFile := filepath.Join(m.rootDir, indexCacheFile)
	var index Index
	if err := m.fetchWithCache(indexURL, cacheFile, &index); err != nil {
		return nil, fmt.Errorf("failed to fetch plugin index: %w", err)
	}
	return &index, nil
}

func (m *Manager) GetCommandIndex() (*CommandIndex, error) {
	commandIndexURL := m.resolvedCommandIndexURL()
	cacheFile := filepath.Join(m.rootDir, commandCacheFile)
	var index CommandIndex
	if err := m.fetchWithCache(commandIndexURL, cacheFile, &index); err != nil {
		return nil, fmt.Errorf("failed to fetch command index: %w", err)
	}
	return &index, nil
}

func (m *Manager) FindPluginByCommand(commandName string) (string, error) {
	index, err := m.GetCommandIndex()
	if err != nil {
		return "", err
	}

	normalizedCmd := strings.ToLower(strings.TrimSpace(commandName))

	if pluginName, found := (*index)[normalizedCmd]; found {
		return pluginName, nil
	}

	return "", fmt.Errorf("no plugin found for command: %s", commandName)
}

func (m *Manager) SearchPluginsByCommandPrefix(commandPrefix string) (map[string][]string, error) {
	index, err := m.GetCommandIndex()
	if err != nil {
		return nil, err
	}

	normalizedPrefix := strings.ToLower(strings.TrimSpace(commandPrefix))
	results := make(map[string][]string)

	for cmd, pluginName := range *index {
		if strings.HasPrefix(cmd, normalizedPrefix) {
			results[pluginName] = append(results[pluginName], cmd)
		}
	}

	return results, nil
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

// FindInstalledPluginInManifest finds an installed plugin when user input
// matches the plugin package name or the short form (user input equals the package name with the "aliyun-cli-" prefix removed, e.g. fc -> aliyun-cli-fc).
// Used for plugins installed from a local path or package URL (--package) that are not listed in the remote plugin index.
func FindInstalledPluginInManifest(manifest *LocalManifest, userInput string) (pluginName string, lp *LocalPlugin, ok bool) {
	if manifest == nil || manifest.Plugins == nil {
		return "", nil, false
	}
	for name, p := range manifest.Plugins {
		if matchPluginName(name, userInput) {
			pl := p
			return name, &pl, true
		}
	}
	return "", nil, false
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

// Pre-release versions contain hyphens (e.g., 1.0.0-alpha, 1.0.0-beta, 1.0.0-rc.1).
func isPrerelease(version string) bool {
	v := version
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return semver.Prerelease(v) != ""
}

func getLatestVersion(plugin *PluginInfo, enablePre bool) (string, error) {
	if len(plugin.Versions) == 0 {
		return "", fmt.Errorf("no versions available for plugin %s", plugin.Name)
	}

	versions := make([]string, 0, len(plugin.Versions))
	for version := range plugin.Versions {
		versions = append(versions, version)
	}

	// Sort versions in descending order (newest first) using semver
	sort.Slice(versions, func(i, j int) bool {
		return compareVersion(versions[i], versions[j]) > 0
	})

	var latestPreRelease string

	for _, version := range versions {
		if enablePre {
			// Return the first version (newest)
			return version, nil
		}
		// Only return stable versions
		if !isPrerelease(version) {
			return version, nil
		}
		if latestPreRelease == "" {
			latestPreRelease = version
		}
	}

	if !enablePre {
		if latestPreRelease != "" {
			// Show the latest pre-release version
			return "", fmt.Errorf("no stable version available for plugin %s. Latest pre-release version: %s. Use --enable-pre to install pre-release versions", plugin.Name, latestPreRelease)
		}
		return "", fmt.Errorf("no stable version available for plugin %s (use --enable-pre to install pre-release versions)", plugin.Name)
	}

	return "", fmt.Errorf("no suitable version found for plugin %s", plugin.Name)
}

func (m *Manager) findLocalPlugin(userInput string) (string, *LocalPlugin, error) {
	localManifest, err := m.GetLocalManifest()
	if err != nil {
		return "", nil, fmt.Errorf("failed to read local plugin manifest: %w", err)
	}

	if name, lp, ok := FindInstalledPluginInManifest(localManifest, userInput); ok {
		return name, lp, nil
	}

	return "", nil, &ErrPluginNotFound{PluginName: userInput}
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

	if strings.HasSuffix(strings.ToLower(downloadURL), ".zip") {
		return unzip(archivePath, extractDir)
	}
	return untar(archivePath, extractDir)
}

func expandPluginSourcePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("source path is empty")
	}
	if strings.HasPrefix(raw, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		raw = filepath.Join(home, raw[2:])
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", fmt.Errorf("resolve source path: %w", err)
	}
	return abs, nil
}

func isPluginArchivePath(absPath string) bool {
	lower := strings.ToLower(absPath)
	return strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz")
}

func isRemotePluginPackageRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	lower := strings.ToLower(ref)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return false
	}
	u, err := url.Parse(ref)
	if err != nil || u.Host == "" {
		return false
	}
	return isPluginArchivePath(u.Path)
}

func readPluginManifestFromDir(extractDir string) (*PluginManifest, error) {
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
	if strings.TrimSpace(pManifest.Name) == "" {
		return nil, fmt.Errorf("invalid plugin manifest: name is empty")
	}
	if strings.TrimSpace(pManifest.Version) == "" {
		return nil, fmt.Errorf("invalid plugin manifest: version is empty")
	}
	return &pManifest, nil
}

func copyDirTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

func (m *Manager) promoteExtractedPlugin(tmpExtract, pluginName string) (string, error) {
	finalDir := filepath.Join(m.rootDir, pluginName)
	if err := os.RemoveAll(finalDir); err != nil {
		return "", fmt.Errorf("failed to remove existing plugin directory: %w", err)
	}
	if err := os.Rename(tmpExtract, finalDir); err == nil {
		return finalDir, nil
	}
	if err := copyDirTree(tmpExtract, finalDir); err != nil {
		return "", fmt.Errorf("failed to copy plugin files: %w", err)
	}
	if err := os.RemoveAll(tmpExtract); err != nil {
		return "", fmt.Errorf("failed to remove temporary extract directory: %w", err)
	}
	return finalDir, nil
}

func (m *Manager) printOverwriteIfPluginInstalled(ctx *cli.Context, pluginName, incomingVersion string) {
	localManifest, err := m.GetLocalManifest()
	if err != nil {
		return
	}
	existing, ok := localManifest.Plugins[pluginName]
	if !ok {
		return
	}
	oldVer := strings.TrimSpace(existing.Version)
	if oldVer == "" {
		oldVer = "unknown"
	}
	cli.Printf(ctx.Stdout(),
		"Note: plugin %q is already installed (version %s); continuing will replace it with version %s.\n",
		pluginName, oldVer, incomingVersion)
}

func (m *Manager) InstallFromPackage(ctx *cli.Context, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("package path or URL is empty")
	}
	lower := strings.ToLower(ref)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		if !isRemotePluginPackageRef(ref) {
			return fmt.Errorf("package URL path must end with .zip, .tar.gz, or .tgz")
		}
		return m.installFromRemotePackageURL(ctx, ref)
	}
	return m.InstallFromLocalFile(ctx, ref)
}

func (m *Manager) InstallFromLocalFile(ctx *cli.Context, sourcePath string) error {
	absPath, err := expandPluginSourcePath(sourcePath)
	if err != nil {
		return err
	}
	st, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("package file: %w", err)
	}
	if st.IsDir() {
		return fmt.Errorf("package must be a plugin archive file (.zip, .tar.gz, .tgz), not a directory")
	}
	return m.installFromPackageFile(ctx, absPath, absPath)
}

func (m *Manager) installFromRemotePackageURL(ctx *cli.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid package URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("package URL must use http or https")
	}
	if !isPluginArchivePath(u.Path) {
		return fmt.Errorf("package URL path must end with .zip, .tar.gz, or .tgz")
	}

	cli.Printf(ctx.Stdout(), "Downloading plugin package from %s...\n", rawURL)

	client := &http.Client{Timeout: pluginArchiveDLTimeout}
	resp, err := client.Get(rawURL)
	if err != nil {
		return fmt.Errorf("download plugin package: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download plugin package: status %d", resp.StatusCode)
	}

	tmpParent, err := os.MkdirTemp("", "aliyun-plugin-dl-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpParent)

	base := path.Base(u.Path)
	if base == "." || base == "/" {
		base = "plugin.tgz"
	}
	dest := filepath.Join(tmpParent, base)
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create temp package file: %w", err)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return fmt.Errorf("write plugin package: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("write plugin package: %w", err)
	}

	return m.installFromPackageFile(ctx, dest, rawURL)
}

func (m *Manager) installFromPackageFile(ctx *cli.Context, absPath, userFacing string) error {
	if !isPluginArchivePath(absPath) {
		return fmt.Errorf("unsupported package format (use .zip, .tar.gz, or .tgz): %s", absPath)
	}

	cli.Printf(ctx.Stdout(), "Installing plugin from %s...\n", userFacing)

	tmpParent, err := os.MkdirTemp("", "aliyun-plugin-local-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpParent)

	tmpExtract := filepath.Join(tmpParent, "extract")
	if err := m.extractPlugin(absPath, tmpExtract, absPath); err != nil {
		return fmt.Errorf("failed to extract plugin package: %w", err)
	}

	pManifest, err := readPluginManifestFromDir(tmpExtract)
	if err != nil {
		return err
	}

	m.printOverwriteIfPluginInstalled(ctx, pManifest.Name, pManifest.Version)

	finalDir, err := m.promoteExtractedPlugin(tmpExtract, pManifest.Name)
	if err != nil {
		return err
	}

	if err := m.savePluginToManifest(pManifest.Name, pManifest.Version, finalDir, pManifest); err != nil {
		return err
	}

	cli.Printf(ctx.Stdout(), "Plugin %s %s installed successfully!\n", pManifest.Name, pManifest.Version)
	return nil
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
		Name:             actualPluginName,
		Version:          version,
		Path:             extractDir,
		ProductCode:      pManifest.ProductCode,
		Command:          pManifest.Command,
		ShortDescription: pManifest.ShortDescription,
		Description:      pManifest.Description,
		CmdNames:         pManifest.CmdNames,
		Inner:            pManifest.Inner,
	}

	return m.saveLocalManifest(localManifest)
}

func (m *Manager) installPlugin(ctx *cli.Context, targetPlugin *PluginInfo, version string, enablePre bool, warnIfAlreadyInstalled bool) error {
	actualPluginName := targetPlugin.Name

	if version == "" {
		// Auto-select version based on enablePre flag
		selectedVersion, err := getLatestVersion(targetPlugin, enablePre)
		if err != nil {
			return err
		}
		version = selectedVersion
		if enablePre && isPrerelease(version) {
			cli.Printf(ctx.Stdout(), "Installing pre-release version %s of plugin %s...\n", version, actualPluginName)
		}
	}

	platInfo, err := m.validateVersionAndPlatform(ctx, targetPlugin, version, actualPluginName)
	if err != nil {
		return err
	}

	downloadURL := m.resolvePackageDownloadURL(platInfo.URL, actualPluginName, version)
	platForDownload := *platInfo
	platForDownload.URL = downloadURL

	archivePath, err := m.downloadAndVerifyPlugin(ctx, &platForDownload, actualPluginName, version)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(archivePath))

	if warnIfAlreadyInstalled {
		m.printOverwriteIfPluginInstalled(ctx, actualPluginName, version)
	}

	extractDir := filepath.Join(m.rootDir, actualPluginName)
	if err := m.extractPlugin(archivePath, extractDir, downloadURL); err != nil {
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

func (m *Manager) Install(ctx *cli.Context, pluginName, version string, enablePre bool) error {
	targetPlugin, err := m.findPluginInIndex(pluginName)
	if err != nil {
		return err
	}

	return m.installPlugin(ctx, targetPlugin, version, enablePre, true)
}

func (m *Manager) Upgrade(ctx *cli.Context, pluginName string, enablePre bool) error {
	actualPluginName, localPlugin, err := m.findLocalPlugin(pluginName)
	if err != nil {
		return err
	}

	targetPlugin, err := m.findPluginInIndex(pluginName)
	if err != nil {
		return fmt.Errorf("plugin %s not found in repository", pluginName)
	}

	latestVersion, err := getLatestVersion(targetPlugin, enablePre)
	if err != nil {
		return err
	}

	if compareVersion(localPlugin.Version, latestVersion) >= 0 {
		cli.Printf(ctx.Stdout(), "Plugin %s is already up to date (version %s).\n", actualPluginName, localPlugin.Version)
		return nil
	}

	if enablePre && isPrerelease(latestVersion) {
		cli.Printf(ctx.Stdout(), "Upgrading plugin %s from %s to %s (pre-release)...\n", actualPluginName, localPlugin.Version, latestVersion)
	} else {
		cli.Printf(ctx.Stdout(), "Upgrading plugin %s from %s to %s...\n", actualPluginName, localPlugin.Version, latestVersion)
	}
	return m.installPlugin(ctx, targetPlugin, latestVersion, enablePre, false)
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

func (m *Manager) UpdateAll(ctx *cli.Context, enablePre bool) error {
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

		latestVersion, err := getLatestVersion(targetPlugin, enablePre)
		if err != nil {
			cli.Printf(ctx.Stdout(), "Skipping %s: %v\n", pluginName, err)
			failed++
			continue
		}

		if compareVersion(localPlugin.Version, latestVersion) >= 0 {
			cli.Printf(ctx.Stdout(), "Skipping %s (already up to date: %s)\n", pluginName, localPlugin.Version)
			upToDate++
			continue
		}

		if enablePre && isPrerelease(latestVersion) {
			cli.Printf(ctx.Stdout(), "Updating %s from %s to %s (pre-release)...\n", pluginName, localPlugin.Version, latestVersion)
		} else {
			cli.Printf(ctx.Stdout(), "Updating %s from %s to %s...\n", pluginName, localPlugin.Version, latestVersion)
		}

		if err := m.installPlugin(ctx, targetPlugin, latestVersion, enablePre, false); err != nil {
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

func (m *Manager) InstallMultiple(ctx *cli.Context, pluginNames []string, version string, enablePre bool) error {
	var installed, failed int

	for _, pluginName := range pluginNames {
		cli.Printf(ctx.Stdout(), "Installing %s...\n", pluginName)
		if err := m.Install(ctx, pluginName, version, enablePre); err != nil {
			cli.Printf(ctx.Stderr(), "Failed to install %s: %v\n", pluginName, err)
			failed++
			continue
		}
		installed++
	}

	if installed > 0 {
		cli.Printf(ctx.Stdout(), "Installed: %d\n", installed)
	}
	if failed > 0 {
		cli.Printf(ctx.Stdout(), "Failed: %d\n", failed)
		return fmt.Errorf("%d plugin(s) failed to install", failed)
	}

	return nil
}

func (m *Manager) InstallAll(ctx *cli.Context, enablePre bool) error {
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

		if err := m.installPlugin(ctx, &plugin, "", enablePre, false); err != nil {
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
