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

func TestMainTelemetryLegacyRouteIgnoresInstalledGoPlugin(t *testing.T) {
	metadataDir, err := filepath.Abs("../aliyun-openapi-meta")
	require.NoError(t, err)
	pluginsDir := t.TempDir()
	installed := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-sls": {
			Name: "aliyun-cli-sls", Version: "9.8.7", Type: plugin.PluginTypeGo,
			Command: "sls", ProductCode: "sls", Inner: true,
			// No executable is installed: these invocations must stay in the
			// host rather than resolve or execute the unrelated Go plugin.
			Path: filepath.Join(pluginsDir, "aliyun-cli-sls"),
		},
	}}
	data, err := json.Marshal(installed)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), data, 0600))
	for _, test := range []struct {
		name string
		args []string
	}{
		{"dry_run", []string{"sls", "ListProject", "--size", "1", "--endpoint", "example.invalid", "--cli-dry-run-json"}},
		{"help", []string{"sls", "ListProject", "--help"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			got := runTelemetryPermissionCommand(t, home, false, test.args,
				plugin.EnvPluginsDir+"="+pluginsDir,
				"ALIYUN_CLI_META_DIR="+metadataDir,
			)
			require.Equal(t, 0, got.exitCode, got.stderr)
			require.NotEmpty(t, got.stdout)
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, "sls ListProject", event.Command)
			require.Equal(t, telemetry.CommandTypeOpenAPILegacy, event.CommandType)
			require.Equal(t, telemetry.PipelinePublic, event.Pipeline)
			require.Empty(t, event.Plugin)
			require.Empty(t, event.OpenAPIRuntimeVersion)
			require.NoDirExists(t, telemetry.GetInnerCacheDir(dir))
		})
	}
}
