package sparksubmit

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

const (
	defaultToolVersion     = "1.16.0"
	defaultDownloadBaseURL = "https://aliyun-cli-pub.oss-cn-hangzhou.aliyuncs.com/cli-ext/spark-submit"

	EnvDownloadBaseURL = "ALIBABA_CLOUD_EMR_SPARK_SUBMIT_DOWNLOAD_BASE_URL"
	EnvExecPath        = "ALIBABA_CLOUD_EMR_SPARK_SUBMIT_EXEC_PATH"
	EnvNoUpdateCheck   = "ALIBABA_CLOUD_EMR_SPARK_SUBMIT_NO_UPDATE_CHECK"
	EnvWorkspaceID     = "ALIBABA_CLOUD_EMR_SERVERLESS_SPARK_WORKSPACE_ID"
	EnvEndpoint        = "ALIBABA_CLOUD_EMR_SERVERLESS_SPARK_ENDPOINT"
)

var VersionCheckTTL = 86400

type Context struct {
	originCtx        *cli.Context
	configPath       string
	installDir       string
	execFilePath     string
	confDir          string
	versionCachePath string
	versionFilePath  string
	installed        bool
	versionLocal     string
	versionRemote    string
}

var (
	getConfigurePathFunc = func() string { return config.GetConfigPath() }
	getLatestVersionFunc = GetLatestVersion
	installFromZipFunc   = installFromZip
	execCommandFunc      = exec.Command
	lookPathFunc         = exec.LookPath
	httpGetFunc          = http.Get
	timeNowFunc          = time.Now
	loadProfileFunc      = func(ctx *cli.Context) (config.Profile, error) {
		return config.LoadProfileWithContext(ctx)
	}
)

func NewContext(origin *cli.Context) *Context {
	return &Context{originCtx: origin}
}

func (c *Context) Run(args []string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("spark-submit is not supported on Windows; use Linux or macOS with Java 8+")
	}
	if err := requireJava(); err != nil {
		return err
	}

	c.initBasicInfo()
	if err := c.ensureInstalledAndUpdated(); err != nil {
		return err
	}
	if err := c.prepareConnectionProperties(); err != nil {
		return err
	}

	childArgs := c.removeFlagsForMainCli(args)
	return c.execute(childArgs)
}

func (c *Context) initBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.installDir = filepath.Join(c.configPath, "emr-serverless-spark-tool")
	c.execFilePath = filepath.Join(c.installDir, "bin", "spark-submit")
	c.confDir = filepath.Join(c.installDir, "conf")
	c.versionCachePath = filepath.Join(c.installDir, ".version_check")
	c.versionFilePath = filepath.Join(c.installDir, ".version")

	if envPath := strings.TrimSpace(os.Getenv(EnvExecPath)); envPath != "" {
		c.execFilePath = envPath
		if strings.HasSuffix(envPath, string(os.PathSeparator)+"spark-submit") {
			c.installDir = filepath.Dir(filepath.Dir(envPath))
			c.confDir = filepath.Join(c.installDir, "conf")
		}
	}
	c.installed = fileExists(c.execFilePath)
	if c.installed {
		_ = ensureToolBinExecutable(c.installDir)
	}
}

func requireJava() error {
	if _, err := lookPathFunc("java"); err != nil {
		return fmt.Errorf("java not found in PATH; EMR Serverless spark-submit requires Java 8 or later (JRE or JDK)")
	}
	return nil
}

func (c *Context) effectiveBaseURL() string {
	if u := strings.TrimSpace(os.Getenv(EnvDownloadBaseURL)); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultDownloadBaseURL
}

func (c *Context) packageZipURL(version string) string {
	return fmt.Sprintf("%s/emr-serverless-spark-tool-%s-bin.zip", c.effectiveBaseURL(), version)
}

func GetLatestVersion(baseURL string) (string, error) {
	url := strings.TrimRight(baseURL, "/") + "/version.txt"
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return defaultToolVersion, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(body))
	if v == "" {
		return defaultToolVersion, nil
	}
	return v, nil
}

func (c *Context) ensureInstalledAndUpdated() error {
	if strings.TrimSpace(os.Getenv(EnvExecPath)) != "" {
		return nil
	}
	if os.Getenv(EnvNoUpdateCheck) == "1" && c.installed {
		return nil
	}

	if !c.installed {
		latest, err := getLatestVersionFunc(c.effectiveBaseURL())
		if err != nil {
			return fmt.Errorf("resolve spark-submit tool version: %w", err)
		}
		c.versionRemote = latest
		return c.downloadAndInstall(latest)
	}

	if !c.cacheStale() {
		return nil
	}
	latest, err := getLatestVersionFunc(c.effectiveBaseURL())
	if err != nil {
		fmt.Fprintf(c.originCtx.Stderr(), "spark-submit: update check failed: %v\n", err)
		_ = c.touchCache()
		return nil
	}
	_ = c.touchCache()
	if latest == c.readLocalVersion() {
		return nil
	}
	c.versionRemote = latest
	return c.downloadAndInstall(latest)
}

func (c *Context) downloadAndInstall(version string) error {
	parent := filepath.Dir(c.installDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	zipPath := filepath.Join(parent, fmt.Sprintf(".emr-spark-dl-%d.zip", timeNowFunc().UnixNano()))
	defer os.Remove(zipPath)

	if err := httpGetToFile(c.packageZipURL(version), zipPath); err != nil {
		return fmt.Errorf("download spark-submit tool: %w", err)
	}
	if err := installFromZipFunc(zipPath, c.installDir); err != nil {
		return err
	}
	c.installed = true
	c.versionLocal = version
	return os.WriteFile(c.versionFilePath, []byte(version), 0o644)
}

func installFromZip(zipPath, installDir string) error {
	parent := filepath.Dir(installDir)
	staging := filepath.Join(parent, fmt.Sprintf(".emr-spark-staging-%d", timeNowFunc().UnixNano()))
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	if err := unzip(zipPath, staging); err != nil {
		return fmt.Errorf("unzip %s: %w", zipPath, err)
	}

	extractedRoot, err := findToolRoot(staging)
	if err != nil {
		return err
	}

	ts := timeNowFunc().UnixNano()
	var oldDir string
	if fileExists(installDir) {
		oldDir = fmt.Sprintf("%s.old.%d", installDir, ts)
		if err := os.Rename(installDir, oldDir); err != nil {
			return fmt.Errorf("rename old install: %w", err)
		}
	}
	if err := os.Rename(extractedRoot, installDir); err != nil {
		if oldDir != "" {
			_ = os.Rename(oldDir, installDir)
		}
		return fmt.Errorf("install spark-submit tool: %w", err)
	}
	if oldDir != "" {
		_ = os.RemoveAll(oldDir)
	}
	return ensureToolBinExecutable(installDir)
}

func findToolRoot(staging string) (string, error) {
	entries, err := os.ReadDir(staging)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "emr-serverless-spark-tool") {
			return filepath.Join(staging, e.Name()), nil
		}
	}
	return "", fmt.Errorf("spark-submit package missing emr-serverless-spark-tool-* directory")
}

func ensureToolBinExecutable(installDir string) error {
	binDir := filepath.Join(installDir, "bin")
	for _, name := range []string{"spark-submit", "spark-sql"} {
		path := filepath.Join(binDir, name)
		if !fileExists(path) {
			continue
		}
		if err := os.Chmod(path, 0o755); err != nil {
			return fmt.Errorf("chmod %s: %w", path, err)
		}
	}
	return nil
}

func (c *Context) prepareConnectionProperties() error {
	profile, err := loadProfileFunc(c.originCtx)
	if err != nil {
		return fmt.Errorf("load profile: %w", err)
	}

	accessKeyID, accessKeySecret, securityToken, err := extractCredentials(c.originCtx, profile)
	if err != nil {
		return err
	}
	if accessKeyID == "" || accessKeySecret == "" {
		return fmt.Errorf("access key is empty, run `aliyun configure` first")
	}

	regionID := profile.RegionId
	if regionID == "" {
		regionID = flagStringValue(c.originCtx, config.RegionFlagName)
	}
	if regionID == "" {
		regionID = flagStringValue(c.originCtx, config.RegionIdFlagName)
	}
	if regionID == "" {
		return fmt.Errorf("region is empty, use --region or `aliyun configure`")
	}

	endpoint := strings.TrimSpace(os.Getenv(EnvEndpoint))
	if endpoint == "" {
		endpoint = fmt.Sprintf("emr-serverless-spark.%s.aliyuncs.com", regionID)
	}

	workspaceID := strings.TrimSpace(os.Getenv(EnvWorkspaceID))
	confPath := filepath.Join(c.confDir, "connection.properties")
	if fileExists(confPath) {
		existing, err := readPropertiesFile(confPath)
		if err != nil {
			return err
		}
		if workspaceID == "" {
			workspaceID = existing["workspaceId"]
		}
		if v := existing["endpoint"]; endpoint == fmt.Sprintf("emr-serverless-spark.%s.aliyuncs.com", regionID) && v != "" {
			endpoint = v
		}
	}

	props := map[string]string{
		"accessKeyId":     accessKeyID,
		"accessKeySecret": accessKeySecret,
		"regionId":        regionID,
		"endpoint":        endpoint,
	}
	if securityToken != "" {
		props["securityToken"] = securityToken
	}
	if workspaceID != "" {
		props["workspaceId"] = workspaceID
	}

	if err := os.MkdirAll(c.confDir, 0o755); err != nil {
		return err
	}
	return writePropertiesFile(confPath, props, fileExists(confPath))
}

func extractCredentials(ctx *cli.Context, p config.Profile) (string, string, string, error) {
	switch p.Mode {
	case config.AK:
		return p.AccessKeyId, p.AccessKeySecret, "", nil
	case config.StsToken:
		return p.AccessKeyId, p.AccessKeySecret, p.StsToken, nil
	case config.Anonymous:
		return "", "", "", fmt.Errorf("Anonymous profile mode is not supported by spark-submit; configure AK/STS or another credential mode")
	case config.BearerToken:
		return "", "", "", fmt.Errorf("BearerToken profile mode is not supported by spark-submit; configure AK/STS or another credential mode")
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

var passthroughFlags = map[string]struct{}{
	"output":    {},
	"force":     {},
	"version":   {},
	"body":      {},
	"body-file": {},
}

func (c *Context) removeFlagsForMainCli(args []string) []string {
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
			if _, keep := passthroughFlags[f.Name]; keep {
				out = append(out, a)
				continue
			}
			if f.AssignedMode != cli.AssignedNone && !hasInlineValue && i+1 < len(args) {
				i++
			}
			continue
		}
		out = append(out, a)
	}
	return out
}

func (c *Context) execute(childArgs []string) error {
	cmd := execCommandFunc(c.execFilePath, childArgs...)
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin

	env := os.Environ()
	if os.Getenv("SPARK_CONF_DIR") == "" && c.confDir != "" {
		env = append(env, "SPARK_CONF_DIR="+c.confDir)
	}
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return &ExitError{Code: ee.ExitCode()}
		}
		return fmt.Errorf("execute %s %v: %w", c.execFilePath, childArgs, err)
	}
	return nil
}

type ExitError struct{ Code int }

func (e *ExitError) Error() string { return fmt.Sprintf("subprocess exited with code %d", e.Code) }
func (e *ExitError) ExitCode() int { return e.Code }

func (c *Context) cacheStale() bool {
	if !c.installed {
		return false
	}
	if !fileExists(c.versionCachePath) {
		return true
	}
	data, err := os.ReadFile(c.versionCachePath)
	if err != nil {
		return true
	}
	var last int64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &last); err != nil {
		return true
	}
	return timeNowFunc().Unix()-last > int64(VersionCheckTTL)
}

func (c *Context) touchCache() error {
	return os.WriteFile(c.versionCachePath, []byte(fmt.Sprintf("%d", timeNowFunc().Unix())), 0o644)
}

func (c *Context) readLocalVersion() string {
	b, err := os.ReadFile(c.versionFilePath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
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
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	for _, file := range r.File {
		p, _ := filepath.Abs(file.Name)
		if strings.Contains(p, "..") {
			return fmt.Errorf("invalid file path in zip: %s", file.Name)
		}
		filePath := filepath.Join(dest, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		outFile, err := os.Create(filePath)
		if err != nil {
			_ = rc.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		_ = rc.Close()
		_ = outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func readPropertiesFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	props := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		props[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return props, scanner.Err()
}

func writePropertiesFile(path string, updates map[string]string, mergeExisting bool) error {
	props := make(map[string]string)
	if mergeExisting {
		existing, err := readPropertiesFile(path)
		if err == nil {
			for k, v := range existing {
				props[k] = v
			}
		}
	}
	for k, v := range updates {
		if v != "" {
			props[k] = v
		}
	}

	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(props[k])
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
