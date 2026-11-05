package plugin

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

type suffixTransport func(*http.Request) (*http.Response, error)

func (f suffixTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPluginPackageEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	m := &Manager{httpTransport: suffixTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "packages.oss.aliyuncs.alibaba", r.URL.Host)
		require.Equal(t, "/plugin.tgz", r.URL.Path)
		require.Equal(t, "identity", r.Header.Get("Accept-Encoding"))
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("package"))}, nil
	})}
	response, err := m.httpGet("https://packages.oss.aliyuncs.com/plugin.tgz", time.Second)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "package", string(body))
}

func TestBinaryPluginReceivesEffectiveEndpoint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell")
	}
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	path := filepath.Join(t.TempDir(), "plugin")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0700))
	var stdout, stderr bytes.Buffer
	require.NoError(t, runPluginCommand(path, []string{"--endpoint=api.aliyuncs.com", "--body", "--endpoint=payload.aliyuncs.com", "--", "--endpoint=opaque.aliyuncs.com"}, &stdout, &stderr, os.Environ()))
	require.Equal(t, "--endpoint=api.aliyuncs.alibaba\n--body\n--endpoint=payload.aliyuncs.com\n--\n--endpoint=opaque.aliyuncs.com\n", stdout.String())
}
