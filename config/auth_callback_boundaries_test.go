package config

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cloudsso"
	"github.com/stretchr/testify/require"
)

var configurationDefaultOAuthStart = oauthStartOauthFlow
var configurationDefaultOAuthExchange = oauthExchangeFromOAuth
var configurationDefaultListUsers = cloudssoListAllUsers
var configurationDefaultListConfigurations = cloudssoListAllAccessConfigurations
var configurationDefaultRefreshSSO = cloudssoTryRefreshStsToken

type configurationWriteFailure struct{ err error }

func (w configurationWriteFailure) Write([]byte) (int, error) { return 0, w.err }

func installConfigurationCallback(t *testing.T, beforeValid bool) <-chan error {
	t.Helper()
	old := utilOpenBrowser
	finished := make(chan error, 1)
	utilOpenBrowser = func(raw string) error {
		u, err := url.Parse(raw)
		if err != nil {
			return err
		}
		callback, err := url.Parse(u.Query().Get("redirect_uri"))
		if err != nil {
			return err
		}
		query := callback.Query()
		query.Set("state", u.Query().Get("state"))
		query.Set("code", "authorization-code")
		callback.RawQuery = query.Encode()
		go func() {
			client := &http.Client{Timeout: time.Second}
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if beforeValid {
					request, _ := http.NewRequest("POST", callback.String(), strings.NewReader("bad=%zz"))
					request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					response, err := client.Do(request)
					if err != nil {
						time.Sleep(5 * time.Millisecond)
						continue
					}
					response.Body.Close()
					beforeValid = false
				}
				response, err := client.Get(callback.String())
				if err != nil {
					time.Sleep(5 * time.Millisecond)
					continue
				}
				response.Body.Close()
				finished <- nil
				return
			}
			finished <- fmt.Errorf("local OAuth callback was not reachable")
		}()
		return nil
	}
	t.Cleanup(func() { utilOpenBrowser = old })
	return finished
}

func TestOAuthAuthorizationTokenExchangeFailures(t *testing.T) {
	oldClient, oldBase := utilNewHttpClient, oauthBaseUrlMap["CN"]
	t.Cleanup(func() { utilNewHttpClient, oauthBaseUrlMap["CN"] = oldClient, oldBase })
	cause := errors.New("token response interrupted")
	for _, tc := range []struct {
		name, body, base, want string
		status                 int
		transportErr, readErr  error
	}{
		{name: "request URL", base: "://invalid", want: "missing protocol scheme"},
		{name: "transport", transportErr: cause, want: cause.Error()},
		{name: "server body", status: 503, readErr: cause, want: cause.Error()},
		{name: "server JSON", status: 503, body: "{", want: "unexpected end"},
		{name: "success JSON", status: 200, body: "{", want: "unexpected end"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			callbackDone := installConfigurationCallback(t, true)
			oauthBaseUrlMap["CN"] = "https://oauth.example.invalid"
			if tc.base != "" {
				oauthBaseUrlMap["CN"] = tc.base
			}
			utilNewHttpClient = func() *http.Client {
				return &http.Client{Transport: configureEdgeTransport(func(req *http.Request) (*http.Response, error) {
					if tc.transportErr != nil {
						return nil, tc.transportErr
					}
					var body io.ReadCloser = io.NopCloser(strings.NewReader(tc.body))
					if tc.readErr != nil {
						body = configureFailingBody{tc.readErr}
					}
					return &http.Response{StatusCode: tc.status, Body: body, Header: make(http.Header)}, nil
				})}
			}
			err := startOauthFlow(io.Discard, &Profile{OAuthSiteType: "CN"})
			require.ErrorContains(t, err, tc.want)
			select {
			case err := <-callbackDone:
				require.NoError(t, err)
			case <-time.After(4 * time.Second):
				t.Fatal("callback did not finish")
			}
		})
	}
}

func TestOAuthDefaultAdaptersCompleteInternationalConfiguration(t *testing.T) {
	oldInput, oldStart, oldExchange, oldClient, oldBase := stdin, oauthStartOauthFlow, oauthExchangeFromOAuth, utilNewHttpClient, oauthBaseUrlMap["INTL"]
	t.Cleanup(func() {
		stdin, oauthStartOauthFlow, oauthExchangeFromOAuth, utilNewHttpClient, oauthBaseUrlMap["INTL"] = oldInput, oldStart, oldExchange, oldClient, oldBase
	})
	callbackDone := installConfigurationCallback(t, false)
	stdin = strings.NewReader("1\n")
	oauthStartOauthFlow = configurationDefaultOAuthStart
	oauthExchangeFromOAuth = configurationDefaultOAuthExchange
	oauthBaseUrlMap["INTL"] = "https://oauth.example.invalid"
	utilNewHttpClient = func() *http.Client {
		return &http.Client{Transport: configureEdgeTransport(func(req *http.Request) (*http.Response, error) {
			body := `{"access_token":"access","refresh_token":"refresh","expires_in":3600}`
			if req.URL.Path == "/v1/exchange" {
				require.Equal(t, "Bearer access", req.Header.Get("Authorization"))
				body = `{"accessKeyId":"id","accessKeySecret":"secret","securityToken":"sts","expiration":"2099-01-01T00:00:00Z"}`
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})}
	}
	profile := &Profile{Name: "international"}
	require.NoError(t, configureOAuth(io.Discard, profile))
	require.Equal(t, "INTL", profile.OAuthSiteType)
	require.Equal(t, "id", profile.AccessKeyId)
	require.Equal(t, "sts", profile.StsToken)
	select {
	case err := <-callbackDone:
		require.NoError(t, err)
	case <-time.After(4 * time.Second):
		t.Fatal("callback did not finish")
	}
}

func TestCloudSSODefaultListAndCredentialAdapters(t *testing.T) {
	oldInput, oldToken, oldUsers, oldConfigurations, oldRefresh, oldClient := stdin, cloudssoGetAccessToken, cloudssoListAllUsers, cloudssoListAllAccessConfigurations, cloudssoTryRefreshStsToken, utilNewHttpClient
	t.Cleanup(func() {
		stdin, cloudssoGetAccessToken, cloudssoListAllUsers, cloudssoListAllAccessConfigurations, cloudssoTryRefreshStsToken, utilNewHttpClient = oldInput, oldToken, oldUsers, oldConfigurations, oldRefresh, oldClient
	})
	stdin = strings.NewReader("")
	cloudssoListAllUsers = configurationDefaultListUsers
	cloudssoListAllAccessConfigurations = configurationDefaultListConfigurations
	cloudssoTryRefreshStsToken = configurationDefaultRefreshSSO
	cloudssoGetAccessToken = func(*cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
		return &cloudsso.AccessTokenResponse{AccessToken: "access", ExpiresIn: 3600}, nil
	}
	var paths []string
	utilNewHttpClient = func() *http.Client {
		return &http.Client{Transport: configureEdgeTransport(func(req *http.Request) (*http.Response, error) {
			paths = append(paths, req.URL.Path)
			require.Equal(t, "Bearer access", req.Header.Get("Authorization"))
			body := ""
			switch req.URL.Path {
			case "/access-assignments/accounts":
				body = `{"Accounts":[{"AccountId":"account","DisplayName":"Primary"}]}`
			case "/access-assignments/access-configurations":
				body = `{"AccessConfigurationsForAccount":[{"AccessConfigurationId":"config"}]}`
			case "/cloud-credentials":
				body = `{"CloudCredential":{"AccessKeyId":"id","AccessKeySecret":"secret","SecurityToken":"sts","Expiration":"2099-01-01T00:00:00Z"}}`
			default:
				return nil, fmt.Errorf("unexpected path %s", req.URL.Path)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})}
	}
	profile := &Profile{Name: "sso", CloudSSOSignInUrl: "https://sso.example.invalid"}
	require.NoError(t, configureCloudSSO(io.Discard, profile))
	require.Equal(t, "id", profile.AccessKeyId)
	require.Equal(t, "account", profile.CloudSSOAccountId)
	require.Equal(t, "config", profile.CloudSSOAccessConfig)
	require.Len(t, paths, 3)
}

func TestOAuthPromptAndRefreshFailures(t *testing.T) {
	oldInput, oldRefresh := stdin, oauthTryRefreshOauthToken
	t.Cleanup(func() { stdin, oauthTryRefreshOauthToken = oldInput, oldRefresh })
	cause := errors.New("output closed")
	stdin = strings.NewReader("invalid\n")
	require.ErrorIs(t, configureOAuth(configurationWriteFailure{cause}, &Profile{OAuthSiteType: "invalid"}), cause)
	require.ErrorIs(t, startOauthFlow(configurationWriteFailure{cause}, &Profile{OAuthSiteType: "CN"}), cause)
	oauthTryRefreshOauthToken = func(io.Writer, *Profile) error { return cause }
	require.ErrorIs(t, exchangeFromOAuth(io.Discard, &Profile{OAuthSiteType: "CN", OAuthRefreshToken: "refresh"}), cause)
}
