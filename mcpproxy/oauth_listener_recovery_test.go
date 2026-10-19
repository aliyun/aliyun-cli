package mcpproxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOAuthListenerRecoveryRegistersAndRendersManualCallbackAfterLogin(t *testing.T) {
	ctx, _ := testCLIProfile(t)
	authorizationInput(t, "authorization-code\n", true)
	previousMux, previousStderr := http.DefaultServeMux, os.Stderr
	http.DefaultServeMux = http.NewServeMux()
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0600)
	require.NoError(t, err)
	os.Stderr = null
	t.Cleanup(func() { http.DefaultServeMux = previousMux; os.Stderr = previousStderr; null.Close() })
	require.Same(t, ctx.Stderr(), getStderrWriter(ctx))
	oauthServer(t, func(w http.ResponseWriter, request *http.Request) {
		require.NoError(t, request.ParseForm())
		require.Equal(t, "authorization-code", request.Form.Get("code"))
		require.Equal(t, "http://127.0.0.1:0/callback", request.Form.Get("redirect_uri"))
		io.WriteString(w, `{"access_token":"manual-access","refresh_token":"manual-refresh","expires_in":3600}`)
	})
	tokens, err := startMCPOAuthFlow(ctx, "app", RegionCN, "127.0.0.1", 0, false, "scope")
	require.NoError(t, err)
	require.Equal(t, "manual-access", tokens.AccessToken)
	require.Equal(t, "manual-refresh", tokens.RefreshToken)
	response := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/callback?code=display-code", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), "display-code")
	require.Contains(t, response.Header().Get("Content-Type"), "text/html")
}
