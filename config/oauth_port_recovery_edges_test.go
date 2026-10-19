package config

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOAuthPortRecoveryRejectsExhaustedCallbackPortsBeforeOpeningBrowser(t *testing.T) {
	for port := 12345; port <= 12349; port++ {
		deadline := time.Now().Add(2 * time.Second)
		for {
			listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err == nil {
				t.Cleanup(func() { listener.Close() })
				break
			}
			if time.Now().After(deadline) {
				t.Skipf("cannot reserve callback port %d: %v", port, err)
			}
			time.Sleep(10 * time.Millisecond)
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
