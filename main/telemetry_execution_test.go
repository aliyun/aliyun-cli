//go:build linux || darwin || windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

// The real bl extension re-executes this binary as its standalone executable.
// Handle only the explicit fixture token, never an ordinary CLI invocation.
func runTelemetryExtensionProcess() {
	code, err := strconv.Atoi(os.Getenv("ALIYUN_TEST_EXTENSION_EXIT_CODE"))
	if err != nil {
		os.Exit(98)
	}
	fmt.Fprintln(os.Stdout, "extension stdout")
	fmt.Fprintln(os.Stderr, "extension stderr")
	os.Exit(code)
}

func telemetryExecutionHome(t *testing.T) (home, dir string) {
	t.Helper()
	home = t.TempDir()
	dir = filepath.Join(home, ".aliyun")
	require.NoError(t, os.Mkdir(dir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"current":"default","profiles":[{"name":"default","mode":"Anonymous","region_id":"cn-hangzhou","language":"en"}]}`), 0600))
	require.NoError(t, telemetry.Save(dir, &telemetry.Config{Enabled: true}))
	return home, dir
}

func readTelemetryExecutionEvent(t *testing.T, dir string) telemetry.Event {
	t.Helper()
	cache := telemetry.GetPublicCacheDir(dir)
	entries, err := os.ReadDir(cache)
	require.NoError(t, err)
	require.Len(t, entries, 1, "each invocation must record exactly one completion event")
	data, err := os.ReadFile(filepath.Join(cache, entries[0].Name()))
	require.NoError(t, err)
	var event telemetry.Event
	require.NoError(t, json.Unmarshal(data, &event))
	return event
}

func TestMainExtensionExitPreservesResultAndCollectsTelemetry(t *testing.T) {
	for _, code := range []int{0, 1, 7} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			args := []string{"bl", "__telemetry-extension-test", "--input", "private-input-sentinel"}
			env := []string{"ALIBABA_CLOUD_BL_EXEC_PATH=" + os.Args[0], "ALIYUN_TEST_EXTENSION_EXIT_CODE=" + strconv.Itoa(code)}
			baseline := runTelemetryPermissionCommand(t, home, true, args, env...)
			require.Equal(t, code, baseline.exitCode)
			require.Equal(t, "extension stdout\n", baseline.stdout)
			require.Equal(t, "extension stderr\n", baseline.stderr)
			require.NoDirExists(t, filepath.Join(dir, telemetry.CacheDirName))
			got := runTelemetryPermissionCommand(t, home, false, args, env...)
			require.Equal(t, baseline, got, "telemetry must preserve subprocess output and the actual exit code")
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, telemetry.CommandTypeCLIExt, event.CommandType)
			require.Equal(t, "bl", event.Command)
			require.NotEmpty(t, event.EventID)
			require.NotEmpty(t, event.InstanceID)
			if code == 0 {
				require.Equal(t, "Success", event.Result)
			} else {
				require.Equal(t, "Failure", event.Result)
			}
		})
	}
}

func TestMainTelemetryRecordsEffectiveAIMode(t *testing.T) {
	for _, test := range []struct {
		name         string
		saved        bool
		env          string
		args         []string
		want         bool
		commandError bool
	}{
		{"saved_on", true, "invalid", []string{"version"}, true, false},
		{"saved_off", false, "invalid", []string{"version"}, false, false},
		{"environment_on", false, "true", []string{"version"}, true, false},
		{"environment_off", true, "false", []string{"version"}, false, false},
		{"leading_flag_on", false, "0", []string{"--cli-ai-mode", "version"}, true, false},
		{"leading_flag_off", true, "1", []string{"--no-cli-ai-mode", "version"}, false, false},
		{"error_on", false, "0", []string{"--cli-ai-mode", "version", "--bad-telemetry-test"}, true, true},
		{"error_off", true, "1", []string{"--no-cli-ai-mode", "version", "--bad-telemetry-test"}, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			require.NoError(t, aimode.Save(dir, &aimode.AiConfig{Enabled: test.saved}))
			got := runTelemetryPermissionCommand(t, home, false, test.args, "ALIBABA_CLOUD_CLI_AI_MODE="+test.env)
			if test.commandError {
				require.Greater(t, got.exitCode, 0)
				require.Equal(t, test.want, json.Valid([]byte(got.stderr)), "telemetry AI mode must agree with the actual error rendering mode")
			} else {
				require.Equal(t, 0, got.exitCode)
				require.Empty(t, got.stderr)
			}
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, test.want, event.AIModeEnabled)
		})
	}
}
