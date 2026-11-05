package dgate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestMain(m *testing.M) {
	if len(os.Args) >= 2 && os.Args[1] == "-dgate-helper" {
		code := 0
		if len(os.Args) >= 3 {
			_, _ = fmt.Sscanf(os.Args[2], "%d", &code)
		}
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func helperExit(code int) *exec.Cmd {
	return exec.Command(os.Args[0], "-dgate-helper", fmt.Sprintf("%d", code))
}

func TestCommandForwardsHelpWithoutRewriting(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-dgate")
	if err := os.WriteFile(bin, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(execPathEnv, bin)
	var got []string
	prev := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		if name != bin {
			t.Errorf("executed %s", name)
		}
		got = append([]string(nil), args...)
		return helperExit(0)
	}
	t.Cleanup(func() { execCommandFunc = prev })

	cmd := NewDgateCommand()
	if !cmd.RawArgs || !cmd.SkipDefaultHelp || cmd.KeepArgs {
		t.Fatalf("raw=%v skip=%v keep=%v", cmd.RawArgs, cmd.SkipDefaultHelp, cmd.KeepArgs)
	}
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	ctx.EnterCommand(cmd)
	cmd.Execute(ctx, []string{"--profile", "demo", "-p", "other", "help", "output"})
	if strings.Join(got, " ") != "help output" {
		t.Fatalf("forwarded %#v", got)
	}
}

func TestCommandPassesHelpFlagsThrough(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-dgate")
	if err := os.WriteFile(bin, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(execPathEnv, bin)
	prev := execCommandFunc
	t.Cleanup(func() { execCommandFunc = prev })
	for _, args := range [][]string{{"--help"}, {"-h"}, {"--region", "cn-beijing", "--quiet"}} {
		var got []string
		execCommandFunc = func(name string, forwarded ...string) *exec.Cmd {
			got = append([]string(nil), forwarded...)
			return helperExit(0)
		}
		cmd := NewDgateCommand()
		ctx := cli.NewCommandContext(io.Discard, io.Discard)
		ctx.EnterCommand(cmd)
		cmd.Execute(ctx, args)
		if strings.Join(got, " ") != strings.Join(args, " ") {
			t.Fatalf("args %#v forwarded %#v", args, got)
		}
	}
}

func TestCommandPropagatesChildExitCode(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-dgate")
	if err := os.WriteFile(bin, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(execPathEnv, bin)
	prev := execCommandFunc
	execCommandFunc = func(string, ...string) *exec.Cmd { return helperExit(4) }
	t.Cleanup(func() { execCommandFunc = prev })
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)

	var stderr bytes.Buffer
	cmd := NewDgateCommand()
	ctx := cli.NewCommandContext(io.Discard, &stderr)
	ctx.EnterCommand(cmd)
	cmd.Execute(ctx, []string{"exec"})
	exitErr := &ExitError{Code: 4}
	if exitErr.ExitCode() != 4 || !strings.Contains(exitErr.Error(), "4") {
		t.Fatalf("exit error = %s code %d", exitErr.Error(), exitErr.ExitCode())
	}
	if stderr.Len() != 0 {
		t.Fatalf("exit code must not be printed as an error: %s", stderr.String())
	}
}

func TestCommandReportsExecFailure(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-dgate")
	if err := os.WriteFile(bin, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(execPathEnv, bin)
	prev := execCommandFunc
	execCommandFunc = func(string, ...string) *exec.Cmd {
		return exec.Command(filepath.Join(t.TempDir(), "missing-bin"))
	}
	t.Cleanup(func() { execCommandFunc = prev })
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)

	var stderr bytes.Buffer
	cmd := NewDgateCommand()
	ctx := cli.NewCommandContext(io.Discard, &stderr)
	ctx.EnterCommand(cmd)
	cmd.Execute(ctx, nil)
	if !strings.Contains(stderr.String(), "failed to execute") {
		t.Fatalf("stderr %q", stderr.String())
	}
}

func TestMissingExternalBinary(t *testing.T) {
	t.Setenv(execPathEnv, filepath.Join(t.TempDir(), "missing"))
	c := NewContext(nil)
	if err := c.initialize(); err != nil {
		t.Fatal(err)
	}
	err := c.ensureInstalled()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err %v", err)
	}
}

func TestUnsupportedOS(t *testing.T) {
	prevOS := runtimeGOOSFunc
	prevArch := runtimeGOARCHFunc
	runtimeGOOSFunc = func() string { return "js" }
	runtimeGOARCHFunc = func() string { return "wasm" }
	t.Cleanup(func() {
		runtimeGOOSFunc = prevOS
		runtimeGOARCHFunc = prevArch
	})
	t.Setenv(execPathEnv, "")
	c := NewContext(nil)
	if err := c.initialize(); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("err %v", err)
	}
}

func TestExternalBinaryAllowsUnsupportedOS(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake")
	if err := os.WriteFile(bin, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(execPathEnv, bin)
	prevOS := runtimeGOOSFunc
	prevArch := runtimeGOARCHFunc
	runtimeGOOSFunc = func() string { return "js" }
	runtimeGOARCHFunc = func() string { return "wasm" }
	t.Cleanup(func() {
		runtimeGOOSFunc = prevOS
		runtimeGOARCHFunc = prevArch
	})
	c := NewContext(nil)
	if err := c.initialize(); err != nil {
		t.Fatal(err)
	}
	if err := c.ensureInstalled(); err != nil {
		t.Fatal(err)
	}
}

func TestFreshInstallFailureAndInvalidVersion(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(execPathEnv, "")
	prevPath := getConfigurePathFunc
	prevVer := getRemoteVersionFunc
	getConfigurePathFunc = func() string { return dir }
	versionCalls := 0
	getRemoteVersionFunc = func(string) (string, error) { versionCalls++; return "", os.ErrClosed }
	t.Cleanup(func() {
		getConfigurePathFunc = prevPath
		getRemoteVersionFunc = prevVer
	})
	c := NewContext(nil)
	if err := c.initialize(); err != nil {
		t.Fatal(err)
	}
	signedDgateManifest(t, c, "1.2.3", []byte("package"))
	if err := c.ensureInstalled(); err == nil || !strings.Contains(err.Error(), "auto-download failed") || !strings.Contains(err.Error(), os.ErrClosed.Error()) {
		t.Fatalf("err %v", err)
	}
	if versionCalls != 1 {
		t.Fatalf("remote-version failure not reached: %d", versionCalls)
	}

	getRemoteVersionFunc = func(string) (string, error) { return "../bad", nil }
	if err := c.ensureInstalled(); err == nil || !strings.Contains(err.Error(), "invalid version") {
		t.Fatalf("err %v", err)
	}
}

func TestVersionCheckSkipsAndUpdates(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(execPathEnv, "")
	prevOS := runtimeGOOSFunc
	prevArch := runtimeGOARCHFunc
	runtimeGOOSFunc = func() string { return "darwin" }
	runtimeGOARCHFunc = func() string { return "arm64" }
	if err := os.WriteFile(filepath.Join(dir, "dgate"), []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0)
	if err := os.WriteFile(filepath.Join(dir, ".dgate_version_check"), []byte(fmt.Sprintf("%d", now.Unix())), 0644); err != nil {
		t.Fatal(err)
	}
	prevPath := getConfigurePathFunc
	prevNow := timeNowFunc
	prevVer := getRemoteVersionFunc
	getConfigurePathFunc = func() string { return dir }
	timeNowFunc = func() time.Time { return now.Add(time.Minute) }
	called := false
	getRemoteVersionFunc = func(string) (string, error) {
		called = true
		return "0.6.23", nil
	}
	t.Cleanup(func() {
		getConfigurePathFunc = prevPath
		timeNowFunc = prevNow
		getRemoteVersionFunc = prevVer
		runtimeGOOSFunc = prevOS
		runtimeGOARCHFunc = prevArch
	})
	c := NewContext(nil)
	if err := c.initialize(); err != nil || c.ensureInstalled() != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("fresh cache should skip the remote check")
	}

	timeNowFunc = func() time.Time { return now.Add(VersionCheckTTL + time.Second) }
	signedDgateManifest(t, c, "0.6.23", []byte("same package"))
	if err := os.WriteFile(filepath.Join(dir, ".dgate_version"), []byte("0.6.23"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := c.ensureInstalled(); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("stale cache should check")
	}

	var stderr bytes.Buffer
	c.originCtx = cli.NewCommandContext(io.Discard, &stderr)
	timeNowFunc = func() time.Time { return now.Add(2*VersionCheckTTL + 2*time.Second) }
	getRemoteVersionFunc = func(string) (string, error) { return "9.9.9", nil }
	prevDL := downloadBinaryFunc
	downloadCalls := 0
	downloadBinaryFunc = func(string, string) error { downloadCalls++; return os.ErrPermission }
	signedDgateManifest(t, c, "9.9.9", []byte("new"))
	t.Cleanup(func() { downloadBinaryFunc = prevDL })
	if err := c.ensureInstalled(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "failed to update") {
		t.Fatalf("stderr %q", stderr.String())
	}
	if downloadCalls != 1 {
		t.Fatalf("original download failure branch not reached: %d", downloadCalls)
	}
}

func TestWindowsReplaceAndChecksumPaths(t *testing.T) {
	payload := []byte("dgate-bytes")
	sum := sha256.Sum256(payload)
	good := hex.EncodeToString(sum[:])
	mux := http.NewServeMux()
	mux.HandleFunc("/latest.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0.6.23\n"))
	})
	mux.HandleFunc("/0.6.23/dgate_windows_amd64.exe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	checksum := good + "  dgate_windows_amd64.exe\n"
	mux.HandleFunc("/0.6.23/dgate_windows_amd64.exe.sha256", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(checksum))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dgate.exe"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(execPathEnv, "")
	t.Setenv(downloadBaseEnv, srv.URL+"/")
	prevPath := getConfigurePathFunc
	prevOS := runtimeGOOSFunc
	prevArch := runtimeGOARCHFunc
	getConfigurePathFunc = func() string { return dir }
	runtimeGOOSFunc = func() string { return "windows" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	t.Cleanup(func() {
		getConfigurePathFunc = prevPath
		runtimeGOOSFunc = prevOS
		runtimeGOARCHFunc = prevArch
	})

	var stderr bytes.Buffer
	c := NewContext(cli.NewCommandContext(io.Discard, &stderr))
	if err := c.initialize(); err != nil {
		t.Fatal(err)
	}
	if c.artifactName() != "dgate_windows_amd64.exe" {
		t.Fatalf("artifact %s", c.artifactName())
	}
	if !strings.HasSuffix(c.execFilePath, "dgate.exe") {
		t.Fatalf("exec %s", c.execFilePath)
	}
	signedDgateManifest(t, c, "0.6.23", payload)
	if err := c.ensureInstalled(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(c.execFilePath)
	if err != nil || string(got) != string(payload) {
		t.Fatalf("installed %q err=%v", got, err)
	}
	if !strings.Contains(stderr.String(), "Installing dgate") {
		t.Fatalf("stderr %q", stderr.String())
	}

	checksum = "deadbeef\n"
	c.installed = false
	_ = os.Remove(c.execFilePath)
	if err := c.ensureInstalled(); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err %v", err)
	}

	checksum = ""
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".sha256") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.HasSuffix(r.URL.Path, "latest.txt") {
			_, _ = w.Write([]byte("0.6.23\n"))
			return
		}
		_, _ = w.Write(payload)
	})
	if err := c.ensureInstalled(); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("missing checksum accepted: %v", err)
	}
	if _, err := os.Stat(c.execFilePath); !os.IsNotExist(err) {
		t.Fatalf("unverified binary installed: %v", err)
	}
}

func TestRemoteVersionAndDownloadErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte("  v1.2.3 \n"))
		case "/bad":
			_, _ = w.Write([]byte("latest\n"))
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		case "/bin":
			_, _ = w.Write([]byte("abc"))
		default:
			w.WriteHeader(http.StatusBadGateway)
		}
	}))
	t.Cleanup(srv.Close)

	ver, err := getRemoteVersion(srv.URL + "/ok")
	if err != nil || ver != "v1.2.3" {
		t.Fatalf("ver %q err %v", ver, err)
	}
	if _, err := getRemoteVersion(srv.URL + "/bad"); err == nil {
		t.Fatal("expected invalid version")
	}
	if _, err := getRemoteVersion(srv.URL + "/missing"); err == nil {
		t.Fatal("expected http error")
	}
	if _, err := getRemoteVersion("://bad"); err == nil {
		t.Fatal("expected request error")
	}

	dest := filepath.Join(t.TempDir(), "nested", "dgate")
	if err := downloadBinary(srv.URL+"/bin", dest); err != nil {
		t.Fatal(err)
	}
	if err := downloadBinary(srv.URL+"/missing", dest); err == nil {
		t.Fatal("expected status error")
	}
	if err := downloadBinary("://bad", dest); err == nil {
		t.Fatal("expected request error")
	}
	if err := downloadBinary(srv.URL+"/bin", filepath.Join(dest, "child")); err == nil {
		t.Fatal("expected parent-is-file error")
	}
}

func TestChecksumAndVersionFileEdges(t *testing.T) {
	body := []byte("payload")
	sum := sha256.Sum256(body)
	dir := t.TempDir()
	file := filepath.Join(dir, "bin")
	if err := os.WriteFile(file, body, 0644); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(dir, "empty-version")
	if err := os.WriteFile(empty, []byte(" \n"), 0644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("k") {
		case "ok":
			_, _ = w.Write([]byte(strings.ToUpper(hex.EncodeToString(sum[:])) + "\n"))
		case "empty":
			_, _ = w.Write([]byte("\n"))
		case "bad":
			w.WriteHeader(http.StatusTeapot)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	if err := verifySHA256(srv.URL+"?k=ok", file); err != nil {
		t.Fatal(err)
	}
	if err := verifySHA256(srv.URL+"?k=missing", file); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("missing checksum accepted: %v", err)
	}
	if err := verifySHA256(srv.URL+"?k=empty", file); err == nil {
		t.Fatal("expected empty checksum")
	}
	if err := verifySHA256(srv.URL+"?k=bad", file); err == nil {
		t.Fatal("expected checksum http error")
	}
	if err := verifySHA256("://bad", file); err == nil {
		t.Fatal("expected request error")
	}

	c := NewContext(nil)
	c.versionFilePath = empty
	if err := c.readLocalVersion(); err == nil {
		t.Fatal("expected empty version")
	}
	c.versionFilePath = filepath.Join(dir, "absent")
	if err := c.readLocalVersion(); err == nil {
		t.Fatal("expected missing version")
	}
	c.checkVersionCacheFilePath = filepath.Join(dir, "absent-cache")
	if !c.needCheckVersion() {
		t.Fatal("missing cache should check")
	}
	if err := os.WriteFile(c.checkVersionCacheFilePath, []byte("nope"), 0644); err != nil {
		t.Fatal(err)
	}
	if !c.needCheckVersion() {
		t.Fatal("bad cache should check")
	}
}

func TestFilterEnvAndBaseURL(t *testing.T) {
	t.Setenv(downloadBaseEnv, "https://example.test/dgate/")
	c := NewContext(nil)
	if c.baseURL() != "https://example.test/dgate" {
		t.Fatalf("base %s", c.baseURL())
	}
	got := filterEnv([]string{"A=1", "ALIBABA_CLOUD_DGATE_COMPAT_MODE=old", "B=2"}, map[string]string{compatModeEnv: compatModeValue})
	if strings.Join(got, ",") != "A=1,B=2" {
		t.Fatalf("env %#v", got)
	}
}

func TestInstallRenameFallback(t *testing.T) {
	dir := t.TempDir()
	prevOS := runtimeGOOSFunc
	prevArch := runtimeGOARCHFunc
	runtimeGOOSFunc = func() string { return "darwin" }
	runtimeGOARCHFunc = func() string { return "arm64" }
	blocked := filepath.Join(dir, "dgate")
	if err := os.WriteFile(blocked, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(execPathEnv, "")
	prevPath := getConfigurePathFunc
	prevDL := downloadBinaryFunc
	getConfigurePathFunc = func() string { return dir }
	downloadBinaryFunc = func(url, dest string) error {
		return os.WriteFile(dest, []byte("new"), 0644)
	}
	t.Cleanup(func() {
		getConfigurePathFunc = prevPath
		downloadBinaryFunc = prevDL
		runtimeGOOSFunc = prevOS
		runtimeGOARCHFunc = prevArch
	})
	c := NewContext(nil)
	if err := c.initialize(); err != nil {
		t.Fatal(err)
	}
	c.versionRemote = "1.2.3"
	signedDgateManifest(t, c, c.versionRemote, []byte("new"))
	dgateChecksumFixture(t, []byte("new"))
	renameCalls := 0
	c.rename = func(string, string) error { renameCalls++; return os.ErrPermission }
	if err := c.install(); err == nil || !strings.Contains(err.Error(), "failed to install") {
		t.Fatalf("err %v", err)
	}
	if renameCalls != 1 {
		t.Fatalf("rename failure branch not reached: %d", renameCalls)
	}
	if got, err := os.ReadFile(blocked); err != nil || string(got) != "old" {
		t.Fatalf("failed rename damaged old binary: %q %v", got, err)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRunSurfacesInitAndInstallErrors(t *testing.T) {
	t.Setenv(execPathEnv, "")
	prevOS := runtimeGOOSFunc
	prevArch := runtimeGOARCHFunc
	runtimeGOOSFunc = func() string { return "js" }
	runtimeGOARCHFunc = func() string { return "wasm" }
	t.Cleanup(func() {
		runtimeGOOSFunc = prevOS
		runtimeGOARCHFunc = prevArch
	})
	if err := NewContext(nil).Run(nil); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("init err %v", err)
	}

	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	t.Setenv(execPathEnv, filepath.Join(t.TempDir(), "missing"))
	if err := NewContext(nil).Run(nil); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("install err %v", err)
	}
}

func TestInstallPermissionAndDirectoryErrors(t *testing.T) {
	prevOS := runtimeGOOSFunc
	prevDL := downloadBinaryFunc
	prevHTTP := httpDoFunc
	t.Cleanup(func() {
		runtimeGOOSFunc = prevOS
		downloadBinaryFunc = prevDL
		httpDoFunc = prevHTTP
	})
	runtimeGOOSFunc = func() string { return "linux" }
	payload := []byte("new executable")
	var downloaded string
	downloadBinaryFunc = func(_ string, dest string) error {
		downloaded = dest
		return os.WriteFile(dest, payload, 0600)
	}
	sum := sha256.Sum256(payload)
	checksumCalls := 0
	httpDoFunc = func(*http.Request) (*http.Response, error) {
		checksumCalls++
		// Remove the real file only after checksum validation has finished,
		// when the response closes, to reach the original chmod branch.
		return &http.Response{StatusCode: http.StatusOK, Body: dgateCloseHook{
			Reader: strings.NewReader(hex.EncodeToString(sum[:]) + "\n"),
			close:  func() error { return os.Remove(downloaded) },
		}}, nil
	}

	c := NewContext(nil)
	c.configPath = t.TempDir()
	c.osType, c.osArch = "linux", "amd64"
	c.versionRemote = "1.2.3"
	c.execFilePath = filepath.Join(c.configPath, "dgate")
	c.versionFilePath = filepath.Join(c.configPath, ".dgate_version")
	signedDgateManifest(t, c, c.versionRemote, payload)
	if err := c.install(); err == nil || !strings.Contains(err.Error(), "exec permission") {
		t.Fatalf("chmod failure not reached: %v", err)
	}
	if checksumCalls != 1 {
		t.Fatalf("checksum/chmod branch not reached: %d", checksumCalls)
	}

	runtimeGOOSFunc = func() string { return "windows" }
	parent := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(parent, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	c.execFilePath = filepath.Join(parent, "dgate.exe")
	c.osType = "windows"
	signedDgateManifest(t, c, c.versionRemote, payload)
	if err := c.install(); err == nil {
		t.Fatal("expected mkdir failure")
	}
}

func TestVersionCacheUnreadable(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, "cache")
	if err := os.Mkdir(cache, 0755); err != nil {
		t.Fatal(err)
	}
	c := NewContext(nil)
	c.checkVersionCacheFilePath = cache
	if !c.needCheckVersion() {
		t.Fatal("unreadable cache should check")
	}
}

func TestRemoteAndDownloadFailureBranches(t *testing.T) {
	prev := httpDoFunc
	t.Cleanup(func() { httpDoFunc = prev })
	httpDoFunc = func(*http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF }
	if _, err := getRemoteVersion("https://example.test/latest.txt"); err == nil {
		t.Fatal("expected transport error")
	}
	if err := verifySHA256("https://example.test/sum", filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected checksum transport error")
	}

	httpDoFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(errReader{})}, nil
	}
	if _, err := getRemoteVersion("https://example.test/latest.txt"); err == nil {
		t.Fatal("expected body error")
	}
	file := filepath.Join(t.TempDir(), "bin")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifySHA256("https://example.test/sum", file); err == nil {
		t.Fatal("expected checksum body error")
	}

	httpDoFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("abc\n"))}, nil
	}
	if err := verifySHA256("https://example.test/sum", filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("expected missing file")
	}
	dir := t.TempDir()
	if err := verifySHA256("https://example.test/sum", dir); err == nil {
		t.Fatal("expected directory read error")
	}

	if err := downloadBinary("http://\x7f", filepath.Join(t.TempDir(), "out")); err == nil {
		t.Fatal("expected bad download url")
	}
	prevClient := httpClient
	t.Cleanup(func() { httpClient = prevClient })
	httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, io.ErrUnexpectedEOF
	})}
	if err := downloadBinary("https://example.test/dgate", filepath.Join(t.TempDir(), "out")); err == nil {
		t.Fatal("expected download transport error")
	}
	httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(errReader{}), Header: make(http.Header)}, nil
	})}
	if err := downloadBinary("https://example.test/dgate", filepath.Join(t.TempDir(), "out")); err == nil {
		t.Fatal("expected download body error")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hi"))
	}))
	t.Cleanup(srv.Close)
	if err := downloadBinary(srv.URL, t.TempDir()); err == nil {
		t.Fatal("expected create failure")
	}
}

func TestDownloadRejectsOversizedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1<<20)
		left := maxDownloadSize + 1
		for left > 0 {
			n := len(buf)
			if left < n {
				n = left
			}
			if _, err := w.Write(buf[:n]); err != nil {
				return
			}
			left -= n
		}
	}))
	t.Cleanup(srv.Close)
	err := downloadBinary(srv.URL, filepath.Join(t.TempDir(), "big"))
	if err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("err %v", err)
	}
}
