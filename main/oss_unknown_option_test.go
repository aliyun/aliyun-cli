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
		cases := []struct {
			flag, want string
		}{
			{"--header=private-header-value", "--header is not an oss command option." + help},
			{"--dryrun", "--dryrun is not an oss command option." + help},
			{"--versin", "invalid option --versin for oss " + command + "." + help},
			{"--zzzzzz", "invalid option --zzzzzz for oss " + command + "." + help},
		}
		if command == "ls" {
			cases = append(cases, []struct{ flag, want string }{
				{"--vers", "invalid option --vers for oss ls, did you mean --version-id-marker?"},
				{"--markr", "invalid option --markr for oss ls, did you mean --marker?"},
				{"--version", "invalid option --version for oss ls." + help},
				{"--version-id", "invalid option --version-id for oss ls." + help},
			}...)
		} else {
			cases = append(cases, []struct{ flag, want string }{
				{"--vers", "invalid option --vers for oss cp, did you mean --version-id?"},
				{"--recursiv", "invalid option --recursiv for oss cp, did you mean --recursive?"},
			}...)
		}
		for _, tc := range cases {
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
