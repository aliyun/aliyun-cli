package maxc

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
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
// Task 3.3+ will fill in EnsureInstalledAndUpdated / PrepareEnv / Execute;
// for the skeleton it errors out so callers don't silently no-op.
func (c *Context) Run(args []string) error {
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	if !c.osSupport {
		return fmt.Errorf("your os type %s and arch %s is not supported", c.osType, c.osArch)
	}
	return fmt.Errorf("maxc launcher: download/exec not implemented yet (Task 3.3+)")
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
