package maxc

import (
	"archive/tar"
	"compress/gzip"
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
)

type Context struct {
	originCtx        *cli.Context
	configPath       string
	installDir       string
	execFilePath     string
	versionCachePath string
	versionFilePath  string
	installed        bool
	versionLocal     string
	versionRemote    string
	osType           string
	osArch           string
	osSupport        bool
	platformKey      string
	envMap           map[string]string
}

type ExitError struct{ Code int }

func (e *ExitError) Error() string { return fmt.Sprintf("subprocess exited with code %d", e.Code) }
func (e *ExitError) ExitCode() int { return e.Code }

var (
	getConfigurePathFunc = func() string { return config.GetConfigPath() }
	runtimeGOOSFunc      = func() string { return runtime.GOOS }
	runtimeGOARCHFunc    = func() string { return runtime.GOARCH }
	execCommandFunc      = exec.Command
	httpGetFunc          = http.Get
	timeNowFunc          = func() time.Time { return time.Now() }
	osSymlinkFunc        = os.Symlink
	ioCopyFunc           = io.Copy
	getLatestVersionFunc = func(c *Context) (string, error) { return c.GetLatestVersion() }
	loadProfileFunc      = func(ctx *cli.Context) (config.Profile, error) {
		return config.LoadProfileWithContext(ctx)
	}
)

var platformPaths = map[string]struct{}{
	"linux-amd64":   {},
	"linux-arm64":   {},
	"darwin-amd64":  {},
	"darwin-arm64":  {},
	"windows-amd64": {},
}

var downloadBaseURL = "https://maxcompute-repo.oss-cn-hangzhou.aliyuncs.com/maxc-cli"

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
	c.envMap["MAXC_CLI_NAME"] = "aliyun maxc"
	childArgs := c.RemoveFlagsForMainCli(args)
	return c.Execute(childArgs)
}

// --- install / update -------------------------------------------------------

func (c *Context) effectiveBaseURL() string {
	if u := os.Getenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return downloadBaseURL
}

func (c *Context) tarballURL(version string) string {
	return fmt.Sprintf("%s/%s/%s/maxc.tar.gz", c.effectiveBaseURL(), version, c.platformKey)
}

func (c *Context) tarballShaURL(version string) string {
	return c.tarballURL(version) + ".sha256"
}

func (c *Context) latestVersionURL() string {
	return c.effectiveBaseURL() + "/versions/latest"
}

func (c *Context) GetLatestVersion() (string, error) {
	resp, err := httpGetFunc(c.latestVersionURL())
	if err != nil {
		return "", fmt.Errorf("GET latest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GET latest: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(body))
	if v == "" {
		return "", fmt.Errorf("empty latest pointer at %s", c.latestVersionURL())
	}
	return v, nil
}

func (c *Context) EnsureInstalledAndUpdated() error {
	if os.Getenv("ALIBABA_CLOUD_MAXC_EXEC_PATH") != "" {
		return nil
	}
	if os.Getenv("ALIBABA_CLOUD_MAXC_NO_UPDATE_CHECK") == "1" {
		return nil
	}

	if !c.installed {
		latest, err := getLatestVersionFunc(c)
		if err != nil {
			return fmt.Errorf("resolve latest maxc version: %w", err)
		}
		return c.downloadAndInstall(latest)
	}

	if !c.cacheStale() {
		return nil
	}
	latest, err := getLatestVersionFunc(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "maxc: update check failed: %v\n", err)
		_ = c.touchCache()
		return nil
	}
	_ = c.touchCache()
	if latest == c.readLocalVersion() {
		return nil
	}
	return c.downloadAndInstall(latest)
}

func (c *Context) downloadAndInstall(version string) error {
	parent := filepath.Dir(c.installDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}

	tarPath := filepath.Join(parent, fmt.Sprintf(".maxc-dl-%d.tar.gz", timeNowFunc().UnixNano()))
	defer os.Remove(tarPath)

	if err := httpGetToFile(c.tarballURL(version), tarPath); err != nil {
		return fmt.Errorf("download tarball: %w", err)
	}

	expectedSha, err := fetchExpectedSha(c.tarballShaURL(version))
	if err != nil {
		return fmt.Errorf("fetch sha256: %w", err)
	}
	actualSha, err := computeFileSha256(tarPath)
	if err != nil {
		return fmt.Errorf("compute sha256: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(actualSha), strings.TrimSpace(expectedSha)) {
		return fmt.Errorf("sha256 mismatch: expected=%s actual=%s", expectedSha, actualSha)
	}

	c.versionRemote = version
	return c.installFromTarball(tarPath)
}

func (c *Context) installFromTarball(tarPath string) error {
	parent := filepath.Dir(c.installDir)
	ts := timeNowFunc().UnixNano()
	stagingDir := filepath.Join(parent, fmt.Sprintf(".maxc-staging-%d", ts))
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return fmt.Errorf("mkdir staging: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	if err := extractTarGz(tarPath, stagingDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	extractedRoot := filepath.Join(stagingDir, "maxc")
	if _, err := os.Stat(extractedRoot); err != nil {
		return fmt.Errorf("tarball missing top-level maxc/ directory: %w", err)
	}

	var oldDir string
	if fileExists(c.installDir) {
		oldDir = fmt.Sprintf("%s.old.%d", c.installDir, ts)
		if err := os.Rename(c.installDir, oldDir); err != nil {
			return fmt.Errorf("rename old install aside: %w", err)
		}
	}

	if err := os.Rename(extractedRoot, c.installDir); err != nil {
		if oldDir != "" {
			_ = os.Rename(oldDir, c.installDir)
		}
		return fmt.Errorf("rename staging into place: %w", err)
	}

	if oldDir != "" {
		_ = os.RemoveAll(oldDir)
	}

	c.installed = true
	c.versionLocal = c.versionRemote
	return c.SaveLocalVersion()
}

func (c *Context) SaveLocalVersion() error {
	return os.WriteFile(c.versionFilePath, []byte(c.versionRemote), 0o644)
}

func httpGetToFile(url, dest string) error {
	resp, err := httpGetFunc(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		return fmt.Errorf("write %s: %w", dest, err)
	}
	return out.Close()
}

func fetchExpectedSha(url string) (string, error) {
	resp, err := httpGetFunc(url)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(string(body))
	if sha == "" {
		return "", fmt.Errorf("empty sha256 from %s", url)
	}
	return sha, nil
}

func computeFileSha256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func extractTarGz(src, destDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gunzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("abs destDir: %w", err)
	}

	type pendingSymlink struct{ path, target string }
	var symlinks []pendingSymlink

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar next: %w", err)
		}

		if filepath.IsAbs(hdr.Name) || strings.Contains(hdr.Name, "..") {
			return fmt.Errorf("tar entry escapes destDir: %s", hdr.Name)
		}

		target := filepath.Join(absDest, hdr.Name)
		if !pathInside(absDest, target) {
			return fmt.Errorf("tar entry escapes destDir: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)|0o700); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			resolved := hdr.Linkname
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(filepath.Dir(target), resolved)
			}
			if !pathInside(absDest, resolved) {
				return fmt.Errorf("symlink %q escapes destDir: -> %s", hdr.Name, hdr.Linkname)
			}
			symlinks = append(symlinks, pendingSymlink{path: target, target: hdr.Linkname})
		default:
			// Skip hardlinks, device files, etc.
		}
	}

	for _, s := range symlinks {
		if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
			return err
		}
		_ = os.Remove(s.path)
		if err := createSymlinkOrCopy(s.path, s.target); err != nil {
			return err
		}
	}
	return nil
}

// createSymlinkOrCopy creates a symlink; on failure (common on non-admin Windows
// without Developer Mode / SeCreateSymbolicLinkPrivilege), falls back to copying
// the target file. Directory links and missing targets are skipped on Windows so
// install is not aborted, matching saectl/plugin untar behavior.
func createSymlinkOrCopy(linkPath, linkTarget string) error {
	err := osSymlinkFunc(linkTarget, linkPath)
	if err == nil {
		return nil
	}
	symlinkErr := err

	src := linkTarget
	if !filepath.IsAbs(src) {
		src = filepath.Join(filepath.Dir(linkPath), src)
	}
	info, statErr := os.Stat(src)
	if statErr == nil && !info.IsDir() {
		if copyErr := copyFileWithMode(src, linkPath, info.Mode().Perm()); copyErr == nil {
			return nil
		}
	}
	if runtimeGOOSFunc() == "windows" {
		// Align with saectl / plugin manager: skip rather than fail install.
		return nil
	}
	return symlinkErr
}

func copyFileWithMode(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := ioCopyFunc(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dst)
		return copyErr
	}
	return closeErr
}

func pathInside(absRoot, p string) bool {
	abs, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (c *Context) cacheStale() bool {
	fi, err := os.Stat(c.versionCachePath)
	if err != nil {
		return true
	}
	return timeNowFunc().Sub(fi.ModTime()).Seconds() > float64(VersionCheckTTL)
}

func (c *Context) touchCache() error {
	if err := os.MkdirAll(filepath.Dir(c.versionCachePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(c.versionCachePath, []byte(timeNowFunc().Format("2006-01-02T15:04:05Z07:00")), 0o644)
}

func (c *Context) readLocalVersion() string {
	b, err := os.ReadFile(c.versionFilePath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// --- credentials ------------------------------------------------------------

func (c *Context) InjectAliyunCredentials(args []string) error {
	if c.envMap == nil {
		c.envMap = map[string]string{}
	}

	profile, err := loadProfileFunc(c.originCtx)
	if err != nil {
		return nil
	}

	id, secret, token, err := extractCredentials(c.originCtx, profile)
	if err != nil {
		return fmt.Errorf("resolve credentials for profile %q: %w", profile.Name, err)
	}

	if id != "" {
		c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"] = id
	}
	if secret != "" {
		c.envMap["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] = secret
	}
	if token != "" {
		c.envMap["ALIBABA_CLOUD_SECURITY_TOKEN"] = token
	}
	if profile.RegionId != "" {
		c.envMap["MAXCOMPUTE_REGION"] = profile.RegionId
	}
	return nil
}

func extractCredentials(ctx *cli.Context, p config.Profile) (string, string, string, error) {
	switch p.Mode {
	case config.AK:
		return p.AccessKeyId, p.AccessKeySecret, "", nil
	case config.StsToken:
		return p.AccessKeyId, p.AccessKeySecret, p.StsToken, nil
	default:
		cred, err := p.GetCredential(ctx, nil)
		if err != nil {
			return "", "", "", err
		}
		model, err := cred.GetCredential()
		if err != nil {
			return "", "", "", err
		}
		var id, secret, token string
		if model.AccessKeyId != nil {
			id = *model.AccessKeyId
		}
		if model.AccessKeySecret != nil {
			secret = *model.AccessKeySecret
		}
		if model.SecurityToken != nil {
			token = *model.SecurityToken
		}
		return id, secret, token, nil
	}
}

// --- flag stripping & execute -----------------------------------------------

// passthrough lists flags that exist in config/openapi but should NOT be
// stripped — the child process may define its own with the same name.
var passthrough = map[string]struct{}{
	"output":    {},
	"force":     {},
	"version":   {},
	"body":      {},
	"body-file": {},
}

func (c *Context) RemoveFlagsForMainCli(args []string) []string {
	allFlags := cli.NewFlagSet()
	config.AddFlags(allFlags)
	openapi.AddFlags(allFlags)

	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		argName := a
		hasInlineValue := false
		if prefix, _, ok := cli.SplitStringWithPrefix(a, "=:"); ok {
			argName = prefix
			hasInlineValue = true
		}

		var f *cli.Flag
		if strings.HasPrefix(argName, "--") {
			f = allFlags.Get(strings.TrimPrefix(argName, "--"))
		} else if strings.HasPrefix(argName, "-") && len(argName) == 2 {
			f = allFlags.GetByShorthand(rune(argName[1]))
		}

		if f != nil {
			if _, keep := passthrough[f.Name]; keep {
				out = append(out, a)
				continue
			}
			needsValue := f.AssignedMode != cli.AssignedNone
			if needsValue && !hasInlineValue && i+1 < len(args) {
				i++
			}
			continue
		}

		out = append(out, a)
	}
	return out
}

func (c *Context) Execute(childArgs []string) error {
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

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
