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

package config

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cloudsso"
	"github.com/aliyun/aliyun-cli/v3/util"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

var oauthClientMap = map[string]string{
	"CN":   "4038181954557748008",
	"INTL": "4103531455503354461",
}

var oauthBaseUrlMap = map[string]string{
	"CN":   "https://oauth.aliyun.com",
	"INTL": "https://oauth.alibabacloud.com",
}

var signInMap = map[string]string{
	"CN":   "https://signin.aliyun.com",
	"INTL": "https://signin.alibabacloud.com",
}

var hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
	return fn
}

var hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
	return fn
}

var stdin io.Reader = os.Stdin

// 为了方便 mock 的函数变量
var cloudssoGetAccessToken = func(ssoLogin *cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
	return ssoLogin.GetAccessToken()
}

var cloudssoListAllUsers = func(userParam *cloudsso.ListUserParameter) ([]cloudsso.AccountDetailResponse, error) {
	return userParam.ListAllUsers()
}

var cloudssoListAllAccessConfigurations = func(accessParam *cloudsso.AccessConfigurationsParameter, req cloudsso.AccessConfigurationsRequest) ([]cloudsso.AccessConfiguration, error) {
	return accessParam.ListAllAccessConfigurations(req)
}

var cloudssoTryRefreshStsToken = func(signInUrl, accessToken, accessConfig, accountId *string, httpClient *http.Client) (*cloudsso.CloudCredentialResponse, error) {
	return cloudsso.TryRefreshStsToken(signInUrl, accessToken, accessConfig, accountId, httpClient)
}

// OAuth 相关的 mock 变量
var oauthStartOauthFlow = func(w io.Writer, cp *Profile) error {
	return startOauthFlow(w, cp)
}

var oauthExchangeFromOAuth = func(w io.Writer, cp *Profile) error {
	return exchangeFromOAuth(w, cp)
}

var oauthTryRefreshOauthToken = func(w io.Writer, cp *Profile) error {
	return tryRefreshOauthToken(w, cp)
}

// Util函数的mock变量，用于测试
var utilOpenBrowser = func(url string) error {
	return util.OpenBrowser(url)
}

var utilNewHttpClient = func() *http.Client {
	return util.NewHttpClient()
}

var utilGetCurrentUnixTime = func() int64 {
	return util.GetCurrentUnixTime()
}

var doConfigureProxy = func(ctx *cli.Context, profileName string, mode string) error {
	return doConfigure(ctx, profileName, mode)
}

func loadConfiguration() (*Configuration, error) {
	return hookLoadConfiguration(LoadConfiguration)(GetConfigPath() + "/" + configFile)
}

func NewConfigureCommand() *cli.Command {
	c := &cli.Command{
		Name: "configure",
		Short: i18n.T(
			"configure credential and settings",
			"配置身份认证和其他信息"),
		Usage: "configure --mode {AK|RamRoleArn|EcsRamRole|OIDC|External|CredentialsURI|ChainableRamRoleArn|CloudSSO} --profile <profileName>",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			profileName, _ := ProfileFlag(ctx.Flags()).GetValue()
			mode, _ := ModeFlag(ctx.Flags()).GetValue()
			if mode == "" {
				// 检查 profileName 是否存在
				conf, err := loadConfiguration()
				if err == nil {
					if profileName == "" {
						profileName = conf.CurrentProfile
					}
					if profileName != "" {
						p, ok := conf.GetProfile(profileName)
						if ok {
							mode = string(p.Mode)
						}
					}
				}
			}
			return doConfigureProxy(ctx, profileName, mode)
		},
	}

	c.AddSubCommand(NewConfigureGetCommand())
	c.AddSubCommand(NewConfigureSetCommand())
	c.AddSubCommand(NewConfigureListCommand())
	c.AddSubCommand(NewConfigureDeleteCommand())
	c.AddSubCommand(NewConfigureSwitchCommand())
	return c
}

func doConfigure(ctx *cli.Context, profileName string, mode string) error {
	w := ctx.Stdout()

	conf, err := loadConfiguration()
	if err != nil {
		return err
	}

	if profileName == "" {
		if conf.CurrentProfile == "" {
			profileName = "default"
		} else {
			profileName = conf.CurrentProfile
			originMode := string(conf.GetCurrentProfile(ctx).Mode)
			if mode == "" {
				mode = originMode
			} else if mode != originMode {
				cli.Printf(w, "Warning: You are changing the authentication type of profile '%s' from '%s' to '%s'\n", profileName, originMode, mode)
			}
		}
	}
	if mode == "" {
		mode = "AK"
	}
	cp, ok := conf.GetProfile(profileName)
	if !ok {
		cp = conf.NewProfile(profileName)
	}

	cli.Printf(w, "Configuring profile '%s' in '%s' authenticate mode...\n", profileName, mode)

	if mode != "" {
		switch AuthenticateMode(mode) {
		case AK:
			cp.Mode = AK
			configureAK(w, &cp)
		case StsToken:
			cp.Mode = StsToken
			configureStsToken(w, &cp)
		case RamRoleArn:
			cp.Mode = RamRoleArn
			configureRamRoleArn(w, &cp)
		case EcsRamRole:
			cp.Mode = EcsRamRole
			configureEcsRamRole(w, &cp)
		case RamRoleArnWithEcs:
			cp.Mode = RamRoleArnWithEcs
			configureRamRoleArnWithEcs(w, &cp)
		case ChainableRamRoleArn:
			cp.Mode = ChainableRamRoleArn
			configureChainableRamRoleArn(w, &cp)
		case RsaKeyPair:
			cp.Mode = RsaKeyPair
			configureRsaKeyPair(w, &cp)
		case External:
			cp.Mode = External
			configureExternal(w, &cp)
		case CredentialsURI:
			cp.Mode = CredentialsURI
			configureCredentialsURI(w, &cp)
		case OIDC:
			cp.Mode = OIDC
			configureOIDC(w, &cp)
		case CloudSSO:
			cp.Mode = CloudSSO
			// parameter from command has higher priority, use it directly
			if CloudSSOSignInUrlFlag(ctx.Flags()).IsAssigned() {
				cp.CloudSSOSignInUrl, _ = CloudSSOSignInUrlFlag(ctx.Flags()).GetValue()
			}
			if CloudSSOAccountIdFlag(ctx.Flags()).IsAssigned() {
				cp.CloudSSOAccountId, _ = CloudSSOAccountIdFlag(ctx.Flags()).GetValue()
			}
			if CloudSSOAccessConfigFlag(ctx.Flags()).IsAssigned() {
				cp.CloudSSOAccessConfig, _ = CloudSSOAccessConfigFlag(ctx.Flags()).GetValue()
			}
			err := configureCloudSSO(w, &cp)
			if err != nil {
				return err
			}
		case OAuth:
			// only siteType is supported now
			cp.Mode = OAuth
			// restart oauth flow
			err := configureOAuth(w, &cp)
			if err != nil {
				return err
			}
			cli.Printf(w, "OAuth configuration completed. The temporary Access Key Id and Access Key Secret have been set in the profile.\n")
		default:
			return fmt.Errorf("unexcepted authenticate mode: %s", mode)
		}
	} else {
		configureAK(w, &cp)
	}

	// configure common
	if cp.Mode != CloudSSO || cp.RegionId == "" {
		cli.Printf(w, "Default Region Id [%s]: ", cp.RegionId)
		cp.RegionId = ReadInput(cp.RegionId)
	}

	if cp.Mode != CloudSSO || cp.OutputFormat == "" {
		cli.Printf(w, "Default Output Format [%s]: json (Only support json)\n", cp.OutputFormat)
		// cp.OutputFormat = ReadInput(cp.OutputFormat)
		cp.OutputFormat = "json"
	}

	if cp.Mode != CloudSSO || cp.Language == "" {
		cli.Printf(w, "Default Language [zh|en] %s: ", cp.Language)

		cp.Language = ReadInput(cp.Language)
		if cp.Language != "zh" && cp.Language != "en" {
			cp.Language = i18n.GetLanguage()
		}
	}

	//fmt.Printf("User site: [china|international|japan] %s", cp.Site)
	//cp.Site = ReadInput(cp.Site)

	cli.Printf(w, "Saving profile[%s] ...", profileName)

	conf.PutProfile(cp)
	conf.CurrentProfile = cp.Name
	err = hookSaveConfiguration(SaveConfiguration)(conf)
	// cp 要在下文的 DoHello 中使用，所以 需要建立 parent 的关系
	cp.parent = conf

	if err != nil {
		return err
	}
	cli.Printf(w, "Done.\n")

	DoHello(ctx, &cp)
	return nil
}

func configureOAuth(w io.Writer, cp *Profile) error {
	var oauthSiteType = cp.OAuthSiteType
	startChooseSiteType := false
	if oauthSiteType != "CN" && oauthSiteType != "INTL" {
		startChooseSiteType = true
	}
	if startChooseSiteType {
		cli.Println(w, "OAuth Site Type (CN: 0 or INTL: 1, default: CN): ")
		oauthSiteTypeNum := ReadInput("0")
		if oauthSiteTypeNum == "0" || strings.EqualFold(oauthSiteTypeNum, "CN") {
			oauthSiteType = "CN"
		}
		if oauthSiteTypeNum == "1" || strings.EqualFold(oauthSiteTypeNum, "INTL") {
			oauthSiteType = "INTL"
		}
	}
	// only support CN or INTL
	if oauthSiteType != "CN" && oauthSiteType != "INTL" {
		// throw error
		_, err := cli.Printf(w, "Invalid OAuth site type: %s, only support CN or INTL\n", oauthSiteType)
		if err != nil {
			return err
		}
		return fmt.Errorf("invalid OAuth site type: %s, only support CN or INTL", oauthSiteType)
	}
	cp.OAuthSiteType = oauthSiteType
	// start oauth flow
	err := oauthStartOauthFlow(w, cp)
	if err != nil {
		return err
	}
	// exchange tmp ak and set
	err = oauthExchangeFromOAuth(w, cp)
	if err != nil {
		return err
	}
	return nil
}

// exchange tmp ak from oauth token
func exchangeFromOAuth(w io.Writer, cp *Profile) error {
	oauthSiteType := cp.OAuthSiteType
	if oauthSiteType != "CN" && oauthSiteType != "INTL" {
		return fmt.Errorf("invalid OAuth site type: %s, only support CN or INTL", oauthSiteType)
	}
	// 检查 refresh token 是否过期
	refreshToken := cp.OAuthRefreshToken
	accessTokenExpire := cp.OAuthAccessTokenExpire
	if accessTokenExpire == 0 || accessTokenExpire < util.GetCurrentUnixTime() {
		if refreshToken == "" {
			return fmt.Errorf("both access token and refresh token are empty, please re-authenticate, cmd: " + buildReLoginOauthCommand(cp))
		}
		// refresh token
		err := oauthTryRefreshOauthToken(w, cp)
		if err != nil {
			return err
		}
	}
	baseUrl := oauthBaseUrlMap[oauthSiteType]
	exchangeUrl := fmt.Sprintf("%s/v1/exchange", baseUrl)
	accessToken := cp.OAuthAccessToken
	if accessToken == "" {
		return fmt.Errorf("access token is empty, please re-authenticate, cmd: " + buildReLoginOauthCommand(cp))
	}

	// 构造请求
	req, err := http.NewRequest("POST", exchangeUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "aliyun-cli")

	client := utilNewHttpClient()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("exchange failed, status code: %d", resp.StatusCode)
	}

	var result struct {
		RequestId       string `json:"requestId"`
		AccessKeyId     string `json:"accessKeyId"`
		AccessKeySecret string `json:"accessKeySecret"`
		Expiration      string `json:"expiration"`
		SecurityToken   string `json:"securityToken"`
	}
	// {
	//  "error" : "expired_token",
	//  "error_description": "The access token is expired.",
	//  "requestId": "xxxxx"
	//}
	var resultError struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		RequestId        string `json:"requestId"`
	}
	// check status code
	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, &resultError)
		if err != nil {
			return fmt.Errorf("exchange failed, status code: %d", resp.StatusCode)
		}
		return fmt.Errorf("exchange failed, error: %s, description: %s, requestId: %s",
			resultError.Error, resultError.ErrorDescription, resultError.RequestId)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return err
	}
	// 写入 cp
	cp.AccessKeyId = result.AccessKeyId
	cp.AccessKeySecret = result.AccessKeySecret
	cp.StsToken = result.SecurityToken
	parsedTime, err := time.Parse(time.RFC3339, result.Expiration)
	if err == nil {
		cp.StsExpiration = parsedTime.Unix()
	}
	return nil
}

// 构建 Oauth 应用重新登录的命令
func buildReLoginOauthCommand(cp *Profile) string {
	var sb strings.Builder
	sb.WriteString("aliyun configure --mode OAuth")
	if cp.OAuthSiteType != "" {
		sb.WriteString(" --oauth-site-type ")
		sb.WriteString(cp.OAuthSiteType)
	}
	if cp.Name != "" && cp.Name != "default" {
		sb.WriteString(" --profile ")
		sb.WriteString(cp.Name)
	}
	return sb.String()
}

func tryRefreshOauthToken(w io.Writer, cp *Profile) error {
	refreshToken := cp.OAuthRefreshToken
	if refreshToken == "" {
		return fmt.Errorf("refresh token is empty, please re-authenticate, cmd: " + buildReLoginOauthCommand(cp))
	}
	currentOauthSiteType := cp.OAuthSiteType
	if currentOauthSiteType != "CN" && currentOauthSiteType != "INTL" {
		return fmt.Errorf("invalid OAuth site type: %s, only support CN or INTL", currentOauthSiteType)
	}
	oauthClientId := oauthClientMap[cp.OAuthSiteType]
	oauthBaseUrl := oauthBaseUrlMap[cp.OAuthSiteType]
	tokenUrl := fmt.Sprintf("%s/v1/token", oauthBaseUrl)
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", oauthClientId)
	req, err := http.NewRequest("POST", tokenUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpClient := utilNewHttpClient()
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to refresh token, status code: %d", resp.StatusCode)
	}
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	err = util.UnmarshalJsonFromReader(resp.Body, &tokenResp)
	if err != nil {
		return err
	}
	if tokenResp.AccessToken == "" {
		return fmt.Errorf("access token not found in response")
	}
	cp.OAuthAccessToken = tokenResp.AccessToken
	cp.OAuthRefreshToken = tokenResp.RefreshToken
	cp.OAuthAccessTokenExpire = utilGetCurrentUnixTime() + tokenResp.ExpiresIn
	return nil
}

func startOauthFlow(w io.Writer, cp *Profile) error {
	currentOauthSiteType := cp.OAuthSiteType
	if currentOauthSiteType != "CN" && currentOauthSiteType != "INTL" {
		return fmt.Errorf("invalid OAuth site type: %s, only support CN or INTL", currentOauthSiteType)
	}
	oauthClientId := oauthClientMap[cp.OAuthSiteType]
	oauthBaseUrl := oauthBaseUrlMap[cp.OAuthSiteType]
	signInUrl := signInMap[cp.OAuthSiteType]
	// start port and listen to callback
	// port range 12345 - 12349
	port, err := detectPortUse(12345, 12349)
	if err != nil {
		return err
	}
	// start http server
	redirectUri := fmt.Sprintf("http://127.0.0.1:%s/cli/callback", strconv.Itoa(port))
	state := util.RandStringBytesMaskImprSrc(16)
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)
	oauthUrl := fmt.Sprintf("%s/oauth2/v1/auth?response_type=code&client_id=%s&redirect_uri=%s&state=%s&code_challenge=%s&code_challenge_method=S256",
		signInUrl, oauthClientId, url.QueryEscape(redirectUri), state, codeChallenge)
	_, err = cli.Printf(w, "Please open the following URL in your browser to authorize:\n%s\n", oauthUrl)
	if err != nil {
		return err
	}
	// open browser
	_ = utilOpenBrowser(oauthUrl)
	fmt.Println(i18n.T("If the browser does not open automatically, use the following URL to complete the login process:",
		"如果浏览器没有自动打开，请使用以下URL完成登录过程:").GetMessage())
	fmt.Println()
	fmt.Printf("%s: %s\n", i18n.T("SignIn url", "登录URL").GetMessage(), oauthUrl)
	fmt.Println()
	fmt.Println(i18n.T("Now you can login to your account with OAuth configuration in the browser.", "现在您可以在浏览器中使用OAuth配置登录您的账户。").GetMessage())

	codeCh := make(chan string)
	errCh := make(chan error)

	// Create a new ServeMux to avoid conflicts with global handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/cli/callback", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			return
		}
		if r.FormValue("state") != state {
			errCh <- fmt.Errorf("invalid state")
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		if code == "" {
			errCh <- fmt.Errorf("code not found")
			http.Error(w, "Code not found", http.StatusBadRequest)
			return
		}
		_, err = fmt.Fprintf(w, "Authorization successful. You can close this window.")
		if err != nil {
			return
		}
		codeCh <- code
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	// wait for code or error
	var code string
	select {
	case code = <-codeCh:
	case err = <-errCh:
	}
	// shutdown server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if srv != nil {
		_ = srv.Shutdown(shutdownCtx)
	}
	if err != nil {
		return err
	}
	if code == "" {
		return fmt.Errorf("code not found")
	}
	// exchange code for token with PKCE
	tokenUrl := fmt.Sprintf("%s/v1/token", oauthBaseUrl)
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", oauthClientId)
	data.Set("redirect_uri", redirectUri)
	data.Set("code_verifier", codeVerifier)
	req, err := http.NewRequest("POST", tokenUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpClient := utilNewHttpClient()
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	var tokenRespErr struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		RequestId        string `json:"requestId"`
	}
	if resp.StatusCode != http.StatusOK {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(d, &tokenRespErr)
		if err != nil {
			return err
		}
		return fmt.Errorf("failed to get token, status code: %d, detail: %v", resp.StatusCode, tokenRespErr)
	}
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	err = util.UnmarshalJsonFromReader(resp.Body, &tokenResp)
	if err != nil {
		return err
	}
	if tokenResp.AccessToken == "" {
		return fmt.Errorf("access token not found in response")
	}
	cp.OAuthAccessToken = tokenResp.AccessToken
	cp.OAuthRefreshToken = tokenResp.RefreshToken
	cp.OAuthAccessTokenExpire = utilGetCurrentUnixTime() + tokenResp.ExpiresIn
	return nil
}

// PKCE 辅助函数
func generateCodeVerifier() string {
	// 生成 43-128 个字符的随机字符串，这里使用 128 个字符
	return util.RandStringBytesMaskImprSrc(128).(string)
}

func generateCodeChallenge(codeVerifier string) string {
	// 使用 SHA256 哈希 code_verifier
	h := sha256.Sum256([]byte(codeVerifier))
	// 使用 base64url 编码（无填充）
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func detectPortUse(start int, end int) (int, error) {
	for port := start; port <= end; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			_ = ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found in range %d-%d", start, end)
}

func configureAK(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Access Key Id [%s]: ", MosaicString(cp.AccessKeyId, 3))
	cp.AccessKeyId = ReadInput(cp.AccessKeyId)
	cli.Printf(w, "Access Key Secret [%s]: ", MosaicString(cp.AccessKeySecret, 3))
	cp.AccessKeySecret = ReadInput(cp.AccessKeySecret)
	return nil
}

func configureStsToken(w io.Writer, cp *Profile) error {
	err := configureAK(w, cp)
	if err != nil {
		return err
	}
	cli.Printf(w, "Sts Token [%s]: ", cp.StsToken)
	cp.StsToken = ReadInput(cp.StsToken)
	return nil
}

func configureRamRoleArn(w io.Writer, cp *Profile) error {
	err := configureAK(w, cp)
	if err != nil {
		return err
	}
	cli.Printf(w, "Sts Region [%s]: ", cp.StsRegion)
	cp.StsRegion = ReadInput(cp.StsRegion)
	cli.Printf(w, "Ram Role Arn [%s]: ", cp.RamRoleArn)
	cp.RamRoleArn = ReadInput(cp.RamRoleArn)
	cli.Printf(w, "Role Session Name [%s]: ", cp.RoleSessionName)
	cp.RoleSessionName = ReadInput(cp.RoleSessionName)
	if cp.ExpiredSeconds == 0 {
		cp.ExpiredSeconds = 900
	}
	cli.Printf(w, "External ID [%s]: ", cp.ExternalId)
	cp.ExternalId = ReadInput(cp.ExternalId)
	cli.Printf(w, "Expired Seconds [%v]: ", cp.ExpiredSeconds)
	cp.ExpiredSeconds, _ = strconv.Atoi(ReadInput(strconv.Itoa(cp.ExpiredSeconds)))
	return nil
}

func configureEcsRamRole(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Ecs Ram Role [%s]: ", cp.RamRoleName)
	cp.RamRoleName = ReadInput(cp.RamRoleName)
	return nil
}

func configureRamRoleArnWithEcs(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Ecs Ram Role [%s]: ", cp.RamRoleName)
	cp.RamRoleName = ReadInput(cp.RamRoleName)
	cli.Printf(w, "Sts Region [%s]: ", cp.StsRegion)
	cp.StsRegion = ReadInput(cp.StsRegion)
	cli.Printf(w, "Ram Role Arn [%s]: ", cp.RamRoleArn)
	cp.RamRoleArn = ReadInput(cp.RamRoleArn)
	cli.Printf(w, "Role Session Name [%s]: ", cp.RoleSessionName)
	cp.RoleSessionName = ReadInput(cp.RoleSessionName)
	if cp.ExpiredSeconds == 0 {
		cp.ExpiredSeconds = 900
	}
	cli.Printf(w, "Expired Seconds [%v]: ", cp.ExpiredSeconds)
	cp.ExpiredSeconds, _ = strconv.Atoi(ReadInput(strconv.Itoa(cp.ExpiredSeconds)))
	return nil
}

func configureChainableRamRoleArn(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Source Profile [%s]: ", cp.SourceProfile)
	cp.SourceProfile = ReadInput(cp.SourceProfile)
	cli.Printf(w, "Sts Region [%s]: ", cp.StsRegion)
	cp.StsRegion = ReadInput(cp.StsRegion)
	cli.Printf(w, "Ram Role Arn [%s]: ", cp.RamRoleArn)
	cp.RamRoleArn = ReadInput(cp.RamRoleArn)
	cli.Printf(w, "Role Session Name [%s]: ", cp.RoleSessionName)
	cp.RoleSessionName = ReadInput(cp.RoleSessionName)
	if cp.ExpiredSeconds == 0 {
		cp.ExpiredSeconds = 900
	}
	cli.Printf(w, "External ID [%s]: ", cp.ExternalId)
	cp.ExternalId = ReadInput(cp.ExternalId)
	cli.Printf(w, "Expired Seconds [%v]: ", cp.ExpiredSeconds)
	cp.ExpiredSeconds, _ = strconv.Atoi(ReadInput(strconv.Itoa(cp.ExpiredSeconds)))
	return nil
}

func configureRsaKeyPair(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Rsa Private Key File: ")
	keyFile := ReadInput("")
	buf, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("read key file %s failed %v", keyFile, err)
	}
	cp.PrivateKey = string(buf)
	cli.Printf(w, "Rsa Key Pair Name: ")
	cp.KeyPairName = ReadInput("")
	cp.ExpiredSeconds = 900
	return nil
}

func configureExternal(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Process Command [%s]: ", cp.ProcessCommand)
	cp.ProcessCommand = ReadInput(cp.ProcessCommand)
	return nil
}

func configureCredentialsURI(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Credentials URI [%s]: ", cp.CredentialsURI)
	cp.CredentialsURI = ReadInput(cp.CredentialsURI)
	return nil
}

func configureOIDC(w io.Writer, cp *Profile) error {
	cli.Printf(w, "OIDC Provider ARN [%s]: ", cp.OIDCProviderARN)
	cp.OIDCProviderARN = ReadInput(cp.OIDCProviderARN)
	cli.Printf(w, "OIDC Token File [%s]: ", cp.OIDCTokenFile)
	cp.OIDCTokenFile = ReadInput(cp.OIDCTokenFile)
	cli.Printf(w, "RAM Role ARN [%s]: ", cp.RamRoleArn)
	cp.RamRoleArn = ReadInput(cp.RamRoleArn)
	cli.Printf(w, "Role Session Name [%s]: ", cp.RoleSessionName)
	cp.RoleSessionName = ReadInput(cp.RoleSessionName)
	cp.ExpiredSeconds = 3600
	return nil
}

func configureCloudSSO(w io.Writer, cp *Profile) error {
	cli.Printf(w, "CloudSSO Sign In Url [%s]: ", cp.CloudSSOSignInUrl)
	userInputCloudSSOSignInUrl := ReadInput(cp.CloudSSOSignInUrl)
	if userInputCloudSSOSignInUrl != cp.CloudSSOSignInUrl && cp.CloudSSOSignInUrl != "" {
		// 需要清空其他的字段，完整的走登录
		cp.AccessKeyId = ""
		cp.AccessKeySecret = ""
		cp.StsToken = ""
		cp.CloudSSOAccessConfig = ""
		cp.CloudSSOAccountId = ""
		cp.CloudSSOSignInUrl = userInputCloudSSOSignInUrl
		cp.AccessToken = ""
		cp.StsExpiration = 0
		cp.CloudSSOAccessTokenExpire = 0
	} else {
		cp.CloudSSOSignInUrl = userInputCloudSSOSignInUrl
	}
	if cp.CloudSSOSignInUrl == "" {
		return fmt.Errorf("CloudSSOSignInUrl is required")
	}
	// start login in, get access token, then list account for choose
	httpClient := utilNewHttpClient()
	ssoLogin := cloudsso.SsoLogin{
		SignInUrl: cp.CloudSSOSignInUrl,
		// force login
		ExpireTime: 0,
		HttpClient: httpClient,
	}
	accessToken, err := cloudssoGetAccessToken(&ssoLogin)
	if err != nil {
		return fmt.Errorf("get access token failed: %s", err)
	}
	cp.AccessToken = accessToken.AccessToken
	cp.CloudSSOAccessTokenExpire = utilGetCurrentUnixTime() + int64(accessToken.ExpiresIn)
	// parse base url
	baseUrl, err := url.Parse(ssoLogin.SignInUrl)
	// list account for choose
	userParameter := cloudsso.ListUserParameter{
		AccessToken: cp.AccessToken,
		BaseUrl:     baseUrl.Scheme + "://" + baseUrl.Host,
		HttpClient:  httpClient,
	}
	allUser, err := cloudssoListAllUsers(&userParameter)
	if err != nil {
		return fmt.Errorf("list account failed: %s", err)
	}
	// if allUser is empty, return error
	if len(allUser) == 0 {
		return fmt.Errorf("no account found")
	}
	accountIdHistory := cp.CloudSSOAccountId
	if accountIdHistory != "" {
		// 已经指定了账号，检查是否存在，如果不存在需要继续指定
		var exist = false
		for _, user := range allUser {
			if user.AccountId == accountIdHistory {
				exist = true
				break
			}
		}
		if !exist {
			cli.Printf(w, "Account %s not found, please choose again\n", accountIdHistory)
			// clear history
			cp.CloudSSOAccountId = ""
		}
	}
	if cp.CloudSSOAccountId == "" {
		// 只有当账户不存在时才需要重新选择
		// if allUser has only one account, use it directly
		if len(allUser) == 1 {
			cp.CloudSSOAccountId = allUser[0].AccountId
			cli.Printf(w, "Account: %s\n", allUser[0].DisplayName)
		} else {
			// print all user id
			cli.Println(w, "Please choose an account:")
			for i, user := range allUser {
				fmt.Printf("%d. %s\n", i+1, user.DisplayName)
			}
			cli.Printf(w, "Please input the account number: ")
			var accountNumber int
			// read input
			input := ReadInput("1")
			// parse input to int
			accountNumber, err = strconv.Atoi(input)
			if err != nil {
				return fmt.Errorf("invalid account number: %s", err)
			}
			if accountNumber < 1 || accountNumber > len(allUser) {
				return fmt.Errorf("invalid account number")
			}
			cp.CloudSSOAccountId = allUser[accountNumber-1].AccountId
		}
	}
	// get access configuration
	accessConfigurationParameter := cloudsso.AccessConfigurationsParameter{
		AccessToken: cp.AccessToken,
		UrlPrefix:   baseUrl.Scheme + "://" + baseUrl.Host,
		HttpClient:  httpClient,
		AccountId:   cp.CloudSSOAccountId,
	}
	accessConfigurations, err := cloudssoListAllAccessConfigurations(&accessConfigurationParameter, cloudsso.AccessConfigurationsRequest{
		AccountId: cp.CloudSSOAccountId,
	})
	if err != nil {
		return fmt.Errorf("list access configuration failed: %s", err)
	}
	if len(accessConfigurations) == 0 {
		return fmt.Errorf("no access configuration found")
	}
	acHistory := cp.CloudSSOAccessConfig
	if acHistory != "" {
		// 判断是否存在
		var exist = false
		for _, accessConfiguration := range accessConfigurations {
			if accessConfiguration.AccessConfigurationId == acHistory {
				exist = true
				break
			}
		}
		if !exist {
			cli.Printf(w, "Access Configuration %s not found, please choose again\n", acHistory)
			// clear history
			cp.CloudSSOAccessConfig = ""
		}
	}
	if cp.CloudSSOAccessConfig == "" {
		// if accessConfigurations has only one access configuration, use it directly
		if len(accessConfigurations) == 1 {
			cp.CloudSSOAccessConfig = accessConfigurations[0].AccessConfigurationId
			cli.Printf(w, "Access Configuration: %s\n", accessConfigurations[0].AccessConfigurationId)
		} else {
			// print all access configuration id
			cli.Println(w, "Please choose an access configuration:")
			for i, accessConfiguration := range accessConfigurations {
				cli.Printf(w, "%d. %s\n", i+1, accessConfiguration.AccessConfigurationName)
			}
			cli.Printf(w, "Please input the access configuration number: ")
			var accessConfigurationNumber int
			// read input
			input := ReadInput("1")
			// parse input to int
			accessConfigurationNumber, err = strconv.Atoi(input)
			if err != nil {
				return fmt.Errorf("invalid access configuration number: %s", err)
			}
			if accessConfigurationNumber < 1 || accessConfigurationNumber > len(accessConfigurations) {
				return fmt.Errorf("invalid access configuration number")
			}
			cp.CloudSSOAccessConfig = accessConfigurations[accessConfigurationNumber-1].AccessConfigurationId
		}
	}
	// create sts token
	stsInfo, err := cloudssoTryRefreshStsToken(&cp.CloudSSOSignInUrl, &cp.AccessToken, &cp.CloudSSOAccessConfig,
		&cp.CloudSSOAccountId, httpClient)
	if err != nil {
		return fmt.Errorf("create sts token failed: %s", err)
	}
	cp.AccessKeyId = stsInfo.AccessKeyId
	cp.AccessKeySecret = stsInfo.AccessKeySecret
	cp.StsToken = stsInfo.SecurityToken
	// Expiration is UTC time, 2015-04-09T11:52:19Z, convert to int
	// Parse the time string
	parsedTime, err := time.Parse(time.RFC3339, stsInfo.Expiration)
	if err != nil {
		return fmt.Errorf("parse expiration time failed: %s", err)
	}

	// Convert to Unix time (int64)
	unixTime := parsedTime.Unix()
	cp.StsExpiration = unixTime - 5
	return nil
}

func ReadInput(defaultValue string) string {
	var s string
	scanner := bufio.NewScanner(stdin)
	if scanner.Scan() {
		s = scanner.Text()
	}
	if s == "" {
		return defaultValue
	}
	return strings.TrimSpace(s)
}

func MosaicString(s string, lastChars int) string {
	r := len(s) - lastChars
	if r > 0 {
		return strings.Repeat("*", r) + s[r:]
	} else {
		return strings.Repeat("*", len(s))
	}
}

func GetLastChars(s string, lastChars int) string {
	r := len(s) - lastChars
	if r > 0 {
		return s[r:]
	} else {
		return strings.Repeat("*", len(s))
	}
}
