package saectl

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/config"
)

func TestPlatformSupportMatrix(t *testing.T) {
	oldOS, oldArch := runtimeGOOSFunc, runtimeGOARCHFunc
	t.Cleanup(func() { runtimeGOOSFunc = oldOS; runtimeGOARCHFunc = oldArch })
	for _, system := range []string{"linux", "darwin", "windows", "freebsd"} {
		for _, arch := range []string{"amd64", "arm64", "386"} {
			runtimeGOOSFunc = func() string { return system }
			runtimeGOARCHFunc = func() string { return arch }
			c := &Context{}
			c.CheckOsTypeAndArch()
			want := (system != "freebsd" && arch != "386")
			if c.osSupport != want {
				t.Errorf("%s/%s support = %v", system, arch, c.osSupport)
			}
			if want {
				suffix := ".tar.gz"
				if system == "windows" {
					suffix = ".zip"
				}
				if c.downloadPathSuffix != system+"-"+arch+suffix {
					t.Errorf("suffix=%q", c.downloadPathSuffix)
				}
			}
		}
	}
}

func TestVersionCacheBoundaries(t *testing.T) {
	old := timeNowFunc
	timeNowFunc = func() time.Time { return time.Unix(100000, 0) }
	t.Cleanup(func() { timeNowFunc = old })
	c := &Context{checkVersionCacheFilePath: filepath.Join(t.TempDir(), "cache")}
	if c.NeedCheckVersion() {
		t.Fatal("uninstalled binary checked version")
	}
	c.installed = true
	if !c.NeedCheckVersion() {
		t.Fatal("missing cache not stale")
	}
	for content, want := range map[string]bool{"invalid": true, "100000": false, fmt.Sprint(100000 - VersionCheckTTL): false, fmt.Sprint(99999 - VersionCheckTTL): true} {
		if err := os.WriteFile(c.checkVersionCacheFilePath, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if got := c.NeedCheckVersion(); got != want {
			t.Errorf("cache %q stale=%v", content, got)
		}
	}
	if err := c.UpdateCheckCacheTime(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(c.checkVersionCacheFilePath)
	if err != nil || string(b) != "100000" {
		t.Fatalf("cache=%q,%v", b, err)
	}
	c.checkVersionCacheFilePath = filepath.Join(t.TempDir(), "missing", "cache")
	if err := c.UpdateCheckCacheTime(); err == nil {
		t.Fatal("missing directory accepted")
	}
}

func TestLatestVersionResponses(t *testing.T) {
	old := saectlBaseUrl
	t.Cleanup(func() { saectlBaseUrl = old })
	for _, tc := range []struct {
		body    string
		status  int
		want    string
		wantErr bool
	}{
		{" 1.2.3 \n", 200, "1.2.3", false}, {" ", 200, "", true}, {"not found", 404, "", true},
	} {
		t.Run(fmt.Sprint(tc.status, tc.body), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/version.txt" {
					t.Errorf("path=%s", r.URL)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			saectlBaseUrl = server.URL + "/"
			got, err := GetLatestSaeCtlVersion()
			if got != tc.want || (err != nil) != tc.wantErr {
				t.Fatalf("version=%q,%v", got, err)
			}
		})
	}
	saectlBaseUrl = "://invalid"
	if _, err := GetLatestSaeCtlVersion(); err == nil {
		t.Fatal("invalid URL accepted")
	}
}

func TestRunInstallAndUpdate(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		installed             bool
		local                 string
		remoteErr, installErr bool
		wantInstall           int
		wantErr               string
	}{
		{name: "fresh install", wantInstall: 1}, {name: "latest", installed: true, local: "2.0"},
		{name: "upgrade", installed: true, local: "1.0", wantInstall: 1},
		{name: "version lookup failure", remoteErr: true, wantErr: "offline"},
		{name: "install failure", installErr: true, wantInstall: 1, wantErr: "install failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			prepareConfig(t, home)
			dir := filepath.Join(home, ".aliyun")
			oldPath, oldRemote, oldInstall, oldExec, oldOS, oldArch := getConfigurePathFunc, getLatestSaeCtlVersionFunc, downloadAndExtractFunc, execCommandFunc, runtimeGOOSFunc, runtimeGOARCHFunc
			t.Cleanup(func() {
				getConfigurePathFunc = oldPath
				getLatestSaeCtlVersionFunc = oldRemote
				downloadAndExtractFunc = oldInstall
				execCommandFunc = oldExec
				runtimeGOOSFunc = oldOS
				runtimeGOARCHFunc = oldArch
			})
			getConfigurePathFunc = func() string { return dir }
			runtimeGOOSFunc = func() string { return "linux" }
			runtimeGOARCHFunc = func() string { return "amd64" }
			if tc.installed {
				executable := "saectl"
				if runtime.GOOS == "windows" {
					executable += ".exe"
				}
				if err := os.WriteFile(filepath.Join(dir, executable), []byte("installed"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			getLatestSaeCtlVersionFunc = func() (string, error) {
				if tc.remoteErr {
					return "", errors.New("offline")
				}
				return "2.0", nil
			}
			installs := 0
			t.Setenv("TMPDIR", t.TempDir())
			downloadAndExtractFunc = func(url, dest, exe, center string) error {
				installs++
				if !strings.Contains(url, "2.0/saectl-2.0-linux-amd64.tar.gz") {
					t.Errorf("download URL=%s", url)
				}
				if tc.installErr {
					return errors.New("install failed")
				}
				return os.WriteFile(exe, []byte("installed"), 0700)
			}
			var childArgs []string
			execCommandFunc = func(name string, args ...string) *exec.Cmd {
				if len(args) == 1 && args[0] == "version" {
					cmd := exec.Command(os.Args[0], "-test.run=^TestSaectlVersionChild$")
					cmd.Env = append(os.Environ(), "CLI_TEST_SAECTL_VERSION="+tc.local)
					return cmd
				}
				childArgs = append([]string(nil), args...)
				return exec.Command(os.Args[0], "-test.run=^TestSaectlChildSuccess$")
			}
			ctx, _, _ := newOriginCtx()
			config.AddFlags(ctx.Flags())
			if err := NewSaectlCommand().Run(ctx, []string{"get", "app"}); tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error=%v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			} else {
				if !reflect.DeepEqual(childArgs, []string{"get", "app"}) {
					t.Fatalf("child args=%v", childArgs)
				}
				if !fileExists(filepath.Join(dir, ".saectl_version_check")) {
					t.Fatal("version cache missing")
				}
			}
			if installs != tc.wantInstall {
				t.Fatalf("installs=%d,want=%d", installs, tc.wantInstall)
			}
		})
	}
}

func TestSaectlChildSuccess(t *testing.T) {}

func TestLocalVersionAndFlagEdges(t *testing.T) {
	c := &Context{}
	if err := c.GetLocalVersion(); err == nil {
		t.Fatal("uninstalled version accepted")
	}
	old := execCommandFunc
	t.Cleanup(func() { execCommandFunc = old })
	execCommandFunc = func(string, ...string) *exec.Cmd { return exec.Command(filepath.Join(t.TempDir(), "missing")) }
	c.installed = true
	if err := c.GetLocalVersion(); err == nil {
		t.Fatal("subprocess error lost")
	}
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "profile", "test")
	ctx.Flags().Get("profile").Shorthand = 'p'
	c.originCtx = ctx
	got, err := c.RemoveFlagsForMainCli([]string{"get", "-p", "test", "--cli-ai-mode", "--no-cli-ai-mode"})
	if err != nil || !reflect.DeepEqual(got, []string{"get"}) {
		t.Fatalf("args=%v,%v", got, err)
	}
	if got := stripArgsEqual([]string{"get"}); !reflect.DeepEqual(got, []string{"get"}) {
		t.Fatalf("strip with no drops=%v", got)
	}
	if got := stripArgsEqual(nil, "--cli-ai-mode"); got != nil {
		t.Fatalf("strip nil=%v", got)
	}
}

func TestSaectlVersionChild(t *testing.T) {
	if value := os.Getenv("CLI_TEST_SAECTL_VERSION"); value != "" {
		fmt.Print(value)
		os.Exit(0)
	}
}
