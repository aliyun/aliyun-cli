package skill

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

// NewSkillCommand 创建 skill 子命令
func NewSkillCommand() *cli.Command {
	return &cli.Command{
		Name:   "skill",
		Short:  i18n.T("ACR Skill Management", "ACR Skill管理"),
		Usage:  "acrutil skill <command> [args...]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			options := NewSkillContext(ctx)
			return options.Run(args)
		},
		// allow unknown args
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
	}
}

// SkillContext 管理 acr-skill 二进制的下载、安装和执行
type SkillContext struct {
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
	runtimeGOOSFunc    = func() string { return runtime.GOOS }
	runtimeGOARCHFunc  = func() string { return runtime.GOARCH }
)

// 阿里云 ACR Skill CLI 下载地址配置
const (
	acrSkillBaseUrl = "https://acr-public-asset.oss-cn-hangzhou.aliyuncs.com/acr-skill/"
	// 下载文件大小限制：50MB
	maxDownloadSize = 50 * 1024 * 1024
)

// 包级别的 HTTP client，带超时设置
var httpClient = &http.Client{Timeout: 5 * time.Minute}

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

func NewSkillContext(originContext *cli.Context) *SkillContext {
	return &SkillContext{
		originCtx: originContext,
	}
}

func (c *SkillContext) Run(args []string) error {
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

	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}

	return c.ExecuteAcrSkill(newArgs)
}

func (c *SkillContext) InitializeAndValidatePlatform() error {
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if !c.osSupport {
		return fmt.Errorf("your os type %s and arch %s is not supported now", c.osType, c.osArch)
	}
	return nil
}

func (c *SkillContext) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.execFilePath = filepath.Join(c.configPath, "acr-skill")
	// check if already installed
	c.installed = fileExists(c.execFilePath)
}

func (c *SkillContext) CheckOsTypeAndArch() {
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
func (c *SkillContext) Install() error {
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
	// 使用带超时的 HTTP client
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %v", url, err)
	}

	resp, err := httpClient.Do(req)
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

	// 使用 LimitReader 限制下载大小，防止磁盘空间耗尽
	// 多读 1 字节用于检测是否超出限制：若实际写入字节数 > maxDownloadSize，说明响应体超限
	written, err := io.Copy(out, io.LimitReader(resp.Body, maxDownloadSize+1))
	if err != nil {
		_ = out.Close()
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to write to file %s: %v", tmpFile, err)
	}
	if written > maxDownloadSize {
		_ = out.Close()
		_ = os.Remove(tmpFile)
		return fmt.Errorf("download size exceeds limit of %d bytes", maxDownloadSize)
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
	if runtimeGOOSFunc() != "windows" {
		err = os.Chmod(exeFilePath, 0755)
		if err != nil {
			return fmt.Errorf("failed to set exec permission for file %s: %v", exeFilePath, err)
		}
	}

	return nil
}

// stripFlags lists parent-CLI flag names that are only meaningful to the
// main CLI and should be removed before forwarding args to the acr-skill
// subprocess.  These are auth/config/caller flags whose values are
// already passed via environment variables in PrepareEnv, or have no
// meaning in the subprocess context.
//
// All other flags (including those that happen to share a name with the
// parent CLI, such as --region, --output, --version, --endpoint) are
// passed through so the subprocess can handle them with its own semantics.
var stripFlags = map[string]bool{
	// config: auth & credential
	"profile":                        true,
	"mode":                           true,
	"sts-region":                     true,
	"ram-role-name":                  true,
	"ram-role-arn":                   true,
	"role-session-name":              true,
	"external-id":                    true,
	"source-profile":                 true,
	"private-key":                    true,
	"key-pair-name":                  true,
	"expired-seconds":                true,
	"process-command":                true,
	"oidc-provider-arn":              true,
	"oidc-token-file":                true,
	"cloud-sso-sign-in-url":          true,
	"cloud-sso-access-config":        true,
	"cloud-sso-account-id":           true,
	"oauth-site-type":                true,
	"external-account-type":          true,
	"auto-plugin-install":            true,
	"auto-plugin-install-enable-pre": true,

	// config: connection & runtime
	"config-path":        true,
	"read-timeout":       true,
	"connect-timeout":    true,
	"retry-count":        true,
	"skip-secure-verify": true,
	"endpoint-type":      true,
	"RegionId":           true,

	// openapi: caller-side-only
	"secure":         true,
	"insecure":       true,
	"header":         true,
	"pager":          true,
	"accept":         true,
	"waiter":         true,
	"dryrun":         true,
	"quiet":          true,
	"yes":            true,
	"cli-query":      true,
	"roa":            true,
	"method":         true,
	"user-agent":     true,
	"cli-ai-mode":    true,
	"no-cli-ai-mode": true,
}

// shortFlagsToStrip lists short flags that should be stripped.
// This is used to avoid conflicts where a short flag is used by both
// the parent CLI and the child CLI (e.g., -p for --profile vs --password).
// Only include short flags that are unique to the parent CLI.
var shortFlagsToStrip = map[string]bool{
	// Currently empty - we avoid stripping short flags to prevent conflicts
}

// RemoveFlagsForMainCli strips only the parent-CLI-only flags listed in
// stripFlags from args before forwarding to the acr-skill subprocess.
// All other args (including flags that the subprocess handles, such as
// --region, --output, --version, --endpoint, etc.) are preserved.
func (c *SkillContext) RemoveFlagsForMainCli(args []string) ([]string, error) {
	// Build a set of flags to strip by parsing args
	// Only strip flags that are actually present in args AND are in stripFlags
	argsFlagSet := make(map[string]bool) // tracks which flags appear in args
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			// Long flag: --flag or --flag=value
			flagName := a[2:]
			if idx := strings.Index(flagName, "="); idx != -1 {
				flagName = flagName[:idx]
			}
			argsFlagSet[flagName] = true
		} else if strings.HasPrefix(a, "-") && len(a) == 2 {
			// Short flag: -f
			argsFlagSet[a] = true
		}
	}

	// Now build the removal set: only flags that are both in stripFlags AND in args
	allFlags := cli.NewFlagSet()
	config.AddFlags(allFlags)
	openapi.AddFlags(allFlags)

	longNeedsValue := make(map[string]bool)
	shortNeedsValue := make(map[string]bool)
	for _, f := range allFlags.Flags() {
		if !stripFlags[f.Name] {
			continue
		}
		// Check if this flag actually appears in args
		hasLongFlag := argsFlagSet[f.Name]
		// Only strip short flags if they're explicitly in shortFlagsToStrip
		// This avoids conflicts where the same shorthand is used by both CLIs
		hasShortFlag := shortFlagsToStrip[string(f.Shorthand)] && argsFlagSet["-"+string(f.Shorthand)]
		if !hasLongFlag && !hasShortFlag {
			continue
		}
		needsValue := f.AssignedMode != cli.AssignedNone
		if f.Name != "" {
			longNeedsValue["--"+f.Name] = needsValue
		}
		for _, alias := range f.Aliases {
			longNeedsValue["--"+alias] = needsValue
		}
		if shortFlagsToStrip[string(f.Shorthand)] && f.Shorthand != 0 {
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

// ExecuteAcrSkill 执行 acr-skill 命令
// acr-skill-cli 的 -u/-p 仅支持 ACR Registry 固定密码认证，不支持 AK/SK。
// acrutil 封装层不注入凭证，用户需通过以下方式提供 registry 认证：
//  1. 命令行 -u/-p flag
//  2. 环境变量 REGISTRY_USERNAME/REGISTRY_PASSWORD
func (c *SkillContext) ExecuteAcrSkill(args []string) error {
	finalArgs := append([]string(nil), args...)

	cmd := execCommandFunc(c.execFilePath, finalArgs...)
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin

	envMap := make(map[string]string)
	envMap["ALIBABA_CLOUD_ACR_SKILL_COMPAT_MODE"] = "aliyun acrutil skill"
	if len(envMap) > 0 {
		envs := filterEnv(os.Environ(), envMap)
		for k, v := range envMap {
			envs = append(envs, k+"="+v)
		}
		cmd.Env = envs
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %s: %v", c.execFilePath, err)
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
	if err != nil {
		return !errors.Is(err, fs.ErrNotExist)
	}
	return true
}
