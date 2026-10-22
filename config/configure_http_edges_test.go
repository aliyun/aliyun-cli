package config

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cloudsso"
	"github.com/stretchr/testify/require"
)

type configureEdgeTransport func(*http.Request) (*http.Response, error)

func (f configureEdgeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type configureFailingBody struct{ err error }

func (b configureFailingBody) Read([]byte) (int, error) { return 0, b.err }
func (configureFailingBody) Close() error               { return nil }

func TestOAuthHTTPFailuresKeepProfileCredentials(t *testing.T) {
	oldClient, oldBase := utilNewHttpClient, oauthBaseUrlMap["CN"]
	t.Cleanup(func() { utilNewHttpClient, oauthBaseUrlMap["CN"] = oldClient, oldBase })
	cause := errors.New("response interrupted")
	for _, call := range []struct {
		name string
		fn   func(io.Writer, *Profile) error
	}{{"exchange", exchangeFromOAuth}, {"refresh", tryRefreshOauthToken}} {
		for _, tc := range []struct {
			name         string
			status       int
			body         io.ReadCloser
			transportErr error
			base         string
		}{
			{name: "invalid URL", base: "://invalid"},
			{name: "transport", transportErr: cause},
			{name: "error body", status: http.StatusBadGateway, body: configureFailingBody{cause}},
			{name: "success body", status: http.StatusOK, body: configureFailingBody{cause}},
			{name: "malformed JSON", status: http.StatusOK, body: io.NopCloser(strings.NewReader("{"))},
		} {
			t.Run(call.name+"/"+tc.name, func(t *testing.T) {
				oauthBaseUrlMap["CN"] = "https://oauth.example.invalid"
				if tc.base != "" {
					oauthBaseUrlMap["CN"] = tc.base
				}
				utilNewHttpClient = func() *http.Client {
					return &http.Client{Transport: configureEdgeTransport(func(req *http.Request) (*http.Response, error) {
						if tc.transportErr != nil {
							return nil, tc.transportErr
						}
						return &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: tc.body}, nil
					})}
				}
				profile := &Profile{Name: "test", OAuthSiteType: "CN", OAuthAccessToken: "access", OAuthRefreshToken: "refresh", OAuthAccessTokenExpire: 9999999999, AccessKeyId: "previous"}
				require.Error(t, call.fn(io.Discard, profile))
				require.Equal(t, "previous", profile.AccessKeyId)
				require.Equal(t, "access", profile.OAuthAccessToken)
			})
		}
	}
	oauthBaseUrlMap["CN"] = "https://oauth.example.invalid"
	utilNewHttpClient = func() *http.Client {
		return &http.Client{Transport: configureEdgeTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"refresh_token":"replacement"}`)), Header: make(http.Header)}, nil
		})}
	}
	require.ErrorContains(t, tryRefreshOauthToken(io.Discard, &Profile{OAuthSiteType: "CN", OAuthRefreshToken: "refresh"}), "access token not found")
	require.ErrorContains(t, exchangeFromOAuth(io.Discard, &Profile{OAuthSiteType: "CN", OAuthAccessTokenExpire: 9999999999}), "access token is empty")
}

type configureLineReader struct{ lines []string }

func (r *configureLineReader) Read(p []byte) (int, error) {
	if len(r.lines) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.lines[0])
	r.lines[0] = r.lines[0][n:]
	if r.lines[0] == "" {
		r.lines = r.lines[1:]
	}
	return n, nil
}

func TestConfigureCloudSSOSelectionFailures(t *testing.T) {
	oldInput, oldToken, oldUsers, oldConfigurations, oldSTS := stdin, cloudssoGetAccessToken, cloudssoListAllUsers, cloudssoListAllAccessConfigurations, cloudssoTryRefreshStsToken
	t.Cleanup(func() {
		stdin, cloudssoGetAccessToken, cloudssoListAllUsers, cloudssoListAllAccessConfigurations, cloudssoTryRefreshStsToken = oldInput, oldToken, oldUsers, oldConfigurations, oldSTS
	})
	cause := errors.New("service unavailable")
	for _, tc := range []struct{ name, stage, input, want string }{
		{"users failure", "users error", "", "list account failed"},
		{"no users", "users empty", "", "no account found"},
		{"account text", "choose user", "wrong", "invalid account number"},
		{"account range", "choose user", "9", "invalid account number"},
		{"config failure", "configs error", "", "list access configuration failed"},
		{"no config", "configs empty", "", "no access configuration found"},
		{"config text", "choose config", "wrong", "invalid access configuration number"},
		{"config range", "choose config", "9", "invalid access configuration number"},
		{"sts failure", "sts error", "", "create sts token failed"},
		{"invalid expiration", "expiration", "", "parse expiration time failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdin = &configureLineReader{lines: []string{"\n", tc.input + "\n"}}
			cloudssoGetAccessToken = func(*cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
				return &cloudsso.AccessTokenResponse{AccessToken: "token", ExpiresIn: 60}, nil
			}
			cloudssoListAllUsers = func(*cloudsso.ListUserParameter) ([]cloudsso.AccountDetailResponse, error) {
				if tc.stage == "users error" {
					return nil, cause
				}
				if tc.stage == "users empty" {
					return nil, nil
				}
				users := []cloudsso.AccountDetailResponse{{AccountId: "account", DisplayName: "Primary"}}
				if tc.stage == "choose user" {
					users = append(users, cloudsso.AccountDetailResponse{AccountId: "second"})
				}
				return users, nil
			}
			cloudssoListAllAccessConfigurations = func(*cloudsso.AccessConfigurationsParameter, cloudsso.AccessConfigurationsRequest) ([]cloudsso.AccessConfiguration, error) {
				if tc.stage == "configs error" {
					return nil, cause
				}
				if tc.stage == "configs empty" {
					return nil, nil
				}
				configs := []cloudsso.AccessConfiguration{{AccessConfigurationId: "config"}}
				if tc.stage == "choose config" {
					configs = append(configs, cloudsso.AccessConfiguration{AccessConfigurationId: "second"})
				}
				return configs, nil
			}
			cloudssoTryRefreshStsToken = func(*string, *string, *string, *string, *http.Client) (*cloudsso.CloudCredentialResponse, error) {
				if tc.stage == "sts error" {
					return nil, cause
				}
				return &cloudsso.CloudCredentialResponse{Expiration: "invalid"}, nil
			}
			profile := &Profile{CloudSSOSignInUrl: "https://sso.example.invalid"}
			if tc.stage == "sts error" || tc.stage == "expiration" {
				profile.CloudSSOAccountId, profile.CloudSSOAccessConfig = "account", "config"
			}
			require.ErrorContains(t, configureCloudSSO(io.Discard, profile), tc.want)
		})
	}
}

func TestConfigureRSAReadsPrivateKeyAndKeyPairName(t *testing.T) {
	oldInput := stdin
	t.Cleanup(func() { stdin = oldInput })
	path := filepath.Join(t.TempDir(), "private.pem")
	require.NoError(t, os.WriteFile(path, []byte("private key fixture"), 0600))
	stdin = &configureLineReader{lines: []string{path + "\n", "key-pair-name\n"}}
	profile := &Profile{}
	require.NoError(t, configureRsaKeyPair(io.Discard, profile))
	require.Equal(t, "private key fixture", profile.PrivateKey)
	require.Equal(t, "key-pair-name", profile.KeyPairName)
	require.Equal(t, 900, profile.ExpiredSeconds)
}
