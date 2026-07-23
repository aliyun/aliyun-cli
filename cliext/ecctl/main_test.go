package ecctl

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestNewEcctlCommand(t *testing.T) {
	cmd := NewEcctlCommand()
	if cmd == nil {
		t.Fatal("nil cmd")
	}
	if cmd.Name != "ecctl" {
		t.Fatalf("name %s", cmd.Name)
	}
	if cmd.Short == nil || cmd.Short.Get("en") == "" {
		t.Fatal("short missing")
	}
	if cmd.Usage != "aliyun ecctl <command> [args...]" {
		t.Fatalf("usage %q", cmd.Usage)
	}
	if !cmd.EnableUnknownFlag || !cmd.KeepArgs || !cmd.SkipDefaultHelp {
		t.Fatal("flag metadata")
	}
	if cmd.Run == nil {
		t.Fatal("run nil")
	}
}

func TestNewEcctlCommandMetadata(t *testing.T) {
	cmd := NewEcctlCommand()
	meta := map[string]*cli.Metadata{}
	cmd.GetMetadata(meta)
	m, ok := meta["ecctl"]
	if !ok || m.Name != "ecctl" {
		t.Fatalf("metadata missing")
	}
}

func TestNewEcctlCommand_RunUnsupportedPlatform(t *testing.T) {
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "s390x" }
	t.Cleanup(func() { runtimeGOOSFunc = origGOOS; runtimeGOARCHFunc = origGOARCH })

	cmd := NewEcctlCommand()
	ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	err := cmd.Run(ctx, []string{"version"})
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected unsupported platform error, got %v", err)
	}
}

func TestNewEcctlCommand_ExitErrorSubprocess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash exit")
	}
	if os.Getenv("ECCTL_EXIT_SUBTEST") == "1" {
		tmp := t.TempDir()
		cfgDir := tmp
		if err := os.MkdirAll(cfgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		configJSON := `{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou","language":"en"}]}`
		if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(configJSON), 0o644); err != nil {
			t.Fatal(err)
		}
		oldGet := getConfigurePathFunc
		getConfigurePathFunc = func() string { return tmp }
		defer func() { getConfigurePathFunc = oldGet }()
		fake := filepath.Join(tmp, "ecctl")
		if err := os.WriteFile(fake, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("ALIBABA_CLOUD_ECCTL_EXEC_PATH", fake)
		origExec := execCommandFunc
		execCommandFunc = func(name string, args ...string) *exec.Cmd {
			return exec.Command("bash", "-c", "exit 3")
		}
		defer func() { execCommandFunc = origExec }()
		cmd := NewEcctlCommand()
		ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
		_ = cmd.Run(ctx, []string{"version"})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestNewEcctlCommand_ExitErrorSubprocess", "-test.count=1")
	cmd.Env = append(os.Environ(), "ECCTL_EXIT_SUBTEST=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit error")
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 3 {
		t.Fatalf("expected exit 3, got %v", err)
	}
}
