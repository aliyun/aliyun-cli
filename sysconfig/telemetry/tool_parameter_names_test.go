package telemetry

import (
	"io"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/stretchr/testify/require"
)

func TestInferToolParameterNames(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{"multiple values", []string{"demo", "list", "--tag", "one", "two", "--other", "three"}, []string{"--tag", "--other"}},
		{"case order repeats and inline", []string{"demo", "list", "--Tag=one", "--tag:two", "--Tag", "three", "-t=four", "-t:five"}, []string{"--Tag", "--tag", "--Tag", "-t", "-t"}},
		{"consecutive options", []string{"--debug", "--verbose", "--output=json"}, []string{"--debug", "--verbose", "--output"}},
		{"negative and certificate values", []string{"--count", "-123", "--cert", "-----BEGIN PRIVATE KEY-----", "--path", "/private/file"}, []string{"--count", "--cert", "--path"}},
		{"ambiguous short value", []string{"--body", "-x", "--output=json", "-q"}, []string{"--body", "--output", "-q"}},
		{"empty inline assignment", []string{"--body=", "-x", "--name:", "-y", "--output=json"}, []string{"--body", "--name", "--output"}},
		{"invalid forms", []string{"-private", "-123", "---certificate", "--/private/path", "--private body", "--{private:body}", "--valid"}, []string{"--valid"}},
		{"literal delimiter", []string{"--valid=one", "--", "--not-an-option", "-x"}, []string{"--valid"}},
		{"accepted unknown boundary ambiguity", []string{"--name", "--ambiguous-value"}, []string{"--name", "--ambiguous-value"}},
		{"no named options", []string{"demo", "list", "positional-value"}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := append([]string(nil), test.args...)
			require.Equal(t, test.want, inferToolParameterNames(test.args))
			require.Equal(t, original, test.args)
		})
	}
}

func TestParameterInferenceIsLimitedToOpaqueToolRoutes(t *testing.T) {
	for _, test := range []struct {
		name  string
		scope Scope
		infer bool
	}{
		{"Go", Scope{CommandType: CommandTypePluginGo}, true},
		{"extension", Scope{CommandType: CommandTypeCLIExt}, true},
		{"core", Scope{CommandType: CommandTypeCore}, false},
		{"OpenAPI runtime", Scope{CommandType: CommandTypeOpenAPIRuntime}, false},
		{"OpenAPI legacy", Scope{CommandType: CommandTypeOpenAPILegacy}, false},
		{"metadata plugin", Scope{CommandType: CommandTypePluginMeta}, false},
		{"installed Go candidate before dispatch", Scope{PluginType: plugin.PluginTypeGo}, false},
		{"selected native route overrides plugin candidate", Scope{CommandType: CommandTypeOpenAPILegacy, PluginType: plugin.PluginTypeGo}, false},
		{"unresolved OpenAPI route", Scope{}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			execution := &activeCommandExecution{scope: test.scope, rootArgs: []string{"demo", "list", "--name", "--ambiguous-value"}}
			if test.infer {
				require.Equal(t, []string{"--name", "--ambiguous-value"}, executionParameterNames(execution))
			} else {
				require.Empty(t, executionParameterNames(execution))
			}
			// Any authoritative snapshot takes precedence, including an empty
			// snapshot after an unknown first flag or a parsing failure.
			execution.runtimeParametersSet = true
			execution.parameterNames = []string{"--name"}
			require.Equal(t, []string{"--name"}, executionParameterNames(execution))
			execution.parameterNames = nil
			require.Empty(t, executionParameterNames(execution))
		})
	}
}

func TestToolParameterInferenceKeepsHostPrefixWithoutDuplicates(t *testing.T) {
	args := []string{"--profile", "default", "demo", "list", "--Tag=one", "--Tag=two", "--region=region-value"}
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	root := &cli.Command{EnableUnknownFlag: true}
	root.Flags().Add(&cli.Flag{Name: "profile", AssignedMode: cli.AssignedOnce})
	root.Flags().Add(&cli.Flag{Name: "Tag", AssignedMode: cli.AssignedRepeatable})
	root.Flags().Add(&cli.Flag{Name: "region", AssignedMode: cli.AssignedOnce})
	ctx.EnterCommand(root)
	ctx.SetInvocationArgs(args)
	_, err := cli.NewParser(args, ctx).ReadAll()
	require.NoError(t, err)
	execution := &activeCommandExecution{
		context: ctx, scope: Scope{CommandType: CommandTypePluginGo},
		rootArgs: args[2:], commandArgOffset: 2,
	}
	require.Equal(t, []string{"--profile", "--Tag", "--Tag", "--region"}, executionParameterNames(execution))
}
