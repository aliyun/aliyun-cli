package dgate

import (
	"crypto/sha256"
	"encoding/hex"
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
	"golang.org/x/mod/semver"
)

const (
	defaultDownloadBase = "https://dg-agent-cn-hangzhou.oss-cn-hangzhou.aliyuncs.com/dgate/prd"
	compatModeEnv       = "ALIBABA_CLOUD_DGATE_COMPAT_MODE"
	compatModeValue     = "aliyun dgate"
	execPathEnv         = "ALIBABA_CLOUD_DGATE_EXEC_PATH"
	downloadBaseEnv     = "ALIBABA_CLOUD_DGATE_DOWNLOAD_BASE_URL"
	maxDownloadSize     = 200 * 1024 * 1024
)

// VersionCheckTTL 是已安装二进制两次远程版本检查的间隔。
var VersionCheckTTL = 24 * time.Hour

// passthroughParentFlags 是 dgate 自己要处理的父 CLI flag，不能剥掉。
// 其余 config / openapi flag（profile、mode、AK、pager 等）都属于 aliyun-cli。
var passthroughParentFlags = map[string]bool{
	"region":  true,
	"output":  true,
	"quiet":   true,
	"version": true,
	"yes":     true,
	"help":    true,
	"debug":   true,
}

// ExitError 把子进程退出码交回 aliyun-cli，避免被收成一条 ERROR 并且码变成 1。
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("subprocess exited with code %d", e.Code)
}

func (e *ExitError) ExitCode() int { return e.Code }

type Context struct {
	originCtx                 *cli.Context
	configPath                string
	execFilePath              string
	versionFilePath           string
	checkVersionCacheFilePath string
	versionLocal              string
	versionRemote             string
	installed                 bool
	useExternalBinary         bool
	osType                    string
	osArch                    string
	osSupport                 bool
	envMap                    map[string]string
}

var (
	getConfigurePathFunc = func() string { return config.GetConfigPath() }
	downloadBinaryFunc   = downloadBinary
	execCommandFunc      = exec.Command
	timeNowFunc          = time.Now
	runtimeGOOSFunc      = func() string { return runtime.GOOS }
	runtimeGOARCHFunc    = func() string { return runtime.GOARCH }
	getRemoteVersionFunc = getRemoteVersion
	httpDoFunc           = func(req *http.Request) (*http.Response, error) {
		client := &http.Client{Timeout: 30 * time.Second}
		return client.Do(req)
	}
	httpClient = &http.Client{Timeout: 5 * time.Minute}
)

func NewContext(origin *cli.Context) *Context {
	return &Context{originCtx: origin}
}

func (c *Context) Run(args []string) error {
	if err := c.initialize(); err != nil {
		return err
	}
	if err := c.ensureInstalled(); err != nil {
		return err
	}
	c.prepareEnv()
	forwarded := removeParentFlags(args)
	return c.execute(forwarded)
}

func (c *Context) initialize() error {
	c.configPath = getConfigurePathFunc()
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, ".dgate_version_check")
	c.versionFilePath = filepath.Join(c.configPath, ".dgate_version")
	c.execFilePath = filepath.Join(c.configPath, "dgate")
	if runtimeGOOSFunc() == "windows" {
		c.execFilePath += ".exe"
	}
	if envPath := strings.TrimSpace(os.Getenv(execPathEnv)); envPath != "" {
		c.execFilePath = envPath
		c.useExternalBinary = true
	}
	c.installed = fileExists(c.execFilePath)
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()
	switch c.osType + "/" + c.osArch {
	case "linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64":
		c.osSupport = true
	default:
		c.osSupport = false
	}
	if !c.osSupport && !c.useExternalBinary {
		return fmt.Errorf("your os type %s and arch %s is not supported now (supported: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64)", c.osType, c.osArch)
	}
	return nil
}

func (c *Context) baseURL() string {
	if u := strings.TrimSpace(os.Getenv(downloadBaseEnv)); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultDownloadBase
}

func (c *Context) artifactName() string {
	name := fmt.Sprintf("dgate_%s_%s", c.osType, c.osArch)
	if c.osType == "windows" {
		name += ".exe"
	}
	return name
}

func (c *Context) ensureInstalled() error {
	if c.useExternalBinary {
		if !c.installed {
			return fmt.Errorf("dgate binary not found at %s=%s", execPathEnv, c.execFilePath)
		}
		return nil
	}
	if !c.installed {
		version, err := getRemoteVersionFunc(c.baseURL() + "/latest.txt")
		if err != nil {
			return fmt.Errorf("dgate is not installed and auto-download failed: %v\ntry: curl -fsSL %s/latest.txt", err, c.baseURL())
		}
		c.versionRemote = version
		if err := c.install(); err != nil {
			return err
		}
		_ = c.writeCheckTime()
		return nil
	}
	if !c.needCheckVersion() {
		return nil
	}
	version, err := getRemoteVersionFunc(c.baseURL() + "/latest.txt")
	if err != nil {
		c.warn("failed to check for dgate updates: %v", err)
		_ = c.writeCheckTime()
		return nil
	}
	c.versionRemote = version
	if err := c.readLocalVersion(); err != nil {
		c.versionLocal = ""
	}
	if c.versionLocal != c.versionRemote {
		if err := c.install(); err != nil {
			c.warn("failed to update dgate to %s: %v", c.versionRemote, err)
		}
	}
	_ = c.writeCheckTime()
	return nil
}

func (c *Context) install() error {
	if !validReleaseVersion(c.versionRemote) {
		return fmt.Errorf("refusing to download dgate: invalid version %q", c.versionRemote)
	}
	url := c.baseURL() + "/" + c.versionRemote + "/" + c.artifactName()
	c.warn("Installing dgate %s from %s ...", c.versionRemote, url)
	tmpFile := c.execFilePath + ".tmp"
	if err := downloadBinaryFunc(url, tmpFile); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to download dgate from %s: %v", url, err)
	}
	if err := verifySHA256(url+".sha256", tmpFile); err != nil {
		_ = os.Remove(tmpFile)
		return err
	}
	if runtimeGOOSFunc() != "windows" {
		if err := os.Chmod(tmpFile, 0755); err != nil {
			_ = os.Remove(tmpFile)
			return fmt.Errorf("failed to set exec permission: %v", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(c.execFilePath), 0755); err != nil {
		_ = os.Remove(tmpFile)
		return err
	}
	if runtimeGOOSFunc() == "windows" && fileExists(c.execFilePath) {
		_ = os.Remove(c.execFilePath)
	}
	if err := os.Rename(tmpFile, c.execFilePath); err != nil {
		if copyErr := util.CopyFileAndRemoveSource(tmpFile, c.execFilePath); copyErr != nil {
			return fmt.Errorf("failed to install dgate binary: %v", copyErr)
		}
	}
	c.versionLocal = c.versionRemote
	c.installed = true
	return os.WriteFile(c.versionFilePath, []byte(c.versionLocal), 0644)
}

func (c *Context) prepareEnv() {
	// 类型 C：不注入阿里云 AK/SK。登录由 dgate auth login / config set-token 自己完成。
	c.envMap = map[string]string{
		compatModeEnv: compatModeValue,
	}
}

func (c *Context) execute(args []string) error {
	http.DefaultClient.CloseIdleConnections()
	cmd := execCommandFunc(c.execFilePath, args...)
	envs := filterEnv(os.Environ(), c.envMap)
	for k, v := range c.envMap {
		envs = append(envs, k+"="+v)
	}
	cmd.Env = envs
	if c.originCtx != nil {
		cmd.Stdout = c.originCtx.Stdout()
		cmd.Stderr = c.originCtx.Stderr()
	}
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ExitError{Code: exitErr.ExitCode()}
		}
		return fmt.Errorf("failed to execute %s: %v", c.execFilePath, err)
	}
	return nil
}

func (c *Context) needCheckVersion() bool {
	if !fileExists(c.checkVersionCacheFilePath) {
		return true
	}
	data, err := os.ReadFile(c.checkVersionCacheFilePath)
	if err != nil {
		return true
	}
	var last int64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &last); err != nil {
		return true
	}
	return timeNowFunc().Unix()-last > int64(VersionCheckTTL.Seconds())
}

func (c *Context) readLocalVersion() error {
	data, err := os.ReadFile(c.versionFilePath)
	if err != nil {
		return err
	}
	c.versionLocal = strings.TrimSpace(string(data))
	if c.versionLocal == "" {
		return fmt.Errorf("empty version file")
	}
	return nil
}

func (c *Context) writeCheckTime() error {
	return os.WriteFile(c.checkVersionCacheFilePath, []byte(fmt.Sprintf("%d", timeNowFunc().Unix())), 0644)
}

func (c *Context) warn(format string, args ...interface{}) {
	if c.originCtx == nil {
		return
	}
	fmt.Fprintf(c.originCtx.Stderr(), format+"\n", args...)
}

func getRemoteVersion(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)
	resp, err := httpDoFunc(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(body))
	if !validReleaseVersion(version) {
		return "", fmt.Errorf("invalid version %q from %s", version, url)
	}
	return version, nil
}

func validReleaseVersion(v string) bool {
	if v == "" || strings.ContainsAny(v, "/\\ \t") {
		return false
	}
	check := v
	if !strings.HasPrefix(check, "v") {
		check = "v" + check
	}
	if !semver.IsValid(check) {
		return false
	}
	core := strings.Split(strings.TrimPrefix(check, "v"), "-")[0]
	return len(strings.Split(core, ".")) == 3
}

func downloadBinary(url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	written, err := io.Copy(out, io.LimitReader(resp.Body, maxDownloadSize+1))
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if written > maxDownloadSize {
		return fmt.Errorf("download size exceeds limit of %d bytes", maxDownloadSize)
	}
	return nil
}

func verifySHA256(checksumURL, filePath string) error {
	req, err := http.NewRequest(http.MethodGet, checksumURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)
	resp, err := httpDoFunc(req)
	if err != nil {
		return fmt.Errorf("failed to fetch checksum %s: %v", checksumURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksum HTTP %d from %s", resp.StatusCode, checksumURL)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return err
	}
	want := strings.Fields(string(body))
	if len(want) == 0 {
		return fmt.Errorf("empty checksum from %s", checksumURL)
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		return err
	}
	got := hex.EncodeToString(sum.Sum(nil))
	if !strings.EqualFold(got, want[0]) {
		return fmt.Errorf("dgate checksum mismatch: expected %s, got %s", want[0], got)
	}
	return nil
}

func removeParentFlags(args []string) []string {
	all := cli.NewFlagSet()
	config.AddFlags(all)
	openapi.AddFlags(all)

	strip := map[string]bool{}
	for _, f := range all.Flags() {
		if f != nil && f.Name != "" && !passthroughParentFlags[f.Name] {
			strip[f.Name] = true
		}
	}

	present := map[string]bool{}
	for _, a := range args {
		if strings.HasPrefix(a, "--") {
			name := strings.TrimPrefix(a, "--")
			if i := strings.IndexAny(name, "=:"); i >= 0 {
				name = name[:i]
			}
			present[name] = true
		} else if strings.HasPrefix(a, "-") && len(a) == 2 {
			present[a] = true
		}
	}

	longNeedsValue := map[string]bool{}
	shortNeedsValue := map[string]bool{}
	for _, f := range all.Flags() {
		if f == nil || !strip[f.Name] {
			continue
		}
		short := ""
		if f.Shorthand != 0 {
			short = "-" + string(f.Shorthand)
		}
		if !present[f.Name] && (short == "" || !present[short]) {
			continue
		}
		needsValue := f.AssignedMode != cli.AssignedNone
		longNeedsValue["--"+f.Name] = needsValue
		for _, alias := range f.Aliases {
			longNeedsValue["--"+alias] = needsValue
		}
		if short != "" {
			shortNeedsValue[short] = needsValue
		}
	}

	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		name := a
		inline := false
		if prefix, _, ok := cli.SplitStringWithPrefix(a, "=:"); ok {
			name = prefix
			inline = true
		}
		if needs, ok := longNeedsValue[name]; ok {
			if needs && !inline && i+1 < len(args) {
				i++
			}
			continue
		}
		if needs, ok := shortNeedsValue[name]; ok {
			if needs && !inline && i+1 < len(args) {
				i++
			}
			continue
		}
		out = append(out, a)
	}
	return out
}

func filterEnv(base []string, overrides map[string]string) []string {
	out := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		key, _, _ := strings.Cut(item, "=")
		if _, conflict := overrides[key]; conflict {
			continue
		}
		out = append(out, item)
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
