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
	t.Setenv("NO_COLOR", "1")
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	for _, command := range []string{"ls", "cp"} {
		help := "\nRun 'aliyun oss " + command + " --help' for supported oss options."
		for _, tc := range []struct {
			flag, want string
		}{
			{"--header=private-header-value", "--header is not an oss command option." + help},
			{"--dryrun", "--dryrun is not an oss command option." + help},
			{"--versin", "invalid option --versin for oss commands, did you mean --version?"},
			{"--zzzzzz", "invalid option --zzzzzz for oss commands." + help},
		} {
			for _, format := range []string{"human", "json"} {
				t.Run(command+"/"+tc.flag+"/"+format, func(t *testing.T) {
					var stdout, stderr bytes.Buffer
					root := newRootCommand(config.NewProfile("default"), &stdout)
					ctx := cli.NewCommandContext(&stdout, &stderr)
					ctx.EnterCommand(root)
					args := []string{"oss", command, tc.flag}
					if format == "json" {
						args = append(args, "--cli-output=json")
					}
					root.Execute(ctx, args)
					require.Empty(t, stdout.String())
					require.NotContains(t, stderr.String(), "private-header-value")
					if format == "json" {
						var envelope struct {
							Message string `json:"message"`
						}
						require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope))
						require.Equal(t, tc.want, envelope.Message)
					} else {
						require.Equal(t, "ERROR: "+tc.want+"\n", stderr.String())
					}
				})
			}
		}
	}
}
