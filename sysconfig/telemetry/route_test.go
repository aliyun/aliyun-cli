package telemetry

import (
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/stretchr/testify/require"
)

func TestLegacyRouteOverridesInstalledPluginCandidate(t *testing.T) {
	resetExecutionState()
	t.Cleanup(resetExecutionState)
	SetOpenAPILegacyRoute()
	require.Nil(t, activeExecution, "a route marker must not start collection")
	for _, pluginType := range []string{plugin.PluginTypeGo, plugin.PluginTypeMeta} {
		for _, operation := range []string{"ListProject", "GET", "get"} {
			t.Run(pluginType+"/"+operation, func(t *testing.T) {
				args := []string{"sls", operation, "--size", "1"}
				beginCommandExecution("", Scope{
					Active: true, Pipeline: PipelineInner, CommandToken: "sls",
					PluginName: "aliyun-cli-sls", PluginVer: "1.2.3", PluginType: pluginType,
				}, args, false)
				SetOpenAPIRuntimeVersion("candidate-version")
				SetOpenAPILegacyRoute()
				event, err := BuildEvent(BuildInput{
					Scope: activeExecution.scope, RootArgs: args,
					OpenAPIRuntimeVersion: activeExecution.openAPIRuntimeVersion,
				})
				require.NoError(t, err)
				require.Equal(t, "sls "+operation, event.Command)
				require.Equal(t, CommandTypeOpenAPILegacy, event.CommandType)
				require.Equal(t, PipelinePublic, event.Pipeline)
				require.Empty(t, event.Plugin)
				require.Empty(t, event.OpenAPIRuntimeVersion)
				uploaded := newUploadEvent(event)
				require.Equal(t, CommandTypeOpenAPILegacy, uploaded.Attributes["commandType"])
				require.NotContains(t, uploaded.Attributes, "plugin")
				require.NotContains(t, uploaded.Attributes, "pipeline")
			})
		}
	}
	resetExecutionState()
	beginCommandExecution("", Scope{CommandToken: "demo", PluginType: plugin.PluginTypeGo}, nil, false)
	_, ok := takeCommandExecutionForEnd()
	require.True(t, ok)
	SetOpenAPILegacyRoute()
	require.Equal(t, plugin.PluginTypeGo, activeExecution.scope.PluginType, "a completed execution must not change routes")
}
