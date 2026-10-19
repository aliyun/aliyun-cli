package cli

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelpOptionErrorsHaveStableHumanMessages(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--help-all", "--help-all"}, "--help-all duplicated"},
		{[]string{"--help", "--help-search", "region"}, "conflicts with"},
		{[]string{"--help-search"}, "requires a non-empty query"},
		{[]string{"--cli-output", "yaml"}, "only supports json"},
		{[]string{"--cli-section", "invalid"}, "must be request or response"},
	} {
		_, err := ParseHelpOptions(tc.args)
		require.ErrorContains(t, err, tc.want)
		var typed *HelpOptionError
		require.ErrorAs(t, err, &typed)
	}
	cause := errors.New("mutually exclusive flags")
	wrapped := &InvalidOptionCombinationError{Err: cause}
	require.ErrorIs(t, wrapped, cause)
	require.Equal(t, cause.Error(), wrapped.Error())
	var absent *InvalidOptionCombinationError
	require.Nil(t, absent.Unwrap())
	require.Equal(t, "invalid option combination", absent.Error())
	require.Equal(t, "invalid option combination", (&InvalidOptionCombinationError{}).Error())
	wrapped.AIRecoveryEligible()
}

func TestFlagRecoveryUsesCompleteCommandPath(t *testing.T) {
	ctx := NewCommandContext(io.Discard, io.Discard)
	root := &Command{Name: "aliyun"}
	ctx.EnterCommand(root)
	err := NewInvalidFlagError("bad", ctx).(*InvalidFlagError)
	require.Equal(t, "aliyun help", err.AgentHelpCommand())
	configure := &Command{Name: "configure"}
	root.AddSubCommand(configure)
	ctx.EnterCommand(configure)
	require.Equal(t, "aliyun help configure", err.AgentHelpCommand())
	get := &Command{Name: "get"}
	configure.AddSubCommand(get)
	ctx.EnterCommand(get)
	require.Equal(t, "aliyun help configure get", err.AgentHelpCommand())
	var absent *Context
	absent.SetInvocationArgs([]string{"aliyun"})
	require.Nil(t, absent.InvocationArgs())
}

func TestCommandKeepsRawArgsAndCustomCompletionResults(t *testing.T) {
	var output bytes.Buffer
	ctx := NewCommandContext(&output, io.Discard)
	var got []string
	command := &Command{Name: "test", KeepArgs: true, Run: func(_ *Context, args []string) error { got = append([]string(nil), args...); return nil }}
	ctx.EnterCommand(command)
	require.NoError(t, command.executeInner(ctx, []string{"input", "--provider-specific=value"}))
	require.Equal(t, []string{"input", "--provider-specific=value"}, got)
	command = &Command{Name: "complete", AutoComplete: func(_ *Context, args []string) []string {
		require.Equal(t, []string{"prefix"}, args)
		return []string{"prefix-one", "prefix-two"}
	}, Run: func(*Context, []string) error { t.Fatal("completion ran command"); return nil }}
	ctx = NewCommandContext(&output, io.Discard)
	ctx.EnterCommand(command)
	ctx.completion = &Completion{}
	require.NoError(t, command.executeInner(ctx, []string{"prefix"}))
	require.Equal(t, "prefix-one\nprefix-two\n", output.String())
	ctx = NewCommandContext(&output, io.Discard)
	ctx.EnterCommand(&Command{Name: "flags"})
	ctx.Flags().Add(&Flag{Name: "selected"})
	ctx.Flags().Add(&Flag{Name: "other"})
	ctx.completion = &Completion{Current: "--sel"}
	output.Reset()
	ctx.command.ExecuteComplete(ctx, nil)
	require.Equal(t, "--selected\n", output.String())
	ctx.SetCommand(&Command{Name: "unknown", EnableUnknownFlag: true})
	require.NotNil(t, ctx.unknownFlags)
}
