package config

import (
	"io"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

func TestSTSEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	for _, tc := range []struct {
		profile Profile
		want    string
	}{
		{Profile{}, "sts.aliyuncs.alibaba"},
		{Profile{StsRegion: "cn-hangzhou"}, "sts.cn-hangzhou.aliyuncs.alibaba"},
		{Profile{StsEndpoint: "https://sts-vpc.cn-hangzhou.aliyuncs.com:443/"}, "sts-vpc.cn-hangzhou.aliyuncs.alibaba:443"},
		{Profile{StsEndpoint: "localhost:8080"}, "localhost:8080"},
	} {
		require.Equal(t, tc.want, resolveSTSEndpoint(&tc.profile))
	}
	t.Setenv(endpoint.EnvSuffix, "evil.com")
	_, err := (&Profile{Mode: External, ProcessCommand: "must-not-run"}).GetCredential(nil, nil)
	require.ErrorContains(t, err, endpoint.EnvSuffix)
}

func TestPluginRuntimeEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	profile := &Profile{Mode: Anonymous, RegionId: "cn-hangzhou", Endpoint: "ecs.aliyuncs.com", StsEndpoint: "https://sts.aliyuncs.com:443"}
	envs, err := profile.GetRuntimeEnv(nil)
	require.NoError(t, err)
	require.Equal(t, "ecs.aliyuncs.alibaba", envs["ALIBABA_CLOUD_ENDPOINT"])
	require.Equal(t, "https://sts.aliyuncs.alibaba:443", envs["ALIBABA_CLOUD_STS_ENDPOINT"])
	require.Equal(t, "ecs.aliyuncs.com", profile.Endpoint)
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	AddFlags(ctx.Flags())
	EndpointFlag(ctx.Flags()).SetValue("override.aliyuncs.com")
	EndpointFlag(ctx.Flags()).SetAssigned(true)
	require.Equal(t, "override.aliyuncs.alibaba", BuildBaselineEnv(ctx)["ALIBABA_CLOUD_ENDPOINT"])
}
