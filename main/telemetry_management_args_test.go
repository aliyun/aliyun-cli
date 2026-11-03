//go:build linux || darwin || windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

const telemetryManagementInstanceID = "d69b3a04-b81c-47e7-a882-504e5865352d\n"
const telemetryManagementCachedEvent = "{\"fixture\":true}\n"

func seedTelemetryManagementData(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(telemetry.GetInstanceIDFilePath(dir), []byte(telemetryManagementInstanceID), 0600))
	for _, cache := range []string{telemetry.GetPublicCacheDir(dir), telemetry.GetInnerCacheDir(dir)} {
		require.NoError(t, os.MkdirAll(cache, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(cache, "existing.json"), []byte(telemetryManagementCachedEvent), 0600))
	}
}

func assertTelemetryManagementDataPreserved(t *testing.T, dir string) {
	t.Helper()
	assertTelemetryPermissionFile(t, telemetry.GetInstanceIDFilePath(dir), telemetryManagementInstanceID)
	entries, err := os.ReadDir(filepath.Join(dir, telemetry.CacheDirName))
	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, cache := range []string{telemetry.GetPublicCacheDir(dir), telemetry.GetInnerCacheDir(dir)} {
		entries, err := os.ReadDir(cache)
		require.NoError(t, err)
		require.Len(t, entries, 1, "management must not add a telemetry event")
		require.Equal(t, "existing.json", entries[0].Name())
		assertTelemetryPermissionFile(t, filepath.Join(cache, "existing.json"), telemetryManagementCachedEvent)
	}
}

func assertTelemetryManagementDataAbsent(t *testing.T, dir string) {
	t.Helper()
	require.NoFileExists(t, telemetry.GetInstanceIDFilePath(dir))
	require.NoDirExists(t, filepath.Join(dir, telemetry.CacheDirName))
}

func TestMainTelemetryPurgeWithLeadingGlobalFlagsDoesNotRecreateData(t *testing.T) {
	for _, args := range [][]string{
		{"--cli-dry-run", "configure", "telemetry", "purge"},
		{"--cli-dry-run-json", "configure", "telemetry", "purge"},
		{"--estimate-cost", "configure", "telemetry", "purge"},
		{"--profile", "default", "--cli-dry-run", "configure", "telemetry", "purge"},
		{"--cli-dry-run-json", "--profile=default", "configure", "telemetry", "purge"},
		{"--estimate-cost", "configure", "--profile", "default", "telemetry", "purge"},
	} {
		t.Run(args[0]+"_"+args[1], func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			seedTelemetryManagementData(t, dir)
			baseline := runTelemetryPermissionCommand(t, home, true, args)
			require.Zero(t, baseline.exitCode, baseline.stderr)
			assertTelemetryManagementDataAbsent(t, dir)

			seedTelemetryManagementData(t, dir)
			got := runTelemetryPermissionCommand(t, home, false, args)
			require.Equal(t, baseline, got, "telemetry must preserve purge output and exit status")
			assertTelemetryManagementDataAbsent(t, dir)
			cfg, err := telemetry.Load(dir)
			require.NoError(t, err)
			require.True(t, cfg.Enabled, "purge must preserve the enabled preference")
		})
	}
}

func TestMainTelemetryShowWithLeadingGlobalFlagsPreservesExistingData(t *testing.T) {
	for _, args := range [][]string{
		{"--cli-dry-run", "configure", "telemetry", "show"},
		{"--cli-dry-run-json", "--profile", "default", "configure", "telemetry", "show"},
		{"--estimate-cost", "configure", "--profile=default", "telemetry"},
	} {
		t.Run(args[0], func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			seedTelemetryManagementData(t, dir)
			baseline := runTelemetryPermissionCommand(t, home, true, args)
			require.Zero(t, baseline.exitCode, baseline.stderr)
			assertTelemetryManagementDataPreserved(t, dir)
			got := runTelemetryPermissionCommand(t, home, false, args)
			require.Equal(t, baseline.exitCode, got.exitCode)
			require.Equal(t, baseline.stderr, got.stderr)
			assertTelemetryManagementDataPreserved(t, dir)

			// show intentionally reports the environment override. Its other
			// output fields must remain identical when collection is permitted.
			var disabled, enabled map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(baseline.stdout), &disabled))
			require.NoError(t, json.Unmarshal([]byte(got.stdout), &enabled))
			require.Equal(t, true, disabled["environment_disabled"])
			require.Equal(t, false, enabled["environment_disabled"])
			disabled["environment_disabled"] = false
			require.Equal(t, disabled, enabled)
		})
	}
}

func TestMainTelemetryPreferenceCommandsWithLeadingGlobalFlagsDoNotCreateData(t *testing.T) {
	for _, command := range []string{"enable", "disable"} {
		for _, flag := range []string{"--cli-dry-run", "--cli-dry-run-json", "--estimate-cost"} {
			t.Run(command+"/"+flag, func(t *testing.T) {
				home, dir := telemetryExecutionHome(t)
				args := []string{flag, "--profile", "default", "configure", "telemetry", command}
				baseline := runTelemetryPermissionCommand(t, home, true, args)
				require.Zero(t, baseline.exitCode, baseline.stderr)
				assertTelemetryManagementDataAbsent(t, dir)
				// The disabled baseline can change this preference. Restore it
				// before the enabled invocation so collection could otherwise run.
				require.NoError(t, telemetry.Save(dir, &telemetry.Config{Enabled: true}))
				got := runTelemetryPermissionCommand(t, home, false, args)
				require.Equal(t, baseline, got)
				assertTelemetryManagementDataAbsent(t, dir)
				cfg, err := telemetry.Load(dir)
				require.NoError(t, err)
				require.Equal(t, command == "enable", cfg.Enabled)
			})
		}
	}
}

func TestMainTelemetryManagementHelpWithLeadingGlobalFlagsDoesNotCollect(t *testing.T) {
	for _, args := range [][]string{
		{"--cli-dry-run", "help", "configure", "telemetry", "purge"},
		{"--cli-dry-run-json", "configure", "telemetry", "purge", "--help"},
		{"--estimate-cost", "--profile", "default", "configure", "help", "telemetry"},
	} {
		t.Run(args[0], func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			seedTelemetryManagementData(t, dir)
			baseline := runTelemetryPermissionCommand(t, home, true, args)
			require.Zero(t, baseline.exitCode, baseline.stderr)
			require.NotEmpty(t, baseline.stdout)
			assertTelemetryManagementDataPreserved(t, dir)
			got := runTelemetryPermissionCommand(t, home, false, args)
			require.Equal(t, baseline, got)
			assertTelemetryManagementDataPreserved(t, dir)
		})
	}
}

func TestMainTelemetryProfileNamedTelemetryStillCollectsCoreCommand(t *testing.T) {
	for _, args := range [][]string{
		{"--cli-dry-run", "--profile", "telemetry", "configure", "set"},
		{"--cli-dry-run-json", "configure", "set", "--profile", "telemetry"},
		{"--estimate-cost", "configure", "--profile=telemetry", "set"},
	} {
		t.Run(args[0], func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			profile := `{"current":"default","profiles":[{"name":"default","mode":"Anonymous","region_id":"cn-hangzhou","language":"en"},{"name":"telemetry","mode":"Anonymous","region_id":"cn-hangzhou","language":"en"}]}`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(profile), 0600))
			baseline := runTelemetryPermissionCommand(t, home, true, args)
			require.Zero(t, baseline.exitCode, baseline.stderr)
			assertTelemetryManagementDataAbsent(t, dir)
			got := runTelemetryPermissionCommand(t, home, false, args)
			require.Equal(t, baseline, got)
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, "configure set", event.Command)
			require.Equal(t, telemetry.CommandTypeCore, event.CommandType)
			require.Equal(t, "Success", event.Result)
		})
	}
}
