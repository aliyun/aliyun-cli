package kmscli

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
	downloadBinaryFunc = downloadBinary
	execCommandFunc    = exec.Command
	timeNowFunc        = time.Now
	runtimeGOOSFunc    = func() string { return runtime.GOOS }
	runtimeGOARCHFunc  = func() string { return runtime.GOARCH }
)

const (
	githubDownloadBase = "https://github.com/aliyun/alibabacloud-kms-cli/releases/download"
)

const currentKmscliVersion = "v0.1.0"

var VersionCheckTTL = 86400 // 1 day, in seconds

var platformBinaryNames = map[string]string{
	"linux-amd64":   "kmscli-linux-amd64",
	"linux-arm64":   "kmscli-linux-arm64",
	"darwin-amd64":  "kmscli-darwin-amd64",
	"darwin-arm64":  "kmscli-darwin-arm64",
	"windows-amd64": "kmscli-windows-amd64.exe",
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

	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}

	err = c.ExecuteKmscli(newArgs)
	if err != nil {
		return err
	}
	return nil
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
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, ".kmscli_version_check")
	c.versionFilePath = filepath.Join(c.configPath, ".kmscli_version")
	c.execFilePath = filepath.Join(c.configPath, "kmscli")
	if runtime.GOOS == "windows" {
		c.execFilePath += ".exe"
	}
	c.installed = fileExists(c.execFilePath)
}

func (c *Context) EnsureInstalledAndUpdated() error {
	if !c.installed {
		c.versionRemote = currentKmscliVersion
		err := c.Install()
		if err != nil {
			return err
		}
		err = c.UpdateCheckCacheTime()
		if err != nil {
			return err
		}
	} else {
		needCheckVersion := c.NeedCheckVersion()
		if needCheckVersion {
			c.versionRemote = currentKmscliVersion
			err := c.GetLocalVersion()
			if err != nil {
				return err
			}
			if c.versionLocal != c.versionRemote {
				err := c.Install()
				if err != nil {
					return err
				}
			}
			err = c.UpdateCheckCacheTime()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Context) Install() error {
	url := fmt.Sprintf("%s/%s/%s", githubDownloadBase, c.versionRemote, c.binaryName)

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, c.binaryName+".tmp")
	if fileExists(tmpFile) {
		err := os.Remove(tmpFile)
		if err != nil {
			return err
		}
	}

	err := downloadBinaryFunc(url, tmpFile)
	if err != nil {
		return fmt.Errorf("failed to download KMS CLI from %s: %v", url, err)
	}

	if fileExists(c.execFilePath) {
		err := os.Remove(c.execFilePath)
		if err != nil {
			return fmt.Errorf("failed to remove existing file %s: %v", c.execFilePath, err)
		}
	}

	err = os.Rename(tmpFile, c.execFilePath)
	if err != nil {
		return fmt.Errorf("failed to move file %s to %s: %v", tmpFile, c.execFilePath, err)
	}

	if runtime.GOOS != "windows" {
		err = os.Chmod(c.execFilePath, 0755)
		if err != nil {
			return fmt.Errorf("failed to set exec permission for file %s: %v", c.execFilePath, err)
		}
	}

	c.versionLocal = c.versionRemote
	c.installed = true

	err = c.SaveLocalVersion()
	if err != nil {
		return fmt.Errorf("failed to save installed version: %v", err)
	}

	return nil
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

	if profile.RegionId != "" {
		envMap["REGION_ID"] = profile.RegionId
	}

	if profile.EndpointType != "" {
		envMap["ENDPOINT_TYPE"] = profile.EndpointType
	}

	envMap["KMSCLI_NAME"] = "aliyun kmscli"

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

func (c *Context) ExecuteKmscli(args []string) error {
	cmd := execCommandFunc(c.execFilePath, args...)
	envs := os.Environ()
	for k, v := range c.envMap {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = envs
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
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
		return fmt.Errorf("KMS CLI not installed")
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

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
