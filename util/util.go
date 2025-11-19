package util

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func GetFromEnv(args ...string) string {
	for _, key := range args {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}

	return ""
}

func GetCurrentUnixTime() int64 {
	return time.Now().Unix()
}

func NewHttpClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 10,
	}
}

func RandStringBytesMaskImprSrc(length int) interface{} {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)

	for i := range b {
		// 使用加密安全的随机数生成器
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			// 如果加密随机数生成失败，降级使用时间种子（但每次都重新生成种子）
			b[i] = letterBytes[(time.Now().UnixNano()+int64(i))%int64(len(letterBytes))]
		} else {
			b[i] = letterBytes[randomIndex.Int64()]
		}
	}
	return string(b)
}

func OpenBrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		fmt.Printf("%s: %s\n", i18n.T("Cannot automatically open browser, please visit manually", "无法自动打开浏览器，请手动访问").GetMessage(), url)
	}

	if err != nil {
		fmt.Printf("%s: %v\n", i18n.T("Failed to open browser", "打开浏览器失败").GetMessage(), err)
	}
	return nil
}

func UnmarshalJsonFromReader(body io.ReadCloser, s *struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}) error {
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, s)
	if err != nil {
		return err
	}
	return nil
}

// CopyFileAndRemoveSource copies a file from source to destination and removes the source file.
// This is useful when moving files across different filesystems where os.Rename might fail.
func CopyFileAndRemoveSource(sourceFile, destFile string) error {
	src, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %v", sourceFile, err)
	}

	dst, err := os.Create(destFile)
	if err != nil {
		_ = src.Close()
		return fmt.Errorf("failed to create destination file %s: %v", destFile, err)
	}

	_, err = io.Copy(dst, src)
	_ = dst.Close()
	_ = src.Close()
	if err != nil {
		_ = os.Remove(destFile)
		return fmt.Errorf("failed to copy file from %s to %s: %v", sourceFile, destFile, err)
	}
	_ = os.Remove(sourceFile)
	return nil
}

func GetAliyunCliUserAgent() string {
	ua := "Aliyun-CLI/" + cli.GetVersion()
	if vendorEnv, ok := os.LookupEnv("ALIBABA_CLOUD_VENDOR"); ok {
		ua += " vendor/" + vendorEnv
	}
	return ua
}
