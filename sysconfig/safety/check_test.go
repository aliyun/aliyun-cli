package safety

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestPromptConfirmAnswers(t *testing.T) {
	original := stdinReader
	t.Cleanup(func() { stdinReader = original })

	for _, test := range []struct {
		name   string
		input  string
		answer bool
	}{
		{"short yes", "y\n", true},
		{"mixed case yes", "  YeS  \n", true},
		{"no", "no\n", false},
		{"empty", "\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdinReader = strings.NewReader(test.input)
			var output bytes.Buffer
			if got := PromptConfirm(&output, "continue? "); got != test.answer {
				t.Fatalf("PromptConfirm() = %v, want %v", got, test.answer)
			}
			if output.String() != "continue? " {
				t.Fatalf("prompt output = %q", output.String())
			}
		})
	}

	stdinReader = failingReader{}
	if PromptConfirm(&bytes.Buffer{}, "prompt") {
		t.Fatal("read errors must reject confirmation")
	}
}

func TestCheckAndConfirmActions(t *testing.T) {
	original := stdinReader
	t.Cleanup(func() { stdinReader = original })
	ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	cmd := CommandInfo{Product: "ecs", ApiOrMethod: "DeleteInstance"}

	if err := CheckAndConfirm(ctx, nil, cmd, false); err != nil {
		t.Fatalf("default policy returned error: %v", err)
	}
	if err := CheckAndConfirm(ctx, &Policy{Enabled: true}, cmd, false); err != nil {
		t.Fatalf("allow policy returned error: %v", err)
	}

	deny := &Policy{Enabled: true, Rules: []Rule{{Pattern: "ecs:Delete*", Action: ActionDeny}}}
	if err := CheckAndConfirm(ctx, deny, cmd, false); err == nil || !strings.Contains(err.Error(), "DeleteInstance") || !strings.Contains(err.Error(), "ecs:Delete*") {
		t.Fatalf("deny error = %v", err)
	}

	confirm := &Policy{Enabled: true, Rules: []Rule{{Pattern: "ecs:Delete*", Action: ActionConfirm}}}
	if err := CheckAndConfirm(ctx, confirm, cmd, true); err != nil {
		t.Fatalf("skip-confirm should approve operation: %v", err)
	}
	stdinReader = strings.NewReader("no\n")
	err := CheckAndConfirm(ctx, confirm, cmd, false)
	if err == nil {
		t.Fatal("unapproved confirmation unexpectedly succeeded")
	}
	if IsInteractive() {
		if !strings.Contains(err.Error(), "cancel") && !strings.Contains(err.Error(), "取消") {
			t.Fatalf("interactive confirmation error = %v", err)
		}
	} else if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("non-interactive confirmation error = %v", err)
	}
}

func TestDisplayCommand(t *testing.T) {
	if got := displayCommand(CommandInfo{ApiOrMethod: "DescribeInstances"}); got != "DescribeInstances" {
		t.Fatalf("RPC display command = %q", got)
	}
	if got := displayCommand(CommandInfo{ApiOrMethod: "delete", Path: "/instances/i-1"}); got != "DELETE /instances/i-1" {
		t.Fatalf("REST display command = %q", got)
	}
	_ = IsInteractive()
}

func TestInferOperationFromAPIName(t *testing.T) {
	tests := map[string]string{
		"DeleteInstance": "delete",
		"UpdateInstance": "update",
		"ModifyInstance": "update",
		"CreateInstance": "create",
		"AddTags":        "create",
		"DescribeThings": "",
	}
	for apiName, expected := range tests {
		if actual := InferOperationFromApiName(apiName); actual != expected {
			t.Fatalf("InferOperationFromApiName(%q) = %q, want %q", apiName, actual, expected)
		}
	}
}
