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
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	openapiTeaUtils "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/util"
)

type MCPInfoUrls struct {
	SSE string `json:"sse"`
	MCP string `json:"mcp"`
}

type MCPServerInfo struct {
	Id         string      `json:"id"`
	Name       string      `json:"name"`
	SourceType string      `json:"sourceType"`
	Product    string      `json:"product"`
	Urls       MCPInfoUrls `json:"urls"`
}

type ListMCPServersResponse struct {
	ApiMcpServers []MCPServerInfo `json:"apiMcpServers"`
}

func ListMCPServers(ctx *cli.Context, regionType RegionType) ([]MCPServerInfo, error) {
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		return nil, err
	}
	client, err := newOpenAPIClient(ctx, profile, EndpointMap[regionType].MCP)
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

type RuntimeStats struct {
	StartTime time.Time

	TotalRequests   int64
	SuccessRequests int64
	ErrorRequests   int64
	ActiveRequests  int64

	TokenRefreshes     int64
	TokenRefreshErrors int64
	LastTokenRefresh   time.Time
}

type MCPProxy struct {
	Host           string
	Port           int
	RegionType     RegionType
	Server         *http.Server // 只会在 Start() 中赋值一次，如果程序改变执行流，则需要加锁保护
	McpServers     []MCPServerInfo
	TokenRefresher *TokenRefresher
	stopCh         chan struct{}
	stats          *RuntimeStats
}

func NewMCPProxy(host string, port int, regionType RegionType, scope string, mcpProfile *McpProfile, servers []MCPServerInfo, manager *OAuthCallbackManager, autoOpenBrowser bool) *MCPProxy {
	stats := &RuntimeStats{
		StartTime: time.Now(),
	}
	return &MCPProxy{
		Host:       host,
		Port:       port,
		RegionType: regionType,
		McpServers: servers,
		TokenRefresher: &TokenRefresher{
			profile:         mcpProfile,
			regionType:      regionType,
			callbackManager: manager,
			host:            host,
			port:            port,
			scope:           scope,
			autoOpenBrowser: autoOpenBrowser,
			stopCh:          make(chan struct{}),
			tokenCh:         make(chan TokenInfo, 1), // 带缓冲的 channel，存储最新的 token
			fatalErrCh:      make(chan error, 1),
			stats:           stats,
		},
		stopCh: make(chan struct{}),
		stats:  stats,
	}
}

func (p *MCPProxy) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", p.handleOAuthCallback)
	mux.HandleFunc("/health", p.handleHealth)
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

func (p *MCPProxy) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	showCode := !p.TokenRefresher.autoOpenBrowser
	handleOAuthCallbackRequest(w, r, p.TokenRefresher.callbackManager.HandleCallback, showCode)
}

func (p *MCPProxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 检查基本健康状态
	now := time.Now()
	health := map[string]any{
		"status":         "healthy",
		"timestamp":      now.Unix(),
		"timestamp_iso":  now.Format(time.RFC3339),
		"uptime":         time.Since(p.stats.StartTime).String(),
		"uptime_seconds": time.Since(p.stats.StartTime).Seconds(),
	}

	p.TokenRefresher.mu.RLock()
	currentTime := util.GetCurrentUnixTime()
	tokenExpired := p.TokenRefresher.profile.MCPOAuthAccessTokenExpire <= currentTime
	tokenExpiresIn := p.TokenRefresher.profile.MCPOAuthAccessTokenExpire - currentTime
	refreshTokenExpired := p.TokenRefresher.profile.MCPOAuthRefreshTokenExpire <= currentTime
	refreshTokenExpiresIn := p.TokenRefresher.profile.MCPOAuthRefreshTokenExpire - currentTime
	p.TokenRefresher.mu.RUnlock()

	if tokenExpired {
		health["status"] = "degraded"
		health["token_status"] = "expired"
	} else {
		health["token_status"] = "valid"
		health["token_expires_in"] = tokenExpiresIn
		health["token_expires_inh"] = time.Duration(tokenExpiresIn * int64(time.Second)).String()
	}

	if refreshTokenExpired {
		health["status"] = "degraded"
		health["refresh_token_status"] = "expired"
	} else {
		health["refresh_token_status"] = "valid"
		health["refresh_token_expires_in"] = refreshTokenExpiresIn
		health["refresh_token_expires_inh"] = time.Duration(refreshTokenExpiresIn * int64(time.Second)).String()
	}

	// 检查内存
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	health["memory"] = map[string]interface{}{
		"alloc_mb":      m.Alloc / 1024 / 1024,
		"sys_mb":        m.Sys / 1024 / 1024,
		"heap_alloc_mb": m.HeapAlloc / 1024 / 1024,
		"heap_inuse_mb": m.HeapInuse / 1024 / 1024,
		"num_gc":        m.NumGC,
	}

	// 内存使用超过 500MB 警告
	if m.Alloc > 500*1024*1024 {
		health["status"] = "degraded"
		health["memory_warning"] = "high memory usage"
	}

	health["goroutines"] = runtime.NumGoroutine()

	health["requests"] = map[string]interface{}{
		"total":   atomic.LoadInt64(&p.stats.TotalRequests),
		"success": atomic.LoadInt64(&p.stats.SuccessRequests),
		"error":   atomic.LoadInt64(&p.stats.ErrorRequests),
		"active":  atomic.LoadInt64(&p.stats.ActiveRequests),
	}

	tokenRefreshes := map[string]interface{}{
		"total":  atomic.LoadInt64(&p.stats.TokenRefreshes),
		"errors": atomic.LoadInt64(&p.stats.TokenRefreshErrors),
	}
	if !p.stats.LastTokenRefresh.IsZero() {
		tokenRefreshes["last_refresh"] = p.stats.LastTokenRefresh.Unix()
	}
	health["token_refreshes"] = tokenRefreshes

	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

func (p *MCPProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&p.stats.TotalRequests, 1)
	atomic.AddInt64(&p.stats.ActiveRequests, 1)
	defer atomic.AddInt64(&p.stats.ActiveRequests, -1)

	// 检查是否正在关闭
	select {
	case <-p.stopCh:
		atomic.AddInt64(&p.stats.ErrorRequests, 1)
		http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
		return
	default:
	}

	accessToken, err := p.getMCPAccessToken()
	if err != nil {
		atomic.AddInt64(&p.stats.ErrorRequests, 1)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	upstreamReq, err := p.buildUpstreamRequest(r, accessToken)
	if err != nil {
		atomic.AddInt64(&p.stats.ErrorRequests, 1)
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
		atomic.AddInt64(&p.stats.ErrorRequests, 1)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 如果响应状态码为 401，则重新授权
	if resp.StatusCode == http.StatusUnauthorized {
		if err := p.TokenRefresher.reauthorizeWithProxy(); err != nil {
			log.Println("MCP Proxy gets mcp server response error from http request when reauthorize with proxy", err.Error())
			atomic.AddInt64(&p.stats.ErrorRequests, 1)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
	}
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") {
		p.handleSSE(w, resp)
		if resp.StatusCode < 400 {
			atomic.AddInt64(&p.stats.SuccessRequests, 1)
		} else {
			atomic.AddInt64(&p.stats.ErrorRequests, 1)
		}
		return
	}

	p.handleHTTP(w, resp)
	if resp.StatusCode < 400 {
		atomic.AddInt64(&p.stats.SuccessRequests, 1)
	} else {
		atomic.AddInt64(&p.stats.ErrorRequests, 1)
	}

}

func (p *MCPProxy) getMCPAccessToken() (string, error) {
	var tokenInfo TokenInfo
	select {
	case tokenInfo = <-p.TokenRefresher.tokenCh:
	default:
		// channel 为空，从 profile 读取（加读锁保护）
		p.TokenRefresher.mu.RLock()
		tokenInfo = TokenInfo{
			Token:     p.TokenRefresher.profile.MCPOAuthAccessToken,
			ExpiresAt: p.TokenRefresher.profile.MCPOAuthAccessTokenExpire,
		}
		p.TokenRefresher.mu.RUnlock()
	}

	currentTime := util.GetCurrentUnixTime()
	// 检查 token 是否过期
	if tokenInfo.ExpiresAt > currentTime {
		// Token 有效，将 token 放回 channel（供其他 goroutine 使用）
		select {
		case p.TokenRefresher.tokenCh <- tokenInfo:
		default:
			// channel 已满，忽略（说明已经有最新的 token 在 channel 中）
		}
		return tokenInfo.Token, nil
	}

	if err := p.TokenRefresher.ForceRefresh(); err != nil {
		return "", fmt.Errorf("failed to refresh access token: %w", err)
	}

	select {
	case tokenInfo = <-p.TokenRefresher.tokenCh:
		return tokenInfo.Token, nil
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("timeout waiting for refreshed token")
	}
}

func (p *MCPProxy) buildUpstreamRequest(r *http.Request, accessToken string) (*http.Request, error) {
	upstreamBaseURL := fmt.Sprintf("https://%s", EndpointMap[p.RegionType].MCP)

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
	upstreamReq.Header.Set("User-Agent", fmt.Sprintf("%s/aliyun-cli-mcp-proxy", util.GetAliyunCliUserAgent()))

	return upstreamReq, nil
}

func (p *MCPProxy) handleSSE(w http.ResponseWriter, resp *http.Response) {
	log.Println("SSE Upstream Request", resp.Request.URL.String())
	log.Println("SSE Upstream Response", resp.StatusCode)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

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
	MaxSaveFailures               = 3
	CheckInterval                 = 30 * time.Second
	AccessTokenRefreshWindow      = 7 * time.Minute  // Access token 提前刷新窗口
	RefreshTokenRefreshWindow     = 13 * time.Minute // Refresh token 提前重新授权窗口
	WaitForRefreshTimeout         = 5 * time.Second
	WaitForReauthorizationTimeout = 120 * time.Second
)

type TokenInfo struct {
	Token     string
	ExpiresAt int64
}

type TokenRefresher struct {
	profile         *McpProfile
	host            string // 代理主机
	port            int    // 代理端口
	regionType      RegionType
	scope           string // OAuth scope
	callbackManager *OAuthCallbackManager
	mu              sync.RWMutex // 保护刷新操作的读写锁
	refreshing      bool         // 标记是否正在刷新，防止重复刷新
	reauthorizing   bool         // 标记是否正在重新授权，防止重复重新授权
	autoOpenBrowser bool         // 是否自动打开浏览器（false 表示手动输入 code 模式）
	stopCh          chan struct{}
	tokenCh         chan TokenInfo // 用于传递 token 的 channel
	ticker          *time.Ticker
	fatalErrCh      chan error    // 用于通知致命错误的 channel
	stats           *RuntimeStats // 运行时统计信息（可选，用于更新 token 刷新统计）
}

func (r *TokenRefresher) Start() {
	r.ticker = time.NewTicker(CheckInterval)
	defer r.ticker.Stop()

	log.Println("Token refresher started")

	r.sendToken()
	for {
		select {
		case <-r.ticker.C:
			r.checkAndRefresh()
		case <-r.stopCh:
			return
		}
	}
}

func (r *TokenRefresher) sendToken() {
	r.mu.RLock()
	token := r.profile.MCPOAuthAccessToken
	expiresAt := r.profile.MCPOAuthAccessTokenExpire
	r.mu.RUnlock()

	select {
	case r.tokenCh <- TokenInfo{Token: token, ExpiresAt: expiresAt}:
		// 成功发送
	default:
		// channel 已满，清空旧值后发送新值
		select {
		case <-r.tokenCh:
		default:
		}
		r.tokenCh <- TokenInfo{Token: token, ExpiresAt: expiresAt}
	}
}

func (r *TokenRefresher) Stop() {
	close(r.stopCh)
}

func (r *TokenRefresher) checkAndRefresh() {
	r.mu.RLock()
	currentTime := util.GetCurrentUnixTime()
	needRefresh := false
	needReauth := false

	// 如果 refresh token 过期，则重新授权
	if r.profile.MCPOAuthRefreshTokenExpire-currentTime < int64(RefreshTokenRefreshWindow.Seconds()) {
		needReauth = true
	}
	// 如果 access token 过期，则刷新 access token
	if r.profile.MCPOAuthAccessTokenExpire-currentTime < int64(AccessTokenRefreshWindow.Seconds()) {
		needRefresh = true
	}
	r.mu.RUnlock()

	if needReauth {
		if err := r.reauthorizeWithProxy(); err != nil {
			r.reportFatalError(fmt.Errorf("re-authorization failed: %v. Please restart aliyun mcp-proxy", err))
			return
		}
	} else if needRefresh {
		if err := r.refreshAccessToken(); err != nil {
			r.reportFatalError(fmt.Errorf("refresh token invalid. Please restart aliyun mcp-proxy"))
			return
		}
	}
}

func (r *TokenRefresher) refreshAccessToken() error {
	r.mu.Lock()

	if r.refreshing {
		currentTime := util.GetCurrentUnixTime()
		currentExpiresAt := r.profile.MCPOAuthAccessTokenExpire
		if currentExpiresAt > currentTime {
			r.mu.Unlock()
			return nil
		}
		// Token 已过期，必须等待刷新完成
		r.mu.Unlock()
		return r.waitForRefresh(currentExpiresAt)
	}

	currentTime := util.GetCurrentUnixTime()
	if r.profile.MCPOAuthAccessTokenExpire > currentTime {
		r.mu.Unlock()
		return nil
	}

	r.refreshing = true
	endpoint := EndpointMap[r.regionType].OAuth
	clientId := r.profile.MCPOAuthAppId
	refreshToken := r.profile.MCPOAuthRefreshToken
	r.mu.Unlock()

	// 执行网络请求（不持有锁，避免阻塞）
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientId)
	data.Set("refresh_token", refreshToken)

	newTokens, err := oauthRefresh(endpoint, data)
	if err != nil {
		r.mu.Lock()
		r.refreshing = false
		r.mu.Unlock()
		if r.stats != nil {
			atomic.AddInt64(&r.stats.TokenRefreshErrors, 1)
		}
		return fmt.Errorf("oauth refresh failed: %w", err)
	}

	r.mu.Lock()
	currentTime = util.GetCurrentUnixTime()
	r.profile.MCPOAuthAccessToken = newTokens.AccessToken
	r.profile.MCPOAuthRefreshToken = newTokens.RefreshToken
	r.profile.MCPOAuthAccessTokenExpire = currentTime + newTokens.ExpiresIn
	r.refreshing = false

	retrySaveProfile(
		r.atomicSaveProfile,
		MaxSaveFailures,
		func() {
			r.mu.Unlock()
			r.reportFatalError(fmt.Errorf("critical: failed to save refreshed tokens after %d attempts. "+
				"Please re-login with: aliyun configure and run 'aliyun mcp-proxy' again", MaxSaveFailures))
		},
	)
	r.mu.Unlock()

	log.Println("Token refreshed successfully")

	if r.stats != nil {
		atomic.AddInt64(&r.stats.TokenRefreshes, 1)
		r.stats.LastTokenRefresh = time.Now()
	}

	r.sendToken()
	return nil
}

func (r *TokenRefresher) waitForRefresh(currentExpiresAt int64) error {
	deadline := time.Now().Add(WaitForRefreshTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)

		r.mu.RLock()
		if !r.refreshing && r.profile.MCPOAuthAccessTokenExpire > currentExpiresAt {
			r.mu.RUnlock()
			return nil
		}
		r.mu.RUnlock()
	}

	return fmt.Errorf("timeout waiting for token refresh")
}

func (r *TokenRefresher) waitForReauthorization(currentRefreshTokenExpire int64) error {
	deadline := time.Now().Add(WaitForReauthorizationTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)

		r.mu.RLock()
		if !r.reauthorizing && r.profile.MCPOAuthRefreshTokenExpire > currentRefreshTokenExpire {
			r.mu.RUnlock()
			return nil
		}
		r.mu.RUnlock()
	}

	return fmt.Errorf("timeout waiting for reauthorization")
}

func (r *TokenRefresher) ForceRefresh() error {
	return r.refreshAccessToken()
}

func (r *TokenRefresher) atomicSaveProfile() error {
	return saveMcpProfile(r.profile)
}

func (r *TokenRefresher) reportFatalError(err error) {
	select {
	case r.fatalErrCh <- err:
	default:
		// channel 已满，说明已经有错误在等待处理，忽略新的错误
	}
}

func retrySaveProfile(saveFn func() error, maxAttempts int, onMaxFailures func()) {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := saveFn(); err == nil {
			return
		}
		if attempt == maxAttempts {
			onMaxFailures()
		}
	}
}

func (r *TokenRefresher) reauthorizeWithProxy() error {
	r.mu.Lock()

	if r.reauthorizing {
		currentTime := util.GetCurrentUnixTime()
		currentRefreshTokenExpire := r.profile.MCPOAuthRefreshTokenExpire
		if currentRefreshTokenExpire > currentTime {
			r.mu.Unlock()
			return nil
		}
		// Refresh token 已过期，必须等待重新授权完成
		r.mu.Unlock()
		return r.waitForReauthorization(currentRefreshTokenExpire)
	}

	r.reauthorizing = true
	profile := r.profile
	r.mu.Unlock()

	// 执行 OAuth 流程（不持有锁，避免阻塞）
	oauthScope := r.scope
	if oauthScope == "" {
		oauthScope = "/acs/mcp-server"
	}
	stderr := getStderrWriter(nil)
	if err := executeOAuthFlow(nil, profile, r.regionType, r.callbackManager, r.host, r.port, r.autoOpenBrowser, oauthScope, func(authURL string) {
		cli.Printf(stderr, "OAuth Re-authorization Required. Please visit: %s\n", authURL)
	}); err != nil {
		r.mu.Lock()
		r.reauthorizing = false
		r.mu.Unlock()
		if r.stats != nil {
			atomic.AddInt64(&r.stats.TokenRefreshErrors, 1)
		}
		return err
	}

	r.mu.Lock()
	currentTime := util.GetCurrentUnixTime()
	r.profile.MCPOAuthRefreshTokenExpire = currentTime + int64(r.profile.MCPOAuthRefreshTokenValidity)
	r.reauthorizing = false

	retrySaveProfile(
		r.atomicSaveProfile,
		MaxSaveFailures,
		func() {
			r.mu.Unlock()
			r.reportFatalError(fmt.Errorf("critical: failed to save reauthorized tokens after %d attempts. "+
				"Please re-login with: aliyun configure and run 'aliyun mcp-proxy' again", MaxSaveFailures))
		},
	)
	r.mu.Unlock()

	if r.stats != nil {
		atomic.AddInt64(&r.stats.TokenRefreshes, 1)
		r.stats.LastTokenRefresh = time.Now()
	}

	r.sendToken()
	return nil
}
