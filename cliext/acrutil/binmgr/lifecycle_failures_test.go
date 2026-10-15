package binmgr

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestManagerCommandAndPlatformDependencies(t *testing.T) {
	t.Setenv("GOCOVERDIR", t.TempDir())
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	flag := &cli.Flag{Name: "user-agent", AssignedMode: cli.AssignedOnce}
	flag.SetAssigned(true)
	flag.SetValue("custom-client/1")
	ctx.Flags().Add(flag)
	m := New(testConfig(), ctx)
	if m.runtimeGOOSFn() != runtime.GOOS || m.runtimeGOARCHFn() != runtime.GOARCH {
		t.Fatal("manager platform does not match host")
	}
	m.SetExecFilePathForTest("managed-binary")
	m.versionLocal = "2.0"
	var child *exec.Cmd
	var gotName string
	var gotArgs []string
	m.SetExecCommandForTest(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = append([]string(nil), args...)
		child = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestManagerBoundaryChild$")
		return child
	})
	if err := m.Execute(context.Background(), []string{"list"}); err != nil {
		t.Fatal(err)
	}
	if gotName != "managed-binary" || !reflect.DeepEqual(gotArgs, []string{"list"}) {
		t.Fatalf("command=%q,%v", gotName, gotArgs)
	}
	if !strings.Contains(strings.Join(child.Env, "\n"), "test-bin/2.0 custom-client/1") {
		t.Fatalf("custom user agent missing: %v", child.Env)
	}
}
func TestManagerBoundaryChild(t *testing.T) {}

func TestManagerRunRejectsUnsupportedPlatformAndInstallFailure(t *testing.T) {
	for _, scenario := range []string{"platform", "install"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			m := New(testConfig(), ctx)
			m.getConfigurePathFn = func() string { return t.TempDir() }
			m.runtimeGOOSFn = func() string { return "linux" }
			m.runtimeGOARCHFn = func() string { return "amd64" }
			m.getLatestVersionFn = func(context.Context, string) (string, error) { return "2.0", nil }
			m.downloadBinaryFn = func(context.Context, string, string) error { return errors.New("download refused") }
			want := "download refused"
			if scenario == "platform" {
				m.runtimeGOOSFn = func() string { return "unsupported" }
				want = "not supported"
			}
			if err := m.Run(nil); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("lifecycle error=%v,want=%q", err, want)
			}
		})
	}
}

func TestManagerUnreadableLocalVersionDoesNotBlockUpdate(t *testing.T) {
	ctx, _, stderr := newOriginCtx()
	m := New(testConfig(), ctx)
	m.configPath = t.TempDir()
	m.execFilePath = filepath.Join(m.configPath, "binary")
	m.versionFilePath = m.configPath
	m.checkVersionCacheFilePath = filepath.Join(m.configPath, "cache")
	m.installed = true
	m.osSupport = true
	m.downloadPathSuffix = "linux-amd64"
	m.getLatestVersionFn = func(context.Context, string) (string, error) { return "2.0", nil }
	downloads := 0
	m.downloadBinaryFn = func(_ context.Context, _ string, dest string) error {
		downloads++
		return os.WriteFile(dest, []byte("new"), 0700)
	}
	if err := m.EnsureInstalledAndUpdated(context.Background()); err != nil {
		t.Fatal(err)
	}
	if downloads != 1 || !m.installed || !strings.Contains(stderr.String(), "read local") {
		t.Fatalf("update downloads=%d,installed=%v,warnings=%s", downloads, m.installed, stderr.String())
	}
	if m.NeedCheckVersion() {
		t.Fatal("completed update did not stamp cache")
	}
}

func TestVersionHTTPClientPropagatesContext(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	marker := struct{}{}
	ctx := context.WithValue(context.Background(), marker, "request")
	http.DefaultTransport = boundaryTransport(func(r *http.Request) (*http.Response, error) {
		if r.Context().Value(marker) != "request" {
			t.Error("version request dropped context")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(" 2.0 "))}, nil
	})
	got, err := GetLatestVersion(ctx, "https://example.invalid/")
	if err != nil || got != "2.0" {
		t.Fatalf("version=%q,%v", got, err)
	}
}
