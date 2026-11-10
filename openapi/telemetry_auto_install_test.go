package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

const telemetryAutoInstallProduct = "autotelemetryfixture"

func TestMain(m *testing.M) {
	// End re-executes this binary for upload. Never run the test suite again or
	// send its fixture events to the production telemetry endpoint.
	if len(os.Args) > 1 && os.Args[1] == "__telemetry-upload" {
		os.Exit(99)
	}
	if os.Getenv("ALIYUN_TEST_TELEMETRY_GO_PLUGIN_CHILD") == "1" &&
		len(os.Args) > 1 && os.Args[1] == telemetryAutoInstallProduct {
		fmt.Fprintln(os.Stdout, "telemetry plugin fixture")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func telemetryAutoInstallExecutable(t *testing.T, dir, name string) {
	t.Helper()
	sourcePath, err := os.Executable()
	require.NoError(t, err)
	source, err := os.Open(sourcePath)
	require.NoError(t, err)
	defer source.Close()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	destination, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	require.NoError(t, err)
	defer destination.Close()
	_, err = io.Copy(destination, source)
	require.NoError(t, err)
	require.NoError(t, destination.Close())
}

func TestTelemetryFirstAutoInstalledPluginUsesSelectedRoute(t *testing.T) {
	pluginName := "aliyun-cli-" + telemetryAutoInstallProduct
	binaryDir := t.TempDir()
	telemetryAutoInstallExecutable(t, binaryDir, pluginName)
	for _, test := range []struct {
		name, pluginType, operation string
		inner, missingBinary        bool
	}{
		{"inner_go", plugin.PluginTypeGo, "function", true, false},
		{"public_go", plugin.PluginTypeGo, "function", false, false},
		{"inner_meta", plugin.PluginTypeMeta, "list", true, false},
		{"public_meta", plugin.PluginTypeMeta, "list", false, false},
		{"inner_meta_version", plugin.PluginTypeMeta, "version", true, false},
		{"public_meta_version", plugin.PluginTypeMeta, "version", false, false},
		{"inner_go_missing_binary", plugin.PluginTypeGo, "function", true, true},
		{"public_go_missing_binary", plugin.PluginTypeGo, "function", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			home := t.TempDir()
			t.Cleanup(setTestHomeDir(t, home))
			writeMinimalConfigJSON(t, home)
			configDir := filepath.Join(home, ".aliyun")
			pluginsDir := filepath.Join(configDir, "plugins")
			require.NoError(t, os.MkdirAll(pluginsDir, 0700))
			t.Setenv(plugin.EnvPluginsDir, pluginsDir)
			t.Setenv(telemetry.TelemetryDisableEnv, "0")
			t.Setenv(telemetry.TelemetryLogDisableEnv, "1")
			t.Setenv("ALIYUN_TEST_TELEMETRY_GO_PLUGIN_CHILD", "1")
			t.Cleanup(func() {
				// Also discard active state if an earlier assertion aborted this case.
				_ = os.Setenv(telemetry.TelemetryDisableEnv, "1")
				telemetry.End(nil)
			})

			stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
			ctx := cli.NewCommandContext(stdout, stderr)
			repo, err := meta.MockLoadRepository(nil)
			require.NoError(t, err)
			command := &Commando{library: &Library{builtinRepo: repo, writer: stdout}}
			cmd := &cli.Command{EnableUnknownFlag: true}
			command.InitWithCommand(cmd)
			AddFlags(cmd.Flags())
			ctx.EnterCommand(cmd)
			ctx.Command().Short = &i18n.Text{}

			args := []string{"AutoTelemetryFixture", test.operation}
			if test.pluginType == plugin.PluginTypeGo {
				args = append(args, "List")
			}
			rawArgs := append([]string(nil), args...)
			if test.pluginType == plugin.PluginTypeGo {
				rawArgs = append(rawArgs, "private-tail-sentinel")
			}
			rawArgs = append(rawArgs, "--private-input", "private-value-sentinel")
			previousArgs := os.Args
			os.Args = append([]string{"aliyun"}, rawArgs...)
			t.Cleanup(func() { os.Args = previousArgs })
			previousTryDispatch, previousDispatch, previousInstall := runtimeTryDispatch, runtimeDispatch, installPluginForCommand
			t.Cleanup(func() {
				runtimeTryDispatch, runtimeDispatch, installPluginForCommand = previousTryDispatch, previousDispatch, previousInstall
			})
			runtimeTryDispatch = func(*cli.Context, []string) (bool, error) { return false, nil }
			runtimeCalls := 0
			runtimeDispatch = func(_ *cli.Context, got []string) error {
				runtimeCalls++
				require.Equal(t, rawArgs, got)
				telemetry.SetOpenAPIRuntimeVersion("runtime-fixture")
				telemetry.SetResolvedMetaCommand(got[0], got[1])
				return nil
			}
			installCalls := 0
			installPluginForCommand = func(_ *Commando, _ *cli.Context, gotName, product string) (string, error) {
				installCalls++
				require.Equal(t, strings.Join(args[:2], " "), gotName)
				require.Equal(t, args[0], product)
				local, err := plugin.LookupLocalPlugin(product)
				require.NoError(t, err)
				require.Nil(t, local, "the plugin must still be absent at the first installation")
				path := binaryDir
				if test.missingBinary {
					path = filepath.Join(pluginsDir, "missing-binary")
				}
				required := false
				manifest := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
					pluginName: {
						Name: pluginName, Command: telemetryAutoInstallProduct, ProductCode: telemetryAutoInstallProduct,
						Version: "1.2.3", Type: test.pluginType, Inner: test.inner,
						Path: path, ProfileRequired: &required,
						CmdNames: []string{"function list"},
					},
				}}
				data, err := json.Marshal(manifest)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), data, 0600))
				return pluginName, nil
			}

			before, err := plugin.LookupLocalPlugin(args[0])
			require.NoError(t, err)
			require.Nil(t, before)
			telemetry.Begin(telemetry.BeginInput{ConfigDir: configDir, CommandToken: args[0], CommandArgs: rawArgs})
			err = command.main(ctx, args)
			telemetry.End(err)
			require.Equal(t, 1, installCalls)
			if test.missingBinary {
				require.ErrorContains(t, err, "plugin binary not found")
			} else {
				require.NoError(t, err, stderr.String())
			}
			pipeline := telemetry.PipelinePublic
			otherPipeline := telemetry.PipelineInner
			if test.inner {
				pipeline, otherPipeline = otherPipeline, pipeline
			}
			cacheDir := telemetry.GetCacheDir(configDir, pipeline)
			files, err := os.ReadDir(cacheDir)
			require.NoError(t, err)
			require.Len(t, files, 1)
			data, err := os.ReadFile(filepath.Join(cacheDir, files[0].Name()))
			require.NoError(t, err)
			var event telemetry.Event
			require.NoError(t, json.Unmarshal(data, &event))
			require.Equal(t, pipeline, event.Pipeline)
			require.Equal(t, pluginName+"@1.2.3", event.Plugin)
			require.Equal(t, strings.Join(args, " "), event.Command)
			if test.pluginType == plugin.PluginTypeGo {
				require.Equal(t, []string{"--private-input"}, event.Params)
			} else {
				require.Empty(t, event.Params, "a metadata route requires a parser snapshot")
			}
			require.NotContains(t, string(data), "private-value-sentinel")
			require.NotContains(t, string(data), "private-tail-sentinel")
			require.NoDirExists(t, telemetry.GetCacheDir(configDir, otherPipeline))
			if test.pluginType == plugin.PluginTypeGo {
				require.Equal(t, telemetry.CommandTypePluginGo, event.CommandType)
				require.Empty(t, event.OpenAPIRuntimeVersion)
				require.Zero(t, runtimeCalls)
				if !test.missingBinary {
					require.Contains(t, stdout.String(), "telemetry plugin fixture")
				}
			} else {
				require.Equal(t, telemetry.CommandTypePluginMeta, event.CommandType)
				if test.operation == "version" {
					require.Zero(t, runtimeCalls)
					require.Empty(t, event.OpenAPIRuntimeVersion)
					require.Contains(t, stdout.String(), pluginName+" 1.2.3")
				} else {
					require.Equal(t, 1, runtimeCalls)
					require.Equal(t, "runtime-fixture", event.OpenAPIRuntimeVersion)
				}
			}
		})
	}
}
