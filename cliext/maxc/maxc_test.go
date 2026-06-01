package maxc

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

// --- test helpers -----------------------------------------------------------

type tarEntry struct {
	Name     string
	Body     string
	Mode     int64
	Linkname string
}

func buildTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for _, e := range entries {
		hdr := &tar.Header{
			Name: e.Name,
			Mode: e.Mode,
			Size: int64(len(e.Body)),
		}
		switch {
		case e.Linkname != "":
			hdr.Typeflag = tar.TypeSymlink
			hdr.Linkname = e.Linkname
			hdr.Size = 0
		case strings.HasSuffix(e.Name, "/"):
			hdr.Typeflag = tar.TypeDir
			hdr.Size = 0
		default:
			hdr.Typeflag = tar.TypeReg
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("tar WriteHeader: %v", err)
		}
		if hdr.Typeflag == tar.TypeReg && len(e.Body) > 0 {
			if _, err := tw.Write([]byte(e.Body)); err != nil {
				t.Fatalf("tar Write: %v", err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar Close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gz Close: %v", err)
	}
	return buf.Bytes()
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type fakeOSS struct {
	srv       *httptest.Server
	requested []string
}

func newFakeOSS(t *testing.T, objects map[string][]byte) *fakeOSS {
	t.Helper()
	f := &fakeOSS{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requested = append(f.requested, r.URL.Path)
		body, ok := objects[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func withCtx(t *testing.T, platform string) *Context {
	t.Helper()
	parts := strings.SplitN(platform, "-", 2)
	if len(parts) != 2 {
		t.Fatalf("bad platform %q", platform)
	}

	origGOOS, origGOARCH := runtimeGOOSFunc, runtimeGOARCHFunc
	origCfg := getConfigurePathFunc
	t.Cleanup(func() {
		runtimeGOOSFunc = origGOOS
		runtimeGOARCHFunc = origGOARCH
		getConfigurePathFunc = origCfg
	})

	tmp := t.TempDir()
	runtimeGOOSFunc = func() string { return parts[0] }
	runtimeGOARCHFunc = func() string { return parts[1] }
	getConfigurePathFunc = func() string { return tmp }

	c := &Context{}
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()
	return c
}

func newOriginCtx() *cli.Context {
	return cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
}

func withLoadProfileStub(t *testing.T, fn func(*cli.Context) (config.Profile, error)) {
	t.Helper()
	orig := loadProfileFunc
	t.Cleanup(func() { loadProfileFunc = orig })
	loadProfileFunc = fn
}

func withExecCommandStub(t *testing.T, fn func(name string, arg ...string) *exec.Cmd) {
	t.Helper()
	orig := execCommandFunc
	t.Cleanup(func() { execCommandFunc = orig })
	execCommandFunc = fn
}

func withGetLatestStub(t *testing.T, fn func(*Context) (string, error)) {
	t.Helper()
	orig := getLatestVersionFunc
	t.Cleanup(func() { getLatestVersionFunc = orig })
	getLatestVersionFunc = fn
}

func stageTarball(t *testing.T, c *Context, version string) {
	t.Helper()
	tar := buildTarGz(t, []tarEntry{
		{Name: "maxc/", Mode: 0o755},
		{Name: "maxc/maxc", Body: "#!/bin/sh\necho fake\n", Mode: 0o755},
	})
	sha := sha256Hex(tar)
	oss := newFakeOSS(t, map[string][]byte{
		version + "/" + c.platformKey + "/maxc.tar.gz":        tar,
		version + "/" + c.platformKey + "/maxc.tar.gz.sha256": []byte(sha),
	})
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", oss.srv.URL)
}

// --- platform / init --------------------------------------------------------

func TestPlatformWhitelist(t *testing.T) {
	cases := map[string]bool{
		"linux-amd64":   true,
		"linux-arm64":   true,
		"darwin-amd64":  true,
		"darwin-arm64":  true,
		"windows-amd64": true,
		"windows-arm64": false,
		"freebsd-amd64": false,
		"linux-386":     false,
	}
	for k, want := range cases {
		_, got := platformPaths[k]
		if got != want {
			t.Errorf("platformPaths[%q] = %v, want %v", k, got, want)
		}
	}
}

func TestInitBasicInfo_DerivesPathFromConfig(t *testing.T) {
	origGetConfigurePath := getConfigurePathFunc
	defer func() { getConfigurePathFunc = origGetConfigurePath }()
	getConfigurePathFunc = func() string { return "/tmp/test-aliyun" }

	c := &Context{}
	c.InitBasicInfo()

	if c.installDir != "/tmp/test-aliyun/maxc" {
		t.Errorf("installDir = %q, want /tmp/test-aliyun/maxc", c.installDir)
	}
	wantExec := "/tmp/test-aliyun/maxc/maxc"
	if runtimeGOOSFunc() == "windows" {
		wantExec += ".exe"
	}
	if c.execFilePath != wantExec {
		t.Errorf("execFilePath = %q, want %q", c.execFilePath, wantExec)
	}
}

func TestInitBasicInfo_ExecPathEnvOverride(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_MAXC_EXEC_PATH", "/opt/maxc/maxc")
	origGetConfigurePath := getConfigurePathFunc
	defer func() { getConfigurePathFunc = origGetConfigurePath }()
	getConfigurePathFunc = func() string { return "/tmp/test-aliyun" }

	c := &Context{}
	c.InitBasicInfo()

	if c.execFilePath != "/opt/maxc/maxc" {
		t.Errorf("env override ignored: execFilePath = %q", c.execFilePath)
	}
}

func TestCheckOsTypeAndArch_SetsSupportFlag(t *testing.T) {
	origGOOS, origGOARCH := runtimeGOOSFunc, runtimeGOARCHFunc
	defer func() {
		runtimeGOOSFunc = origGOOS
		runtimeGOARCHFunc = origGOARCH
	}()

	cases := []struct {
		os, arch string
		support  bool
	}{
		{"linux", "amd64", true},
		{"darwin", "arm64", true},
		{"windows", "amd64", true},
		{"windows", "arm64", false},
		{"freebsd", "amd64", false},
		{"linux", "386", false},
	}
	for _, tc := range cases {
		runtimeGOOSFunc = func() string { return tc.os }
		runtimeGOARCHFunc = func() string { return tc.arch }
		c := &Context{}
		c.CheckOsTypeAndArch()
		if c.osSupport != tc.support {
			t.Errorf("%s-%s: osSupport = %v, want %v", tc.os, tc.arch, c.osSupport, tc.support)
		}
	}
}

// --- credentials ------------------------------------------------------------

func TestInjectCreds_AK(t *testing.T) {
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name:            "default",
			Mode:            config.AK,
			AccessKeyId:     "ak-id",
			AccessKeySecret: "ak-secret",
			RegionId:        "cn-hangzhou",
		}, nil
	})

	c := &Context{originCtx: newOriginCtx()}
	if err := c.InjectAliyunCredentials(nil); err != nil {
		t.Fatalf("InjectAliyunCredentials: %v", err)
	}
	if got := c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"]; got != "ak-id" {
		t.Errorf("ACCESS_KEY_ID = %q", got)
	}
	if got := c.envMap["ALIBABA_CLOUD_ACCESS_KEY_SECRET"]; got != "ak-secret" {
		t.Errorf("ACCESS_KEY_SECRET = %q", got)
	}
	if _, ok := c.envMap["ALIBABA_CLOUD_SECURITY_TOKEN"]; ok {
		t.Errorf("SECURITY_TOKEN must be absent for plain AK")
	}
	if got := c.envMap["MAXCOMPUTE_REGION"]; got != "cn-hangzhou" {
		t.Errorf("MAXCOMPUTE_REGION = %q", got)
	}
}

func TestInjectCreds_StsToken(t *testing.T) {
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name:            "sts-profile",
			Mode:            config.StsToken,
			AccessKeyId:     "sts-id",
			AccessKeySecret: "sts-secret",
			StsToken:        "sts-token-xyz",
			RegionId:        "cn-shanghai",
		}, nil
	})

	c := &Context{originCtx: newOriginCtx()}
	if err := c.InjectAliyunCredentials(nil); err != nil {
		t.Fatalf("InjectAliyunCredentials: %v", err)
	}
	if got := c.envMap["ALIBABA_CLOUD_SECURITY_TOKEN"]; got != "sts-token-xyz" {
		t.Errorf("SECURITY_TOKEN = %q, want sts-token-xyz", got)
	}
	if got := c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"]; got != "sts-id" {
		t.Errorf("ACCESS_KEY_ID = %q", got)
	}
}

func TestInjectCreds_NoProfile_Silent(t *testing.T) {
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{}, errors.New("profile not found")
	})

	c := &Context{originCtx: newOriginCtx()}
	if err := c.InjectAliyunCredentials(nil); err != nil {
		t.Errorf("expected silent fallback (nil), got %v", err)
	}
	if len(c.envMap) != 0 {
		t.Errorf("envMap should be empty on profile miss, got %v", c.envMap)
	}
}

func TestInjectCreds_GetCredentialFails(t *testing.T) {
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name: "broken",
			Mode: config.AuthenticateMode("DefinitelyNotAValidMode"),
		}, nil
	})

	c := &Context{originCtx: newOriginCtx()}
	err := c.InjectAliyunCredentials(nil)
	if err == nil {
		t.Fatal("expected credential resolution error, got nil")
	}
}

func TestInjectCreds_NoRegion_DoesNotSetMAXCOMPUTE_REGION(t *testing.T) {
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name:            "no-region",
			Mode:            config.AK,
			AccessKeyId:     "x",
			AccessKeySecret: "y",
		}, nil
	})

	c := &Context{originCtx: newOriginCtx()}
	if err := c.InjectAliyunCredentials(nil); err != nil {
		t.Fatalf("InjectAliyunCredentials: %v", err)
	}
	if _, ok := c.envMap["MAXCOMPUTE_REGION"]; ok {
		t.Errorf("MAXCOMPUTE_REGION must not be set when profile.RegionId is empty")
	}
}

func TestInjectCreds_PreservesExistingEnvMap(t *testing.T) {
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name: "ak", Mode: config.AK, AccessKeyId: "i", AccessKeySecret: "s",
		}, nil
	})

	c := &Context{originCtx: newOriginCtx(), envMap: map[string]string{"PREEXISTING": "v"}}
	if err := c.InjectAliyunCredentials(nil); err != nil {
		t.Fatalf("InjectAliyunCredentials: %v", err)
	}
	if c.envMap["PREEXISTING"] != "v" {
		t.Errorf("pre-existing env entries were dropped")
	}
}

// --- execute ----------------------------------------------------------------

func newCtxForExecute(t *testing.T) *Context {
	t.Helper()
	return &Context{
		originCtx:    cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{}),
		execFilePath: "/path/to/maxc",
		envMap:       map[string]string{"INJECTED": "yes"},
	}
}

func TestExecute_PassesArgsAndEnv(t *testing.T) {
	var spyName string
	var spyArgs []string
	withExecCommandStub(t, func(name string, arg ...string) *exec.Cmd {
		spyName = name
		spyArgs = append([]string(nil), arg...)
		return exec.Command("/bin/sh", "-c", "exit 0")
	})

	c := newCtxForExecute(t)
	if err := c.Execute([]string{"query", "--sql", "select 1"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if spyName != "/path/to/maxc" {
		t.Errorf("exec name = %q", spyName)
	}
	if !reflect.DeepEqual(spyArgs, []string{"query", "--sql", "select 1"}) {
		t.Errorf("exec args = %v", spyArgs)
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
	withExecCommandStub(t, func(name string, arg ...string) *exec.Cmd {
		return exec.Command("/bin/sh", "-c", `test "$INJECTED" = "yes"`)
	})
	c := newCtxForExecute(t)
	if err := c.Execute(nil); err != nil {
		t.Errorf("INJECTED env not propagated to child: %v", err)
	}
}

func TestExecute_OverrideWinsOverInherited(t *testing.T) {
	t.Setenv("INJECTED", "parent-value")
	withExecCommandStub(t, func(name string, arg ...string) *exec.Cmd {
		return exec.Command("/bin/sh", "-c", `test "$INJECTED" = "yes"`)
	})
	c := newCtxForExecute(t)
	if err := c.Execute(nil); err != nil {
		t.Errorf("env override did not win over parent: %v", err)
	}
}

// --- RemoveFlagsForMainCli --------------------------------------------------

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

func TestRemoveFlagsForMainCli_PreservesChildFlags(t *testing.T) {
	c := &Context{}
	in := []string{"query", "--sql", "show tables", "--output", "json"}
	got := c.RemoveFlagsForMainCli(in)
	if !reflect.DeepEqual(got, in) {
		t.Errorf("child flags were modified: got %v want %v", got, in)
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
	for _, e := range out {
		if e == "B=2" {
			t.Errorf("base B=2 still present alongside override: %v", out)
		}
	}
}

// --- install / download -----------------------------------------------------

func TestEffectiveBaseURL_EnvOverride(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", "https://override.example.com/staging/")
	c := &Context{}
	got := c.effectiveBaseURL()
	if got != "https://override.example.com/staging" {
		t.Errorf("effectiveBaseURL = %q, want trailing slash trimmed", got)
	}
}

func TestEffectiveBaseURL_DefaultWhenEnvEmpty(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", "")
	c := &Context{}
	if got := c.effectiveBaseURL(); got != downloadBaseURL {
		t.Errorf("effectiveBaseURL = %q, want package default %q", got, downloadBaseURL)
	}
}

func TestTarballURL_Format(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", "https://b.example.com")
	c := &Context{platformKey: "darwin-arm64"}

	want := "https://b.example.com/0.3.0/darwin-arm64/maxc.tar.gz"
	if got := c.tarballURL("0.3.0"); got != want {
		t.Errorf("tarballURL = %q, want %q", got, want)
	}
	if got := c.tarballShaURL("0.3.0"); got != want+".sha256" {
		t.Errorf("tarballShaURL = %q, want %q", got, want+".sha256")
	}
	if got := c.latestVersionURL(); got != "https://b.example.com/versions/latest" {
		t.Errorf("latestVersionURL = %q", got)
	}
}

func TestDownloadAndInstall_HappyPath(t *testing.T) {
	tar := buildTarGz(t, []tarEntry{
		{Name: "maxc/", Mode: 0o755},
		{Name: "maxc/maxc", Body: "#!/bin/sh\necho fake\n", Mode: 0o755},
		{Name: "maxc/_internal/libpython.so", Body: "elf data", Mode: 0o644},
	})
	sha := sha256Hex(tar)

	oss := newFakeOSS(t, map[string][]byte{
		"0.3.0/linux-amd64/maxc.tar.gz":        tar,
		"0.3.0/linux-amd64/maxc.tar.gz.sha256": []byte(sha + "\n"),
	})
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", oss.srv.URL)

	c := withCtx(t, "linux-amd64")
	if err := c.downloadAndInstall("0.3.0"); err != nil {
		t.Fatalf("downloadAndInstall: %v", err)
	}

	if !fileExists(c.execFilePath) {
		t.Errorf("execFilePath %s should exist after install", c.execFilePath)
	}
	if !fileExists(filepath.Join(c.installDir, "_internal", "libpython.so")) {
		t.Errorf("nested file missing — extract was shallow")
	}

	fi, err := os.Stat(c.execFilePath)
	if err != nil {
		t.Fatalf("stat exec: %v", err)
	}
	if fi.Mode().Perm()&0o100 == 0 {
		t.Errorf("execFilePath mode = %v, expected execute bit set", fi.Mode())
	}

	verBytes, err := os.ReadFile(c.versionFilePath)
	if err != nil {
		t.Fatalf("read .version: %v", err)
	}
	if string(verBytes) != "0.3.0" {
		t.Errorf(".version = %q, want 0.3.0", string(verBytes))
	}
}

func TestDownloadAndInstall_RejectsBadSha(t *testing.T) {
	tar := buildTarGz(t, []tarEntry{
		{Name: "maxc/", Mode: 0o755},
		{Name: "maxc/maxc", Body: "real", Mode: 0o755},
	})

	oss := newFakeOSS(t, map[string][]byte{
		"0.3.0/linux-amd64/maxc.tar.gz":        tar,
		"0.3.0/linux-amd64/maxc.tar.gz.sha256": []byte("deadbeef" + strings.Repeat("0", 56)),
	})
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", oss.srv.URL)

	c := withCtx(t, "linux-amd64")
	err := c.downloadAndInstall("0.3.0")
	if err == nil {
		t.Fatal("expected sha256 mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Errorf("error = %v, want 'sha256 mismatch'", err)
	}
	if fileExists(c.installDir) {
		t.Errorf("installDir %s should not exist after bad-sha rejection", c.installDir)
	}
}

func TestDownloadAndInstall_PreservesOldOnNetworkFail(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	if err := os.MkdirAll(c.installDir, 0o755); err != nil {
		t.Fatalf("seed installDir: %v", err)
	}
	if err := os.WriteFile(c.execFilePath, []byte("OLD"), 0o755); err != nil {
		t.Fatalf("seed exec: %v", err)
	}

	oss := newFakeOSS(t, map[string][]byte{})
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", oss.srv.URL)

	err := c.downloadAndInstall("9.9.9")
	if err == nil {
		t.Fatal("expected download error, got nil")
	}

	data, readErr := os.ReadFile(c.execFilePath)
	if readErr != nil {
		t.Fatalf("old exec missing: %v", readErr)
	}
	if string(data) != "OLD" {
		t.Errorf("old exec contents = %q, want OLD", string(data))
	}
}

// --- extractTarGz -----------------------------------------------------------

func TestExtractTarGz_RejectsTraversalPath(t *testing.T) {
	tar := buildTarGz(t, []tarEntry{
		{Name: "../escape", Body: "x", Mode: 0o644},
	})
	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "evil.tar.gz")
	if err := os.WriteFile(tarPath, tar, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(tmp, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	err := extractTarGz(tarPath, dest)
	if err == nil {
		t.Fatal("expected escape error, got nil")
	}
	if !strings.Contains(err.Error(), "escapes destDir") {
		t.Errorf("error = %v, want 'escapes destDir'", err)
	}
}

func TestExtractTarGz_PreservesSymlink(t *testing.T) {
	tarBytes := buildTarGz(t, []tarEntry{
		{Name: "pkg/", Mode: 0o755},
		{Name: "pkg/real.txt", Body: "hello", Mode: 0o644},
		{Name: "pkg/link.txt", Linkname: "real.txt"},
	})
	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "t.tar.gz")
	if err := os.WriteFile(tarPath, tarBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(tmp, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := extractTarGz(tarPath, dest); err != nil {
		t.Fatalf("extract: %v", err)
	}

	linkPath := filepath.Join(dest, "pkg", "link.txt")
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat link: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link.txt is not a symlink (mode=%v)", fi.Mode())
	}
	body, err := os.ReadFile(linkPath)
	if err != nil {
		t.Fatalf("readfile via symlink: %v", err)
	}
	if string(body) != "hello" {
		t.Errorf("symlink dereferenced = %q, want 'hello'", string(body))
	}
}

func TestExtractTarGz_RejectsSymlinkEscape(t *testing.T) {
	tarBytes := buildTarGz(t, []tarEntry{
		{Name: "pkg/", Mode: 0o755},
		{Name: "pkg/badlink", Linkname: "/etc/passwd"},
	})
	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "evil.tar.gz")
	if err := os.WriteFile(tarPath, tarBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(tmp, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	err := extractTarGz(tarPath, dest)
	if err == nil {
		t.Fatal("expected symlink-escape error, got nil")
	}
	if !strings.Contains(err.Error(), "escapes destDir") {
		t.Errorf("error = %v, want 'escapes destDir'", err)
	}
}

func TestHttpGetToFile_PropagatesNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out")
	err := httpGetToFile(srv.URL+"/anything", dest)
	if err == nil {
		t.Fatal("expected non-2xx error, got nil")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("error = %v, want 'status 500'", err)
	}
}

// --- update -----------------------------------------------------------------

func TestEnsureInstalledAndUpdated_NotInstalled_DownloadsLatest(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	stageTarball(t, c, "0.4.0")
	withGetLatestStub(t, func(*Context) (string, error) { return "0.4.0", nil })

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("EnsureInstalledAndUpdated: %v", err)
	}
	if !fileExists(c.execFilePath) {
		t.Errorf("expected exec at %s after first install", c.execFilePath)
	}
	if got := c.readLocalVersion(); got != "0.4.0" {
		t.Errorf(".version = %q, want 0.4.0", got)
	}
}

func TestEnsureInstalledAndUpdated_FreshCache_Skips(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	if err := os.MkdirAll(c.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.execFilePath, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	c.installed = true
	if err := c.touchCache(); err != nil {
		t.Fatal(err)
	}

	called := 0
	withGetLatestStub(t, func(*Context) (string, error) {
		called++
		return "9.9.9", nil
	})

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("EnsureInstalledAndUpdated: %v", err)
	}
	if called != 0 {
		t.Errorf("getLatestVersionFunc called %d times — fresh cache must skip remote", called)
	}
	if b, _ := os.ReadFile(c.execFilePath); string(b) != "OLD" {
		t.Errorf("old exec was modified: %q", string(b))
	}
}

func TestEnsureInstalledAndUpdated_StaleCache_ChecksRemote(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	if err := os.MkdirAll(c.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.execFilePath, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.versionFilePath, []byte("0.3.0"), 0o644); err != nil {
		t.Fatal(err)
	}
	c.installed = true
	if err := os.MkdirAll(filepath.Dir(c.versionCachePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.versionCachePath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	ancient := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(c.versionCachePath, ancient, ancient); err != nil {
		t.Fatal(err)
	}

	stageTarball(t, c, "0.4.0")
	withGetLatestStub(t, func(*Context) (string, error) { return "0.4.0", nil })

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("EnsureInstalledAndUpdated: %v", err)
	}
	if got := c.readLocalVersion(); got != "0.4.0" {
		t.Errorf("expected upgrade to 0.4.0, got %q", got)
	}
}

func TestEnsureInstalledAndUpdated_RemoteCheckFails_KeepsOldVersion(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	if err := os.MkdirAll(c.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.execFilePath, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	c.installed = true
	if err := os.MkdirAll(filepath.Dir(c.versionCachePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.versionCachePath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	ancient := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(c.versionCachePath, ancient, ancient)

	withGetLatestStub(t, func(*Context) (string, error) {
		return "", errors.New("network down")
	})

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Errorf("expected soft-fail (nil), got %v", err)
	}
	if b, _ := os.ReadFile(c.execFilePath); string(b) != "OLD" {
		t.Errorf("old exec was modified after soft-fail")
	}
	fi, err := os.Stat(c.versionCachePath)
	if err != nil {
		t.Fatalf("cache file missing: %v", err)
	}
	if time.Since(fi.ModTime()) > time.Minute {
		t.Errorf("cache mtime not refreshed; mtime=%v", fi.ModTime())
	}
}

func TestEnsureInstalledAndUpdated_NoUpdateCheckEnv_Skips(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	t.Setenv("ALIBABA_CLOUD_MAXC_NO_UPDATE_CHECK", "1")

	called := 0
	withGetLatestStub(t, func(*Context) (string, error) {
		called++
		return "9.9.9", nil
	})

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("EnsureInstalledAndUpdated: %v", err)
	}
	if called != 0 {
		t.Errorf("NO_UPDATE_CHECK=1 must short-circuit; called=%d", called)
	}
}

func TestEnsureInstalledAndUpdated_ExecPathEnv_SkipsEverything(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	t.Setenv("ALIBABA_CLOUD_MAXC_EXEC_PATH", "/opt/maxc/maxc")

	withGetLatestStub(t, func(*Context) (string, error) {
		t.Fatal("must not be called when EXEC_PATH is set")
		return "", nil
	})

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("EnsureInstalledAndUpdated: %v", err)
	}
}

func TestReadLocalVersion_Trim(t *testing.T) {
	c := withCtx(t, "linux-amd64")
	if err := os.MkdirAll(c.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.versionFilePath, []byte("0.5.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := c.readLocalVersion(); got != "0.5.0" {
		t.Errorf("readLocalVersion = %q, want 0.5.0", got)
	}
	if err := os.Remove(c.versionFilePath); err != nil {
		t.Fatal(err)
	}
	if got := c.readLocalVersion(); got != "" {
		t.Errorf("missing .version = %q, want empty", got)
	}
}

// --- Run end-to-end ---------------------------------------------------------

func TestRun_FullChain_Mocked(t *testing.T) {
	platform := "linux-amd64"
	origGOOS, origGOARCH := runtimeGOOSFunc, runtimeGOARCHFunc
	origCfg := getConfigurePathFunc
	t.Cleanup(func() {
		runtimeGOOSFunc, runtimeGOARCHFunc = origGOOS, origGOARCH
		getConfigurePathFunc = origCfg
	})
	tmp := t.TempDir()
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	getConfigurePathFunc = func() string { return tmp }

	tarBytes := buildTarGz(t, []tarEntry{
		{Name: "maxc/", Mode: 0o755},
		{Name: "maxc/maxc", Body: "#!/bin/sh\necho ok\n", Mode: 0o755},
	})
	sha := sha256Hex(tarBytes)
	mux := http.NewServeMux()
	mux.HandleFunc("/versions/latest", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("0.7.0\n"))
	})
	mux.HandleFunc("/0.7.0/"+platform+"/maxc.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(tarBytes)
	})
	mux.HandleFunc("/0.7.0/"+platform+"/maxc.tar.gz.sha256", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sha + "\n"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv("ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", srv.URL)

	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name:            "test",
			Mode:            config.AK,
			AccessKeyId:     "test-id",
			AccessKeySecret: "test-secret",
			RegionId:        "cn-hangzhou",
		}, nil
	})

	type spawned struct {
		name string
		args []string
		cmd  *exec.Cmd
	}
	var got spawned
	withExecCommandStub(t, func(name string, arg ...string) *exec.Cmd {
		got.name = name
		got.args = append([]string(nil), arg...)
		got.cmd = exec.Command("/bin/sh", "-c", "exit 0")
		return got.cmd
	})

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	originCtx := cli.NewCommandContext(out, errOut)
	c := NewContext(originCtx)

	args := []string{"--profile", "prod", "query", "--sql", "select 1"}
	if err := c.Run(args); err != nil {
		t.Fatalf("Run: %v", err)
	}

	wantExec := filepath.Join(tmp, "maxc", "maxc")
	if _, err := os.Stat(wantExec); err != nil {
		t.Errorf("expected installed binary at %s: %v", wantExec, err)
	}
	if v, _ := os.ReadFile(filepath.Join(tmp, "maxc", ".version")); strings.TrimSpace(string(v)) != "0.7.0" {
		t.Errorf(".version = %q, want 0.7.0", string(v))
	}

	if got.name != wantExec {
		t.Errorf("execCommand name = %q, want %q", got.name, wantExec)
	}
	wantArgs := []string{"query", "--sql", "select 1"}
	if strings.Join(got.args, "|") != strings.Join(wantArgs, "|") {
		t.Errorf("execCommand args = %v, want %v (--profile must be stripped)", got.args, wantArgs)
	}

	envStr := strings.Join(got.cmd.Env, "\n")
	for _, want := range []string{
		"ALIBABA_CLOUD_ACCESS_KEY_ID=test-id",
		"ALIBABA_CLOUD_ACCESS_KEY_SECRET=test-secret",
		"MAXCOMPUTE_REGION=cn-hangzhou",
		"MAXC_CLI_NAME=aliyun maxc",
	} {
		if !strings.Contains(envStr, want) {
			t.Errorf("env missing %q\nfull env:\n%s", want, envStr)
		}
	}
}

func TestRun_UnsupportedPlatform_FailsFast(t *testing.T) {
	origGOOS, origGOARCH := runtimeGOOSFunc, runtimeGOARCHFunc
	origCfg := getConfigurePathFunc
	t.Cleanup(func() {
		runtimeGOOSFunc, runtimeGOARCHFunc = origGOOS, origGOARCH
		getConfigurePathFunc = origCfg
	})
	getConfigurePathFunc = func() string { return t.TempDir() }
	runtimeGOOSFunc = func() string { return "freebsd" }
	runtimeGOARCHFunc = func() string { return "amd64" }

	c := NewContext(cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{}))
	err := c.Run(nil)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Errorf("expected 'not supported' error, got %v", err)
	}
}

// Compile-time sanity: ensure imports are used.
var _ = fmt.Sprintf
var _ = time.Now
