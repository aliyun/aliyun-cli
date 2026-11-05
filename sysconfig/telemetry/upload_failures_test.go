package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUploadRejectsInvalidCacheWithoutSending(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1) }))
	defer server.Close()
	t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT", server.URL)
	for _, test := range []struct {
		name, contents                    string
		outside, wrongExtension, retained bool
	}{
		{name: "empty"},
		{name: "whitespace", contents: " \n\t\n"},
		{name: "malformed", contents: "{\n"},
		{name: "wrong_pipeline", contents: `{"pipeline":"inner"}`},
		{name: "oversized_line", contents: strings.Repeat("a", 128*1024)},
		{name: "outside_cache", contents: `{"pipeline":"public"}`, outside: true, retained: true},
		{name: "wrong_extension", contents: `{"pipeline":"public"}`, wrongExtension: true, retained: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			cache := GetPublicCacheDir(dir)
			if test.outside {
				cache = t.TempDir()
			}
			require.NoError(t, os.MkdirAll(cache, 0700))
			name := "event.ndjson"
			if test.wrongExtension {
				name = "keep.json"
			}
			path := filepath.Join(cache, name)
			require.NoError(t, os.WriteFile(path, []byte(test.contents), 0600))
			require.NoError(t, UploadFile(dir, PipelinePublic, path))
			if test.retained {
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, test.contents, string(data))
			} else {
				require.NoFileExists(t, path, "invalid events must not accumulate or be retried")
			}
		})
	}
	dir := t.TempDir()
	require.NoError(t, UploadFile(dir, PipelinePublic, filepath.Join(GetPublicCacheDir(dir), "missing.ndjson")))
	require.Zero(t, calls.Load())
}

func TestUploadCommandReadsFlagsAndConsumesOnlyFirstEvent(t *testing.T) {
	for _, inline := range []bool{false, true} {
		t.Run(map[bool]string{false: "separate", true: "inline"}[inline], func(t *testing.T) {
			var received uploadBatch
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("unexpected upload request: %s %v", r.Method, r.Header)
				}
				if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
					t.Error(err)
				}
				w.WriteHeader(http.StatusAccepted)
			}))
			defer server.Close()
			t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT", server.URL)
			dir := t.TempDir()
			cache := GetPublicCacheDir(dir)
			require.NoError(t, os.MkdirAll(cache, 0700))
			path := filepath.Join(cache, "event.ndjson")
			require.NoError(t, os.WriteFile(path, []byte("\n{\"event_id\":\"first\",\"pipeline\":\"public\",\"result\":\"UserFault\"}\n{\"event_id\":\"second\"}\n"), 0600))
			args := []string{"--config-dir", dir, "--pipeline", PipelinePublic, "--cache-file", path}
			if inline {
				args = []string{"--config-dir=" + dir, "--pipeline=public", "--cache-file=" + path}
			}
			require.Zero(t, RunUploadCommand(args))
			require.Len(t, received.Events, 1)
			require.Equal(t, "first", received.Events[0].EventID)
			require.Equal(t, "user_fault", received.Events[0].Status)
			require.NoFileExists(t, path)
		})
	}
}

func TestPostTelemetryBatchFailures(t *testing.T) {
	t.Run("unserializable_attribute", func(t *testing.T) {
		require.False(t, postTelemetryBatch([]uploadEvent{{Attributes: map[string]any{"invalid": make(chan int)}}}))
	})
	for _, endpoint := range []string{"://invalid", "unsupported://telemetry"} {
		t.Run(endpoint, func(t *testing.T) {
			t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT", endpoint)
			require.False(t, postTelemetryBatch(nil))
		})
	}
	t.Run("server_rejects_event", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
		defer server.Close()
		t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT", server.URL)
		dir := t.TempDir()
		path, err := WriteEvent(GetPublicCacheDir(dir), Event{EventID: "failed-upload", Pipeline: PipelinePublic})
		require.NoError(t, err)
		require.NoError(t, UploadFile(dir, PipelinePublic, path))
		require.NoFileExists(t, path, "upload failures must not cause retries on the next invocation")
	})
}

func TestUploadEventIncludesExplicitFalseInner(t *testing.T) {
	event := newUploadEvent(Event{Result: "Failure", PluginAction: &PluginManagementEvent{Inner: boolPointer(false)}})
	require.Equal(t, false, event.Attributes["inner"])
	require.Equal(t, "failure", event.Status)
	for _, flag := range []string{"--pipeline", "--cache-file", "--config-dir"} {
		require.Zero(t, RunUploadCommand([]string{flag}), "missing flag values must be ignored safely")
	}
}
