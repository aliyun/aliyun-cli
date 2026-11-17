package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

func TestEndpointSuffixDryRunCommands(t *testing.T) {
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	clearAgentDetectionEnv(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"current":"default","profiles":[{"name":"default","mode":"External","process_command":"/must-not-run-for-dry-run","region_id":"cn-hangzhou","language":"en"}]}`), 0600))
	for _, suffix := range []string{"", "aliyuncs.alibaba"} {
		for _, command := range []string{"DescribeRegions", "describe-regions"} {
			t.Run(suffix+"/"+command, func(t *testing.T) {
				t.Setenv(endpoint.EnvSuffix, suffix)
				args := []string{"ecs", command, "--cli-dry-run-json", "--config-path", configPath}
				oldArgs := os.Args
				os.Args = append([]string{"aliyun"}, args...)
				t.Cleanup(func() { os.Args = oldArgs })
				profile, err := config.LoadProfile(configPath, "default")
				require.NoError(t, err)
				var stdout, stderr bytes.Buffer
				root := newRootCommand(profile, &stdout)
				ctx := newCommandContext(&stdout, &stderr)
				ctx.EnterCommand(root)
				root.Execute(ctx, args)
				require.Empty(t, stderr.String())
				var output struct {
					Endpoint string `json:"endpoint"`
				}
				require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
				want := ".aliyuncs.com"
				if suffix != "" {
					want = ".aliyuncs.alibaba"
				}
				require.True(t, strings.HasSuffix(output.Endpoint, want), "endpoint: %s", output.Endpoint)
			})
		}
	}
}
