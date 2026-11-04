package telemetry

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/stretchr/testify/require"
)

func TestBuildEventUsesOnlyConfirmedParameterNames(t *testing.T) {
	params := []meta.Parameter{
		{Name: "name", RawName: "Name", Type: meta.TypeString, Options: []string{"--Name", "--alias"}},
		{Name: "names", RawName: "Names", Type: meta.TypeArray, ItemType: &meta.Parameter{Type: meta.TypeString}, Options: []string{"--names"}},
		{Name: "count", RawName: "Count", Type: meta.TypeInteger, Options: []string{"--count"}},
	}
	for _, test := range []struct {
		name  string
		args  []string
		want  []string
		fails bool
	}{
		{"scalar", []string{"--Name", "--private-business-value"}, []string{"--Name"}, false},
		{"array", []string{"--names", "a", "--private-business-value", "--Name", "public"}, []string{"--names", "--Name"}, false},
		{"case order repeats", []string{"--Name=public", "--alias", "public", "--Name", "public"}, []string{"--Name", "--alias", "--Name"}, false},
		{"unknown flag", []string{"--Name=public", "--private-business-value"}, []string{"--Name"}, true},
		{"invalid type", []string{"--Name=public", "--count", "--private-business-value"}, []string{"--Name", "--count"}, true},
		{"missing value", []string{"--Name=public", "--count"}, []string{"--Name", "--count"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var names []string
			_, parseErr := argparser.ParseWithOptions(params, test.args, argparser.ParseOptions{OnParameterNames: func(snapshot []string) { names = snapshot }})
			if test.fails {
				require.Error(t, parseErr)
			} else {
				require.NoError(t, parseErr)
			}
			input := BuildInput{RootArgs: append([]string{"demo", "run"}, test.args...), ParameterNames: names, Err: parseErr}
			event, err := BuildEvent(input)
			require.NoError(t, err)
			require.Equal(t, test.want, event.Params)
			require.Equal(t, test.want, newUploadEvent(event).Attributes["parameterNames"])
			encoded, err := json.Marshal(newUploadEvent(event))
			require.NoError(t, err)
			require.NotContains(t, string(encoded), "private-business-value")
			names[0] = "changed"
			require.Equal(t, test.want, event.Params, "events own their parameter snapshot")
			input.ParameterNames = nil
			event, err = BuildEvent(input)
			require.NoError(t, err)
			require.Empty(t, event.Params)
			require.NotContains(t, newUploadEvent(event).Attributes, "parameterNames", "no raw argv fallback")
		})
	}
}

func TestRuntimeParameterSnapshotReplacesHostTail(t *testing.T) {
	t.Setenv(TelemetryDisableEnv, "0")
	t.Setenv(TelemetryLogDisableEnv, "1")
	t.Cleanup(resetExecutionState)
	dir := collectionPanicConfigDir(t)
	root := &cli.Command{Name: "aliyun", EnableUnknownFlag: true, AfterExecute: HookEnd}
	root.Flags().Add(&cli.Flag{Name: "profile", AssignedMode: cli.AssignedOnce})
	root.Flags().Add(&cli.Flag{Name: "region", AssignedMode: cli.AssignedOnce})
	root.Flags().Add(&cli.Flag{Name: "Name", AssignedMode: cli.AssignedRepeatable})
	root.Flags().Add(&cli.Flag{Name: "cli-dry-run", AssignedMode: cli.AssignedNone})
	root.Run = func(*cli.Context, []string) error {
		SetRuntimeParameterNames([]string{"--discarded-snapshot"})
		names := []string{"--Name", "--Name", "--region", "--cli-dry-run"}
		SetRuntimeParameterNames(names)
		names[0] = "--mutated-snapshot"
		return nil
	}
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	ctx.EnterCommand(root)
	args := []string{"--profile", "default", "demo", "run", "--Name", "private-value", "--Name", "private-value", "--region", "other", "--cli-dry-run"}
	Begin(BeginInput{ConfigDir: dir, Context: ctx, CommandArgs: args})
	root.Execute(ctx, args)
	event := readPluginRouteEvent(t, dir, PipelinePublic)
	require.Equal(t, []string{"--profile", "--Name", "--Name", "--region", "--cli-dry-run"}, event.Params)
	SetRuntimeParameterNames([]string{"--late"})
	require.NotContains(t, activeExecution.parameterNames, "--late")
}

func TestRuntimeParameterObserverWithoutExecutionDoesNothing(t *testing.T) {
	resetExecutionState()
	t.Cleanup(resetExecutionState)
	SetRuntimeParameterNames([]string{"--Name"})
	require.Nil(t, activeExecution)
}
