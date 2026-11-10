package cms2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

func TestCMSPrepareEnvUsesEndpointSuffix(t *testing.T) {
	clearAgentEnvs(t)
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	path := filepath.Join(t.TempDir(), "config.json")
	content := []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"id","access_key_secret":"secret","region_id":"cn-hangzhou"}]}`)
	require.NoError(t, os.WriteFile(path, content, 0600))
	ctx, _, _ := newTestCtx()
	config.AddFlags(ctx.Flags())
	ctx.SetInConfigureMode(true)
	for name, value := range map[string]string{config.ConfigurePathFlagName: path, config.EndpointFlagName: "https://cms.cn-hangzhou.aliyuncs.com:443"} {
		flag := ctx.Flags().Get(name)
		flag.SetValue(value)
		flag.SetAssigned(true)
	}
	c := NewContext(ctx)
	require.NoError(t, c.PrepareEnv(nil))
	require.Equal(t, "https://cms.cn-hangzhou.aliyuncs.alibaba:443", c.envMap["ALIBABA_CLOUD_CMS_ENDPOINT"])
	stored, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, content, stored)
	t.Setenv(endpoint.EnvSuffix, "invalid.example")
	require.ErrorContains(t, NewContext(nil).PrepareEnv(nil), endpoint.EnvSuffix)
}
