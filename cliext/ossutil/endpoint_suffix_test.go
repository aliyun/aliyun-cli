package ossutil

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

type suffixTransport func(*http.Request) (*http.Response, error)

func TestOSSUtilRejectsInvalidEndpointSuffixBeforeLoadingProfile(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "invalid.example")
	require.ErrorContains(t, NewContext(nil).PrepareEnv(), endpoint.EnvSuffix)
}

func (f suffixTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOSSUtilVersionAndDownloadEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	filename := "ossutil"
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}
	file, err := writer.Create("package/" + filename)
	require.NoError(t, err)
	_, err = io.WriteString(file, "test executable")
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	previousTransport, previousBase := http.DefaultTransport, versionUpdateBaseUrl
	t.Cleanup(func() { http.DefaultTransport, versionUpdateBaseUrl = previousTransport, previousBase })
	versionUpdateBaseUrl = "https://assets.oss.aliyuncs.com/releases/"
	calls := 0
	http.DefaultTransport = suffixTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "assets.oss.aliyuncs.alibaba", r.URL.Host)
		body := archive.Bytes()
		if strings.HasSuffix(r.URL.Path, "version.txt") {
			body = []byte("ossutil version: 2.1.2")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(body))}, nil
	})
	version, err := GetLatestOssUtilVersion()
	require.NoError(t, err)
	require.Equal(t, "2.1.2", version)
	dir := t.TempDir()
	installed := filepath.Join(dir, "installed", filename)
	require.NoError(t, os.MkdirAll(filepath.Dir(installed), 0700))
	require.NoError(t, DownloadAndUnzip(versionUpdateBaseUrl+"tool.zip", filepath.Join(dir, "tool.zip"), installed, "package"))
	content, err := os.ReadFile(installed)
	require.NoError(t, err)
	require.Equal(t, "test executable", string(content))
	require.Equal(t, 2, calls)
}

func TestOSSUtilReceivesRewrittenProfileEndpoint(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	ctx.SetInConfigureMode(true)
	path := filepath.Join(t.TempDir(), "config.json")
	content := []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"id","access_key_secret":"secret","region_id":"cn-hangzhou","endpoint":"https://oss-cn-hangzhou.aliyuncs.com"}]}`)
	require.NoError(t, os.WriteFile(path, content, 0600))
	flag := config.ConfigurePathFlag(ctx.Flags())
	flag.SetValue(path)
	flag.SetAssigned(true)
	c := NewContext(ctx)
	require.NoError(t, c.PrepareEnv())
	decoded, err := base64.StdEncoding.DecodeString(c.envMap["OSSUTIL_CONFIG_VALUE"])
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(decoded, &payload))
	require.Equal(t, "https://oss-cn-hangzhou.aliyuncs.alibaba", payload["endpoint"])
	stored, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, content, stored)
}
