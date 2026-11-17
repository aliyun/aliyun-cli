package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiagnosticLogDisableEnvironment(t *testing.T) {
	for _, value := range []string{"1", "true", "yes", " TRUE ", "Yes", "", "0", "false", "no"} {
		t.Run("value="+value, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv(TelemetryLogDisableEnv, value)
			writeDiagnostic(dir, diagnosticRecord{Stage: "collection", Status: "started"})
			switch value {
			case "1", "true", "yes", " TRUE ", "Yes":
				entries, err := os.ReadDir(dir)
				require.NoError(t, err)
				require.Empty(t, entries, "disabling diagnostics must not create logs or locks")
			default:
				require.FileExists(t, filepath.Join(dir, "logs", diagnosticLogName))
			}
		})
	}
}

func TestDisabledDiagnosticsLeaveExistingFilesUntouched(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(TelemetryLogDisableEnv, "0")
	path := filepath.Join(dir, "logs", diagnosticLogName)
	record := diagnosticRecord{Stage: "collection", Status: "started"}
	writeDiagnostic(dir, record)
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path+".1", []byte("existing-backup\n"), 0600))
	t.Setenv(TelemetryLogDisableEnv, "1")
	// The small limit would force rotation if the disable guard were skipped.
	writeDiagnosticWithLimits(dir, record, int64(len(before)), diagnosticBackups)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after)
	backup, err := os.ReadFile(path + ".1")
	require.NoError(t, err)
	require.Equal(t, "existing-backup\n", string(backup))
	require.NoFileExists(t, path+".2")

	require.NoError(t, os.Unsetenv(TelemetryLogDisableEnv))
	writeDiagnostic(dir, record)
	require.Len(t, readDiagnosticEntries(t, path), 2, "removing the override must resume local logging")
}

func TestDisablingDiagnosticsPreservesCollectionAndUpload(t *testing.T) {
	dir := collectionPanicConfigDir(t)
	t.Setenv(TelemetryLogDisableEnv, "1")
	t.Setenv(TelemetryDisableEnv, "0")
	t.Cleanup(resetExecutionState)
	Begin(BeginInput{ConfigDir: dir, CommandToken: "version", CommandArgs: []string{"version"}})
	End(nil)
	entries, err := os.ReadDir(GetPublicCacheDir(dir))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	cacheFile := filepath.Join(GetPublicCacheDir(dir), entries[0].Name())
	data, err := os.ReadFile(cacheFile)
	require.NoError(t, err)
	var event Event
	require.NoError(t, json.Unmarshal(data, &event))
	require.Equal(t, "version", event.Command)
	require.Equal(t, "Success", event.Result)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT", server.URL)
	require.Zero(t, RunUploadCommand([]string{
		"--config-dir", dir, "--pipeline", PipelinePublic, "--cache-file", cacheFile, "--event-id", event.EventID,
	}))
	require.EqualValues(t, 1, requests.Load(), "the log switch must not disable telemetry upload")
	require.NoFileExists(t, cacheFile)
	require.NoDirExists(t, filepath.Join(dir, "logs"))
}
