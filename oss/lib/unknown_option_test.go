package lib

import (
	"io"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/openapi"
	"github.com/stretchr/testify/require"
)

func TestUnknownOSSOptionDiagnostics(t *testing.T) {
	const help = "\nRun 'aliyun oss ls --help' for supported oss options."
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--header", "private-header-value"}, "--header is not an oss command option." + help},
		{[]string{"--header=private-header-value"}, "--header is not an oss command option." + help},
		{[]string{"--dryrun"}, "--dryrun is not an oss command option." + help},
		{[]string{"--waiter", "expr=value"}, "--waiter is not an oss command option." + help},
		{[]string{"--api-version=2026-01-01"}, "--api-version is not an oss command option." + help},
		{[]string{"--cli-dry-run-json"}, "--cli-dry-run-json is not an oss command option." + help},
		{[]string{"-q"}, "-q is not an oss command option." + help},
		{[]string{"--versin"}, "invalid option --versin for oss ls." + help},
		{[]string{"--recursiv"}, "invalid option --recursiv for oss ls." + help},
		{[]string{"--vers"}, "invalid option --vers for oss ls, did you mean --version-id-marker?"},
		{[]string{"--markr"}, "invalid option --markr for oss ls, did you mean --marker?"},
		{[]string{"--m"}, "invalid option --m for oss ls, did you mean -m?"},
		{[]string{"--zzzzzz=private-value"}, "invalid option --zzzzzz for oss ls." + help},
	} {
		t.Run(tc.args[0], func(t *testing.T) {
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			ctx.SetCommand(NewCommandBridge(listCommand.command))
			_, _, err := parseBridgeFlags(ctx, tc.args)
			require.EqualError(t, err, tc.want)
			_, _, err = parseOSSOptions(tc.args, "ls")
			require.EqualError(t, err, tc.want)
		})
	}
}

func TestUnknownOSSOptionSuggestionsAreSortedAndLimited(t *testing.T) {
	original := OptionMap
	originalValid := listCommand.command.validOptionNames
	t.Cleanup(func() {
		OptionMap = original
		listCommand.command.validOptionNames = originalValid
	})
	OptionMap = make(map[string]Option)
	listCommand.command.validOptionNames = []string{"missing-definition"}
	for _, suffix := range []string{"g", "d", "b", "f", "c", "e", "a"} {
		name := "custom" + suffix
		OptionMap[name] = Option{nameAlias: "--" + name}
		listCommand.command.validOptionNames = append(listCommand.command.validOptionNames, name)
	}
	OptionMap["unrelated"] = Option{nameAlias: "--custom0"}
	for _, input := range []string{"--customz", "--cust"} {
		t.Run(input, func(t *testing.T) {
			want := "invalid option " + input + " for oss ls, did you mean --customa or --customb or --customc or --customd or --custome?"
			for i := 0; i < 20; i++ {
				require.EqualError(t, ossUnknownOptionError(input, "ls"), want)
			}
		})
	}
}

func TestUnknownOSSOptionGlobalClassificationPrecedesSuggestions(t *testing.T) {
	original := OptionMap
	originalValid := listCommand.command.validOptionNames
	t.Cleanup(func() {
		OptionMap = original
		listCommand.command.validOptionNames = originalValid
	})
	OptionMap = map[string]Option{"headers": {nameAlias: "--headers"}}
	listCommand.command.validOptionNames = []string{"headers"}
	require.EqualError(t, ossUnknownOptionError("--header", "ls"), "--header is not an oss command option.\nRun 'aliyun oss ls --help' for supported oss options.")
}

func TestKnownOSSOptionRejectedByBridgeIsNotGlobalOrSuggested(t *testing.T) {
	for _, name := range []string{"--force", "--version", "--version-id"} {
		t.Run(name, func(t *testing.T) {
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			ctx.SetCommand(NewCommandBridge(listCommand.command))
			_, _, err := parseBridgeFlags(ctx, []string{name})
			require.EqualError(t, err, "invalid option "+name+" for oss ls.\nRun 'aliyun oss ls --help' for supported oss options.")
		})
	}
}

func TestUnknownOSSOptionSuggestionsFollowCommand(t *testing.T) {
	for _, tc := range []struct {
		command Command
		input   string
		want    string
	}{
		{copyCommand.command, "--vers", "--version-id"},
		{copyCommand.command, "--recursiv", "--recursive"},
		{listCommand.command, "--vers", "--version-id-marker"},
		{listCommand.command, "--markr", "--marker"},
	} {
		t.Run(tc.command.name+"/"+tc.input, func(t *testing.T) {
			want := "invalid option " + tc.input + " for oss " + tc.command.name + ", did you mean " + tc.want + "?"
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			ctx.SetCommand(NewCommandBridge(tc.command))
			_, _, err := parseBridgeFlags(ctx, []string{tc.input})
			require.EqualError(t, err, want)
			_, _, err = parseOSSOptions([]string{tc.input}, tc.command.name)
			require.EqualError(t, err, want)
		})
	}
}

func TestUnknownOSSOptionWithoutKnownCommandDoesNotSuggest(t *testing.T) {
	for _, command := range []string{"", "list", "not-a-command"} {
		t.Run(command, func(t *testing.T) {
			require.EqualError(t, ossUnknownOptionError("--vers", command),
				"invalid option --vers for oss commands.\nRun 'aliyun oss --help' for supported oss options.")
		})
	}
}

func TestUnknownOSSOptionAfterTerminatorRemainsPositional(t *testing.T) {
	args := []string{"--", "--header=private-value", "--versin"}
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	ctx.SetCommand(NewCommandBridge(listCommand.command))
	positional, help, err := parseBridgeFlags(ctx, args)
	require.NoError(t, err)
	require.False(t, help)
	require.Equal(t, args[1:], positional)
	positional, _, err = parseOSSOptions(args, "ls")
	require.NoError(t, err)
	require.Equal(t, args[1:], positional)
}

func TestUnsupportedOpenAPIOptionsSurviveBridgeForwarding(t *testing.T) {
	for _, args := range [][]string{
		{"--header", "private-header-value"},
		{"--header=private-header-value"},
		{"--waiter", "expr=value"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			ctx.SetCommand(NewCommandBridge(listCommand.command))
			ctx.Flags().Add(openapi.NewHeaderFlag())
			waiter := *openapi.WaiterFlag
			ctx.Flags().Add(&waiter)
			_, _, err := parseBridgeFlags(ctx, args)
			require.NoError(t, err)
			forwarded := stripCliOnlyFlagsFromArgs(append([]string{"--profile", "test-profile"}, args...))
			require.Equal(t, args, forwarded)
			_, _, err = parseOSSOptions(forwarded, "ls")
			key, _, _ := strings.Cut(args[0], "=")
			require.EqualError(t, err, key+" is not an oss command option.\nRun 'aliyun oss ls --help' for supported oss options.")
			require.NotContains(t, err.Error(), "private-header-value")
		})
	}
}
