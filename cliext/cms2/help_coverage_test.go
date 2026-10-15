package cms2

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func TestCommandHelpTranslation(t *testing.T) {
	oldPath, oldExec, oldOS, oldArch := getConfigurePathFunc, execCommandFunc, runtimeGOOSFunc, runtimeGOARCHFunc
	t.Cleanup(func() {
		getConfigurePathFunc = oldPath
		execCommandFunc = oldExec
		runtimeGOOSFunc = oldOS
		runtimeGOARCHFunc = oldArch
	})
	t.Setenv("ALIBABA_CLOUD_CMS2_EXEC_PATH", "")
	t.Setenv("ALIBABA_CLOUD_MSEUTIL_EXEC_PATH", "")
	dir := t.TempDir()
	getConfigurePathFunc = func() string { return dir }
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	if err := os.WriteFile(filepath.Join(dir, "aliyuncms2"), []byte("binary"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".cms2_version_check"), []byte(fmt.Sprint(time.Now().Unix())), 0600); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"id","access_key_secret":"secret","region_id":"cn-hangzhou"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"--help"}, {"help"}} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			probe := &cli.Command{SkipDefaultHelp: true, Run: func(*cli.Context, []string) error { return nil }}
			ctx.EnterCommand(probe)
			probe.Execute(ctx, []string{"--help"})
			config.AddFlags(ctx.Flags())
			f := ctx.Flags().Get(config.ConfigurePathFlagName)
			f.SetAssigned(true)
			f.SetValue(cfgPath)
			var got []string
			execCommandFunc = func(_ string, args ...string) *exec.Cmd {
				got = append([]string(nil), args...)
				return exec.Command(os.Args[0], "-test.run=^TestCommandHelpChild$")
			}
			if err := NewCms2Command().Run(ctx, append([]string(nil), args...)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, []string{"help"}) {
				t.Fatalf("help arguments=%v,want %v", got, []string{"help"})
			}
		})
	}
}
func TestCommandHelpChild(t *testing.T) {}
