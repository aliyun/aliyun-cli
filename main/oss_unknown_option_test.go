package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/require"
)

func TestOSSUnknownOptionsThroughRoot(t *testing.T) {
	clearAgentDetectionEnv(t)
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	for _, tc := range []struct {
		flag, want string
	}{
		{"--header=private-header-value", "flag --header is a global aliyun CLI flag for OpenAPI commands, not an oss command option."},
		{"--dryrun", "flag --dryrun is a global aliyun CLI flag for OpenAPI commands, not an oss command option."},
		{"--versin", "invalid flag --versin for oss commands, did you mean --version?"},
		{"--zzzzzz", "invalid flag --zzzzzz for oss commands.\nRun 'aliyun oss --help' for supported oss options."},
	} {
		for _, format := range []string{"human", "json"} {
			t.Run(tc.flag+"/"+format, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				root := newRootCommand(config.NewProfile("default"), &stdout)
				ctx := cli.NewCommandContext(&stdout, &stderr)
				ctx.EnterCommand(root)
				args := []string{"oss", "ls", tc.flag}
				if format == "json" {
					args = append(args, "--cli-output=json")
				}
				root.Execute(ctx, args)
				require.Empty(t, stdout.String())
				require.NotContains(t, stderr.String(), "private-header-value")
				message := stderr.String()
				if format == "json" {
					var envelope struct {
						Message string `json:"message"`
					}
					require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope))
					message = envelope.Message
				}
				require.Contains(t, message, tc.want)
			})
		}
	}
}
