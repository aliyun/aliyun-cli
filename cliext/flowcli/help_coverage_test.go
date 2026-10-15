package flowcli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func TestCommandHelpTranslation(t *testing.T) {
	t.Setenv("GOCOVERDIR", t.TempDir())
	oldPath, oldExec := getConfigurePathFunc, execCommandFunc
	t.Cleanup(func() { getConfigurePathFunc = oldPath; execCommandFunc = oldExec })
	dir := t.TempDir()
	getConfigurePathFunc = func() string { return dir }
	configFile := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configFile, []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"id","access_key_secret":"secret","region_id":"cn-hangzhou"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"--help"}, {"-h"}} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			probe := &cli.Command{SkipDefaultHelp: true, Run: func(*cli.Context, []string) error { return nil }}
			ctx.EnterCommand(probe)
			probe.Execute(ctx, []string{"--help"})
			config.AddFlags(ctx.Flags())
			flag := ctx.Flags().Get(config.ConfigurePathFlagName)
			flag.SetAssigned(true)
			flag.SetValue(configFile)
			fixture := NewContext(ctx)
			binary := filepath.Join(dir, "tool")
			if err := os.WriteFile(binary, []byte("installed"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("ALIBABA_CLOUD_FLOW_CLI_EXEC_PATH", binary)
			t.Setenv("ALIBABA_CLOUD_FLOW_CLI_NODE_PATH", binary)
			if err := fixture.InitBasicInfo(); err != nil {
				t.Fatal(err)
			}
			var got []string
			execCommandFunc = func(_ string, args ...string) *exec.Cmd {
				got = append([]string(nil), args...)
				child := exec.Command(os.Args[0], "-test.run=^TestCommandHelpChild$")
				child.Env = append(os.Environ(), "GOCOVERDIR="+t.TempDir())
				return child
			}
			if err := NewFlowcliCommand().Run(ctx, append([]string(nil), args...)); err != nil {
				t.Fatal(err)
			}
			want := args
			if len(want) == 0 {
				want = []string{"--help"}
			}

			if !reflect.DeepEqual(got, want) {
				t.Fatalf("help arguments=%v,want=%v", got, want)
			}
		})
	}
}
func TestCommandHelpChild(t *testing.T) {}
