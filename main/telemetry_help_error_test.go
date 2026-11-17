//go:build linux || darwin || windows

package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

func TestMainTelemetryHelpPreservesOriginalErrorClassification(t *testing.T) {
	metadataDir, err := filepath.Abs("../aliyun-openapi-meta")
	require.NoError(t, err)
	for _, test := range []struct {
		name      string
		args      []string
		aiMode    bool
		jsonError bool
		command   string
	}{
		{"help_flag", []string{"ecs", "InvalidReviewAPI", "--help"}, false, false, "ecs InvalidReviewAPI"},
		{"help_prefix", []string{"help", "ecs", "InvalidReviewAPI"}, false, false, "help"},
		{"help_flag_ai", []string{"ecs", "InvalidReviewAPI", "--help"}, true, true, "ecs InvalidReviewAPI"},
		{"help_prefix_ai", []string{"help", "ecs", "InvalidReviewAPI"}, true, true, "help"},
		{"explicit_json", []string{"ecs", "InvalidReviewAPI", "--help", "--cli-output", "json"}, false, true, "ecs InvalidReviewAPI"},
		{"help_prefix_json", []string{"help", "ecs", "InvalidReviewAPI", "--cli-output", "json"}, false, true, "help"},
		{"help_with_private_value", []string{"ecs", "InvalidReviewAPI", "--help", "--RegionId", "private-help-value-sentinel"}, false, false, "ecs InvalidReviewAPI"},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			mode := "0"
			if test.aiMode {
				mode = "1"
			}
			env := []string{"ALIYUN_CLI_META_DIR=" + metadataDir, "ALIBABA_CLOUD_CLI_AI_MODE=" + mode}
			baseline := runTelemetryPermissionCommand(t, home, true, test.args, env...)
			require.Greater(t, baseline.exitCode, 0)
			require.NotEmpty(t, baseline.stderr)
			require.Equal(t, test.jsonError, json.Valid([]byte(baseline.stderr)))
			require.NoDirExists(t, telemetry.GetPublicCacheDir(dir))

			got := runTelemetryPermissionCommand(t, home, false, test.args, env...)
			require.Equal(t, baseline, got, "recording the Help error must preserve its output and exit code")
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, "UserFault", event.Result)
			require.Equal(t, telemetry.LocalErrorInvalidCommand, event.LocalErrorCode)
			require.Equal(t, telemetry.CommandTypeOpenAPILegacy, event.CommandType)
			require.Equal(t, test.command, event.Command)
			require.Equal(t, test.aiMode, event.AIModeEnabled)
			data, err := json.Marshal(event)
			require.NoError(t, err)
			require.NotContains(t, string(data), "private-help-value-sentinel")
			require.NotContains(t, string(data), "is not a valid")
			require.NotContains(t, string(data), "did_you_mean")
		})
	}
}

func TestMainTelemetrySuccessfulHelpRemainsSuccessful(t *testing.T) {
	metadataDir, err := filepath.Abs("../aliyun-openapi-meta")
	require.NoError(t, err)
	home, dir := telemetryExecutionHome(t)
	args := []string{"ecs", "DescribeRegions", "--help"}
	env := []string{"ALIYUN_CLI_META_DIR=" + metadataDir}
	baseline := runTelemetryPermissionCommand(t, home, true, args, env...)
	require.Zero(t, baseline.exitCode, baseline.stderr)
	require.NotEmpty(t, baseline.stdout)
	require.Empty(t, baseline.stderr)
	got := runTelemetryPermissionCommand(t, home, false, args, env...)
	require.Equal(t, baseline, got)
	event := readTelemetryExecutionEvent(t, dir)
	require.Equal(t, "Success", event.Result)
	require.Empty(t, event.LocalErrorCode)
	require.Equal(t, telemetry.CommandTypeOpenAPILegacy, event.CommandType)
	require.Equal(t, "ecs DescribeRegions", event.Command)
}
