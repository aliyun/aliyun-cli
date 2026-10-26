package bl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/util"
)

const (
	npmPackageName = "bailian-cli"
	binName        = "bl"
	prefixDirName  = "bl-prefix"
	minNodeMajor   = 18
	minNodeMinor   = 17

	execPathEnv = "ALIBABA_CLOUD_BL_EXEC_PATH"
	nodePathEnv = "ALIBABA_CLOUD_BL_NODE_PATH"
	npmPathEnv  = "ALIBABA_CLOUD_BL_NPM_PATH"
	versionEnv  = "ALIBABA_CLOUD_BL_VERSION"
	registryEnv = "ALIBABA_CLOUD_BL_NPM_REGISTRY"
)

var hostProfileFlags = []string{
	config.ProfileFlagName,
	config.ConfigurePathFlagName,
	config.StsTokenFlagName,
	config.StsRegionFlagName,
	config.StsEndpointFlagName,
	config.RamRoleNameFlagName,
	config.RamRoleArnFlagName,
	config.RoleSessionNameFlagName,
	config.ExternalIdFlagName,
	config.SourceProfileFlagName,
	config.PrivateKeyFlagName,
	config.KeyPairNameFlagName,
	config.ProcessCommandFlagName,
	config.OIDCProviderARNFlagName,
	config.OIDCTokenFileFlagName,
	config.SSOSessionFlagName,
	config.CloudSSOSignInUrlFlagName,
	config.CloudSSOAccessConfigFlagName,
	config.CloudSSOAccountIdFlagName,
	config.ExternalAccountTypeFlagName,
	config.BearerTokenFlagName,
	config.BearerTokenHeaderKeyFlagName,
}

type Context struct {
	originCtx    *cli.Context
	prefixPath   string
	nodePath     string
	npmPath      string
	execFilePath string
	installed    bool

	configPath  func() string
	goos        func() string
	lookPath    func(string) (string, error)
	nodeVersion func(string) (int, int, int, error)
	command     func(string, ...string) *exec.Cmd
	environ     func() []string
	getwd       func() (string, error)
	loadProfile func(*cli.Context) (config.Profile, error)
}

type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("subprocess exited with code %d", e.Code)
}

func (e *ExitError) ExitCode() int {
	return e.Code
}

func NewContext(originCtx *cli.Context) *Context {
	return &Context{
		originCtx:   originCtx,
		configPath:  config.GetConfigPath,
		goos:        func() string { return runtime.GOOS },
		lookPath:    exec.LookPath,
		nodeVersion: getNodeVersion,
		command:     exec.Command,
		environ:     os.Environ,
		getwd:       os.Getwd,
		loadProfile: loadBlProfile,
	}
}

func (c *Context) Run(args []string) error {
	forwarded, err := c.PrepareArgs(args)
	if err != nil {
		return err
	}
	if err := c.InitBasicInfo(); err != nil {
		return err
	}
	if err := c.EnsureNodeAvailable(); err != nil {
		return err
	}
	if c.usingExecPathOverride() {
		if err := c.validateExecPathOverride(); err != nil {
			return err
		}
	} else {
		if err := c.EnsureNpmAvailable(); err != nil {
			return err
		}
		if err := c.EnsurePrefixAndPackage(); err != nil {
			return err
		}
	}
	return c.Execute(forwarded)
}

func (c *Context) PrepareArgs(args []string) ([]string, error) {
	longFlags, shortFlags := hostFlagDefinitions()
	result := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return append(result, args[i:]...), nil
		}

		if strings.HasPrefix(arg, "--") {
			nameValue := strings.TrimPrefix(arg, "--")
			name, value, hasValue := strings.Cut(nameValue, "=")
			if flag, ok := longFlags[name]; ok {
				if !hasValue {
					if i+1 >= len(args) || args[i+1] == "--" {
						return nil, fmt.Errorf("--%s requires a value", name)
					}
					i++
					value = args[i]
				}
				if value == "" {
					return nil, fmt.Errorf("--%s requires a value", name)
				}
				if err := c.assignFlag(flag.Name, value); err != nil {
					return nil, err
				}
				continue
			}
		}

		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 1 {
			if flag, ok := shortFlags[rune(arg[1])]; ok {
				value := strings.TrimPrefix(arg[2:], "=")
				if value == "" {
					if i+1 >= len(args) || args[i+1] == "--" {
						return nil, fmt.Errorf("-%c requires a value", arg[1])
					}
					i++
					value = args[i]
				}
				if err := c.assignFlag(flag.Name, value); err != nil {
					return nil, err
				}
				continue
			}
		}

		result = append(result, arg)
	}
	return result, nil
}

func hostFlagDefinitions() (map[string]*cli.Flag, map[rune]*cli.Flag) {
	flags := cli.NewFlagSet()
	config.AddFlags(flags)
	longFlags := make(map[string]*cli.Flag)
	shortFlags := make(map[rune]*cli.Flag)
	for _, name := range hostProfileFlags {
		flag := flags.Get(name)
		longFlags[name] = flag
		if flag.Shorthand != 0 {
			shortFlags[flag.Shorthand] = flag
		}
	}
	return longFlags, shortFlags
}

func (c *Context) assignFlag(name, value string) error {
	if c.originCtx == nil || c.originCtx.Flags() == nil {
		return fmt.Errorf("host flag --%s is not registered", name)
	}
	flag := c.originCtx.Flags().Get(name)
	if flag == nil {
		return fmt.Errorf("host flag --%s is not registered", name)
	}
	flag.SetAssigned(true)
	flag.SetValue(value)
	return nil
}

func (c *Context) InitBasicInfo() error {
	c.prefixPath = filepath.Join(c.configPath(), prefixDirName)
	if c.goos() == "windows" {
		c.execFilePath = filepath.Join(c.prefixPath, binName+".cmd")
	} else {
		c.execFilePath = filepath.Join(c.prefixPath, "bin", binName)
	}
	if path := strings.TrimSpace(os.Getenv(execPathEnv)); path != "" {
		c.execFilePath = path
	}
	info, err := os.Stat(c.execFilePath)
	c.installed = err == nil && info.Mode().IsRegular()
	return nil
}

func (c *Context) EnsureNodeAvailable() error {
	path := strings.TrimSpace(os.Getenv(nodePathEnv))
	if path == "" {
		var err error
		path, err = c.lookPath("node")
		if err != nil {
			return fmt.Errorf("node >= %d.%d is required for `aliyun bl`; install Node.js LTS or set %s", minNodeMajor, minNodeMinor, nodePathEnv)
		}
	}
	major, minor, patch, err := c.nodeVersion(path)
	if err != nil {
		return fmt.Errorf("failed to detect node version at %s: %v", path, err)
	}
	if major < minNodeMajor || major == minNodeMajor && minor < minNodeMinor {
		return fmt.Errorf("node version too old (got %d.%d.%d, need >= %d.%d.0)", major, minor, patch, minNodeMajor, minNodeMinor)
	}
	c.nodePath = path
	return nil
}

func (c *Context) EnsureNpmAvailable() error {
	if path := strings.TrimSpace(os.Getenv(npmPathEnv)); path != "" {
		c.npmPath = path
		return nil
	}
	candidate := filepath.Join(filepath.Dir(c.nodePath), "npm")
	if c.goos() == "windows" {
		candidate += ".cmd"
	}
	if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
		c.npmPath = candidate
		return nil
	}
	path, err := c.lookPath("npm")
	if err != nil {
		return fmt.Errorf("npm not found; install npm or set %s", npmPathEnv)
	}
	c.npmPath = path
	return nil
}

func (c *Context) EnsurePrefixAndPackage() error {
	if err := os.MkdirAll(c.prefixPath, 0o755); err != nil {
		return fmt.Errorf("failed to create prefix %s: %v", c.prefixPath, err)
	}
	if c.installed {
		return nil
	}
	fmt.Fprintln(c.originCtx.Stderr(), "Installing bailian-cli...")
	if err := c.npmInstall(); err != nil {
		return err
	}
	info, err := os.Stat(c.execFilePath)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("bailian-cli install completed but binary not found at %s", c.execFilePath)
	}
	c.installed = true
	return nil
}

func (c *Context) npmInstall() error {
	target := npmPackageName
	if version := strings.TrimSpace(os.Getenv(versionEnv)); version != "" {
		target += "@" + version
	}
	args := []string{"install", "-g", "--prefix", c.prefixPath, "--no-fund", "--no-audit", "--loglevel=warn"}
	if registry := strings.TrimSpace(os.Getenv(registryEnv)); registry != "" {
		args = append(args, "--registry", registry)
	}
	args = append(args, target)
	cmd := c.command(c.npmPath, args...)
	cmd.Stdout = c.originCtx.Stderr()
	cmd.Stderr = c.originCtx.Stderr()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s into %s via npm: %v", target, c.prefixPath, err)
	}
	return nil
}

func (c *Context) BuildExecEnv() ([]string, error) {
	profile, err := c.loadProfile(c.originCtx)
	if err != nil {
		return nil, err
	}
	base := c.buildRuntimeEnv(c.environ())
	if profile.Mode == "" && profile.AccessKeyId == "" && profile.AccessKeySecret == "" && profile.StsToken == "" {
		return base, nil
	}

	var accessKeyID, accessKeySecret, securityToken string
	switch profile.Mode {
	case config.AK:
		if profile.AccessKeyId == "" || profile.AccessKeySecret == "" {
			return nil, fmt.Errorf("access key ID and secret must be configured together")
		}
		accessKeyID = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
	case config.StsToken:
		if profile.AccessKeyId == "" || profile.AccessKeySecret == "" {
			return nil, fmt.Errorf("access key ID and secret must be configured together")
		}
		if profile.StsToken == "" {
			return nil, fmt.Errorf("security token must be configured for StsToken mode")
		}
		accessKeyID = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
		securityToken = profile.StsToken
	case config.Anonymous:
	default:
		credential, err := profile.GetCredential(c.originCtx, nil)
		if err != nil {
			return nil, fmt.Errorf("can't get credential: %v", err)
		}
		model, err := credential.GetCredential()
		if err != nil {
			return nil, fmt.Errorf("can't get credential: %v", err)
		}
		if model.AccessKeyId == nil || model.AccessKeySecret == nil {
			return nil, fmt.Errorf("credential provider returned incomplete access key credentials")
		}
		accessKeyID = *model.AccessKeyId
		accessKeySecret = *model.AccessKeySecret
		if model.SecurityToken != nil {
			securityToken = *model.SecurityToken
		}
	}
	return mergeCredentialEnv(base, accessKeyID, accessKeySecret, securityToken), nil
}

func (c *Context) buildRuntimeEnv(base []string) []string {
	windows := c.goos() == "windows"
	noUpdate := false
	for _, entry := range base {
		key, value, _ := strings.Cut(entry, "=")
		if windows {
			key = strings.ToUpper(key)
		}
		if key == "ALIBABA_CLOUD_BL_NO_UPDATE_CHECK" {
			noUpdate = value == "1"
		}
	}
	result := make([]string, 0, len(base)+5)
	pathValue := ""
	for _, entry := range base {
		key, value, _ := strings.Cut(entry, "=")
		if windows {
			key = strings.ToUpper(key)
		}
		switch {
		case strings.EqualFold(key, "npm_config_prefix"):
			continue
		case key == "PATH":
			pathValue = value
			continue
		case key == "BAILIAN_INSTALL_METHOD" || key == "ALIBABA_CLOUD_BL_COMPAT_MODE":
			continue
		case noUpdate && key == "NO_UPDATE_NOTIFIER":
			continue
		}
		result = append(result, entry)
	}
	binDir := filepath.Join(c.prefixPath, "bin")
	separator := string(os.PathListSeparator)
	if windows {
		binDir = c.prefixPath
		separator = ";"
	}
	if pathValue != "" {
		binDir += separator + pathValue
	}
	result = append(result,
		"npm_config_prefix="+c.prefixPath,
		"PATH="+binDir,
		"BAILIAN_INSTALL_METHOD=npm",
		"ALIBABA_CLOUD_BL_COMPAT_MODE=aliyun bl",
	)
	if noUpdate {
		result = append(result, "NO_UPDATE_NOTIFIER=1")
	}
	return result
}

func mergeCredentialEnv(base []string, accessKeyID, accessKeySecret, securityToken string) []string {
	replaced := map[string]bool{
		"ALIBABA_CLOUD_ACCESS_KEY_ID":     true,
		"ALIBABA_CLOUD_ACCESS_KEY_SECRET": true,
		"ALIBABA_CLOUD_SECURITY_TOKEN":    true,
	}
	result := make([]string, 0, len(base)+3)
	for _, item := range base {
		key, _, ok := strings.Cut(item, "=")
		if ok && replaced[key] {
			continue
		}
		result = append(result, item)
	}
	if accessKeyID != "" {
		result = append(result, "ALIBABA_CLOUD_ACCESS_KEY_ID="+accessKeyID)
	}
	if accessKeySecret != "" {
		result = append(result, "ALIBABA_CLOUD_ACCESS_KEY_SECRET="+accessKeySecret)
	}
	if securityToken != "" {
		result = append(result, "ALIBABA_CLOUD_SECURITY_TOKEN="+securityToken)
	}
	return result
}

func (c *Context) Execute(args []string) error {
	env, err := c.BuildExecEnv()
	if err != nil {
		return err
	}
	wd, err := c.getwd()
	if err != nil {
		return fmt.Errorf("failed to determine current working directory: %v", err)
	}
	cmd := c.command(c.execFilePath, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Dir = wd
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ExitError{Code: exitErr.ExitCode()}
		}
		return fmt.Errorf("failed to execute %s: %v", c.execFilePath, err)
	}
	return nil
}

func (c *Context) usingExecPathOverride() bool {
	return strings.TrimSpace(os.Getenv(execPathEnv)) != ""
}

func (c *Context) validateExecPathOverride() error {
	info, err := os.Stat(c.execFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s=%q does not point to an existing file", execPathEnv, c.execFilePath)
		}
		return fmt.Errorf("%s=%q: %v", execPathEnv, c.execFilePath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s=%q is not a regular file", execPathEnv, c.execFilePath)
	}
	if c.goos() != "windows" && info.Mode()&0o111 == 0 {
		return fmt.Errorf("%s=%q is not executable", execPathEnv, c.execFilePath)
	}
	return nil
}

func getNodeVersion(nodeBin string) (int, int, int, error) {
	cmd := exec.Command(nodeBin, "--version")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return 0, 0, 0, err
	}
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(output.String()), "v"), ".")
	if len(parts) < 2 {
		return 0, 0, 0, fmt.Errorf("unexpected node --version output: %q", output.String())
	}
	values := [3]int{}
	for i := 0; i < len(values) && i < len(parts); i++ {
		value, err := strconv.Atoi(parts[i])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("unexpected node --version output: %q", output.String())
		}
		values[i] = value
	}
	return values[0], values[1], values[2], nil
}

func loadBlProfile(ctx *cli.Context) (config.Profile, error) {
	var profile config.Profile
	if util.GetFromEnv("ALIBABA_CLOUD_IGNORE_PROFILE", "ALIBABACLOUD_IGNORE_PROFILE") != "TRUE" {
		configPath := filepath.Join(config.GetConfigPath(), "config.json")
		if value, ok := assignedFlagValue(ctx, config.ConfigurePathFlagName); ok && strings.TrimSpace(value) != "" {
			configPath = value
		}
		profileName := ""
		if value, ok := assignedFlagValue(ctx, config.ProfileFlagName); ok {
			profileName = strings.TrimSpace(value)
		}
		if profileName == "" {
			profileName = util.GetFromEnv("ALIBABACLOUD_PROFILE", "ALIBABA_CLOUD_PROFILE", "ALICLOUD_PROFILE")
		}
		var err error
		profile, err = config.LoadProfile(configPath, profileName)
		if err != nil {
			return profile, err
		}
	}
	applyExplicitProfileFlags(ctx, &profile)
	config.AutoModeRecognition(&profile)
	return profile, nil
}

func applyExplicitProfileFlags(ctx *cli.Context, profile *config.Profile) {
	setString := func(name string, target *string) {
		if value, ok := assignedFlagValue(ctx, name); ok {
			*target = value
		}
	}
	if value, ok := assignedFlagValue(ctx, config.ModeFlagName); ok && strings.TrimSpace(value) != "" {
		profile.Mode = config.NormalizeMode(value)
	}
	setString(config.AccessKeyIdFlagName, &profile.AccessKeyId)
	setString(config.AccessKeySecretFlagName, &profile.AccessKeySecret)
	setString(config.StsTokenFlagName, &profile.StsToken)
	setString(config.StsRegionFlagName, &profile.StsRegion)
	setString(config.StsEndpointFlagName, &profile.StsEndpoint)
	setString(config.RamRoleNameFlagName, &profile.RamRoleName)
	setString(config.RamRoleArnFlagName, &profile.RamRoleArn)
	setString(config.RoleSessionNameFlagName, &profile.RoleSessionName)
	setString(config.ExternalIdFlagName, &profile.ExternalId)
	setString(config.SourceProfileFlagName, &profile.SourceProfile)
	setString(config.PrivateKeyFlagName, &profile.PrivateKey)
	setString(config.KeyPairNameFlagName, &profile.KeyPairName)
	setString(config.ProcessCommandFlagName, &profile.ProcessCommand)
	setString(config.OIDCProviderARNFlagName, &profile.OIDCProviderARN)
	setString(config.OIDCTokenFileFlagName, &profile.OIDCTokenFile)
	setString(config.SSOSessionFlagName, &profile.SSOSession)
	setString(config.CloudSSOSignInUrlFlagName, &profile.CloudSSOSignInUrl)
	setString(config.CloudSSOAccessConfigFlagName, &profile.CloudSSOAccessConfig)
	setString(config.CloudSSOAccountIdFlagName, &profile.CloudSSOAccountId)
	setString(config.ExternalAccountTypeFlagName, &profile.ExternalAccountType)
	setString(config.BearerTokenFlagName, &profile.BearerTokenValue)
	setString(config.BearerTokenHeaderKeyFlagName, &profile.BearerTokenHeaderKey)
}

func assignedFlagValue(ctx *cli.Context, name string) (string, bool) {
	if ctx == nil || ctx.Flags() == nil {
		return "", false
	}
	flag := ctx.Flags().Get(name)
	if flag == nil || !flag.IsAssigned() {
		return "", false
	}
	return flag.GetValue()
}
