package cloudsso

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/util"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"
)

const ClientId = "app-vaz16tltdxs96audqf35"

type SsoLogin struct {
	SignInUrl    string       `json:"signInUrl"`
	AccessToken  string       `json:"accessToken"`
	AccountId    string       `json:"accountId"`
	AccessConfig string       `json:"accessConfig"`
	ExpireTime   int64        `json:"expireTime"` // accessToken expire time, unix format
	HttpClient   *http.Client `json:"-"`
	CodeVerifier string       `json:"-"` // 用于PKCE认证流程的代码验证器
	ClientId     string       `json:"-"` // 客户端ID
}

func getClientId() string {
	// if env exist ALIBABACLOUD_SSO_CLIENT_ID, use ALIBABACLOUD_SSO_CLIENT_ID
	clientId := util.GetFromEnv("ALIBABACLOUD_SSO_CLIENT_ID", "ALIBABA_CLOUD_SSO_CLIENT_ID")
	if clientId != "" {
		return clientId
	}
	return ClientId
}

// JudgeLogin CreateCloudCredential Confirm whether CloudSSO login is required
func (receiver SsoLogin) JudgeLogin() (bool, error) {
	// signInUrl is required
	if receiver.SignInUrl == "" {
		return false, errors.New("signInUrl is required")
	}
	// if accessToken/accessConfig/accountId is empty, login is required
	if receiver.AccessToken == "" {
		return true, nil
	}
	// if accessToken is not empty, check if it is expired
	if receiver.ExpireTime == 0 || receiver.ExpireTime < util.GetCurrentUnixTime() {
		return true, nil
	}
	return false, nil
}

func (receiver SsoLogin) GetAccessToken() (*AccessTokenResponse, error) {

	// 解析登录URL
	parsedURL, err := url.Parse(receiver.SignInUrl)
	if err != nil {
		fmt.Printf("%s: %v\n", i18n.T("Parse sign in url error", "解析登录地址错误").GetMessage(), err)
		return nil, err
	}

	// 创建设备授权请求
	deviceAuthResult, err := startDeviceAuthorization(parsedURL)
	if err != nil {
		fmt.Printf("%s: %v\n", i18n.T("Start Device Authorization error", "启动设备授权失败").GetMessage(), err)
		return nil, err
	}

	// 打开浏览器
	openBrowser(deviceAuthResult.VerificationUriComplete)

	// 显示登录信息
	fmt.Println(i18n.T("If the browser does not open automatically, use the following URL to complete the login process:",
		"如果浏览器没有自动打开，请使用以下URL完成登录过程:").GetMessage())
	fmt.Println()
	fmt.Printf("%s: %s\n", i18n.T("SignIn url", "登录URL").GetMessage(), deviceAuthResult.VerificationUri)
	fmt.Printf("%s: %s\n", i18n.T("User code", "用户码").GetMessage(), deviceAuthResult.UserCode)
	fmt.Println()
	fmt.Println(i18n.T("Now you can login to your account with SSO configuration in the browser.", "现在您可以在浏览器中使用SSO配置登录您的账户。").GetMessage())

	// 等待用户完成登录
	deviceCode := deviceAuthResult.DeviceCode
	var accessTokenResponse *AccessTokenResponse

	var maxRetryTime = 100
	var count = 0
	for {
		if count > maxRetryTime {
			fmt.Println(i18n.T("Login timeout, please try again.", "登录超时，请重新登录。").GetMessage())
			os.Exit(-1)
		}
		response, err := createAccessToken(receiver.SignInUrl, deviceCode)
		count++
		if err == nil {
			accessTokenResponse = response
			break
		}

		if err.Error() == "AuthorizationPending" {
			time.Sleep(time.Duration(deviceAuthResult.Interval) * time.Second)
			continue
		}

		if err.Error() == "InvalidDeviceCodeError" {
			fmt.Println(i18n.T("Your request has expired, please login again.", "您的请求已过期，请重新登录。").GetMessage())
			os.Exit(-1)
		}

		fmt.Printf("%s: %v\n", i18n.T("Error occurred while obtaining access token", "获取访问令牌时发生错误").GetMessage(), err)
	}

	fmt.Println(i18n.T("You have successfully logged in.", "您已成功登录。").GetMessage())

	// 设置访问令牌和过期时间（提前10秒过期）
	receiver.AccessToken = accessTokenResponse.AccessToken
	receiver.ExpireTime = util.GetCurrentUnixTime() + int64(accessTokenResponse.ExpiresIn) - 10

	return accessTokenResponse, nil
}

// DeviceAuthorizationResponse 设备授权响应
type DeviceAuthorizationResponse struct {
	DeviceCode              string `json:"DeviceCode"`
	UserCode                string `json:"UserCode"`
	VerificationUri         string `json:"VerificationUri"`
	VerificationUriComplete string `json:"VerificationUriComplete"`
	Interval                int    `json:"Interval"`
}

// AccessTokenResponse 访问令牌响应
type AccessTokenResponse struct {
	AccessToken string `json:"AccessToken"`
	ExpiresIn   int    `json:"ExpiresIn"`
}

// transformCodeVerifier 使用S256方法将代码验证器转换为代码挑战
func transformCodeVerifier(codeVerifier string) string {
	// 使用SHA-256哈希算法
	hash := sha256.Sum256([]byte(codeVerifier))
	// Base64-URL编码，不需要填充
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// getCodeVerifier 生成代码验证器
func getCodeVerifier() string {
	// 生成随机字符串(nonce)
	nonce := generateNonce()

	// 获取主机名
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}
	uptime := float64(time.Now().Unix())
	processUptime := time.Since(processStartTime).Seconds()

	// 组合数据并计算MD5
	data := hostname + fmt.Sprintf("%f", uptime) + fmt.Sprintf("%f", processUptime)
	md5Hash := calculateMD5(data)

	// 返回随机字符串加MD5哈希值
	return nonce + md5Hash
}

// 程序启动时间，用于计算进程运行时间
var processStartTime = time.Now()
var codeVerifier = getCodeVerifier()

// generateNonce 生成随机字符串
func generateNonce() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 16)

	// 使用加密安全的随机数生成器
	_, err := rand.Read(result)
	if err != nil {
		// 如果随机生成失败，使用时间戳作为后备方案
		for i := range result {
			result[i] = chars[int(time.Now().UnixNano()%int64(len(chars)))]
		}
	} else {
		// 将随机字节映射到字符集
		for i := range result {
			result[i] = chars[int(result[i])%len(chars)]
		}
	}

	return string(result)
}

// calculateMD5 计算字符串的MD5哈希值，返回十六进制表示
func calculateMD5(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// startDeviceAuthorization 启动设备授权流程
func startDeviceAuthorization(signInUrl *url.URL) (*DeviceAuthorizationResponse, error) {
	// 构建请求URL
	requestURL := fmt.Sprintf("%s://%s/device-authorization", signInUrl.Scheme, signInUrl.Host)

	// 生成代码验证器和获取客户端ID
	clientId := getClientId()

	// 创建请求数据
	requestData := map[string]string{
		"PortalUrl":           signInUrl.String(),
		"CodeChallenge":       transformCodeVerifier(codeVerifier),
		"ClientId":            clientId,
		"CodeChallengeMethod": "S256",
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", i18n.T("Error serializing request data", "序列化请求数据错误").GetMessage(), err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", i18n.T("Error creating HTTP request", "创建HTTP请求错误").GetMessage(), err)
	}

	// 设置请求头
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", i18n.T("Error sending HTTP request", "发送HTTP请求错误").GetMessage(), err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", i18n.T("Error reading response", "读取响应错误").GetMessage(), err)
	}

	// 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 解析错误响应
		var errResp struct {
			ErrorCode    string `json:"ErrorCode"`
			ErrorMessage string `json:"ErrorMessage"`
			RequestId    string `json:"RequestId"`
		}

		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("%s: %s", i18n.T("HTTP error", "HTTP错误").GetMessage(), string(body))
		}

		return nil, fmt.Errorf("%s: %s %s", errResp.ErrorCode, errResp.ErrorMessage, errResp.RequestId)
	}

	// 解析响应数据
	var deviceAuthResponse DeviceAuthorizationResponse
	if err := json.Unmarshal(body, &deviceAuthResponse); err != nil {
		return nil, fmt.Errorf("%s: %v", i18n.T("Error parsing response data", "解析响应数据错误").GetMessage(), err)
	}

	return &deviceAuthResponse, nil
}

// createAccessToken 创建访问令牌
func createAccessToken(parsedSignInUrl string, deviceCode string) (*AccessTokenResponse, error) {
	// 构造请求URL - 需要从SignInUrl中获取基础URL
	parsedURL, err := url.Parse(parsedSignInUrl)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", i18n.T("Error parsing login URL", "解析登录URL错误").GetMessage(), err)
	}
	requestURL := fmt.Sprintf("%s://%s/token", parsedURL.Scheme, parsedURL.Host)

	// 准备请求数据
	requestData := map[string]string{
		"CodeVerifier": codeVerifier,  // 使用生成的代码验证器
		"ClientId":     getClientId(), // 使用获取的客户端ID
		"DeviceCode":   deviceCode,
		"GrantType":    "urn:ietf:params:oauth:grant-type:device_code",
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", i18n.T("Error serializing request data", "序列化请求数据错误").GetMessage(), err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", i18n.T("Error creating HTTP request", "创建HTTP请求错误").GetMessage(), err)
	}

	// 设置请求头
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", i18n.T("Error sending HTTP request", "发送HTTP请求错误").GetMessage(), err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", i18n.T("Error reading response", "读取响应错误").GetMessage(), err)
	}

	// 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 解析错误响应
		var errResp struct {
			ErrorCode    string `json:"ErrorCode"`
			ErrorMessage string `json:"ErrorMessage"`
			RequestId    string `json:"RequestId"`
		}

		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("%s: %s", i18n.T("HTTP error", "HTTP错误").GetMessage(), string(body))
		}

		// 返回特定错误类型，便于上层函数处理不同情况
		return nil, errors.New(errResp.ErrorCode)
	}

	// 解析响应数据
	var tokenResponse AccessTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("%s: %v", i18n.T("Error parsing response data", "解析响应数据错误").GetMessage(), err)
	}

	return &tokenResponse, nil
}

// openBrowser 打开浏览器
func openBrowser(url string) {
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
		return
	}

	if err != nil {
		fmt.Printf("%s: %v\n", i18n.T("Failed to open browser", "打开浏览器失败").GetMessage(), err)
	}
}
