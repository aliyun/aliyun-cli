package ossutil

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

type Context struct {
	originCtx                 *cli.Context
	configPath                string // aliyun config path, all bin and cache file store in the same dir
	checkVersionCacheFilePath string // cache file path to store last check version time, unix timestamp
	execFilePath              string // ossutil exec file path
	installed                 bool   // whether ossutil is installed
	versionLocal              string
	versionRemote             string
	isLatest                  bool
	osType                    string
	osArch                    string
	osSupport                 bool
	downloadPathSuffix        string
	envMap                    map[string]string
	defaultLanguage           string // profile中设置的默认语言
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

// 可替换的函数变量, 便于单元测试 mock
var (
	getLatestOssUtilVersionFunc = GetLatestOssUtilVersion
	downloadAndUnzipFunc        = DownloadAndUnzip
	execCommandFunc             = exec.Command
	httpGetFunc                 = http.Get
	timeNowFunc                 = time.Now
	// runtime hooks for test
	runtimeGOOSFunc   = func() string { return runtime.GOOS }
	runtimeGOARCHFunc = func() string { return runtime.GOARCH }
)

// 提供可替换的版本文件 URL，测试中可指向本地 httptest.Server
var latestVersionURL = "https://gosspublic.alicdn.com/ossutil/v2/version.txt"

var VersionCheckTTL = 86400 // 1 day, in seconds

func NewContext(originContext *cli.Context) *Context {
	return &Context{
		originCtx: originContext,
	}
}

func (c *Context) info(a ...interface{}) {
	if c == nil || c.originCtx == nil {
		return
	}
	if len(a) == 0 {
		return
	}
	if format, ok := a[0].(string); ok && strings.Contains(format, "%") {
		_, _ = fmt.Fprintf(c.originCtx.Stdout(), format, a[1:]...)
		return
	}
	_, _ = fmt.Fprintln(c.originCtx.Stdout(), a...)
}

func (c *Context) errorf(format string, a ...interface{}) {
	_, err := fmt.Fprintf(c.originCtx.Stderr(), format, a...)
	if err != nil {
		return
	}
}

func (c *Context) Run(args []string) error {
	// init config path and some basic info
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if !c.osSupport {
		return fmt.Errorf("your os type %s and arch %s is not supported now", c.osType, c.osArch)
	}
	if !c.installed {
		latestVersionRemote, err := getLatestOssUtilVersionFunc()
		if err != nil {
			return err
		}
		c.versionRemote = latestVersionRemote
		err = c.Install()
		if err != nil {
			return err
		}
		// 更新版本检查时间
		err = c.UpdateCheckCacheTime()
		if err != nil {
			return err
		}
	} else {
		needCheckVersion := c.NeedCheckVersion()
		if needCheckVersion {
			latestVersionRemote, err := getLatestOssUtilVersionFunc()
			if err != nil {
				return err
			}
			c.versionRemote = latestVersionRemote
			err = c.GetLocalVersion()
			if err != nil {
				return err
			}
			if c.versionLocal != c.versionRemote {
				c.info(fmt.Sprintf("A new version of ossutil is available: %s (currently installed: %s)", c.versionRemote, c.versionLocal))
				c.info("update automatically...")
				err := c.Install()
				if err != nil {
					return err
				}
			}
			// 更新版本检查时间
			err = c.UpdateCheckCacheTime()
			if err != nil {
				return err
			}
		}
	}
	// 这里不再需要重新 parse，上游已经解析，直接获取凭证即可
	// Prepare user identity credentials
	err := c.PrepareEnv()
	if err != nil {
		return err
	}
	// remove flags for main cli
	// 移除本身使用的 flag
	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}
	// run ossutil with newArgs
	cmd := execCommandFunc(c.execFilePath, newArgs...)
	// set env
	envs := os.Environ()
	for k, v := range c.envMap {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = envs
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	// 关键: 透传 stdin, 以便 `aliyun ossutil cp - oss://...` 能从上游管道读取数据
	cmd.Stdin = os.Stdin
	// exec
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute %s %v: %v", c.execFilePath, newArgs, err)
	}
	return nil
}

func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()
	// support linux/darwin/windows
	if c.osType == "linux" || c.osType == "darwin" || c.osType == "windows" {
		switch c.osType {
		case "linux":
			// support amd64/386/arm64/arm
			if c.osArch == "amd64" || c.osArch == "386" || c.osArch == "arm64" || c.osArch == "arm" {
				c.osSupport = true
				c.downloadPathSuffix = c.osType + "-" + c.osArch + ".zip"
			} else {
				c.osSupport = false
			}
			break
		case "darwin":
			// support amd64/arm64
			if c.osArch == "amd64" || c.osArch == "arm64" {
				c.osSupport = true
				c.downloadPathSuffix = "mac-" + c.osArch + ".zip"
			} else {
				c.osSupport = false
			}
			break
		case "windows":
			// support amd64/386
			if c.osArch == "amd64" || c.osArch == "386" {
				c.osSupport = true
				c.downloadPathSuffix = c.osType + "-" + c.osArch + ".zip"
			} else {
				c.osSupport = false
			}
		}

	} else {
		c.osSupport = false
	}
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, ".ossutil_version_check")
	c.execFilePath = filepath.Join(c.configPath, "ossutil")
	// check file exist
	c.installed = false
	if runtime.GOOS == "windows" {
		c.execFilePath += ".exe"
	}
	if fileExists(c.execFilePath) {
		c.installed = true
	}
}

func (c *Context) NeedCheckVersion() bool {
	if !c.installed {
		// not installed, no need to check version
		return false
	}
	// check cache file exist
	if !fileExists(c.checkVersionCacheFilePath) {
		// cache file not exist, need to check version
		return true
	}
	// read cache file
	data, err := os.ReadFile(c.checkVersionCacheFilePath)
	if err != nil {
		// read cache file error, need to check version
		return true
	}
	// parse cache file content to int64
	var lastCheckTime int64
	_, err = fmt.Sscanf(string(data), "%d", &lastCheckTime)
	if err != nil {
		// parse cache file content error, need to check version
		return true
	}
	// check if need to check version
	currentTime := time.Now().Unix()
	if currentTime-lastCheckTime > int64(VersionCheckTTL) {
		// need to check version
		return true
	}
	// no need to check version
	return false
}

// Install latest ossutil
func (c *Context) Install() error {
	// like https://gosspublic.alicdn.com/ossutil/v2/2.1.2/ossutil-2.1.2-linux-386.zip
	url := fmt.Sprintf("https://gosspublic.alicdn.com/ossutil/v2/%s/ossutil-%s-%s", c.versionRemote, c.versionRemote, c.downloadPathSuffix)
	// download then unzip
	// download to /tmp/ossutil.zip then unzip to c.execFilePath
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "ossutil.zip")
	if fileExists(tmpFile) {
		err := os.Remove(tmpFile)
		if err != nil {
			return err
		}
	}
	// ossutil-2.1.2-mac-arm64
	extractCenterDir := fmt.Sprintf("ossutil-%s-%s", c.versionRemote, strings.TrimSuffix(c.downloadPathSuffix, ".zip"))
	err := downloadAndUnzipFunc(url, tmpFile, c.execFilePath, extractCenterDir)
	if err != nil {
		return fmt.Errorf("failed to download and unzip ossutil from %s: %v", url, err)
	}
	return nil
}

func DownloadAndUnzip(url string, destFile string, exeFilePath string, extractCenterDir string) error {
	// download file
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
	// 不能 defer Close: 后续需要在函数内删除该文件, Windows 下未关闭会导致删除失败
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		_ = out.Close()
		return fmt.Errorf("failed to write to file %s: %v", destFile, err)
	}
	if err = out.Close(); err != nil { // 显式关闭，避免 Windows 下文件句柄占用
		return fmt.Errorf("failed to close file %s: %v", destFile, err)
	}
	// unzip file
	destDir := filepath.Dir(destFile)
	destDirUse := filepath.Join(destDir, "ossutil_unzip")
	if fileExists(destDirUse) {
		err := os.RemoveAll(destDirUse)
		if err != nil {
			return fmt.Errorf("failed to remove existing dir %s: %v", destDirUse, err)
		}
	}
	err = unzip(destFile, destDirUse)
	if err != nil {
		return fmt.Errorf("failed to unzip file %s: %v", destFile, err)
	}
	// move exe file to exeFilePath
	// check is windows
	originExtractFile := "ossutil"
	if runtime.GOOS == "windows" {
		originExtractFile += ".exe"
	}
	// 这里会多出一个类似ossutil-2.1.2-mac-arm64的层级
	sourceFile := filepath.Join(destDirUse, extractCenterDir, originExtractFile)
	if !fileExists(sourceFile) {
		return fmt.Errorf("extracted file %s not exist", sourceFile)
	}
	if fileExists(exeFilePath) {
		err := os.Remove(exeFilePath)
		if err != nil {
			return fmt.Errorf("failed to remove existing file %s: %v", exeFilePath, err)
		}
	}
	err = os.Rename(sourceFile, exeFilePath)
	if err != nil {
		return fmt.Errorf("failed to move file from %s to %s: %v", sourceFile, exeFilePath, err)
	}
	// set exec permission
	if runtime.GOOS != "windows" {
		err = os.Chmod(exeFilePath, 0755)
		if err != nil {
			return fmt.Errorf("failed to set exec permission for file %s: %v", exeFilePath, err)
		}
	}
	// clean up (已关闭文件句柄, Windows 下可删除)
	err = os.Remove(destFile)
	if err != nil {
		return fmt.Errorf("failed to remove temp file %s: %v", destFile, err)
	}
	err = os.RemoveAll(destDirUse)
	if err != nil {
		return fmt.Errorf("failed to remove temp dir %s: %v", destDirUse, err)
	}
	return nil
}

func unzip(src, dest string) error {
	// 打开 zip 文件
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func(r *zip.ReadCloser) {
		err := r.Close()
		if err != nil {
			fmt.Printf("failed to close zip file: %v\n", err)
		}
	}(r)

	// 遍历 zip 中的每个文件/目录
	for _, file := range r.File {
		// 构造目标路径
		filePath := filepath.Join(dest, file.Name)

		// 检查是否是目录
		if file.FileInfo().IsDir() {
			err := os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		// 打开 zip 中的文件
		rc, err := file.Open()
		if err != nil {
			return err
		}

		// 创建目标文件
		outFile, err := os.Create(filePath)
		if err != nil {
			err := rc.Close()
			if err != nil {
				return err
			}
			return err
		}

		// 复��数据
		_, err = io.Copy(outFile, rc)

		// 关闭资源
		err = rc.Close()
		if err != nil {
			return err
		}
		err = outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) GetLocalVersion() error {
	if !c.installed {
		c.versionLocal = ""
		return fmt.Errorf("ossutil not installed")
	}
	cmd := execCommandFunc(c.execFilePath, "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute %s %s: %v", c.execFilePath, "version", err)
	}
	c.versionLocal = strings.TrimSpace(string(output))
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

// PrepareEnv 准备用户身份的环境变量
func (c *Context) PrepareEnv() error {
	// 获取原始的所有的 env
	envs := os.Environ()
	envMap := make(map[string]string)
	for _, env := range envs {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	// 从 originCtx 获取用户身份信息
	profile, err := config.LoadProfileWithContext(c.originCtx)
	if err != nil {
		return fmt.Errorf("config failed: %s", err.Error())
	}

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

	// reset envMap with credential
	if model.AccessKeyId != nil {
		envMap["OSS_ACCESS_KEY_ID"] = *model.AccessKeyId
	}

	if model.AccessKeySecret != nil {
		envMap["OSS_ACCESS_KEY_SECRET"] = *model.AccessKeySecret
	}

	if model.SecurityToken != nil {
		envMap["OSS_SESSION_TOKEN"] = *model.SecurityToken
	}
	// check is region id and language in profile
	// default exist
	envExistRegion := false
	if data, ok := envMap["OSS_REGION"]; ok {
		if data != "" {
			envExistRegion = true
		}
	}
	// only set when not exist
	if profile.RegionId != "" && !envExistRegion {
		envMap["OSS_REGION"] = profile.RegionId
	}
	if profile.Language != "" {
		c.defaultLanguage = profile.Language
	}
	c.envMap = envMap
	return nil
}

// RemoveFlagsForMainCli 移除主程序使用的 flag，避免传递给 ossutil 出错
func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
	// 如果类��是 config 的 flag，如果已经赋值了，从原始的 args 中移除
	argsNew := make([]string, 0, len(args))
	argsNew = args
	if c.originCtx.Flags() != nil && c.originCtx.Flags().Flags() != nil {
		for _, f := range c.originCtx.Flags().Flags() {
			if f.IsAssigned() && f.Category == "config" {
				// 根据其赋值的类别
				// 检查是否在 argsNew 中
				for i, arg := range argsNew {
					if arg == "--"+f.Name || (f.Shorthand != 0 && arg == "-"+string(f.Shorthand)) {
						// found, remove it and its value if any
						// remove flag
						argsNew = append(argsNew[:i], argsNew[i+1:]...)
						// check if next arg is value
						if i < len(argsNew) {
							// 检��当前 flag 的赋值模式，如果不是AssignedNone，则下一个一定是值，忽略掉
							if f.AssignedMode != cli.AssignedNone {
								// next arg is value, remove it
								argsNew = append(argsNew[:i], argsNew[i+1:]...)
							}
						}
						break
					}
				}
			}
		}
	}
	if c.defaultLanguage != "" {
		// check if --language or -L already in argsNew
		languageFlagExists := false
		for _, arg := range argsNew {
			if arg == "--language" {
				languageFlagExists = true
				break
			}
		}
		if !languageFlagExists {
			// append to end
			argsNew = append(argsNew, "--language", c.defaultLanguage)
		}
	}
	return argsNew, nil
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func GetLatestOssUtilVersion() (string, error) {
	url := latestVersionURL
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 创建HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %v", url, err)
	}

	// 设置User-Agent
	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content from %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 检查HTTP状态码，如果不是2xx就报错
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP request failed with status code %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from %s: %v", url, err)
	}

	data := string(body)
	// example format[ossutil version: 2.1.2]
	// parse result
	var version string
	_, err = fmt.Sscanf(data, "ossutil version: %s", &version)
	if err != nil {
		return "", fmt.Errorf("failed to parse version from response body: %v", err)
	}
	return version, nil
}
