package cms2

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
	downloadPathSuffix        string
	envMap                    map[string]string
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

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

var (
	getLatestCms2VersionFunc = GetLatestCms2Version
	downloadFileFunc         = downloadFile
	execCommandFunc          = exec.Command
	httpGetFunc              = http.Get
	httpDoFunc               = func(req *http.Request) (*http.Response, error) {
		client := &http.Client{Timeout: 30 * time.Second}
		return client.Do(req)
	}
	timeNowFunc       = time.Now
	runtimeGOOSFunc   = func() string { return runtime.GOOS }
	runtimeGOARCHFunc = func() string { return runtime.GOARCH }
)

var downloadBaseURL = "https://o11y-addon-hangzhou-public.oss-cn-hangzhou.aliyuncs.com/share/aliyuncms/"

var VersionCheckTTL = 86400

var platformPaths = map[string]struct{}{
	"linux-amd64":   {},
	"linux-arm64":   {},
	"darwin-amd64":  {},
	"darwin-arm64":  {},
	"windows-amd64": {},
	"windows-arm64": {},
}

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

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		if !c.installed {
			return err
		}
		_, _ = fmt.Fprintf(c.originCtx.Stderr(),
			"Warning: failed to check for cms2 updates: %v\n", err)
	}

	if !c.installed {
		return fmt.Errorf("cms2 binary not found at %s, please install manually or set ALIBABA_CLOUD_CMS2_EXEC_PATH", c.execFilePath)
	}

	if err := c.PrepareEnv(); err != nil {
		return err
	}

	childArgs := c.RemoveFlagsForMainCli(args)
	return c.Execute(childArgs)
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, ".cms2_version_check")
	c.versionFilePath = filepath.Join(c.configPath, ".cms2_version")
	c.execFilePath = filepath.Join(c.configPath, "aliyuncms")
	if runtimeGOOSFunc() == "windows" {
		c.execFilePath += ".exe"
	}

	if envPath := os.Getenv("ALIBABA_CLOUD_CMS2_EXEC_PATH"); envPath != "" {
		c.execFilePath = envPath
	}

	c.installed = fileExists(c.execFilePath)
}

func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()

	platformKey := c.osType + "-" + c.osArch
	if _, exists := platformPaths[platformKey]; exists {
		c.osSupport = true
		c.downloadPathSuffix = platformKey
	}
}

func (c *Context) EnsureInstalledAndUpdated() error {
	if os.Getenv("ALIBABA_CLOUD_CMS2_EXEC_PATH") != "" {
		return nil
	}

	if !c.installed {
		latestVersion, err := getLatestCms2VersionFunc()
		if err != nil {
			return fmt.Errorf("cms2 is not installed and auto-download failed: %v", err)
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

	latestVersion, err := getLatestCms2VersionFunc()
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
		return fmt.Errorf("cms2 not installed")
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
	return os.WriteFile(c.versionFilePath, []byte(c.versionLocal), 0644)
}

func (c *Context) UpdateCheckCacheTime() error {
	data := fmt.Sprintf("%d", timeNowFunc().Unix())
	return os.WriteFile(c.checkVersionCacheFilePath, []byte(data), 0644)
}

func (c *Context) Install() error {
	suffix := c.downloadPathSuffix
	if runtimeGOOSFunc() == "windows" {
		suffix += ".exe"
	}
	url := fmt.Sprintf("%s%s/aliyuncms-%s", downloadBaseURL, c.versionRemote, suffix)

	tmpFile := c.execFilePath + ".tmp"
	if err := downloadFileFunc(url, tmpFile); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to download cms2 from %s: %v", url, err)
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
			return fmt.Errorf("failed to install cms2 binary: %v", copyErr)
		}
	}

	c.versionLocal = c.versionRemote
	c.installed = true
	return c.SaveLocalVersion()
}

func (c *Context) PrepareEnv() error {
	profile, err := config.LoadProfileWithContext(c.originCtx)
	if err != nil {
		return fmt.Errorf("config failed: %s", err.Error())
	}

	var accessKeyId, accessKeySecret, stsToken string

	switch profile.Mode {
	case config.AK:
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
	case config.StsToken:
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
		stsToken = profile.StsToken
	default:
		credential, err := profile.GetCredential(c.originCtx, nil)
		if err != nil {
			return fmt.Errorf("can't get credential: %s", err)
		}
		model, err := credential.GetCredential()
		if err != nil {
			return fmt.Errorf("can't get credential: %s", err)
		}
		accessKeyId = *model.AccessKeyId
		accessKeySecret = *model.AccessKeySecret
		if model.SecurityToken != nil {
			stsToken = *model.SecurityToken
		}
	}

	if accessKeyId == "" || accessKeySecret == "" {
		return fmt.Errorf("access key id or access key secret is empty, please run `aliyun configure` first")
	}

	c.envMap = map[string]string{
		"ALIBABA_CLOUD_CMS_ACCESS_KEY_ID":     accessKeyId,
		"ALIBABA_CLOUD_CMS_ACCESS_KEY_SECRET": accessKeySecret,
		"ALIBABA_CLOUD_CMS_COMPAT_MODE":       "aliyun cms2",
	}
	if stsToken != "" {
		c.envMap["ALIBABA_CLOUD_CMS_SECURITY_TOKEN"] = stsToken
	}

	if region := flagStringValue(c.originCtx, "region"); region != "" {
		c.envMap["ALIBABA_CLOUD_CMS_REGION"] = region
	} else if profile.RegionId != "" {
		c.envMap["ALIBABA_CLOUD_CMS_REGION"] = profile.RegionId
	}

	if endpoint := flagStringValue(c.originCtx, "endpoint"); endpoint != "" {
		c.envMap["ALIBABA_CLOUD_CMS_ENDPOINT"] = endpoint
	}

	c.envMap["ALIBABA_CLOUD_CMS_USER_AGENT"] = util.GetAliyunCliUserAgent()

	return nil
}

// stripFlags lists parent-CLI flag names that are only meaningful to the
// main CLI and should be removed before forwarding args to the cms2
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

// RemoveFlagsForMainCli strips only the parent-CLI-only flags listed in
// stripFlags from args before forwarding to the cms2 subprocess.  All
// other args (including flags that the subprocess handles, such as
// --region, --output, --version, --endpoint, --access-key-id, etc.)
// are preserved.
func (c *Context) RemoveFlagsForMainCli(args []string) []string {
	allFlags := cli.NewFlagSet()
	config.AddFlags(allFlags)
	openapi.AddFlags(allFlags)

	longNeedsValue := make(map[string]bool)
	shortNeedsValue := make(map[string]bool)
	for _, f := range allFlags.Flags() {
		if !stripFlags[f.Name] {
			continue
		}
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
	return out
}

func (c *Context) Execute(childArgs []string) error {
	// Drain idle HTTP connections from the default transport so that no
	// lingering sockets are inherited by the child process.  On macOS
	// there is a race window between socket() and fcntl(FD_CLOEXEC)
	// that can cause the child to inherit a half-ready fd, leading to
	// "connect: bad file descriptor" errors.
	http.DefaultClient.CloseIdleConnections()

	cmd := execCommandFunc(c.execFilePath, childArgs...)

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
		return fmt.Errorf("failed to execute %s %v: %v", c.execFilePath, childArgs, err)
	}
	return nil
}

func GetLatestCms2Version() (string, error) {
	url := downloadBaseURL + "latest/version.txt"

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

func downloadFile(url string, destFile string) error {
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

	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		return fmt.Errorf("failed to write to file %s: %v", destFile, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close file %s: %v", destFile, err)
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

func flagStringValue(ctx *cli.Context, name string) string {
	if ctx == nil {
		return ""
	}
	if ctx.Flags() != nil {
		if flag := ctx.Flags().Get(name); flag != nil {
			if v, ok := flag.GetValue(); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	if ctx.UnknownFlags() != nil {
		if flag := ctx.UnknownFlags().Get(name); flag != nil {
			if v, ok := flag.GetValue(); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
