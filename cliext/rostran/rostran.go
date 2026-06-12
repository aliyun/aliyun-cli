package rostran

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

type Context struct {
	originCtx          *cli.Context
	configPath         string
	installDirPath     string
	execFilePath       string
	installed          bool
	versionLocal       string
	versionRemote      string
	osType             string
	osArch             string
	osSupport          bool
	downloadPathSuffix string
	envMap             map[string]string
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

var (
	getLatestRostranVersionFunc = GetLatestRostranVersion
	downloadAndExtractFunc      = DownloadAndExtract
	execCommandFunc             = exec.Command
	httpGetFunc                 = downloadHTTPGet
	runtimeGOOSFunc             = func() string { return runtime.GOOS }
	runtimeGOARCHFunc           = func() string { return runtime.GOARCH }
	timeNowFunc                 = func() time.Time { return time.Now() }
)

const versionCheckTTL = 24 * time.Hour

const versionCacheFileName = ".rostran_version_check"

type rostranVersionCache struct {
	InstalledVersion string `json:"installed_version"`
	LastKnownRemote  string `json:"last_known_remote"`
	LastRemoteCheck  int64  `json:"last_remote_check"`
}

func downloadHTTPGet(url string) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Minute}
	return client.Get(url)
}

var rostranBaseUrl = "https://ros-public-tools.oss-cn-beijing.aliyuncs.com/github-releases/aliyun/alibabacloud-ros-tool-transformer/"

func NewContext(originContext *cli.Context) *Context {
	return &Context{
		originCtx: originContext,
	}
}

func (c *Context) Run(args []string) error {
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if !c.osSupport {
		return fmt.Errorf("your os type %s and arch %s is not supported now", c.osType, c.osArch)
	}

	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}

	if len(newArgs) > 0 && newArgs[0] == "upgrade" {
		return c.SelfUpgrade()
	}

	if !c.installed {
		latestVersionRemote, err := getLatestRostranVersionFunc()
		if err != nil {
			return err
		}
		c.versionRemote = latestVersionRemote
		err = c.Install()
		if err != nil {
			return err
		}
		c.installed = true
		c.writeVersionCache(latestVersionRemote, latestVersionRemote)
	}

	err = c.PrepareEnv()
	if err != nil {
		return err
	}

	cmd := execCommandFunc(c.execFilePath, newArgs...)
	envs := os.Environ()
	for k, v := range c.envMap {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = envs
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute %s %v: %v", c.execFilePath, newArgs, err)
	}

	c.maybeShowUpgradeHint()
	return nil
}

func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()

	switch {
	case c.osType == "linux" && c.osArch == "amd64":
		c.osSupport = true
		c.downloadPathSuffix = "linux-amd64.tar.gz"
	case c.osType == "darwin" && c.osArch == "arm64":
		c.osSupport = true
		c.downloadPathSuffix = "darwin-arm64.tar.gz"
	}
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.installDirPath = filepath.Join(c.configPath, "rostran")
	if execPath, ok := findRostranExecutable(c.installDirPath); ok {
		c.execFilePath = execPath
		c.installed = true
		return
	}
	c.execFilePath = filepath.Join(c.installDirPath, "rostran")
	c.installed = false
}

func (c *Context) Install() error {
	url := fmt.Sprintf("%s%s/rostran-%s-%s", rostranBaseUrl, c.versionRemote, c.versionRemote, c.downloadPathSuffix)

	tmpFile := filepath.Join(os.TempDir(), "rostran_download.tar.gz")

	if fileExists(tmpFile) {
		if err := os.Remove(tmpFile); err != nil {
			return err
		}
	}

	err := downloadAndExtractFunc(url, tmpFile, c.installDirPath)
	if err != nil {
		return fmt.Errorf("failed to download and extract rostran from %s: %v", url, err)
	}

	execPath, ok := findRostranExecutable(c.installDirPath)
	if !ok {
		return fmt.Errorf("extracted rostran executable not found under %s", c.installDirPath)
	}
	c.execFilePath = execPath
	if err := os.Chmod(c.execFilePath, 0755); err != nil {
		return fmt.Errorf("failed to set exec permission for file %s: %v", c.execFilePath, err)
	}
	return nil
}

func DownloadAndExtract(url string, destFile string, installDir string) error {
	resp, err := httpGetFunc(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: status code %d", url, resp.StatusCode)
	}
	out, err := os.Create(destFile)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", destFile, err)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		_ = out.Close()
		return fmt.Errorf("failed to write to file %s: %v", destFile, err)
	}
	if err = out.Close(); err != nil {
		return fmt.Errorf("failed to close file %s: %v", destFile, err)
	}

	destDir := filepath.Dir(destFile)
	extractDir := filepath.Join(destDir, "rostran_extract")
	if fileExists(extractDir) {
		if err := os.RemoveAll(extractDir); err != nil {
			return fmt.Errorf("failed to remove existing dir %s: %v", extractDir, err)
		}
	}

	if err := extractTarGz(destFile, extractDir); err != nil {
		return fmt.Errorf("failed to extract file %s: %v", destFile, err)
	}

	sourceDir, err := packageRootDir(extractDir)
	if err != nil {
		return err
	}

	if fileExists(installDir) {
		if err := os.RemoveAll(installDir); err != nil {
			return fmt.Errorf("failed to remove existing dir %s: %v", installDir, err)
		}
	}
	if err := moveDir(sourceDir, installDir); err != nil {
		return err
	}

	_ = os.Remove(destFile)
	_ = os.RemoveAll(extractDir)
	return nil
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	destClean := filepath.Clean(dest)
	var totalSize int64

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		cleanName, err := sanitizeArchivePath(hdr.Name)
		if err != nil {
			continue
		}
		if cleanName == "" {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeSymlink:
			if !isSafeSymlink(cleanName, hdr.Linkname) {
				continue
			}
			filePath, err := safeJoinUnderDir(destClean, cleanName)
			if err != nil {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return err
			}
			_ = os.Remove(filePath)
			if err := os.Symlink(hdr.Linkname, filePath); err != nil {
				return err
			}
			continue
		case tar.TypeLink:
			continue
		case tar.TypeDir:
			filePath, err := safeJoinUnderDir(destClean, cleanName)
			if err != nil {
				continue
			}
			if err := os.MkdirAll(filePath, 0755); err != nil {
				return err
			}
			continue
		case tar.TypeReg, tar.TypeRegA:
		default:
			continue
		}

		if hdr.Size < 0 || hdr.Size > maxExtractSize {
			continue
		}
		totalSize += hdr.Size
		if totalSize > maxExtractTotalSize {
			return fmt.Errorf("archive exceeds maximum extracted size")
		}

		filePath, err := safeJoinUnderDir(destClean, cleanName)
		if err != nil {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		mode := hdr.FileInfo().Mode().Perm()
		if mode == 0 {
			mode = 0600
		}
		outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			return err
		}
		written, copyErr := io.Copy(outFile, io.LimitReader(tr, maxExtractSize+1))
		closeErr := outFile.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if written > maxExtractSize {
			return fmt.Errorf("archive entry %s exceeds maximum extracted size", cleanName)
		}
	}
	return nil
}

const maxExtractSize int64 = 200 << 20
const maxExtractTotalSize int64 = 500 << 20

func packageRootDir(extractDir string) (string, error) {
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", fmt.Errorf("failed to read extracted dir %s: %v", extractDir, err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("extracted dir %s is empty", extractDir)
	}
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(extractDir, entries[0].Name()), nil
	}
	return extractDir, nil
}

func moveDir(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create parent dir for %s: %v", dst, err)
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyDir(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			linkName, err := os.Readlink(p)
			if err != nil {
				return err
			}
			cleanRel := filepath.ToSlash(rel)
			if !isSafeSymlink(cleanRel, linkName) {
				return nil
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			_ = os.Remove(target)
			return os.Symlink(linkName, target)
		}
		return copyFileWithMode(p, target, info.Mode().Perm())
	})
}

func copyFileWithMode(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %v", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %v", dst, err)
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %v", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %v", src, dst, err)
	}
	return nil
}

func findRostranExecutable(root string) (string, bool) {
	if !fileExists(root) {
		return "", false
	}

	var found string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || found != "" {
			return nil
		}
		if filepath.Base(p) == "rostran" {
			found = p
		}
		return nil
	})
	return found, found != ""
}

func sanitizeArchivePath(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty entry name")
	}
	if strings.ContainsRune(name, '\x00') {
		return "", fmt.Errorf("NUL byte in entry name")
	}

	name = strings.ReplaceAll(name, "\\", "/")
	for strings.HasPrefix(name, "./") {
		name = strings.TrimPrefix(name, "./")
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "//") {
		return "", fmt.Errorf("absolute/UNC path is not allowed")
	}
	if len(name) >= 2 && isASCIIAlpha(name[0]) && name[1] == ':' {
		return "", fmt.Errorf("windows drive path is not allowed")
	}

	clean := path.Clean(name)
	if clean == "." {
		return "", nil
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path traversal is not allowed")
	}
	return clean, nil
}

func safeJoinUnderDir(destDir, cleanRel string) (string, error) {
	destClean := filepath.Clean(destDir)
	joined := filepath.Join(destClean, filepath.FromSlash(cleanRel))
	rel, err := filepath.Rel(destClean, joined)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes destination directory")
	}
	return joined, nil
}

func isASCIIAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isSafeSymlink(cleanName, linkName string) bool {
	if linkName == "" || strings.ContainsRune(linkName, '\x00') {
		return false
	}

	linkName = strings.ReplaceAll(linkName, "\\", "/")
	if strings.HasPrefix(linkName, "/") || strings.HasPrefix(linkName, "//") {
		return false
	}
	if len(linkName) >= 2 && isASCIIAlpha(linkName[0]) && linkName[1] == ':' {
		return false
	}

	linkTarget := path.Clean(path.Join(path.Dir(cleanName), linkName))
	if linkTarget == ".." || strings.HasPrefix(linkTarget, "../") {
		return false
	}
	return firstPathComponent(linkTarget) == firstPathComponent(cleanName)
}

func firstPathComponent(p string) string {
	if p == "" || p == "." {
		return ""
	}
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return p
}

func (c *Context) GetLocalVersion() error {
	if !c.installed {
		c.versionLocal = ""
		return fmt.Errorf("rostran not installed")
	}
	cmd := execCommandFunc(c.execFilePath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute %s --version: %v", c.execFilePath, err)
	}
	c.versionLocal = strings.TrimSpace(string(output))
	return nil
}

func (c *Context) SelfUpgrade() error {
	out := c.originCtx.Stdout()

	latestVersionRemote, err := getLatestRostranVersionFunc()
	if err != nil {
		return err
	}
	c.versionRemote = latestVersionRemote

	if c.installed {
		if err := c.GetLocalVersion(); err != nil {
			return err
		}
		fmt.Fprintf(out, "Current installed rostran version: %s\n", c.versionLocal)
	} else {
		fmt.Fprintln(out, "rostran is not installed yet.")
	}
	fmt.Fprintf(out, "Latest available rostran version: %s\n", c.versionRemote)

	if c.installed && c.versionLocal == c.versionRemote {
		fmt.Fprintln(out, "Already up to date.")
		c.writeVersionCache(c.versionLocal, c.versionRemote)
		return nil
	}

	fmt.Fprintf(out, "Installing rostran %s ...\n", c.versionRemote)
	if err := c.Install(); err != nil {
		return err
	}
	fmt.Fprintf(out, "rostran upgraded to %s successfully.\n", c.versionRemote)
	c.writeVersionCache(c.versionRemote, c.versionRemote)
	return nil
}

func (c *Context) PrepareEnv() error {
	profile, err := config.LoadProfileWithContext(c.originCtx)
	if err != nil {
		return fmt.Errorf("config failed: %s", err.Error())
	}

	var accessKeyId, accessKeySecret, stsToken string

	mode := profile.Mode
	switch mode {
	case config.AK:
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
	case config.StsToken:
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
		stsToken = profile.StsToken
	default:
		proxyHost, ok := c.originCtx.Flags().GetValue("proxy-host")
		if !ok {
			proxyHost = ""
		}
		credential, err := profile.GetCredential(c.originCtx, tea.String(proxyHost))
		if err != nil {
			return fmt.Errorf("can't get credential %s", err)
		}

		model, err := credential.GetCredential()
		if err != nil {
			return fmt.Errorf("can't get credential %s", err)
		}
		accessKeyId = *model.AccessKeyId
		accessKeySecret = *model.AccessKeySecret
		if model.SecurityToken != nil {
			stsToken = *model.SecurityToken
		}
	}

	envMapNew := make(map[string]string)
	envMapNew["ALIBABA_CLOUD_ROSTRAN_COMPAT_MODE"] = "aliyun"
	if accessKeyId != "" {
		envMapNew["ALIBABA_CLOUD_ACCESS_KEY_ID"] = accessKeyId
	}
	if accessKeySecret != "" {
		envMapNew["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] = accessKeySecret
	}
	if profile.RegionId != "" {
		envMapNew["ALIBABA_CLOUD_REGION_ID"] = profile.RegionId
	}
	if stsToken != "" {
		envMapNew["ALIBABA_CLOUD_SECURITY_TOKEN"] = stsToken
	}
	c.envMap = envMapNew
	return nil
}

func (c *Context) versionCachePath() string {
	return filepath.Join(c.configPath, versionCacheFileName)
}

func (c *Context) writeVersionCache(installed, remote string) {
	cache := rostranVersionCache{
		InstalledVersion: installed,
		LastKnownRemote:  remote,
		LastRemoteCheck:  timeNowFunc().Unix(),
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	_ = os.WriteFile(c.versionCachePath(), data, 0600)
}

func (c *Context) readVersionCache() (rostranVersionCache, bool) {
	var cache rostranVersionCache
	data, err := os.ReadFile(c.versionCachePath())
	if err != nil {
		return cache, false
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return cache, false
	}
	return cache, true
}

func (c *Context) maybeShowUpgradeHint() {
	if !c.installed {
		return
	}

	cache, ok := c.readVersionCache()
	if !ok {
		c.bootstrapVersionCache()
		return
	}

	now := timeNowFunc().Unix()
	if now-cache.LastRemoteCheck > int64(versionCheckTTL.Seconds()) {
		if remote, err := getLatestRostranVersionFunc(); err == nil && remote != "" {
			cache.LastKnownRemote = remote
			cache.LastRemoteCheck = now
			if data, err := json.Marshal(cache); err == nil {
				_ = os.WriteFile(c.versionCachePath(), data, 0600)
			}
		}
	}

	c.printUpgradeHint(cache.InstalledVersion, cache.LastKnownRemote)
}

func (c *Context) bootstrapVersionCache() {
	if err := c.GetLocalVersion(); err != nil || c.versionLocal == "" {
		return
	}
	remote, err := getLatestRostranVersionFunc()
	if err != nil || remote == "" {
		return
	}
	c.writeVersionCache(c.versionLocal, remote)
	c.printUpgradeHint(c.versionLocal, remote)
}

func (c *Context) printUpgradeHint(installed, remote string) {
	if remote == "" || installed == "" || remote == installed {
		return
	}
	fmt.Fprintf(c.originCtx.Stderr(),
		"\nNote: a newer rostran version %s is available (you have %s). Run `aliyun rostran upgrade` to update.\n",
		remote, installed)
}

func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
	allFlags := cli.NewFlagSet()
	config.AddFlags(allFlags)
	openapi.AddFlags(allFlags)

	longNeedsValue := make(map[string]bool)
	shortNeedsValue := make(map[string]bool)
	for _, f := range allFlags.Flags() {
		needsValue := f.AssignedMode != cli.AssignedNone
		if f.Name != "" {
			longNeedsValue["--"+f.Name] = needsValue
		}
		for _, alias := range f.Aliases {
			longNeedsValue["--"+alias] = needsValue
		}
		if f.Shorthand != 0 {
			shortNeedsValue["-"+string(f.Shorthand)] = needsValue
		}
	}

	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		argName := a
		hasInlineValue := false
		if prefix, _, ok := cli.SplitStringWithPrefix(a, "=:"); ok {
			argName = prefix
			hasInlineValue = true
		}
		if argName == "--version" && !hasInlineValue && (i+1 >= len(args) || strings.HasPrefix(args[i+1], "-")) {
			out = append(out, a)
			continue
		}
		if needs, ok := longNeedsValue[argName]; ok {
			if needs && !hasInlineValue && i+1 < len(args) {
				i++
			}
			continue
		}
		if needs, ok := shortNeedsValue[argName]; ok {
			if needs && !hasInlineValue && i+1 < len(args) {
				i++
			}
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func GetLatestRostranVersion() (string, error) {
	url := rostranBaseUrl + "version.txt"
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %v", url, err)
	}

	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content from %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP request failed with status code %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from %s: %v", url, err)
	}

	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", fmt.Errorf("failed to parse version from response body: empty string")
	}

	return version, nil
}
