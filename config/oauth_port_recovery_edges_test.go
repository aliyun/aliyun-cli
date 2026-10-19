package config

import (
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOAuthPortRecoveryRejectsExhaustedCallbackPortsBeforeOpeningBrowser(t *testing.T) {
	for port := 12345; port <= 12349; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			t.Cleanup(func() { listener.Close() })
		}
	}
	previous := utilOpenBrowser
	t.Cleanup(func() { utilOpenBrowser = previous })
	utilOpenBrowser = func(string) error { t.Fatal("no browser may open without a callback port"); return nil }
	profile := &Profile{OAuthSiteType: "CN"}
	require.ErrorContains(t, startOauthFlow(io.Discard, profile), "no available port found in range 12345-12349")
	require.Empty(t, profile.OAuthAccessToken)
	require.Empty(t, profile.OAuthRefreshToken)
}
