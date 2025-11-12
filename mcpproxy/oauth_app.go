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
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	openapiTeaUtils "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

const (
	OAuthTimeout            = 5 * time.Minute
	AccessTokenValiditySec  = 10800    // 3 hours
	RefreshTokenValiditySec = 31536000 // 365 days (1 year)
)

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
	case <-time.After(OAuthTimeout):
		return "", fmt.Errorf("timeout waiting for authorization")
	}
}

func handleOAuthCallbackRequest(w http.ResponseWriter, r *http.Request, handler func(string, error) bool) bool {
	if r.URL.Path != "/callback" {
		return false
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		handler("", fmt.Errorf("no authorization code received"))
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error: No authorization code received")
		return true
	}

	handler(code, nil)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<html><body><script>window.close();</script></body></html>`)
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
	req, err := http.NewRequest("POST", endpoint+"/v1/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := util.NewHttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed: status %d", resp.StatusCode)
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("%s: %s", tokenResp.Error, tokenResp.ErrorDescription)
	}

	return &tokenResp, nil
}

func exchangeCodeForTokenWithPKCE(profile *McpProfile, code, codeVerifier, redirectURI, oauthEndpoint string) error {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", profile.MCPOAuthAppId)
	data.Set("redirect_uri", redirectURI)
	data.Set("code_verifier", codeVerifier)

	tokenResp, err := oauthRefresh(oauthEndpoint, data)
	if err != nil {
		return fmt.Errorf("oauth refresh failed: %w", err)
	}

	currentTime := util.GetCurrentUnixTime()
	profile.MCPOAuthAccessToken = tokenResp.AccessToken
	profile.MCPOAuthRefreshToken = tokenResp.RefreshToken
	profile.MCPOAuthAccessTokenExpire = currentTime + tokenResp.ExpiresIn

	return nil
}

// buildOAuthURL 构建 OAuth 授权 URL
func buildOAuthURL(profile *McpProfile, region RegionType, host string, port int, codeChallenge string) string {
	endpoints := EndpointMap[region]
	redirectURI := buildRedirectUri(host, port)
	return fmt.Sprintf("%s/oauth2/v1/auth?client_id=%s&response_type=code&scope=/acs/mcp-server&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256",
		endpoints.SignIn, profile.MCPOAuthAppId, redirectURI, codeChallenge)
}

// 执行完整的 OAuth 授权流程
func executeOAuthFlow(profile *McpProfile, region RegionType, manager *OAuthCallbackManager,
	host string, port int, logAuthURL func(string)) error {
	endpoints := EndpointMap[region]

	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	authURL := buildOAuthURL(profile, region, host, port, codeChallenge)

	if logAuthURL != nil {
		logAuthURL(authURL)
	}

	manager.StartWaiting()
	defer manager.StopWaiting()

	OpenBrowser(authURL)

	code, err := manager.WaitForCode()
	if err != nil {
		return fmt.Errorf("failed to get authorization code: %w", err)
	}

	redirectURI := buildRedirectUri(host, port)
	if err = exchangeCodeForTokenWithPKCE(profile, code, codeVerifier, redirectURI, endpoints.OAuth); err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}

	return nil
}

func startMCPOAuthFlowWithManager(ctx *cli.Context, profile *McpProfile, region RegionType,
	manager *OAuthCallbackManager, host string, port int) error {
	if err := executeOAuthFlow(profile, region, manager, host, port, func(authURL string) {
		cli.Printf(ctx.Stdout(), "Opening browser for OAuth login...\nURL: %s\n\n", authURL)
	}); err != nil {
		return err
	}

	cli.Println(ctx.Stdout(), "OAuth login successful!")
	return nil
}

func startMCPOAuthFlow(ctx *cli.Context, profile *McpProfile, region RegionType, host string, port int) error {
	manager := NewOAuthCallbackManager()

	server := &http.Server{Addr: fmt.Sprintf("%s:%d", host, port)}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		handleOAuthCallbackRequest(w, r, manager.HandleCallback)
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			manager.HandleCallback("", err)
		}
	}()

	defer server.Close()

	return startMCPOAuthFlowWithManager(ctx, profile, region, manager, host, port)
}

func OpenBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return
	}
	cmd.Start()
}

const (
	MCPOAuthAppName     = "aliyun-cli-mcp-proxy"
	MCPOAuthDisplayName = "AliyunCLI-MCP-Proxy"
)

type OAuthApplication struct {
	ApplicationId        string   `json:"ApplicationId"`
	AppName              string   `json:"AppName"`
	DisplayName          string   `json:"DisplayName"`
	AppType              string   `json:"AppType"`
	RedirectUris         []string `json:"RedirectUris"`
	Scopes               []string `json:"Scopes"`
	GrantTypes           []string `json:"GrantTypes"`
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
type RegionType string

const (
	RegionCN              RegionType = "CN"
	RegionINTL            RegionType = "INTL"
	DefaultMcpProfileName            = "default-mcp"
)

type EndpointConfig struct {
	SignIn    string
	OAuth     string
	IMSDomain string
	MCP       string
}

// 国内/国际站端点映射
var EndpointMap = map[RegionType]EndpointConfig{
	RegionCN: {
		SignIn:    "https://signin.aliyun.com",
		OAuth:     "https://oauth.aliyun.com",
		IMSDomain: "ims.aliyuncs.com",
		MCP:       "openapi-mcp.cn-hangzhou.aliyuncs.com",
	},
	RegionINTL: {
		SignIn:    "https://signin.alibabacloud.com",
		OAuth:     "https://oauth.alibabacloud.com",
		IMSDomain: "ims.ap-southeast-1.aliyuncs.com",
		MCP:       "openapi-mcp.ap-southeast-1.aliyuncs.com",
	},
}

func getOrCreateMCPOAuthApplication(ctx *cli.Context, profile config.Profile, region RegionType, host string, port int) (*OAuthApplication, error) {
	app, err := findExistingMCPOauthApplication(ctx, profile, region)
	if err != nil {
		return nil, err
	}

	if app != nil {
		return app, nil
	}

	return createMCPOauthApplication(ctx, profile, region, host, port)
}

func findExistingMCPOauthApplicationById(ctx *cli.Context, profile config.Profile, mcpProfile *McpProfile, region RegionType) error {
	credential, err := profile.GetCredential(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get credential: %w", err)
	}
	conf := &openapiClient.Config{
		Credential: credential,
		RegionId:   tea.String(profile.RegionId),
		Endpoint:   tea.String(EndpointMap[region].IMSDomain),
	}
	client, err := openapiClient.NewClient(conf)
	if err != nil {
		return err
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
	runtime := &openapiTeaUtils.RuntimeOptions{}
	request := &openapiutil.OpenApiRequest{
		Query: map[string]*string{
			"AppId": tea.String(mcpProfile.MCPOAuthAppId),
		},
	}
	response, err := client.CallApi(params, request, runtime)
	if err != nil {
		return err
	}
	bodyBytes, err := GetContentFromApiResponse(response)
	if err != nil {
		return fmt.Errorf("failed to get content from api response: %w", err)
	}
	var responseGet GetApplicationResponse
	if err := json.Unmarshal(bodyBytes, &responseGet); err != nil {
		return err
	}
	return nil
}

func findExistingMCPOauthApplication(ctx *cli.Context, profile config.Profile, region RegionType) (*OAuthApplication, error) {
	credential, err := profile.GetCredential(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}
	conf := &openapiClient.Config{
		Credential: credential,
		RegionId:   tea.String(profile.RegionId),
		Endpoint:   tea.String(EndpointMap[region].IMSDomain),
	}
	client, err := openapiClient.NewClient(conf)
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
	runtime := &openapiTeaUtils.RuntimeOptions{}
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
		if app.AppName == MCPOAuthAppName {
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
				GrantTypes:           []string{"authorization_code", "refresh_token"},
				AccessTokenValidity:  app.AccessTokenValidity,
				RefreshTokenValidity: app.RefreshTokenValidity,
			}, nil
		}
	}
	return nil, nil
}

func buildRedirectUri(host string, port int) string {
	return fmt.Sprintf("http://%s:%d/callback", host, port)
}

func createMCPOauthApplication(ctx *cli.Context, profile config.Profile, region RegionType, host string, port int) (*OAuthApplication, error) {
	credential, err := profile.GetCredential(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	redirectUri := buildRedirectUri(host, port)

	conf := &openapiClient.Config{
		Credential: credential,
		RegionId:   tea.String(profile.RegionId),
		Endpoint:   tea.String(EndpointMap[region].IMSDomain),
	}
	client, err := openapiClient.NewClient(conf)
	if err != nil {
		return nil, err
	}

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
			"SecretRequired":       tea.String("false"),
			"PredefinedScopes":     tea.String("/acs/mcp-server"),
			"ProtocolVersion":      tea.String("2.1"),
			"GrantTypes.1":         tea.String("authorization_code"),
			"GrantTypes.2":         tea.String("refresh_token"),
			"AccessTokenValidity":  tea.String(fmt.Sprintf("%d", AccessTokenValiditySec)),
			"RefreshTokenValidity": tea.String(fmt.Sprintf("%d", RefreshTokenValiditySec)),
			"RedirectUris":         tea.String(redirectUri),
		},
	}

	runtime := &openapiTeaUtils.RuntimeOptions{}
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
		GrantTypes:           []string{"authorization_code", "refresh_token"},
		AccessTokenValidity:  responseCreate.Application.AccessTokenValidity,
		RefreshTokenValidity: responseCreate.Application.RefreshTokenValidity,
	}, nil
}
