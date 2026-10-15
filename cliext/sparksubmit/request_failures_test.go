package sparksubmit

import (
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

type sparkTransport func(*http.Request) (*http.Response, error)

func (f sparkTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type sparkBodyError struct{}

func (sparkBodyError) Read([]byte) (int, error) { return 0, errors.New("body failed") }
func (sparkBodyError) Close() error             { return nil }

func TestSparkDownloadRequestFailures(t *testing.T) {
	old := httpGetFunc
	t.Cleanup(func() { httpGetFunc = old })
	for _, tc := range []struct {
		name          string
		status        int
		requestErr    error
		body          io.ReadCloser
		missingParent bool
	}{
		{name: "network", requestErr: errors.New("offline")}, {name: "status", status: 404, body: io.NopCloser(strings.NewReader("missing"))}, {name: "body", status: 200, body: sparkBodyError{}}, {name: "destination", status: 200, body: io.NopCloser(strings.NewReader("data")), missingParent: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			httpGetFunc = func(string) (*http.Response, error) {
				if tc.requestErr != nil {
					return nil, tc.requestErr
				}
				return &http.Response{StatusCode: tc.status, Body: tc.body}, nil
			}
			dest := filepath.Join(t.TempDir(), "file")
			if tc.missingParent {
				dest = filepath.Join(dest, "missing")
			}
			if err := httpGetToFile("https://example.invalid/tool.zip", dest); err == nil {
				t.Fatal("download failure lost")
			}
		})
	}
}

func TestSparkTemporaryCredentialResolution(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	t.Setenv(config.EnvDisableExternalProcess, "")
	http.DefaultTransport = sparkTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"Code":"Success","AccessKeyId":"id","AccessKeySecret":"secret","SecurityToken":"token"}`))}, nil
	})
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	id, secret, token, err := extractCredentials(ctx, config.Profile{Mode: config.CredentialsURI, CredentialsURI: "https://credentials.example.invalid/temporary"})
	if err != nil || id != "id" || secret != "secret" || token != "token" {
		t.Fatalf("temporary credentials=(%q,%q,%q),%v", id, secret, token, err)
	}
	if _, _, _, err := extractCredentials(ctx, config.Profile{Mode: config.External}); err == nil {
		t.Fatal("invalid external credentials accepted")
	}
}

func TestSparkLocalConfigurationAndRunFailures(t *testing.T) {
	oldPath, oldLoad, oldLook, oldLatest := getConfigurePathFunc, loadProfileFunc, lookPathFunc, getLatestVersionFunc
	t.Cleanup(func() {
		getConfigurePathFunc = oldPath
		loadProfileFunc = oldLoad
		lookPathFunc = oldLook
		getLatestVersionFunc = oldLatest
	})
	dir := t.TempDir()
	getConfigurePathFunc = func() string { return dir }
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	c := NewContext(ctx)
	lookPathFunc = func(string) (string, error) { return "", errors.New("missing java") }
	err := c.Run(nil)
	if runtime.GOOS == "windows" {
		if err == nil || !strings.Contains(err.Error(), "Windows") {
			t.Fatalf("unsupported platform=%v", err)
		}
	} else {
		if err == nil || !strings.Contains(err.Error(), "java not found") {
			t.Fatalf("missing Java=%v", err)
		}
		lookPathFunc = func(string) (string, error) { return "java", nil }
		t.Setenv(EnvExecPath, "")
		getLatestVersionFunc = func(string) (string, error) { return "", errors.New("offline") }
		if err := c.Run(nil); err == nil || !strings.Contains(err.Error(), "offline") {
			t.Fatalf("installation failure=%v", err)
		}
	}
	install := filepath.Join(dir, "tool")
	binary := filepath.Join(install, "bin", "spark-submit")
	if err := os.MkdirAll(filepath.Dir(binary), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte("binary"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvExecPath, binary)
	c.initBasicInfo()
	if c.installDir != install || c.confDir != filepath.Join(install, "conf") || !c.installed {
		t.Fatalf("custom installation paths=%+v", c)
	}
	if err := os.MkdirAll(c.confDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(c.confDir, "connection.properties"), 0700); err != nil {
		t.Fatal(err)
	}
	loadProfileFunc = func(*cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou"}, nil
	}
	if err := c.prepareConnectionProperties(); err == nil {
		t.Fatal("properties directory accepted as file")
	}
	block := filepath.Join(dir, "block")
	if err := os.WriteFile(block, []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}
	c.confDir = filepath.Join(block, "conf")
	if err := c.prepareConnectionProperties(); err == nil {
		t.Fatal("blocked configuration directory accepted")
	}
	c.versionCachePath = dir
	c.installed = true
	if !c.cacheStale() {
		t.Fatal("unreadable version cache treated fresh")
	}
}

func TestSparkExecutePreservesExitCode(t *testing.T) {
	old := execCommandFunc
	t.Cleanup(func() { execCommandFunc = old })
	t.Setenv("CLI_TEST_SPARK_EXIT", "1")
	t.Setenv("GOCOVERDIR", t.TempDir())
	execCommandFunc = func(string, ...string) *exec.Cmd { return exec.Command(os.Args[0], "-test.run=^TestSparkExitChild$") }
	c := NewContext(cli.NewCommandContext(io.Discard, io.Discard))
	err := c.execute(nil)
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
		t.Fatalf("child exit error=%v", err)
	}
}
func TestSparkExitChild(t *testing.T) {
	if os.Getenv("CLI_TEST_SPARK_EXIT") == "1" {
		os.Exit(7)
	}
}
