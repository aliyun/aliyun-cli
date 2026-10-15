package cms2

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func TestRunFailureBoundaries(t *testing.T) {
	for _, scenario := range []string{"unsupported platform", "version lookup", "installation", "profile", "execution"} {
		t.Run(scenario, func(t *testing.T) {
			oldPath, oldOS, oldArch, oldLatest, oldDownload, oldExec := getConfigurePathFunc, runtimeGOOSFunc, runtimeGOARCHFunc, getLatestCms2VersionFunc, downloadFileFunc, execCommandFunc
			t.Cleanup(func() {
				getConfigurePathFunc = oldPath
				runtimeGOOSFunc = oldOS
				runtimeGOARCHFunc = oldArch
				getLatestCms2VersionFunc = oldLatest
				downloadFileFunc = oldDownload
				execCommandFunc = oldExec
			})
			dir := t.TempDir()
			getConfigurePathFunc = func() string { return dir }
			runtimeGOOSFunc = func() string { return "linux" }
			runtimeGOARCHFunc = func() string { return "amd64" }
			downloadTemp := t.TempDir()
			t.Setenv("TMPDIR", downloadTemp)
			t.Setenv("TMP", downloadTemp)
			t.Setenv("TEMP", downloadTemp)
			t.Setenv("ALIBABA_CLOUD_CMS2_EXEC_PATH", "")
			t.Setenv("ALIBABA_CLOUD_MSEUTIL_EXEC_PATH", "")
			getLatestCms2VersionFunc = func() (string, error) {
				if scenario == "version lookup" {
					return "", errors.New("version lookup failed")
				}
				return "2.0", nil
			}
			downloadFileFunc = func(string, string) error { return errors.New("install failed") }
			execCommandFunc = func(string, ...string) *exec.Cmd { return exec.Command(filepath.Join(dir, "missing-command")) }
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			config.AddFlags(ctx.Flags())
			configFile := filepath.Join(dir, "config.json")
			f := ctx.Flags().Get(config.ConfigurePathFlagName)
			f.SetAssigned(true)
			f.SetValue(configFile)
			if scenario != "profile" {
				if err := os.WriteFile(configFile, []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"id","access_key_secret":"secret","region_id":"cn-hangzhou"}]}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			c := NewContext(ctx)
			c.InitBasicInfo()
			want := ""
			switch scenario {
			case "unsupported platform":
				runtimeGOOSFunc = func() string { return "unsupported" }
				want = "not supported"
			case "version lookup":
				want = "version lookup failed"
			case "installation":
				want = "install failed"
			case "profile", "execution":
				if err := os.MkdirAll(filepath.Dir(c.execFilePath), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(c.execFilePath, []byte("installed"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(c.checkVersionCacheFilePath, []byte(fmt.Sprint(time.Now().Unix())), 0600); err != nil {
					t.Fatal(err)
				}
				if scenario == "profile" {
					want = "config"
				} else {
					want = "execute"
				}
			}
			err := c.Run([]string{"list"})
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), want) {
				t.Fatalf("%s error=%v,want=%q", scenario, err, want)
			}
		})
	}
}
