package ecctl

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/util"
)

type Context struct {
	originCtx                 *cli.Context
	configPath                string
	checkVersionCacheFilePath string
	versionFilePath           string
	execFilePath              string
	installed                 bool
	versionLocal              string
	versionRemote             string
	osType                    string
	osArch                    string
	osSupport                 bool
	useExternalBinary         bool
	archiveExt                string
	envMap                    map[string]string
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

var (
	downloadAndExtractFunc    = DownloadAndExtract
	execCommandFunc           = exec.Command
	timeNowFunc               = time.Now
	runtimeGOOSFunc           = func() string { return runtime.GOOS }
	runtimeGOARCHFunc         = func() string { return runtime.GOARCH }
	getLatestEcctlVersionFunc = GetLatestEcctlVersion
	httpNewRequestFunc        = http.NewRequest
	httpDoFunc                = func(req *http.Request) (*http.Response, error) {
		client := &http.Client{Timeout: 30 * time.Second}
		return client.Do(req)
	}
	httpGetFunc     = http.Get
	osRemoveAllFunc = os.RemoveAll
	osChmodFunc     = os.Chmod
	copyFileFunc    = util.CopyFileAndRemoveSource
	osRemoveFunc    = os.Remove
)

const (
	defaultDownloadBase = "https://ros-public-tools.oss-cn-beijing.aliyuncs.com/github-releases/aliyun/elastic-compute-control-cli"
)

var VersionCheckTTL = 86400 // 1 day, in seconds

var ecctlMaxExtractSize int64 = 200 << 20 // 200MiB, overridable in tests

// ExitError carries the exit code from the child process so the caller
// can propagate it without calling os.Exit directly (which skips defers).
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("subprocess exited with code %d", e.Code)
}

func (e *ExitError) ExitCode() int {
	return e.Code
}

// AppendHelpIfNeeded appends --help when the parent CLI is in help mode but
// the forwarded args do not already request help.
func AppendHelpIfNeeded(isHelp bool, args []string) []string {
	if !isHelp {
		return args
	}
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return args
		}
	}
	return append(args, "--help")
}

func NewContext(originContext *cli.Context) *Context {
	return &Context{originCtx: originContext}
}

func effectiveDownloadBaseURL() string {
	if u := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_ECCTL_DOWNLOAD_BASE_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultDownloadBase
}

func (c *Context) effectiveBaseURL() string {
	return effectiveDownloadBaseURL()
}

func (c *Context) Run(args []string) error {
	if err := c.InitializeAndValidatePlatform(); err != nil {
		return err
	}
	if err := c.EnsureInstalledAndUpdated(); err != nil {
		return err
	}
	c.applyMainCliFlagsFromArgs(args)
	if err := c.PrepareEnv(); err != nil {
		return err
	}
	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}
	return c.ExecuteEcctl(newArgs)
}

func (c *Context) InitializeAndValidatePlatform() error {
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if !c.osSupport && !c.useExternalBinary {
		return fmt.Errorf("your os type %s and arch %s is not supported now", c.osType, c.osArch)
	}
	return nil
}

func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()

	if c.osType != "linux" && c.osType != "darwin" && c.osType != "windows" {
		c.osSupport = false
		c.archiveExt = ""
		return
	}
	if c.osArch != "amd64" && c.osArch != "arm64" {
		c.osSupport = false
		c.archiveExt = ""
		return
	}
	c.osSupport = true
	if c.osType == "windows" {
		c.archiveExt = "zip"
	} else {
		c.archiveExt = "tar.gz"
	}
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, ".ecctl_version_check")
	c.versionFilePath = filepath.Join(c.configPath, ".ecctl_version")
	c.execFilePath = filepath.Join(c.configPath, "ecctl")
	if runtimeGOOSFunc() == "windows" {
		c.execFilePath += ".exe"
	}
	if envPath := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_ECCTL_EXEC_PATH")); envPath != "" {
		c.execFilePath = envPath
		c.useExternalBinary = true
	}
	c.installed = fileExists(c.execFilePath)
}

func (c *Context) usingExecPathOverride() bool {
	return c.useExternalBinary
}

func (c *Context) validateExecPathOverride() error {
	info, err := os.Stat(c.execFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("ALIBABA_CLOUD_ECCTL_EXEC_PATH=%q does not point to an existing file", c.execFilePath)
		}
		return fmt.Errorf("ALIBABA_CLOUD_ECCTL_EXEC_PATH=%q: %v", c.execFilePath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("ALIBABA_CLOUD_ECCTL_EXEC_PATH=%q is not a regular file", c.execFilePath)
	}
	if c.osType != "windows" && info.Mode()&0o111 == 0 {
		return fmt.Errorf("ALIBABA_CLOUD_ECCTL_EXEC_PATH=%q is not executable", c.execFilePath)
	}
	return nil
}

func (c *Context) archiveFileName(version string) string {
	return fmt.Sprintf("ecctl_%s_%s_%s.%s", version, c.osType, c.osArch, c.archiveExt)
}

func (c *Context) archiveDownloadURL(version string) string {
	base := c.effectiveBaseURL()
	return fmt.Sprintf("%s/%s/%s", base, version, c.archiveFileName(version))
}

func GetLatestEcctlVersion() (string, error) {
	url := effectiveDownloadBaseURL() + "/version.txt"

	req, err := httpNewRequestFunc(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %v", url, err)
	}
	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)

	resp, err := httpDoFunc(req)
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
		return "", fmt.Errorf("version.txt is empty")
	}
	return version, nil
}

func (c *Context) EnsureInstalledAndUpdated() error {
	if c.usingExecPathOverride() {
		return c.validateExecPathOverride()
	}

	if !c.installed {
		latestVersion, err := getLatestEcctlVersionFunc()
		if err != nil {
			return fmt.Errorf("ecctl is not installed and auto-download failed: %v", err)
		}
		c.versionRemote = latestVersion
		if err := c.Install(); err != nil {
			return err
		}
		_ = c.UpdateCheckCacheTime()
		return nil
	}

	if os.Getenv("ALIBABA_CLOUD_ECCTL_NO_UPDATE_CHECK") == "1" {
		return nil
	}
	if !c.NeedCheckVersion() {
		return nil
	}

	latestVersion, err := getLatestEcctlVersionFunc()
	if err != nil {
		return nil
	}
	c.versionRemote = latestVersion

	if err := c.GetLocalVersion(); err != nil {
		c.versionLocal = ""
	}
	if c.versionLocal != c.versionRemote {
		if err := c.Install(); err != nil {
			return nil
		}
	}
	_ = c.UpdateCheckCacheTime()
	return nil
}

func (c *Context) Install() error {
	url := c.archiveDownloadURL(c.versionRemote)
	tmpDir := os.TempDir()
	tmpArchive := filepath.Join(tmpDir, "ecctl_download."+c.archiveExt)
	if fileExists(tmpArchive) {
		_ = os.Remove(tmpArchive)
	}
	if err := downloadAndExtractFunc(url, tmpArchive, c.execFilePath); err != nil {
		return fmt.Errorf("failed to download and extract ecctl from %s: %v", url, err)
	}
	c.versionLocal = c.versionRemote
	c.installed = true
	return c.SaveLocalVersion()
}

func DownloadAndExtract(url string, destArchive string, exeFilePath string) error {
	resp, err := httpGetFunc(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: status code %d", url, resp.StatusCode)
	}
	out, err := os.Create(destArchive)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", destArchive, err)
	}
	if _, err = io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		return fmt.Errorf("failed to write to file %s: %v", destArchive, err)
	}
	if err = out.Close(); err != nil {
		return fmt.Errorf("failed to close file %s: %v", destArchive, err)
	}

	destDir := filepath.Join(filepath.Dir(destArchive), "ecctl_extract")
	if fileExists(destDir) {
		if err := osRemoveAllFunc(destDir); err != nil {
			return fmt.Errorf("failed to remove existing dir %s: %v", destDir, err)
		}
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create extract dir %s: %v", destDir, err)
	}

	if strings.HasSuffix(destArchive, ".zip") {
		err = unzipArchive(destArchive, destDir)
	} else if strings.HasSuffix(destArchive, ".tar.gz") {
		err = extractTarGz(destArchive, destDir)
	} else {
		return fmt.Errorf("unsupported archive format for %s", destArchive)
	}
	if err != nil {
		return fmt.Errorf("failed to extract file %s: %v", destArchive, err)
	}

	binaryBase := "ecctl"
	if runtimeGOOSFunc() == "windows" {
		binaryBase += ".exe"
	}
	sourceFile := filepath.Join(destDir, binaryBase)
	if !fileExists(sourceFile) {
		found, walkErr := findEcctlBinary(destDir, binaryBase)
		if walkErr != nil {
			return walkErr
		}
		if found == "" {
			return fmt.Errorf("extracted file %s not exist", binaryBase)
		}
		sourceFile = found
	}

	if fileExists(exeFilePath) {
		if err := osRemoveFunc(exeFilePath); err != nil {
			return fmt.Errorf("failed to remove existing file %s: %v", exeFilePath, err)
		}
	}
	if err := copyFileFunc(sourceFile, exeFilePath); err != nil {
		return err
	}
	if runtimeGOOSFunc() != "windows" {
		if err := osChmodFunc(exeFilePath, 0o755); err != nil {
			return fmt.Errorf("failed to set exec permission for file %s: %v", exeFilePath, err)
		}
	}

	_ = os.Remove(destArchive)
	_ = osRemoveAllFunc(destDir)
	return nil
}

func findEcctlBinary(root, binaryBase string) (string, error) {
	var found string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && filepath.Base(path) == binaryBase {
			if found == "" {
				found = path
			}
		}
		return nil
	})
	return found, err
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
		case tar.TypeSymlink, tar.TypeLink:
			continue
		case tar.TypeDir:
			continue
		case tar.TypeReg, tar.TypeRegA:
			if !isWantedEcctlEntry(cleanName) {
				continue
			}
			if hdr.Size < 0 || hdr.Size > ecctlMaxExtractSize {
				continue
			}

			filePath, err := safeJoinUnderDir(destClean, cleanName)
			if err != nil {
				continue
			}

			if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
				return err
			}

			outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				return err
			}
			if _, err := io.CopyN(outFile, tr, hdr.Size); err != nil {
				outFile.Close()
				return err
			}
			if err := outFile.Close(); err != nil {
				return err
			}
		default:
			continue
		}
	}
	return nil
}

func unzipArchive(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	destClean := filepath.Clean(dest)

	for _, file := range r.File {
		cleanName, err := sanitizeArchivePath(file.Name)
		if err != nil {
			continue
		}
		if cleanName == "" {
			continue
		}
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if !isWantedEcctlEntry(cleanName) {
			continue
		}
		if file.UncompressedSize64 > uint64(ecctlMaxExtractSize) {
			continue
		}

		filePath, err := safeJoinUnderDir(destClean, cleanName)
		if err != nil {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			_ = rc.Close()
			return err
		}

		limit := ecctlMaxExtractSize + 1
		written, copyErr := io.Copy(outFile, io.LimitReader(rc, limit))
		rcErr := rc.Close()
		outErr := outFile.Close()

		if written > ecctlMaxExtractSize {
			_ = os.Remove(filePath)
			continue
		}
		if copyErr != nil {
			return copyErr
		}
		if rcErr != nil {
			return rcErr
		}
		if outErr != nil {
			return outErr
		}
	}
	return nil
}

func isWantedEcctlEntry(cleanName string) bool {
	base := path.Base(cleanName)
	if runtimeGOOSFunc() == "windows" {
		return base == "ecctl.exe"
	}
	return base == "ecctl"
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

func (c *Context) applyMainCliFlagsFromArgs(args []string) {
	if c.originCtx == nil || c.originCtx.Flags() == nil {
		return
	}
	flags := c.originCtx.Flags()
	for i := 0; i < len(args); i++ {
		a := args[i]

		var f *cli.Flag
		var value string
		var hasValue bool

		switch {
		case strings.HasPrefix(a, "--"):
			name := a[2:]
			if idx := strings.Index(a, "="); idx > 0 {
				name = a[2:idx]
				value = a[idx+1:]
				hasValue = true
			}
			f = flags.Get(name)
		case strings.HasPrefix(a, "-") && len(a) > 1:
			f = flags.GetByShorthand(rune(a[1]))
			if rest := a[2:]; rest != "" {
				value = strings.TrimPrefix(rest, "=")
				hasValue = true
			}
		default:
			continue
		}

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

func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
	longNeedsValue := make(map[string]bool)
	shortNeedsValue := make(map[string]bool)

	mainCliFs := cli.NewFlagSet()
	config.AddFlags(mainCliFs)
	for _, f := range mainCliFs.Flags() {
		if f.Category != "config" {
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

	if c.originCtx != nil && c.originCtx.Flags() != nil && c.originCtx.Flags().Flags() != nil {
		for _, f := range c.originCtx.Flags().Flags() {
			if !f.IsAssigned() || f.Category != "config" {
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
			if needs && i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && len(a) >= 2 {
			sh := rune(a[1])
			if f := mainCliFs.GetByShorthand(sh); f != nil && f.Category == "config" {
				if len(a) > 2 {
					continue
				}
				if needs, ok := shortNeedsValue["-"+string(sh)]; ok {
					if needs && i+1 < len(args) {
						i++
					}
					continue
				}
			}
		}
		if needs, ok := shortNeedsValue[a]; ok {
			if needs && i+1 < len(args) {
				i++
			}
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func (c *Context) PrepareEnv() error {
	profile, err := config.LoadProfileWithContext(c.originCtx)
	if err != nil {
		return fmt.Errorf("config failed: %s", err.Error())
	}

	envMap, err := profile.GetRuntimeEnv(c.originCtx)
	if err != nil {
		return fmt.Errorf("failed to get runtime env: %s", err.Error())
	}

	envMap["ALIBABA_CLOUD_ECCTL_COMPAT_MODE"] = "aliyun ecctl"
	c.envMap = envMap
	return nil
}

func (c *Context) ExecuteEcctl(args []string) error {
	http.DefaultClient.CloseIdleConnections()

	cmd := execCommandFunc(c.execFilePath, args...)
	envs := filterEnv(os.Environ(), c.envMap)
	for k, v := range c.envMap {
		envs = append(envs, k+"="+v)
	}
	cmd.Env = envs
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ExitError{Code: exitErr.ExitCode()}
		}
		return fmt.Errorf("failed to execute %s %v: %v", c.execFilePath, args, err)
	}
	return nil
}

func (c *Context) NeedCheckVersion() bool {
	if !c.installed {
		return false
	}
	if !fileExists(c.checkVersionCacheFilePath) {
		return true
	}
	data, err := os.ReadFile(c.checkVersionCacheFilePath)
	if err != nil {
		return true
	}
	var lastCheckTime int64
	if _, err := fmt.Sscanf(string(data), "%d", &lastCheckTime); err != nil {
		return true
	}
	return timeNowFunc().Unix()-lastCheckTime > int64(VersionCheckTTL)
}

func (c *Context) GetLocalVersion() error {
	if !c.installed {
		c.versionLocal = ""
		return fmt.Errorf("ecctl not installed")
	}
	if fileExists(c.versionFilePath) {
		data, err := os.ReadFile(c.versionFilePath)
		if err != nil {
			return fmt.Errorf("failed to read version file %s: %v", c.versionFilePath, err)
		}
		c.versionLocal = strings.TrimSpace(string(data))
		if c.versionLocal == "" {
			return fmt.Errorf("version file %s is empty", c.versionFilePath)
		}
		return nil
	}
	return nil
}

func (c *Context) SaveLocalVersion() error {
	if err := os.WriteFile(c.versionFilePath, []byte(c.versionLocal), 0o644); err != nil {
		return fmt.Errorf("failed to write version file %s: %v", c.versionFilePath, err)
	}
	return nil
}

func (c *Context) UpdateCheckCacheTime() error {
	currentTime := timeNowFunc().Unix()
	data := fmt.Sprintf("%d", currentTime)
	if err := os.WriteFile(c.checkVersionCacheFilePath, []byte(data), 0o644); err != nil {
		return fmt.Errorf("failed to write cache file %s: %v", c.checkVersionCacheFilePath, err)
	}
	return nil
}

func filterEnv(base []string, overrides map[string]string) []string {
	result := make([]string, 0, len(base))
	for _, item := range base {
		key, _, _ := strings.Cut(item, "=")
		if _, conflict := overrides[key]; conflict {
			continue
		}
		result = append(result, item)
	}
	return result
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
