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
	openapiruntime "github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/stretchr/testify/require"
)

func TestMainTelemetryRuntimeVersionFollowsExecutionRoute(t *testing.T) {
	metadataDir, err := filepath.Abs("../aliyun-openapi-meta")
	require.NoError(t, err)
	for _, test := range []struct {
		name string
		args []string
		want bool
	}{
		{"runtime_help", []string{"ecs", "describe-regions", "--help"}, true},
		{"legacy_help", []string{"ecs", "DescribeRegions", "--help"}, false},
		{"core_version", []string{"version"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			got := runTelemetryPermissionCommand(t, home, false, test.args, "ALIYUN_CLI_META_DIR="+metadataDir)
			require.Equal(t, 0, got.exitCode, got.stderr)
			require.NotEmpty(t, got.stdout)
			event := readTelemetryExecutionEvent(t, dir)
			if test.want {
				require.Equal(t, openapiruntime.Version, event.OpenAPIRuntimeVersion)
			} else {
				require.Empty(t, event.OpenAPIRuntimeVersion)
			}
			cache := telemetry.GetPublicCacheDir(dir)
			entries, err := os.ReadDir(cache)
			require.NoError(t, err)
			require.Len(t, entries, 1)
			data, err := os.ReadFile(filepath.Join(cache, entries[0].Name()))
			require.NoError(t, err)
			var raw map[string]any
			require.NoError(t, json.Unmarshal(data, &raw))
			if test.want {
				require.Equal(t, openapiruntime.Version, raw["openapi_runtime_version"])
			} else {
				require.NotContains(t, raw, "openapi_runtime_version")
			}
		})
	}
}

func TestMainTelemetryRuntimeVersionForInstalledMetaPlugin(t *testing.T) {
	pluginsDir := t.TempDir()
	packageDir := filepath.Join(pluginsDir, "aliyun-cli-telemetryfixture")
	require.NoError(t, os.Mkdir(packageDir, 0700))
	const manifest = `{"name":"aliyun-cli-telemetryfixture","version":"1.2.3","type":"meta","productCode":"telemetryfixture","command":"telemetryfixture","apiVersions":{"default":"2024-01-01","supported":["2024-01-01"]},"metadata":{"format":"json","schema":"aliyun-openapi-meta","schemaVersion":1,"layout":"jsonl","layoutVersion":1,"index":"metadata.index.json","data":"metadata.jsonl"}}`
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, "manifest.json"), []byte(manifest), 0600))
	definition := schema.CommandDefinition{
		Name: "DescribeThing", CmdName: "describe-thing",
		Operation: &schema.OperationConfig{Action: "DescribeThing", APIVersion: "2024-01-01", Method: "POST", APIStyle: "RPC", Protocol: "HTTPS"},
	}
	raw, err := json.Marshal(definition)
	require.NoError(t, err)
	data := append(append([]byte(nil), raw...), '\n')
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, schema.MetadataDataFile), data, 0600))
	digest := sha256.Sum256(data)
	index, err := json.Marshal(map[string]any{
		"schema": schema.SchemaName, "schemaVersion": schema.SchemaVersion,
		"layoutVersion": schema.LayoutVersion, "dataFile": schema.MetadataDataFile,
		"dataSize": len(data), "dataSha256": "sha256:" + hex.EncodeToString(digest[:]),
		"apis": []map[string]any{{"apiVersion": "2024-01-01", "apiName": definition.Name, "commandName": definition.CmdName, "offset": 0, "length": len(raw)}},
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, schema.MetadataIndexFile), index, 0600))
	installed := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-telemetryfixture": {Name: "aliyun-cli-telemetryfixture", Version: "1.2.3", Command: "telemetryfixture", ProductCode: "telemetryfixture", Type: plugin.PluginTypeMeta, Path: packageDir},
	}}
	installedJSON, err := json.Marshal(installed)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), installedJSON, 0600))
	for _, test := range []struct {
		name string
		args []string
		want bool
	}{
		{"help", []string{"telemetryfixture", "describe-thing", "--help"}, true},
		{"dry_run", []string{"telemetryfixture", "describe-thing", "--endpoint", "example.invalid", "--cli-dry-run-json"}, true},
		{"manifest_version", []string{"telemetryfixture", "version"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			got := runTelemetryPermissionCommand(t, home, false, test.args, plugin.EnvPluginsDir+"="+pluginsDir)
			require.Equal(t, 0, got.exitCode, got.stderr)
			require.NotEmpty(t, got.stdout)
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, telemetry.CommandTypePluginMeta, event.CommandType)
			require.Equal(t, "aliyun-cli-telemetryfixture@1.2.3", event.Plugin)
			if test.want {
				require.Equal(t, openapiruntime.Version, event.OpenAPIRuntimeVersion)
			} else {
				require.Empty(t, event.OpenAPIRuntimeVersion, "printing a manifest version must not claim engine execution")
			}
		})
	}
}
