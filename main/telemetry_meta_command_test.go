//go:build linux || darwin || windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/indexed"
	"github.com/stretchr/testify/require"
)

const telemetryMetaCommandProduct = "telemetrymetafixture"

func telemetryMetaCommandFixture(t *testing.T, corruptAPI bool) string {
	t.Helper()
	pluginsDir := t.TempDir()
	const packageName = "aliyun-cli-telemetrymetafixture"
	packageDir := filepath.Join(pluginsDir, packageName)
	require.NoError(t, os.Mkdir(packageDir, 0700))
	manifest := schema.PluginManifest{
		Name: packageName, Version: "1.2.3", Type: schema.DistributionMeta,
		ProductCode: telemetryMetaCommandProduct, Command: telemetryMetaCommandProduct,
		APIVersions: schema.ManifestAPIVersions{Default: "2024-01-01", Supported: []string{"2024-01-01", "2023-01-01"}},
		Metadata: &schema.MetadataDescriptor{
			Format: schema.FormatJSON, Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion,
			Layout: "jsonl", LayoutVersion: schema.LayoutVersion, Index: schema.MetadataIndexFile, Data: schema.MetadataDataFile,
		},
	}
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, "manifest.json"), data, 0600))

	var payload []byte
	var records []indexed.Record
	for i, api := range []struct{ name, command, version string }{
		{"DescribeThing", "describe-thing", "2024-01-01"},
		{"ExportThing", "export-thing", "2023-01-01"},
	} {
		definition := schema.CommandDefinition{
			Name: api.name, CmdName: api.command,
			Operation:  &schema.OperationConfig{Action: api.name, APIVersion: api.version, Method: "POST", APIStyle: "RPC", Protocol: "HTTPS"},
			Parameters: []schema.ArgumentDefinition{{Name: "name", RawName: "Name", Type: "string", Location: "Query", Options: []string{"--name"}}},
		}
		raw, err := json.Marshal(definition)
		require.NoError(t, err)
		if corruptAPI && i == 0 {
			// The routing index and data digest remain valid. Only loading the
			// selected API's definition fails, after command/version lookup.
			raw = []byte(`{"telemetry-private-invalid-payload":`)
		}
		records = append(records, indexed.Record{
			APIVersion: api.version, APIName: api.name, CommandName: api.command,
			Offset: int64(len(payload)), Length: int64(len(raw)),
		})
		payload = append(payload, raw...)
		payload = append(payload, '\n')
	}
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, schema.MetadataDataFile), payload, 0600))
	digest := sha256.Sum256(payload)
	index := indexed.Index{
		Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion, LayoutVersion: schema.LayoutVersion,
		DataFile: schema.MetadataDataFile, DataSize: int64(len(payload)), DataSHA256: "sha256:" + hex.EncodeToString(digest[:]),
		APIs: records,
	}
	data, err = json.Marshal(index)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, schema.MetadataIndexFile), data, 0600))
	installed := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		packageName: {
			Name: packageName, Version: "1.2.3", Command: telemetryMetaCommandProduct,
			ProductCode: telemetryMetaCommandProduct, Type: plugin.PluginTypeMeta, Path: packageDir,
		},
	}}
	data, err = json.Marshal(installed)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), data, 0600))
	return pluginsDir
}

func TestMainTelemetryMetaCommandRequiresSelectedVersionResolution(t *testing.T) {
	const privatePath = "oss://telemetry-private-bucket/private-input.jar"
	const privateScalar = "telemetry-private-positional-input"
	const privateValue = "telemetry-private-parameter-value"
	const privateCommand = "telemetry-private-unknown-command"
	pluginsDir := telemetryMetaCommandFixture(t, false)
	corruptPluginsDir := telemetryMetaCommandFixture(t, true)
	for _, test := range []struct {
		name        string
		args        []string
		wantCommand string
		success     bool
		corruptAPI  bool
	}{
		{"explicit_default_api", []string{"describe-thing", "--api-version", "2024-01-01", "--name", privateValue, "--endpoint", "example.invalid", "--cli-dry-run-json"}, "describe-thing", true, false},
		{"known_api_positional_path", []string{"describe-thing", privatePath, "--endpoint", "example.invalid", "--cli-dry-run-json"}, "describe-thing", false, false},
		{"known_api_scalar_surplus", []string{"describe-thing", "--name", privateValue, privateScalar, "--endpoint", "example.invalid", "--cli-dry-run-json"}, "describe-thing", false, false},
		{"unknown_command", []string{privateCommand, privateScalar, "--cli-dry-run-json"}, "", false, false},
		{"explicit_nondefault_api", []string{"export-thing", "--api-version=2023-01-01", "--endpoint", "example.invalid", "--cli-dry-run-json"}, "export-thing", true, false},
		{"api_only_in_other_version", []string{"export-thing", "--endpoint", "example.invalid", "--cli-dry-run-json"}, "", false, false},
		{"api_rejected_in_explicit_version", []string{"describe-thing", "--api-version", "2023-01-01", "--cli-dry-run-json"}, "", false, false},
		{"api_help", []string{"describe-thing", "--help"}, "describe-thing", true, false},
		{"nondefault_api_help", []string{"export-thing", "--api-version", "2023-01-01", "--help"}, "export-thing", true, false},
		{"help_rejected_in_default_version", []string{"export-thing", "--help"}, "", false, false},
		{"manifest_version_ignores_surplus", []string{"version", privateScalar}, "version", true, false},
		{"list_api_versions", []string{"list-api-versions"}, "list-api-versions", true, false},
		{"list_api_versions_rejects_path", []string{"list-api-versions", privatePath}, "list-api-versions", false, false},
		{"api_definition_load_failure", []string{"describe-thing", "--cli-dry-run-json"}, "", false, true},
		{"help_definition_load_failure", []string{"describe-thing", "--help"}, "", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			selectedPluginsDir := pluginsDir
			if test.corruptAPI {
				selectedPluginsDir = corruptPluginsDir
			}
			env := []string{plugin.EnvPluginsDir + "=" + selectedPluginsDir}
			args := append([]string{telemetryMetaCommandProduct}, test.args...)
			baseline := runTelemetryPermissionCommand(t, home, true, args, env...)
			if test.success {
				require.Zero(t, baseline.exitCode, baseline.stderr)
				require.NotEmpty(t, baseline.stdout)
			} else {
				require.Greater(t, baseline.exitCode, 0)
			}
			require.NoFileExists(t, telemetry.GetInstanceIDFilePath(dir))
			require.NoDirExists(t, filepath.Join(dir, telemetry.CacheDirName))
			got := runTelemetryPermissionCommand(t, home, false, args, env...)
			require.Equal(t, baseline, got, "telemetry must preserve metadata command output and exit status")
			event := readTelemetryExecutionEvent(t, dir)
			wantCommand := telemetryMetaCommandProduct
			if test.wantCommand != "" {
				wantCommand += " " + test.wantCommand
			}
			require.Equal(t, wantCommand, event.Command)
			require.Equal(t, telemetry.CommandTypePluginMeta, event.CommandType)
			require.Equal(t, "aliyun-cli-telemetrymetafixture@1.2.3", event.Plugin)
			if test.success {
				require.Equal(t, "Success", event.Result)
			} else {
				require.NotEqual(t, "Success", event.Result)
			}
			data, err := json.Marshal(event)
			require.NoError(t, err)
			for _, private := range []string{privatePath, privateScalar, privateValue, privateCommand, "telemetry-private-invalid-payload", selectedPluginsDir, home} {
				require.NotContains(t, string(data), private)
			}
		})
	}
}
