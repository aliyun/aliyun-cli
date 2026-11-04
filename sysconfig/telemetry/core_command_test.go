package telemetry

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/stretchr/testify/require"
)

func TestCoreCommandUsesParsedPathBeforeDirectExit(t *testing.T) {
	t.Setenv(TelemetryDisableEnv, "0")
	t.Setenv(TelemetryLogDisableEnv, "1")
	t.Cleanup(resetExecutionState)
	dir := collectionPanicConfigDir(t)
	root := &cli.Command{Name: "aliyun", AfterExecute: HookEnd}
	configure := &cli.Command{Name: "configure"}
	set := &cli.Command{Name: "set", Run: func(*cli.Context, []string) error {
		// Direct exits finalize telemetry before AfterExecute can observe it.
		EndWithExitCode(7)
		return nil
	}}
	set.Flags().Add(&cli.Flag{Name: "secret", AssignedMode: cli.AssignedOnce})
	configure.AddSubCommand(set)
	root.AddSubCommand(configure)
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	ctx.EnterCommand(root)
	args := []string{"configure", "set", "--secret", "private-value-sentinel", "private-tail-sentinel"}
	Begin(BeginInput{ConfigDir: dir, Context: ctx, CommandToken: "configure", CommandArgs: args})
	root.Execute(ctx, args)
	event := readPluginRouteEvent(t, dir, PipelinePublic)
	require.Equal(t, "configure set", event.Command)
	require.Equal(t, "Failure", event.Result)
	require.Equal(t, []string{"--secret"}, event.Params)
	data, err := json.Marshal(newUploadEvent(event))
	require.NoError(t, err)
	require.NotContains(t, string(data), "private-value-sentinel")
	require.NotContains(t, string(data), "private-tail-sentinel")
}

func TestCoreCommandWithoutParsedPathKeepsOnlyRoot(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		path []string
		want string
	}{
		{"unresolved core path", []string{"configure", "set", "private-tail-sentinel"}, nil, "configure"},
		{"different resolved command", []string{"configure", "private-tail-sentinel"}, []string{"version"}, "configure"},
		{"unresolved help target", []string{"help", "private-tail-sentinel"}, nil, "help"},
		{"non-core help target", []string{"help", "private-tail-sentinel"}, []string{"unknown-product"}, "help"},
		{"different root arguments", []string{"private-tail-sentinel", "set"}, []string{"configure", "set"}, "configure"},
	} {
		t.Run(test.name, func(t *testing.T) {
			event, err := BuildEvent(BuildInput{
				Scope:    Scope{CommandToken: test.want, Pipeline: PipelinePublic},
				RootArgs: test.args, CommandPath: test.path, InstanceID: "test-installation",
			})
			require.NoError(t, err)
			require.Equal(t, test.want, event.Command)
			data, err := json.Marshal(newUploadEvent(event))
			require.NoError(t, err)
			require.NotContains(t, string(data), "private-tail-sentinel")
		})
	}
}

func TestTelemetryManagementParsedPathDiscardsMisclassifiedExecution(t *testing.T) {
	t.Setenv(TelemetryDisableEnv, "0")
	t.Setenv(TelemetryLogDisableEnv, "1")
	t.Setenv(plugin.EnvPluginsDir, t.TempDir())
	t.Cleanup(resetExecutionState)
	for _, directExit := range []bool{false, true} {
		name := "completion_callback"
		if directExit {
			name = "exit_callback"
		}
		t.Run(name, func(t *testing.T) {
			dir := collectionPanicConfigDir(t)
			cacheDir := GetPublicCacheDir(dir)
			require.NoError(t, os.MkdirAll(cacheDir, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "old-event.json"), []byte("{}"), 0600))
			root := &cli.Command{Name: "aliyun", AfterExecute: HookEnd}
			configure := &cli.Command{Name: "configure"}
			management := &cli.Command{Name: "telemetry"}
			management.AddSubCommand(&cli.Command{Name: "purge", Run: func(*cli.Context, []string) error {
				require.NoError(t, Purge(dir))
				if directExit {
					EndWithExitCode(0)
				}
				return nil
			}})
			configure.AddSubCommand(management)
			root.AddSubCommand(configure)
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			ctx.EnterCommand(root)
			// Model an entry classifier that already lost "configure". The
			// final guard must use the actual route despite this stale input.
			Begin(BeginInput{ConfigDir: dir, Context: ctx, CommandArgs: []string{"telemetry", "purge"}})
			require.NotNil(t, activeExecution)
			root.Execute(ctx, []string{"configure", "telemetry", "purge"})
			End(nil)
			require.True(t, executionEnded)
			require.NoFileExists(t, GetInstanceIDFilePath(dir))
			require.NoDirExists(t, filepath.Join(dir, CacheDirName))
		})
	}
}
