package cli

import (
	"io"
	"reflect"
	"testing"
)

func TestRawArgsPreservesTokens(t *testing.T) {
	args := []string{"--include", "*.txt", "--recursive", "help", "--", "--profile"}
	called := false
	command := &Command{Name: "raw", RawArgs: true, Run: func(_ *Context, got []string) error {
		called = true
		if !reflect.DeepEqual(args, got) {
			t.Fatalf("got %v, want %v", got, args)
		}
		return nil
	}}
	ctx := NewCommandContext(io.Discard, io.Discard)
	ctx.SetCommand(command)
	if err := command.executeInner(ctx, args); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("raw command was not invoked")
	}
}

func TestRawArgsHelpBoundary(t *testing.T) {
	command := &Command{Name: "raw", RawArgs: true}
	command.Flags().Add(&Flag{Name: "value", Shorthand: 'v', AssignedMode: AssignedOnce})
	ctx := NewCommandContext(io.Discard, io.Discard)
	ctx.SetCommand(command)
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"--help"}, true}, {[]string{"-h"}, true},
		{[]string{"--", "--help"}, false}, {[]string{"--value", "--help"}, false},
		{[]string{"-v", "-h"}, false}, {[]string{"--value=--help"}, false},
		{[]string{"help"}, false},
	} {
		if got := rawArgsRequestHelp(ctx, tc.args); got != tc.want {
			t.Fatalf("%v: got %v, want %v", tc.args, got, tc.want)
		}
	}
}
