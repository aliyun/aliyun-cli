//go:build linux || darwin || windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/internal/updatecheck"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/updatenotifier"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestUpgradeCheckOutputFormatsPreserveFailureExitCode(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		json bool
	}{
		{name: "text", args: []string{"upgrade", "--check"}},
		{name: "json", args: []string{"upgrade", "--check", "--cli-output", "json"}, json: true},
		{name: "json_before_command", args: []string{"--cli-output", "json", "upgrade", "--check"}, json: true},
		{name: "json_equals", args: []string{"upgrade", "--check", "--cli-output=json"}, json: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			dir := filepath.Join(home, ".aliyun")
			require.NoError(t, os.MkdirAll(dir, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"current":"default","profiles":[{"name":"default","mode":"Anonymous","language":"en","region_id":"cn-hangzhou"}]}`), 0600))
			got := runTelemetryPermissionCommand(t, home, true, tc.args,
				updatenotifier.DisableEnv+"=1",
				"HTTPS_PROXY=http://127.0.0.1:1", "HTTP_PROXY=http://127.0.0.1:1", "NO_PROXY=",
			)
			require.Equal(t, 1, got.exitCode)
			require.Contains(t, got.stderr, "CLI version check failed")
			require.NotContains(t, got.stdout, "CLI version check failed")
			if tc.json {
				var report updatecheck.Report
				require.NoError(t, json.Unmarshal([]byte(got.stdout), &report))
				require.Equal(t, "partial", report.Status)
				require.Equal(t, "failed", report.CLI.Status)
				require.Equal(t, "complete", report.Plugins.Status)
			} else {
				require.Contains(t, got.stdout, "Update check: Incomplete; some checks failed")
				require.Contains(t, got.stdout, "Status: Check failed")
				require.Contains(t, got.stdout, "No plugins installed.")
			}
		})
	}
}

func TestUpgradeCheckRejectsInvalidOutputOptions(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "invalid_format", args: []string{"upgrade", "--check", "--cli-output", "yaml"}, want: "only supports json"},
		{name: "removed_json_flag", args: []string{"upgrade", "--check", "--json"}, want: "--json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := runTelemetryPermissionCommand(t, t.TempDir(), true, tc.args, updatenotifier.DisableEnv+"=1")
			require.Equal(t, 2, got.exitCode)
			require.Empty(t, got.stdout)
			require.Contains(t, got.stderr, tc.want)
		})
	}
}

func TestNotifierPreservesSuccessAndFailureProcessResults(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"version", "--update-test-unknown"}, {"--cli-ai-mode", "version", "--update-test-unknown"}} {
		t.Run(args[len(args)-1]+args[0], func(t *testing.T) {
			home := t.TempDir()
			dir := filepath.Join(home, ".aliyun")
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "update-notifier-cache"), 0700))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"current":"default","profiles":[{"name":"default","mode":"Anonymous","language":"en","region_id":"cn-hangzhou"}]}`), 0600))
			scope, _ := json.Marshal([]string{filepath.Join(dir, "plugins"), "https://aliyuncli.alicdn.com/plugins-v2", "cn", ""})
			sum := sha256.Sum256(scope)
			state, _ := json.Marshal(map[string]any{"attempt_at": time.Now(), "cli_valid_until": time.Now().Add(24 * time.Hour), "cli": map[string]any{"current": cli.Version, "latest": "99.0.0", "target": "99.0.0", "status": "update_available", "update_available": true}})
			require.NoError(t, os.WriteFile(filepath.Join(dir, "update-notifier-cache", hex.EncodeToString(sum[:16])+".json"), state, 0600))
			baseline := runTelemetryPermissionCommand(t, home, true, args, updatenotifier.DisableEnv+"=1")
			got := runTelemetryPermissionCommand(t, home, true, args, updatenotifier.DisableEnv+"=0")
			require.Equal(t, baseline.stdout, got.stdout)
			require.Equal(t, baseline.exitCode, got.exitCode)
			if len(args) == 1 {
				require.Equal(t, baseline, got)
				require.Empty(t, got.stderr)
			} else if args[0] == "--cli-ai-mode" {
				var before, after map[string]any
				require.NoError(t, json.Unmarshal([]byte(baseline.stderr), &before))
				require.NoError(t, json.Unmarshal([]byte(got.stderr), &after))
				require.NotEmpty(t, after["update_notices"])
				delete(after, "update_notices")
				require.Equal(t, before, after)
			} else {
				require.Contains(t, got.stderr, baseline.stderr)
				require.Contains(t, got.stderr, "aliyun upgrade")
			}
		})
	}
}

func TestNotifierPreservesLegacySDKResponses(t *testing.T) {
	metadataDir, err := filepath.Abs("../aliyun-openapi-meta")
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		status int
		aiMode bool
	}{
		{"success", http.StatusOK, false},
		{"server_error", http.StatusBadRequest, false},
		{"ai_success", http.StatusOK, true},
		{"ai_server_error", http.StatusBadRequest, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("Action") != "DescribeRegions" || r.URL.Query().Get("SignatureMethod") != "HMAC-SHA1" {
					t.Error("request did not use the PascalCase SDK compatibility path")
				}
				// Keep diagnostics identical across the two CLI processes.
				w.Header().Set("Date", "Mon, 02 Jan 2006 15:04:05 GMT")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				body := `{"RequestId":"update-sdk-test","Regions":{"Region":[]}}`
				if tc.status != http.StatusOK {
					body = `{"Code":"InvalidParameter.UpdateTest","Message":"fixture failure","RequestId":"update-sdk-test"}`
				}
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()

			home := t.TempDir()
			dir := filepath.Join(home, ".aliyun")
			cacheDir := filepath.Join(dir, "update-notifier-cache")
			require.NoError(t, os.MkdirAll(cacheDir, 0700))
			// These fixture credentials are sent only to the local test server.
			require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"fixture-id","access_key_secret":"fixture-secret","language":"en","region_id":"cn-hangzhou"}]}`), 0600))
			scope, err := json.Marshal([]string{filepath.Join(dir, "plugins"), "https://aliyuncli.alicdn.com/plugins-v2", "cn", ""})
			require.NoError(t, err)
			sum := sha256.Sum256(scope)
			state, err := json.Marshal(map[string]any{"attempt_at": time.Now(), "cli_valid_until": time.Now().Add(24 * time.Hour), "cli": map[string]any{"current": cli.Version, "latest": "99.0.0", "target": "99.0.0", "status": "update_available", "update_available": true}})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(cacheDir, hex.EncodeToString(sum[:16])+".json"), state, 0600))

			args := []string{"ecs", "DescribeRegions", "--endpoint", strings.TrimPrefix(server.URL, "http://"), "--insecure"}
			if tc.aiMode {
				args = append([]string{"--cli-ai-mode"}, args...)
			}
			env := "ALIYUN_CLI_META_DIR=" + metadataDir
			baseline := runTelemetryPermissionCommand(t, home, true, args, env, updatenotifier.DisableEnv+"=1")
			got := runTelemetryPermissionCommand(t, home, true, args, env, updatenotifier.DisableEnv+"=0")
			require.Equal(t, int32(2), calls.Load(), "both commands must reach the SDK backend")
			require.Equal(t, baseline.stdout, got.stdout)
			require.Equal(t, baseline.exitCode, got.exitCode)
			if tc.status == http.StatusOK {
				require.Zero(t, got.exitCode, got.stderr)
				require.Empty(t, got.stderr)
				require.Contains(t, got.stdout, "update-sdk-test")
			} else if tc.aiMode {
				require.NotZero(t, got.exitCode)
				var before, after map[string]any
				require.NoError(t, json.Unmarshal([]byte(baseline.stderr), &before))
				require.NoError(t, json.Unmarshal([]byte(got.stderr), &after))
				require.Equal(t, "InvalidParameter.UpdateTest", after["error_code"])
				require.Equal(t, float64(http.StatusBadRequest), after["status_code"])
				require.Equal(t, "update-sdk-test", after["request_id"])
				require.NotEmpty(t, after["update_notices"])
				delete(after, "update_notices")
				require.Equal(t, before, after)
			} else {
				require.NotZero(t, got.exitCode)
				require.Contains(t, baseline.stderr, "InvalidParameter.UpdateTest")
				require.Contains(t, baseline.stderr, "update-sdk-test")
				require.True(t, strings.HasPrefix(got.stderr, baseline.stderr), got.stderr)
				require.Contains(t, got.stderr, "aliyun upgrade")
				require.Contains(t, got.stderr, "99.0.0")
			}
		})
	}
}
