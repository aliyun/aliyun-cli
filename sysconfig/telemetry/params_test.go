package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
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
			[]string{"--name", "--name", "--Tags.1.Key", "--output", "--region-id"},
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
		path  []string
	}{
		{"legacy API", []string{"ecs", "DescribeRegions", "/private/file", "private body"}, Scope{CommandToken: "ecs"}, "ecs DescribeRegions", nil},
		{"runtime API", []string{"ecs", "describe-regions", "customer-id"}, Scope{CommandToken: "ecs"}, "ecs describe-regions", nil},
		{"unresolved route without metadata", []string{"unknown-token", "list", "customer-id", "private body"}, Scope{}, "unknown-token list", nil},
		{"path in operation position", []string{"ecs", "/private/file"}, Scope{CommandToken: "ecs"}, "ecs", nil},
		{"body in operation position", []string{"ecs", "{\"private\":true}"}, Scope{CommandToken: "ecs"}, "ecs", nil},
		{"invalid entry point", []string{"/private/file", "private body"}, Scope{}, "", nil},
		{"flag before operation", []string{"ecs", "--region-id:private-region", "DescribeRegions"}, Scope{CommandToken: "ecs"}, "ecs DescribeRegions", nil},
		{"extension hierarchy is unverified", []string{"test-extension", "resource", "run", "--path", "/private/file", "--body:private body"}, Scope{CommandToken: "test-extension"}, "test-extension", nil},
		{"registered ossutil extension", []string{"ossutil", "ls", "oss://private-bucket/path"}, Scope{CommandToken: "ossutil"}, "ossutil ls", nil},
		{"registered ossutil copy", []string{"ossutil", "cp", "/private/file", "oss://private-bucket/path"}, Scope{CommandToken: "ossutil"}, "ossutil cp", nil},
		{"oss copy", []string{"oss", "cp", "/private/file", "oss://private-bucket/path"}, Scope{CommandToken: "oss"}, "oss cp", nil},
		{"oss unknown operation", []string{"oss", "private-business-input"}, Scope{CommandToken: "oss"}, "oss", nil},
		{"ossutil path without operation", []string{"ossutil", "oss://private-bucket/path"}, Scope{CommandToken: "ossutil"}, "ossutil", nil},
		{"ossutil operation case", []string{"ossutil", "LS", "oss://private-bucket/path"}, Scope{CommandToken: "ossutil"}, "ossutil LS", nil},
		{"plugin metadata without name", []string{"custom-plugin", "resource", "list", "--body=private-body"}, Scope{CommandToken: "custom-plugin", PluginType: plugin.PluginTypeGo, PluginCommandPaths: []string{"resource list"}}, "custom-plugin resource list", nil},
		{"plugin empty inline values", []string{"custom-plugin", "resource", "list", "--path=", "/private/file", "--body:", "private body"}, Scope{CommandToken: "custom-plugin", PluginType: plugin.PluginTypeGo, PluginCommandPaths: []string{"resource list"}}, "custom-plugin resource list", nil},
		{"configure profile", []string{"configure", "switch", "--profile", "private-profile"}, Scope{CommandToken: "configure"}, "configure switch", []string{"configure", "switch"}},
		{"configure fields", []string{"configure", "get", "region", "language"}, Scope{CommandToken: "configure"}, "configure get", []string{"configure", "get"}},
		{"configure nested", []string{"configure", "ai-mode", "set-user-agent", "--user-agent", "private-agent"}, Scope{CommandToken: "configure"}, "configure ai-mode set-user-agent", []string{"configure", "ai-mode", "set-user-agent"}},
		{"core hierarchy without a whitelist", []string{"mock", "import", "--file", "/private/mock.json"}, Scope{CommandToken: "mock"}, "mock import", []string{"mock", "import"}},
		{"flag between core levels", []string{"configure", "--profile", "private-profile", "get", "region"}, Scope{CommandToken: "configure"}, "configure get", []string{"configure", "get"}},
		{"plugin management", []string{"plugin", "show", "--name", "private-plugin-name"}, Scope{CommandToken: "plugin"}, "plugin show", []string{"plugin", "show"}},
		{"plugin search", []string{"plugin", "search", "private-search-term"}, Scope{CommandToken: "plugin"}, "plugin search", []string{"plugin", "search"}},
		{"plugin search with flag", []string{"plugin", "--source-base=https://private.example", "search", "private-search-term"}, Scope{CommandToken: "plugin"}, "plugin search", []string{"plugin", "search"}},
		{"migration path", []string{"go-migrate", "/private/source", "--yes"}, Scope{CommandToken: "go-migrate"}, "go-migrate", nil},
		{"migration utility path", []string{"utils", "go-migrate", "/private/source", "--yes"}, Scope{CommandToken: "utils"}, "utils go-migrate", []string{"utils", "go-migrate"}},
		{"ROA request path", []string{"cs", "GET", "/clusters/private-id"}, Scope{CommandToken: "cs"}, "cs GET", nil},
		{"ROA path before flag", []string{"cs", "/clusters/private-id", "--roa", "GET"}, Scope{CommandToken: "cs"}, "cs", nil},
		{"version", []string{"version"}, Scope{CommandToken: "version"}, "version", nil},
		{"known help path", []string{"help", "configure", "get", "region"}, Scope{CommandToken: "help"}, "help configure get", []string{"configure", "get"}},
		{"unknown help path", []string{"help", "private-value"}, Scope{CommandToken: "help"}, "help", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, normalizeCommand(commandParts(test.args, test.scope, test.path)))
		})
	}
}

func TestBuildEventOmitsKnownPositionalInputs(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
		path []string
	}{
		{"search", []string{"plugin", "search", "private-input-sentinel"}, "plugin search", []string{"plugin", "search"}},
		{"migration", []string{"go-migrate", "/private-input-sentinel/src", "--yes"}, "go-migrate", nil},
		{"migration utility", []string{"utils", "go-migrate", "/private-input-sentinel/src", "--yes"}, "utils go-migrate", []string{"utils", "go-migrate"}},
		{"ROA path", []string{"cs", "GET", "/clusters/private-input-sentinel"}, "cs GET", nil},
		{"ROA flag", []string{"cs", "/clusters/private-input-sentinel", "--roa", "GET"}, "cs", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			event, err := BuildEvent(BuildInput{
				Scope:    Scope{Active: true, Pipeline: PipelinePublic, CommandToken: test.args[0]},
				RootArgs: test.args, InstanceID: "instance-id", CommandPath: test.path,
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

func TestExtensionEventsDoNotCollectPositionalBusinessInputs(t *testing.T) {
	previousExtensions := cliExtensions
	RegisterCLIExtensions(map[string]bool{
		"spark-submit": true, "kmscli": true, "bl": true, "dgate": true,
		"ossutil": true, "test-extension": true,
	})
	t.Cleanup(func() { cliExtensions = previousExtensions })
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"spark application and arguments", []string{"spark-submit", "--class", "private-sentinel.Main", "oss://private-sentinel/main.jar", "private-sentinel-business-input"}, "spark-submit"},
		{"spark local application", []string{"spark-submit", "/private-sentinel/main.py", "private-sentinel-business-input"}, "spark-submit"},
		{"spark only scalar input", []string{"spark-submit", "private-sentinel"}, "spark-submit"},
		{"secret name", []string{"kmscli", "secret", "getsecret", "private-sentinel"}, "kmscli secret"},
		{"tool positional input", []string{"bl", "chat", "private-sentinel"}, "bl"},
		{"tool query", []string{"dgate", "query", "SELECT private-sentinel"}, "dgate"},
		{"unverified extension hierarchy", []string{"test-extension", "resource", "private-sentinel"}, "test-extension"},
		{"ossutil unknown operation", []string{"ossutil", "private-sentinel", "oss://private-sentinel/path"}, "ossutil"},
		{"ossutil verified operation", []string{"ossutil", "cp", "/private-sentinel/file", "oss://private-sentinel/path"}, "ossutil cp"},
		{"ossutil flags before operation", []string{"ossutil", "--region", "private-sentinel-region", "ls", "oss://private-sentinel/path"}, "ossutil"},
		{"end of options", []string{"spark-submit", "--", "--private-sentinel", "private-sentinel"}, "spark-submit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			event, err := BuildEvent(BuildInput{
				Scope:    Scope{Active: true, Pipeline: PipelinePublic, CommandToken: test.args[0]},
				RootArgs: test.args, InstanceID: "instance-id",
			})
			require.NoError(t, err)
			require.Equal(t, test.want, event.Command)
			require.Equal(t, CommandTypeCLIExt, event.CommandType)
			// Check both the persisted event and the transformed upload contract.
			for _, payload := range []any{event, newUploadEvent(event)} {
				data, err := json.Marshal(payload)
				require.NoError(t, err)
				require.NotContains(t, string(data), "private-sentinel")
			}
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
			CmdNames: []string{"resource policy list"},
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
		RootArgs: []string{"private-plugin", "Resource", "Policy", "List", "--path", "/private/customer.json", "--body=private business body", "--secret:private-secret", "--debug", "--verbose"},
	})
	require.NoError(t, err)
	uploaded := newUploadEvent(event)
	require.Equal(t, "private-plugin Resource Policy List", uploaded.Attributes["command"])
	require.Equal(t, CommandTypePluginGo, uploaded.Attributes["commandType"])
	require.Equal(t, []string{"--path", "--body", "--secret", "--debug", "--verbose"}, uploaded.Attributes["parameterNames"])
	data, err = json.Marshal(uploaded)
	require.NoError(t, err)
	for _, sensitive := range []string{"/private/customer.json", "private business body", "private-secret"} {
		require.NotContains(t, string(data), sensitive)
	}
}

func TestEventCommandPreservesCaseAcrossRoutes(t *testing.T) {
	previousExtensions := cliExtensions
	RegisterCLIExtensions(map[string]bool{"ossutil": true})
	t.Cleanup(func() { cliExtensions = previousExtensions })
	for _, test := range []struct {
		name                 string
		args                 []string
		command, commandType string
	}{
		{"legacy", []string{"ecs", "DescribeRegions", "--RegionId", "private-region"}, "ecs DescribeRegions", CommandTypeOpenAPILegacy},
		{"runtime", []string{"ecs", "describe-regions", "--region-id=private-region"}, "ecs describe-regions", CommandTypeOpenAPIRuntime},
		{"product case", []string{"Ecs", "DescribeRegions"}, "Ecs DescribeRegions", CommandTypeOpenAPILegacy},
		{"ROA", []string{"cs", "GET", "/private/path"}, "cs GET", CommandTypeOpenAPILegacy},
		{"extension", []string{"ossutil", "ls", "oss://private-bucket/path"}, "ossutil ls", CommandTypeCLIExt},
	} {
		t.Run(test.name, func(t *testing.T) {
			event, err := BuildEvent(BuildInput{
				Scope:    Scope{Active: true, Pipeline: PipelinePublic, CommandToken: test.args[0]},
				RootArgs: test.args, InstanceID: "instance-id",
			})
			require.NoError(t, err)
			require.Equal(t, test.command, event.Command)
			uploaded := newUploadEvent(event)
			require.Equal(t, test.command, uploaded.Attributes["command"])
			require.Equal(t, test.commandType, uploaded.Attributes["commandType"])
			data, err := json.Marshal(uploaded)
			require.NoError(t, err)
			require.NotContains(t, string(data), "private-region")
			require.NotContains(t, string(data), "/private/path")
			require.NotContains(t, string(data), "private-bucket")
		})
	}
}

func TestStripLeadingTelemetryFlagsUsesDefinitions(t *testing.T) {
	flags := cli.NewFlagSet()
	flags.Add(&cli.Flag{Name: "cli-dry-run", Aliases: []string{"dry-run"}, AssignedMode: cli.AssignedNone})
	flags.Add(&cli.Flag{Name: "quiet", Shorthand: 'q', AssignedMode: cli.AssignedNone})
	flags.Add(&cli.Flag{Name: "profile", Aliases: []string{"profile-alias"}, Shorthand: 'p', AssignedMode: cli.AssignedOnce})
	flags.Add(&cli.Flag{Name: "fallback", AssignedMode: cli.AssignedDefault})
	flags.Add(&cli.Flag{Name: "header", Shorthand: 'H', AssignedMode: cli.AssignedRepeatable})
	command := []string{"configure", "telemetry", "purge"}
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{"unprefixed command", command, command},
		{"no value option", []string{"--cli-dry-run", "configure", "telemetry", "purge"}, command},
		{"long alias", []string{"--dry-run", "configure", "telemetry", "purge"}, command},
		{"short no value option", []string{"-q", "configure", "telemetry", "purge"}, command},
		{"separate option value", []string{"--profile", "private-profile", "configure", "telemetry", "purge"}, command},
		{"long value alias", []string{"--profile-alias", "private-profile", "configure", "telemetry", "purge"}, command},
		{"short option value", []string{"-p", "private-profile", "configure", "telemetry", "purge"}, command},
		{"default assignment", []string{"--fallback", "private-value", "configure", "telemetry", "purge"}, command},
		{"equals inline", []string{"--profile=private-profile", "configure", "telemetry", "purge"}, command},
		{"colon inline", []string{"--profile:private-profile", "configure", "telemetry", "purge"}, command},
		{"short inline", []string{"-p:private-profile", "configure", "telemetry", "purge"}, command},
		{"empty equals consumes next value", []string{"--profile=", "private-profile", "configure", "telemetry", "purge"}, command},
		{"empty colon consumes next value", []string{"--fallback:", "private-value", "configure", "telemetry", "purge"}, command},
		{"unknown short option is a single value", []string{"--profile", "-x", "configure", "telemetry", "purge"}, command},
		{"negative single value", []string{"--profile", "-123", "configure", "telemetry", "purge"}, command},
		{"known short option starts a boundary", []string{"--fallback", "-q", "configure", "telemetry", "purge"}, command},
		{"repeatable values until long option", []string{"--header", "private-one", "private-two", "--cli-dry-run", "configure", "telemetry", "purge"}, command},
		{"repeatable values until short option", []string{"-H", "private-one", "private-two", "-q", "configure", "telemetry", "purge"}, command},
		{"repeatable equals inline closes assignment", []string{"--header=private-value", "configure", "telemetry", "purge"}, command},
		{"repeatable colon inline closes assignment", []string{"--header:private-value", "configure", "telemetry", "purge"}, command},
		{"repeatable empty inline consumes all values", []string{"--header=", "private-one", "private-two", "--quiet", "configure", "telemetry", "purge"}, command},
		{"repeatable values include command looking tokens", []string{"--header", "private-value", "configure", "telemetry", "purge"}, nil},
		{"profile value resembles a command", []string{"--profile", "configure", "telemetry", "purge"}, []string{"telemetry", "purge"}},
		{"unknown long option", []string{"--unknown-switch", "configure", "telemetry", "purge"}, nil},
		{"unknown inline option", []string{"--unknown-switch=private-value", "configure", "telemetry", "purge"}, nil},
		{"unknown short option", []string{"-x", "configure", "telemetry", "purge"}, nil},
		{"unsupported short group", []string{"-pq", "configure", "telemetry", "purge"}, nil},
		{"positional terminator is unsupported by host", []string{"--", "configure", "telemetry", "purge"}, nil},
		{"missing option value", []string{"--profile"}, nil},
		{"no remaining command", []string{"--quiet"}, nil},
		{"empty invocation", nil, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, stripLeadingTelemetryFlags(test.args, flags))
			for _, flag := range flags.Flags() {
				require.False(t, flag.IsAssigned(), flag.Name)
			}
		})
	}
	// A new root switch is recognized from its definition without extending any
	// telemetry-specific list of boolean option names.
	flags.Add(&cli.Flag{Name: "future-root-switch", AssignedMode: cli.AssignedNone})
	require.Equal(t, command, stripLeadingTelemetryFlags(append([]string{"--future-root-switch"}, command...), flags))
}

func TestStripLeadingTelemetryFlagsDoesNotMutateFlagState(t *testing.T) {
	flags := cli.NewFlagSet()
	for _, flag := range []*cli.Flag{
		{Name: "switch", AssignedMode: cli.AssignedNone},
		{Name: "once", AssignedMode: cli.AssignedOnce},
		{Name: "default", AssignedMode: cli.AssignedDefault},
		{Name: "repeat", AssignedMode: cli.AssignedRepeatable},
	} {
		flag.SetAssigned(true)
		flag.SetValue("existing-value")
		if flag.AssignedMode == cli.AssignedRepeatable {
			flag.SetValues([]string{"existing-one", "existing-two"})
		}
		flags.Add(flag)
	}
	args := []string{"--switch", "--once", "private-once", "--default:private-default", "--repeat", "private-one", "private-two", "--switch", "configure", "telemetry", "purge"}
	require.Equal(t, []string{"configure", "telemetry", "purge"}, stripLeadingTelemetryFlags(args, flags))
	for _, flag := range flags.Flags() {
		value, assigned := flag.GetValue()
		require.True(t, assigned, flag.Name)
		require.True(t, flag.IsAssigned(), flag.Name)
		require.Equal(t, "existing-value", value, flag.Name)
		if flag.AssignedMode == cli.AssignedRepeatable {
			require.Equal(t, []string{"existing-one", "existing-two"}, flag.GetValues())
		}
	}
}

func TestStripLeadingTelemetryFlagsWithoutDefinitionsRetainsCompatibility(t *testing.T) {
	args := []string{"--cli-ai-mode", "--profile:private-profile", "-q", "configure", "telemetry", "purge"}
	require.Equal(t, []string{"configure", "telemetry", "purge"}, stripLeadingTelemetryFlags(args, nil))
}
