package util

import (
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type unreadableTokenBody struct{ closed bool }

func (*unreadableTokenBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (b *unreadableTokenBody) Close() error           { b.closed = true; return nil }

func TestRandomStringFallbackAndTokenReadFailure(t *testing.T) {
	previous := rand.Reader
	t.Cleanup(func() { rand.Reader = previous })
	rand.Reader = &unreadableTokenBody{}
	value := RandStringBytesMaskImprSrc(32).(string)
	require.Len(t, value, 32)
	for _, character := range value {
		require.Contains(t, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", string(character))
	}
	body := &unreadableTokenBody{}
	var token struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	require.ErrorIs(t, UnmarshalJsonFromReader(body, &token), io.ErrUnexpectedEOF)
	require.True(t, body.closed)
}

func TestBrowserLaunchFailureWithoutInstalledOpener(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	require.NoError(t, OpenBrowser("https://example.invalid/authorize"))
}
