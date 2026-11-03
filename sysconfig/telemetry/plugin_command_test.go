package telemetry

import (
	"encoding/json"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/stretchr/testify/require"
)

func TestGoPluginCommandStopsAtFirstFlag(t *testing.T) {
	for _, pluginType := range []string{plugin.PluginTypeGo, ""} {
		name := pluginType
		if name == "" {
			name = "legacy_default_go"
		}
		t.Run(name, func(t *testing.T) {
			for _, test := range []struct {
				name    string
				args    []string
				command string
				params  []string
			}{
				{"reported example", []string{"demo", "list", "--tag", "1", "2", "--vvv", "r"}, "demo list", []string{"--tag", "--vvv"}},
				{"multiple values", []string{"demo", "List", "--tag", "private-sentinel-one", "private-sentinel-two", "--vvv", "private-sentinel-three"}, "demo List", []string{"--tag", "--vvv"}},
				{"repeated flag", []string{"demo", "List", "--tag", "private-sentinel-one", "--tag", "private-sentinel-two", "--vvv", "private-sentinel-three"}, "demo List", []string{"--tag", "--tag", "--vvv"}},
				{"mixed case and repeated inline flags", []string{"demo", "List", "--Tag=private-sentinel-one", "--tag:private-sentinel-two", "--Tag=private-sentinel-three", "-t=private-sentinel-four", "-t:private-sentinel-five"}, "demo List", []string{"--Tag", "--tag", "--Tag", "-t", "-t"}},
				{"inline multiple values", []string{"demo", "Resource", "List", "--tag=private-sentinel-one", "private-sentinel-two", "--vvv:private-sentinel-three"}, "demo Resource List", []string{"--tag", "--vvv"}},
				{"flag before subcommand", []string{"demo", "Resource", "--tag", "private-sentinel-one", "private-sentinel-two", "List"}, "demo Resource", []string{"--tag"}},
				{"short flag multiple values", []string{"demo", "List", "-t", "private-sentinel-one", "private-sentinel-two", "--vvv", "private-sentinel-three"}, "demo List", []string{"-t", "--vvv"}},
				{"end of options", []string{"demo", "Resource", "List", "--", "private-sentinel-one", "--private-sentinel-flag"}, "demo Resource List", nil},
				{"root flag", []string{"demo", "--tag", "private-sentinel-one", "List"}, "demo", []string{"--tag"}},
				{"hierarchy without flags", []string{"demo", "Resource", "Policy", "List"}, "demo Resource Policy List", nil},
			} {
				t.Run(test.name, func(t *testing.T) {
					event, err := BuildEvent(BuildInput{
						Scope: Scope{
							Active: true, Pipeline: PipelinePublic, CommandToken: "demo",
							PluginName: "aliyun-cli-demo", PluginType: pluginType,
						},
						RootArgs: test.args, InstanceID: "test-installation",
					})
					require.NoError(t, err)
					require.Equal(t, test.command, event.Command)
					require.Equal(t, test.params, event.Params)
					uploaded := newUploadEvent(event)
					require.Equal(t, test.command, uploaded.Attributes["command"])
					if len(test.params) == 0 {
						require.NotContains(t, uploaded.Attributes, "parameterNames")
					} else {
						require.Equal(t, test.params, uploaded.Attributes["parameterNames"])
					}
					for _, payload := range []any{event, uploaded} {
						encoded, err := json.Marshal(payload)
						require.NoError(t, err)
						require.NotContains(t, string(encoded), "private-sentinel")
					}
				})
			}
		})
	}
}
