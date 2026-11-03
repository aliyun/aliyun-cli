package telemetry

import (
	"encoding/json"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/stretchr/testify/require"
)

func TestMetaCommandUsesOnlyCurrentResolution(t *testing.T) {
	t.Cleanup(resetExecutionState)
	for _, resolved := range []bool{false, true} {
		name := "unresolved"
		want := "MetaAlias"
		if resolved {
			name, want = "resolved", "MetaAlias describe-thing"
		}
		t.Run(name, func(t *testing.T) {
			args := []string{"MetaAlias", "describe-thing", "/private-customer.csv", "private-customer", "--Tag", "private-one", "--Tag", "private-two"}
			beginCommandExecution("", Scope{
				CommandToken: "MetaAlias", Pipeline: PipelineInner,
				PluginName: "aliyun-cli-meta", PluginType: plugin.PluginTypeMeta,
				// A manifest declaration alone must not confirm the selected
				// version's API, even if it happens to contain the input tokens.
				PluginCommandPaths: []string{"describe-thing /private-customer.csv private-customer"},
			}, args, false)
			if resolved {
				SetResolvedMetaCommand("metaalias", "describe-thing")
			}
			exitCode := 1
			event, err := BuildEvent(BuildInput{
				Scope: activeExecution.scope, RootArgs: args, InstanceID: "fixture-installation", ExitCode: &exitCode,
			})
			require.NoError(t, err)
			require.Equal(t, want, event.Command)
			require.Equal(t, CommandTypePluginMeta, event.CommandType)
			require.Equal(t, "Failure", event.Result)
			require.Equal(t, []string{"--Tag", "--Tag"}, event.Params)
			for _, payload := range []any{event, newUploadEvent(event)} {
				data, err := json.Marshal(payload)
				require.NoError(t, err)
				for _, private := range []string{"/private-customer.csv", "private-customer", "private-one", "private-two"} {
					require.NotContains(t, string(data), private)
				}
			}
		})
	}
}

func TestMetaResolutionMarkerDoesNotChangeOtherExecutions(t *testing.T) {
	t.Cleanup(resetExecutionState)
	resetExecutionState()
	SetResolvedMetaCommand("meta", "describe-thing")
	require.Nil(t, activeExecution)
	for _, test := range []struct {
		name, pluginType, product, command string
	}{
		{"Go route", plugin.PluginTypeGo, "meta", "describe-thing"},
		{"native route", "", "meta", "describe-thing"},
		{"other product", plugin.PluginTypeMeta, "other", "describe-thing"},
		{"path instead of command", plugin.PluginTypeMeta, "meta", "/private-customer.csv"},
		{"multiple tokens", plugin.PluginTypeMeta, "meta", "describe-thing private-customer"},
	} {
		t.Run(test.name, func(t *testing.T) {
			beginCommandExecution("", Scope{CommandToken: "meta", PluginType: test.pluginType}, nil, false)
			SetResolvedMetaCommand(test.product, test.command)
			require.Empty(t, activeExecution.scope.ResolvedCommand)
		})
	}
	beginCommandExecution("", Scope{CommandToken: "Meta", PluginType: plugin.PluginTypeMeta}, nil, false)
	SetResolvedMetaCommand("meta", "DescribeThing")
	require.Equal(t, "DescribeThing", activeExecution.scope.ResolvedCommand)
	_, ok := takeCommandExecutionForEnd()
	require.True(t, ok)
	SetResolvedMetaCommand("meta", "another-command")
	require.Equal(t, "DescribeThing", activeExecution.scope.ResolvedCommand, "a completed command cannot be rewritten")
}

func TestMetaResolutionIsClearedWhenRouteChanges(t *testing.T) {
	t.Cleanup(resetExecutionState)
	beginCommandExecution("", Scope{CommandToken: "meta", PluginType: plugin.PluginTypeMeta}, nil, false)
	SetResolvedMetaCommand("meta", "previous-command")
	SetPluginRoute(&plugin.LocalPlugin{Name: "aliyun-cli-meta", Type: plugin.PluginTypeMeta})
	require.Empty(t, activeExecution.scope.ResolvedCommand)
	SetResolvedMetaCommand("meta", "current-command")
	SetOpenAPILegacyRoute()
	require.Empty(t, activeExecution.scope.ResolvedCommand)
	SetResolvedMetaCommand("meta", "current-command")
	require.Empty(t, activeExecution.scope.ResolvedCommand)
}
