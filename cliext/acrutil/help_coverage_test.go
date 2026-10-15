package acrutil

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestAcrutilHelpAndUnknownCommand(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		help    bool
		wantErr bool
	}{
		{name: "no command"}, {name: "help command", args: []string{"help"}},
		{name: "help flag", args: []string{"--help"}, help: true},
		{name: "implicit help", help: true},
		{name: "existing help", args: []string{"help"}, help: true},
		{name: "unknown command", args: []string{"unknown"}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			ctx := cli.NewCommandContext(&out, &errOut)
			if tc.help {
				probe := &cli.Command{SkipDefaultHelp: true, Run: func(*cli.Context, []string) error { return nil }}
				ctx.EnterCommand(probe)
				probe.Execute(ctx, []string{"--help"})
			}
			err := NewAcrutilCommand().Run(ctx, tc.args)
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "unknown command") {
					t.Fatalf("unknown command error=%v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"Available Commands:", "skill", "diagnosis", "aliyun acrutil"} {
				if !strings.Contains(out.String(), want) {
					t.Errorf("help missing %q: %s", want, out.String())
				}
			}
		})
	}
}
