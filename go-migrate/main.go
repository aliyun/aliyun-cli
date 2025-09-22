package go_migrate

import (
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
)

var cdnBaseUrl = "https://aliyuncli.alicdn.com"

//var cdnBaseUrl = "https://aliyun-cli-demo.oss-cn-beijing.aliyuncs.com"

// 为测试注入可替换的函数
var fetchRemoteContentFunc = fetchRemoteContent
var downloadGoMigrateFunc = DownloadGoMigrate

var getConfigurePathFunc = func() string {
	return config.GetConfigPath()
}

var cdnBaseUrlFunc = func() string {
	return cdnBaseUrl
}

type GoMigrateOptionsContext struct {
	originCtx *cli.Context
	// Already download
	download bool
	// Is latest version
	isLatest      bool
	configurePath string
	osType        string
	// current os arch
	osArch    string
	osSupport bool
	// latest version from remote
	latestVersionRemote string
	// local installed
	localInstalled bool
	// local version
	localVersion string
	// local exec bin absolute path
	localExecAbs string
}

// Run parse and validate args
func (c *GoMigrateOptionsContext) Run(args []string) error {
	// only parse support flags
	configPath := getConfigurePathFunc()
	c.configurePath = configPath
	// check
	err := c.checkBeforeRun()
	if err != nil {
		_, err := cli.Println(c.originCtx.Stderr(), err)
		if err != nil {
			return err
		}
	}
	// all done, start run
	if c.localInstalled {
		cmd := exec.Command(c.localExecAbs, args...)
		cmd.Stdout = c.originCtx.Stdout()
		cmd.Stderr = c.originCtx.Stderr()
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("execute go-migrate failed: %v", err)
		}
	}
	return nil
}

func NewGoMigrateOptionsContext(ctx *cli.Context) *GoMigrateOptionsContext {
	return &GoMigrateOptionsContext{
		originCtx: ctx,
	}
}

func (c *GoMigrateOptionsContext) Stdout() io.Writer {
	return c.originCtx.Stdout()
}

func (c *GoMigrateOptionsContext) Stderr() io.Writer {
	return c.originCtx.Stderr()
}

func (c *GoMigrateOptionsContext) GetOs() string {
	return runtime.GOOS
}

func (c *GoMigrateOptionsContext) GetCurrentOsArch() string {
	return runtime.GOARCH
}

func (c *GoMigrateOptionsContext) GetLatestVersion() (string, error) {
	versionCheckUrl := cdnBaseUrlFunc() + "/go-migrate/latest_version.txt"
	return fetchRemoteContentFunc(versionCheckUrl)
}

func (c *GoMigrateOptionsContext) checkBeforeRun() error {
	// check os support
	c.osType = c.GetOs()
	c.osArch = c.GetCurrentOsArch()
	if c.osType != "linux" && c.osType != "darwin" && c.osType != "windows" {
		return fmt.Errorf("os type %s not support", c.osType)
	}
	if c.osArch != "amd64" && c.osArch != "arm64" {
		return fmt.Errorf("os arch %s not support", c.osArch)
	}
	c.osSupport = true
	// check latest version
	latestVersion, err := c.GetLatestVersion()
	if err != nil {
		return fmt.Errorf("get latest version failed: %v", err)
	}
	c.latestVersionRemote = latestVersion
	// check local exec install
	c.checkLocalInstall()
	if c.localInstalled {
		// get local version
		localVersion, err := GetLocalGoMigrateVersion(c.localExecAbs)
		if err != nil {
			return fmt.Errorf("get local version failed: %v", err)
		}
		c.localVersion = localVersion
		if c.localVersion == c.latestVersionRemote {
			c.isLatest = true
			return nil
		} else {
			c.isLatest = false
			_, err := cli.Println(c.originCtx.Stdout(), fmt.Sprintf("go-migrate local version %s, latest version %s", c.localVersion, c.latestVersionRemote))
			if err != nil {
				return err
			}
			c.removeLocalFile()
			// download latest version
			err = downloadGoMigrateFunc(c.osType, c.osArch, c.latestVersionRemote, c.configurePath)
			if err != nil {
				return fmt.Errorf("download latest version failed: %v", err)
			}
			_, err = cli.Println(c.originCtx.Stdout(), fmt.Sprintf("go-migrate upgrade to latest version %s successfully", c.latestVersionRemote))
			if err != nil {
				return err
			}
			c.download = true
			c.localInstalled = true
			// 下载完成就是最新
			c.isLatest = true
		}
	} else {
		_, err = cli.Println(c.originCtx.Stdout(), "go-migrate not installed, start install latest version "+c.latestVersionRemote)
		if err != nil {
			return err
		}
		// not installed, download
		err = downloadGoMigrateFunc(c.osType, c.osArch, c.latestVersionRemote, c.configurePath)
		if err != nil {
			return fmt.Errorf("download latest version failed: %v", err)
		}
		_, err := cli.Println(c.originCtx.Stdout(), fmt.Sprintf("go-migrate version %s install successfully", c.latestVersionRemote))
		if err != nil {
			return err
		}
		c.download = true
		c.localInstalled = true
		// 下载完成就是最新
		c.isLatest = true
	}
	return err
}

func DownloadGoMigrate(osType, osArch, version, configurePath string) error {
	execName := getExecName(osType)
	downloadUrl := fmt.Sprintf("%s/go-migrate/%s/%s-%s/%s", cdnBaseUrlFunc(), version, osType, osArch, execName)
	destPath := filepath.Join(configurePath, execName)

	resp, err := http.Get(downloadUrl)
	if err != nil {
		return fmt.Errorf("failed to download %s: %v", downloadUrl, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: status code %d", downloadUrl, resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", destPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file %s: %v", destPath, err)
	}

	if osType != "windows" {
		err = os.Chmod(destPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to set executable permission for %s: %v", destPath, err)
		}
	}

	return nil
}

func GetLocalGoMigrateVersion(bin string) (string, error) {
	cmd := exec.Command(bin, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute %s: %v", bin, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (c *GoMigrateOptionsContext) checkLocalInstall() {
	execName := getExecName(c.osType)
	c.localExecAbs = filepath.Join(c.configurePath, execName)

	// 检查文件是否存在
	if _, err := os.Stat(c.localExecAbs); err == nil {
		c.localInstalled = true
	} else {
		c.localInstalled = false
	}
}

func (c *GoMigrateOptionsContext) removeLocalFile() {
	if c.localInstalled {
		err := os.Remove(c.localExecAbs)
		if err != nil {
			_, _ = cli.Println(c.originCtx.Stderr(), fmt.Sprintf("failed to remove old go-migrate file: %v", err))
		} else {
			c.localInstalled = false
		}
	}
}

func getExecName(osType string) string {
	switch osType {
	case "windows":
		return "go-migrate.exe"
	default:
		return "go-migrate"
	}
}

func fetchRemoteContent(url string) (string, error) {
	// 创建带30秒超时的HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 创建HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %v", url, err)
	}

	// 设置User-Agent
	req.Header.Set("User-Agent", "aliyun-cli/"+cli.Version)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content from %s: %v", url, err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，如果不是2xx就报错
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP request failed with status code %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from %s: %v", url, err)
	}

	return string(body), nil
}
