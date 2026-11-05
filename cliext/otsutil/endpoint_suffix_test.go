package otsutil

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

func TestOtsUtilHTTPRequestsUseEndpointSuffix(t *testing.T) {
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
	version, err := GetLatestOtsUtilVersion()
	require.NoError(t, err)
	require.Equal(t, "1.2.3", version)
	require.True(t, body.closed)
	t.Setenv(endpoint.EnvSuffix, "invalid.example")
	_, err = GetLatestOtsUtilVersion()
	require.ErrorContains(t, err, endpoint.EnvSuffix)
	require.Equal(t, 1, calls)
}
