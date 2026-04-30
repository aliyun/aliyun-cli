package acrskill

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

type Context struct {
	originCtx          *cli.Context
	configPath         string // aliyun config path, all bin and cache file store in the same dir
	execFilePath       string // acr-skill exec file path
	installed          bool   // whether acr-skill is installed
	osType             string
	osArch             string
	osSupport          bool
	downloadPathSuffix string
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

// 可替换的函数变量, 便于单元测试 mock
var (
	downloadBinaryFunc = DownloadBinary
	execCommandFunc    = exec.Command
	httpGetFunc        = http.Get
	runtimeGOOSFunc    = func() string { return runtime.GOOS }
	runtimeGOARCHFunc  = func() string { return runtime.GOARCH }
)

// 阿里云 ACR Skill CLI 下载地址配置
const (
	acrSkillBaseUrl = "https://acr-public-asset.oss-cn-hangzhou.aliyuncs.com/acr-skill/"
)

// 平台对应的下载路径标识（仅提供 linux-amd64 和 darwin-arm64）
var platformPaths = map[string]struct{}{
	"linux-amd64":  {},
	"darwin-arm64": {},
}

// getDownloadURL 根据平台生成下载地址
// 格式: {baseUrl}acr-skill-{os}-{arch}
func getDownloadURL(platform string) (string, error) {
	_, exists := platformPaths[platform]
	if !exists {
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
	return fmt.Sprintf("%sacr-skill-%s", acrSkillBaseUrl, platform), nil
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

	if !c.installed {
		err = c.Install()
		if err != nil {
			return err
		}
	}

	envMap, err := c.PrepareEnv()
	if err != nil {
		return err
	}

	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}

	return c.ExecuteAcrSkill(newArgs, envMap)
}

func (c *Context) InitializeAndValidatePlatform() error {
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if !c.osSupport {
		return fmt.Errorf("your os type %s and arch %s is not supported now", c.osType, c.osArch)
	}
	return nil
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.execFilePath = filepath.Join(c.configPath, "acr-skill")
	if runtime.GOOS == "windows" {
		c.execFilePath += ".exe"
	}
	// check if already installed
	c.installed = fileExists(c.execFilePath)
}

func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()

	platformKey := c.osType + "-" + c.osArch

	if _, exists := platformPaths[platformKey]; exists {
		c.osSupport = true
		c.downloadPathSuffix = platformKey
	} else {
		c.osSupport = false
	}
}

// Install 下载 acr-skill 二进制文件
func (c *Context) Install() error {
	url, err := getDownloadURL(c.downloadPathSuffix)
	if err != nil {
		return err
	}

	err = downloadBinaryFunc(url, c.execFilePath)
	if err != nil {
		return fmt.Errorf("failed to download acr-skill from %s: %v", url, err)
	}
	c.installed = true

	return nil
}

// DownloadBinary 直接下载二进制文件到目标路径
func DownloadBinary(url string, exeFilePath string) error {
	resp, err := httpGetFunc(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: status code %d", url, resp.StatusCode)
	}

	// 下载到临时文件，完成后再移动，避免写入中断导致损坏
	tmpFile := exeFilePath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", tmpFile, err)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		_ = out.Close()
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to write to file %s: %v", tmpFile, err)
	}
	if err = out.Close(); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to close file %s: %v", tmpFile, err)
	}

	// 移除旧的可执行文件
	if fileExists(exeFilePath) {
		err = os.Remove(exeFilePath)
		if err != nil {
			_ = os.Remove(tmpFile)
			return fmt.Errorf("failed to remove existing file %s: %v", exeFilePath, err)
		}
	}

	// 重命名临时文件为目标文件
	err = os.Rename(tmpFile, exeFilePath)
	if err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to rename %s to %s: %v", tmpFile, exeFilePath, err)
	}

	// 设置执行权限
	if runtime.GOOS != "windows" {
		err = os.Chmod(exeFilePath, 0755)
		if err != nil {
			return fmt.Errorf("failed to set exec permission for file %s: %v", exeFilePath, err)
		}
	}

	return nil
}

// PrepareEnv 准备用户身份环境变量, 通过 REGISTRY_USERNAME/REGISTRY_PASSWORD 传递给 acr-skill
func (c *Context) PrepareEnv() (map[string]string, error) {
	profile, err := config.LoadProfileWithContext(c.originCtx)
	if err != nil {
		return nil, fmt.Errorf("config failed: %s", err.Error())
	}

	var accessKeyId, accessKeySecret string

	mode := profile.Mode
	switch mode {
	case config.AK:
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
	case config.StsToken:
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
	case config.RamRoleArn:
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
	default:
		proxyHost, ok := c.originCtx.Flags().GetValue("proxy-host")
		if !ok {
			proxyHost = ""
		}
		credential, err := profile.GetCredential(c.originCtx, tea.String(proxyHost))
		if err != nil {
			return nil, fmt.Errorf("can't get credential %s", err)
		}
		model, err := credential.GetCredential()
		if err != nil {
			return nil, fmt.Errorf("can't get credential %s", err)
		}
		accessKeyId = *model.AccessKeyId
		accessKeySecret = *model.AccessKeySecret
	}

	if accessKeyId == "" || accessKeySecret == "" {
		return nil, fmt.Errorf("access key id or access key secret is empty, please run `aliyun configure` first")
	}

	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// 通过环境变量传递 ACR 注册凭证
	envMap["REGISTRY_USERNAME"] = accessKeyId
	envMap["REGISTRY_PASSWORD"] = accessKeySecret

	return envMap, nil
}

// RemoveFlagsForMainCli 移除主程序使用的 flag，避免传递给 acr-skill 出错
func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
	if c.originCtx.Flags() == nil || c.originCtx.Flags().Flags() == nil {
		return append([]string(nil), args...), nil
	}
	longNeedsValue := make(map[string]bool)  // key: --name
	shortNeedsValue := make(map[string]bool) // key: -x
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

// ExecuteAcrSkill 执行 acr-skill 命令
func (c *Context) ExecuteAcrSkill(args []string, envMap map[string]string) error {
	cmd := execCommandFunc(c.execFilePath, args...)
	envs := make([]string, 0, len(envMap))
	for k, v := range envMap {
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

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
