package mcpproxy

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

func TestMCPUpstreamEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	proxy := &MCPProxy{RegionType: RegionCN}
	r := httptest.NewRequest("POST", "/mcp/service?next=a%2Bb", nil)
	upstream, err := proxy.buildUpstreamRequest(r, "token")
	require.NoError(t, err)
	require.Equal(t, "openapi-mcp.cn-hangzhou.aliyuncs.alibaba", upstream.URL.Host)
	require.Equal(t, "next=a%2Bb", upstream.URL.RawQuery)
	require.Equal(t, "/mcp/service", upstream.URL.Path)
	t.Setenv(endpoint.EnvSuffix, "a/b")
	_, err = proxy.buildUpstreamRequest(r, "token")
	require.ErrorContains(t, err, endpoint.EnvSuffix)
	_, _, err = (&credentialAuth{}).exchange(time.Now())
	require.ErrorContains(t, err, endpoint.EnvSuffix)
}
