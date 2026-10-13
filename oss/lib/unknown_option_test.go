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
		{[]string{"--versin"}, "invalid option --versin for oss commands, did you mean --version?"},
		{[]string{"--recursiv"}, "invalid option --recursiv for oss commands, did you mean --recursive?"},
		{[]string{"--zzzzzz=private-value"}, "invalid option --zzzzzz for oss commands." + help},
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
	t.Cleanup(func() { OptionMap = original })
	OptionMap = make(map[string]Option)
	for _, suffix := range []string{"g", "d", "b", "f", "c", "e", "a"} {
		name := "custom" + suffix
		OptionMap[name] = Option{nameAlias: "--" + name}
	}
	for _, input := range []string{"--customz", "--cust"} {
		t.Run(input, func(t *testing.T) {
			want := "invalid option " + input + " for oss commands, did you mean --customa or --customb or --customc or --customd or --custome?"
			for i := 0; i < 20; i++ {
				require.EqualError(t, ossUnknownOptionError(input, ""), want)
			}
		})
	}
}

func TestUnknownOSSOptionGlobalClassificationPrecedesSuggestions(t *testing.T) {
	original := OptionMap
	t.Cleanup(func() { OptionMap = original })
	OptionMap = map[string]Option{"headers": {nameAlias: "--headers"}}
	require.EqualError(t, ossUnknownOptionError("--header", "ls"), "--header is not an oss command option.\nRun 'aliyun oss ls --help' for supported oss options.")
}

func TestKnownOSSOptionRejectedByBridgeIsNotGlobalOrSuggested(t *testing.T) {
	for _, name := range []string{"--force", "--version", "--endpoint"} {
		t.Run(name, func(t *testing.T) {
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			ctx.SetCommand(&cli.Command{Name: "ls"})
			_, _, err := parseBridgeFlags(ctx, []string{name})
			require.EqualError(t, err, "invalid option "+name+" for oss commands.\nRun 'aliyun oss ls --help' for supported oss options.")
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
