package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsedFlagNamesFollowActualParser(t *testing.T) {
	for _, test := range []struct {
		name  string
		args  []string
		want  []string
		fails bool
	}{
		{"case aliases and repeats", []string{"--Name=private", "--alias:private", "-n", "private", "--Name", "private"}, []string{"--Name", "--alias", "-n", "--Name"}, false},
		{"dash leading scalar", []string{"--Name", "-private-value"}, []string{"--Name"}, false},
		{"unknown dynamic names", []string{"--Name", "private", "--private-business-value", "private"}, []string{"--Name"}, false},
		{"assignment failure retains name", []string{"--Name", "one", "--Name", "two"}, []string{"--Name", "--Name"}, true},
		{"value assignment failure retains name", []string{"--switch=private"}, []string{"--switch"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := NewCommandContext(io.Discard, io.Discard)
			command := &Command{Name: "aliyun", EnableUnknownFlag: true}
			mode := AssignedRepeatable
			if test.name == "assignment failure retains name" {
				mode = AssignedOnce
			}
			command.Flags().Add(&Flag{Name: "Name", Aliases: []string{"alias"}, Shorthand: 'n', AssignedMode: mode})
			command.Flags().Add(&Flag{Name: "switch", AssignedMode: AssignedNone})
			ctx.EnterCommand(command)
			ctx.SetInvocationArgs(test.args)
			parser := NewParser(test.args, ctx)
			_, err := parser.ReadAll()
			if err == nil {
				err = ctx.CheckFlags()
			}
			if test.fails {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, test.want, ctx.ParsedFlagNames(-1))
			names := ctx.ParsedFlagNames(-1)
			names[0] = "mutated"
			require.Equal(t, test.want, ctx.ParsedFlagNames(-1))
			ctx.SetInvocationArgs(nil)
			require.Empty(t, ctx.ParsedFlagNames(-1))
		})
	}
}

func TestParsedFlagNamesTrackNestedCommandPositions(t *testing.T) {
	root := &Command{Name: "aliyun"}
	root.Flags().Add(&Flag{Name: "profile", AssignedMode: AssignedOnce, Persistent: true})
	group := &Command{Name: "group"}
	group.Flags().Add(&Flag{Name: "region", AssignedMode: AssignedOnce, Persistent: true})
	leaf := &Command{Name: "leaf", Run: func(*Context, []string) error { return nil }}
	leaf.Flags().Add(&Flag{Name: "Name", AssignedMode: AssignedRepeatable})
	leaf.Flags().Add(&Flag{Name: "debug", AssignedMode: AssignedNone})
	group.AddSubCommand(leaf)
	root.AddSubCommand(group)
	ctx := NewCommandContext(io.Discard, io.Discard)
	ctx.EnterCommand(root)
	args := []string{"--profile", "private", "group", "--region=private", "leaf", "--Name:private", "--debug"}
	ctx.SetInvocationArgs(args)
	require.NoError(t, root.executeInner(ctx, args))
	require.Equal(t, []string{"--profile", "--region", "--Name", "--debug"}, ctx.ParsedFlagNames(-1))
	require.Equal(t, []string{"--profile"}, ctx.ParsedFlagNames(2))
	require.Equal(t, []string{"--profile", "--region"}, ctx.ParsedFlagNames(5))
}
