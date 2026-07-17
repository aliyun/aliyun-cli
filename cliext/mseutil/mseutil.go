package mseutil

import (
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
	platformOS                string // OSS path segment: linux/darwin/windows
	platformArch              string // OSS path segment: x86_64/arm64
	envMap                    map[string]string
	useExternalBinary         bool
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

var (
	downloadBinaryFunc         = downloadBinary
	execCommandFunc            = exec.Command
	timeNowFunc                = time.Now
	runtimeGOOSFunc            = func() string { return runtime.GOOS }
	runtimeGOARCHFunc          = func() string { return runtime.GOARCH }
	getRemoteBinaryVersionFunc = GetRemoteBinaryVersion
	httpDoFunc                 = func(req *http.Request) (*http.Response, error) {
		client := &http.Client{Timeout: 30 * time.Second}
		return client.Do(req)
	}
)

// Official mseutil distribution (see help.aliyun.com mseutil docs).
const defaultDownloadBase = "https://msetools.oss-cn-hangzhou.aliyuncs.com/mseutil"

var VersionCheckTTL = 86400 // 1 day, in seconds

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

// platformPaths maps goos-goarch to OSS path segments used by mseutil releases.
// Official URLs use x86_64 (not amd64).
var platformPaths = map[string][2]string{
	"linux-amd64":   {"linux", "x86_64"},
	"linux-arm64":   {"linux", "arm64"},
	"darwin-amd64":  {"darwin", "x86_64"},
	"darwin-arm64":  {"darwin", "arm64"},
	"windows-amd64": {"windows", "x86_64"},
	"windows-arm64": {"windows", "arm64"},
}

func NewContext(originContext *cli.Context) *Context {
	return &Context{
		originCtx: originContext,
	}
}

func (c *Context) Run(args []string) error {
	if err := c.InitializeAndValidatePlatform(); err != nil {
		return err
	}
	if err := c.EnsureInstalledAndUpdated(); err != nil {
		return err
	}
	if err := c.PrepareEnv(); err != nil {
		return err
	}
	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}
	return c.ExecuteMseutil(newArgs)
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
	key := c.osType + "-" + c.osArch
	if parts, ok := platformPaths[key]; ok {
		c.osSupport = true
		c.platformOS = parts[0]
		c.platformArch = parts[1]
	} else {
		c.osSupport = false
		c.platformOS = ""
		c.platformArch = ""
	}
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, ".mseutil_version_check")
	c.versionFilePath = filepath.Join(c.configPath, ".mseutil_version")
	c.execFilePath = filepath.Join(c.configPath, "mseutil")
	if runtimeGOOSFunc() == "windows" {
		c.execFilePath += ".exe"
	}

	if envPath := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_MSEUTIL_EXEC_PATH")); envPath != "" {
		c.execFilePath = envPath
		c.useExternalBinary = true
	}
	c.installed = fileExists(c.execFilePath)
}

func (c *Context) effectiveBaseURL() string {
	if u := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_MSEUTIL_DOWNLOAD_BASE_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultDownloadBase
}

func (c *Context) downloadURL() string {
	return fmt.Sprintf("%s/%s/%s/mseutil", c.effectiveBaseURL(), c.platformOS, c.platformArch)
}

// GetRemoteBinaryVersion returns a fingerprint for the remote binary via HEAD ETag
// (mseutil OSS has no version.txt channel).
func GetRemoteBinaryVersion(downloadURL string) (string, error) {
	req, err := http.NewRequest(http.MethodHead, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %v", downloadURL, err)
	}
	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)

	resp, err := httpDoFunc(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch headers from %s: %v", downloadURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP request failed with status code %d from %s", resp.StatusCode, downloadURL)
	}

	etag := strings.Trim(resp.Header.Get("ETag"), `"`)
	if etag == "" {
		etag = strings.TrimSpace(resp.Header.Get("Content-MD5"))
	}
	if etag == "" {
		etag = strings.TrimSpace(resp.Header.Get("Last-Modified"))
	}
	if etag == "" {
		return "", fmt.Errorf("no ETag/Content-MD5/Last-Modified on %s", downloadURL)
	}
	return etag, nil
}

func (c *Context) EnsureInstalledAndUpdated() error {
	if c.useExternalBinary {
		if !c.installed {
			return fmt.Errorf("mseutil binary not found at ALIBABA_CLOUD_MSEUTIL_EXEC_PATH=%s", c.execFilePath)
		}
		return nil
	}

	if !c.installed {
		version, err := getRemoteBinaryVersionFunc(c.downloadURL())
		if err != nil {
			return fmt.Errorf("mseutil is not installed and auto-download failed: %v", err)
		}
		c.versionRemote = version
		if err := c.Install(); err != nil {
			return err
		}
		_ = c.UpdateCheckCacheTime()
		return nil
	}

	if os.Getenv("ALIBABA_CLOUD_MSEUTIL_NO_UPDATE_CHECK") == "1" {
		return nil
	}
	if !c.NeedCheckVersion() {
		return nil
	}

	version, err := getRemoteBinaryVersionFunc(c.downloadURL())
	if err != nil {
		// upgrade is best-effort when already installed
		return nil
	}
	c.versionRemote = version

	if err := c.GetLocalVersion(); err != nil {
		// missing local fingerprint → treat as need update
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
	url := c.downloadURL()
	if c.originCtx != nil {
		fmt.Fprintf(c.originCtx.Stderr(), "Installing mseutil from %s ...\n", url)
	}

	tmpFile := c.execFilePath + ".tmp"
	if err := downloadBinaryFunc(url, tmpFile); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to download mseutil from %s: %v", url, err)
	}

	if runtimeGOOSFunc() != "windows" {
		if err := os.Chmod(tmpFile, 0755); err != nil {
			_ = os.Remove(tmpFile)
			return fmt.Errorf("failed to set exec permission: %v", err)
		}
	}

	if runtimeGOOSFunc() == "windows" && fileExists(c.execFilePath) {
		_ = os.Remove(c.execFilePath)
	}
	if err := os.Rename(tmpFile, c.execFilePath); err != nil {
		if copyErr := util.CopyFileAndRemoveSource(tmpFile, c.execFilePath); copyErr != nil {
			return fmt.Errorf("failed to install mseutil binary: %v", copyErr)
		}
	}

	c.versionLocal = c.versionRemote
	c.installed = true
	return c.SaveLocalVersion()
}

func downloadBinary(url string, destFile string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %v", url, err)
	}
	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)

	resp, err := httpDoFunc(req)
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
	return nil
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

	if profile.RegionId != "" {
		envMap["REGION_ID"] = profile.RegionId
	}
	envMap["ALIBABA_CLOUD_MSEUTIL_COMPAT_MODE"] = "aliyun mseutil"

	c.envMap = envMap
	return nil
}

func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
	if c.originCtx.Flags() == nil || c.originCtx.Flags().Flags() == nil {
		return append([]string(nil), args...), nil
	}
	longNeedsValue := make(map[string]bool)
	shortNeedsValue := make(map[string]bool)
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

	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if needs, ok := longNeedsValue[a]; ok {
			if needs && i+1 < len(args) {
				i++
			}
			continue
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

func (c *Context) ExecuteMseutil(args []string) error {
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
		return fmt.Errorf("mseutil not installed")
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
	return fmt.Errorf("version file %s not found", c.versionFilePath)
}

func (c *Context) SaveLocalVersion() error {
	if err := os.WriteFile(c.versionFilePath, []byte(c.versionLocal), 0644); err != nil {
		return fmt.Errorf("failed to write version file %s: %v", c.versionFilePath, err)
	}
	return nil
}

func (c *Context) UpdateCheckCacheTime() error {
	data := fmt.Sprintf("%d", timeNowFunc().Unix())
	if err := os.WriteFile(c.checkVersionCacheFilePath, []byte(data), 0644); err != nil {
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
