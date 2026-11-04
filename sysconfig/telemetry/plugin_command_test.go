package telemetry

import (
	"encoding/json"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/stretchr/testify/require"
)

func TestGoPluginCommandUsesConfirmedPaths(t *testing.T) {
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
			}{
				{"reported example", []string{"demo", "list", "--tag", "1", "2", "--vvv", "r"}, "demo list"},
				{"multiple values", []string{"demo", "List", "--tag", "private-sentinel-one", "private-sentinel-two", "--vvv", "private-sentinel-three"}, "demo List"},
				{"repeated flag", []string{"demo", "List", "--tag", "private-sentinel-one", "--tag", "private-sentinel-two", "--vvv", "private-sentinel-three"}, "demo List"},
				{"mixed case and repeated inline flags", []string{"demo", "List", "--Tag=private-sentinel-one", "--tag:private-sentinel-two", "--Tag=private-sentinel-three", "-t=private-sentinel-four", "-t:private-sentinel-five"}, "demo List"},
				{"inline multiple values", []string{"demo", "Resource", "List", "--tag=private-sentinel-one", "private-sentinel-two", "--vvv:private-sentinel-three"}, "demo Resource List"},
				{"flag before subcommand", []string{"demo", "Resource", "--tag", "private-sentinel-one", "private-sentinel-two", "List"}, "demo Resource"},
				{"short flag multiple values", []string{"demo", "List", "-t", "private-sentinel-one", "private-sentinel-two", "--vvv", "private-sentinel-three"}, "demo List"},
				{"end of options", []string{"demo", "Resource", "List", "--", "private-sentinel-one", "--private-sentinel-flag"}, "demo Resource List"},
				{"root flag", []string{"demo", "--tag", "private-sentinel-one", "List"}, "demo"},
				{"hierarchy without flags", []string{"demo", "Resource", "Policy", "List"}, "demo Resource Policy List"},
				{"malformed upload with path", []string{"demo", "upload", "/private-sentinel/customer.csv"}, "demo upload"},
				{"malformed upload with scalar", []string{"demo", "upload", "private-sentinel-customer"}, "demo upload"},
				{"unrecognized command", []string{"demo", "private-sentinel", "Resource", "List"}, "demo"},
				{"unrecognized nested command", []string{"demo", "Resource", "private-sentinel", "Policy", "List"}, "demo Resource"},
				{"intermediate group", []string{"demo", "Resource", "Policy"}, "demo Resource Policy"},
				{"registered product alias", []string{"DemoAlias", "Resource", "Policy", "List", "private-sentinel"}, "DemoAlias Resource Policy List"},
				{"root only", []string{"demo"}, "demo"},
			} {
				t.Run(test.name, func(t *testing.T) {
					event, err := BuildEvent(BuildInput{
						Scope: Scope{
							Active: true, Pipeline: PipelinePublic, CommandToken: test.args[0],
							PluginName: "aliyun-cli-demo", PluginType: pluginType,
							PluginCommandPaths: []string{"list", "resource list", "resource policy list", "upload"},
						},
						RootArgs: test.args, InstanceID: "test-installation",
					})
					require.NoError(t, err)
					require.Equal(t, test.command, event.Command)
					require.Empty(t, event.Params, "manifest command paths do not confirm plugin flags")
					uploaded := newUploadEvent(event)
					require.Equal(t, test.command, uploaded.Attributes["command"])
					require.NotContains(t, uploaded.Attributes, "parameterNames")
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

func TestGoPluginCommandWithoutDefinitionsKeepsOnlyRoot(t *testing.T) {
	for _, paths := range [][]string{nil, {}, {"", " "}} {
		event, err := BuildEvent(BuildInput{
			Scope: Scope{
				Active: true, Pipeline: PipelinePublic, CommandToken: "demo",
				PluginType: plugin.PluginTypeGo, PluginCommandPaths: paths,
			},
			RootArgs: []string{"demo", "upload", "/private-sentinel/customer.csv"}, InstanceID: "test-installation",
		})
		require.NoError(t, err)
		require.Equal(t, "demo", event.Command)
		uploaded, err := json.Marshal(newUploadEvent(event))
		require.NoError(t, err)
		require.NotContains(t, string(uploaded), "private-sentinel")
	}
}

func TestFailedGoPluginCommandOmitsUnrecognizedInput(t *testing.T) {
	exitCode := 1
	event, err := BuildEvent(BuildInput{
		Scope: Scope{
			Active: true, Pipeline: PipelinePublic, CommandToken: "demo",
			PluginType: plugin.PluginTypeGo, PluginCommandPaths: []string{"upload"},
		},
		RootArgs:   []string{"demo", "upload", "/private-sentinel/customer.csv"},
		InstanceID: "test-installation", ExitCode: &exitCode,
	})
	require.NoError(t, err)
	require.Equal(t, "Failure", event.Result)
	for _, payload := range []any{event, newUploadEvent(event)} {
		data, err := json.Marshal(payload)
		require.NoError(t, err)
		require.NotContains(t, string(data), "private-sentinel")
	}
}
