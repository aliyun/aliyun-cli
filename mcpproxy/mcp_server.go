package mcpproxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	openapiTeaUtils "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/util"
)

// MCP Info Urls 结构
type MCPInfoUrls struct {
	SSE string `json:"sse"`
	MCP string `json:"mcp"`
}

// MCP Server 信息
type MCPServerInfo struct {
	Id         string      `json:"id"`
	Name       string      `json:"name"`
	SourceType string      `json:"sourceType"`
	Product    string      `json:"product"`
	Urls       MCPInfoUrls `json:"urls"`
}

// 列举 MCP Server 响应
type ListMCPServersResponse struct {
	ApiMcpServers []MCPServerInfo `json:"apiMcpServers"`
}

// 列举 MCP Servers
func ListMCPServers(ctx *cli.Context, region RegionType) ([]MCPServerInfo, error) {
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		return nil, err
	}
	credential, err := profile.GetCredential(ctx, nil)
	if err != nil {
		return nil, err
	}
	conf := &openapiClient.Config{
		Credential: credential,
		RegionId:   tea.String(profile.RegionId),
		Endpoint:   tea.String(EndpointMap[region].MCP),
	}
	client, err := openapiClient.NewClient(conf)
	if err != nil {
		return nil, err
	}
	params := &openapiClient.Params{
		Action:      tea.String("ListApiMcpServers"),
		Version:     tea.String("2024-11-30"),
		Protocol:    tea.String("HTTPS"),
		Method:      tea.String("GET"),
		AuthType:    tea.String("AK"),
		Style:       tea.String("ROA"),
		Pathname:    tea.String("/apimcpservers"),
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
	var responseList ListMCPServersResponse
	if err := json.Unmarshal(bodyBytes, &responseList); err != nil {
		return nil, err
	}
	return responseList.ApiMcpServers, nil
}

// MCP 代理服务
type MCPProxy struct {
	Host            string
	Port            int
	Region          RegionType
	McpProfile      *McpProfile
	Server          *http.Server
	McpServers      []MCPServerInfo
	Refresher       *TokenRefresher
	CallbackManager *OAuthCallbackManager
	stopCh          chan struct{}
}

func NewMCPProxy(host string, port int, region RegionType, mcpProfile *McpProfile, servers []MCPServerInfo, manager *OAuthCallbackManager) *MCPProxy {
	return &MCPProxy{
		Host:            host,
		Port:            port,
		Region:          region,
		McpProfile:      mcpProfile,
		McpServers:      servers,
		CallbackManager: manager,
		Refresher: &TokenRefresher{
			profile:         mcpProfile,
			region:          region,
			callbackManager: manager,
			host:            host,
			port:            port,
			stopCh:          make(chan struct{}),
		},
		stopCh: make(chan struct{}),
	}
}

func (p *MCPProxy) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.ServeHTTP)

	p.Server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", p.Host, p.Port),
		Handler: mux,
	}

	log.Printf("MCP Proxy starting on %s:%d\n", p.Host, p.Port)

	if err := p.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("proxy server failed: %w", err)
	}

	return nil
}

func (p *MCPProxy) Stop() error {
	close(p.stopCh)

	if p.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.Server.Shutdown(ctx); err != nil {
			// 如果优雅关闭超时，强制关闭
			if err == context.DeadlineExceeded {
				log.Println("Graceful shutdown timeout, forcing close...")
				return p.Server.Close()
			}
			return err
		}
	}

	return nil
}

func (p *MCPProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 检查是否正在关闭
	select {
	case <-p.stopCh:
		http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
		return
	default:
	}

	if handleOAuthCallbackRequest(w, r, p.CallbackManager.HandleCallback) {
		return
	}

	accessToken, err := p.getMCPAccessToken()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	upstreamReq, err := p.buildUpstreamRequest(r, accessToken)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	log.Println("Upstream Request", upstreamReq.URL.String())
	// 排除 Authorization 和 User-Agent 头
	// for k, v := range upstreamReq.Header {
	// 	if strings.ToLower(k) != "authorization" && strings.ToLower(k) != "user-agent" {
	// 		log.Println("Upstream Request Header", k, v)
	// 	}
	// }
	// 打印 Upstream Request Body 内容
	if upstreamReq.Body != nil {
		bodyBytes, err := io.ReadAll(upstreamReq.Body)
		if err != nil {
			log.Println("Upstream Request Body Error", err.Error())
		} else {
			log.Println("Upstream Request Body", string(bodyBytes))
			_ = upstreamReq.Body.Close()
			upstreamReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	} else {
		log.Println("Upstream Request Body <nil>")
	}
	client := &http.Client{Timeout: 0}
	resp, err := client.Do(upstreamReq)
	if err != nil {
		log.Println("MCP Proxy gets mcp server response error", err.Error())
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 如果响应状态码为 401，则重新授权
	if resp.StatusCode == http.StatusUnauthorized {
		if err := p.Refresher.reauthorizeWithProxy(); err != nil {
			log.Println("MCP Proxy gets mcp server response error from http request when reauthorize with proxy", err.Error())
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
	}
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") {
		p.handleSSE(w, resp)
		return
	}

	p.handleHTTP(w, resp)

}

func (p *MCPProxy) getMCPAccessToken() (string, error) {
	if p.McpProfile.MCPOAuthAccessTokenExpire > util.GetCurrentUnixTime() {
		return p.McpProfile.MCPOAuthAccessToken, nil
	}

	if err := p.Refresher.ForceRefresh(); err != nil {
		return "", fmt.Errorf("failed to refresh access token: %w", err)
	}

	return p.McpProfile.MCPOAuthAccessToken, nil
}

func (p *MCPProxy) buildUpstreamRequest(r *http.Request, accessToken string) (*http.Request, error) {
	upstreamBaseURL := "https://openapi-mcp.cn-hangzhou.aliyuncs.com"

	upstreamURL, err := url.Parse(upstreamBaseURL)
	if err != nil {
		return nil, err
	}

	newURL := *r.URL
	newURL.Scheme = upstreamURL.Scheme
	newURL.Host = upstreamURL.Host
	if newURL.Path == "" {
		newURL.Path = "/"
	}

	method := r.Method
	var body io.ReadCloser = r.Body

	upstreamReq, err := http.NewRequest(method, newURL.String(), body)
	if err != nil {
		return nil, err
	}

	for k, v := range r.Header {
		if strings.ToLower(k) != "host" && strings.ToLower(k) != "authorization" {
			upstreamReq.Header[k] = v
		}
	}

	upstreamReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	upstreamReq.Header.Set("User-Agent", "aliyun-cli-mcp-proxy")

	return upstreamReq, nil
}

func (p *MCPProxy) handleSSE(w http.ResponseWriter, resp *http.Response) {
	log.Println("SSE Upstream Request", resp.Request.URL.String())

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	log.Println("SSE Upstream Response", resp.StatusCode)

	for k, v := range resp.Header {
		if strings.ToLower(k) == "content-length" {
			continue
		}
		w.Header()[k] = v
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/event-stream")
	}

	w.WriteHeader(resp.StatusCode)

	reader := bufio.NewReader(resp.Body)
	for {
		// 检查是否正在关闭
		select {
		case <-p.stopCh:
			log.Println("SSE connection closed due to server shutdown")
			return
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}

		if _, err = w.Write(line); err != nil {
			break
		}
		log.Println("SSE Upstream Response Line", string(line))

		flusher.Flush()
	}
}

func (p *MCPProxy) handleHTTP(w http.ResponseWriter, resp *http.Response) {
	log.Println("HTTP Upstream Request", resp.Request.URL.String())
	log.Println("HTTP Upstream Response", resp.StatusCode)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		log.Println("MCP Proxy gets mcp server response error from http request", err.Error())

		return
	}

	// 检查是否正在关闭
	select {
	case <-p.stopCh:
		log.Println("HTTP response cancelled due to server shutdown")
		return
	default:
	}

	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(bodyBytes)

	log.Println("MCP Proxy gets mcp server response successfully from http request", resp.StatusCode)
}

const (
	MaxSaveFailures = 3
	CheckInterval   = 30 * time.Second
	RefreshWindow   = 5 * time.Minute
)

type TokenRefresher struct {
	profile         *McpProfile
	region          RegionType
	callbackManager *OAuthCallbackManager // OAuth 回调管理器（用于代理运行时的重新授权）
	host            string                // 代理主机
	port            int                   // 代理端口
	mu              sync.RWMutex
	stopCh          chan struct{}
	ticker          *time.Ticker
	saveFailures    int // 连续保存失败次数
}

func (r *TokenRefresher) Start() {
	r.ticker = time.NewTicker(CheckInterval)
	defer r.ticker.Stop()

	log.Println("Token refresher started")

	for {
		select {
		case <-r.ticker.C:
			r.checkAndRefresh()
		case <-r.stopCh:
			return
		}
	}
}

func (r *TokenRefresher) Stop() {
	close(r.stopCh)
}

func (r *TokenRefresher) checkAndRefresh() {
	r.mu.Lock()
	defer r.mu.Unlock()

	currentTime := util.GetCurrentUnixTime()
	// 如果 refresh token 过期，则重新授权
	if r.profile.MCPOAuthRefreshTokenExpire-currentTime < int64(RefreshWindow.Seconds()) {
		if err := r.reauthorizeWithProxy(); err != nil {
			log.Fatalf("Re-authorization failed: %v. Please restart aliyun mcp-proxy.", err)
		}
	}
	// 如果 access token 过期，则刷新 access token
	if r.profile.MCPOAuthAccessTokenExpire-currentTime < int64(RefreshWindow.Seconds()) {
		if err := r.refreshAccessToken(); err != nil {
			log.Fatalf("Refresh token invalid. Please restart aliyun mcp-proxy.")
		}
	}
}

func (r *TokenRefresher) refreshAccessToken() error {
	endpoint := EndpointMap[r.region].OAuth

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", r.profile.MCPOAuthAppId)
	data.Set("refresh_token", r.profile.MCPOAuthRefreshToken)

	newTokens, err := oauthRefresh(endpoint, data)
	if err != nil {
		return fmt.Errorf("oauth refresh failed: %w", err)
	}

	currentTime := util.GetCurrentUnixTime()
	r.profile.MCPOAuthAccessToken = newTokens.AccessToken
	r.profile.MCPOAuthRefreshToken = newTokens.RefreshToken
	r.profile.MCPOAuthAccessTokenExpire = currentTime + newTokens.ExpiresIn

	if err = r.atomicSaveProfile(); err != nil {
		r.saveFailures++
		if r.saveFailures >= MaxSaveFailures {
			log.Fatalf("Critical: Failed to save refreshed tokens after %d attempts. "+
				"Please re-login with: aliyun configure --mode MCPOAuth", MaxSaveFailures)
		}
		return fmt.Errorf("failed to save profile (attempt %d/%d): %w",
			r.saveFailures, MaxSaveFailures, err)
	}

	r.saveFailures = 0

	log.Println("Token refreshed successfully")
	return nil
}

func (r *TokenRefresher) ForceRefresh() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.refreshAccessToken()
}

func (r *TokenRefresher) atomicSaveProfile() error {
	return saveMcpProfile(r.profile)
}

func (r *TokenRefresher) reauthorizeWithProxy() error {
	if err := executeOAuthFlow(r.profile, r.region, r.callbackManager, r.host, r.port, func(authURL string) {
		log.Printf("OAuth Re-authorization Required. Please visit: %s\n", authURL)
	}); err != nil {
		return err
	}

	// 重新授权后，更新 refresh token 过期时间
	currentTime := util.GetCurrentUnixTime()
	r.profile.MCPOAuthRefreshTokenExpire = currentTime + int64(r.profile.MCPOAuthRefreshTokenValidity)
	if err := r.atomicSaveProfile(); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	return nil
}
