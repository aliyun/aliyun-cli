package upgrade

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
	"github.com/stretchr/testify/require"
)

func TestUpgradeCommandPropagatesDownloadAndInstallationFailures(t *testing.T) {
	oldClient, oldSource, oldResolve, oldDetect, oldInput := httpClient, ossVersionURL, resolveExecPathFunc, detectInstallerFunc, stdin
	t.Cleanup(func() {
		httpClient, ossVersionURL, resolveExecPathFunc, detectInstallerFunc, stdin = oldClient, oldSource, oldResolve, oldDetect, oldInput
	})
	assetName, _, archive := createUpgradeArchiveInMemory(t, "99.0.0", []byte("replacement"))
	priv := useTestPublisher(t)
	cause := errors.New("executable disappeared")
	for _, stage := range []string{"download", "resolve", "replace"} {
		t.Run(stage, func(t *testing.T) {
			ossVersionURL = "https://updates.example.invalid/version"
			stdin = strings.NewReader("y\n")
			detectInstallerFunc = func() installerType { return installerDirect }
			httpClient = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				body := archive.Bytes()
				status := 200
				switch {
				case req.URL.Path == "/version":
					body = []byte("99.0.0")
				case req.URL.Path == "/versions":
					status = http.StatusNotFound
				case strings.HasSuffix(req.URL.Path, ".minisig"):
					doc, err := publishsig.Sign(priv, assetName, bytes.NewReader(archive.Bytes()))
					if err != nil {
						return nil, err
					}
					body = doc
				case stage == "download":
					status = 503
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}, nil
			})}
			resolveExecPathFunc = func() (string, error) {
				if stage == "resolve" {
					return "", cause
				}
				return filepath.Join(t.TempDir(), "missing-binary"), nil
			}
			ctx := newTestContext()
			err := NewUpgradeCommand().Run(ctx, nil)
			require.Error(t, err)
			switch stage {
			case "download":
				require.ErrorContains(t, err, "download failed")
			case "resolve":
				require.ErrorIs(t, err, cause)
			case "replace":
				require.ErrorContains(t, err, "installation failed")
			}
		})
	}
}

func TestArchiveHeaderFailuresAndBinaryReplacementRollback(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "invalid.tgz")
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write([]byte("truncated header"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, os.WriteFile(archive, compressed.Bytes(), 0600))
	require.ErrorIs(t, extractFromTarGz(archive, filepath.Join(dir, "output"), "aliyun"), io.ErrUnexpectedEOF)
	var zipped bytes.Buffer
	zw := zip.NewWriter(&zipped)
	_, err = zw.CreateRaw(&zip.FileHeader{Name: "aliyun", Method: 99})
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	archive = filepath.Join(dir, "invalid.zip")
	require.NoError(t, os.WriteFile(archive, zipped.Bytes(), 0600))
	require.ErrorIs(t, extractFromZip(archive, filepath.Join(dir, "output"), "aliyun"), zip.ErrAlgorithm)
	current := filepath.Join(dir, "current-directory")
	require.NoError(t, os.Mkdir(current, 0700))
	replacement := filepath.Join(current, "replacement")
	require.NoError(t, os.WriteFile(replacement, []byte("keep"), 0600))
	require.Error(t, replaceBinary(replacement, current))
	require.DirExists(t, current)
	data, err := os.ReadFile(replacement)
	require.NoError(t, err)
	require.Equal(t, "keep", string(data))
}
