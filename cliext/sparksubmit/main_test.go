package sparksubmit

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewSparkSubmitCommand_HelpOutput(t *testing.T) {
	stdout := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, &bytes.Buffer{})
	cmd := NewSparkSubmitCommand()
	ctx.EnterCommand(cmd)

	if err := cmd.Help(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	for _, want := range []string{"spark-submit", EnvExecPath, EnvWorkspaceID, "Java"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q:\n%s", want, out)
		}
	}
}
