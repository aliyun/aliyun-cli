package flowcli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
)

type Context struct {
	originCtx                 *cli.Context
	configPath                string
	checkVersionCacheFilePath string
	prefixPath                string
	nodePath                  string
	npmPath                   string
	execFilePath              string
	installed                 bool
	versionLocal              string
	versionRemote             string
	osType                    string
	osArch                    string
}

var (
	getConfigurePathFunc = func() string { return config.GetConfigPath() }
	execCommandFunc      = exec.Command
	httpGetFunc          = http.Get
	timeNowFunc          = time.Now
	runtimeGOOSFunc      = func() string { return runtime.GOOS }
	runtimeGOARCHFunc    = func() string { return runtime.GOARCH }
	lookPathFunc         = exec.LookPath
)

const (
	npmPackageName   = "@flow-step/flow-cli"
	binName          = "flow-cli"
	prefixDirName    = "flow-cli-prefix"
	minNodeMajor     = 18
	versionCacheFile = ".flow_cli_version_check"
	downloadBaseURL  = "https://aliyun-cli-pub.oss-cn-hangzhou.aliyuncs.com/cli-ext/flow-cli/downloads"
	defaultRegistry  = "https://registry.npmmirror.com"
)

var VersionCheckTTL = 86400

func NewContext(originContext *cli.Context) *Context {
	return &Context{originCtx: originContext}
}

func (c *Context) Run(args []string) error {
	if err := c.InitBasicInfo(); err != nil {
		return err
	}
	if err := c.EnsureNodeAvailable(); err != nil {
		return err
	}
	if err := c.EnsureNpmAvailable(); err != nil {
		return err
	}
	if err := c.EnsurePrefixAndPackage(); err != nil {
		return err
	}

	c.applyMainCliFlagsFromArgs(args)

	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}
	return c.ExecuteFlowcli(newArgs)
}

func (c *Context) InitBasicInfo() error {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()
	c.configPath = getConfigurePathFunc()
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, versionCacheFile)
	c.prefixPath = filepath.Join(c.configPath, prefixDirName)

	// npm prefix install layout:
	//   unix:    <prefix>/bin/flow-cli
	//   windows: <prefix>/flow-cli.cmd
	if c.osType == "windows" {
		c.execFilePath = filepath.Join(c.prefixPath, binName+".cmd")
	} else {
		c.execFilePath = filepath.Join(c.prefixPath, "bin", binName)
	}

	if envPath := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_FLOW_CLI_EXEC_PATH")); envPath != "" {
		c.execFilePath = envPath
	}
	c.installed = fileExists(c.execFilePath)
	return nil
}

// EnsureNodeAvailable detects system node >= minNodeMajor. Flow-CLI V2 is a
// Node.js/TypeScript CLI; older runtimes fail with cryptic module errors.
func (c *Context) EnsureNodeAvailable() error {
	if envPath := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_FLOW_CLI_NODE_PATH")); envPath != "" {
		c.nodePath = envPath
		return nil
	}
	path, err := lookPathFunc("node")
	if err != nil {
		return fmt.Errorf("node >= %d is required for `aliyun flow-cli`.\n"+
			"  macOS:   brew install node@%d\n"+
			"  Linux:   https://nodejs.org/en/download (LTS)\n"+
			"  Windows: https://nodejs.org/en/download\n"+
			"Or set ALIBABA_CLOUD_FLOW_CLI_NODE_PATH=/path/to/node",
			minNodeMajor, minNodeMajor)
	}
	major, err := getNodeMajorFunc(path)
	if err != nil {
		return fmt.Errorf("failed to detect node version at %s: %v", path, err)
	}
	if major < minNodeMajor {
		return fmt.Errorf("node version too old (got %d.x, need >= %d). Install Node %d LTS from https://nodejs.org/",
			major, minNodeMajor, minNodeMajor)
	}
	c.nodePath = path
	return nil
}

// EnsureNpmAvailable picks an npm matching the chosen node. On some
// distros (debian/ubuntu) nodejs and npm are separate packages, so we
// prefer the npm that ships next to the detected node binary before
// falling back to PATH.
func (c *Context) EnsureNpmAvailable() error {
	if envPath := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_FLOW_CLI_NPM_PATH")); envPath != "" {
		c.npmPath = envPath
		return nil
	}
	// Look for npm next to node first.
	nodeDir := filepath.Dir(c.nodePath)
	candidate := filepath.Join(nodeDir, "npm")
	if c.osType == "windows" {
		candidate = filepath.Join(nodeDir, "npm.cmd")
	}
	if fileExists(candidate) {
		c.npmPath = candidate
		return nil
	}
	path, err := lookPathFunc("npm")
	if err != nil {
		return fmt.Errorf("npm not found in PATH (looked next to %s and via $PATH).\n"+
			"  Debian/Ubuntu: sudo apt install npm\n"+
			"  RHEL/CentOS:   sudo dnf install npm\n"+
			"Or set ALIBABA_CLOUD_FLOW_CLI_NPM_PATH=/path/to/npm",
			c.nodePath)
	}
	c.npmPath = path
	return nil
}

func (c *Context) EnsurePrefixAndPackage() error {
	if os.Getenv("ALIBABA_CLOUD_FLOW_CLI_EXEC_PATH") != "" {
		return nil
	}

	if err := os.MkdirAll(c.prefixPath, 0o755); err != nil {
		return fmt.Errorf("failed to create prefix %s: %v", c.prefixPath, err)
	}

	if !c.installed {
		fmt.Fprintln(c.originCtx.Stderr(), "Installing flow-cli...")
		if err := c.npmInstall(false); err != nil {
			return err
		}
		c.installed = fileExists(c.execFilePath)
		if !c.installed {
			return fmt.Errorf("flow-cli install completed but binary not found at %s", c.execFilePath)
		}
		_ = c.UpdateCheckCacheTime()
		return nil
	}

	if os.Getenv("ALIBABA_CLOUD_FLOW_CLI_NO_UPDATE_CHECK") == "1" {
		return nil
	}
	if c.NeedCheckVersion() {
		// upgrade is best-effort: failure must not block existing install
		_ = c.npmInstall(true)
		_ = c.UpdateCheckCacheTime()
	}
	return nil
}

// npmInstall resolves a target version (OSS version.txt if available,
// otherwise the npm "latest" tag) and then runs `npm install --prefix=...`.
// Official Flow-CLI docs recommend npmmirror; that is the default registry
// unless ALIBABA_CLOUD_FLOW_CLI_NPM_REGISTRY overrides it.
func (c *Context) npmInstall(upgrade bool) error {
	target := npmPackageName
	if v := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_FLOW_CLI_VERSION")); v != "" {
		target = npmPackageName + "@" + v
	} else if v, err := c.fetchOSSVersion(); err == nil && v != "" {
		c.versionRemote = v
		target = npmPackageName + "@" + v
	}

	// -g + --prefix gives global install layout: <prefix>/bin/flow-cli (unix)
	// or <prefix>/flow-cli.cmd (windows). Without -g, npm creates a local
	// install whose bin symlinks land in node_modules/.bin instead.
	args := []string{
		"install", "-g",
		"--prefix", c.prefixPath,
		"--no-fund", "--no-audit",
		"--loglevel=warn",
		"--registry", c.effectiveRegistry(),
		target,
	}

	cmd := execCommandFunc(c.npmPath, args...)
	cmd.Stdout = c.originCtx.Stderr()
	cmd.Stderr = c.originCtx.Stderr()
	if err := cmd.Run(); err != nil {
		if upgrade {
			// upgrade is best-effort — never block on it
			return nil
		}
		return fmt.Errorf("failed to install %s into %s via npm: %v\n"+
			"Hints:\n"+
			"  - Use a custom registry: export ALIBABA_CLOUD_FLOW_CLI_NPM_REGISTRY=...\n"+
			"  - Pin a version:         export ALIBABA_CLOUD_FLOW_CLI_VERSION=x.y.z\n"+
			"  - Use a local binary:    export ALIBABA_CLOUD_FLOW_CLI_EXEC_PATH=$(which flow-cli)",
			target, c.prefixPath, err)
	}
	return nil
}

func (c *Context) effectiveRegistry() string {
	if u := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_FLOW_CLI_NPM_REGISTRY")); u != "" {
		return u
	}
	return defaultRegistry
}

func (c *Context) fetchOSSVersion() (string, error) {
	base := strings.TrimRight(c.effectiveBaseURL(), "/")
	resp, err := httpGetFunc(base + "/version.txt")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("version.txt HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func (c *Context) effectiveBaseURL() string {
	if u := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_FLOW_CLI_DOWNLOAD_BASE_URL")); u != "" {
		return u
	}
	return downloadBaseURL
}

// applyMainCliFlagsFromArgs mirrors esacli: KeepArgs skips flag
// parsing, so we manually fold aliyun config flags from raw args into
// ctx.Flags() to make LoadProfileWithContext see command-line overrides.
func (c *Context) applyMainCliFlagsFromArgs(args []string) {
	if c.originCtx == nil || c.originCtx.Flags() == nil {
		return
	}
	flags := c.originCtx.Flags()
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			continue
		}
		var name, value string
		var hasValue bool
		if idx := strings.Index(a, "="); idx > 0 {
			name = a[2:idx]
			value = a[idx+1:]
			hasValue = true
		} else {
			name = a[2:]
		}
		f := flags.Get(name)
		if f == nil || f.Category != "config" {
			continue
		}
		needsValue := f.AssignedMode != cli.AssignedNone
		if needsValue {
			if !hasValue {
				if i+1 >= len(args) {
					continue
				}
				value = args[i+1]
				i++
			}
			f.SetAssigned(true)
			f.SetValue(value)
		} else {
			f.SetAssigned(true)
		}
	}
}

func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
	longNeedsValue := make(map[string]bool)
	shortNeedsValue := make(map[string]bool)

	mainCliFs := cli.NewFlagSet()
	config.AddFlags(mainCliFs)
	for _, f := range mainCliFs.Flags() {
		if f.Category != "config" {
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

	if c.originCtx != nil && c.originCtx.Flags() != nil && c.originCtx.Flags().Flags() != nil {
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
	}

	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			if idx := strings.Index(a, "="); idx > 0 {
				if _, ok := longNeedsValue[a[:idx]]; ok {
					continue
				}
			}
		}
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

// PrepareEnv builds the env for the flow-cli subprocess. Mirrors the
// esacli approach: profile-derived ALIBABA_CLOUD_* creds (incl.
// RamRoleArn → STS exchange), region, ai-mode UA injection, and a
// COMPAT_MODE marker so flow-cli can adapt help/telemetry if needed.
func (c *Context) PrepareEnv() ([]string, error) {
	profile, err := config.LoadProfileWithContext(c.originCtx)
	if err != nil {
		return nil, fmt.Errorf("config failed: %s", err.Error())
	}
	if c.originCtx != nil && c.originCtx.Flags() != nil {
		profile.OverwriteWithFlags(c.originCtx)
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
			return nil, fmt.Errorf("can't get credential: %s", err)
		}
		model, err := credential.GetCredential()
		if err != nil {
			return nil, fmt.Errorf("can't get credential: %s", err)
		}
		accessKeyId = *model.AccessKeyId
		accessKeySecret = *model.AccessKeySecret
		if model.SecurityToken != nil {
			stsToken = *model.SecurityToken
		}
	}

	envs := os.Environ()
	if accessKeyId != "" {
		envs = append(envs, "ALIBABA_CLOUD_ACCESS_KEY_ID="+accessKeyId)
	}
	if accessKeySecret != "" {
		envs = append(envs, "ALIBABA_CLOUD_ACCESS_KEY_SECRET="+accessKeySecret)
	}
	if stsToken != "" {
		envs = append(envs, "ALIBABA_CLOUD_SECURITY_TOKEN="+stsToken)
	}
	if profile.RegionId != "" {
		envs = append(envs, "ALIBABA_CLOUD_REGION_ID="+profile.RegionId)
	}

	envs = append(envs, "ALIBABA_CLOUD_FLOW_CLI_COMPAT_MODE=aliyun flow-cli")

	if _, ok := os.LookupEnv("ALIBABA_CLOUD_USER_AGENT"); !ok {
		configDir := config.GetConfigDir(c.originCtx)
		if cfg, err := aimode.Load(configDir); err == nil && cfg != nil {
			if ua := strings.TrimSpace(cfg.UserAgent); ua != "" {
				envs = append(envs, "ALIBABA_CLOUD_USER_AGENT="+ua)
			}
		}
	}
	return envs, nil
}

// ExecuteFlowcli invokes the resolved flow-cli binary with the same stdio
// wiring as esacli/computenestutil.
func (c *Context) ExecuteFlowcli(args []string) error {
	cmd := execCommandFunc(c.execFilePath, args...)
	envs, err := c.PrepareEnv()
	if err != nil {
		return err
	}
	cmd.Env = envs
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin
	wd, _ := os.Getwd()
	cmd.Dir = wd

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %s %v: %v", c.execFilePath, args, err)
	}
	return nil
}

// --- version cache (24h TTL) ----------------------------------------------

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
	var lastCheck int64
	if _, err := fmt.Sscanf(string(data), "%d", &lastCheck); err != nil {
		return true
	}
	return timeNowFunc().Unix()-lastCheck > int64(VersionCheckTTL)
}

func (c *Context) UpdateCheckCacheTime() error {
	now := timeNowFunc().Unix()
	return os.WriteFile(c.checkVersionCacheFilePath, []byte(fmt.Sprintf("%d", now)), 0o644)
}

// --- helpers --------------------------------------------------------------

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

var getNodeMajorFunc = getNodeMajor

func getNodeMajor(nodeBin string) (int, error) {
	cmd := execCommandFunc(nodeBin, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return 0, err
	}
	// "v20.18.0\n" → "20"
	s := strings.TrimSpace(out.String())
	s = strings.TrimPrefix(s, "v")
	dot := strings.Index(s, ".")
	if dot < 0 {
		return 0, fmt.Errorf("unexpected node --version output: %q", out.String())
	}
	return strconv.Atoi(s[:dot])
}
