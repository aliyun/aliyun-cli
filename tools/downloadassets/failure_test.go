package downloadassets

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type failingAssetReader struct{}

func (failingAssetReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func TestAssetFilesystemFailures(t *testing.T) {
	dir := t.TempDir()
	_, err := sha256File(filepath.Join(dir, "missing"))
	require.Error(t, err)
	_, err = sha256File(dir)
	require.Error(t, err)
	err = ensureAsset(filepath.Join(dir, "missing"))
	require.ErrorContains(t, err, "missing asset")
	for _, name := range []string{"", "bad\nname"} {
		_, err = checksumLine(strings.Repeat("a", 64), name)
		require.ErrorContains(t, err, "invalid asset name")
	}
	require.False(t, isHex("!"))
	line := strings.Repeat("a", 64) + "  a\n"
	require.ErrorContains(t, verifyChecksum(line+line, []string{"a", "a"}), "duplicate")
	require.ErrorContains(t, commitAssets(dir, t.TempDir(), []string{"missing"}), "commit missing")
	require.ErrorContains(t, commitAssets(dir, t.TempDir(), nil), "commit checksum")
}
func TestAssetDownloadPropagatesCopyAndChecksumErrors(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(failingAssetReader{}), Header: make(http.Header)}, nil
	})}
	require.ErrorIs(t, downloadFile(context.Background(), client, "https://example.invalid/asset", "", filepath.Join(t.TempDir(), "asset"), time.Second), io.ErrUnexpectedEOF)
	client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("asset")), Header: make(http.Header)}, nil
	})
	cause := errors.New("checksum failed")
	for _, summer := range []func(string) (string, error){func(string) (string, error) { return "", cause }, func(string) (string, error) { return "invalid", nil }} {
		dir := t.TempDir()
		err := Download(context.Background(), Options{Version: "1.2.3", BaseURL: "https://example.invalid", WorkDir: dir, Client: client, Summer: summer})
		require.ErrorContains(t, err, "checksum")
		require.NoFileExists(t, filepath.Join(dir, checksumName))
		entries, e := os.ReadDir(dir)
		require.NoError(t, e)
		require.Empty(t, entries)
	}
}
