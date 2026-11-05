package cli

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsedFlagValidatorConfirmsDynamicOccurrences(t *testing.T) {
	ctx := NewCommandContext(io.Discard, io.Discard)
	command := &Command{Name: "aliyun", EnableUnknownFlag: true}
	command.Flags().Add(&Flag{Name: "profile", AssignedMode: AssignedOnce})
	ctx.EnterCommand(command)
	ctx.hasPluginSubCmd, ctx.hasPluginSubCmdSet = true, true
	args := []string{"--Name=private", "--profile:private", "--private-customer-token", "private", "--Name", "another"}
	ctx.SetInvocationArgs(args)
	_, err := NewParser(args, ctx).ReadAll()
	require.NoError(t, err)
	require.Equal(t, []string{"--profile"}, ctx.ParsedFlagNames(-1))

	ctx.SetParsedFlagValidator(func(name string) bool { return name == "--Name" })
	require.Equal(t, []string{"--Name", "--profile", "--Name"}, ctx.ParsedFlagNames(-1))
	require.Equal(t, []string{"--Name", "--profile"}, ctx.ParsedFlagNames(2))
	names := ctx.ParsedFlagNames(-1)
	names[0] = "mutated"
	require.Equal(t, []string{"--Name", "--profile", "--Name"}, ctx.ParsedFlagNames(-1))

	ctx.SetParsedFlagValidator(nil)
	require.Equal(t, []string{"--profile"}, ctx.ParsedFlagNames(-1))
	ctx.SetParsedFlagValidator(func(string) bool { return true })
	ctx.SetInvocationArgs([]string{"--Other", "private"})
	_, err = NewParser([]string{"--Other", "private"}, ctx).ReadAll()
	require.NoError(t, err)
	require.Empty(t, ctx.ParsedFlagNames(-1), "a new invocation must clear the previous schema")

	// A schema is applied only to occurrences actually identified as flags.
	ctx.SetParsedFlagValidator(func(name string) bool { return strings.HasPrefix(name, "--") })
	require.Equal(t, []string{"--Other"}, ctx.ParsedFlagNames(-1))
	var absent *Context
	absent.SetParsedFlagValidator(nil)
	require.Empty(t, absent.ParsedFlagNames(-1))
}
