package esacli

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

type npmReadError struct{}

func (npmReadError) Read([]byte) (int, error) { return 0, errors.New("body read failed") }
func (npmReadError) Close() error             { return nil }

func TestNpmInstallationAndUpdatePaths(t *testing.T) {
	oldExec, oldHTTP := execCommandFunc, httpGetFunc
	t.Cleanup(func() { execCommandFunc = oldExec; httpGetFunc = oldHTTP })
	t.Setenv("ALIBABA_CLOUD_ESA_CLI_EXEC_PATH", "")
	t.Setenv("ALIBABA_CLOUD_ESA_CLI_NO_UPDATE_CHECK", "")
	t.Setenv("ALIBABA_CLOUD_ESA_CLI_VERSION", "")
	t.Setenv("ALIBABA_CLOUD_ESA_CLI_NPM_REGISTRY", "https://registry.example.invalid")
	for _, tc := range []struct {
		name                               string
		installed, failCommand, omitBinary bool
		wantErr                            bool
	}{
		{name: "fresh install"}, {name: "fresh command failure", failCommand: true, wantErr: true}, {name: "binary missing after install", omitBinary: true, wantErr: true}, {name: "upgrade", installed: true}, {name: "upgrade failure remains usable", installed: true, failCommand: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ctx, _, _ := newOriginCtx()
			c := NewContext(ctx)
			c.prefixPath = filepath.Join(dir, "prefix")
			c.execFilePath = filepath.Join(c.prefixPath, "bin", "tool")
			c.checkVersionCacheFilePath = filepath.Join(dir, "version")
			c.installed = tc.installed
			c.npmPath = "npm"
			var gotArgs []string
			httpGetFunc = func(url string) (*http.Response, error) {
				if !strings.HasSuffix(url, "/version.txt") {
					t.Errorf("version URL=%s", url)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(" 2.3.4 \n"))}, nil
			}
			execCommandFunc = func(name string, args ...string) *exec.Cmd {
				gotArgs = append([]string(nil), args...)
				if tc.failCommand {
					return exec.Command(filepath.Join(dir, "missing-command"))
				}
				if !tc.omitBinary {
					if err := os.MkdirAll(filepath.Dir(c.execFilePath), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(c.execFilePath, []byte("binary"), 0700); err != nil {
						t.Fatal(err)
					}
				}
				return exec.Command(os.Args[0], "-test.run=^TestNpmChildSuccess$")
			}
			err := c.EnsurePrefixAndPackage()
			if (err != nil) != tc.wantErr {
				t.Fatalf("installation error=%v", err)
			}
			joined := strings.Join(gotArgs, " ")
			if !strings.Contains(joined, npmPackageName+"@2.3.4") || !strings.Contains(joined, "--prefix "+c.prefixPath) || !strings.Contains(joined, "--registry https://registry.example.invalid") {
				t.Fatalf("npm arguments=%v", gotArgs)
			}
			if err == nil {
				if !c.installed {
					t.Fatal("successful installation not marked installed")
				}
				if !fileExists(c.checkVersionCacheFilePath) {
					t.Fatal("version check cache missing")
				}
				if c.NeedCheckVersion() {
					t.Fatal("new cache stale")
				}
			}
		})
	}
}
func TestNpmChildSuccess(t *testing.T) {}

func TestNpmVersionLookupAndPinning(t *testing.T) {
	oldHTTP, oldExec := httpGetFunc, execCommandFunc
	t.Cleanup(func() { httpGetFunc = oldHTTP; execCommandFunc = oldExec })
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	for _, tc := range []struct {
		name   string
		status int
		body   io.ReadCloser
		reqErr error
	}{
		{name: "status", status: 404, body: io.NopCloser(strings.NewReader("missing"))}, {name: "read error", status: 200, body: npmReadError{}}, {name: "network", reqErr: errors.New("offline")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			httpGetFunc = func(string) (*http.Response, error) {
				if tc.reqErr != nil {
					return nil, tc.reqErr
				}
				return &http.Response{StatusCode: tc.status, Body: tc.body}, nil
			}
			if _, err := c.fetchOSSVersion(); err == nil {
				t.Fatal("failed lookup accepted")
			}
		})
	}
	t.Setenv("ALIBABA_CLOUD_ESA_CLI_VERSION", "1.2.3")
	var gotArgs []string
	execCommandFunc = func(_ string, args ...string) *exec.Cmd {
		gotArgs = append([]string(nil), args...)
		return exec.Command(os.Args[0], "-test.run=^TestNpmChildSuccess$")
	}
	if err := c.npmInstall(false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(gotArgs, " "), npmPackageName+"@1.2.3") {
		t.Fatalf("pinned package=%v", gotArgs)
	}
	t.Setenv("ALIBABA_CLOUD_ESA_CLI_VERSION", "")
	httpGetFunc = func(string) (*http.Response, error) { return nil, errors.New("offline") }
	if err := c.npmInstall(false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(gotArgs, " "), npmPackageName+"@") {
		t.Fatalf("fallback package pinned=%v", gotArgs)
	}
}

func TestNodeVersionParsing(t *testing.T) {
	old := execCommandFunc
	t.Cleanup(func() { execCommandFunc = old })
	for _, tc := range []struct {
		output  string
		want    int
		wantErr bool
	}{{"v22.12.0", 22, false}, {"20.1.0", 20, false}, {"invalid", 0, true}, {"vx.1.0", 0, true}} {
		execCommandFunc = func(string, ...string) *exec.Cmd {
			cmd := exec.Command(os.Args[0], "-test.run=^TestNodeVersionChild$")
			cmd.Env = append(os.Environ(), "CLI_TEST_NODE_VERSION="+tc.output)
			return cmd
		}
		got, err := getNodeMajor("node")
		if got != tc.want || (err != nil) != tc.wantErr {
			t.Errorf("version %q=%d,%v", tc.output, got, err)
		}
	}
	execCommandFunc = func(string, ...string) *exec.Cmd { return exec.Command(filepath.Join(t.TempDir(), "missing")) }
	if _, err := getNodeMajor("node"); err == nil {
		t.Fatal("node execution error lost")
	}
}

func TestNpmFlagApplicationAndPrefixFailure(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	ctx.Flags().Add(&cli.Flag{Name: "profile", Shorthand: 'p', AssignedMode: cli.AssignedOnce, Category: "config"})
	ctx.Flags().Add(&cli.Flag{Name: "feature", AssignedMode: cli.AssignedNone, Category: "config"})
	c := NewContext(ctx)
	c.applyMainCliFlagsFromArgs([]string{"run", "--unknown", "--profile=work", "--feature"})
	value, ok := ctx.Flags().GetValue("profile")
	if !ok || value != "work" || !ctx.Flags().Get("feature").IsAssigned() {
		t.Fatal("host flags were not applied")
	}
	got, err := c.RemoveFlagsForMainCli([]string{"run", "-p", "work", "--profile=work", "--feature", "--provider", "value"})
	if err != nil || !reflect.DeepEqual(got, []string{"run", "--provider", "value"}) {
		t.Fatalf("child arguments=%v,%v", got, err)
	}
	dir := t.TempDir()
	blocking := filepath.Join(dir, "file")
	if err := os.WriteFile(blocking, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	c.prefixPath = filepath.Join(blocking, "prefix")
	t.Setenv("ALIBABA_CLOUD_ESA_CLI_EXEC_PATH", "")
	if err := c.EnsurePrefixAndPackage(); err == nil {
		t.Fatal("unwritable prefix accepted")
	}
}

func TestNodeVersionChild(t *testing.T) {
	if value := os.Getenv("CLI_TEST_NODE_VERSION"); value != "" {
		fmt.Print(value)
		os.Exit(0)
	}
}
