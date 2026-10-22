package downloadassets

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloadChecksumAndCommitFailuresKeepPublishedChecksum(t *testing.T) {
	for _, stage := range []string{"checksum", "commit"} {
		t.Run(stage, func(t *testing.T) {
			work := t.TempDir()
			checksum := filepath.Join(work, checksumName)
			require.NoError(t, os.WriteFile(checksum, []byte("published"), 0600))
			names := AssetNames("1.2.3")
			server := httptest.NewServer(signedHandler(func(string) []byte { return []byte("asset") }))
			t.Cleanup(server.Close)
			err := Download(context.Background(), Options{Version: "1.2.3", BaseURL: server.URL, WorkDir: work, Client: server.Client(), Summer: func(path string) (string, error) {
				if filepath.Base(path) == names[len(names)-1] {
					if stage == "checksum" {
						require.NoError(t, os.Mkdir(filepath.Join(filepath.Dir(path), checksumName), 0700))
					} else {
						require.NoError(t, os.Mkdir(filepath.Join(work, names[0]), 0700))
					}
				}
				return strings.Repeat("a", 64), nil
			}})
			require.Error(t, err)
			if stage == "checksum" {
				require.ErrorContains(t, err, "write checksum")
			} else {
				require.ErrorContains(t, err, "commit")
			}
			body, err := os.ReadFile(checksum)
			require.NoError(t, err)
			require.Equal(t, "published", string(body))
			matches, err := filepath.Glob(filepath.Join(work, ".download-assets-*"))
			require.NoError(t, err)
			require.Empty(t, matches)
		})
	}
	require.Error(t, ensureAsset("invalid\x00"))
}
