package iact3

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

type Context struct {
	originCtx          *cli.Context
	configPath         string
	execFilePath       string
	installed          bool
	versionLocal       string
	versionRemote      string
	osType             string
	osArch             string
	osSupport          bool
	downloadPathSuffix string
	envMap             map[string]string
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

var (
	getLatestIact3VersionFunc = GetLatestIact3Version
	downloadAndExtractFunc    = DownloadAndExtract
	execCommandFunc           = exec.Command
	httpGetFunc               = downloadHTTPGet
	runtimeGOOSFunc           = func() string { return runtime.GOOS }
	runtimeGOARCHFunc         = func() string { return runtime.GOARCH }
	timeNowFunc               = func() time.Time { return time.Now() }
)

// versionCheckTTL bounds how often we ask upstream for the latest version
// when emitting the "new version available" hint. Set to 24h to stay light
// (≤1 HTTP GET per user per day) while still surfacing new releases within
// a reasonable window.
const versionCheckTTL = 24 * time.Hour

// versionCacheFileName lives next to the cached iact3 binary in the aliyun
// configure dir (~/.aliyun by default).
const versionCacheFileName = ".iact3_version_check"

// iact3VersionCache persists what we know about the local + upstream iact3
// versions so we can show an upgrade hint without paying the cost of a
// network call on every aliyun iact3 invocation. Soft-failing on any I/O
// or JSON error keeps the hint mechanism strictly best-effort.
type iact3VersionCache struct {
	InstalledVersion string `json:"installed_version"`
	LastKnownRemote  string `json:"last_known_remote"`
	LastRemoteCheck  int64  `json:"last_remote_check"`
}

// downloadHTTPGet wraps http.Get with an overall timeout so a stalled
// connection or slow OSS source can not hang the CLI indefinitely.
// 10 minutes is generous enough for a ~16MB tar.gz over a slow link.
func downloadHTTPGet(url string) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Minute}
	return client.Get(url)
}

var iact3BaseUrl = "https://ros-public-tools.oss-cn-beijing.aliyuncs.com/github-releases/aliyun/alibabacloud-ros-tool-iact3/"

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

	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}

	if len(newArgs) > 0 && newArgs[0] == "upgrade" {
		return c.SelfUpgrade()
	}

	if !c.installed {
		latestVersionRemote, err := getLatestIact3VersionFunc()
		if err != nil {
			return err
		}
		c.versionRemote = latestVersionRemote
		err = c.Install()
		if err != nil {
			return err
		}
		c.installed = true
		c.writeVersionCache(latestVersionRemote, latestVersionRemote)
	}

	err = c.PrepareEnv()
	if err != nil {
		return err
	}

	cmd := execCommandFunc(c.execFilePath, newArgs...)
	envs := os.Environ()
	for k, v := range c.envMap {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = envs
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute %s %v: %v", c.execFilePath, newArgs, err)
	}

	c.maybeShowUpgradeHint()
	return nil
}

// CheckOsTypeAndArch marks osSupport=true only for the platforms that
// upstream ros-public-tools actually publishes iact3 binaries for:
//   - linux/amd64
//   - darwin/arm64 (Apple Silicon)
//
// Other OS/arch combinations (incl. all of windows, linux/arm64,
// darwin/amd64) are intentionally unsupported here so Run() surfaces a
// clear error instead of attempting a download that will 404.
func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()

	switch {
	case c.osType == "linux" && c.osArch == "amd64":
		c.osSupport = true
		c.downloadPathSuffix = "linux-amd64.tar.gz"
	case c.osType == "darwin" && c.osArch == "arm64":
		c.osSupport = true
		c.downloadPathSuffix = "darwin-arm64.tar.gz"
	}
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.execFilePath = filepath.Join(c.configPath, "iact3")
	c.installed = fileExists(c.execFilePath)
}

func (c *Context) Install() error {
	url := fmt.Sprintf("%s%s/iact3-%s-%s", iact3BaseUrl, c.versionRemote, c.versionRemote, c.downloadPathSuffix)

	tmpFile := filepath.Join(os.TempDir(), "iact3_download.tar.gz")

	if fileExists(tmpFile) {
		if err := os.Remove(tmpFile); err != nil {
			return err
		}
	}

	err := downloadAndExtractFunc(url, tmpFile, c.execFilePath)
	if err != nil {
		return fmt.Errorf("failed to download and extract iact3 from %s: %v", url, err)
	}
	return nil
}

func DownloadAndExtract(url string, destFile string, exeFilePath string) error {
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
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		_ = out.Close()
		return fmt.Errorf("failed to write to file %s: %v", destFile, err)
	}
	if err = out.Close(); err != nil {
		return fmt.Errorf("failed to close file %s: %v", destFile, err)
	}

	destDir := filepath.Dir(destFile)
	destDirUse := filepath.Join(destDir, "iact3_extract")
	if fileExists(destDirUse) {
		if err := os.RemoveAll(destDirUse); err != nil {
			return fmt.Errorf("failed to remove existing dir %s: %v", destDirUse, err)
		}
	}

	if err := extractTarGz(destFile, destDirUse); err != nil {
		return fmt.Errorf("failed to extract file %s: %v", destFile, err)
	}

	sourceFile := filepath.Join(destDirUse, "iact3")
	if !fileExists(sourceFile) {
		return fmt.Errorf("extracted file %s not exist", sourceFile)
	}

	if fileExists(exeFilePath) {
		if err := os.Remove(exeFilePath); err != nil {
			return fmt.Errorf("failed to remove existing file %s: %v", exeFilePath, err)
		}
	}

	if err := copyFile(sourceFile, exeFilePath); err != nil {
		return err
	}

	if err := os.Chmod(exeFilePath, 0755); err != nil {
		return fmt.Errorf("failed to set exec permission for file %s: %v", exeFilePath, err)
	}

	_ = os.Remove(destFile)
	_ = os.RemoveAll(destDirUse)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %v", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %v", dst, err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %v", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %v", src, dst, err)
	}
	return nil
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	destClean := filepath.Clean(dest)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		cleanName, err := sanitizeArchivePath(hdr.Name)
		if err != nil {
			continue
		}
		if cleanName == "" {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeSymlink, tar.TypeLink:
			continue
		case tar.TypeDir:
			continue
		case tar.TypeReg, tar.TypeRegA:
			if !isWantedIact3Entry(cleanName) {
				continue
			}
			if hdr.Size < 0 || hdr.Size > maxExtractSize {
				continue
			}

			filePath, err := safeJoinUnderDir(destClean, cleanName)
			if err != nil {
				continue
			}

			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return err
			}

			outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
			if err != nil {
				return err
			}
			if _, err := io.CopyN(outFile, tr, hdr.Size); err != nil {
				outFile.Close()
				return err
			}
			if err := outFile.Close(); err != nil {
				return err
			}
		default:
			continue
		}
	}
	return nil
}

const maxExtractSize int64 = 200 << 20

func isWantedIact3Entry(cleanName string) bool {
	return path.Base(cleanName) == "iact3"
}

func sanitizeArchivePath(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty entry name")
	}
	if strings.ContainsRune(name, '\x00') {
		return "", fmt.Errorf("NUL byte in entry name")
	}

	name = strings.ReplaceAll(name, "\\", "/")
	for strings.HasPrefix(name, "./") {
		name = strings.TrimPrefix(name, "./")
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "//") {
		return "", fmt.Errorf("absolute/UNC path is not allowed")
	}
	if len(name) >= 2 && isASCIIAlpha(name[0]) && name[1] == ':' {
		return "", fmt.Errorf("windows drive path is not allowed")
	}

	clean := path.Clean(name)
	if clean == "." {
		return "", nil
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path traversal is not allowed")
	}
	return clean, nil
}

func safeJoinUnderDir(destDir, cleanRel string) (string, error) {
	destClean := filepath.Clean(destDir)
	joined := filepath.Join(destClean, filepath.FromSlash(cleanRel))
	rel, err := filepath.Rel(destClean, joined)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes destination directory")
	}
	return joined, nil
}

func isASCIIAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func (c *Context) GetLocalVersion() error {
	if !c.installed {
		c.versionLocal = ""
		return fmt.Errorf("iact3 not installed")
	}
	cmd := execCommandFunc(c.execFilePath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute %s --version: %v", c.execFilePath, err)
	}
	c.versionLocal = strings.TrimSpace(string(output))
	return nil
}

func (c *Context) SelfUpgrade() error {
	out := c.originCtx.Stdout()

	latestVersionRemote, err := getLatestIact3VersionFunc()
	if err != nil {
		return err
	}
	c.versionRemote = latestVersionRemote

	if c.installed {
		if err := c.GetLocalVersion(); err != nil {
			return err
		}
		fmt.Fprintf(out, "Current installed iact3 version: %s\n", c.versionLocal)
	} else {
		fmt.Fprintln(out, "iact3 is not installed yet.")
	}
	fmt.Fprintf(out, "Latest available iact3 version: %s\n", c.versionRemote)

	if c.installed && c.versionLocal == c.versionRemote {
		fmt.Fprintln(out, "Already up to date.")
		c.writeVersionCache(c.versionLocal, c.versionRemote)
		return nil
	}

	fmt.Fprintf(out, "Installing iact3 %s ...\n", c.versionRemote)
	if err := c.Install(); err != nil {
		return err
	}
	fmt.Fprintf(out, "iact3 upgraded to %s successfully.\n", c.versionRemote)
	c.writeVersionCache(c.versionRemote, c.versionRemote)
	return nil
}

func (c *Context) PrepareEnv() error {
	profile, err := config.LoadProfileWithContext(c.originCtx)
	if err != nil {
		return fmt.Errorf("config failed: %s", err.Error())
	}

	var accessKeyId, accessKeySecret, stsToken string

	mode := profile.Mode
	switch mode {
	case config.AK:
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
	case config.StsToken:
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
		stsToken = profile.StsToken
	default:
		proxyHost, ok := c.originCtx.Flags().GetValue("proxy-host")
		if !ok {
			proxyHost = ""
		}
		credential, err := profile.GetCredential(c.originCtx, tea.String(proxyHost))
		if err != nil {
			return fmt.Errorf("can't get credential %s", err)
		}

		model, err := credential.GetCredential()
		if err != nil {
			return fmt.Errorf("can't get credential %s", err)
		}
		accessKeyId = *model.AccessKeyId
		accessKeySecret = *model.AccessKeySecret
		if model.SecurityToken != nil {
			stsToken = *model.SecurityToken
		}
	}

	envMapNew := make(map[string]string)
	// Signal to the iact3 binary that it is being driven by aliyun-cli, so
	// it can adapt UA/telemetry or behavior accordingly. Always set,
	// independent of the credential mode resolved above.
	envMapNew["ALIBABA_CLOUD_IACT3_COMPAT_MODE"] = "aliyun"
	if accessKeyId != "" {
		envMapNew["ALIBABA_CLOUD_ACCESS_KEY_ID"] = accessKeyId
	}
	if accessKeySecret != "" {
		envMapNew["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] = accessKeySecret
	}
	if profile.RegionId != "" {
		envMapNew["ALIBABA_CLOUD_REGION_ID"] = profile.RegionId
	}
	if stsToken != "" {
		envMapNew["ALIBABA_CLOUD_SECURITY_TOKEN"] = stsToken
	}
	c.envMap = envMapNew
	return nil
}

// versionCachePath returns the absolute path to the version-check cache
// file. configPath must be initialized first (via InitBasicInfo).
func (c *Context) versionCachePath() string {
	return filepath.Join(c.configPath, versionCacheFileName)
}

// writeVersionCache persists what we currently know about the local +
// remote iact3 versions, with "now" as the last-check timestamp. Errors
// are swallowed: a missing/unwritable cache only means the next run pays
// the bootstrap cost again.
func (c *Context) writeVersionCache(installed, remote string) {
	cache := iact3VersionCache{
		InstalledVersion: installed,
		LastKnownRemote:  remote,
		LastRemoteCheck:  timeNowFunc().Unix(),
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	_ = os.WriteFile(c.versionCachePath(), data, 0600)
}

func (c *Context) readVersionCache() (iact3VersionCache, bool) {
	var cache iact3VersionCache
	data, err := os.ReadFile(c.versionCachePath())
	if err != nil {
		return cache, false
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return cache, false
	}
	return cache, true
}

// maybeShowUpgradeHint emits a single-line "new version available" hint
// to stderr after the iact3 subprocess returns. The whole path is
// best-effort: any error (cache I/O, network, parsing) silently aborts
// the hint — we never want this mechanism to surface noise or alter
// command exit behavior.
//
// Behavior:
//   - No cache yet but binary is installed → bootstrap: query local
//     version via the binary itself + fetch remote once; write the cache
//     so subsequent runs are cheap.
//   - Cache present, fresh (<TTL) → use cached remote, no network.
//   - Cache present, stale → refresh remote (single GET, soft-fail) and
//     update the cache timestamp.
//   - InstalledVersion != LastKnownRemote → print the hint.
func (c *Context) maybeShowUpgradeHint() {
	if !c.installed {
		return
	}

	cache, ok := c.readVersionCache()
	if !ok {
		c.bootstrapVersionCache()
		return
	}

	now := timeNowFunc().Unix()
	if now-cache.LastRemoteCheck > int64(versionCheckTTL.Seconds()) {
		if remote, err := getLatestIact3VersionFunc(); err == nil && remote != "" {
			cache.LastKnownRemote = remote
			cache.LastRemoteCheck = now
			if data, err := json.Marshal(cache); err == nil {
				_ = os.WriteFile(c.versionCachePath(), data, 0600)
			}
		}
	}

	c.printUpgradeHint(cache.InstalledVersion, cache.LastKnownRemote)
}

// bootstrapVersionCache covers the "installed by an older aliyun-cli that
// didn't track versions" case: we have a binary but no cache record.
// Determine its version, fetch the remote, write a fresh cache entry,
// and emit a hint if they differ.
func (c *Context) bootstrapVersionCache() {
	if err := c.GetLocalVersion(); err != nil || c.versionLocal == "" {
		return
	}
	remote, err := getLatestIact3VersionFunc()
	if err != nil || remote == "" {
		return
	}
	c.writeVersionCache(c.versionLocal, remote)
	c.printUpgradeHint(c.versionLocal, remote)
}

func (c *Context) printUpgradeHint(installed, remote string) {
	if remote == "" || installed == "" || remote == installed {
		return
	}
	fmt.Fprintf(c.originCtx.Stderr(),
		"\nNote: a newer iact3 version %s is available (you have %s). Run `aliyun iact3 upgrade` to update.\n",
		remote, installed)
}

// RemoveFlagsForMainCli strips all flags registered by the parent CLI
// (config.AddFlags + openapi.AddFlags) from args before forwarding to the
// iact3 subprocess. Values for these flags are passed via environment
// variables in PrepareEnv. Handles `--name value`, `--name=value`, aliases,
// and shorthand. If a new global flag source is added to the main CLI, it
// must be registered here as well.
func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
	allFlags := cli.NewFlagSet()
	config.AddFlags(allFlags)
	openapi.AddFlags(allFlags)

	longNeedsValue := make(map[string]bool)
	shortNeedsValue := make(map[string]bool)
	for _, f := range allFlags.Flags() {
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
	return out, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func GetLatestIact3Version() (string, error) {
	url := iact3BaseUrl + "version.txt"
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %v", url, err)
	}

	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)

	resp, err := client.Do(req)
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
		return "", fmt.Errorf("failed to parse version from response body: empty string")
	}

	return version, nil
}
