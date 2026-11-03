package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelpCompletionObservesOriginalResultBeforeRendering(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	previousExitCode, previousObserver := withExitCode, beforeExit
	t.Cleanup(func() {
		withExitCode = previousExitCode
		SetBeforeExit(previousObserver)
	})
	DisableExitCode()
	SetBeforeExit(func(int) { t.Fatal("disabled process exits must not notify the exit observer") })

	for _, test := range []struct {
		name                  string
		args                  []string
		nested, raw, fallback bool
		presetHelp            bool
	}{
		{name: "root_flag", args: []string{"InvalidReviewAPI", "--help"}},
		{name: "root_help_prefix", args: []string{"help", "InvalidReviewAPI"}},
		{name: "nested_flag", args: []string{"child", "InvalidReviewAPI", "--help"}, nested: true},
		{name: "nested_help_prefix", args: []string{"help", "child", "InvalidReviewAPI"}, nested: true},
		{name: "raw_flag", args: []string{"InvalidReviewAPI", "--help"}, raw: true},
		{name: "raw_inherited_help", args: []string{"InvalidReviewAPI"}, raw: true, presetHelp: true},
		{name: "root_fallback", fallback: true},
		{name: "nested_fallback", args: []string{"child"}, nested: true, fallback: true},
	} {
		for _, fails := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/fails=%v", test.name, fails), func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				ctx := NewCommandContext(&stdout, &stderr)
				root := &Command{Name: "aliyun"}
				target := root
				if test.nested {
					target = &Command{Name: "child"}
					root.AddSubCommand(target)
				}
				target.RawArgs = test.raw
				if !test.fallback {
					target.Run = func(*Context, []string) error {
						t.Fatal("a Help invocation must not execute Run")
						return nil
					}
				}
				ctx.EnterCommand(root)
				ctx.help = test.presetHelp
				original := NewInvalidCommandError("InvalidReviewAPI", ctx)
				var order []string
				helpCalls, completionCalls, normalizationCalls, nestedCompletions := 0, 0, 0, 0
				target.Help = func(*Context, []string) error {
					helpCalls++
					order = append(order, "help")
					fmt.Fprintln(&stdout, "help output")
					if fails {
						return original
					}
					return nil
				}
				root.AfterExecute = func(got *Context, args []string, err error) {
					completionCalls++
					order = append(order, "after")
					require.Same(t, target, got.Command())
					require.Equal(t, test.args, args, "completion retains the original invocation")
					require.Empty(t, stderr.String(), "completion must run before error rendering")
					require.Zero(t, normalizationCalls)
					if fails {
						require.Same(t, original, err, "Help errors must reach completion without replacement")
						var invalid *InvalidCommandError
						require.ErrorAs(t, err, &invalid)
						require.Equal(t, "INVALID_COMMAND", invalid.LocalErrorCode())
					} else {
						require.NoError(t, err)
					}
				}
				root.NormalizeError = func(_ *Context, _ []string, err error) error {
					normalizationCalls++
					order = append(order, "normalize")
					require.Same(t, original, err)
					return err
				}
				if target != root {
					target.AfterExecute = func(*Context, []string, error) { nestedCompletions++ }
				}

				root.Execute(ctx, test.args)
				require.Equal(t, 1, helpCalls)
				require.Equal(t, 1, completionCalls)
				require.Zero(t, nestedCompletions, "nested Help must not finalize the root twice")
				require.Equal(t, "help output\n", stdout.String())
				if fails {
					require.Equal(t, []string{"help", "after", "normalize"}, order)
					require.Equal(t, 1, normalizationCalls)
					require.Equal(t, 1, strings.Count(stderr.String(), "ERROR:"))
					require.Equal(t, 1, strings.Count(stderr.String(), original.Error()))
				} else {
					require.Equal(t, []string{"help", "after"}, order)
					require.Zero(t, normalizationCalls)
					require.Empty(t, stderr.String())
				}
			})
		}
	}
}
