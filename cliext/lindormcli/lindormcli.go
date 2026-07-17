package lindormcli

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
	"github.com/aliyun/aliyun-cli/v3/openapi"
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
	binaryName                string
	envMap                    map[string]string
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

var (
	downloadBinaryFunc              = downloadBinary
	execCommandFunc                 = exec.Command
	timeNowFunc                     = time.Now
	runtimeGOOSFunc                 = func() string { return runtime.GOOS }
	runtimeGOARCHFunc               = func() string { return runtime.GOARCH }
	getLatestLindormCliVersionFunc  = GetLatestLindormCliVersion
	httpDoFunc                      = func(req *http.Request) (*http.Response, error) {
		client := &http.Client{Timeout: 30 * time.Second}
		return client.Do(req)
	}
)

const (
	ossDownloadBase = "https://lindorm-open-api-cli.oss-cn-hangzhou.aliyuncs.com/"
)

var VersionCheckTTL = 86400 // 1 day, in seconds

var platformBinaryNames = map[string]string{
	"linux-amd64":   "lindorm-open-api-cli_linux_amd64",
	"linux-arm64":   "lindorm-open-api-cli_linux_arm64",
	"darwin-amd64":  "lindorm-open-api-cli_darwin_amd64",
	"darwin-arm64":  "lindorm-open-api-cli_darwin_arm64",
	"windows-amd64": "lindorm-open-api-cli_windows_amd64.exe",
}

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

// passthrough lists flags that exist in config/openapi but should NOT be
// stripped — the child process may define its own with the same name.
var passthrough = map[string]struct{}{
	"output":    {},
	"force":     {},
	"version":   {},
	"body":      {},
	"body-file": {},
}

func NewContext(originContext *cli.Context) *Context {
	return &Context{
		originCtx: originContext,
	}
}

func (c *Context) Run(args []string) error {
	err := c.InitializeAndValidatePlatform()
	if err != nil {
		return err
	}

	err = c.EnsureInstalledAndUpdated()
	if err != nil {
		return err
	}

	err = c.PrepareEnv()
	if err != nil {
		return err
	}

	childArgs := c.RemoveFlagsForMainCli(args)
	return c.ExecuteLindormCli(childArgs)
}

func (c *Context) InitializeAndValidatePlatform() error {
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if !c.osSupport {
		return fmt.Errorf("your os type %s and arch %s is not supported now", c.osType, c.osArch)
	}
	return nil
}

func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()

	platformKey := c.osType + "-" + c.osArch
	if name, exists := platformBinaryNames[platformKey]; exists {
		c.osSupport = true
		c.binaryName = name
	} else {
		c.osSupport = false
		c.binaryName = ""
	}
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, ".lindormcli_version_check")
	c.versionFilePath = filepath.Join(c.configPath, ".lindormcli_version")
	c.execFilePath = filepath.Join(c.configPath, "lindorm-open-api-cli")
	if runtimeGOOSFunc() == "windows" {
		c.execFilePath += ".exe"
	}
	c.installed = fileExists(c.execFilePath)
}

func GetLatestLindormCliVersion() (string, error) {
	url := ossDownloadBase + "stable.txt"

	req, err := http.NewRequest("GET", url, nil)
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
	if !c.installed {
		latestVersion, err := getLatestLindormCliVersionFunc()
		if err != nil {
			return fmt.Errorf("lindorm-open-api-cli is not installed and auto-download failed: %v", err)
		}
		c.versionRemote = latestVersion
		if err := c.Install(); err != nil {
			return err
		}
		_ = c.UpdateCheckCacheTime()
		return nil
	}

	if !c.NeedCheckVersion() {
		return nil
	}

	latestVersion, err := getLatestLindormCliVersionFunc()
	if err != nil {
		return nil
	}
	c.versionRemote = latestVersion

	if err := c.GetLocalVersion(); err != nil {
		return nil
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
	url := ossDownloadBase + c.versionRemote + "/" + c.binaryName

	tmpFile := c.execFilePath + ".tmp"
	if err := downloadBinaryFunc(url, tmpFile); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to download lindorm-open-api-cli from %s: %v", url, err)
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
			return fmt.Errorf("failed to install lindorm-open-api-cli binary: %v", copyErr)
		}
	}

	c.versionLocal = c.versionRemote
	c.installed = true
	return c.SaveLocalVersion()
}

func downloadBinary(url string, destFile string) error {
	resp, err := http.Get(url)
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

	envMap["ALIBABA_CLOUD_LINDORM_COMPAT_MODE"] = "aliyun lindorm"

	c.envMap = envMap
	return nil
}

func (c *Context) RemoveFlagsForMainCli(args []string) []string {
	allFlags := cli.NewFlagSet()
	config.AddFlags(allFlags)
	openapi.AddFlags(allFlags)

	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		argName := a
		hasInlineValue := false
		if prefix, _, ok := cli.SplitStringWithPrefix(a, "=:"); ok {
			argName = prefix
			hasInlineValue = true
		}

		var f *cli.Flag
		if strings.HasPrefix(argName, "--") {
			f = allFlags.Get(strings.TrimPrefix(argName, "--"))
		} else if strings.HasPrefix(argName, "-") && len(argName) == 2 {
			f = allFlags.GetByShorthand(rune(argName[1]))
		}

		if f != nil {
			if _, keep := passthrough[f.Name]; keep {
				out = append(out, a)
				continue
			}
			needsValue := f.AssignedMode != cli.AssignedNone
			if needsValue && !hasInlineValue && i+1 < len(args) {
				i++
			}
			continue
		}

		out = append(out, a)
	}
	return out
}

func (c *Context) ExecuteLindormCli(args []string) error {
	// Drain idle HTTP connections from the default transport so that no
	// lingering sockets are inherited by the child process.  On macOS
	// there is a race window between socket() and fcntl(FD_CLOEXEC)
	// that can cause the child to inherit a half-ready fd, leading to
	// "connect: bad file descriptor" errors.
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
	_, err = fmt.Sscanf(string(data), "%d", &lastCheckTime)
	if err != nil {
		return true
	}
	currentTime := timeNowFunc().Unix()
	return currentTime-lastCheckTime > int64(VersionCheckTTL)
}

func (c *Context) GetLocalVersion() error {
	if !c.installed {
		c.versionLocal = ""
		return fmt.Errorf("Lindorm Open API CLI not installed")
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
	err := os.WriteFile(c.versionFilePath, []byte(c.versionLocal), 0644)
	if err != nil {
		return fmt.Errorf("failed to write version file %s: %v", c.versionFilePath, err)
	}
	return nil
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
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
