package telemetry

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const diagnosticUploadEventID = "9d30cb18-244d-4bda-9f4a-8d6fd2267c83"

func readUploadDiagnostics(t *testing.T, configDir string) ([]diagnosticRecord, string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(configDir, "logs", "telemetry.log"))
	require.NoError(t, err)
	var records []diagnosticRecord
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte{'\n'}) {
		var record diagnosticRecord
		require.NoError(t, json.Unmarshal(line, &record))
		records = append(records, record)
	}
	return records, string(data)
}

func TestUploadDiagnosticsHTTPResultAndPrivacy(t *testing.T) {
	for _, status := range []int{http.StatusAccepted, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
				_, _ = w.Write([]byte("private-response-content"))
			}))
			defer server.Close()
			endpoint := strings.Replace(server.URL, "://", "://private-user:private-password@", 1) + "/private-path?token=private-token"
			t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT", endpoint)
			dir := t.TempDir()
			path, err := WriteEvent(GetPublicCacheDir(dir), Event{
				EventID: diagnosticUploadEventID, Pipeline: PipelinePublic, Result: "Success",
				Command: "private-business-command", Params: []string{"private-argument"}, InstanceID: "private-installation-id",
			})
			require.NoError(t, err)
			require.Zero(t, RunUploadCommand([]string{
				"--config-dir", dir, "--pipeline", PipelinePublic, "--cache-file", path, "--event-id", diagnosticUploadEventID,
			}))
			require.EqualValues(t, 1, calls.Load())
			require.NoFileExists(t, path)
			records, data := readUploadDiagnostics(t, dir)
			require.Len(t, records, 5)
			require.Equal(t, "cache_read", records[0].Stage)
			require.Equal(t, "started", records[0].Status)
			require.Equal(t, "succeeded", records[1].Status)
			require.Equal(t, "cache_remove", records[2].Stage)
			require.Equal(t, "succeeded", records[2].Status)
			require.Equal(t, "upload", records[3].Stage)
			require.Equal(t, "started", records[3].Status)
			require.Equal(t, "upload", records[4].Stage)
			require.Equal(t, status, records[4].HTTPStatus)
			if status == http.StatusAccepted {
				require.Equal(t, "succeeded", records[4].Status)
				require.Empty(t, records[4].Reason)
			} else {
				require.Equal(t, "failed", records[4].Status)
				require.Equal(t, "http_error", records[4].Reason)
			}
			for _, record := range records {
				require.Equal(t, diagnosticUploadEventID, record.EventID)
				require.Equal(t, PipelinePublic, record.Pipeline)
			}
			for _, secret := range []string{"private-", dir, server.URL, path} {
				require.NotContains(t, data, secret)
			}
		})
	}
}

func TestUploadDiagnosticsTransportFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()
	for _, test := range []struct {
		name, endpoint, reason, errorKind string
	}{
		{name: "invalid_request", endpoint: "://private-endpoint?private-token", reason: "request_invalid"},
		{name: "connection_refused", endpoint: server.URL + "/private-path?token=private-token", errorKind: "network_error"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT", test.endpoint)
			dir := t.TempDir()
			path, err := WriteEvent(GetPublicCacheDir(dir), Event{EventID: diagnosticUploadEventID, Pipeline: PipelinePublic})
			require.NoError(t, err)
			require.NoError(t, UploadFile(dir, PipelinePublic, path))
			require.NoFileExists(t, path, "upload failures must not leave an event for retries")
			records, data := readUploadDiagnostics(t, dir)
			last := records[len(records)-1]
			require.Equal(t, "upload", last.Stage)
			require.Equal(t, "failed", last.Status)
			require.Equal(t, test.reason, last.Reason)
			if test.errorKind != "" {
				require.Equal(t, test.errorKind, last.ErrorKind)
			}
			require.Zero(t, last.HTTPStatus)
			require.NotContains(t, data, "private-")
			require.NotContains(t, data, test.endpoint)
			require.NotContains(t, data, path)
		})
	}
}

func TestUploadDiagnosticsRejectedCache(t *testing.T) {
	for _, test := range []struct {
		name, contents, reason string
		outside                bool
	}{
		{name: "empty", reason: "cache_empty"},
		{name: "invalid_json", contents: `{"private-secret":`, reason: "invalid_event"},
		{name: "wrong_pipeline", contents: `{"event_id":"private-secret","pipeline":"inner"}`, reason: "pipeline_mismatch"},
		{name: "outside_cache", contents: "private-secret", reason: "invalid_cache_path", outside: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			cacheDir := GetPublicCacheDir(dir)
			if test.outside {
				cacheDir = t.TempDir()
			}
			require.NoError(t, os.MkdirAll(cacheDir, 0700))
			path := filepath.Join(cacheDir, "private-secret.ndjson")
			require.NoError(t, os.WriteFile(path, []byte(test.contents), 0600))
			require.NoError(t, UploadFile(dir, PipelinePublic, path))
			records, data := readUploadDiagnostics(t, dir)
			last := records[len(records)-1]
			require.Equal(t, "skipped", last.Status)
			require.Equal(t, test.reason, last.Reason)
			if test.name == "invalid_json" {
				require.Equal(t, "invalid_json", last.ErrorKind)
			}
			if test.outside {
				require.FileExists(t, path)
			} else {
				require.NoFileExists(t, path)
			}
			require.NotContains(t, data, "private-secret")
			require.NotContains(t, data, path)
		})
	}
}

func TestUploadDiagnosticsCacheReadAndRemovalFailures(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing_file", true: "nonempty_directory"}[directory], func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(GetPublicCacheDir(dir), "private-secret.ndjson")
			if directory {
				require.NoError(t, os.MkdirAll(path, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(path, "child"), nil, 0600))
			}
			require.NoError(t, UploadFile(dir, PipelinePublic, path))
			records, data := readUploadDiagnostics(t, dir)
			require.Len(t, records, 3)
			require.Equal(t, "cache_read", records[1].Stage)
			require.Equal(t, "failed", records[1].Status)
			require.NotEmpty(t, records[1].ErrorKind)
			require.Equal(t, "cache_remove", records[2].Stage)
			require.Equal(t, "failed", records[2].Status)
			require.NotEmpty(t, records[2].ErrorKind)
			require.NotContains(t, data, "private-secret")
			require.NotContains(t, data, path)
		})
	}
}

func TestUploadDiagnosticsInvalidArguments(t *testing.T) {
	for _, test := range []struct {
		name, reason string
		args         []string
	}{
		{name: "missing_cache_file", reason: "missing_arguments", args: []string{"--pipeline=public", "--event-id=private-secret"}},
		{name: "invalid_pipeline", reason: "invalid_pipeline", args: []string{"--pipeline=private-secret", "--cache-file=private-secret", "--event-id=private-secret"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			require.Zero(t, RunUploadCommand(append([]string{"--config-dir=" + dir}, test.args...)))
			records, data := readUploadDiagnostics(t, dir)
			require.Len(t, records, 1)
			require.Equal(t, "upload", records[0].Stage)
			require.Equal(t, "skipped", records[0].Status)
			require.Equal(t, test.reason, records[0].Reason)
			require.NotContains(t, data, "private-secret")
		})
	}
	t.Run("direct_invalid_pipeline", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, UploadFile(dir, "private-secret", "private-secret"))
		records, data := readUploadDiagnostics(t, dir)
		require.Len(t, records, 1)
		require.Equal(t, "invalid_pipeline", records[0].Reason)
		require.NotContains(t, data, "private-secret")
	})
}

func TestSpawnUploadDiagnosticsStartFailure(t *testing.T) {
	dir := t.TempDir()
	// A NUL cannot be passed to a subprocess on any supported platform.
	path := filepath.Join(GetPublicCacheDir(dir), "private-secret\x00.ndjson")
	spawnUpload(dir, PipelinePublic, path, diagnosticUploadEventID)
	records, data := readUploadDiagnostics(t, dir)
	require.Len(t, records, 3)
	require.Equal(t, "upload_spawn", records[0].Stage)
	require.Equal(t, "started", records[0].Status)
	require.Equal(t, "upload_spawn", records[1].Stage)
	require.Equal(t, "failed", records[1].Status)
	require.Equal(t, "process_start_failed", records[1].Reason)
	require.NotEmpty(t, records[1].ErrorKind)
	require.Equal(t, "cache_remove", records[2].Stage)
	require.Equal(t, "failed", records[2].Status)
	for _, record := range records {
		require.Equal(t, diagnosticUploadEventID, record.EventID)
	}
	require.NotContains(t, data, "private-secret")
	require.NotContains(t, data, dir)
}

func TestTelemetryUploadResultClassifiesTimeoutAndMarshalFailure(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		defer server.Close()
		t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT", server.URL)
		result := postTelemetryBatchResult(nil, &http.Client{Timeout: time.Nanosecond})
		require.False(t, result.success)
		require.Equal(t, "timeout", result.errorKind)
		require.Zero(t, result.httpStatus)
	})
	t.Run("marshal_failure", func(t *testing.T) {
		result := postTelemetryBatchResult([]uploadEvent{{Attributes: map[string]any{"private-secret": make(chan int)}}}, nil)
		require.False(t, result.success)
		require.Equal(t, "marshal_failed", result.reason)
		require.Equal(t, "other", result.errorKind)
	})
}
