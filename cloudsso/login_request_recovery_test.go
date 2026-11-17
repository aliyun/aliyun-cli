package cloudsso

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSSOTokenRequestRejectsSignInURLWithoutScheme(t *testing.T) {
	requests := 0
	installLoginTransport(t, func(*http.Request) (*http.Response, error) {
		requests++
		return mockHTTPResponse(http.StatusOK, `{"AccessToken":"unexpected"}`), nil
	})

	token, err := createAccessToken("sso.example.invalid/start", "device")
	require.Nil(t, token)
	require.ErrorContains(t, err, "missing protocol scheme")
	require.Zero(t, requests, "invalid sign-in URL must be rejected before sending credentials")
}
