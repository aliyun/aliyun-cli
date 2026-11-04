package bl

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/util"
)

const (
	binName       = "bl"
	prefixDirName = "bl"
	releaseBase   = "https://bailian-wiki.oss-cn-hangzhou.aliyuncs.com/release"
	metadataLimit = 1 << 20
	binaryLimit   = 256 << 20
	updateTTL     = 24 * time.Hour

	execPathEnv = "ALIBABA_CLOUD_BL_EXEC_PATH"
	noUpdateEnv = "ALIBABA_CLOUD_BL_NO_UPDATE_CHECK"
)

var numericVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

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
	execFilePath string
	installed    bool

	configPath  func() string
	goos        func() string
	goarch      func() string
	command     func(string, ...string) *exec.Cmd
	environ     func() []string
	getwd       func() (string, error)
	loadProfile func(*cli.Context) (config.Profile, error)
	httpClient  *http.Client
	releaseURL  string
	now         func() time.Time
	rename      func(string, string) error
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
		goarch:      func() string { return runtime.GOARCH },
		command:     exec.Command,
		environ:     os.Environ,
		getwd:       os.Getwd,
		loadProfile: loadBlProfile,
		httpClient:  &http.Client{Timeout: 120 * time.Second},
		releaseURL:  releaseBase,
		now:         time.Now,
		rename:      os.Rename,
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
	if c.usingExecPathOverride() {
		if err := c.validateExecPathOverride(); err != nil {
			return err
		}
	} else if err := c.EnsureBinary(); err != nil {
		return err
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
	name := binName
	if c.goos() == "windows" {
		name += ".exe"
	}
	c.execFilePath = filepath.Join(c.prefixPath, "bin", name)
	if path := strings.TrimSpace(os.Getenv(execPathEnv)); path != "" {
		c.execFilePath = path
	}
	info, err := os.Stat(c.execFilePath)
	c.installed = err == nil && info.Mode().IsRegular() && (c.goos() == "windows" || info.Mode()&0o111 != 0)
	return nil
}

func (c *Context) platform() (string, error) {
	goos, arch := c.goos(), c.goarch()
	if arch == "amd64" {
		arch = "x64"
	}
	platform := goos + "-" + arch
	switch platform {
	case "darwin-arm64", "darwin-x64", "linux-x64", "windows-x64":
		return platform, nil
	default:
		return "", fmt.Errorf("unsupported bl platform %s/%s; supported platforms: darwin-arm64, darwin-x64, linux-x64, windows-x64; or set %s", goos, c.goarch(), execPathEnv)
	}
}

func (c *Context) checkManagedPaths() error {
	for _, path := range []string{c.prefixPath, filepath.Dir(c.execFilePath), c.execFilePath} {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if path == c.prefixPath {
				return fmt.Errorf("managed bl prefix %s must not be a symlink", path)
			}
			root, err := filepath.EvalSymlinks(c.prefixPath)
			if err != nil {
				return err
			}
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("invalid managed bl symlink %s: %w", path, err)
			}
			relative, err := filepath.Rel(root, target)
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || filepath.IsAbs(relative) {
				return fmt.Errorf("managed bl symlink %s escapes install directory", path)
			}
			info, err = os.Stat(target)
			if err != nil {
				return err
			}
		}
		if path != c.execFilePath && !info.IsDir() || path == c.execFilePath && !info.Mode().IsRegular() {
			return fmt.Errorf("invalid managed bl path %s", path)
		}
	}
	return nil
}

func (c *Context) EnsureBinary() error {
	platform, err := c.platform()
	if err != nil {
		return err
	}
	if err := c.checkManagedPaths(); err != nil {
		return err
	}
	if c.installed && (c.envValue(noUpdateEnv) == "1" || c.checkFresh()) {
		return nil
	}
	if err := os.MkdirAll(c.prefixPath, 0755); err != nil {
		return err
	}
	lock := filepath.Join(c.prefixPath, ".install-lock")
	if err := os.Mkdir(lock, 0700); err != nil {
		if os.IsExist(err) {
			if c.installed {
				fmt.Fprintln(c.originCtx.Stderr(), "Warning: another bl installer holds the install lock; continuing with installed binary")
				return nil
			}
			return fmt.Errorf("bl install lock exists at %s; retry after the installer finishes; remove a stale lock only after confirming no installer is running", lock)
		}
		return err
	}
	defer os.Remove(lock)
	if err := c.checkManagedPaths(); err != nil {
		return err
	}
	info, statErr := os.Stat(c.execFilePath)
	c.installed = statErr == nil && info.Mode().IsRegular() && (c.goos() == "windows" || info.Mode()&0o111 != 0)
	var current string
	if c.installed {
		current, err = c.installedVersion()
	}
	if err == nil {
		var asset releaseAsset
		asset, err = c.resolveAsset(platform)
		if err == nil && (!c.installed || newerVersion(asset.Version, current)) {
			fmt.Fprintf(c.originCtx.Stderr(), "Installing bailian-cli %s...\n", asset.Version)
			err = c.installAsset(asset)
			if err == nil {
				c.installed = true
			}
		}
	}
	if c.installed {
		// Throttle failed checks too; never trust cached metadata for the binary's version.
		c.saveCheck()
		if err != nil {
			fmt.Fprintf(c.originCtx.Stderr(), "Warning: bl update check/install failed: %v; continuing with installed binary\n", err)
		}
		return nil
	}
	return fmt.Errorf("failed to install standalone bl: %w", err)
}

func (c *Context) envValue(name string) string {
	value := ""
	for _, entry := range c.environ() {
		key, v, _ := strings.Cut(entry, "=")
		if key == name || c.goos() == "windows" && strings.EqualFold(key, name) {
			value = v
		}
	}
	return value
}

type updateCheck struct {
	CheckedAt time.Time `json:"checkedAt"`
}

func (c *Context) checkFresh() bool {
	f, err := os.Open(filepath.Join(c.prefixPath, "update-check.json"))
	if err != nil {
		return false
	}
	defer f.Close()
	data, err := readBounded(f, metadataLimit)
	var check updateCheck
	if err != nil || json.Unmarshal(data, &check) != nil {
		return false
	}
	age := c.now().Sub(check.CheckedAt)
	return age >= 0 && age < updateTTL
}

func (c *Context) saveCheck() {
	data, _ := json.Marshal(updateCheck{CheckedAt: c.now()})
	f, err := os.CreateTemp(c.prefixPath, ".update-check-*")
	if err != nil {
		return
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	closeErr := f.Close()
	if err == nil && closeErr == nil {
		_ = os.Rename(f.Name(), filepath.Join(c.prefixPath, "update-check.json"))
	}
}

func newerVersion(candidate, current string) bool {
	a, b := strings.Split(candidate, "."), strings.Split(current, ".")
	for i := 0; i < 3; i++ {
		if len(a[i]) != len(b[i]) {
			return len(a[i]) > len(b[i])
		}
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

type releaseAsset struct {
	Version string `json:"-"`
	File    string `json:"file"`
	Inner   string `json:"inner"`
	SHA256  string `json:"sha256"`
}

func expectedAsset(version, platform string) releaseAsset {
	stem := "bl-" + version + "-" + platform
	inner := stem
	if platform == "windows-x64" {
		inner += ".exe"
	}
	return releaseAsset{Version: version, File: stem + ".zip", Inner: inner}
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func (c *Context) resolveAsset(platform string) (releaseAsset, error) {
	data, err := c.fetch("/manifest.json", metadataLimit)
	if err != nil {
		return releaseAsset{}, err
	}
	var manifest struct {
		Name    string                  `json:"name"`
		Channel string                  `json:"channel"`
		Version string                  `json:"version"`
		Assets  map[string]releaseAsset `json:"assets"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return releaseAsset{}, fmt.Errorf("invalid bl manifest: %w", err)
	}
	if manifest.Name != "bailian-cli" || manifest.Channel != "latest" || !numericVersion.MatchString(manifest.Version) {
		return releaseAsset{}, fmt.Errorf("invalid bl manifest identity or version")
	}
	asset := manifest.Assets[platform]
	expected := expectedAsset(manifest.Version, platform)
	if asset.File != expected.File || asset.Inner != expected.Inner || !validDigest(asset.SHA256) {
		return asset, fmt.Errorf("invalid bl manifest asset for %s", platform)
	}
	asset.Version = manifest.Version
	return asset, nil
}

func readBounded(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("bl download exceeds %d bytes", limit)
	}
	return data, nil
}

func (c *Context) fetch(path string, limit int64) ([]byte, error) {
	resp, err := c.httpClient.Get(c.releaseURL + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bl download %s: HTTP %d", path, resp.StatusCode)
	}
	return readBounded(resp.Body, limit)
}

func (c *Context) installAsset(asset releaseAsset) error {
	data, err := c.fetch("/v"+asset.Version+"/"+asset.File, binaryLimit)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), asset.SHA256) {
		return fmt.Errorf("SHA256 mismatch for %s", asset.File)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	if len(archive.File) != 1 {
		return fmt.Errorf("bl ZIP must contain exactly one root executable")
	}
	entry := archive.File[0]
	if entry.Name != asset.Inner || strings.ContainsAny(entry.Name, `/\`) || !entry.Mode().IsRegular() {
		return fmt.Errorf("unsafe bl ZIP entry %q", entry.Name)
	}
	if entry.UncompressedSize64 > binaryLimit {
		return fmt.Errorf("bl executable exceeds %d bytes", binaryLimit)
	}
	r, err := entry.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	if err := c.checkManagedPaths(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.execFilePath), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(c.execFilePath), ".bl-install-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	n, copyErr := io.Copy(f, io.LimitReader(r, binaryLimit+1))
	if copyErr == nil && (n == 0 || n > binaryLimit) {
		copyErr = fmt.Errorf("invalid bl executable size %d", n)
	}
	if copyErr == nil {
		copyErr = f.Chmod(0o755)
	}
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return c.replaceExecutable(f.Name())
}

func (c *Context) replaceExecutable(temp string) error {
	err := c.rename(temp, c.execFilePath)
	if err == nil || c.goos() != "windows" {
		return err
	}
	info, statErr := os.Lstat(c.execFilePath)
	if statErr != nil || !info.Mode().IsRegular() {
		return err
	}
	backup, backupErr := os.CreateTemp(filepath.Dir(c.execFilePath), ".bl-backup-*")
	if backupErr != nil {
		return backupErr
	}
	backupPath := backup.Name()
	if err := backup.Close(); err != nil {
		_ = os.Remove(backupPath)
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		return err
	}
	if err := c.rename(c.execFilePath, backupPath); err != nil {
		return err
	}
	if err := c.rename(temp, c.execFilePath); err != nil {
		if rollbackErr := c.rename(backupPath, c.execFilePath); rollbackErr != nil {
			return fmt.Errorf("bl replacement failed: %v; rollback failed: %v; previous binary retained at %s", err, rollbackErr, backupPath)
		}
		return err
	}
	_ = os.Remove(backupPath)
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
		case key == "PATH":
			pathValue = value
			continue
		case key == "BAILIAN_INSTALL_METHOD" || key == "BAILIAN_SHARE_DIR" || key == "BAILIAN_BIN_DIR" || key == "ALIBABA_CLOUD_BL_COMPAT_MODE":
			continue
		case noUpdate && key == "NO_UPDATE_NOTIFIER":
			continue
		}
		result = append(result, entry)
	}
	binDir := filepath.Join(c.prefixPath, "bin")
	separator := string(os.PathListSeparator)
	if windows {
		separator = ";"
	}
	path := binDir
	if pathValue != "" {
		path += separator + pathValue
	}
	result = append(result,
		"PATH="+path,
		"BAILIAN_INSTALL_METHOD=binary",
		"BAILIAN_SHARE_DIR="+c.prefixPath,
		"BAILIAN_BIN_DIR="+binDir,
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

type versionBuffer struct {
	buffer bytes.Buffer
}

func (b *versionBuffer) Write(p []byte) (int, error) {
	if len(p) > metadataLimit-b.buffer.Len() {
		return 0, fmt.Errorf("bl --version output exceeds limit")
	}
	return b.buffer.Write(p)
}

func (c *Context) installedVersion() (string, error) {
	cmd := c.command(c.execFilePath, "--version")
	cmd.Env = c.buildRuntimeEnv(c.environ())
	var output versionBuffer
	cmd.Stdout, cmd.Stderr = &output, &output
	cmd.WaitDelay = 5 * time.Second
	if err := cmd.Start(); err != nil {
		return "", err
	}
	timer := time.AfterFunc(30*time.Second, func() { _ = cmd.Process.Kill() })
	defer timer.Stop()
	if err := cmd.Wait(); err != nil {
		return "", err
	}
	for _, field := range strings.Fields(output.buffer.String()) {
		version := strings.TrimPrefix(field, "v")
		if numericVersion.MatchString(version) {
			return version, nil
		}
	}
	return "", fmt.Errorf("unexpected bl --version output")
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
