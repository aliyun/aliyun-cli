package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeVersionIsRecordedOnlyForActiveExecution(t *testing.T) {
	resetExecutionState()
	t.Cleanup(resetExecutionState)
	SetOpenAPIRuntimeVersion("ignored-before-begin")
	require.Nil(t, activeExecution)
	t.Setenv(TelemetryDisableEnv, "0")
	dir := collectionPanicConfigDir(t)
	Begin(BeginInput{ConfigDir: dir, CommandToken: "ecs", CommandArgs: []string{"ecs", "describe-regions"}})
	SetOpenAPIRuntimeVersion("9.8.7-build")
	End(nil)
	SetOpenAPIRuntimeVersion("ignored-after-end")
	entries, err := os.ReadDir(GetPublicCacheDir(dir))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	data, err := os.ReadFile(filepath.Join(GetPublicCacheDir(dir), entries[0].Name()))
	require.NoError(t, err)
	var event Event
	require.NoError(t, json.Unmarshal(data, &event))
	require.Equal(t, "9.8.7-build", event.OpenAPIRuntimeVersion)
	require.Equal(t, "9.8.7-build", newUploadEvent(event).Attributes["openapiRuntimeVersion"])
	require.Equal(t, "9.8.7-build", activeExecution.openAPIRuntimeVersion)

	// A new invocation must not inherit a runtime used by a previous command.
	Begin(BeginInput{ConfigDir: dir, CommandToken: "version", CommandArgs: []string{"version"}})
	require.Empty(t, activeExecution.openAPIRuntimeVersion)
}

func TestRuntimeVersionUploadContract(t *testing.T) {
	for _, version := range []string{"", "9.8.7-build"} {
		t.Run(version, func(t *testing.T) {
			event, err := BuildEvent(BuildInput{OpenAPIRuntimeVersion: version})
			require.NoError(t, err)
			data, err := json.Marshal(event)
			require.NoError(t, err)
			var stored map[string]any
			require.NoError(t, json.Unmarshal(data, &stored))
			uploaded := newUploadEvent(event)
			if version == "" {
				require.NotContains(t, stored, "openapi_runtime_version")
				require.NotContains(t, uploaded.Attributes, "openapiRuntimeVersion")
			} else {
				require.Equal(t, version, stored["openapi_runtime_version"])
				require.Equal(t, version, uploaded.Attributes["openapiRuntimeVersion"])
			}
		})
	}
	var oldEvent Event
	require.NoError(t, json.Unmarshal([]byte(`{"schema":"1","event_id":"old-event"}`), &oldEvent))
	require.NotContains(t, newUploadEvent(oldEvent).Attributes, "openapiRuntimeVersion")
}
