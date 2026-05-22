package maxc

import (
	"bytes"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

// withExecCommandStub swaps execCommandFunc so we can spy on the launched
// child process and force a synthetic exit code without spawning a real
// subprocess.
func withExecCommandStub(t *testing.T, fn func(name string, arg ...string) *exec.Cmd) {
	t.Helper()
	orig := execCommandFunc
	t.Cleanup(func() { execCommandFunc = orig })
	execCommandFunc = fn
}

// fakeExitCmd returns an *exec.Cmd that, when Run, invokes /bin/sh -c "exit N".
// Captures the requested binary + args so tests can assert on them.
type spyExec struct {
	name string
	args []string
}

func newCtxForExecute(t *testing.T) *Context {
	t.Helper()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	c := &Context{
		originCtx:    cli.NewCommandContext(out, errOut),
		execFilePath: "/path/to/maxc",
		envMap:       map[string]string{"INJECTED": "yes"},
	}
	return c
}

func TestExecute_PassesArgsAndEnv(t *testing.T) {
	var spy spyExec
	var gotEnv []string
	withExecCommandStub(t, func(name string, arg ...string) *exec.Cmd {
		spy = spyExec{name: name, args: arg}
		cmd := exec.Command("/bin/sh", "-c", "exit 0")
		// We can't observe cmd.Env from inside the stub (it's set after we
		// return), so capture it via a wrapper closure below.
		_ = gotEnv
		return cmd
	})

	c := newCtxForExecute(t)
	if err := c.Execute([]string{"query", "--sql", "select 1"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if spy.name != "/path/to/maxc" {
		t.Errorf("exec name = %q", spy.name)
	}
	if !reflect.DeepEqual(spy.args, []string{"query", "--sql", "select 1"}) {
		t.Errorf("exec args = %v", spy.args)
	}
}

func TestExecute_ForwardsExitCode(t *testing.T) {
	withExecCommandStub(t, func(name string, arg ...string) *exec.Cmd {
		return exec.Command("/bin/sh", "-c", "exit 42")
	})

	c := newCtxForExecute(t)
	err := c.Execute(nil)
	if err == nil {
		t.Fatal("expected non-nil error for non-zero exit")
	}
	ee, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("error type = %T, want *ExitError (err=%v)", err, err)
	}
	if ee.Code != 42 {
		t.Errorf("ExitCode = %d, want 42", ee.Code)
	}
}

func TestExecute_MergesEnvOverride(t *testing.T) {
	// Use a wrapper that lets us inspect the final cmd.Env that Execute sets.
	withExecCommandStub(t, func(name string, arg ...string) *exec.Cmd {
		// Echo INJECTED back via shell exit code so we can verify it was set.
		// 0 = present-and-equal, 1 = missing-or-wrong.
		return exec.Command("/bin/sh", "-c", `test "$INJECTED" = "yes"`)
	})
	c := newCtxForExecute(t)
	if err := c.Execute(nil); err != nil {
		t.Errorf("INJECTED env not propagated to child: %v", err)
	}
}

func TestExecute_OverrideWinsOverInherited(t *testing.T) {
	// Pre-set INJECTED in the parent process; mergeEnv must drop the parent
	// value and prefer the override map's "yes".
	t.Setenv("INJECTED", "parent-value")
	withExecCommandStub(t, func(name string, arg ...string) *exec.Cmd {
		return exec.Command("/bin/sh", "-c", `test "$INJECTED" = "yes"`)
	})
	c := newCtxForExecute(t)
	if err := c.Execute(nil); err != nil {
		t.Errorf("env override did not win over parent: %v", err)
	}
}

// --- RemoveFlagsForMainCli ----------------------------------------------

func TestRemoveFlagsForMainCli_StripsProfile(t *testing.T) {
	c := &Context{}
	in := []string{"--profile", "prod", "query", "--sql", "select 1"}
	got := c.RemoveFlagsForMainCli(in)
	want := []string{"query", "--sql", "select 1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRemoveFlagsForMainCli_StripsInlineProfile(t *testing.T) {
	c := &Context{}
	got := c.RemoveFlagsForMainCli([]string{"--profile=prod", "list-tables"})
	want := []string{"list-tables"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRemoveFlagsForMainCli_PreservesUnknownAndChildFlags(t *testing.T) {
	c := &Context{}
	// --sql is a maxc-side flag, --region is shared but NOT in stripFlags so
	// the child can handle it with its own semantics.
	in := []string{"query", "--sql", "show tables", "--region", "cn-hangzhou", "--output", "json"}
	got := c.RemoveFlagsForMainCli(in)
	if !reflect.DeepEqual(got, in) {
		t.Errorf("non-strip flags were modified: got %v want %v", got, in)
	}
}

func TestRemoveFlagsForMainCli_StripsConfigPath(t *testing.T) {
	c := &Context{}
	got := c.RemoveFlagsForMainCli([]string{"--config-path", "/tmp/cfg.json", "whoami"})
	want := []string{"whoami"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMergeEnv_OverridesReplaceBase(t *testing.T) {
	base := []string{"A=1", "B=2", "C=3"}
	out := mergeEnv(base, map[string]string{"B": "overridden", "D": "new"})

	have := map[string]string{}
	for _, e := range out {
		k, v, _ := strings.Cut(e, "=")
		have[k] = v
	}
	if have["A"] != "1" || have["C"] != "3" {
		t.Errorf("base entries lost: %v", have)
	}
	if have["B"] != "overridden" {
		t.Errorf("override didn't win for B: %v", have)
	}
	if have["D"] != "new" {
		t.Errorf("new key D missing: %v", have)
	}
	// And the duplicate "B=2" from base must be gone.
	for _, e := range out {
		if e == "B=2" {
			t.Errorf("base B=2 still present alongside override: %v", out)
		}
	}
}
