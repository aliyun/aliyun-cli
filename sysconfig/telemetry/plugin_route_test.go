package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/stretchr/testify/require"
)

func writePluginRouteManifest(t *testing.T, dir string, local plugin.LocalPlugin) {
	t.Helper()
	data, err := json.Marshal(plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{local.Name: local}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600))
}

func readPluginRouteEvent(t *testing.T, dir, pipeline string) Event {
	t.Helper()
	cache := GetCacheDir(dir, pipeline)
	entries, err := os.ReadDir(cache)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	data, err := os.ReadFile(filepath.Join(cache, entries[0].Name()))
	require.NoError(t, err)
	var event Event
	require.NoError(t, json.Unmarshal(data, &event))
	return event
}

func TestSelectedPluginRouteRefreshesFirstInstallation(t *testing.T) {
	for _, test := range []struct {
		name, pluginType, commandType, pipeline string
		inner                                   bool
	}{
		{"public_go", plugin.PluginTypeGo, CommandTypePluginGo, PipelinePublic, false},
		{"inner_go", plugin.PluginTypeGo, CommandTypePluginGo, PipelineInner, true},
		{"legacy_default_go", "", CommandTypePluginGo, PipelineInner, true},
		{"public_meta", plugin.PluginTypeMeta, CommandTypePluginMeta, PipelinePublic, false},
		{"inner_meta", plugin.PluginTypeMeta, CommandTypePluginMeta, PipelineInner, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("USERPROFILE", os.Getenv("HOME"))
			t.Setenv(TelemetryDisableEnv, "0")
			t.Setenv(TelemetryLogDisableEnv, "1")
			pluginsDir := t.TempDir()
			t.Setenv(plugin.EnvPluginsDir, pluginsDir)
			t.Cleanup(resetExecutionState)
			dir := collectionPanicConfigDir(t)
			args := []string{"RouteFixture", "list-items", "--Tag", "private-value", "--Tag", "private-second"}
			Begin(BeginInput{ConfigDir: dir, CommandToken: args[0], CommandArgs: args})
			require.NotNil(t, activeExecution)
			require.Equal(t, PipelinePublic, activeExecution.scope.Pipeline)
			require.Empty(t, activeExecution.scope.PluginName)
			require.Empty(t, activeExecution.scope.PluginType)

			// The host installs the package only after Begin's initial lookup.
			local := plugin.LocalPlugin{
				Name: "aliyun-cli-routefixture", Version: "1.2.3", Command: "routefixture",
				Type: test.pluginType, Inner: test.inner,
			}
			writePluginRouteManifest(t, pluginsDir, local)
			selected, err := plugin.LookupLocalPlugin(args[0])
			require.NoError(t, err)
			require.NotNil(t, selected)
			SetOpenAPIRuntimeVersion("stale-candidate")
			SetPluginRoute(selected)
			require.Equal(t, args[0], activeExecution.scope.CommandToken)
			require.Empty(t, activeExecution.openAPIRuntimeVersion)
			if test.pluginType == plugin.PluginTypeMeta {
				SetOpenAPIRuntimeVersion("9.8.7")
			}
			End(nil)

			event := readPluginRouteEvent(t, dir, test.pipeline)
			require.Equal(t, test.commandType, event.CommandType)
			require.Equal(t, test.pipeline, event.Pipeline)
			require.Equal(t, "aliyun-cli-routefixture@1.2.3", event.Plugin)
			require.Equal(t, "RouteFixture list-items", event.Command)
			require.Equal(t, []string{"--Tag", "--Tag"}, event.Params)
			if test.pluginType == plugin.PluginTypeMeta {
				require.Equal(t, "9.8.7", event.OpenAPIRuntimeVersion)
			} else {
				require.Empty(t, event.OpenAPIRuntimeVersion)
			}
			uploaded, err := json.Marshal(newUploadEvent(event))
			require.NoError(t, err)
			require.NotContains(t, string(uploaded), "private-value")
			require.NotContains(t, string(uploaded), "private-second")
			otherPipeline := PipelineInner
			if test.inner {
				otherPipeline = PipelinePublic
			}
			require.NoDirExists(t, GetCacheDir(dir, otherPipeline))
		})
	}
}

func TestSelectedPluginRouteKeepsExecutionSnapshot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	t.Setenv(TelemetryDisableEnv, "0")
	t.Setenv(TelemetryLogDisableEnv, "1")
	pluginsDir := t.TempDir()
	t.Setenv(plugin.EnvPluginsDir, pluginsDir)
	t.Cleanup(resetExecutionState)
	dir := collectionPanicConfigDir(t)
	args := []string{"RouteFixture", "list-items"}
	Begin(BeginInput{ConfigDir: dir, CommandToken: args[0], CommandArgs: args})
	selected := plugin.LocalPlugin{
		Name: "aliyun-cli-routefixture", Version: "1.2.3", Command: "routefixture",
		Type: plugin.PluginTypeGo, Inner: true,
	}
	SetPluginRoute(&selected)
	// An unrelated install/update after route selection must not rewrite the
	// provenance of the command whose executable has already been selected.
	selected.Version, selected.Type, selected.Inner = "9.9.9", plugin.PluginTypeMeta, false
	writePluginRouteManifest(t, pluginsDir, selected)
	End(nil)
	event := readPluginRouteEvent(t, dir, PipelineInner)
	require.Equal(t, CommandTypePluginGo, event.CommandType)
	require.Equal(t, PipelineInner, event.Pipeline)
	require.Equal(t, "aliyun-cli-routefixture@1.2.3", event.Plugin)
	require.NoDirExists(t, GetPublicCacheDir(dir))
}

func TestPluginRouteMarkerDoesNotStartOrReopenCollection(t *testing.T) {
	t.Cleanup(resetExecutionState)
	resetExecutionState()
	local := &plugin.LocalPlugin{Name: "aliyun-cli-demo", Version: "1.2.3", Type: plugin.PluginTypeGo, Inner: true}
	SetPluginRoute(local)
	require.Nil(t, activeExecution)
	beginCommandExecution("", Scope{CommandToken: "demo", Pipeline: PipelinePublic}, []string{"demo", "list"}, false)
	SetOpenAPIRuntimeVersion("candidate-runtime")
	scope := activeExecution.scope
	SetPluginRoute(nil)
	require.Equal(t, scope, activeExecution.scope)
	require.Equal(t, "candidate-runtime", activeExecution.openAPIRuntimeVersion)
	_, ok := takeCommandExecutionForEnd()
	require.True(t, ok)
	SetPluginRoute(local)
	require.Equal(t, scope, activeExecution.scope)
	require.Equal(t, "candidate-runtime", activeExecution.openAPIRuntimeVersion)
}

func TestSelectedPluginAndLegacyRouteMarkersFollowFinalOwner(t *testing.T) {
	t.Cleanup(resetExecutionState)
	args := []string{"SLs", "ListProject", "--size", "1"}
	beginCommandExecution("", Scope{CommandToken: args[0], Pipeline: PipelinePublic}, args, false)
	local := &plugin.LocalPlugin{Name: "aliyun-cli-sls", Version: "1.2.3", Type: plugin.PluginTypeGo, Inner: true}
	SetPluginRoute(local)
	SetOpenAPILegacyRoute()
	require.Equal(t, CommandTypeOpenAPILegacy, activeExecution.scope.CommandType)
	require.Equal(t, PipelinePublic, activeExecution.scope.Pipeline)
	require.Empty(t, activeExecution.scope.PluginName)
	SetPluginRoute(local)
	require.Equal(t, CommandTypePluginGo, activeExecution.scope.CommandType)
	require.Equal(t, PipelineInner, activeExecution.scope.Pipeline)
	require.Equal(t, args[0], activeExecution.scope.CommandToken)
}
