package upgrade

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

func TestUpgradeEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	previousClient, previousURL := httpClient, ossVersionURL
	t.Cleanup(func() { httpClient, ossVersionURL = previousClient, previousURL })
	ossVersionURL = "https://releases.oss.aliyuncs.com/version"
	calls := 0
	httpClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "releases.oss.aliyuncs.alibaba", r.URL.Host)
		body := "archive"
		if r.URL.Path == "/version" {
			body = "3.5.2"
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	version, err := fetchVersionFromOSS()
	require.NoError(t, err)
	require.Equal(t, "3.5.2", version)
	file := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, downloadFile(io.Discard, "https://releases.oss.aliyuncs.com/cli.tgz", file))
	content, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, "archive", string(content))
	require.Equal(t, 2, calls)
}
