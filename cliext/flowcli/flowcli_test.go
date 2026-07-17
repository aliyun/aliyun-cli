package flowcli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func newOriginCtx() (*cli.Context, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, errOut)
	return ctx, out, errOut
}

func addConfigFlag(ctx *cli.Context, name string, value string) {
	f := &cli.Flag{Name: name, AssignedMode: cli.AssignedOnce, Category: "config"}
	f.SetAssigned(true)
	f.SetValue(value)
	ctx.Flags().Add(f)
}

func prepareConfig(t *testing.T, home string) {
	cfgDir := filepath.Join(home, ".aliyun")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	configJSON := `{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou","language":"en"}]}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestNewContext(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if c == nil {
		t.Fatalf("NewContext returned nil")
	}
	if c.originCtx != ctx {
		t.Errorf("originCtx mismatch")
	}
}

func TestInitBasicInfoUnix(t *testing.T) {
	tmp := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmp }
	defer func() { getConfigurePathFunc = oldGet }()

	oldGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "linux" }
	defer func() { runtimeGOOSFunc = oldGOOS }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.InitBasicInfo(); err != nil {
		t.Fatalf("InitBasicInfo failed: %v", err)
	}

	wantPrefix := filepath.Join(tmp, prefixDirName)
	if c.prefixPath != wantPrefix {
		t.Errorf("prefixPath: want %s got %s", wantPrefix, c.prefixPath)
	}
	wantExec := filepath.Join(wantPrefix, "bin", binName)
	if c.execFilePath != wantExec {
		t.Errorf("execFilePath: want %s got %s", wantExec, c.execFilePath)
	}
	if c.installed {
		t.Errorf("should not be installed initially")
	}
}

func TestInitBasicInfoWindows(t *testing.T) {
	tmp := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmp }
	defer func() { getConfigurePathFunc = oldGet }()

	oldGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	defer func() { runtimeGOOSFunc = oldGOOS }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	_ = c.InitBasicInfo()

	want := filepath.Join(tmp, prefixDirName, binName+".cmd")
	if c.execFilePath != want {
		t.Errorf("execFilePath: want %s got %s", want, c.execFilePath)
	}
}

func TestInitBasicInfoExecPathOverride(t *testing.T) {
	tmp := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmp }
	defer func() { getConfigurePathFunc = oldGet }()

	fake := filepath.Join(tmp, "custom-flow-cli")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake: %v", err)
	}
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_EXEC_PATH", fake)

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	_ = c.InitBasicInfo()
	if c.execFilePath != fake {
		t.Errorf("execFilePath override: want %s got %s", fake, c.execFilePath)
	}
	if !c.installed {
		t.Errorf("should be installed via override")
	}
}

func TestEnsureNodeAvailableEnvOverride(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_NODE_PATH", "/tmp/fake-node")
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.EnsureNodeAvailable(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if c.nodePath != "/tmp/fake-node" {
		t.Errorf("nodePath: %s", c.nodePath)
	}
}

func TestEnsureNodeAvailableMissingHints(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_NODE_PATH", "")
	oldLook := lookPathFunc
	lookPathFunc = func(string) (string, error) { return "", exec.ErrNotFound }
	defer func() { lookPathFunc = oldLook }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	err := c.EnsureNodeAvailable()
	if err == nil {
		t.Fatalf("expected error when node missing")
	}
	msg := err.Error()
	for _, want := range []string{"node >= 18", "brew install node@18", "nodejs.org", "ALIBABA_CLOUD_FLOW_CLI_NODE_PATH"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error msg missing %q: %s", want, msg)
		}
	}
}

func TestEnsureNodeAvailableTooOld(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_NODE_PATH", "")
	oldLook := lookPathFunc
	lookPathFunc = func(string) (string, error) { return "/usr/bin/node", nil }
	defer func() { lookPathFunc = oldLook }()
	oldMaj := getNodeMajorFunc
	getNodeMajorFunc = func(string) (int, error) { return 16, nil }
	defer func() { getNodeMajorFunc = oldMaj }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	err := c.EnsureNodeAvailable()
	if err == nil {
		t.Fatalf("expected error for old node")
	}
	if !strings.Contains(err.Error(), "node version too old") {
		t.Errorf("error: %s", err.Error())
	}
}

func TestEnsureNodeAvailableOK(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_NODE_PATH", "")
	oldLook := lookPathFunc
	lookPathFunc = func(string) (string, error) { return "/usr/local/bin/node", nil }
	defer func() { lookPathFunc = oldLook }()
	oldMaj := getNodeMajorFunc
	getNodeMajorFunc = func(string) (int, error) { return 20, nil }
	defer func() { getNodeMajorFunc = oldMaj }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.EnsureNodeAvailable(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if c.nodePath != "/usr/local/bin/node" {
		t.Errorf("nodePath: %s", c.nodePath)
	}
}

func TestEnsureNpmAvailableEnvOverride(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_NPM_PATH", "/tmp/fake-npm")
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.EnsureNpmAvailable(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if c.npmPath != "/tmp/fake-npm" {
		t.Errorf("npmPath: %s", c.npmPath)
	}
}

func TestEnsureNpmAvailablePrefersNextToNode(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_NPM_PATH", "")
	tmp := t.TempDir()
	nodeBin := filepath.Join(tmp, "node")
	npmBin := filepath.Join(tmp, "npm")
	if err := os.WriteFile(nodeBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write node: %v", err)
	}
	if err := os.WriteFile(npmBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write npm: %v", err)
	}

	oldGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "linux" }
	defer func() { runtimeGOOSFunc = oldGOOS }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.osType = "linux"
	c.nodePath = nodeBin
	if err := c.EnsureNpmAvailable(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if c.npmPath != npmBin {
		t.Errorf("expected npm next to node, got %s", c.npmPath)
	}
}

func TestEnsurePrefixAndPackage_ExecPathOverrideMissing(t *testing.T) {
	tmp := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmp }
	defer func() { getConfigurePathFunc = oldGet }()

	missing := filepath.Join(tmp, "nope", "flow-cli")
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_EXEC_PATH", missing)

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.InitBasicInfo(); err != nil {
		t.Fatalf("InitBasicInfo: %v", err)
	}
	err := c.EnsurePrefixAndPackage()
	if err == nil {
		t.Fatalf("expected error for missing exec path override")
	}
	if !strings.Contains(err.Error(), "ALIBABA_CLOUD_FLOW_CLI_EXEC_PATH") || !strings.Contains(err.Error(), missing) {
		t.Errorf("error should name the env var and path: %v", err)
	}
}

func TestEnsurePrefixAndPackage_ExecPathOverrideExists(t *testing.T) {
	tmp := t.TempDir()
	oldGet := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmp }
	defer func() { getConfigurePathFunc = oldGet }()

	fake := filepath.Join(tmp, "flow-cli")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake: %v", err)
	}
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_EXEC_PATH", fake)

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.InitBasicInfo(); err != nil {
		t.Fatalf("InitBasicInfo: %v", err)
	}
	if err := c.EnsurePrefixAndPackage(); err != nil {
		t.Fatalf("existing override should not error: %v", err)
	}
}

func TestUsingExecPathOverride(t *testing.T) {
	c := &Context{}
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_EXEC_PATH", "")
	if c.usingExecPathOverride() {
		t.Errorf("empty env should not count as override")
	}
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_EXEC_PATH", "  /some/path  ")
	if !c.usingExecPathOverride() {
		t.Errorf("non-empty env should count as override")
	}
}

func TestApplyMainCliFlagsFromArgs_Shorthand(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	ctx.Flags().Add(&cli.Flag{Name: "profile", Shorthand: 'p', AssignedMode: cli.AssignedOnce, Category: "config"})
	ctx.Flags().Add(&cli.Flag{Name: "endpoint", Shorthand: 'e', AssignedMode: cli.AssignedOnce, Category: "config"})

	c := NewContext(ctx)
	c.applyMainCliFlagsFromArgs([]string{"step", "list", "-p", "prod", "-e=https://x", "--name", "n"})

	pf := ctx.Flags().Get("profile")
	if pf == nil || !pf.IsAssigned() {
		t.Fatalf("profile flag should be assigned via -p")
	}
	if v, _ := pf.GetValue(); v != "prod" {
		t.Errorf("profile value: want prod got %q", v)
	}
	ef := ctx.Flags().Get("endpoint")
	if ef == nil || !ef.IsAssigned() {
		t.Fatalf("endpoint flag should be assigned via -e=")
	}
	if v, _ := ef.GetValue(); v != "https://x" {
		t.Errorf("endpoint value: want https://x got %q", v)
	}
}

func TestRemoveFlagsForMainCli(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")
	addConfigFlag(ctx, "profile", "test")

	c := NewContext(ctx)
	args := []string{"step", "list", "--region", "cn-hangzhou", "--profile", "test", "--name", "my-step"}
	out, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("RemoveFlagsForMainCli: %v", err)
	}
	for _, a := range out {
		if a == "--region" || a == "--profile" || a == "cn-hangzhou" || a == "test" {
			t.Errorf("config flag should be stripped: %s — got %v", a, out)
		}
	}
	hasName := false
	hasValue := false
	for i, a := range out {
		if a == "--name" {
			hasName = true
			if i+1 < len(out) && out[i+1] == "my-step" {
				hasValue = true
			}
		}
	}
	if !hasName || !hasValue {
		t.Errorf("--name should be preserved: %v", out)
	}
}

func TestRemoveFlagsForMainCli_InlineValue(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")
	c := NewContext(ctx)
	args := []string{"step", "list", "--region=cn-hangzhou", "--name=x"}
	out, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("RemoveFlagsForMainCli: %v", err)
	}
	for _, a := range out {
		if strings.HasPrefix(a, "--region") {
			t.Errorf("--region=... should be stripped: got %v", out)
		}
	}
	if len(out) != 3 || out[2] != "--name=x" {
		t.Errorf("--name=x should be preserved: got %v", out)
	}
}

func TestPrepareEnv_AKMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	prepareConfig(t, home)

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	envs, err := c.PrepareEnv()
	if err != nil {
		t.Fatalf("PrepareEnv: %v", err)
	}
	want := map[string]string{
		"ALIBABA_CLOUD_ACCESS_KEY_ID":        "ak",
		"ALIBABA_CLOUD_ACCESS_KEY_SECRET":    "sk",
		"ALIBABA_CLOUD_REGION_ID":            "cn-hangzhou",
		"ALIBABA_CLOUD_FLOW_CLI_COMPAT_MODE": "aliyun flow-cli",
	}
	for k, v := range want {
		needle := k + "=" + v
		if !envContains(envs, needle) {
			t.Errorf("env %q missing — got %v", needle, filterAliEnv(envs))
		}
	}
}

func TestPrepareEnv_AIModeUserAgentInjected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	os.Unsetenv("ALIBABA_CLOUD_USER_AGENT")
	prepareConfig(t, home)

	cfgDir := filepath.Join(home, ".aliyun")
	aiJSON := `{"enabled":false,"user_agent":"AlibabaCloud-Agent-Skills/flow-cli-test"}`
	if err := os.WriteFile(filepath.Join(cfgDir, "ai-mode.json"), []byte(aiJSON), 0o600); err != nil {
		t.Fatalf("write ai-mode: %v", err)
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	envs, err := c.PrepareEnv()
	if err != nil {
		t.Fatalf("PrepareEnv: %v", err)
	}
	want := "ALIBABA_CLOUD_USER_AGENT=AlibabaCloud-Agent-Skills/flow-cli-test"
	if !envContains(envs, want) {
		t.Errorf("UA env missing: %v", filterAliEnv(envs))
	}
}

func TestPrepareEnv_AIModeUserAgentDoesNotOverrideExported(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("ALIBABA_CLOUD_USER_AGENT", "UserExported/1")
	prepareConfig(t, home)

	cfgDir := filepath.Join(home, ".aliyun")
	aiJSON := `{"enabled":true,"user_agent":"FromAiMode/2"}`
	if err := os.WriteFile(filepath.Join(cfgDir, "ai-mode.json"), []byte(aiJSON), 0o600); err != nil {
		t.Fatalf("write ai-mode: %v", err)
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	envs, err := c.PrepareEnv()
	if err != nil {
		t.Fatalf("PrepareEnv: %v", err)
	}
	if envContains(envs, "ALIBABA_CLOUD_USER_AGENT=FromAiMode/2") {
		t.Errorf("ai-mode UA should NOT override user-exported: %v", filterAliEnv(envs))
	}
}

func TestNeedCheckVersion(t *testing.T) {
	tmp := t.TempDir()
	cache := filepath.Join(tmp, "cache")

	cases := []struct {
		name        string
		installed   bool
		cacheExists bool
		content     string
		now         int64
		want        bool
	}{
		{"not installed", false, false, "", 0, false},
		{"no cache", true, false, "", 0, true},
		{"invalid cache", true, true, "garbage", 0, true},
		{"expired", true, true, "1000000", 2000000, true},
		{"fresh", true, true, fmt.Sprintf("%d", time.Now().Unix()), time.Now().Unix(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := cache + "-" + tc.name
			c := &Context{installed: tc.installed, checkVersionCacheFilePath: f}
			if tc.cacheExists {
				if err := os.WriteFile(f, []byte(tc.content), 0o644); err != nil {
					t.Fatalf("write cache: %v", err)
				}
			}
			if tc.now > 0 {
				old := timeNowFunc
				timeNowFunc = func() time.Time { return time.Unix(tc.now, 0) }
				defer func() { timeNowFunc = old }()
			}
			if got := c.NeedCheckVersion(); got != tc.want {
				t.Errorf("want %v got %v", tc.want, got)
			}
		})
	}
}

func TestEffectiveBaseURL(t *testing.T) {
	c := &Context{}
	if got := c.effectiveBaseURL(); got != downloadBaseURL {
		t.Errorf("default base: %s", got)
	}
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_DOWNLOAD_BASE_URL", "https://example.com/m")
	if got := c.effectiveBaseURL(); got != "https://example.com/m" {
		t.Errorf("override base: %s", got)
	}
}

func TestEffectiveRegistry(t *testing.T) {
	c := &Context{}
	if got := c.effectiveRegistry(); got != defaultRegistry {
		t.Errorf("default registry: %s", got)
	}
	t.Setenv("ALIBABA_CLOUD_FLOW_CLI_NPM_REGISTRY", "https://registry.npmjs.org")
	if got := c.effectiveRegistry(); got != "https://registry.npmjs.org" {
		t.Errorf("override registry: %s", got)
	}
}

func TestFileExists(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.txt")
	if fileExists(p) {
		t.Errorf("should not exist")
	}
	_ = os.WriteFile(p, []byte("x"), 0o644)
	if !fileExists(p) {
		t.Errorf("should exist")
	}
}

// --- ExitError ---

func TestExitError(t *testing.T) {
	e := &ExitError{Code: 42}
	if e.Error() != "subprocess exited with code 42" {
		t.Errorf("Error() mismatch: %s", e.Error())
	}
	if e.ExitCode() != 42 {
		t.Errorf("ExitCode() mismatch: %d", e.ExitCode())
	}
}

func TestExecuteFlowcli_ExitCode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	prepareConfig(t, home)

	origExec := execCommandFunc
	defer func() { execCommandFunc = origExec }()
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "exit 42")
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.execFilePath = "/any/path"

	err := c.ExecuteFlowcli([]string{"version"})
	if err == nil {
		t.Fatalf("expected error for non-zero exit")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 42 {
		t.Errorf("exit code: got %d, want 42", exitErr.Code)
	}
}

func TestExecuteFlowcli_Success(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	prepareConfig(t, home)

	origExec := execCommandFunc
	defer func() { execCommandFunc = origExec }()
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "exit 0")
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.execFilePath = "/any/path"

	if err := c.ExecuteFlowcli([]string{"version"}); err != nil {
		t.Fatalf("ExecuteFlowcli: %v", err)
	}
}

// --- helpers ---

func envContains(envs []string, needle string) bool {
	for _, e := range envs {
		if e == needle {
			return true
		}
	}
	return false
}

func filterAliEnv(envs []string) []string {
	out := []string{}
	for _, e := range envs {
		if strings.HasPrefix(e, "ALIBABA_CLOUD_") {
			out = append(out, e)
		}
	}
	return out
}
