package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/stretchr/testify/require"
)

func TestTelemetryRecordsHostResolvedAIMode(t *testing.T) {
	for _, test := range []struct {
		name      string
		saved     bool
		env       string
		effective bool
	}{
		{"explicit_on_overrides_disabled_config_and_environment", false, "0", true},
		{"explicit_off_overrides_enabled_config_and_environment", true, "1", false},
		{"environment_on_overrides_disabled_config", false, "1", true},
		{"environment_off_overrides_enabled_config", true, "0", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(TelemetryDisableEnv, "0")
			t.Setenv(aimode.EnvAIMode, test.env)
			t.Cleanup(resetExecutionState)
			dir := collectionPanicConfigDir(t)
			require.NoError(t, aimode.Save(dir, &aimode.AiConfig{Enabled: test.saved}))
			Begin(BeginInput{
				ConfigDir: dir, CommandArgs: []string{"version"}, CommandToken: "version",
				AIMode: test.effective,
			})
			require.NotNil(t, activeExecution)
			require.Equal(t, test.effective, activeExecution.aiMode)
			// Configuration commands may change the preference during execution.
			// The completed event describes the mode used for this invocation.
			require.NoError(t, aimode.Save(dir, &aimode.AiConfig{Enabled: !test.effective}))
			End(nil)
			entries, err := os.ReadDir(GetPublicCacheDir(dir))
			require.NoError(t, err)
			require.Len(t, entries, 1)
			data, err := os.ReadFile(filepath.Join(GetPublicCacheDir(dir), entries[0].Name()))
			require.NoError(t, err)
			var event Event
			require.NoError(t, json.Unmarshal(data, &event))
			require.Equal(t, test.effective, event.AIModeEnabled)
			require.Equal(t, test.effective, newUploadEvent(event).Attributes["aiModeEnabled"])
		})
	}
}
