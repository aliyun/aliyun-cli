package codeup

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

type Context struct {
	originCtx          *cli.Context
	configPath         string
	execFilePath       string
	installed          bool
	osType             string
	osArch             string
	osSupport          bool
	downloadPathSuffix string
}

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

var (
	downloadAndExtractFunc = DownloadAndExtract
	execCommandFunc        = exec.Command
	runtimeGOOSFunc        = func() string { return runtime.GOOS }
	runtimeGOARCHFunc      = func() string { return runtime.GOARCH }
)

const (
	codeupCliBaseUrl   = "https://codeuputil.oss-cn-hangzhou.aliyuncs.com/current/"
	maxDownloadSize    = 100 * 1024 * 1024 // 100MB
	maxExtractFileSize = 200 * 1024 * 1024 // 200MB
)

var httpClient = &http.Client{Timeout: 5 * time.Minute}

var platformArchMap = map[string]string{
	"darwin-amd64":  "macOS-64",
	"darwin-arm64":  "macOS-arm64",
	"linux-386":     "Linux-32",
	"linux-amd64":   "Linux-64",
	"linux-arm64":   "Linux-64-arm64",
	"windows-386":   "Windows-32",
	"windows-amd64": "Windows-64",
}

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
		err := c.Install()
		if err != nil {
			return err
		}
	}

	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		return err
	}

	return c.ExecuteCodeupCli(newArgs)
}

func (c *Context) InitBasicInfo() {
	c.configPath = getConfigurePathFunc()
	c.execFilePath = filepath.Join(c.configPath, "codeup-cli")
	if runtimeGOOSFunc() == "windows" {
		c.execFilePath += ".exe"
	}
	c.installed = fileExists(c.execFilePath)
}

func (c *Context) CheckOsTypeAndArch() {
	c.osType = runtimeGOOSFunc()
	c.osArch = runtimeGOARCHFunc()

	key := c.osType + "-" + c.osArch
	suffix, ok := platformArchMap[key]
	if !ok {
		c.osSupport = false
		return
	}

	c.osSupport = true
	c.downloadPathSuffix = fmt.Sprintf("codeup-cli-%s.zip", suffix)
}

func (c *Context) Install() error {
	url := codeupCliBaseUrl + c.downloadPathSuffix

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "codeup-cli_download.zip")

	if fileExists(tmpFile) {
		_ = os.Remove(tmpFile)
	}

	err := downloadAndExtractFunc(url, tmpFile, c.execFilePath)
	if err != nil {
		return fmt.Errorf("failed to download and extract codeup-cli from %s: %v", url, err)
	}
	c.installed = true
	return nil
}

func DownloadAndExtract(url string, destFile string, exeFilePath string) error {
	req, err := http.NewRequest("GET", url, nil)
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

	out, err := os.Create(destFile)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", destFile, err)
	}

	written, err := io.Copy(out, io.LimitReader(resp.Body, int64(maxDownloadSize)+1))
	if err != nil {
		_ = out.Close()
		_ = os.Remove(destFile)
		return fmt.Errorf("failed to write to file %s: %v", destFile, err)
	}
	if written > int64(maxDownloadSize) {
		_ = out.Close()
		_ = os.Remove(destFile)
		return fmt.Errorf("download size exceeds limit of %d bytes", maxDownloadSize)
	}
	if err = out.Close(); err != nil {
		_ = os.Remove(destFile)
		return fmt.Errorf("failed to close file %s: %v", destFile, err)
	}

	err = extractZip(destFile, exeFilePath)
	if err != nil {
		_ = os.Remove(destFile)
		return err
	}

	_ = os.Remove(destFile)
	return nil
}

func extractZip(zipPath string, exeFilePath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip %s: %v", zipPath, err)
	}
	defer r.Close()

	targetName := "codeup-cli"
	if runtimeGOOSFunc() == "windows" {
		targetName = "codeup-cli.exe"
	}

	for _, file := range r.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Mode()&os.ModeSymlink != 0 {
			continue
		}

		baseName := filepath.Base(file.Name)
		if baseName != targetName {
			continue
		}

		if file.UncompressedSize64 > uint64(maxExtractFileSize) {
			return fmt.Errorf("file size exceeds limit of %d bytes", maxExtractFileSize)
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %v", err)
		}

		if fileExists(exeFilePath) {
			_ = os.Remove(exeFilePath)
		}

		outFile, err := os.OpenFile(exeFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			_ = rc.Close()
			return fmt.Errorf("failed to create file %s: %v", exeFilePath, err)
		}

		_, err = io.Copy(outFile, io.LimitReader(rc, int64(maxExtractFileSize)+1))
		_ = rc.Close()
		if err != nil {
			_ = outFile.Close()
			_ = os.Remove(exeFilePath)
			return fmt.Errorf("failed to extract file: %v", err)
		}
		if err = outFile.Close(); err != nil {
			_ = os.Remove(exeFilePath)
			return fmt.Errorf("failed to close file %s: %v", exeFilePath, err)
		}

		return nil
	}

	return fmt.Errorf("codeup-cli binary not found in zip archive")
}

var stripFlags = map[string]bool{
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
	"config-path":                    true,
	"read-timeout":                   true,
	"connect-timeout":                true,
	"retry-count":                    true,
	"skip-secure-verify":             true,
	"endpoint-type":                  true,
	"RegionId":                       true,
	"secure":                         true,
	"insecure":                       true,
	"header":                         true,
	"pager":                          true,
	"accept":                         true,
	"waiter":                         true,
	"dryrun":                         true,
	"quiet":                          true,
	"yes":                            true,
	"cli-query":                      true,
	"roa":                            true,
	"method":                         true,
	"user-agent":                     true,
	"cli-ai-mode":                    true,
	"no-cli-ai-mode":                 true,
}

func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
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
	return out, nil
}

func (c *Context) ExecuteCodeupCli(args []string) error {
	cmd := execCommandFunc(c.execFilePath, args...)
	cmd.Stdout = c.originCtx.Stdout()
	cmd.Stderr = c.originCtx.Stderr()
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %s: %v", c.execFilePath, err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
