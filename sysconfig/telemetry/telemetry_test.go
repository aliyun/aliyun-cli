package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
)

func TestGlobalAndInnerOptOut(t *testing.T) {
	t.Setenv("ALIYUN_CLI_TELEMETRY_OPTOUT", "1")
	if !GlobalOptOut() {
		t.Fatal("expected global opt out")
	}
	t.Setenv("ALIYUN_CLI_TELEMETRY_OPTOUT", "")
	t.Setenv("ALIYUN_CLI_TELEMETRY_INNER_OPTOUT", "yes")
	if !InnerOptOut() {
		t.Fatal("expected inner opt out")
	}
}

func TestL1HardDisabled(t *testing.T) {
	if !L1HardDisabled(ProfileSnapshot{RegionID: "cn-shanghai-finance-1"}) {
		t.Fatal("finance region should disable")
	}
	if !L1HardDisabled(ProfileSnapshot{Endpoint: "https://ecs.apsara.local"}) {
		t.Fatal("apsara endpoint should disable")
	}
	if L1HardDisabled(ProfileSnapshot{RegionID: "cn-hangzhou"}) {
		t.Fatal("public region should not disable")
	}
}

func TestExtractParamNames(t *testing.T) {
	got := ExtractParamNames([]string{"rdc", "list", "--region-id", "x", "--output=json", "-q"})
	want := []string{"region-id", "output", "q"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSanitizeSummary(t *testing.T) {
	in := "failed for /Users/alice/proj secret LTAIabcdefghijklmnop012345"
	out := SanitizeSummary(in)
	if out == in {
		t.Fatal("expected sanitization")
	}
	if contains(out, "alice") || contains(out, "LTAI") {
		t.Fatalf("leaked sensitive data: %q", out)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestShouldCollectInnerBuiltInCommand(t *testing.T) {
	ok, _ := ShouldCollectInner("configure", ProfileSnapshot{}, DefaultConfig())
	if ok {
		t.Fatal("configure should not trigger inner telemetry")
	}
}

func TestDetectInnerScopeUsesDynamicReservedCommands(t *testing.T) {
	const command = "telemetry-dynamic-reserved-test"
	plugin.RegisterReservedTopLevelCommands([]string{command})
	if scope := DetectInnerScope(command); scope.Active {
		t.Fatal("dynamically registered root command should not trigger inner telemetry")
	}
}

func TestInnerAutoOnDefault(t *testing.T) {
	cfg := DefaultConfig()
	if !InnerAutoOnEnabled(cfg) {
		t.Fatal("inner auto on should default true")
	}
}

func TestWriteEvent(t *testing.T) {
	dir := t.TempDir()
	cache := GetInnerCacheDir(dir)
	ev, err := BuildEvent(BuildInput{
		Scope:      InnerScope{Active: true, Trigger: "plugin_manifest", PluginName: "p", PluginVer: "1"},
		RootArgs:   []string{"rdc", "list"},
		Profile:    ProfileSnapshot{RegionID: "cn-hangzhou"},
		InstanceID: "test-id",
	})
	if err != nil {
		t.Fatal(err)
	}
	path, err := WriteEvent(cache, ev)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	st := InnerCacheStats(cache)
	if st.Files != 1 || st.TotalBytes == 0 {
		t.Fatalf("unexpected stats: %+v", st)
	}
	_ = os.RemoveAll(dir)
}

func TestEventCacheFileNameIncludesSortableUTCTimestamp(t *testing.T) {
	createdAt := time.Date(
		2026, time.October, 10, 16, 38, 12, 345000000,
		time.FixedZone("UTC+8", 8*60*60),
	)
	got := eventCacheFileName(createdAt, "event-id")
	want := "event-20261010T083812.345Z-event-id.ndjson"
	if got != want {
		t.Fatalf("eventCacheFileName() = %q, want %q", got, want)
	}
}

func TestWriteEventCleansExpiredOrphanEventFiles(t *testing.T) {
	cacheDir := t.TempDir()
	oldEvent := filepath.Join(cacheDir, "event-old.ndjson")
	recentEvent := filepath.Join(cacheDir, "event-recent.ndjson")
	unrelated := filepath.Join(cacheDir, "other-old.ndjson")
	for _, path := range []string{oldEvent, recentEvent, unrelated} {
		if err := os.WriteFile(path, []byte("{}\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	oldTime := now.Add(-4 * 24 * time.Hour)
	for _, path := range []string{oldEvent, unrelated} {
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(cacheRetentionDaysEnv, "3")

	newFile, err := WriteEvent(cacheDir, Event{EventID: "new"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(oldEvent); !os.IsNotExist(err) {
		t.Fatalf("expired event should be deleted, stat error: %v", err)
	}
	for _, path := range []string{recentEvent, unrelated, newFile} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("file should be retained: %s: %v", path, err)
		}
	}
}

func TestCacheRetentionUsesSafeDefaultForInvalidValues(t *testing.T) {
	for _, value := range []string{"", "0", "-1", "invalid", "3651"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv(cacheRetentionDaysEnv, value)
			want := time.Duration(defaultCacheRetentionDays) * 24 * time.Hour
			if got := cacheRetention(); got != want {
				t.Fatalf("cacheRetention() = %s, want %s", got, want)
			}
		})
	}
}

func TestUploadInnerFileUsesVersionedExtensibleContract(t *testing.T) {
	var batches [][]map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/telemetry/events" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var batch []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		batches = append(batches, batch)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv(
		"ALIYUN_CLI_TELEMETRY_INNER_ENDPOINT",
		server.URL+"/v1/telemetry/events",
	)

	configDir := t.TempDir()
	cacheDir := GetInnerCacheDir(configDir)
	event := Event{
		Schema:       "1",
		EventID:      "44f505ad-0b0d-4ae8-bbd0-8db24b288fa2",
		Timestamp:    "2026-10-09T12:20:30Z",
		Pipeline:     "inner",
		InnerTrigger: "plugin_manifest",
		Command:      "rdc list",
		RawCommand:   "rdc list secret-position-value",
		Params:       []string{"region-id", "output"},
		Result:       "Success",
		ErrorSummary: "must not be uploaded",
		DurationMs:   42,
		Plugin:       "aliyun-cli-rdc@1.0.0",
		CLIVersion:   "3.0.300",
		OS:           "darwin",
		Arch:         "arm64",
		GoVersion:    "go1.25.1",
		RegionID:     "cn-hangzhou",
		Mode:         "default",
		InstanceID:   "d69b3a04-b81c-47e7-a882-504e5865352d",
	}
	cacheFile, err := WriteEvent(cacheDir, event)
	if err != nil {
		t.Fatal(err)
	}

	if err := UploadInnerFile(configDir, cacheFile); err != nil {
		t.Fatal(err)
	}

	if len(batches) != 1 || len(batches[0]) != 1 {
		t.Fatalf("unexpected batches: %v", batches)
	}
	got := batches[0][0]
	if got["schemaVersion"] != "1" ||
		got["eventType"] != "cli.command.completed" ||
		got["clientName"] != "aliyun-cli" ||
		got["instanceId"] != event.InstanceID ||
		got["status"] != "success" {
		t.Fatalf("unexpected upload event: %#v", got)
	}
	attributes, ok := got["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("missing attributes: %#v", got)
	}
	if attributes["command"] != event.Command {
		t.Fatalf("unexpected attributes: %#v", attributes)
	}
	if _, exists := got["rawCommand"]; exists {
		t.Fatal("raw command must not be uploaded")
	}
	if _, exists := attributes["errorSummary"]; exists {
		t.Fatal("error summary must not be uploaded")
	}
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Fatalf("uploaded cache should be removed, stat error: %v", err)
	}
}

func TestUploadInnerFileDropsEventAfterFailedAttempt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	t.Setenv("ALIYUN_CLI_TELEMETRY_INNER_ENDPOINT", server.URL)

	configDir := t.TempDir()
	cacheFile, err := WriteEvent(GetInnerCacheDir(configDir), Event{
		Schema:     "1",
		EventID:    "44f505ad-0b0d-4ae8-bbd0-8db24b288fa2",
		Timestamp:  "2026-10-09T12:20:30Z",
		Pipeline:   "inner",
		Command:    "rdc list",
		Result:     "Success",
		CLIVersion: "3.0.300",
		OS:         "darwin",
		Arch:       "arm64",
		GoVersion:  "go1.25.1",
		Mode:       "default",
		InstanceID: "d69b3a04-b81c-47e7-a882-504e5865352d",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := UploadInnerFile(configDir, cacheFile); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Fatalf("failed upload cache should be dropped, stat error: %v", err)
	}
}

func TestRunUploadCommandAcceptsHiddenArgumentForms(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("ALIYUN_CLI_TELEMETRY_INNER_ENDPOINT", server.URL)

	for _, test := range []struct {
		name string
		args func(configDir, cacheFile string) []string
	}{
		{
			name: "separate",
			args: func(configDir, cacheFile string) []string {
				return []string{
					"--pipeline", "inner",
					"--config-dir", configDir,
					"--cache-file", cacheFile,
				}
			},
		},
		{
			name: "equals",
			args: func(configDir, cacheFile string) []string {
				return []string{
					"--pipeline=inner",
					"--config-dir=" + configDir,
					"--cache-file=" + cacheFile,
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			configDir := t.TempDir()
			cacheFile, err := WriteEvent(GetInnerCacheDir(configDir), Event{
				Schema:     "1",
				EventID:    "44f505ad-0b0d-4ae8-bbd0-8db24b288fa2",
				Timestamp:  "2026-10-09T12:20:30Z",
				Pipeline:   "inner",
				Command:    "cspec",
				Result:     "Success",
				CLIVersion: "3.0.300",
				OS:         "darwin",
				Arch:       "arm64",
				GoVersion:  "go1.25.1",
				Mode:       "default",
				InstanceID: "d69b3a04-b81c-47e7-a882-504e5865352d",
			})
			if err != nil {
				t.Fatal(err)
			}

			if code := RunUploadCommand(test.args(configDir, cacheFile)); code != 0 {
				t.Fatalf("unexpected exit code: %d", code)
			}
			if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
				t.Fatalf("cache should be consumed, stat error: %v", err)
			}
		})
	}

	if requests != 2 {
		t.Fatalf("got %d upload requests, want 2", requests)
	}
}
