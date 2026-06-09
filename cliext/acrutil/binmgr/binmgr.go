// Package binmgr provides a shared installer/updater for the external
// binaries that the acrutil subcommands wrap (acr-skill, cr-diagnosis, ...).
//
// Each subcommand supplies a Config and binmgr.Manager handles download,
// version check, TTL caching, atomic file replace, argv-flag scrubbing and
// subprocess execution.
package binmgr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/mod/semver"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

// parseVersionParts 将 "v0.1.0-a1b2c3d" 拆分为 semver 部分和 commit 部分
// 注意：semver 本身可能包含 pre-release 如 "v1.0.0-rc.1"，所以需要找到 commit hash 的分隔符
// 策略：最后一个 "-" 后面如果是 7-40 位 hex 字符则视为 commit
func parseVersionParts(version string) (semverPart string, commitPart string, valid bool) {
	idx := strings.LastIndex(version, "-")
	if idx <= 0 {
		return "", version, false
	}
	commit := version[idx+1:]
	ver := version[:idx]
	// commit 应该是短/长 git hash（7-40位十六进制）
	if len(commit) < 7 || len(commit) > 40 {
		return "", version, false
	}
	for _, c := range commit {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", version, false
		}
	}
	if !semver.IsValid(ver) {
		return "", version, false
	}
	return ver, commit, true
}

// needsUpdate 判断是否需要从 local 更新到 remote
// 逻辑：
//  1. 相同则不更新
//  2. 两者都能解析为 {VERSION}-{COMMIT} → semver 比较 VERSION 部分
//     - remote VERSION > local VERSION → 更新（升级）
//     - remote VERSION == local VERSION 但 COMMIT 不同 → 更新（同版本 patch）
//     - remote VERSION < local VERSION → 不更新（防止降级）
//  3. 无法解析 → 降级为字符串不等比较（local != remote 则更新）
func needsUpdate(local, remote string) bool {
	if local == remote {
		return false
	}
	if local == "" {
		return true
	}
	localVer, localCommit, localOK := parseVersionParts(local)
	remoteVer, remoteCommit, remoteOK := parseVersionParts(remote)
	if !localOK || !remoteOK {
		return true
	}
	cmp := semver.Compare(localVer, remoteVer)
	if cmp < 0 {
		return true
	}
	if cmp > 0 {
		return false
	}
	// 同版本，比较 commit
	return localCommit != remoteCommit
}

// VersionCheckTTL is the throttle interval between remote version checks.
const VersionCheckTTL = 24 * time.Hour

// MaxDownloadSize caps a single binary download.
const MaxDownloadSize = 50 * 1024 * 1024

// Config describes one managed binary.
type Config struct {
	// Name is the binary file name (e.g. "acr-skill").
	Name string
	// BaseURL is the OSS prefix; the published layout is {BaseURL}{version}/{Name}-{os}-{arch}.
	// Must end with a trailing slash.
	BaseURL string
	// EnvCompatMode is the env var name used to tell the child it was invoked through the parent CLI.
	EnvCompatMode string
	// EnvCompatModeVal is the value for EnvCompatMode (e.g. "aliyun acrutil skill").
	EnvCompatModeVal string
	// EnvUserAgent is the env var name through which we propagate the parent CLI's user agent.
	EnvUserAgent string
	// PlatformPaths lists the os-arch combinations (e.g. "linux-amd64",
	// "darwin-arm64") the binary is published for. Each subcommand supplies its
	// own set; when empty no platform is considered supported.
	PlatformPaths map[string]struct{}
	// StripFlags is the set of parent-CLI long flag names that must NOT be
	// forwarded to the child binary. Each subcommand supplies its own complete
	// set based on its child CLI's flags: a parent flag whose long name collides
	// with one the child owns (e.g. cr-diagnosis's --mode/--yes/--quiet) must be
	// omitted so it is passed through. When nil, no long flag is stripped.
	StripFlags map[string]bool
	// ShortFlagsToStrip is the set of parent-CLI short flags (e.g. "-p") to
	// strip. Normally empty because short flags frequently collide with child
	// semantics (e.g. -p = --profile vs --password). When nil, nothing is
	// stripped by short name.
	ShortFlagsToStrip map[string]bool
}

// httpClient is reused for the (potentially large) binary download.
var httpClient = &http.Client{Timeout: 5 * time.Minute}

// httpDoFunc is used by GetLatestVersion for the version-check HTTP call.
// Package-level to allow internal unit tests of GetLatestVersion to swap it.
var httpDoFunc = func(req *http.Request) (*http.Response, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

// Manager handles install/update/exec for one managed binary.
// All injectable dependencies are per-instance fields, making Manager safe
// for use in parallel tests without global state races.
type Manager struct {
	cfg                       Config
	originCtx                 *cli.Context
	configPath                string
	execFilePath              string
	versionFilePath           string
	checkVersionCacheFilePath string
	versionLocal              string
	versionRemote             string
	installed                 bool
	osType                    string
	osArch                    string
	osSupport                 bool
	downloadPathSuffix        string

	// Per-instance deps (set to production defaults by New).
	downloadBinaryFn   func(ctx context.Context, url string, exeFilePath string) error
	execCommandFn      func(ctx context.Context, name string, arg ...string) *exec.Cmd
	runtimeGOOSFn      func() string
	runtimeGOARCHFn    func() string
	getLatestVersionFn func(ctx context.Context, baseURL string) (string, error)
	timeNowFn          func() time.Time
	getConfigurePathFn func() string
}

// New returns a Manager bound to cfg and the CLI invocation ctx.
func New(cfg Config, ctx *cli.Context) *Manager {
	return &Manager{
		cfg:                cfg,
		originCtx:          ctx,
		downloadBinaryFn:   DownloadBinary,
		execCommandFn:      exec.CommandContext,
		runtimeGOOSFn:      func() string { return runtime.GOOS },
		runtimeGOARCHFn:    func() string { return runtime.GOARCH },
		getLatestVersionFn: GetLatestVersion,
		timeNowFn:          time.Now,
		getConfigurePathFn: func() string { return config.GetConfigPath() },
	}
}

// Run executes the full lifecycle: validate platform, ensure installed and
// up-to-date, scrub parent-CLI-only flags, exec the child binary.
func (m *Manager) Run(args []string) error {
	if err := m.InitializeAndValidatePlatform(); err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := m.EnsureInstalledAndUpdated(ctx); err != nil {
		return err
	}
	if !m.installed {
		return fmt.Errorf("%s binary not found at %s", m.cfg.Name, m.execFilePath)
	}

	newArgs, err := m.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}
	return m.Execute(ctx, newArgs)
}

func (m *Manager) InitializeAndValidatePlatform() error {
	m.InitBasicInfo()
	m.CheckOsTypeAndArch()
	if !m.osSupport {
		return fmt.Errorf("your os type %s and arch %s is not supported now", m.osType, m.osArch)
	}
	return nil
}

func (m *Manager) InitBasicInfo() {
	m.configPath = m.getConfigurePathFn()
	m.execFilePath = filepath.Join(m.configPath, m.cfg.Name)
	m.versionFilePath = filepath.Join(m.configPath, "."+m.cfg.Name+"_version")
	m.checkVersionCacheFilePath = filepath.Join(m.configPath, "."+m.cfg.Name+"_version_check")
	m.installed = fileExists(m.execFilePath)
}

func (m *Manager) CheckOsTypeAndArch() {
	m.osType = m.runtimeGOOSFn()
	m.osArch = m.runtimeGOARCHFn()
	platformKey := m.osType + "-" + m.osArch
	if _, exists := m.cfg.PlatformPaths[platformKey]; exists {
		m.osSupport = true
		m.downloadPathSuffix = platformKey
	}
}

// EnsureInstalledAndUpdated ensures the binary exists locally and is up to date.
// Fresh-install failures propagate as errors. Update failures are surfaced to
// stderr but never block execution of the already-installed binary, and the
// version-check cache is stamped on every completed check (success or failure)
// to avoid a thundering herd on transient network errors.
func (m *Manager) EnsureInstalledAndUpdated(ctx context.Context) error {
	if !m.installed {
		latest, err := m.getLatestVersionFn(ctx, m.cfg.BaseURL)
		if err != nil {
			return fmt.Errorf("%s is not installed and auto-download failed: %v", m.cfg.Name, err)
		}
		m.versionRemote = latest
		if err := m.Install(ctx); err != nil {
			return err
		}
		m.warnIfError("update version cache time", m.UpdateCheckCacheTime())
		return nil
	}

	if !m.NeedCheckVersion() {
		return nil
	}

	// Stamp the cache time before returning regardless of which branch we
	// take below — a failed check that doesn't stamp the cache forces every
	// subsequent CLI invocation to repeat the network round-trip.
	defer func() { m.warnIfError("update version cache time", m.UpdateCheckCacheTime()) }()

	latest, err := m.getLatestVersionFn(ctx, m.cfg.BaseURL)
	if err != nil {
		fmt.Fprintf(m.originCtx.Stderr(),
			"Warning: failed to check for %s updates: %v\n", m.cfg.Name, err)
		return nil
	}
	m.versionRemote = latest

	if err := m.GetLocalVersion(); err != nil {
		fmt.Fprintf(m.originCtx.Stderr(),
			"Warning: failed to read local %s version: %v\n", m.cfg.Name, err)
	}
	if needsUpdate(m.versionLocal, m.versionRemote) {
		if err := m.Install(ctx); err != nil {
			fmt.Fprintf(m.originCtx.Stderr(),
				"Warning: failed to update %s to %s: %v\n", m.cfg.Name, m.versionRemote, err)
		}
	}
	return nil
}

// NeedCheckVersion returns true if the cached check timestamp is stale,
// missing, corrupted, or in the future (clock skew protection).
// It also returns true when the version file is absent — this covers legacy
// installs (e.g. an old acr-skill without version tracking) that must be
// upgraded to a version-aware build.
func (m *Manager) NeedCheckVersion() bool {
	if !m.installed {
		return false
	}
	if !fileExists(m.versionFilePath) {
		return true
	}
	if !fileExists(m.checkVersionCacheFilePath) {
		return true
	}
	data, err := os.ReadFile(m.checkVersionCacheFilePath)
	if err != nil {
		return true
	}
	lastCheckTime, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || lastCheckTime <= 0 {
		return true
	}
	now := m.timeNowFn().Unix()
	delta := now - lastCheckTime
	// Treat future cache timestamps as expired so a clock skew (NTP glitch,
	// container clock jump) cannot permanently disable the version check.
	if delta < 0 {
		return true
	}
	return delta > int64(VersionCheckTTL.Seconds())
}

// GetLatestVersion fetches the latest published version string from {baseURL}latest/version.txt.
func GetLatestVersion(ctx context.Context, baseURL string) (string, error) {
	url := baseURL + "latest/version.txt"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", fmt.Errorf("failed to read response body from %s: %v", url, err)
	}

	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", fmt.Errorf("version.txt is empty")
	}
	return version, nil
}

// GetLocalVersion reads the local version file. A missing version file is not
// an error — versionLocal is set to "" so the caller can detect a mismatch and
// reinstall. Returns a non-nil error only when the file exists but is unreadable
// or empty.
func (m *Manager) GetLocalVersion() error {
	m.versionLocal = ""
	if !m.installed {
		return fmt.Errorf("%s not installed", m.cfg.Name)
	}
	if !fileExists(m.versionFilePath) {
		return nil
	}
	data, err := os.ReadFile(m.versionFilePath)
	if err != nil {
		return fmt.Errorf("failed to read version file %s: %v", m.versionFilePath, err)
	}
	m.versionLocal = strings.TrimSpace(string(data))
	if m.versionLocal == "" {
		return fmt.Errorf("version file %s is empty", m.versionFilePath)
	}
	return nil
}

func (m *Manager) SaveLocalVersion() error {
	return os.WriteFile(m.versionFilePath, []byte(m.versionLocal+"\n"), 0644)
}

func (m *Manager) UpdateCheckCacheTime() error {
	data := strconv.FormatInt(m.timeNowFn().Unix(), 10)
	return os.WriteFile(m.checkVersionCacheFilePath, []byte(data), 0644)
}

// GetDownloadURL builds {BaseURL}{version}/{Name}-{os}-{arch}.
func (m *Manager) GetDownloadURL() (string, error) {
	if _, ok := m.cfg.PlatformPaths[m.downloadPathSuffix]; !ok {
		return "", fmt.Errorf("unsupported platform: %s", m.downloadPathSuffix)
	}
	if m.versionRemote == "" {
		return "", fmt.Errorf("cannot build download URL: version is empty")
	}
	return fmt.Sprintf("%s%s/%s-%s", m.cfg.BaseURL, m.versionRemote, m.cfg.Name, m.downloadPathSuffix), nil
}

// Install downloads the binary at versionRemote. A failed SaveLocalVersion is
// surfaced as a warning but does not fail Install — the binary itself is on
// disk and the version-file mismatch will self-heal on the next run.
func (m *Manager) Install(ctx context.Context) error {
	url, err := m.GetDownloadURL()
	if err != nil {
		return err
	}
	if err := m.downloadBinaryFn(ctx, url, m.execFilePath); err != nil {
		return fmt.Errorf("failed to download %s from %s: %v", m.cfg.Name, url, err)
	}
	m.installed = true
	m.versionLocal = m.versionRemote
	m.warnIfError("save version file", m.SaveLocalVersion())
	return nil
}

// DownloadBinary fetches url and atomically replaces exeFilePath.
// Uses a unique per-process temp file so concurrent CLI invocations cannot
// trample each other's download.
func DownloadBinary(ctx context.Context, url string, exeFilePath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	tmp, err := os.CreateTemp(filepath.Dir(exeFilePath), filepath.Base(exeFilePath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	tmpPath := tmp.Name()
	cleanupTmp := func() { _ = os.Remove(tmpPath) }

	// LimitReader is given size+1 so a body that exactly equals MaxDownloadSize
	// passes but anything larger trips the post-Copy size check.
	written, err := io.Copy(tmp, io.LimitReader(resp.Body, MaxDownloadSize+1))
	if err != nil {
		_ = tmp.Close()
		cleanupTmp()
		return fmt.Errorf("failed to write to file %s: %v", tmpPath, err)
	}
	if written > MaxDownloadSize {
		_ = tmp.Close()
		cleanupTmp()
		return fmt.Errorf("download size exceeds limit of %d bytes", MaxDownloadSize)
	}
	if err := tmp.Close(); err != nil {
		cleanupTmp()
		return fmt.Errorf("failed to close file %s: %v", tmpPath, err)
	}

	// On POSIX, Rename atomically replaces the target — no need to Remove
	// the old binary first (doing so would open an ENOENT window where a
	// concurrent caller could see the binary as missing).
	if err := os.Rename(tmpPath, exeFilePath); err != nil {
		cleanupTmp()
		return fmt.Errorf("failed to rename %s to %s: %v", tmpPath, exeFilePath, err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(exeFilePath, 0755); err != nil {
			return fmt.Errorf("failed to set exec permission for file %s: %v", exeFilePath, err)
		}
	}
	return nil
}

// RemoveFlagsForMainCli strips parent-CLI-only flags (per the Config's
// StripFlags / ShortFlagsToStrip) from args before forwarding them to the child
// subprocess. When the Config leaves StripFlags nil, no long flag is stripped.
func (m *Manager) RemoveFlagsForMainCli(args []string) ([]string, error) {
	return removeFlagsForMainCli(args, m.cfg.StripFlags, m.cfg.ShortFlagsToStrip)
}

func removeFlagsForMainCli(args []string, stripFlags, shortFlagsToStrip map[string]bool) ([]string, error) {
	argsFlagSet := make(map[string]bool)
	for _, a := range args {
		if strings.HasPrefix(a, "--") {
			flagName := a[2:]
			if idx := strings.Index(flagName, "="); idx != -1 {
				flagName = flagName[:idx]
			}
			argsFlagSet[flagName] = true
		} else if strings.HasPrefix(a, "-") && len(a) == 2 {
			argsFlagSet[a] = true
		}
	}

	allFlags := cli.NewFlagSet()
	config.AddFlags(allFlags)
	openapi.AddFlags(allFlags)

	longNeedsValue := make(map[string]bool)
	shortNeedsValue := make(map[string]bool)
	for _, f := range allFlags.Flags() {
		if !stripFlags[f.Name] {
			continue
		}
		hasLongFlag := argsFlagSet[f.Name]
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

// Execute runs the managed binary with the given args, propagating stdio and
// the configured compat-mode / user-agent env vars.
func (m *Manager) Execute(ctx context.Context, args []string) error {
	cmd := m.execCommandFn(ctx, m.execFilePath, args...)
	cmd.Stdout = m.originCtx.Stdout()
	cmd.Stderr = m.originCtx.Stderr()
	cmd.Stdin = os.Stdin

	ua := "aliyun-cli/" + cli.Version
	if m.versionLocal != "" {
		ua += " " + m.cfg.Name + "/" + m.versionLocal
	}
	if f := m.originCtx.Flags().Get("user-agent"); f != nil {
		if v, ok := f.GetValue(); ok && v != "" {
			ua += " " + v
		}
	}
	overrides := map[string]string{
		m.cfg.EnvCompatMode: m.cfg.EnvCompatModeVal,
		m.cfg.EnvUserAgent:  ua,
	}
	envs := filterEnv(os.Environ(), overrides)
	for k, v := range overrides {
		envs = append(envs, k+"="+v)
	}
	cmd.Env = envs

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %s: %v", m.execFilePath, err)
	}
	return nil
}

func (m *Manager) warnIfError(action string, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(m.originCtx.Stderr(),
		"Warning: failed to %s for %s: %v\n", action, m.cfg.Name, err)
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

// BaseStripFlags returns the standard set of parent-CLI long flag names that
// child binaries typically do not use. Subcommands should copy and customize:
// delete entries the child owns (so they pass through), or add extra entries.
func BaseStripFlags() map[string]bool {
	return map[string]bool{
		// config: auth & credential
		"profile":                        true,
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
		"cli-query":      true,
		"roa":            true,
		"method":         true,
		"user-agent":     true,
		"cli-ai-mode":    true,
		"no-cli-ai-mode": true,
	}
}

// SetExecFilePathForTest is exported for use by test packages outside binmgr.
func (m *Manager) SetExecFilePathForTest(path string) {
	m.execFilePath = path
}

// SetExecCommandForTest overrides the exec.CommandContext function for this
// Manager instance. Exported for use by test packages outside binmgr.
func (m *Manager) SetExecCommandForTest(fn func(ctx context.Context, name string, arg ...string) *exec.Cmd) {
	m.execCommandFn = fn
}
