package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandPathContainsOnlyRegisteredLevels(t *testing.T) {
	previousExitCode := withExitCode
	t.Cleanup(func() { withExitCode = previousExitCode })
	DisableExitCode()
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{"root", nil, nil},
		{"root positional input", []string{"private-input"}, nil},
		{"ignored positional input", []string{"configure", "private-input"}, []string{"configure"}},
		{"surplus flag value", []string{"configure", "set", "--secret", "private-input", "private-surplus"}, []string{"configure", "set"}},
		{"nested command and interspersed flag", []string{"configure", "--profile", "private-profile", "Group", "Leaf", "--secret", "private-input", "private-surplus"}, []string{"configure", "Group", "Leaf"}},
		{"invalid nested command", []string{"configure", "Group", "private-input"}, []string{"configure", "Group"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ignoreArgs := func(*Context, []string) error { return nil }
			root := &Command{Name: "aliyun", Run: ignoreArgs}
			configure := &Command{Name: "configure", Run: ignoreArgs}
			configure.Flags().Add(&Flag{Name: "profile", AssignedMode: AssignedOnce, Persistent: true})
			set := &Command{Name: "set", Run: ignoreArgs}
			set.Flags().Add(&Flag{Name: "secret", AssignedMode: AssignedOnce})
			group := &Command{Name: "Group"}
			leaf := &Command{Name: "Leaf", Run: ignoreArgs}
			leaf.Flags().Add(&Flag{Name: "secret", AssignedMode: AssignedOnce})
			group.AddSubCommand(leaf)
			configure.AddSubCommand(set)
			configure.AddSubCommand(group)
			root.AddSubCommand(configure)
			var observed []string
			root.AfterExecute = func(ctx *Context, _ []string, _ error) {
				observed = ctx.Command().CommandPath()
			}
			ctx := NewCommandContext(io.Discard, io.Discard)
			ctx.EnterCommand(root)
			root.Execute(ctx, test.args)
			require.Equal(t, test.want, observed)
		})
	}
}
