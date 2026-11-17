package lib

import (
	"net/url"
	"testing"

	cloudendpoint "github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestOSSEndpointSuffix(t *testing.T) {
	t.Setenv(cloudendpoint.EnvSuffix, "aliyuncs.alibaba")
	cmd := localOSSCommand(t, "https://oss-cn-hangzhou.aliyuncs.com")
	cmd.options[OptionForcePathStyle] = new(bool)
	client, err := cmd.ossClient("bucket")
	require.NoError(t, err)
	require.Equal(t, "https://oss-cn-hangzhou.aliyuncs.alibaba", client.Config.Endpoint)
	bucket, err := client.Bucket("bucket")
	require.NoError(t, err)
	signedURL, err := bucket.SignURL("object", oss.HTTPGet, 60)
	require.NoError(t, err)
	parsedOSS, err := url.Parse(signedURL)
	require.NoError(t, err)
	require.Equal(t, "bucket.oss-cn-hangzhou.aliyuncs.alibaba", parsedOSS.Host)
	require.NotEmpty(t, parsedOSS.Query().Get("Signature"))
	stsURL, err := (&Client{AccessKeyId: "id", AccessKeySecret: "secret", RoleArn: "role", SessionName: "session"}).generateSignedURL(3600)
	require.NoError(t, err)
	parsed, err := url.Parse(stsURL)
	require.NoError(t, err)
	require.Equal(t, "sts.aliyuncs.alibaba", parsed.Host)
	require.NotEmpty(t, parsed.Query().Get("Signature"))
	t.Setenv(cloudendpoint.EnvSuffix, "host:8080")
	_, err = cmd.ossClient("bucket")
	require.ErrorContains(t, err, cloudendpoint.EnvSuffix)
	_, err = (&Client{}).generateSignedURL(3600)
	require.ErrorContains(t, err, cloudendpoint.EnvSuffix)
}
