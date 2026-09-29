package config

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

type oauthInputFunc func([]byte) (int, error)

func (f oauthInputFunc) Read(p []byte) (int, error) { return f(p) }

func TestManualOAuthFlow(t *testing.T) {
	// Manual mode must work even if all automatic callback ports are occupied.
	for port := 12345; port <= 12349; port++ {
		ln, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
		require.NoError(t, err)
		defer ln.Close()
	}
	oldInput, oldBrowser, oldBase := stdin, utilOpenBrowser, oauthBaseUrlMap["CN"]
	defer func() { stdin = oldInput; utilOpenBrowser = oldBrowser; oauthBaseUrlMap["CN"] = oldBase }()
	utilOpenBrowser = func(string) error { t.Error("manual mode opened browser"); return nil }
	var output bytes.Buffer
	var input *strings.Reader
	var auth *url.URL
	stdin = oauthInputFunc(func(p []byte) (int, error) {
		if input == nil {
			var err error
			auth, err = url.Parse(strings.Split(output.String(), "\n")[1])
			require.NoError(t, err)
			callback := auth.Query().Get("redirect_uri")
			require.Equal(t, "http://127.0.0.1:12345/cli/callback", callback)
			values := url.Values{"state": {auth.Query().Get("state")}, "code": {"test+code"}}
			input = strings.NewReader("wrong-url\n" + callback + "?" + values.Encode() + "\ncn-hangzhou\n")
		}
		return input.Read(p)
	})
	tokenCalls, exchangeCalls := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/token":
			tokenCalls++
			require.NoError(t, r.ParseForm())
			require.Equal(t, "authorization_code", r.Form.Get("grant_type"))
			require.Equal(t, "test+code", r.Form.Get("code"))
			require.Equal(t, auth.Query().Get("redirect_uri"), r.Form.Get("redirect_uri"))
			require.Equal(t, auth.Query().Get("client_id"), r.Form.Get("client_id"))
			require.Equal(t, auth.Query().Get("code_challenge"), generateCodeChallenge(r.Form.Get("code_verifier")))
			_, _ = io.WriteString(w, `{"access_token":"manual-token","refresh_token":"refresh","expires_in":3600}`)
		case "/v1/exchange":
			exchangeCalls++
			require.Equal(t, "Bearer manual-token", r.Header.Get("Authorization"))
			_, _ = io.WriteString(w, `{"accessKeyId":"ak","accessKeySecret":"sk","securityToken":"sts","expiration":"2099-01-01T00:00:00Z"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	oauthBaseUrlMap["CN"] = server.URL
	profile := &Profile{OAuthSiteType: "CN"}
	require.NoError(t, configureOAuthWithFlow(&output, profile, startManualOauthFlow))
	require.Equal(t, 1, tokenCalls)
	require.Equal(t, 1, exchangeCalls)
	require.Equal(t, "ak", profile.AccessKeyId)
	require.Equal(t, "sts", profile.StsToken)
	require.Contains(t, output.String(), "invalid OAuth callback address")
	require.NotContains(t, output.String(), "test+code")
	require.Equal(t, "cn-hangzhou", ReadInput(""), "callback reading must not consume subsequent answers")
}

func TestParseOAuthCallbackURL(t *testing.T) {
	const base = "http://127.0.0.1:12345/cli/callback"
	tests := []struct {
		name, raw string
		denied    bool
	}{
		{"code only", "secret", false},
		{"wrong host", "http://example.com:12345/cli/callback?state=s&code=c", false},
		{"wrong port", "http://127.0.0.1:12346/cli/callback?state=s&code=c", false},
		{"wrong path", "http://127.0.0.1:12345/other?state=s&code=c", false},
		{"wrong scheme", "https://127.0.0.1:12345/cli/callback?state=s&code=c", false},
		{"missing state", base + "?code=c", false},
		{"wrong state", base + "?state=old&code=c", false},
		{"duplicate state", base + "?state=s&state=s&code=c", false},
		{"missing code", base + "?state=s", false},
		{"duplicate code", base + "?state=s&code=a&code=b", false},
		{"bad encoding", base + "?state=s&code=%zz", false},
		{"fragment", base + "?state=s&code=c#extra", false},
		{"denial", base + "?state=s&error=access_denied", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, denied, err := parseOAuthCallbackURL(tt.raw, base, "s")
			require.Error(t, err)
			require.Empty(t, code)
			require.Equal(t, tt.denied, denied)
		})
	}
	code, denied, err := parseOAuthCallbackURL(base+"?code=a%2Bb&state=s", base, "s")
	require.NoError(t, err)
	require.False(t, denied)
	require.Equal(t, "a+b", code)
}

func TestManualOAuthEOF(t *testing.T) {
	old := stdin
	defer func() { stdin = old }()
	stdin = strings.NewReader("")
	err := startManualOauthFlow(io.Discard, &Profile{OAuthSiteType: "CN"})
	require.ErrorIs(t, err, io.EOF)
}

func TestConfigureNoBrowserFlag(t *testing.T) {
	flag := NewConfigureCommand().Flags().Get("no-browser")
	require.NotNil(t, flag)
	require.Equal(t, cli.AssignedNone, flag.AssignedMode)
}

func TestConfigureSelectsManualOAuth(t *testing.T) {
	oldLoad, oldInput, oldFlow := hookLoadOrCreateConfiguration, stdin, oauthStartOauthFlow
	defer func() { hookLoadOrCreateConfiguration = oldLoad; stdin = oldInput; oauthStartOauthFlow = oldFlow }()
	hookLoadOrCreateConfiguration = func(func(string) (*Configuration, error)) func(string) (*Configuration, error) {
		return func(string) (*Configuration, error) {
			return &Configuration{Profiles: []Profile{{Name: "manual", Mode: OAuth, OAuthSiteType: "CN"}}}, nil
		}
	}
	stdin = strings.NewReader("")
	oauthStartOauthFlow = func(io.Writer, *Profile) error { t.Error("selected automatic flow"); return nil }
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	AddFlags(ctx.Flags())
	flag := NewConfigureCommand().Flags().Get("no-browser")
	flag.SetAssigned(true)
	ctx.Flags().Add(flag)
	require.ErrorIs(t, doConfigure(ctx, "manual", "OAuth"), io.EOF)
	require.ErrorContains(t, doConfigure(ctx, "manual", "AK"), "--no-browser is only supported")
}
