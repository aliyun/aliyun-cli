package agentbay

import (
	"archive/tar"
	"compress/gzip"
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

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/util"
)

type Context struct {
	originCtx                 *cli.Context
	configPath                string
	checkVersionCacheFilePath string
	execFilePath              string
	installed                 bool
	versionLocal              string
	versionRemote             string
	osType                    string
	osArch                    string
	osSupport                 bool
	downloadFileName          string
}

type versionManifest struct {
	Channels map[string]struct {
		LatestVersion string `json:"latest_version"`
	} `json:"channels"`
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

var (
	getLatestAgentBayVersionFunc = GetLatestAgentBayVersion
	downloadAndInstallFunc       = DownloadAndInstall
	loadAgentBayProfileFunc      = loadAgentBayProfile
	execCommandFunc              = exec.Command
	httpGetFunc                  = http.Get
	httpDoFunc                   = func(req *http.Request) (*http.Response, error) {
		client := &http.Client{
			Timeout: 30 * time.Second,
		}
		return client.Do(req)
	}
	timeNowFunc       = time.Now
	runtimeGOOSFunc   = func() string { return runtime.GOOS }
	runtimeGOARCHFunc = func() string { return runtime.GOARCH }
)

var versionManifestURL = "https://cli-build-packages.oss-cn-beijing.aliyuncs.com/version_manifest.json"
var downloadBaseURL = "https://cli-build-packages.oss-cn-beijing.aliyuncs.com/"

var VersionCheckTTL = 86400

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

	if !c.installed {
		latestVersionRemote, err := getLatestAgentBayVersionFunc()
		if err != nil {
			return err
		}
		c.versionRemote = latestVersionRemote
		if err := c.Install(); err != nil {
			return err
		}
		if err := c.UpdateCheckCacheTime(); err != nil {
			return err
		}
	} else if c.NeedCheckVersion() {
		latestVersionRemote, err := getLatestAgentBayVersionFunc()
		if err != nil {
			return err
		}
		c.versionRemote = latestVersionRemote
		if err := c.GetLocalVersion(); err != nil {
			return err
		}
		if c.versionLocal != c.versionRemote {
			if err := c.Install(); err != nil {
				return err
			}
		}
		if err := c.UpdateCheckCacheTime(); err != nil {
			return err
		}
	}

	childArgs, agentBayOptions := extractAgentBayCLIOptions(args)
	cmd := execCommandFunc(c.execFilePath, childArgs...)
	env, err := c.BuildExecEnv(agentBayOptions)
	if err != nil {
		return err
	}
	cmd.Env = env
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %s %v: %v", c.execFilePath, args, err)
	}
	return nil
}

type agentBayCLIOptions struct {
	endpoint string
	region   string
}

func (c *Context) BuildExecEnv(options agentBayCLIOptions) ([]string, error) {
	profile, err := loadAgentBayProfileFunc(c.originCtx)
	if err != nil {
		return nil, err
	}
	envs, err := agentBayCredentialEnvFromProfile(c.originCtx, profile)
	if err != nil {
		return nil, err
	}
	if endpoint := c.agentBayEndpoint(profile, options); endpoint != "" {
		envs["AGENTBAY_CLI_ENDPOINT"] = endpoint
	}
	return mergeAgentBayEnv(os.Environ(), envs), nil
}

func (c *Context) agentBayEndpoint(profile config.Profile, options agentBayCLIOptions) string {
	if options.endpoint != "" {
		return options.endpoint
	}
	if endpoint := flagStringValue(c.originCtx, config.EndpointFlagName); endpoint != "" {
		return endpoint
	}

	region := options.region
	if region == "" {
		region = flagStringValue(c.originCtx, config.RegionFlagName)
	}
	if region == "" {
		region = flagStringValue(c.originCtx, config.RegionIdFlagName)
	}
	if region == "" {
		region = profile.RegionId
	}
	if region == "" {
		return ""
	}
	return fmt.Sprintf("xiaoying.%s.aliyuncs.com", region)
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

func extractAgentBayCLIOptions(args []string) ([]string, agentBayCLIOptions) {
	filtered := make([]string, 0, len(args))
	options := agentBayCLIOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if key, value, ok := strings.Cut(arg, "="); ok {
			switch key {
			case "--endpoint", "-e":
				options.endpoint = strings.TrimSpace(value)
				continue
			case "--region", "--RegionId":
				options.region = strings.TrimSpace(value)
				continue
			}
		}

		switch arg {
		case "--endpoint", "-e":
			if i+1 < len(args) {
				options.endpoint = strings.TrimSpace(args[i+1])
				i++
			}
			continue
		case "--region", "--RegionId":
			if i+1 < len(args) {
				options.region = strings.TrimSpace(args[i+1])
				i++
			}
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, options
}

func agentBayCredentialEnvFromProfile(ctx *cli.Context, profile config.Profile) (map[string]string, error) {
	switch profile.Mode {
	case config.AK:
		if profile.AccessKeyId == "" || profile.AccessKeySecret == "" {
			return nil, fmt.Errorf("AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first")
		}
		return map[string]string{
			"AGENTBAY_ACCESS_KEY_ID":     profile.AccessKeyId,
			"AGENTBAY_ACCESS_KEY_SECRET": profile.AccessKeySecret,
		}, nil
	case config.StsToken:
		if profile.AccessKeyId == "" || profile.AccessKeySecret == "" || profile.StsToken == "" {
			return nil, fmt.Errorf("AccessKeyId/AccessKeySecret/StsToken is empty! run `aliyun configure` first")
		}
		return map[string]string{
			"AGENTBAY_ACCESS_KEY_ID":            profile.AccessKeyId,
			"AGENTBAY_ACCESS_KEY_SECRET":        profile.AccessKeySecret,
			"AGENTBAY_ACCESS_KEY_SESSION_TOKEN": profile.StsToken,
		}, nil
	}

	cred, err := profile.GetCredential(ctx, nil)
	if err != nil {
		return nil, err
	}
	model, err := cred.GetCredential()
	if err != nil {
		return nil, err
	}

	envs := map[string]string{
		"AGENTBAY_ACCESS_KEY_ID":     *model.AccessKeyId,
		"AGENTBAY_ACCESS_KEY_SECRET": *model.AccessKeySecret,
	}
	if model.SecurityToken != nil {
		envs["AGENTBAY_ACCESS_KEY_SESSION_TOKEN"] = *model.SecurityToken
	}
	return envs, nil
}

func loadAgentBayProfile(ctx *cli.Context) (config.Profile, error) {
	configPath := filepath.Join(config.GetConfigPath(), "config.json")
	if flag := config.ConfigurePathFlag(ctx.Flags()); flag != nil {
		if v, ok := flag.GetValue(); ok && strings.TrimSpace(v) != "" {
			configPath = v
		}
	}

	profileName := ""
	if flag := config.ProfileFlag(ctx.Flags()); flag != nil {
		if v, ok := flag.GetValue(); ok && strings.TrimSpace(v) != "" {
			profileName = v
		}
	}
	if profileName == "" {
		profileName = util.GetFromEnv("ALIBABACLOUD_PROFILE", "ALIBABA_CLOUD_PROFILE", "ALICLOUD_PROFILE")
	}

	profile, err := config.LoadProfile(configPath, profileName)
	if err != nil {
		return profile, err
	}
	if ctx.InConfigureMode() {
		profile.OverwriteWithFlags(ctx)
	}
	return profile, nil
}

func mergeAgentBayEnv(base []string, agentBayEnvs map[string]string) []string {
	skipKeys := map[string]bool{
		"AGENTBAY_ACCESS_KEY_ID":            true,
		"AGENTBAY_ACCESS_KEY_SECRET":        true,
		"AGENTBAY_ACCESS_KEY_SESSION_TOKEN": true,
		"AGENTBAY_CLI_ENDPOINT":             true,
		"ALIBABA_CLOUD_REGION_ID":           true,
		"ALIBABACLOUD_REGION_ID":            true,
		"ALICLOUD_REGION_ID":                true,
		"ALIBABA_CLOUD_USER_AGENT":          true,
		"ALIBABA_CLOUD_CLI_AI_USER_AGENT":   true,
		"ALIYUN_USER_AGENT":                 true,
	}

	result := make([]string, 0, len(base)+len(agentBayEnvs))
	for _, item := range base {
		key, _, ok := strings.Cut(item, "=")
		if !ok || skipKeys[key] {
			continue
		}
		result = append(result, item)
	}
	for key, value := range agentBayEnvs {
		if value == "" {
			continue
		}
		result = append(result, key+"="+value)
	}
	return result
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, ".agentbay_version_check")
	c.execFilePath = filepath.Join(c.configPath, "agentbay")
	c.installed = false
	if runtimeGOOSFunc() == "windows" {
		c.execFilePath += ".exe"
	}
	if fileExists(c.execFilePath) {
		c.installed = true
	}
}

func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()

	switch c.osType {
	case "linux", "darwin":
		if c.osArch == "amd64" || c.osArch == "arm64" {
			c.osSupport = true
		}
	case "windows":
		if c.osArch == "amd64" || c.osArch == "arm64" {
			c.osSupport = true
		}
	}
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

func (c *Context) Install() error {
	fileName := agentBayFileName(c.versionRemote, c.osType, c.osArch)
	if fileName == "" {
		return fmt.Errorf("your os type %s and arch %s is not supported now", c.osType, c.osArch)
	}
	c.downloadFileName = fileName
	url := fmt.Sprintf("%s%s/%s", downloadBaseURL, c.versionRemote, fileName)
	tmpFile := filepath.Join(os.TempDir(), fileName)
	if fileExists(tmpFile) {
		if err := os.Remove(tmpFile); err != nil {
			return err
		}
	}
	if err := downloadAndInstallFunc(url, tmpFile, c.execFilePath); err != nil {
		return fmt.Errorf("failed to download and install agentbay from %s: %v", url, err)
	}
	return nil
}

func agentBayFileName(version, goos, goarch string) string {
	switch goos {
	case "linux", "darwin":
		if goarch == "amd64" || goarch == "arm64" {
			return fmt.Sprintf("agentbay-%s-%s-%s.tar.gz", version, goos, goarch)
		}
	case "windows":
		if goarch == "amd64" || goarch == "arm64" {
			return fmt.Sprintf("agentbay-%s-windows-%s.exe", version, goarch)
		}
	}
	return ""
}

func DownloadAndInstall(url string, destFile string, exeFilePath string) error {
	if err := downloadFile(url, destFile); err != nil {
		return err
	}
	defer func() { _ = os.Remove(destFile) }()

	if runtimeGOOSFunc() == "windows" {
		if fileExists(exeFilePath) {
			if err := os.Remove(exeFilePath); err != nil {
				return fmt.Errorf("failed to remove existing file %s: %v", exeFilePath, err)
			}
		}
		return util.CopyFileAndRemoveSource(destFile, exeFilePath)
	}
	if err := extractAgentBayTarGz(destFile, exeFilePath); err != nil {
		return err
	}
	if err := os.Chmod(exeFilePath, 0755); err != nil {
		return fmt.Errorf("failed to set exec permission for file %s: %v", exeFilePath, err)
	}
	return nil
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

func extractAgentBayTarGz(src string, exeFilePath string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	tmpDir, err := os.MkdirTemp(filepath.Dir(src), "agentbay_unzip_")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.FileInfo().IsDir() {
			continue
		}
		if filepath.Base(header.Name) != "agentbay" {
			continue
		}
		p, _ := filepath.Abs(header.Name)
		if strings.Contains(p, "..") {
			return fmt.Errorf("invalid file path in tar.gz: %s", header.Name)
		}
		sourceFile := filepath.Join(tmpDir, "agentbay")
		out, err := os.Create(sourceFile)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if fileExists(exeFilePath) {
			if err := os.Remove(exeFilePath); err != nil {
				return fmt.Errorf("failed to remove existing file %s: %v", exeFilePath, err)
			}
		}
		return util.CopyFileAndRemoveSource(sourceFile, exeFilePath)
	}
	return fmt.Errorf("extracted file agentbay not exist")
}

func (c *Context) GetLocalVersion() error {
	if !c.installed {
		c.versionLocal = ""
		return fmt.Errorf("agentbay not installed")
	}
	cmd := execCommandFunc(c.execFilePath, "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute %s %s: %v", c.execFilePath, "version", err)
	}
	version, err := parseAgentBayVersion(string(output))
	if err != nil {
		return err
	}
	c.versionLocal = version
	return nil
}

func parseAgentBayVersion(data string) (string, error) {
	lines := strings.Split(strings.TrimSpace(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "", fmt.Errorf("failed to parse version from output")
	}
	var version string
	if _, err := fmt.Sscanf(strings.TrimSpace(lines[0]), "AgentBay CLI version %s", &version); err != nil {
		return "", fmt.Errorf("failed to parse version from output: %v", err)
	}
	return strings.TrimSpace(version), nil
}

func (c *Context) UpdateCheckCacheTime() error {
	currentTime := timeNowFunc().Unix()
	data := fmt.Sprintf("%d", currentTime)
	if err := os.WriteFile(c.checkVersionCacheFilePath, []byte(data), 0644); err != nil {
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

func GetLatestAgentBayVersion() (string, error) {
	req, err := http.NewRequest("GET", versionManifestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %v", versionManifestURL, err)
	}
	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)

	resp, err := httpDoFunc(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content from %s: %v", versionManifestURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP request failed with status code %d from %s", resp.StatusCode, versionManifestURL)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from %s: %v", versionManifestURL, err)
	}
	var manifest versionManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return "", fmt.Errorf("failed to parse version manifest: %v", err)
	}
	stable, ok := manifest.Channels["stable"]
	if !ok || strings.TrimSpace(stable.LatestVersion) == "" {
		return "", fmt.Errorf("failed to parse latest stable version from version manifest")
	}
	return strings.TrimSpace(stable.LatestVersion), nil
}
