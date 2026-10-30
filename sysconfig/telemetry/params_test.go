package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/stretchr/testify/require"
)

func TestExtractParamNamesExcludesValues(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{
			"inline separators",
			[]string{"ecs", "DescribeRegions", "--access-key-id:private-key", "--profile:private-profile", "--body={\"private\":true}", "--path:C:\\private\\file", "-q:private-query"},
			[]string{"--access-key-id", "--profile", "--body", "--path", "-q"},
		},
		{
			"separate values and duplicate flags",
			[]string{"ecs", "DescribeRegions", "--name", "private-name", "--name", "another-private-name", "--Tags.1.Key", "private-tag", "--output=json", "--region-id:cn-private"},
			[]string{"--name", "--Tags.1.Key", "--output", "--region-id"},
		},
		{
			"dash leading values",
			[]string{"ecs", "DescribeRegions", "--count", "-123", "--offset", "-1.25", "--body", "-private-body", "--cert", "-----BEGIN PRIVATE KEY-----", "--region-id=cn-test"},
			[]string{"--count", "--offset", "--body", "--cert", "--region-id"},
		},
		{
			"ambiguous short value is omitted",
			[]string{"ecs", "DescribeRegions", "--body", "-x", "--output=json", "-q"},
			[]string{"--body", "--output", "-q"},
		},
		{
			"empty inline assignment still consumes a value",
			[]string{"ecs", "DescribeRegions", "--body=", "-x", "--name:", "-y", "--output=json"},
			[]string{"--body", "--name", "--output"},
		},
		{
			"invalid flag forms",
			[]string{"plugin-entry", "-private", "-123", "---certificate", "--/private/path", "--private body", "--{private:body}", "--valid"},
			[]string{"--valid"},
		},
		{
			"consecutive long flags are names",
			[]string{"plugin-entry", "resource", "list", "--debug", "--verbose", "--output=json"},
			[]string{"--debug", "--verbose", "--output"},
		},
		{
			"end of options",
			[]string{"plugin-entry", "--valid=ok", "--", "--private-business-content", "-x"},
			[]string{"--valid"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, ExtractParamNames(test.args))
		})
	}
}

func TestCommandPartsDoNotPromotePositionalBusinessValues(t *testing.T) {
	previousExtensions := cliExtensions
	RegisterCLIExtensions(map[string]bool{"test-extension": true, "ossutil": true})
	t.Cleanup(func() { cliExtensions = previousExtensions })
	for _, test := range []struct {
		name  string
		args  []string
		scope Scope
		want  string
	}{
		{"legacy API", []string{"ecs", "DescribeRegions", "/private/file", "private body"}, Scope{CommandToken: "ecs"}, "ecs describeregions"},
		{"runtime API", []string{"ecs", "describe-regions", "customer-id"}, Scope{CommandToken: "ecs"}, "ecs describe-regions"},
		{"unresolved route without metadata", []string{"unknown-token", "list", "customer-id", "private body"}, Scope{}, "unknown-token list"},
		{"path in operation position", []string{"ecs", "/private/file"}, Scope{CommandToken: "ecs"}, "ecs"},
		{"body in operation position", []string{"ecs", "{\"private\":true}"}, Scope{CommandToken: "ecs"}, "ecs"},
		{"invalid entry point", []string{"/private/file", "private body"}, Scope{}, ""},
		{"flag before operation", []string{"ecs", "--region-id:private-region", "DescribeRegions"}, Scope{CommandToken: "ecs"}, "ecs describeregions"},
		{"extension hierarchy", []string{"test-extension", "resource", "run", "--path", "/private/file", "--body:private body"}, Scope{CommandToken: "test-extension"}, "test-extension resource run"},
		{"registered ossutil extension", []string{"ossutil", "ls", "oss://private-bucket/path"}, Scope{CommandToken: "ossutil"}, "ossutil ls"},
		{"registered ossutil copy", []string{"ossutil", "cp", "/private/file", "oss://private-bucket/path"}, Scope{CommandToken: "ossutil"}, "ossutil cp"},
		{"oss copy", []string{"oss", "cp", "/private/file", "oss://private-bucket/path"}, Scope{CommandToken: "oss"}, "oss cp"},
		{"plugin metadata without name", []string{"custom-plugin", "resource", "list", "--body=private-body"}, Scope{CommandToken: "custom-plugin", PluginType: plugin.PluginTypeGo}, "custom-plugin resource list"},
		{"plugin empty inline values", []string{"custom-plugin", "resource", "list", "--path=", "/private/file", "--body:", "private body"}, Scope{CommandToken: "custom-plugin", PluginType: plugin.PluginTypeGo}, "custom-plugin resource list"},
		{"configure profile", []string{"configure", "switch", "--profile", "private-profile"}, Scope{CommandToken: "configure"}, "configure switch"},
		{"configure fields", []string{"configure", "get", "region", "language"}, Scope{CommandToken: "configure"}, "configure get"},
		{"configure nested", []string{"configure", "ai-mode", "set-user-agent", "--user-agent", "private-agent"}, Scope{CommandToken: "configure"}, "configure ai-mode set-user-agent"},
		{"core hierarchy without a whitelist", []string{"mock", "import", "--file", "/private/mock.json"}, Scope{CommandToken: "mock"}, "mock import"},
		{"flag between core levels", []string{"configure", "--profile", "private-profile", "get", "region"}, Scope{CommandToken: "configure"}, "configure get"},
		{"plugin management", []string{"plugin", "show", "--name", "private-plugin-name"}, Scope{CommandToken: "plugin"}, "plugin show"},
		{"plugin search", []string{"plugin", "search", "private-search-term"}, Scope{CommandToken: "plugin"}, "plugin search"},
		{"plugin search with flag", []string{"plugin", "--source-base=https://private.example", "search", "private-search-term"}, Scope{CommandToken: "plugin"}, "plugin search"},
		{"migration path", []string{"go-migrate", "/private/source", "--yes"}, Scope{CommandToken: "go-migrate"}, "go-migrate"},
		{"migration utility path", []string{"utils", "go-migrate", "/private/source", "--yes"}, Scope{CommandToken: "utils"}, "utils go-migrate"},
		{"ROA request path", []string{"cs", "GET", "/clusters/private-id"}, Scope{CommandToken: "cs"}, "cs get"},
		{"ROA path before flag", []string{"cs", "/clusters/private-id", "--roa", "GET"}, Scope{CommandToken: "cs"}, "cs"},
		{"version", []string{"version"}, Scope{CommandToken: "version"}, "version"},
		{"known help path", []string{"help", "configure", "get", "region"}, Scope{CommandToken: "help"}, "help configure get"},
		{"unknown help path", []string{"help", "private-value"}, Scope{CommandToken: "help"}, "help"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, normalizeCommand(commandParts(test.args, test.scope)))
		})
	}
}

func TestBuildEventOmitsKnownPositionalInputs(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"search", []string{"plugin", "search", "private-input-sentinel"}, "plugin search"},
		{"migration", []string{"go-migrate", "/private-input-sentinel/src", "--yes"}, "go-migrate"},
		{"migration utility", []string{"utils", "go-migrate", "/private-input-sentinel/src", "--yes"}, "utils go-migrate"},
		{"ROA path", []string{"cs", "GET", "/clusters/private-input-sentinel"}, "cs get"},
		{"ROA flag", []string{"cs", "/clusters/private-input-sentinel", "--roa", "GET"}, "cs"},
	} {
		t.Run(test.name, func(t *testing.T) {
			event, err := BuildEvent(BuildInput{
				Scope:    Scope{Active: true, Pipeline: PipelinePublic, CommandToken: test.args[0]},
				RootArgs: test.args, InstanceID: "instance-id",
			})
			require.NoError(t, err)
			uploaded := newUploadEvent(event)
			require.Equal(t, test.want, uploaded.Attributes["command"])
			data, err := json.Marshal(uploaded)
			require.NoError(t, err)
			require.NotContains(t, string(data), "private-input-sentinel")
		})
	}
}

func TestBuildEventForInstalledPluginPreservesProtocolWithoutValues(t *testing.T) {
	pluginsDir := t.TempDir()
	t.Setenv(plugin.EnvPluginsDir, pluginsDir)
	t.Setenv(TelemetryDisableEnv, "")
	manifest := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-private-plugin": {
			Name: "aliyun-cli-private-plugin", Command: "private-plugin", Type: plugin.PluginTypeGo,
		},
	}}
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), data, 0600))
	ok, scope := SelectScope("private-plugin", DefaultConfig())
	require.True(t, ok)
	require.Equal(t, "aliyun-cli-private-plugin", scope.PluginName)
	require.Equal(t, plugin.PluginTypeGo, scope.PluginType)
	event, err := BuildEvent(BuildInput{
		Scope: scope, InstanceID: "instance-id",
		RootArgs: []string{"private-plugin", "resource", "policy", "list", "--path", "/private/customer.json", "--body=private business body", "--secret:private-secret", "--debug", "--verbose"},
	})
	require.NoError(t, err)
	uploaded := newUploadEvent(event)
	require.Equal(t, "private-plugin resource policy list", uploaded.Attributes["command"])
	require.Equal(t, []string{"--path", "--body", "--secret", "--debug", "--verbose"}, uploaded.Attributes["parameterNames"])
	data, err = json.Marshal(uploaded)
	require.NoError(t, err)
	for _, sensitive := range []string{"/private/customer.json", "private business body", "private-secret"} {
		require.NotContains(t, string(data), sensitive)
	}
}
