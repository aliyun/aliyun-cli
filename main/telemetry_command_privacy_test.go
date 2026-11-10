//go:build linux || darwin || windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

func TestMainTelemetryCoreCommandPathsExcludeBusinessInputs(t *testing.T) {
	const secret = "telemetry-private-secret-value"
	const surplus = "telemetry-private-surplus-value"
	const name = "telemetry-private-plugin-name"
	const userAgent = "telemetry-private-user-agent"
	for _, test := range []struct {
		name, command string
		args          []string
		success       bool
	}{
		{"configure_set_secret_surplus", "configure set", []string{"configure", "set", "--access-key-secret", secret, surplus}, true},
		{"configure_invalid_flag_before_subcommand", "configure", []string{"configure", "--access-key-secret", secret, "set", surplus}, false},
		{"configure_profile_between_commands", "configure set", []string{"configure", "--profile", "default", "set", "--access-key-secret", secret, surplus}, true},
		{"configure_invalid_mode", "configure set", []string{"configure", "set", "--mode", "telemetry-private-invalid-mode", surplus}, false},
		{"configure_unknown_subcommand", "configure", []string{"configure", surplus}, false},
		{"nested_command_interspersed_flag", "configure ai-mode set-user-agent", []string{"configure", "ai-mode", "--user-agent", userAgent, "set-user-agent"}, true},
		{"nested_command_invalid_tail", "configure ai-mode set-user-agent", []string{"configure", "ai-mode", "--user-agent", userAgent, "set-user-agent", surplus}, false},
		{"plugin_list_surplus", "plugin list", []string{"plugin", "list", surplus}, true},
		{"plugin_uninstall_unknown_name", "plugin uninstall", []string{"plugin", "uninstall", "--name", name, surplus}, false},
		{"plugin_install_invalid_combination", "plugin install", []string{"plugin", "install", "--name", name, "--names", "telemetry-private-second-name", surplus}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			// These are fixture credentials in an isolated profile. Core commands
			// below never issue a business API request or install remote plugins.
			profile := `{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"fixture-id","access_key_secret":"fixture-secret","region_id":"cn-hangzhou","language":"en"}]}`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(profile), 0600))
			baseline := runTelemetryPermissionCommand(t, home, true, test.args)
			if test.success {
				require.Zero(t, baseline.exitCode, baseline.stderr)
			} else {
				require.Greater(t, baseline.exitCode, 0)
			}
			require.NoDirExists(t, telemetry.GetPublicCacheDir(dir))
			got := runTelemetryPermissionCommand(t, home, false, test.args)
			require.Equal(t, baseline, got, "telemetry must not change core command output or exit status")
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, test.command, event.Command)
			require.Equal(t, telemetry.CommandTypeCore, event.CommandType)
			if test.success {
				require.Equal(t, "Success", event.Result)
			} else {
				require.NotEqual(t, "Success", event.Result)
			}
			data, err := json.Marshal(event)
			require.NoError(t, err)
			for _, private := range []string{secret, surplus, name, userAgent, "telemetry-private-invalid-mode", "telemetry-private-second-name"} {
				require.NotContains(t, string(data), private)
			}
		})
	}
}

func TestMainTelemetryGoCommandPathsExcludeUnconfirmedTailOnFailure(t *testing.T) {
	const private = "telemetry-private-go-input"
	for _, test := range []struct {
		name, command string
		paths         []string
	}{
		{"declared_path", "privacyfixture function list", []string{"function list"}},
		{"missing_command_list", "privacyfixture", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			pluginsDir := filepath.Join(dir, "plugins")
			require.NoError(t, os.Mkdir(pluginsDir, 0700))
			profileRequired := false
			manifest := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
				"aliyun-cli-privacyfixture": {
					Name: "aliyun-cli-privacyfixture", Version: "1.2.3", Command: "privacyfixture",
					Type: plugin.PluginTypeGo, CmdNames: test.paths, ProfileRequired: &profileRequired,
					// Selecting the plugin is sufficient to exercise failure-side
					// telemetry; no executable or business action is provided.
					Path: filepath.Join(pluginsDir, "missing-executable"),
				},
			}}
			data, err := json.Marshal(manifest)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), data, 0600))
			args := []string{"privacyfixture", "function", "list", private}
			env := []string{plugin.EnvPluginsDir + "=" + pluginsDir}
			baseline := runTelemetryPermissionCommand(t, home, true, args, env...)
			require.Greater(t, baseline.exitCode, 0)
			require.Contains(t, baseline.stderr, "plugin binary not found")
			got := runTelemetryPermissionCommand(t, home, false, args, env...)
			require.Equal(t, baseline, got)
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, telemetry.CommandTypePluginGo, event.CommandType)
			require.Equal(t, "Failure", event.Result)
			require.Equal(t, test.command, event.Command)
			data, err = json.Marshal(event)
			require.NoError(t, err)
			require.NotContains(t, string(data), private)
		})
	}
}
