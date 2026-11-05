package saectl

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

type endpointSuffixTransport func(*http.Request) (*http.Response, error)

func (f endpointSuffixTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type endpointSuffixBody struct {
	io.Reader
	closed bool
}

func (b *endpointSuffixBody) Close() error { b.closed = true; return nil }

func TestSaeCtlHTTPRequestsUseEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	calls := 0
	var body *endpointSuffixBody
	http.DefaultTransport = endpointSuffixTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.True(t, strings.HasSuffix(r.URL.Hostname(), ".aliyuncs.alibaba"), r.URL.String())
		body = &endpointSuffixBody{Reader: strings.NewReader("1.2.3\n")}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body, Request: r}, nil
	})
	version, err := GetLatestSaeCtlVersion()
	require.NoError(t, err)
	require.Equal(t, "1.2.3", version)
	require.True(t, body.closed)
	t.Setenv(endpoint.EnvSuffix, "invalid.example")
	_, err = GetLatestSaeCtlVersion()
	require.ErrorContains(t, err, endpoint.EnvSuffix)
	require.Equal(t, 1, calls)
}

func TestSaeCtlDownloadRewritesEndpointAndClosesFailureResponse(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	previous := httpGetFunc
	t.Cleanup(func() { httpGetFunc = previous })
	body := &endpointSuffixBody{Reader: strings.NewReader("unavailable")}
	httpGetFunc = func(address string) (*http.Response, error) {
		require.Equal(t, "https://assets.oss.aliyuncs.alibaba/tool?q=old.aliyuncs.com", address)
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: body}, nil
	}
	err := DownloadAndExtract("https://assets.oss.aliyuncs.com/tool?q=old.aliyuncs.com", "", "", "")
	require.ErrorContains(t, err, "503")
	require.True(t, body.closed)
}
