package ots

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
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
	execFilePath              string // tablestore CLI exec file path
	installed                 bool   // whether tablestore CLI is installed
	versionLocal              string
	versionRemote             string
	osType                    string
	osArch                    string
	osSupport                 bool
	downloadPathSuffix        string
	envMap                    map[string]string
	defaultLanguage           string
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

var (
	downloadAndUnzipFunc = DownloadAndUnzip
	execCommandFunc      = exec.Command
	httpGetFunc          = http.Get
	timeNowFunc          = time.Now
	runtimeGOOSFunc      = func() string { return runtime.GOOS }
	runtimeGOARCHFunc    = func() string { return runtime.GOARCH }
)

// 阿里云官方 Tablestore CLI 下载地址配置
// 参考: https://help.aliyun.com/zh/tablestore/developer-reference/download-the-tablestore-cli
const (
	tablestoreCliBaseUrl = "https://help-static-aliyun-doc.aliyuncs.com/file-manage-files/zh-CN/20231225"
	currentVersion       = "2023-10-08-8612e96"
)

var VersionCheckTTL = 86400 // 1 day, in seconds

// 平台对应的下载路径标识
var platformPaths = map[string]string{
	"linux-amd64":   "yhst",
	"linux-arm64":   "jrob",
	"darwin-amd64":  "hgpd",
	"darwin-arm64":  "ahhl",
	"windows-amd64": "phfd",
}

// getDownloadURL 根据平台生成下载地址
func getDownloadURL(platform string) (string, error) {
	pathID, exists := platformPaths[platform]
	if !exists {
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
	// 格式: {baseUrl}/{pathID}/aliyun-tablestore-cli-{platform}-{version}.zip
	return fmt.Sprintf("%s/%s/aliyun-tablestore-cli-%s-%s.zip",
		tablestoreCliBaseUrl, pathID, platform, currentVersion), nil
}

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

	err = c.ExecuteTablestoreCli(newArgs)
	if err != nil {
		return err
	}
	return nil
}

// InitializeAndValidatePlatform 初始化基础信息并验证平台支持
func (c *Context) InitializeAndValidatePlatform() error {
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if !c.osSupport {
		return fmt.Errorf("your os type %s and arch %s is not supported now", c.osType, c.osArch)
	}
	return nil
}

// ExecuteTablestoreCli 执行 Tablestore CLI 命令
func (c *Context) ExecuteTablestoreCli(args []string) error {
	cmd := execCommandFunc(c.execFilePath, args...)
	// set env
	envs := os.Environ()
	for k, v := range c.envMap {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = envs
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin
	// 设置工作目录为 aliyun 配置目录，以便 tablestore CLI 能找到 .tablestore_config 文件
	cmd.Dir = c.configPath

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute %s %v: %v", c.execFilePath, args, err)
	}
	return nil
}

// EnsureInstalledAndUpdated 确保 Tablestore CLI 已安装且为最新版本
// 如果未安装则安装，如果已安装则检查并更新到最新版本
func (c *Context) EnsureInstalledAndUpdated() error {
	if !c.installed {
		// 未安装时，设置版本并安装
		c.versionRemote = currentVersion
		err := c.Install()
		if err != nil {
			return err
		}
		// 更新版本检查时间
		err = c.UpdateCheckCacheTime()
		if err != nil {
			return err
		}
	} else {
		// 已安装时，检查是否需要更新版本
		needCheckVersion := c.NeedCheckVersion()
		if needCheckVersion {
			c.versionRemote = currentVersion
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
			// 更新版本检查时间
			err = c.UpdateCheckCacheTime()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()

	// 构建平台标识符
	platformKey := c.osType + "-" + c.osArch

	// 检查是否支持该平台
	if _, exists := platformPaths[platformKey]; exists {
		c.osSupport = true
		c.downloadPathSuffix = platformKey
	} else {
		c.osSupport = false
	}
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, ".tsutil_version_check")
	c.execFilePath = filepath.Join(c.configPath, "ts")
	if runtime.GOOS == "windows" {
		c.execFilePath += ".exe"
	}
	// check if already installed
	c.installed = fileExists(c.execFilePath)
}

// Install latest tablestore CLI
func (c *Context) Install() error {
	// 生成下载地址
	url, err := getDownloadURL(c.downloadPathSuffix)
	if err != nil {
		return err
	}

	// download then unzip
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "tablestore-cli.zip")
	if fileExists(tmpFile) {
		err := os.Remove(tmpFile)
		if err != nil {
			return err
		}
	}

	err = downloadAndUnzipFunc(url, tmpFile, c.execFilePath, "")
	if err != nil {
		return fmt.Errorf("failed to download and unzip tablestore CLI from %s: %v", url, err)
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
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		_ = out.Close()
		return fmt.Errorf("failed to write to file %s: %v", destFile, err)
	}
	if err = out.Close(); err != nil {
		return fmt.Errorf("failed to close file %s: %v", destFile, err)
	}
	// unzip file
	destDir := filepath.Dir(destFile)
	destDirUse := filepath.Join(destDir, "tablestore_cli_unzip")
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
	// Tablestore CLI 的二进制文件名为 "ts"
	originExtractFile := "ts"
	if runtime.GOOS == "windows" {
		originExtractFile += ".exe"
	}
	// ZIP 包内的文件直接在根目录，没有中间目录
	var sourceFile string
	if extractCenterDir == "" {
		sourceFile = filepath.Join(destDirUse, originExtractFile)
	} else {
		sourceFile = filepath.Join(destDirUse, extractCenterDir, originExtractFile)
	}
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
	// clean up
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

	for _, file := range r.File {
		p, _ := filepath.Abs(file.Name)
		if strings.Contains(p, "..") {
			return fmt.Errorf("invalid file path in zip: %s", file.Name)
		}
		filePath := filepath.Join(dest, file.Name)

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

		rc, err := file.Open()
		if err != nil {
			return err
		}

		outFile, err := os.Create(filePath)
		if err != nil {
			_ = rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		if err != nil {
			_ = rc.Close()
			_ = outFile.Close()
			return err
		}

		err = rc.Close()
		if err != nil {
			_ = outFile.Close()
			return err
		}
		err = outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// EncodeMapBase64 将 map 序列化为 JSON 后做 base64 编码
func EncodeMapBase64(m map[string]any) (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("json marshal failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// tablestoreConfig 表格存储 CLI 的配置结构
type tablestoreConfig struct {
	Endpoint             string `json:"Endpoint"`
	AccessKeyId          string `json:"AccessKeyId"`
	AccessKeySecret      string `json:"AccessKeySecret"`
	AccessKeySecretToken string `json:"AccessKeySecretToken,omitempty"`
	Instance             string `json:"Instance,omitempty"`
}

// PrepareEnv 准备用户身份环境变量并创建配置文件
func (c *Context) PrepareEnv() error {
	// 从 originCtx 获取用户身份信息
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
	case config.RamRoleArn:
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
		if profile.StsToken != "" {
			stsToken = profile.StsToken
		}
	case config.EcsRamRole, config.CredentialsURI, config.OIDC:
		// 这些模式需要先获取临时凭证
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
	default:
		// 其他模式，获取临时凭证
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

	// 创建 tablestore CLI 配置文件
	tsConfig := tablestoreConfig{
		Endpoint:             "", // 将从命令行参数获取
		AccessKeyId:          accessKeyId,
		AccessKeySecret:      accessKeySecret,
		AccessKeySecretToken: stsToken,
		Instance:             "", // 将从命令行参数获取
	}

	// 将配置写入 aliyun 配置目录下的 .tablestore_config 文件
	// tablestore CLI 会从其工作目录读取配置
	configPath := filepath.Join(c.configPath, ".tablestore_config")
	configJSON, err := json.MarshalIndent(tsConfig, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	err = os.WriteFile(configPath, configJSON, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	// 初始化并设置环境变量 map
	if c.envMap == nil {
		c.envMap = make(map[string]string)
	}
	c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"] = accessKeyId
	c.envMap["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] = accessKeySecret
	if stsToken != "" {
		c.envMap["ALIBABA_CLOUD_SECURITY_TOKEN"] = stsToken
	}

	return nil
}

// RemoveFlagsForMainCli 移除主程序使用的 flag，避免传递给 otsutil 出错
func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
	argsNew := make([]string, len(args))
	copy(argsNew, args)
	if c.originCtx.Flags() != nil && c.originCtx.Flags().Flags() != nil {
		for _, f := range c.originCtx.Flags().Flags() {
			if f.IsAssigned() && f.Category == "config" {
				for i, arg := range argsNew {
					if arg == "--"+f.Name || (f.Shorthand != 0 && arg == "-"+string(f.Shorthand)) {
						argsNew = append(argsNew[:i], argsNew[i+1:]...)
						if i < len(argsNew) {
							if f.AssignedMode != cli.AssignedNone {
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
		languageFlagExists := false
		for _, arg := range argsNew {
			if arg == "--language" {
				languageFlagExists = true
				break
			}
		}
		if !languageFlagExists {
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

func (c *Context) GetLocalVersion() error {
	if !c.installed {
		c.versionLocal = ""
		return fmt.Errorf("tablestore CLI not installed")
	}
	c.versionLocal = currentVersion
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
