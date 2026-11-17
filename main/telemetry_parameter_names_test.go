//go:build linux || darwin || windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/stretchr/testify/require"
)

func TestMainTelemetryUsesParsedParameterNames(t *testing.T) {
	const privateToken = "--telemetry-private-customer-token"
	const privateKey = "--telemetry-private-customer-key"
	pluginsDir := telemetryMetaCommandFixture(t, false,
		schema.ArgumentDefinition{Name: "name", RawName: "Name", Type: "string", Location: "Query", Options: []string{"--name"}},
		schema.ArgumentDefinition{Name: "display_name", RawName: "DisplayName", Type: "string", Location: "Query", Options: []string{"--DisplayName"}},
		schema.ArgumentDefinition{Name: "names", RawName: "Names", Type: "array", Location: "Query", Options: []string{"--names"}, Element: &schema.TypeShape{Type: "string"}},
		schema.ArgumentDefinition{Name: "labels", RawName: "Labels", Type: "map", Location: "Query", Options: []string{"--labels"}, Value: &schema.TypeShape{Type: "string"}},
		schema.ArgumentDefinition{Name: "config", RawName: "Config", Type: "object", Location: "Query", Options: []string{"--config"}, Fields: []schema.ArgumentDefinition{{Name: "value", RawName: "Value", Type: "string"}}},
		schema.ArgumentDefinition{Name: "count", RawName: "Count", Type: "integer", Location: "Query", Options: []string{"--count"}},
	)
	dryRun := []string{"--endpoint", "example.invalid", "--cli-dry-run-json"}
	for _, test := range []struct {
		name    string
		leading []string
		args    []string
		want    []string
		success bool
		dryRun  bool
	}{
		{"scalar_dash_value", nil, []string{"--name", privateToken}, []string{"--name", "--endpoint", "--cli-dry-run-json"}, true, true},
		{"array_dash_values", nil, []string{"--names", "first", privateToken, privateKey}, []string{"--names", "--endpoint", "--cli-dry-run-json"}, true, true},
		{"map_dash_key", nil, []string{"--labels", privateKey + "=private-value"}, []string{"--labels", "--endpoint", "--cli-dry-run-json"}, true, true},
		{"object_dash_value", nil, []string{"--config", "Value=" + privateToken}, []string{"--config", "--endpoint", "--cli-dry-run-json"}, true, true},
		{"inline_scalar_value", nil, []string{"--name=" + privateToken}, []string{"--name", "--endpoint", "--cli-dry-run-json"}, true, true},
		{"case_order_and_repetition", nil, []string{"--DisplayName", privateToken, "--names", privateKey, "--name", "second", "--names", "third"}, []string{"--DisplayName", "--names", "--name", "--names", "--endpoint", "--cli-dry-run-json"}, true, true},
		{"leading_host_flag", []string{"--profile", "default"}, []string{"--name", privateToken}, []string{"--profile", "--name", "--endpoint", "--cli-dry-run-json"}, true, true},
		{"type_failure", nil, []string{"--name", privateToken, "--count", privateKey}, []string{"--name", "--count"}, false, false},
		{"missing_value", nil, []string{"--name", privateToken, "--count"}, []string{"--name", "--count"}, false, false},
		{"unknown_after_scalar", nil, []string{"--name", privateToken, privateKey}, []string{"--name"}, false, false},
		{"unknown_first", nil, []string{privateToken}, nil, false, false},
		{"object_failure", nil, []string{"--config", privateToken}, []string{"--config"}, false, false},
		{"array_inline_failure", nil, []string{"--names=" + privateToken}, []string{"--names"}, false, false},
		{"api_help", nil, []string{"--names", privateToken, "--names", privateKey, "--help"}, []string{"--names", "--names", "--help"}, true, false},
		{"parameter_help", nil, []string{"--names", privateToken, "--DisplayName", "--help"}, []string{"--names", "--DisplayName", "--help"}, true, false},
		{"parameter_help_parse_failure", nil, []string{"--count", privateToken, "--DisplayName", "--help"}, []string{"--count", "--DisplayName"}, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			env := []string{plugin.EnvPluginsDir + "=" + pluginsDir}
			args := append(append([]string(nil), test.leading...), telemetryMetaCommandProduct, "describe-thing")
			args = append(args, test.args...)
			if test.dryRun {
				args = append(args, dryRun...)
			}
			baseline := runTelemetryPermissionCommand(t, home, true, args, env...)
			if test.success {
				require.Zero(t, baseline.exitCode, baseline.stderr)
				require.NotEmpty(t, baseline.stdout)
			} else {
				require.Greater(t, baseline.exitCode, 0)
			}
			require.NoDirExists(t, filepath.Join(dir, telemetry.CacheDirName))
			got := runTelemetryPermissionCommand(t, home, false, args, env...)
			require.Equal(t, baseline, got, "collecting confirmed parameter names must preserve output and exit status")
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, test.want, event.Params)
			data, err := json.Marshal(event)
			require.NoError(t, err)
			for _, private := range []string{privateToken, privateKey, "private-value", home, pluginsDir} {
				require.NotContains(t, string(data), private)
			}
		})
	}
}

func TestMainTelemetryInfersOpaqueToolParameterNames(t *testing.T) {
	const privateValue = "telemetry-private-opaque-value"
	for _, test := range []struct {
		name      string
		leading   []string
		args      []string
		extension bool
		want      []string
	}{
		{"go_multiple_values", nil, []string{"--Tag", privateValue, "another-private-opaque-value", "--verbose"}, false, []string{"--Tag", "--verbose"}},
		{"go_case_repetitions_and_inline", nil, []string{"--Tag=" + privateValue, "--tag:" + privateValue, "--Tag=" + privateValue, "-t=" + privateValue, "-t:" + privateValue}, false, []string{"--Tag", "--tag", "--Tag", "-t", "-t"}},
		{"go_host_flags_not_duplicated", []string{"--profile", "default"}, []string{"--region", "cn-hangzhou", "--tag", privateValue}, false, []string{"--profile", "--region", "--tag"}},
		// Without downstream definitions, these two tokens cannot be reliably
		// distinguished from a flag followed by a dash-leading business value.
		{"go_ambiguous_dash_token", nil, []string{"--name", "--ambiguous-value"}, false, []string{"--name", "--ambiguous-value"}},
		{"extension_multiple_values", nil, []string{"--Tag", privateValue, "another-private-opaque-value", "--verbose"}, true, []string{"--Tag", "--verbose"}},
		{"extension_case_repetitions_and_inline", nil, []string{"--Tag=" + privateValue, "--tag:" + privateValue, "--Tag=" + privateValue, "-t=" + privateValue, "-t:" + privateValue}, true, []string{"--Tag", "--tag", "--Tag", "-t", "-t"}},
		{"extension_host_flag", []string{"--profile", "default"}, []string{"--tag", privateValue}, true, []string{"--profile", "--tag"}},
		{"extension_end_of_options", nil, []string{"--tag", privateValue, "--", "--uncollected-token", "another-private-opaque-value"}, true, []string{"--tag"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			var env, commandArgs []string
			wantCommand, wantType := "privacyfixture function list", telemetry.CommandTypePluginGo
			if test.extension {
				commandArgs = []string{"bl", "__telemetry-extension-test"}
				env = []string{"ALIBABA_CLOUD_BL_EXEC_PATH=" + os.Args[0], "ALIYUN_TEST_EXTENSION_EXIT_CODE=0"}
				wantCommand, wantType = "bl", telemetry.CommandTypeCLIExt
			} else {
				pluginsDir := filepath.Join(dir, "plugins")
				require.NoError(t, os.Mkdir(pluginsDir, 0700))
				profileRequired := false
				manifest := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
					"aliyun-cli-privacyfixture": {
						Name: "aliyun-cli-privacyfixture", Version: "1.2.3", Command: "privacyfixture",
						Type: plugin.PluginTypeGo, CmdNames: []string{"function list"}, ProfileRequired: &profileRequired,
						Path: filepath.Join(pluginsDir, "missing-executable"),
					},
				}}
				data, err := json.Marshal(manifest)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), data, 0600))
				env = []string{plugin.EnvPluginsDir + "=" + pluginsDir}
				commandArgs = []string{"privacyfixture", "function", "list"}
			}
			commandArgs = append(commandArgs, test.args...)
			args := append(append([]string(nil), test.leading...), commandArgs...)
			baseline := runTelemetryPermissionCommand(t, home, true, args, env...)
			if test.extension {
				require.Zero(t, baseline.exitCode, baseline.stderr)
			} else {
				require.Greater(t, baseline.exitCode, 0)
				require.Contains(t, baseline.stderr, "plugin binary not found")
			}
			got := runTelemetryPermissionCommand(t, home, false, args, env...)
			require.Equal(t, baseline, got, "inferring names must preserve opaque tool output and exit status")
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, wantCommand, event.Command)
			require.Equal(t, wantType, event.CommandType)
			require.Equal(t, test.want, event.Params)
			data, err := json.Marshal(event)
			require.NoError(t, err)
			for _, value := range []string{privateValue, "another-private-opaque-value", "--uncollected-token"} {
				require.NotContains(t, string(data), value)
			}
		})
	}
}

func TestMainTelemetryDoesNotInferGoCandidateBeforeRouteSelection(t *testing.T) {
	home, dir := telemetryExecutionHome(t)
	pluginsDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.Mkdir(pluginsDir, 0700))
	profileRequired := false
	manifest := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-privacyfixture": {
			Name: "aliyun-cli-privacyfixture", Version: "1.2.3", Command: "privacyfixture",
			Type: plugin.PluginTypeGo, CmdNames: []string{"function list"}, ProfileRequired: &profileRequired,
			Path: filepath.Join(pluginsDir, "missing-executable"),
		},
	}}
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), data, 0600))
	args := []string{
		"privacyfixture", "ListThings", "--body=telemetry-private-candidate-body",
		"--ambiguous-value", "ordinary-value", "--profile", "default", "--profile", "default",
	}
	env := []string{plugin.EnvPluginsDir + "=" + pluginsDir}
	baseline := runTelemetryPermissionCommand(t, home, true, args, env...)
	require.Greater(t, baseline.exitCode, 0)
	require.Contains(t, baseline.stderr, "--profile")
	require.NotContains(t, baseline.stderr, "plugin binary not found", "generic parsing must fail before plugin routing")
	got := runTelemetryPermissionCommand(t, home, false, args, env...)
	require.Equal(t, baseline, got)
	event := readTelemetryExecutionEvent(t, dir)
	require.Equal(t, []string{"--body", "--profile", "--profile"}, event.Params)
	data, err = json.Marshal(event)
	require.NoError(t, err)
	for _, value := range []string{"--ambiguous-value", "ordinary-value", "telemetry-private-candidate-body"} {
		require.NotContains(t, string(data), value)
	}
}
