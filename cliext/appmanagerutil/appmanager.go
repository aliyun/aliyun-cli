package appmanagerutil

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
)

type Context struct {
	originCtx                 *cli.Context
	configPath                string // aliyun config path
	checkVersionCacheFilePath string // cache file path to store last check version time
	venvPath                  string // venv directory path
	pythonPath                string // detected python3 path (system or embedded)
	venvPythonPath            string // python inside venv
	venvPipPath               string // pip inside venv
	execFilePath              string // appmanager entry point inside venv
	embeddedPythonDir         string // embedded python install directory
	installed                 bool   // whether appmanager-cli is installed in venv
	versionLocal              string
	osType                    string
	osArch                    string
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

var (
	execCommandFunc   = exec.Command
	timeNowFunc       = time.Now
	runtimeGOOSFunc   = func() string { return runtime.GOOS }
	runtimeGOARCHFunc = func() string { return runtime.GOARCH }
)

const (
	// PyPI 包名
	pypiPackageName = "appmanager-cli"
	// pip 包名（下划线形式，用于 whl 文件名）
	pipPackageNameUnderscore = "appmanager_cli"
	// venv 目录名
	venvDirName = "appmanager-venv"
	// 嵌入式 Python 目录名
	embeddedPythonDirName = "python-embedded"
	// 最低 Python 版本要求
	minPythonVersion = "3.10"
	// 嵌入式 Python 版本（python-build-standalone）
	embeddedPythonVersion = "3.12"
	// 下载基础 URL
	downloadBaseURL = "https://aliyun-cli-pub.oss-cn-hangzhou.aliyuncs.com/cli-ext/appmanager-cli/downloads"
)

var VersionCheckTTL = 86400 // 1 day, in seconds

func NewContext(originContext *cli.Context) *Context {
	return &Context{
		originCtx: originContext,
	}
}

func (c *Context) Run(args []string) error {
	err := c.InitBasicInfo()
	if err != nil {
		return err
	}

	err = c.EnsurePythonAvailable()
	if err != nil {
		return err
	}

	err = c.EnsureVenvAndPackage()
	if err != nil {
		return err
	}

	// KeepArgs: true 让 cli 框架跳过了 flag 解析，
	// 这里手动把命令行 aliyun 主程序 config 类 flag（如 --access-key-id / --region 等）
	// 同步到 ctx.Flags()，使 LoadProfileWithContext 能正确合并命令行覆盖值。
	c.applyMainCliFlagsFromArgs(args)

	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}

	err = c.ExecuteAppManagerCli(newArgs)
	if err != nil {
		return err
	}
	return nil
}

// applyMainCliFlagsFromArgs 扫描 args，将 aliyun 主程序 config 类 flag 的值
// 同步到 c.originCtx.Flags()，使后续 LoadProfileWithContext 能拿到命令行覆盖值。
// 仅处理已注册在 ctx.Flags() 中的 flag（即通过 NewAppManagerCommand 中的
// config.AddFlags(cmd.Flags()) 注册的 flag），不会污染未知 flag。
func (c *Context) applyMainCliFlagsFromArgs(args []string) {
	if c.originCtx == nil || c.originCtx.Flags() == nil {
		return
	}
	flags := c.originCtx.Flags()
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			continue
		}
		var name, value string
		var hasValue bool
		if idx := strings.Index(a, "="); idx > 0 {
			name = a[2:idx]
			value = a[idx+1:]
			hasValue = true
		} else {
			name = a[2:]
		}
		f := flags.Get(name)
		if f == nil || f.Category != "config" {
			continue
		}
		needsValue := f.AssignedMode != cli.AssignedNone
		if needsValue {
			if !hasValue {
				if i+1 >= len(args) {
					continue
				}
				value = args[i+1]
				i++
			}
			f.SetAssigned(true)
			f.SetValue(value)
		} else {
			f.SetAssigned(true)
		}
	}
}

// InitBasicInfo sets up all paths
func (c *Context) InitBasicInfo() error {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()
	c.configPath = getConfigurePathFunc()
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, ".appmanager_version_check")
	c.venvPath = filepath.Join(c.configPath, venvDirName)
	c.embeddedPythonDir = filepath.Join(c.configPath, embeddedPythonDirName)

	if c.osType == "windows" {
		c.venvPythonPath = filepath.Join(c.venvPath, "Scripts", "python.exe")
		c.venvPipPath = filepath.Join(c.venvPath, "Scripts", "pip.exe")
		c.execFilePath = filepath.Join(c.venvPath, "Scripts", "appmanager.exe")
	} else {
		c.venvPythonPath = filepath.Join(c.venvPath, "bin", "python3")
		c.venvPipPath = filepath.Join(c.venvPath, "bin", "pip3")
		c.execFilePath = filepath.Join(c.venvPath, "bin", "appmanager")
	}

	// check if venv and entry point exist
	c.installed = fileExists(c.execFilePath)
	return nil
}

// EnsurePythonAvailable detects python3 on the system, or auto-downloads embedded Python
func (c *Context) EnsurePythonAvailable() error {
	// Step 1: Check if embedded Python already exists (previously downloaded)
	embeddedPython := c.getEmbeddedPythonPath()
	if fileExists(embeddedPython) {
		c.pythonPath = embeddedPython
		return nil
	}

	// Step 2: Try system python3/python
	candidates := []string{"python3", "python"}
	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		version, err := c.getPythonVersion(path)
		if err != nil {
			continue
		}
		if c.isPythonVersionSufficient(version) {
			c.pythonPath = path
			return nil
		}
	}

	// Step 3: No suitable Python found, auto-download from OSS
	fmt.Fprintf(c.originCtx.Stderr(), "Python %s+ not found. Downloading Python %s runtime...\n", minPythonVersion, embeddedPythonVersion)
	if err := c.downloadEmbeddedPython(); err != nil {
		return fmt.Errorf("failed to download Python runtime: %v\n"+
			"You can also install Python manually:\n"+
			"  macOS:   brew install python@3.12\n"+
			"  Linux:   sudo apt install python3 (or yum install python3)\n"+
			"  Windows: https://www.python.org/downloads/", err)
	}
	c.pythonPath = embeddedPython
	fmt.Fprintf(c.originCtx.Stderr(), "Python %s runtime installed successfully.\n", embeddedPythonVersion)
	return nil
}

// getEmbeddedPythonPath returns the expected path of the embedded python binary
func (c *Context) getEmbeddedPythonPath() string {
	if c.osType == "windows" {
		return filepath.Join(c.embeddedPythonDir, "python", "python.exe")
	}
	return filepath.Join(c.embeddedPythonDir, "python", "bin", "python3")
}

// getEmbeddedPythonDownloadURL builds the download URL for current platform
func (c *Context) getEmbeddedPythonDownloadURL() string {
	// Map Go arch to download arch naming
	arch := c.osArch
	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	}

	ext := "tar.gz"
	if c.osType == "windows" {
		ext = "zip"
	}

	// Format: python-3.12-darwin-arm64.tar.gz
	filename := fmt.Sprintf("python-%s-%s-%s.%s", embeddedPythonVersion, c.osType, arch, ext)
	return fmt.Sprintf("%s/%s", downloadBaseURL, filename)
}

// downloadEmbeddedPython downloads and extracts python-build-standalone from OSS
func (c *Context) downloadEmbeddedPython() error {
	downloadURL := c.getEmbeddedPythonDownloadURL()

	if err := os.MkdirAll(c.embeddedPythonDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", c.embeddedPythonDir, err)
	}

	fmt.Fprintf(c.originCtx.Stderr(), "Downloading %s\n", downloadURL)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d from %s", resp.StatusCode, downloadURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Fprintf(c.originCtx.Stderr(), "Extracting Python runtime (%d MB)...\n", len(body)/1024/1024)

	// Extract based on file type
	if c.osType == "windows" {
		err = extractZip(body, c.embeddedPythonDir)
	} else {
		err = extractTarGz(body, c.embeddedPythonDir)
	}
	if err != nil {
		os.RemoveAll(c.embeddedPythonDir)
		return fmt.Errorf("extraction failed: %v", err)
	}

	// Verify python binary exists
	pythonBin := c.getEmbeddedPythonPath()
	if !fileExists(pythonBin) {
		os.RemoveAll(c.embeddedPythonDir)
		return fmt.Errorf("python binary not found at expected path: %s", pythonBin)
	}

	return nil
}

// sanitizeArchivePath validates that the archive entry name resolves within destDir.
// Returns the cleaned absolute path or an error if it escapes.
func sanitizeArchivePath(destDir, entryName string) (string, error) {
	cleanDest := filepath.Clean(destDir)
	target := filepath.Join(cleanDest, entryName)
	rel, err := filepath.Rel(cleanDest, target)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", entryName)
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path traversal detected: %s", entryName)
	}
	return target, nil
}

// extractTarGz extracts a .tar.gz archive to destDir
func extractTarGz(data []byte, destDir string) error {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	cleanDest := filepath.Clean(destDir)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Security: only process regular files and directories.
		// Skip symlinks and all other entry types to prevent path traversal attacks.
		if header.Typeflag != tar.TypeDir && header.Typeflag != tar.TypeReg {
			continue
		}

		// Security: inline path validation to ensure target stays within destDir.
		// Reconstruct targetPath from the validated relative path so that no
		// unresolved archive header data flows directly into file operations.
		target := filepath.Join(cleanDest, header.Name)
		rel, err := filepath.Rel(cleanDest, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		targetPath := filepath.Join(cleanDest, rel)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tarReader); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

// extractZip extracts a .zip archive to destDir
func extractZip(data []byte, destDir string) error {
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	for _, f := range zipReader.File {
		targetPath, err := sanitizeArchivePath(destDir, f.Name)
		if err != nil {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(targetPath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// getPythonVersion returns the version string of a python binary
func (c *Context) getPythonVersion(pythonPath string) (string, error) {
	cmd := execCommandFunc(pythonPath, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	// Output: "Python 3.12.1"
	output := strings.TrimSpace(out.String())
	parts := strings.Fields(output)
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected python version output: %s", output)
	}
	return parts[1], nil
}

// isPythonVersionSufficient checks if version >= minPythonVersion
func (c *Context) isPythonVersionSufficient(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	var major, minor int
	fmt.Sscanf(parts[0], "%d", &major)
	fmt.Sscanf(parts[1], "%d", &minor)

	minParts := strings.Split(minPythonVersion, ".")
	var minMajor, minMinor int
	fmt.Sscanf(minParts[0], "%d", &minMajor)
	fmt.Sscanf(minParts[1], "%d", &minMinor)

	if major > minMajor {
		return true
	}
	if major == minMajor && minor >= minMinor {
		return true
	}
	return false
}

// EnsureVenvAndPackage creates venv if needed and installs/upgrades appmanager-cli
func (c *Context) EnsureVenvAndPackage() error {
	// Create venv if it doesn't exist
	if !fileExists(c.venvPythonPath) {
		fmt.Fprintf(c.originCtx.Stderr(), "Setting up appmanager environment...\n")
		cmd := execCommandFunc(c.pythonPath, "-m", "venv", c.venvPath)
		cmd.Stdout = c.originCtx.Stderr()
		cmd.Stderr = c.originCtx.Stderr()
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create venv at %s: %v", c.venvPath, err)
		}
	}

	// Install or upgrade package
	if !c.installed {
		// First install
		fmt.Fprintf(c.originCtx.Stderr(), "Installing appmanager-cli...\n")
		err := c.pipInstall(false)
		if err != nil {
			return err
		}
		c.installed = true
		_ = c.UpdateCheckCacheTime()
	} else {
		// Check if upgrade is needed (once per day)
		if c.NeedCheckVersion() {
			// Upgrade silently in background-like fashion
			_ = c.pipInstall(true)
			_ = c.UpdateCheckCacheTime()
		}
	}
	return nil
}

// pipInstall runs pip install (or upgrade) for the package.
// Strategy: try OSS first (faster, no PyPI publish needed), if fails, fallback to PyPI.
func (c *Context) pipInstall(upgrade bool) error {
	// Step 1: Try OSS
	whlPath, err := c.downloadWhlFromOSS()
	if err == nil {
		defer os.RemoveAll(filepath.Dir(whlPath))
		args := []string{"install", whlPath}
		if upgrade {
			args = []string{"install", "--upgrade", whlPath}
		}
		cmd := execCommandFunc(c.venvPipPath, args...)
		var stderr bytes.Buffer
		cmd.Stdout = c.originCtx.Stderr()
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			return nil // OSS succeeded
		}
	}

	// Step 2: OSS failed, fallback to PyPI
	fmt.Fprintf(c.originCtx.Stderr(), "OSS not available, trying PyPI...\n")
	args := []string{"install", pypiPackageName}
	if upgrade {
		args = []string{"install", "--upgrade", pypiPackageName}
	}
	cmd := execCommandFunc(c.venvPipPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = c.originCtx.Stderr()
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		return nil // PyPI succeeded
	}

	if upgrade {
		// Upgrade failure is not fatal
		return nil
	}
	return fmt.Errorf("failed to install %s: both OSS and PyPI unavailable", pypiPackageName)
}

// downloadWhlFromOSS downloads the latest whl from OSS bucket.
// It reads version.txt first, then downloads the corresponding whl file.
func (c *Context) downloadWhlFromOSS() (string, error) {
	// Read version.txt from OSS
	versionURL := fmt.Sprintf("%s/version.txt", downloadBaseURL)
	resp, err := http.Get(versionURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch version.txt: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("version.txt not found: HTTP %d", resp.StatusCode)
	}
	versionData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read version.txt: %v", err)
	}
	version := strings.TrimSpace(string(versionData))
	if version == "" {
		return "", fmt.Errorf("version.txt is empty")
	}

	// Download whl: appmanager_cli-{version}-py3-none-any.whl
	whlName := fmt.Sprintf("%s-%s-py3-none-any.whl", pipPackageNameUnderscore, version)
	whlURL := fmt.Sprintf("%s/%s", downloadBaseURL, whlName)

	fmt.Fprintf(c.originCtx.Stderr(), "Downloading %s\n", whlURL)
	whlResp, err := http.Get(whlURL)
	if err != nil {
		return "", fmt.Errorf("failed to download whl: %v", err)
	}
	defer whlResp.Body.Close()
	if whlResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whl download failed: HTTP %d from %s", whlResp.StatusCode, whlURL)
	}

	// Save to temp dir with correct whl filename (pip validates filename)
	tmpDir, err := os.MkdirTemp("", "appmanager-cli-download")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}
	whlPath := filepath.Join(tmpDir, whlName)
	tmpFile, err := os.Create(whlPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	if _, err := io.Copy(tmpFile, whlResp.Body); err != nil {
		tmpFile.Close()
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to save whl: %v", err)
	}
	tmpFile.Close()
	return whlPath, nil
}

func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
	longNeedsValue := make(map[string]bool)  // key: --name
	shortNeedsValue := make(map[string]bool) // key: -x

	// 1) 收集 aliyun 主程序需要剥离的 config 类 flag（如 --access-key-id），
	//    这些 flag 在子命令 ctx.flags 中不一定可见（多数 config flag 未 Persistent），
	//    因此直接通过 config.AddFlags 列出权威名单。
	//    注意：--region 等子进程也需要的 flag 不剥离，通过 passthrough 白名单豁免。
	passthrough := map[string]bool{
		"region":   true,
		"RegionId": true,
		"language": true,
		"endpoint": true,
	}
	mainCliFs := cli.NewFlagSet()
	config.AddFlags(mainCliFs)
	for _, f := range mainCliFs.Flags() {
		if f.Category != "config" {
			continue
		}
		if passthrough[f.Name] {
			continue
		}
		needsValue := f.AssignedMode != cli.AssignedNone
		if f.Name != "" {
			longNeedsValue["--"+f.Name] = needsValue
		}
		if f.Shorthand != 0 {
			shortNeedsValue["-"+string(f.Shorthand)] = needsValue
		}
	}

	// 2) 兼容：再合并 ctx.flags 中已 Assigned 的 config flag（保留旧行为）
	if c.originCtx != nil && c.originCtx.Flags() != nil && c.originCtx.Flags().Flags() != nil {
		for _, f := range c.originCtx.Flags().Flags() {
			if !f.IsAssigned() || f.Category != "config" {
				continue
			}
			if passthrough[f.Name] {
				continue
			}
			needsValue := f.AssignedMode != cli.AssignedNone
			if f.Name != "" {
				longNeedsValue["--"+f.Name] = needsValue
			}
			if f.Shorthand != 0 {
				shortNeedsValue["-"+string(f.Shorthand)] = needsValue
			}
		}
	}

	// 3) 单次扫描，剔除主程序 config flag（同时支持 --name=value 写法）
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			if idx := strings.Index(a, "="); idx > 0 {
				if _, ok := longNeedsValue[a[:idx]]; ok {
					continue
				}
			}
		}
		if needs, ok := longNeedsValue[a]; ok {
			if needs && i+1 < len(args) { // skip value
				i++
			}
			continue
		}
		if needs, ok := shortNeedsValue[a]; ok {
			if needs && i+1 < len(args) { // skip value
				i++
			}
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func (c *Context) NeedCheckVersion() bool {
	if !c.installed {
		return false
	}
	if !fileExists(c.checkVersionCacheFilePath) {
		return true
	}
	// read cache file
	data, err := os.ReadFile(c.checkVersionCacheFilePath)
	if err != nil {
		return true
	}
	var lastCheckTime int64
	_, err = fmt.Sscanf(string(data), "%d", &lastCheckTime)
	if err != nil {
		return true
	}
	currentTime := timeNowFunc().Unix()
	return currentTime-lastCheckTime > int64(VersionCheckTTL)
}

func (c *Context) UpdateCheckCacheTime() error {
	currentTime := timeNowFunc().Unix()
	data := fmt.Sprintf("%d", currentTime)
	err := os.WriteFile(c.checkVersionCacheFilePath, []byte(data), 0644)
	if err != nil {
		return fmt.Errorf("failed to write cache file %s: %v", c.checkVersionCacheFilePath, err)
	}
	return nil
}

// PrepareEnv 从 aliyun CLI 配置中提取凭证，通过环境变量透传给 appmanager-cli 子进程
// 支持 AK 和 StsToken 两种认证模式
func (c *Context) PrepareEnv() ([]string, error) {
	profile, err := config.LoadProfileWithContext(c.originCtx)
	if err != nil {
		// 获取 profile 失败时不阻断，仅继承父进程环境变量
		return os.Environ(), nil
	}

	// LoadProfileWithContext 仅在 InConfigureMode 时才合并命令行 flag，
	// 子命令默认不在 ConfigureMode，这里显式合并一次以支持
	// `aliyun appmanager <cmd> --access-key-id ... --region ...` 这类命令行覆盖。
	if c.originCtx != nil && c.originCtx.Flags() != nil {
		profile.OverwriteWithFlags(c.originCtx)
	}

	var accessKeyId, accessKeySecret, stsToken, bearerToken, bearerTokenHeaderKey string

	mode := profile.Mode
	switch mode {
	case config.AK:
		// 直接凭证：静态 AK/SK
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
	case config.StsToken:
		// 直接凭证：AK/SK + STS Token
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
		stsToken = profile.StsToken
	case config.BearerToken:
		// Bearer Token 模式：直接注入 bearer token，不走 STS 凭证链
		bearerToken = profile.BearerTokenValue
		bearerTokenHeaderKey = profile.BearerTokenHeaderKey
	default:
		// 间接凭证模式（RamRoleArn/OAuth/CloudSSO/OIDC/EcsRamRole/
		// ChainableRamRoleArn/RsaKeyPair/CredentialsURI/External 等）：
		// 统一通过 GetCredential() 获取最终可用的临时 STS 凭证。
		// 其中 OAuth/CloudSSO 会自动用 token 换临时 STS 并处理过期刷新，子进程无感；
		// RamRoleArn 的静态母凭证只用于换取临时 STS，不会泄漏给子进程。
		cred, err := profile.GetCredential(c.originCtx, nil)
		if err != nil {
			return os.Environ(), nil
		}
		model, err := cred.GetCredential()
		if err != nil {
			return os.Environ(), nil
		}
		if model.AccessKeyId != nil {
			accessKeyId = *model.AccessKeyId
		}
		if model.AccessKeySecret != nil {
			accessKeySecret = *model.AccessKeySecret
		}
		if model.SecurityToken != nil {
			stsToken = *model.SecurityToken
		}
	}

	envs := os.Environ()
	if accessKeyId != "" {
		envs = append(envs, "ALIBABA_CLOUD_ACCESS_KEY_ID="+accessKeyId)
	}
	if accessKeySecret != "" {
		envs = append(envs, "ALIBABA_CLOUD_ACCESS_KEY_SECRET="+accessKeySecret)
	}
	if stsToken != "" {
		envs = append(envs, "ALIBABA_CLOUD_SECURITY_TOKEN="+stsToken)
	}
	if bearerToken != "" {
		envs = append(envs, "ALIBABA_CLOUD_BEARER_TOKEN="+bearerToken)
	}
	if bearerTokenHeaderKey != "" {
		envs = append(envs, "ALIBABA_CLOUD_BEARER_TOKEN_HEADER_KEY="+bearerTokenHeaderKey)
	}
	if profile.RegionId != "" {
		envs = append(envs, "ALIBABA_CLOUD_REGION_ID="+profile.RegionId)
	}

	// 标识调用方，让子进程能识别自己是被 aliyun appmanager 调起的
	envs = append(envs, "ALIBABA_CLOUD_APPMANAGER_COMPAT_MODE=aliyun appmanager")

	// 注入 ai-mode.json 中的 user_agent 给 Python SDK（Python SDK 只认 ALIBABA_CLOUD_USER_AGENT）。
	// 父进程已 export ALIBABA_CLOUD_USER_AGENT 时不覆盖，尊重用户显式设置。
	if _, ok := os.LookupEnv("ALIBABA_CLOUD_USER_AGENT"); !ok {
		configDir := config.GetConfigDir(c.originCtx)
		if cfg, err := aimode.Load(configDir); err == nil && cfg != nil {
			if ua := strings.TrimSpace(cfg.UserAgent); ua != "" {
				envs = append(envs, "ALIBABA_CLOUD_USER_AGENT="+ua)
			}
		}
	}

	return envs, nil
}

func (c *Context) ExecuteAppManagerCli(args []string) error {
	cmd := execCommandFunc(c.execFilePath, args...)
	envs, _ := c.PrepareEnv()
	cmd.Env = envs
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin
	// 使用当前工作目录
	wd, _ := os.Getwd()
	cmd.Dir = wd

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute %s %v: %v", c.execFilePath, args, err)
	}
	return nil
}
