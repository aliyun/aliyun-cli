package config

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOAuthRefreshWithoutRotationRetainsRefreshToken(t *testing.T) {
	oldURL, oldClient := oauthBaseUrlMap, utilNewHttpClient
	t.Cleanup(func() { oauthBaseUrlMap, utilNewHttpClient = oldURL, oldClient })
	for _, body := range []string{`{"access_token":"new","expires_in":3600}`, `{"refresh_token":"rotated","expires_in":3600}`, `{"access_token":"new","refresh_token":"rotated","expires_in":0}`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
			defer server.Close()
			oauthBaseUrlMap = map[string]string{"CN": server.URL}
			utilNewHttpClient = server.Client
			p := Profile{OAuthSiteType: "CN", OAuthRefreshToken: "original"}
			err := tryRefreshOauthToken(io.Discard, &p)
			if body == `{"access_token":"new","expires_in":3600}` {
				require.NoError(t, err)
				require.Equal(t, "original", p.OAuthRefreshToken)
				require.Equal(t, "new", p.OAuthAccessToken)
			} else {
				require.Error(t, err)
				require.Equal(t, "rotated", p.OAuthRefreshToken)
				require.Zero(t, p.OAuthAccessTokenExpire)
			}
		})
	}
}
func TestOAuthRejectsIncompleteSTSWithoutReplacingCachedTuple(t *testing.T) {
	oldURL, oldClient := oauthBaseUrlMap, utilNewHttpClient
	t.Cleanup(func() { oauthBaseUrlMap, utilNewHttpClient = oldURL, oldClient })
	for _, body := range []string{`{}`, `{"accessKeyId":"new","accessKeySecret":"new","securityToken":"new","expiration":"invalid"}`, `{"accessKeyId":"new","accessKeySecret":"new","securityToken":"new","expiration":"2000-01-01T00:00:00Z"}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		oauthBaseUrlMap = map[string]string{"CN": server.URL}
		utilNewHttpClient = server.Client
		p := Profile{OAuthSiteType: "CN", OAuthAccessToken: "access", OAuthAccessTokenExpire: time.Now().Add(time.Hour).Unix(), AccessKeyId: "old", AccessKeySecret: "old-secret", StsToken: "old-token", StsExpiration: 1}
		err := exchangeFromOAuth(io.Discard, &p)
		require.Error(t, err)
		require.Equal(t, "old", p.AccessKeyId)
		require.Equal(t, "old-token", p.StsToken)
		require.Equal(t, int64(1), p.StsExpiration)
		server.Close()
	}
}

func TestOAuthCheckpointFailureDoesNotStartSTSExchange(t *testing.T) {
	oldURL, oldClient, oldRefresh := oauthBaseUrlMap, utilNewHttpClient, oauthTryRefreshOauthToken
	t.Cleanup(func() { oauthBaseUrlMap, utilNewHttpClient, oauthTryRefreshOauthToken = oldURL, oldClient, oldRefresh })
	exchanges := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/token" {
			fmt.Fprint(w, `{"access_token":"new","refresh_token":"rotated","expires_in":3600}`)
			return
		}
		exchanges++
		w.WriteHeader(500)
	}))
	defer server.Close()
	oauthBaseUrlMap = map[string]string{"CN": server.URL}
	utilNewHttpClient = server.Client
	oauthTryRefreshOauthToken = tryRefreshOauthToken
	p := Profile{OAuthSiteType: "CN", OAuthRefreshToken: "old", OAuthAccessTokenExpire: 1}
	sentinel := fmt.Errorf("checkpoint write failed")
	err := exchangeFromOAuthWithCheckpoint(io.Discard, &p, func() error { require.Equal(t, "rotated", p.OAuthRefreshToken); return sentinel })
	require.ErrorIs(t, err, sentinel)
	require.Zero(t, exchanges)
}
