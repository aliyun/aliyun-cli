package maxc

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewMaxcCommand_BasicShape(t *testing.T) {
	cmd := NewMaxcCommand()
	if cmd == nil {
		t.Fatal("NewMaxcCommand returned nil")
	}
	if cmd.Name != "maxc" {
		t.Errorf("expected Name=maxc, got %q", cmd.Name)
	}
	if cmd.Short == nil {
		t.Fatal("Short i18n text nil")
	}
	if cmd.Short.Get("en") == "" {
		t.Error("Short en is empty")
	}
	if cmd.Short.Get("zh") == "" {
		t.Error("Short zh is empty")
	}
	if cmd.Usage != "maxc <command> [args...] [options...]" {
		t.Errorf("Usage mismatch: %s", cmd.Usage)
	}
	if cmd.Hidden {
		t.Error("Hidden expected false")
	}
	if !cmd.EnableUnknownFlag {
		t.Error("EnableUnknownFlag must be true so aliyun root does not parse maxc subcommand flags")
	}
	if !cmd.KeepArgs {
		t.Error("KeepArgs must be true")
	}
	if cmd.SkipDefaultHelp {
		t.Error("SkipDefaultHelp must be false so framework auto-renders aliyun-style help on --help")
	}
	if cmd.Run == nil {
		t.Fatal("Run function should not be nil")
	}
	if cmd.Help == nil {
		t.Fatal("Help function should not be nil — we render a custom command-group + env-var section")
	}
}

func TestNewMaxcCommand_HelpOutput(t *testing.T) {
	// Attach to a parent so GetUsageWithParent emits "aliyun maxc ...".
	root := &cli.Command{Name: "aliyun"}
	cmd := NewMaxcCommand()
	root.AddSubCommand(cmd)

	stdout := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, &bytes.Buffer{})
	ctx.EnterCommand(cmd)

	if err := cmd.Help(ctx, nil); err != nil {
		t.Fatalf("Help returned error: %v", err)
	}
	out := stdout.String()

	for _, want := range []string{
		"MaxCompute CLI",                                   // PrintHead
		"Usage:",                                           // PrintUsage
		"aliyun maxc <command>",                            // usage body (parent path prepended by framework)
		"Sample:",                                          // PrintSample
		"Commands:",                                        // our custom groups
		"query",                                            // one group from the table
		"auth",                                             // another group
		"ALIBABA_CLOUD_MAXC_NO_UPDATE_CHECK",               // env var section
		"Use `maxc --help` for more information.",          // PrintTail
	} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q\nfull output:\n%s", want, out)
		}
	}
}
