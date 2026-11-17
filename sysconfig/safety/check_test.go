package safety

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

type trackingReader struct {
	read bool
}

func (r *trackingReader) Read([]byte) (int, error) {
	r.read = true
	return 0, errors.New("unexpected read")
}

func useInteractive(t *testing.T, interactive bool) {
	t.Helper()
	original := isInteractive
	isInteractive = func() bool { return interactive }
	t.Cleanup(func() { isInteractive = original })
}

func useLanguage(t *testing.T, language string) {
	t.Helper()
	original := i18n.GetLanguage()
	i18n.SetLanguage(language)
	t.Cleanup(func() { i18n.SetLanguage(original) })
}

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

func TestCheckAndConfirmExplainsLocalConfirmationBoundary(t *testing.T) {
	originalReader := stdinReader
	t.Cleanup(func() { stdinReader = originalReader })

	cmd := CommandInfo{Product: "ecs", ApiOrMethod: "DeleteInstance"}
	confirm := &Policy{Enabled: true, Rules: []Rule{{Pattern: "ecs:Delete*", Action: ActionConfirm}}}

	for _, test := range []struct {
		name     string
		language string
		boundary string
	}{
		{
			name:     "English",
			language: "en",
			boundary: "This confirmation applies only to local CLI operations. It does not grant cloud permissions or verify external approval. Cloud access permissions are controlled by RAM.",
		},
		{
			name:     "Chinese",
			language: "zh",
			boundary: "此确认仅用于 CLI 本地操作确认，不授予云端权限，也不验证外部审批。云端访问权限由 RAM 控制。",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			useLanguage(t, test.language)

			t.Run("interactive yes proceeds and explains boundary", func(t *testing.T) {
				useInteractive(t, true)
				stdinReader = strings.NewReader("yes\n")

				var stderr bytes.Buffer
				ctx := cli.NewCommandContext(&bytes.Buffer{}, &stderr)
				if err := CheckAndConfirm(ctx, confirm, cmd, false); err != nil {
					t.Fatalf("confirmed operation returned error: %v", err)
				}
				if !strings.Contains(stderr.String(), test.boundary) {
					t.Fatalf("interactive prompt missing boundary explanation: %q", stderr.String())
				}
			})

			t.Run("interactive no cancels after explaining boundary", func(t *testing.T) {
				useInteractive(t, true)
				stdinReader = strings.NewReader("no\n")

				var stderr bytes.Buffer
				ctx := cli.NewCommandContext(&bytes.Buffer{}, &stderr)
				err := CheckAndConfirm(ctx, confirm, cmd, false)
				if err == nil {
					t.Fatal("unapproved confirmation unexpectedly succeeded")
				}
				if !strings.Contains(stderr.String(), test.boundary) {
					t.Fatalf("interactive prompt missing boundary explanation: %q", stderr.String())
				}
			})

			t.Run("non-interactive rejection explains boundary", func(t *testing.T) {
				useInteractive(t, false)

				ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
				err := CheckAndConfirm(ctx, confirm, cmd, false)
				if err == nil || !strings.Contains(err.Error(), test.boundary) {
					t.Fatalf("non-interactive error missing boundary explanation: %v", err)
				}
			})
		})
	}
}

func TestCheckAndConfirmSkipConfirmDoesNotBypassDenyOrReadStdin(t *testing.T) {
	originalReader := stdinReader
	t.Cleanup(func() { stdinReader = originalReader })
	reader := &trackingReader{}
	stdinReader = reader

	ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	cmd := CommandInfo{Product: "ecs", ApiOrMethod: "DeleteInstance"}
	deny := &Policy{Enabled: true, Rules: []Rule{{Pattern: "ecs:Delete*", Action: ActionDeny}}}
	if err := CheckAndConfirm(ctx, deny, cmd, true); err == nil {
		t.Fatal("skip-confirm bypassed deny rule")
	}
	if reader.read {
		t.Fatal("deny with skip-confirm read stdin")
	}

	confirm := &Policy{Enabled: true, Rules: []Rule{{Pattern: "ecs:Delete*", Action: ActionConfirm}}}
	if err := CheckAndConfirm(ctx, confirm, cmd, true); err != nil {
		t.Fatalf("skip-confirm returned error: %v", err)
	}
	if reader.read {
		t.Fatal("skip-confirm read stdin")
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
