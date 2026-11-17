package telemetry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func readExecutionDiagnostics(t *testing.T, dir string) []map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "logs", "telemetry.log"))
	require.NoError(t, err)
	var records []map[string]interface{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var record map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(line), &record))
		records = append(records, record)
	}
	return records
}

func TestExecutionDiagnosticsExplainSkippedOrFailedCollection(t *testing.T) {
	t.Setenv(TelemetryDisableEnv, "0")
	t.Cleanup(resetExecutionState)
	for _, scenario := range []struct {
		name, stage, status, reason string
	}{
		{"management", "collection", "skipped", "management_command"},
		{"disabled", "collection", "skipped", "disabled"},
		{"empty_command", "collection", "skipped", "scope_inactive"},
		{"invalid_config_at_begin", "collection", "failed", "invalid_config"},
		{"invalid_config_at_end", "collection", "failed", "invalid_config"},
		{"disabled_at_end", "collection", "skipped", "disabled"},
		{"invalid_id", "collection", "failed", "instance_id_failed"},
		{"cache_blocked", "cache_write", "failed", ""},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			dir := collectionPanicConfigDir(t)
			args := []string{"version"}
			if scenario.name == "management" {
				args = []string{"configure", "telemetry", "show"}
			}
			if scenario.name == "empty_command" {
				args = []string{""}
			}
			if scenario.name == "disabled" {
				require.NoError(t, Save(dir, &Config{Enabled: false}))
			}
			if scenario.name == "invalid_config_at_begin" {
				require.NoError(t, os.WriteFile(GetConfigFilePath(dir), []byte(`{"private-config-secret`), 0600))
			}
			// Saved preferences are only consulted without the disabling override.
			t.Setenv(TelemetryDisableEnv, "")
			Begin(BeginInput{ConfigDir: dir, CommandToken: args[0], CommandArgs: args})
			switch scenario.name {
			case "invalid_config_at_end":
				require.NoError(t, os.WriteFile(GetConfigFilePath(dir), []byte(`{"private-config-secret`), 0600))
			case "disabled_at_end":
				require.NoError(t, Save(dir, &Config{Enabled: false}))
			case "invalid_id":
				require.NoError(t, os.WriteFile(GetInstanceIDFilePath(dir), []byte("private-instance-secret"), 0600))
			case "cache_blocked":
				require.NoError(t, os.WriteFile(filepath.Join(dir, CacheDirName), []byte("private-cache-secret"), 0600))
			}
			End(errors.New("private-business-secret"))
			records := readExecutionDiagnostics(t, dir)
			last := records[len(records)-1]
			require.Equal(t, scenario.stage, last["stage"])
			require.Equal(t, scenario.status, last["status"])
			if scenario.reason != "" {
				require.Equal(t, scenario.reason, last["reason"])
			}
			encoded, err := json.Marshal(records)
			require.NoError(t, err)
			require.NotContains(t, string(encoded), "private-")
			require.NotContains(t, string(encoded), dir)
			require.NoDirExists(t, GetPublicCacheDir(dir))
			EndWithExitCode(1)
			require.Len(t, readExecutionDiagnostics(t, dir), len(records), "duplicate callbacks must not report another event")
		})
	}
}

func TestExecutionDiagnosticsCorrelateEventWithoutBusinessData(t *testing.T) {
	dir := collectionPanicConfigDir(t)
	t.Setenv(TelemetryDisableEnv, "0")
	t.Cleanup(resetExecutionState)
	Begin(BeginInput{
		ConfigDir: dir, CommandToken: "configure",
		CommandArgs: []string{"configure", "set", "--access-key-id", "private-access-key", "--access-key-secret", "private-secret-key"},
	})
	End(errors.New("private-business-error /private/customer/file"))
	entries, err := os.ReadDir(GetPublicCacheDir(dir))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	data, err := os.ReadFile(filepath.Join(GetPublicCacheDir(dir), entries[0].Name()))
	require.NoError(t, err)
	var event Event
	require.NoError(t, json.Unmarshal(data, &event))
	stages := map[string]bool{}
	for _, record := range readExecutionDiagnostics(t, dir) {
		stage := record["stage"].(string)
		stages[stage] = true
		if stage == "event_build" || stage == "cache_write" || stage == "upload_spawn" {
			require.Equal(t, event.EventID, record["eventId"])
			require.Equal(t, PipelinePublic, record["pipeline"])
		}
	}
	for _, stage := range []string{"collection", "event_build", "cache_write", "upload_spawn"} {
		require.True(t, stages[stage], "missing stage %s", stage)
	}
	logData, err := os.ReadFile(filepath.Join(dir, "logs", "telemetry.log"))
	require.NoError(t, err)
	for _, secret := range []string{"private-", "/private/customer", "configure", "access-key", event.InstanceID, dir} {
		require.NotContains(t, string(logData), secret)
	}
	require.NoError(t, Purge(dir))
	require.FileExists(t, filepath.Join(dir, "logs", "telemetry.log"))
}

func TestExecutionDiagnosticsRecordRecoveredPanicWithoutPayload(t *testing.T) {
	dir := collectionPanicConfigDir(t)
	t.Setenv(TelemetryDisableEnv, "0")
	originalTTY := stdinIsTerminal
	t.Cleanup(func() { stdinIsTerminal = originalTTY; resetExecutionState() })
	stdinIsTerminal = func(int) bool { panic("private-panic-secret") }
	Begin(BeginInput{ConfigDir: dir, CommandToken: "version", CommandArgs: []string{"version"}})
	End(nil)
	records := readExecutionDiagnostics(t, dir)
	require.Equal(t, "panic", records[len(records)-1]["reason"])
	require.Equal(t, "failed", records[len(records)-1]["status"])
	data, err := json.Marshal(records)
	require.NoError(t, err)
	require.NotContains(t, string(data), "private-panic-secret")
	require.Nil(t, activeExecution)
	require.NoDirExists(t, GetPublicCacheDir(dir))
}

func TestDiagnosticWriteFailurePreservesEventCollection(t *testing.T) {
	dir := collectionPanicConfigDir(t)
	t.Setenv(TelemetryDisableEnv, "0")
	t.Cleanup(resetExecutionState)
	// Portable on Windows as well: a file cannot be used as a log directory.
	logDir := filepath.Join(dir, "logs")
	require.NoError(t, os.WriteFile(logDir, []byte("keep"), 0600))
	Begin(BeginInput{ConfigDir: dir, CommandToken: "version", CommandArgs: []string{"version"}})
	End(nil)
	entries, err := os.ReadDir(GetPublicCacheDir(dir))
	require.NoError(t, err)
	require.Len(t, entries, 1, "diagnostics failure must not discard the event")
	data, err := os.ReadFile(logDir)
	require.NoError(t, err)
	require.Equal(t, "keep", string(data))
}
