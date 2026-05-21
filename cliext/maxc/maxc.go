package maxc

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

// Context holds per-invocation state for the maxc launcher. Mirrors the
// shape of cliext/cms2.Context but adapted for an onedir bundle (the maxc
// "binary" is actually a directory tree extracted from maxc.tar.gz, and
// the runtime entry point is installDir/maxc[.exe]).
type Context struct {
	originCtx        *cli.Context
	configPath       string
	installDir       string // <configPath>/maxc, contains the extracted onedir
	execFilePath     string // <installDir>/maxc[.exe]
	versionCachePath string // <installDir>/.version_check
	versionFilePath  string // <installDir>/.version
	installed        bool
	versionLocal     string
	versionRemote    string
	osType           string
	osArch           string
	osSupport        bool
	platformKey      string
	envMap           map[string]string
}

// ExitError carries the child process exit code so the caller can propagate
// it without calling os.Exit directly (which would skip deferred cleanup).
type ExitError struct{ Code int }

func (e *ExitError) Error() string { return fmt.Sprintf("subprocess exited with code %d", e.Code) }
func (e *ExitError) ExitCode() int { return e.Code }

// Package-level function vars are the cms2 testability pattern — every
// external dependency is monkey-patchable from tests. Add more as later
// tasks introduce them (httpGetFunc, httpDoFunc, downloadFileFunc, etc).
var (
	getConfigurePathFunc = func() string { return config.GetConfigPath() }
	runtimeGOOSFunc      = func() string { return runtime.GOOS }
	runtimeGOARCHFunc    = func() string { return runtime.GOARCH }
	execCommandFunc      = exec.Command
)

// The six platforms the maxc release pipeline produces, matching the OSS
// directory layout under maxc-cli/{version}/{platform}/maxc.tar.gz.
// Tarball platform names use Go's GOOS/GOARCH-style strings.
var platformPaths = map[string]struct{}{
	"linux-amd64":   {},
	"linux-arm64":   {},
	"darwin-amd64":  {},
	"darwin-arm64":  {},
	"windows-amd64": {},
	"windows-arm64": {},
}

// downloadBaseURL is the canonical public bucket prefix. Overridable at
// runtime via ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL (see Task 3.3 for the
// env-override accessor). Trailing slash is significant — keep it off.
var downloadBaseURL = "https://maxcompute-repo.oss-cn-hangzhou.aliyuncs.com/maxc-cli"

// VersionCheckTTL throttles the "is there a newer maxc?" HTTP call to once
// per day. Matches cms2's TTL — there's no reason to be more aggressive.
const VersionCheckTTL = 86400

func NewContext(origin *cli.Context) *Context {
	return &Context{originCtx: origin}
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.installDir = filepath.Join(c.configPath, "maxc")
	binName := "maxc"
	if runtimeGOOSFunc() == "windows" {
		binName += ".exe"
	}
	c.execFilePath = filepath.Join(c.installDir, binName)
	c.versionCachePath = filepath.Join(c.installDir, ".version_check")
	c.versionFilePath = filepath.Join(c.installDir, ".version")

	if envPath := os.Getenv("ALIBABA_CLOUD_MAXC_EXEC_PATH"); envPath != "" {
		c.execFilePath = envPath
	}
	c.installed = fileExists(c.execFilePath)
}

func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()
	c.platformKey = c.osType + "-" + c.osArch
	if _, ok := platformPaths[c.platformKey]; ok {
		c.osSupport = true
	}
}

// Run is the top-level entrypoint invoked by main.go's NewMaxcCommand.
// Sequence: derive paths → check platform → install/update if needed →
// inject aliyun credentials into env → strip parent-only flags → exec.
// On a soft update-check failure when something is already installed we
// continue (with a stderr warning) rather than blocking the user.
func (c *Context) Run(args []string) error {
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if !c.osSupport {
		return fmt.Errorf("your os type %s and arch %s is not supported", c.osType, c.osArch)
	}
	if err := c.EnsureInstalledAndUpdated(); err != nil {
		if !c.installed {
			return err
		}
		fmt.Fprintf(c.originCtx.Stderr(), "Warning: maxc update check failed: %v\n", err)
	}
	if err := c.InjectAliyunCredentials(args); err != nil {
		return err
	}
	childArgs := c.RemoveFlagsForMainCli(args)
	return c.Execute(childArgs)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// cacheStale returns true when the version-check sentinel is missing or older
// than VersionCheckTTL seconds. Mirrors cms2's once-per-day throttle policy.
func (c *Context) cacheStale() bool {
	fi, err := os.Stat(c.versionCachePath)
	if err != nil {
		return true
	}
	return timeNowFunc().Sub(fi.ModTime()).Seconds() > float64(VersionCheckTTL)
}

// touchCache writes (or rewrites) the version-check sentinel so cacheStale
// reports false for the next TTL window. The file content is irrelevant —
// only its mtime matters.
func (c *Context) touchCache() error {
	if err := os.MkdirAll(filepath.Dir(c.versionCachePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(c.versionCachePath, []byte(timeNowFunc().Format("2006-01-02T15:04:05Z07:00")), 0o644)
}

// readLocalVersion returns the trimmed .version file contents, or "" if
// unreadable. Used to decide whether a remote latest pointer actually means
// "upgrade" or "already on it".
func (c *Context) readLocalVersion() string {
	b, err := os.ReadFile(c.versionFilePath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// stripFlags lists parent-CLI flag names that are only meaningful to the
// aliyun root command and must NOT leak into the maxc child process.
// These are auth/config flags whose values are already exported via
// envMap (see InjectAliyunCredentials), or runtime knobs that have no
// equivalent in the maxc-cli vocabulary.
//
// Anything not listed here passes through, so legitimate maxc flags
// (e.g. --sql, --project, --output) keep their child-side meaning even
// if the parent CLI happens to define a flag of the same name.
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

// RemoveFlagsForMainCli walks args and drops every parent-CLI-only flag
// listed in stripFlags (handling both `--flag value` and `--flag=value`
// forms, plus shorthand `-x value`). Identical strategy to cliext/cms2 —
// kept verbatim so behaviour stays consistent across cliext launchers.
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

// Execute spawns c.execFilePath with childArgs, wiring stdio through and
// merging c.envMap on top of the inherited environment. Non-zero exit codes
// surface as *ExitError so the caller can propagate without calling os.Exit
// directly (which would skip deferred cleanup).
func (c *Context) Execute(childArgs []string) error {
	// Close idle HTTP connections before fork — on macOS there's a race
	// between socket() and the FD_CLOEXEC flag where a child can inherit
	// a half-ready fd and then fail with "bad file descriptor". cms2
	// hit this in production; we copy the workaround.
	http.DefaultClient.CloseIdleConnections()

	cmd := execCommandFunc(c.execFilePath, childArgs...)
	cmd.Env = mergeEnv(os.Environ(), c.envMap)
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return &ExitError{Code: ee.ExitCode()}
		}
		return fmt.Errorf("execute %s %v: %w", c.execFilePath, childArgs, err)
	}
	return nil
}

// mergeEnv returns base with any keys present in overrides removed, then
// appends "k=v" pairs for every override. Ensures the override map wins
// even when the parent already had the same variable set.
func mergeEnv(base []string, overrides map[string]string) []string {
	out := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		key, _, _ := strings.Cut(item, "=")
		if _, conflict := overrides[key]; conflict {
			continue
		}
		out = append(out, item)
	}
	for k, v := range overrides {
		out = append(out, k+"="+v)
	}
	return out
}
