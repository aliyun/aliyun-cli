package otsutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

func TestPluginExecutionForwardsEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	configPath := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"id","access_key_secret":"secret","region_id":"cn-hangzhou"}]}`), 0600))
	flag := ctx.Flags().Get(config.ConfigurePathFlagName)
	flag.SetValue(configPath)
	flag.SetAssigned(true)
	c := NewContext(ctx)
	c.execFilePath = "test-plugin"
	c.configPath = t.TempDir()
	previous := execCommandFunc
	t.Cleanup(func() { execCommandFunc = previous })
	var command *exec.Cmd
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		require.Equal(t, c.execFilePath, name)
		require.Equal(t, []string{"list", "--endpoint", "https://service.aliyuncs.alibaba:443", "--", "-eopaque.aliyuncs.com"}, args)
		command = exec.Command(os.Args[0], "-test.run=^$")
		return command
	}
	args := []string{"list", "--endpoint", "https://service.aliyuncs.com:443", "--", "-eopaque.aliyuncs.com"}
	require.NoError(t, c.ExecuteTablestoreCli(args))
	require.Equal(t, "https://service.aliyuncs.com:443", args[2])
	require.Same(t, os.Stdin, command.Stdin)
	require.Equal(t, c.configPath, command.Dir)
}
