// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mcpproxy

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/util"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
)

// OAuthPermanentError 表示不可重试的 OAuth 错误（如 refresh token 失效、应用被删除等）
type OAuthPermanentError struct {
	StatusCode int
	ErrorCode  string
	Message    string
	Body       string
}

func (e *OAuthPermanentError) Error() string {
	return e.Message
}

// IsPermanentError 检查是否是永久性错误
func IsPermanentError(err error) bool {
	_, ok := err.(*OAuthPermanentError)
	return ok
}

type RegionType string

const (
	RegionCN              RegionType = "CN"
	RegionINTL            RegionType = "INTL"
	DefaultMcpProfileName            = "default-mcp"
	MCPOAuthAppName                  = "aliyun-cli-mcp-proxy"
	MCPOAuthDisplayName              = "AliyunCLI-MCP-Proxy"
)

type EndpointConfig struct {
	SignIn string
	OAuth  string
	IMS    string
	MCP    string
}

// 国内/国际站端点映射
var EndpointMap = map[RegionType]EndpointConfig{
	RegionCN: {
		SignIn: "https://signin.aliyun.com",
		OAuth:  "https://oauth.aliyun.com",
		IMS:    "ims.aliyuncs.com",
		MCP:    "openapi-mcp.cn-hangzhou.aliyuncs.com",
	},
	RegionINTL: {
		SignIn: "https://signin.alibabacloud.com",
		OAuth:  "https://oauth.alibabacloud.com",
		IMS:    "ims.aliyuncs.com",
		MCP:    "openapi-mcp.ap-southeast-1.aliyuncs.com",
	},
}

const (
	OAuthTimeout            = 5 * time.Minute
	AccessTokenValiditySec  = 10800    // 3 hours
	RefreshTokenValiditySec = 31536000 // 365 days (1 year)

	AliyunCLIHomepageURL     = "https://help.aliyun.com/zh/cli/what-is-alibaba-cloud-cli"
	RedirectCountdownSeconds = 10   // 自动跳转倒计时（秒）
	ManualModeCloseDelayMs   = 3000 // 手动模式自动关闭延迟（毫秒）
)

var oauthTimeout = OAuthTimeout

const (
	oauthErrorPageHTML = `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>OAuth Authorization Error</title>
	<style>
		body { font-family: Arial, sans-serif; text-align: center; padding: 50px; }
		.error { color: #d32f2f; }
	</style>
</head>
<body>
	<h1 class="error">Authorization Error</h1>
	<p>No authorization code received. Please try again.</p>
</body>
</html>`
	oauthSuccessPageManualHTML = `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>OAuth Authorization Success</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			text-align: center;
			padding: 50px;
			background-color: #f5f5f5;
		}
		.container {
			background-color: white;
			border-radius: 8px;
			padding: 30px;
			max-width: 600px;
			margin: 0 auto;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
		}
		.success {
			color: #2e7d32;
			font-size: 24px;
			margin-bottom: 20px;
		}
		.code-container {
			background-color: #f5f5f5;
			border: 2px solid #e0e0e0;
			border-radius: 4px;
			padding: 15px;
			margin: 20px 0;
			word-break: break-all;
		}
		.code {
			font-family: 'Courier New', monospace;
			font-size: 16px;
			color: #1976d2;
			font-weight: bold;
		}
		.copy-btn {
			background-color: #1976d2;
			color: white;
			border: none;
			padding: 10px 20px;
			border-radius: 4px;
			cursor: pointer;
			font-size: 14px;
			margin-top: 10px;
		}
		.copy-btn:hover {
			background-color: #1565c0;
		}
		.hint {
			color: #666;
			font-size: 14px;
			margin-top: 20px;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1 class="success">✓ Authorization Successful</h1>
		<p>Your authorization code is:</p>
		<div class="code-container">
			<code class="code" id="authCode">{{.Code}}</code>
		</div>
		<button class="copy-btn" onclick="copyCode()">Copy Code</button>
		<p class="hint">You can close this window now.</p>
	</div>
	<script>
		function copyCode() {
			const code = document.getElementById('authCode').textContent;
			navigator.clipboard.writeText(code).then(function() {
				const btn = document.querySelector('.copy-btn');
				const originalText = btn.textContent;
				btn.textContent = 'Copied!';
				btn.style.backgroundColor = '#2e7d32';
				setTimeout(function() {
					btn.textContent = originalText;
					btn.style.backgroundColor = '#1976d2';
				}, 2000);
			}).catch(function(err) {
				alert('Failed to copy: ' + err);
			});
		}
		// 自动关闭窗口（如果是在弹出窗口中打开的）
		setTimeout(function() {
			if (window.opener) {
				window.close();
			}
		}, {{.CloseDelayMs}});
	</script>
</body>
</html>`
	oauthSuccessPageAutoHTML = `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>OAuth Authorization Success</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			text-align: center;
			padding: 50px;
			background-color: #f5f5f5;
		}
		.container {
			background-color: white;
			border-radius: 8px;
			padding: 30px;
			max-width: 600px;
			margin: 0 auto;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
		}
		.success {
			color: #2e7d32;
			font-size: 24px;
			margin-bottom: 20px;
		}
		.hint {
			color: #666;
			font-size: 14px;
			margin-top: 20px;
		}
		.countdown {
			color: #1976d2;
			font-weight: bold;
			font-size: 16px;
		}
		.link {
			color: #1976d2;
			text-decoration: none;
		}
		.link:hover {
			text-decoration: underline;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1 class="success">✓ Authorization Successful</h1>
		<p>Redirecting to Aliyun CLI homepage in <span class="countdown" id="countdown">{{.Countdown}}</span> seconds...</p>
		<p class="hint">Or <a href="{{.HomepageURL}}" class="link" id="homepageLink" target="_blank">click here</a> to visit now.</p>
	</div>
	<script>
		var countdown = {{.Countdown}};
		var countdownEl = document.getElementById('countdown');
		var homepageLink = document.getElementById('homepageLink');
		var homepageURL = {{.HomepageURL | js}};
		var isPopup = window.opener != null;
		
		// 倒计时
		var timer = setInterval(function() {
			countdown--;
			if (countdown > 0) {
				countdownEl.textContent = countdown;
			} else {
				clearInterval(timer);
				countdownEl.textContent = '0';
				
				// 如果是弹出窗口，尝试关闭；否则跳转到主页
				if (isPopup) {
					window.close();
				} else {
					window.location.href = homepageURL;
				}
			}
		}, 1000);
	</script>
</body>
</html>`
)

var (
	oauthErrorPageTemplate         *template.Template
	oauthSuccessPageManualTemplate *template.Template
	oauthSuccessPageAutoTemplate   *template.Template
)

func init() {
	var err error

	oauthErrorPageTemplate, err = template.New("error").Parse(oauthErrorPageHTML)
	if err != nil {
		panic(fmt.Sprintf("failed to parse oauth error page template: %v", err))
	}

	oauthSuccessPageManualTemplate, err = template.New("successManual").Parse(oauthSuccessPageManualHTML)
	if err != nil {
		panic(fmt.Sprintf("failed to parse oauth success manual page template: %v", err))
	}

	oauthSuccessPageAutoTemplate, err = template.New("successAuto").Funcs(template.FuncMap{
		"js": func(s string) template.JS {
			return template.JS(fmt.Sprintf("%q", s))
		},
	}).Parse(oauthSuccessPageAutoHTML)
	if err != nil {
		panic(fmt.Sprintf("failed to parse oauth success auto page template: %v", err))
	}
}

type OAuthPageData struct {
	Code         string
	Countdown    int
	HomepageURL  string
	CloseDelayMs int
}

type OAuthCallbackManager struct {
	mu          sync.RWMutex
	pendingAuth chan string // 用于传递授权码
	errorCh     chan error  // 用于传递错误
	isWaiting   bool        // 是否正在等待回调
}

type OAuthTokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int64  `json:"expires_in"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func NewOAuthCallbackManager() *OAuthCallbackManager {
	return &OAuthCallbackManager{
		pendingAuth: make(chan string, 1),
		errorCh:     make(chan error, 1),
		isWaiting:   false,
	}
}

func (m *OAuthCallbackManager) StartWaiting() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isWaiting = true
	select {
	case <-m.pendingAuth:
	default:
	}
	select {
	case <-m.errorCh:
	default:
	}
}

func (m *OAuthCallbackManager) StopWaiting() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isWaiting = false
}

func (m *OAuthCallbackManager) IsWaiting() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isWaiting
}

func (m *OAuthCallbackManager) HandleCallback(code string, err error) bool {
	if !m.IsWaiting() {
		return false
	}

	if err != nil {
		select {
		case m.errorCh <- err:
		default:
		}
		return true
	}

	if code != "" {
		select {
		case m.pendingAuth <- code:
		default:
		}
		return true
	}

	return false
}

func (m *OAuthCallbackManager) WaitForCode() (string, error) {
	select {
	case code := <-m.pendingAuth:
		return code, nil
	case err := <-m.errorCh:
		return "", err
	case <-time.After(oauthTimeout):
		return "", fmt.Errorf("timeout waiting for authorization")
	}
}

func handleOAuthCallbackRequest(w http.ResponseWriter, r *http.Request, handler func(string, error) bool, showCode bool) bool {
	if r.URL.Path != "/callback" {
		return false
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		handler("", fmt.Errorf("no authorization code received"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		if err := oauthErrorPageTemplate.Execute(w, nil); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return true
	}

	handler(code, nil)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if showCode {
		data := OAuthPageData{
			Code:         code,
			CloseDelayMs: ManualModeCloseDelayMs,
		}
		if err := oauthSuccessPageManualTemplate.Execute(w, data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	} else {
		data := OAuthPageData{
			Countdown:   RedirectCountdownSeconds,
			HomepageURL: AliyunCLIHomepageURL,
		}
		if err := oauthSuccessPageAutoTemplate.Execute(w, data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
	return true
}

func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func oauthRefresh(endpoint string, data url.Values) (*OAuthTokenResponse, error) {
	fullURL := endpoint + "/v1/token"
	log.Printf("OAuth refresh: attempting to refresh token at %s", fullURL)

	req, err := http.NewRequest("POST", fullURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	startTime := time.Now()
	resp, err := util.NewHttpClient().Do(req)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("OAuth refresh: HTTP request to %s failed after %v: %v", fullURL, duration, err)
		return nil, fmt.Errorf("http request to %s failed after %v: %w", fullURL, duration, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("OAuth refresh: received non-OK status %d after %v, body: %s", resp.StatusCode, duration, string(body))

		// 检查是否是 OAuth 认证错误（不可重试的错误）
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
			var errorResp map[string]interface{}
			if json.Unmarshal(body, &errorResp) == nil {
				if errorCode, ok := errorResp["error"].(string); ok {
					// 永久性错误，不应该重试
					if errorCode == "invalid_grant" || errorCode == "invalid_client" ||
						errorCode == "unauthorized_client" || errorCode == "invalid_token" {
						return nil, &OAuthPermanentError{
							StatusCode: resp.StatusCode,
							ErrorCode:  errorCode,
							Message:    fmt.Sprintf("OAuth permanent error: %s (status %d)", errorCode, resp.StatusCode),
							Body:       string(body),
						}
					}
				}
			}
		}

		return nil, fmt.Errorf("refresh failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("oauth error: %s: %s", tokenResp.Error, tokenResp.ErrorDescription)
	}

	log.Printf("OAuth refresh: token refreshed successfully in %v", duration)
	return &tokenResp, nil
}

type OAuthTokenResult struct {
	AccessToken       string
	RefreshToken      string
	AccessTokenExpire int64
}

func exchangeCodeForTokenWithPKCE(clientId, code, codeVerifier, redirectURI, oauthEndpoint string) (*OAuthTokenResult, error) {
	log.Println("Start to exchange code for token with PKCE")
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", clientId)
	data.Set("redirect_uri", redirectURI)
	data.Set("code_verifier", codeVerifier)

	tokenResp, err := oauthRefresh(oauthEndpoint, data)
	if err != nil {
		log.Println("Exchange code for token with PKCE failed:", err)
		return nil, fmt.Errorf("oauth refresh failed: %w", err)
	}
	log.Println("Exchange code for token with PKCE successfully")

	log.Printf("OAuth token response for authorization code exchange: AccessToken length=%d, RefreshToken length=%d, ExpiresIn=%d",
		len(tokenResp.AccessToken), len(tokenResp.RefreshToken), tokenResp.ExpiresIn)
	if tokenResp.RefreshToken == "" {
		log.Printf("WARNING: OAuth response has empty RefreshToken! TokenType=%s, Error=%s",
			tokenResp.TokenType, tokenResp.Error)
	}

	currentTime := util.GetCurrentUnixTime()
	tokenResult := &OAuthTokenResult{
		AccessToken:       tokenResp.AccessToken,
		RefreshToken:      tokenResp.RefreshToken,
		AccessTokenExpire: currentTime + tokenResp.ExpiresIn,
	}

	return tokenResult, nil
}

func buildOAuthURL(clientId string, region RegionType, host string, port int, codeChallenge string, scope string) string {
	redirectURI := buildRedirectUri(host, port)
	return fmt.Sprintf("%s/oauth2/v1/auth?client_id=%s&response_type=code&scope=%s&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256",
		EndpointMap[region].SignIn, clientId, url.QueryEscape(scope), redirectURI, codeChallenge)
}

func executeOAuthFlow(ctx *cli.Context, clientId string, regionType RegionType, manager *OAuthCallbackManager,
	host string, port int, autoOpenBrowser bool, scope string, logAuthURL func(string)) (*OAuthTokenResult, error) {
	stderr := getStderrWriter(ctx)
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	redirectURI := buildRedirectUri(host, port)
	authURL := buildOAuthURL(clientId, regionType, host, port, codeChallenge, scope)

	if logAuthURL != nil {
		logAuthURL(authURL)
	}
	waitStarted := false
	stopWaiting := func() {
		if waitStarted {
			manager.StopWaiting()
			waitStarted = false
		}
	}
	defer stopWaiting()

	var code string

	if autoOpenBrowser {
		if err := OpenBrowser(authURL); err != nil {
			// 错误信息输出到 stderr，确保用户能看到
			cli.Printf(stderr, "Failed to open browser automatically: %v\n", err)
			cli.Printf(stderr, "Falling back to manual code input mode...\n")
			if !isInteractiveInput() {
				return nil, fmt.Errorf("manual authorization required but standard input is not interactive")
			}
			reader := bufio.NewReader(os.Stdin)
			code, err = promptAuthorizationCode(stderr, reader)
			if err != nil {
				return nil, fmt.Errorf("failed to read authorization code: %w", err)
			}
		} else {
			manager.StartWaiting()
			waitStarted = true
			code, err = manager.WaitForCode()
			if err != nil {
				return nil, fmt.Errorf("failed to get authorization code: %w", err)
			}
		}
	} else {
		if !isInteractiveInput() {
			return nil, fmt.Errorf("manual authorization required but standard input is not interactive")
		}
		reader := bufio.NewReader(os.Stdin)
		code, err = promptAuthorizationCode(stderr, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read authorization code: %w", err)
		}
	}

	if code == "" {
		return nil, fmt.Errorf("authorization code is empty")
	}
	log.Println("Oauth authorization successfully, code received:", code)

	return exchangeCodeForTokenWithPKCE(clientId, code, codeVerifier, redirectURI, EndpointMap[regionType].OAuth)
}

func startMCPOAuthFlowWithManager(ctx *cli.Context, clientId string, region RegionType,
	manager *OAuthCallbackManager, host string, port int, autoOpenBrowser bool, scope string) (*OAuthTokenResult, error) {
	stderr := getStderrWriter(ctx)
	tokenResult, err := executeOAuthFlow(ctx, clientId, region, manager, host, port, autoOpenBrowser, scope, func(authURL string) {
		cli.Printf(stderr, "Opening browser for OAuth login...\nURL: %s\n\n", authURL)
	})
	if err != nil {
		log.Println("Execute OAuth flow failed:", err)
		return nil, err
	}

	cli.Println(stderr, "OAuth login successful!")
	return tokenResult, nil
}

func startMCPOAuthFlow(ctx *cli.Context, clientId string, region RegionType, host string, port int, autoOpenBrowser bool, scope string) (*OAuthTokenResult, error) {
	manager := NewOAuthCallbackManager()

	server := &http.Server{Addr: fmt.Sprintf("%s:%d", host, port)}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// 手动输入模式需要显示授权码（autoOpenBrowser=false 表示需要显示）
		showCode := !autoOpenBrowser
		handleOAuthCallbackRequest(w, r, manager.HandleCallback, showCode)
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			manager.HandleCallback("", err)
		}
	}()

	defer server.Close()

	return startMCPOAuthFlowWithManager(ctx, clientId, region, manager, host, port, autoOpenBrowser, scope)
}

func isStderrRedirected() bool {
	info, err := os.Stderr.Stat()
	if err != nil {
		return true
	}
	return (info.Mode() & os.ModeCharDevice) == 0
}

type teeWriter struct {
	writers []io.Writer
}

func (t *teeWriter) Write(p []byte) (n int, err error) {
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
	}
	return len(p), nil
}

// 获取 stderr writer用于交互式提示
func getStderrWriter(ctx *cli.Context) io.Writer {
	var stderrWriter io.Writer
	if ctx != nil && ctx.Stderr() != nil {
		stderrWriter = ctx.Stderr()
	} else {
		stderrWriter = os.Stderr
	}

	if isStderrRedirected() {
		if tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
			return &teeWriter{writers: []io.Writer{stderrWriter, tty}}
		}
		return stderrWriter
	}
	return stderrWriter
}

func isInteractiveInput() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func promptAuthorizationCode(stderr io.Writer, reader *bufio.Reader) (string, error) {
	cli.Println(stderr, "\nPlease open the authorization URL on a machine with a browser and complete the sign-in.")
	cli.Println(stderr, "")
	cli.Println(stderr, "After authorization, the browser will redirect to a callback URL.")
	cli.Println(stderr, "Even if the page fails to load (connection error), the authorization code is in the URL.")
	cli.Println(stderr, "Please copy the value of the `code` parameter from the browser's address bar.")
	cli.Println(stderr, "")
	cli.Println(stderr, "Example: If the URL is:")
	cli.Println(stderr, "  http://127.0.0.1:8088/callback?code=abc123xyz&state=...")
	cli.Println(stderr, "  Then copy only: abc123xyz")
	cli.Println(stderr, "")

	for {
		cli.Print(stderr, "Enter authorization code: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			cli.Println(stderr, "Input is empty. Please try again.")
			continue
		}

		if strings.HasPrefix(strings.ToLower(line), "http://") ||
			strings.HasPrefix(strings.ToLower(line), "https://") ||
			strings.Contains(line, "?") ||
			strings.Contains(strings.ToLower(line), "code=") {
			cli.Println(stderr, "Please paste the authorization code only, not the entire URL.")
			continue
		}

		return line, nil
	}
}

func OpenBrowser(url string) error {
	// return errors.New("not implemented")
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

type OAuthApplication struct {
	ApplicationId        string   `json:"ApplicationId"`
	AppName              string   `json:"AppName"`
	DisplayName          string   `json:"DisplayName"`
	AppType              string   `json:"AppType"`
	RedirectUris         []string `json:"RedirectUris"`
	Scopes               []string `json:"Scopes"`
	AccessTokenValidity  int      `json:"AccessTokenValidity"`
	RefreshTokenValidity int      `json:"RefreshTokenValidity"`
}

type IMSApplication struct {
	AppId        string `json:"AppId"`
	AppName      string `json:"AppName"`
	DisplayName  string `json:"DisplayName"`
	AppType      string `json:"AppType"`
	RedirectUris struct {
		RedirectUri []string `json:"RedirectUri"`
	} `json:"RedirectUris"`
	DelegatedScope struct {
		PredefinedScopes struct {
			PredefinedScope []struct {
				Name string `json:"Name"`
			} `json:"PredefinedScope"`
		} `json:"PredefinedScopes"`
	} `json:"DelegatedScope"`
	AccessTokenValidity  int `json:"AccessTokenValidity"`
	RefreshTokenValidity int `json:"RefreshTokenValidity"`
}

// 创建应用响应
type CreateApplicationResponse struct {
	Application IMSApplication `json:"Application"`
}

type ListApplicationsResponse struct {
	Applications struct {
		Application []IMSApplication `json:"Application"`
	} `json:"Applications"`
}

type GetApplicationResponse struct {
	Application IMSApplication `json:"Application"`
}

func newOpenAPIClient(ctx *cli.Context, profile config.Profile, endpoint string) (*openapiClient.Client, error) {
	credential, err := profile.GetCredential(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	conf := &openapiClient.Config{
		Credential: credential,
		RegionId:   tea.String(profile.RegionId),
		Endpoint:   tea.String(endpoint),
		UserAgent:  tea.String(util.GetAliyunCliUserAgent()),
	}

	client, err := openapiClient.NewClient(conf)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// err means process error, application not found is not error, whether app exists based on the first return value
func findOAuthApplicationById(ctx *cli.Context, profile config.Profile, appId string, region RegionType) (*OAuthApplication, error) {
	client, err := newOpenAPIClient(ctx, profile, EndpointMap[region].IMS)
	if err != nil {
		return nil, err
	}
	params := &openapiClient.Params{
		Action:   tea.String("GetApplication"),
		Version:  tea.String("2019-08-15"),
		Protocol: tea.String("HTTPS"),
		Method:   tea.String("GET"),
		AuthType: tea.String("AK"),
		Style:    tea.String("RPC"),
		Pathname: tea.String("/"),
	}
	runtime := &dara.RuntimeOptions{}
	request := &openapiutil.OpenApiRequest{
		Query: map[string]*string{
			"AppId": tea.String(appId),
		},
	}
	response, err := client.CallApi(params, request, runtime)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := GetContentFromApiResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to get content from api response: %w", err)
	}
	var responseGet GetApplicationResponse
	if err := json.Unmarshal(bodyBytes, &responseGet); err != nil {
		return nil, err
	}

	app := responseGet.Application
	scopes := make([]string, 0, len(app.DelegatedScope.PredefinedScopes.PredefinedScope))
	for _, s := range app.DelegatedScope.PredefinedScopes.PredefinedScope {
		scopes = append(scopes, s.Name)
	}

	return &OAuthApplication{
		ApplicationId:        app.AppId,
		AppName:              app.AppName,
		DisplayName:          app.DisplayName,
		AppType:              app.AppType,
		RedirectUris:         app.RedirectUris.RedirectUri,
		Scopes:               scopes,
		AccessTokenValidity:  app.AccessTokenValidity,
		RefreshTokenValidity: app.RefreshTokenValidity,
	}, nil
}

// err means process error, application not found is not error, whether app exists based on the first return value
func findOAuthApplicationByName(ctx *cli.Context, profile config.Profile, region RegionType, appName string) (*OAuthApplication, error) {
	client, err := newOpenAPIClient(ctx, profile, EndpointMap[region].IMS)
	if err != nil {
		return nil, err
	}
	params := &openapiClient.Params{
		Action:      tea.String("ListApplications"),
		Version:     tea.String("2019-08-15"),
		Protocol:    tea.String("HTTPS"),
		Method:      tea.String("POST"),
		AuthType:    tea.String("AK"),
		Style:       tea.String("RPC"),
		Pathname:    tea.String("/"),
		ReqBodyType: tea.String("json"),
		BodyType:    tea.String("json"),
	}
	runtime := &dara.RuntimeOptions{}
	request := &openapiutil.OpenApiRequest{}
	response, err := client.CallApi(params, request, runtime)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := GetContentFromApiResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to get content from api response: %w", err)
	}
	var responseList ListApplicationsResponse
	if err := json.Unmarshal(bodyBytes, &responseList); err != nil {
		return nil, err
	}

	for _, app := range responseList.Applications.Application {
		if app.AppName == appName {
			scopes := make([]string, 0, len(app.DelegatedScope.PredefinedScopes.PredefinedScope))
			for _, s := range app.DelegatedScope.PredefinedScopes.PredefinedScope {
				scopes = append(scopes, s.Name)
			}

			return &OAuthApplication{
				ApplicationId:        app.AppId,
				AppName:              app.AppName,
				DisplayName:          app.DisplayName,
				AppType:              app.AppType,
				RedirectUris:         app.RedirectUris.RedirectUri,
				Scopes:               scopes,
				AccessTokenValidity:  app.AccessTokenValidity,
				RefreshTokenValidity: app.RefreshTokenValidity,
			}, nil
		}
	}
	return nil, nil
}

// validateOAuthApplication 验证 OAuth 应用的 Scopes, AppType 和 Callback URI 是否符合要求
func validateOAuthApplication(app *OAuthApplication, requiredScope string, host string, port int) error {
	if app == nil {
		return fmt.Errorf("OAuth application is nil")
	}

	log.Printf("Validating OAuth application: Name=%s, AppType=%s, AccessTokenValidity=%d, RefreshTokenValidity=%d",
		app.AppName, app.AppType, app.AccessTokenValidity, app.RefreshTokenValidity)

	if app.AppType != "NativeApp" {
		log.Printf("WARNING: OAuth application type is '%s', not 'NativeApp', refresh token is not supported!",
			app.AppType)
		return fmt.Errorf("OAuth application type is '%s', must be 'NativeApp' to get refresh token. "+
			"Please delete this application and let the system create a new NativeApp, or manually create a NativeApp", app.AppType)
	}

	// 验证 Scopes
	scopeFound := false
	for _, scope := range app.Scopes {
		if scope == requiredScope {
			scopeFound = true
			break
		}
	}
	if !scopeFound {
		return fmt.Errorf("OAuth application '%s' does not have required scope '%s'. Available scopes: %v",
			app.AppName, requiredScope, app.Scopes)
	}

	requiredRedirectURI := buildRedirectUri(host, port)
	redirectURIFound := false
	for _, uri := range app.RedirectUris {
		if uri == requiredRedirectURI {
			redirectURIFound = true
			break
		}
	}
	if !redirectURIFound {
		return fmt.Errorf("OAuth application '%s' does not have required redirect URI '%s'. Available redirect URIs: %v",
			app.AppName, requiredRedirectURI, app.RedirectUris)
	}

	return nil
}

func buildRedirectUri(host string, port int) string {
	return fmt.Sprintf("http://%s:%d/callback", host, port)
}

func createDefaultMCPOauthApplication(ctx *cli.Context, profile config.Profile, region RegionType, host string, port int, scope string) (*OAuthApplication, error) {
	client, err := newOpenAPIClient(ctx, profile, EndpointMap[region].IMS)
	if err != nil {
		return nil, err
	}

	redirectUri := buildRedirectUri(host, port)

	params := &openapiClient.Params{
		Action:      tea.String("CreateApplication"),
		Version:     tea.String("2019-08-15"),
		Protocol:    tea.String("HTTPS"),
		Method:      tea.String("POST"),
		AuthType:    tea.String("AK"),
		Style:       tea.String("RPC"),
		Pathname:    tea.String("/"),
		ReqBodyType: tea.String("json"),
		BodyType:    tea.String("json"),
	}

	request := &openapiutil.OpenApiRequest{
		Query: map[string]*string{
			"AppName":              tea.String(MCPOAuthAppName),
			"AppType":              tea.String("NativeApp"),
			"DisplayName":          tea.String(MCPOAuthDisplayName),
			"PredefinedScopes":     tea.String(scope),
			"ProtocolVersion":      tea.String("2.1"),
			"AccessTokenValidity":  tea.String(fmt.Sprintf("%d", AccessTokenValiditySec)),
			"RefreshTokenValidity": tea.String(fmt.Sprintf("%d", RefreshTokenValiditySec)),
			"RedirectUris":         tea.String(redirectUri),
		},
	}

	runtime := &dara.RuntimeOptions{}
	response, err := client.CallApi(params, request, runtime)
	if err != nil {
		return nil, fmt.Errorf("create application failed: %w", err)
	}

	bodyBytes, err := GetContentFromApiResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to get content from api response: %w", err)
	}

	var responseCreate CreateApplicationResponse
	if err := json.Unmarshal(bodyBytes, &responseCreate); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	scopes := make([]string, 0, len(responseCreate.Application.DelegatedScope.PredefinedScopes.PredefinedScope))
	for _, s := range responseCreate.Application.DelegatedScope.PredefinedScopes.PredefinedScope {
		scopes = append(scopes, s.Name)
	}

	return &OAuthApplication{
		ApplicationId:        responseCreate.Application.AppId,
		AppName:              responseCreate.Application.AppName,
		DisplayName:          responseCreate.Application.DisplayName,
		AppType:              responseCreate.Application.AppType,
		RedirectUris:         responseCreate.Application.RedirectUris.RedirectUri,
		Scopes:               scopes,
		AccessTokenValidity:  responseCreate.Application.AccessTokenValidity,
		RefreshTokenValidity: responseCreate.Application.RefreshTokenValidity,
	}, nil
}
