package config

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOAuthCallbackSkipsOccupiedIPv4Port(t *testing.T) {
	occupied, err := net.Listen("tcp4", "127.0.0.1:12345")
	require.NoError(t, err)
	defer occupied.Close()

	originalBrowser, originalBase := utilOpenBrowser, oauthBaseUrlMap["CN"]
	defer func() { utilOpenBrowser = originalBrowser; oauthBaseUrlMap["CN"] = originalBase }()
	var redirect string
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, redirect, r.Form.Get("redirect_uri"))
		require.Equal(t, "test-code", r.Form.Get("code"))
		_, _ = io.WriteString(w, `{"access_token":"token","refresh_token":"refresh","expires_in":3600}`)
	}))
	defer tokenServer.Close()
	oauthBaseUrlMap["CN"] = tokenServer.URL
	utilOpenBrowser = func(authURL string) error {
		u, err := url.Parse(authURL)
		require.NoError(t, err)
		redirect = u.Query().Get("redirect_uri")
		callback, err := url.Parse(redirect)
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1", callback.Hostname())
		port, err := strconv.Atoi(callback.Port())
		require.NoError(t, err)
		require.GreaterOrEqual(t, port, 12346)
		require.LessOrEqual(t, port, 12349)
		query := url.Values{"state": {u.Query().Get("state")}, "code": {"test-code"}}
		callback.RawQuery = query.Encode()
		client := &http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil}}
		defer client.CloseIdleConnections()
		// Synchronous requests prove the listener is ready before opening the browser.
		// More than one duplicate would block the old unbuffered callback channel.
		for i := 0; i < 3; i++ {
			resp, err := client.Get(callback.String())
			require.NoError(t, err)
			resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	profile := &Profile{OAuthSiteType: "CN"}
	require.NoError(t, startOauthFlowWithContext(ctx, io.Discard, profile))
	require.Equal(t, "token", profile.OAuthAccessToken)
	u, err := url.Parse(redirect)
	require.NoError(t, err)
	listener, err := net.Listen("tcp4", u.Host)
	require.NoError(t, err)
	listener.Close()
}

func TestOAuthCallbackPortsExhausted(t *testing.T) {
	for port := 12345; port <= 12349; port++ {
		listener, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		require.NoError(t, err)
		defer listener.Close()
	}
	original := utilOpenBrowser
	defer func() { utilOpenBrowser = original }()
	opened := false
	utilOpenBrowser = func(string) error { opened = true; return nil }
	err := startOauthFlowWithContext(context.Background(), io.Discard, &Profile{OAuthSiteType: "CN"})
	require.ErrorContains(t, err, "no available OAuth callback port")
	require.False(t, opened)
}

func TestOAuthCallbackWaitCancellation(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		name := "cancel"
		if timeout {
			name = "timeout"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			expected := context.Canceled
			if timeout {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
				expected = context.DeadlineExceeded
			}
			defer cancel()
			original := utilOpenBrowser
			defer func() { utilOpenBrowser = original }()
			var address string
			utilOpenBrowser = func(authURL string) error {
				u, err := url.Parse(authURL)
				require.NoError(t, err)
				callback, err := url.Parse(u.Query().Get("redirect_uri"))
				require.NoError(t, err)
				address = callback.Host
				if !timeout {
					cancel()
				}
				return nil
			}
			err := startOauthFlowWithContext(ctx, io.Discard, &Profile{OAuthSiteType: "CN"})
			require.ErrorIs(t, err, expected)
			require.ErrorContains(t, err, address)
			listener, err := net.Listen("tcp4", address)
			require.NoError(t, err)
			listener.Close()
		})
	}
}
