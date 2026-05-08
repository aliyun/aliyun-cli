package saectl

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/aliyun/aliyun-cli/v3/util"
)

type Context struct {
	originCtx                 *cli.Context
	configPath                string // aliyun config path, all bin and cache file store in the same dir
	checkVersionCacheFilePath string // cache file path to store last check version time, unix timestamp
	execFilePath              string // saectl exec file path
	installed                 bool   // whether saectl is installed
	versionLocal              string
	versionRemote             string
	isLatest                  bool
	osType                    string
	osArch                    string
	osSupport                 bool
	downloadPathSuffix        string
	envMap                    map[string]string
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

// 可替换的函数变量, 便于单元测试 mock
var (
	getLatestSaeCtlVersionFunc = GetLatestSaeCtlVersion
	downloadAndExtractFunc     = DownloadAndExtract
	execCommandFunc            = exec.Command
	httpGetFunc                = http.Get
	timeNowFunc                = time.Now
	runtimeGOOSFunc            = func() string { return runtime.GOOS }
	runtimeGOARCHFunc          = func() string { return runtime.GOARCH }
)

// saectlBaseUrl is the base url for saectl download
var saectlBaseUrl = "https://sae-component-software.oss-cn-hangzhou.aliyuncs.com/saectl/"

var VersionCheckTTL = 86400 // 1 day, in seconds

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

	if !c.installed {
		latestVersionRemote, err := getLatestSaeCtlVersionFunc()
		if err != nil {
			return err
		}
		c.versionRemote = latestVersionRemote
		err = c.Install()
		if err != nil {
			return err
		}
		err = c.UpdateCheckCacheTime()
		if err != nil {
			return err
		}
	} else {
		needCheckVersion := c.NeedCheckVersion()
		if needCheckVersion {
			latestVersionRemote, err := getLatestSaeCtlVersionFunc()
			if err != nil {
				return err
			}
			c.versionRemote = latestVersionRemote
			err = c.GetLocalVersion()
			if err != nil {
				return err
			}
			if c.versionLocal != c.versionRemote {
				err := c.Install()
				if err != nil {
					return err
				}
			}
			err = c.UpdateCheckCacheTime()
			if err != nil {
				return err
			}
		}
	}

	err := c.PrepareEnv()
	if err != nil {
		return err
	}

	newArgs, err := c.RemoveFlagsForMainCli(args)
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
	return nil
}

func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()

	if c.osType == "linux" || c.osType == "darwin" || c.osType == "windows" {
		switch c.osType {
		case "linux":
			if c.osArch == "amd64" || c.osArch == "arm64" {
				c.osSupport = true
				c.downloadPathSuffix = c.osType + "-" + c.osArch + ".tar.gz"
			} else {
				c.osSupport = false
			}
		case "darwin":
			if c.osArch == "amd64" || c.osArch == "arm64" {
				c.osSupport = true
				c.downloadPathSuffix = "darwin-" + c.osArch + ".tar.gz"
			} else {
				c.osSupport = false
			}
		case "windows":
			if c.osArch == "amd64" || c.osArch == "arm64" {
				c.osSupport = true
				c.downloadPathSuffix = c.osType + "-" + c.osArch + ".tar.gz"
			} else {
				c.osSupport = false
			}
		}
	} else {
		c.osSupport = false
	}
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.checkVersionCacheFilePath = filepath.Join(c.configPath, ".saectl_version_check")
	c.execFilePath = filepath.Join(c.configPath, "saectl")
	
	if runtime.GOOS == "windows" {
		c.execFilePath += ".exe"
	}
	
	c.installed = fileExists(c.execFilePath)
}

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
	var lastCheckTime int64
	_, err = fmt.Sscanf(string(data), "%d", &lastCheckTime)
	if err != nil {
		return true
	}
	currentTime := timeNowFunc().Unix()
	return currentTime-lastCheckTime > int64(VersionCheckTTL)
}

func (c *Context) Install() error {
	// TODO: 占位符，待确认具体的包命名规则和下载路径
	url := fmt.Sprintf("%s%s/saectl-%s-%s", saectlBaseUrl, c.versionRemote, c.versionRemote, c.downloadPathSuffix)
	
	tmpDir := os.TempDir()
	
	ext := ".zip"
	if strings.HasSuffix(c.downloadPathSuffix, ".tar.gz") {
		ext = ".tar.gz"
	}
	tmpFile := filepath.Join(tmpDir, "saectl_download"+ext)
	
	if fileExists(tmpFile) {
		err := os.Remove(tmpFile)
		if err != nil {
			return err
		}
	}
	
	extractCenterDir := fmt.Sprintf("saectl-%s-%s", c.versionRemote, strings.TrimSuffix(c.downloadPathSuffix, ext))
	err := downloadAndExtractFunc(url, tmpFile, c.execFilePath, extractCenterDir)
	if err != nil {
		return fmt.Errorf("failed to download and extract saectl from %s: %v", url, err)
	}
	return nil
}

func DownloadAndExtract(url string, destFile string, exeFilePath string, extractCenterDir string) error {
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
	destDirUse := filepath.Join(destDir, "saectl_extract")
	if fileExists(destDirUse) {
		err := os.RemoveAll(destDirUse)
		if err != nil {
			return fmt.Errorf("failed to remove existing dir %s: %v", destDirUse, err)
		}
	}
	
	if strings.HasSuffix(destFile, ".zip") {
		err = unzip(destFile, destDirUse)
	} else if strings.HasSuffix(destFile, ".tar.gz") {
		err = extractTarGz(destFile, destDirUse)
	} else {
		return fmt.Errorf("unsupported file extension for extraction: %s", destFile)
	}
	
	if err != nil {
		return fmt.Errorf("failed to extract file %s: %v", destFile, err)
	}
	
	originExtractFile := "saectl"
	if runtime.GOOS == "windows" {
		originExtractFile += ".exe"
	}
	
	sourceFile := filepath.Join(destDirUse, extractCenterDir, originExtractFile)
	if !fileExists(sourceFile) {
		// 回退查找根目录
		sourceFile = filepath.Join(destDirUse, originExtractFile)
		if !fileExists(sourceFile) {
			return fmt.Errorf("extracted file %s not exist", sourceFile)
		}
	}
	
	if fileExists(exeFilePath) {
		err := os.Remove(exeFilePath)
		if err != nil {
			return fmt.Errorf("failed to remove existing file %s: %v", exeFilePath, err)
		}
	}
	err = util.CopyFileAndRemoveSource(sourceFile, exeFilePath)
	if err != nil {
		return err
	}
	
	if runtime.GOOS != "windows" {
		err = os.Chmod(exeFilePath, 0755)
		if err != nil {
			return fmt.Errorf("failed to set exec permission for file %s: %v", exeFilePath, err)
		}
	}
	
	err = os.Remove(destFile)
	if err != nil {
		return fmt.Errorf("failed to remove temp file %s: %v", destFile, err)
	}
	err = os.RemoveAll(destDirUse)
	if err != nil {
		return fmt.Errorf("failed to remove temp dir %s: %v", destDirUse, err)
	}
	return nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func(r *zip.ReadCloser) {
		_ = r.Close()
	}(r)

	for _, file := range r.File {
		p, _ := filepath.Abs(file.Name)
		if strings.Contains(p, "..") {
			return fmt.Errorf("invalid file path in zip: %s", file.Name)
		}
		filePath := filepath.Join(dest, file.Name)

		if file.FileInfo().IsDir() {
			err := os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
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
		
		err = rc.Close()
		if err != nil {
			return err
		}
		err = outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer func(gzr *gzip.Reader) {
		_ = gzr.Close()
	}(gzr)

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if strings.Contains(header.Name, "..") {
			return fmt.Errorf("invalid file path in tar: %s", header.Name)
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				return err
			}
			_ = outFile.Close()
		}
	}
	return nil
}

func (c *Context) GetLocalVersion() error {
	if !c.installed {
		c.versionLocal = ""
		return fmt.Errorf("saectl not installed")
	}
	cmd := execCommandFunc(c.execFilePath, "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute %s %s: %v", c.execFilePath, "version", err)
	}
	c.versionLocal = strings.TrimSpace(string(output))
	return nil
}

func (c *Context) UpdateCheckCacheTime() error {
	currentTime := timeNowFunc().Unix()
	data := fmt.Sprintf("%d", currentTime)
	err := os.WriteFile(c.checkVersionCacheFilePath, []byte(data), 0644)
	if err != nil {
		return fmt.Errorf("failed to write cache file %s: %v", c.checkVersionCacheFilePath, err)
	}
	return nil
}

func EncodeMapBase64(m map[string]any) (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("json marshal failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (c *Context) PrepareEnv() error {
	envMap := make(map[string]any)
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
	case config.RamRoleArn:
		accessKeyId = profile.AccessKeyId
		accessKeySecret = profile.AccessKeySecret
		if profile.StsToken != "" {
			stsToken = profile.StsToken
		}
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

	envMap["access_key_id"] = accessKeyId
	envMap["access_key_secret"] = accessKeySecret
	if stsToken != "" {
		envMap["sts_token"] = stsToken
	}

	if profile.RegionId != "" {
		envMap["region"] = profile.RegionId
	}
	
	configDir := config.GetConfigDir(c.originCtx)
	forceOn, forceOff := openapi.CliAIOverrides(c.originCtx.Flags())
	aimode.MergeAiModeIntoOssutilPayload(configDir, envMap, forceOn, forceOff)

	base64Result, err := EncodeMapBase64(envMap)
	if err != nil {
		return fmt.Errorf("failed to encode env map to base64: %v", err)
	}
	
	envMapNew := make(map[string]string)
	envMapNew["SAECTL_COMPAT_MODE"] = "alicli"
	envMapNew["SAECTL_CONFIG_VALUE"] = base64Result

	c.envMap = envMapNew
	return nil
}

func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
	argsNew := make([]string, 0, len(args))
	argsNew = args
	if c.originCtx.Flags() != nil && c.originCtx.Flags().Flags() != nil {
		for _, f := range c.originCtx.Flags().Flags() {
			if f.IsAssigned() && f.Category == "config" {
				for i, arg := range argsNew {
					if arg == "--"+f.Name || (f.Shorthand != 0 && arg == "-"+string(f.Shorthand)) {
						argsNew = append(argsNew[:i], argsNew[i+1:]...)
						if i < len(argsNew) {
							if f.AssignedMode != cli.AssignedNone {
								argsNew = append(argsNew[:i], argsNew[i+1:]...)
							}
						}
						break
					}
				}
			}
		}
	}
	
	argsNew = stripArgsEqual(argsNew,
		"--"+openapi.CliAIModeFlagName,
		"--"+openapi.CliNoAIModeFlagName,
	)
	
	return argsNew, nil
}

func stripArgsEqual(args []string, drops ...string) []string {
	if len(args) == 0 || len(drops) == 0 {
		return args
	}
	drop := make(map[string]struct{}, len(drops))
	for _, d := range drops {
		drop[d] = struct{}{}
	}
	out := make([]string, 0, len(args))
	for _, a := range args {
		if _, ok := drop[a]; ok {
			continue
		}
		out = append(out, a)
	}
	return out
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func GetLatestSaeCtlVersion() (string, error) {
	url := saectlBaseUrl + "version.txt"
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

	data := string(body)
	version := strings.TrimSpace(data)
	if version == "" {
		return "", fmt.Errorf("failed to parse version from response body: empty string")
	}
	
	return version, nil
}
