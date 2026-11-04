package util

import (
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

func TestOpenBrowserRejectsInvalidEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "invalid.example")
	require.ErrorContains(t, OpenBrowser("https://signin.aliyuncs.com/"), endpoint.EnvSuffix)
}
