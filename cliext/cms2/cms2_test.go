package cms2

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

func newTestCtx() (*cli.Context, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, errOut)
	return ctx, out, errOut
}

func writeExec(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write exec: %v", err)
	}
}

func saveAndRestore(t *testing.T) {
	t.Helper()
	origConfig := getConfigurePathFunc
	origGetLatest := getLatestCms2VersionFunc
	origDownload := downloadFileFunc
	origExec := execCommandFunc
	origHTTPGet := httpGetFunc
	origHTTPDo := httpDoFunc
	origTime := timeNowFunc
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	t.Cleanup(func() {
		getConfigurePathFunc = origConfig
		getLatestCms2VersionFunc = origGetLatest
		downloadFileFunc = origDownload
		execCommandFunc = origExec
		httpGetFunc = origHTTPGet
		httpDoFunc = origHTTPDo
		timeNowFunc = origTime
		runtimeGOOSFunc = origGOOS
		runtimeGOARCHFunc = origGOARCH
	})
}

// --- InitBasicInfo ---

func TestInitBasicInfo(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()

	if c.configPath != tmpDir {
		t.Errorf("configPath: got %s, want %s", c.configPath, tmpDir)
	}
	if c.execFilePath != filepath.Join(tmpDir, "aliyuncms") {
		t.Errorf("execFilePath: got %s", c.execFilePath)
	}
	if c.installed {
		t.Errorf("installed should be false")
	}
}

func TestInitBasicInfo_Windows(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "windows" }

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()

	if !strings.HasSuffix(c.execFilePath, ".exe") {
		t.Errorf("execFilePath should end with .exe: %s", c.execFilePath)
	}
}

func TestInitBasicInfo_EnvOverride(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "my-aliyuncms")
	writeExec(t, customPath)
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }
	t.Setenv("ALIYUN_CMS2_EXEC_PATH", customPath)

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()

	if c.execFilePath != customPath {
		t.Errorf("execFilePath: got %s, want %s", c.execFilePath, customPath)
	}
	if !c.installed {
		t.Errorf("installed should be true when env path exists")
	}
}

// --- CheckOsTypeAndArch ---

func TestCheckOsTypeAndArch(t *testing.T) {
	saveAndRestore(t)
	tests := []struct {
		os, arch string
		support  bool
		suffix   string
	}{
		{"linux", "amd64", true, "linux-amd64"},
		{"linux", "arm64", true, "linux-arm64"},
		{"darwin", "amd64", true, "darwin-amd64"},
		{"darwin", "arm64", true, "darwin-arm64"},
		{"windows", "amd64", true, "windows-amd64"},
		{"windows", "arm64", true, "windows-arm64"},
		{"linux", "386", false, ""},
		{"freebsd", "amd64", false, ""},
	}
	for _, tc := range tests {
		runtimeGOOSFunc = func() string { return tc.os }
		runtimeGOARCHFunc = func() string { return tc.arch }
		ctx, _, _ := newTestCtx()
		c := NewContext(ctx)
		c.CheckOsTypeAndArch()
		if c.osSupport != tc.support {
			t.Errorf("%s/%s: support got %v, want %v", tc.os, tc.arch, c.osSupport, tc.support)
		}
		if tc.support && c.downloadPathSuffix != tc.suffix {
			t.Errorf("%s/%s: suffix got %s, want %s", tc.os, tc.arch, c.downloadPathSuffix, tc.suffix)
		}
	}
}

// --- NeedCheckVersion ---

func TestNeedCheckVersion(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }
	fixedNow := time.Unix(1800000000, 0)
	timeNowFunc = func() time.Time { return fixedNow }

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()

	if c.NeedCheckVersion() {
		t.Fatalf("not installed should return false")
	}

	writeExec(t, filepath.Join(tmpDir, "aliyuncms"))
	c.InitBasicInfo()
	if !c.NeedCheckVersion() {
		t.Fatalf("installed no cache => true")
	}

	_ = os.WriteFile(c.checkVersionCacheFilePath, []byte("abc"), 0644)
	if !c.NeedCheckVersion() {
		t.Fatalf("invalid content => true")
	}

	_ = os.WriteFile(c.checkVersionCacheFilePath, []byte(fmt.Sprintf("%d", fixedNow.Unix())), 0644)
	if c.NeedCheckVersion() {
		t.Fatalf("fresh cache => false")
	}

	expired := fixedNow.Unix() - int64(VersionCheckTTL) - 5
	_ = os.WriteFile(c.checkVersionCacheFilePath, []byte(fmt.Sprintf("%d", expired)), 0644)
	if !c.NeedCheckVersion() {
		t.Fatalf("expired => true")
	}
}

// --- EnsureInstalledAndUpdated ---

func TestEnsureInstalledAndUpdated_NotInstalled(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }

	getLatestCms2VersionFunc = func() (string, error) { return "v1.0.0", nil }
	installCalled := false
	downloadFileFunc = func(url, dest string) error {
		installCalled = true
		return os.WriteFile(dest, []byte("binary"), 0755)
	}

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	c.CheckOsTypeAndArch()

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !installCalled {
		t.Fatalf("install should have been called")
	}
	if !c.installed {
		t.Fatalf("installed should be true after install")
	}
}

func TestEnsureInstalledAndUpdated_NotInstalled_VersionError(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }

	getLatestCms2VersionFunc = func() (string, error) { return "", fmt.Errorf("network error") }

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()

	err := c.EnsureInstalledAndUpdated()
	if err == nil || !strings.Contains(err.Error(), "auto-download failed") {
		t.Fatalf("expected auto-download error, got %v", err)
	}
}

func TestEnsureInstalledAndUpdated_Installed_NoUpdate(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }

	writeExec(t, filepath.Join(tmpDir, "aliyuncms"))
	_ = os.WriteFile(filepath.Join(tmpDir, ".cms2_version_check"),
		[]byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)

	getCalls := 0
	getLatestCms2VersionFunc = func() (string, error) { getCalls++; return "v1.0.0", nil }

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if getCalls != 0 {
		t.Fatalf("should not check version within TTL, got %d calls", getCalls)
	}
}

func TestEnsureInstalledAndUpdated_Installed_VersionCheckFails(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }

	writeExec(t, filepath.Join(tmpDir, "aliyuncms"))

	getLatestCms2VersionFunc = func() (string, error) { return "", fmt.Errorf("network error") }

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("should not error when installed and version check fails: %v", err)
	}
}

func TestEnsureInstalledAndUpdated_SkipWhenEnvOverride(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "my-cms")
	writeExec(t, customPath)
	t.Setenv("ALIYUN_CMS2_EXEC_PATH", customPath)
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }

	getCalls := 0
	getLatestCms2VersionFunc = func() (string, error) { getCalls++; return "v1.0.0", nil }

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()

	if err := c.EnsureInstalledAndUpdated(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if getCalls != 0 {
		t.Fatalf("should skip version check when env override set")
	}
}

// --- FilterEnv ---

func TestFilterEnv(t *testing.T) {
	base := []string{
		"HOME=/home/user",
		"ALIYUN_CMS_CLI_ACCESS_KEY_ID=old-ak",
		"PATH=/usr/bin",
		"ALIYUN_CMS_CLI_REGION=cn-hangzhou",
	}
	overrides := map[string]string{
		"ALIYUN_CMS_CLI_ACCESS_KEY_ID": "new-ak",
		"ALIYUN_CMS_CLI_REGION":        "cn-shanghai",
	}
	result := filterEnv(base, overrides)
	for _, item := range result {
		key, _, _ := strings.Cut(item, "=")
		if key == "ALIYUN_CMS_CLI_ACCESS_KEY_ID" || key == "ALIYUN_CMS_CLI_REGION" {
			t.Errorf("conflicting key %s should be filtered", key)
		}
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d: %v", len(result), result)
	}
}

// --- RemoveFlagsForMainCli ---

func TestRemoveFlagsForMainCli(t *testing.T) {
	ctx, _, _ := newTestCtx()
	config.AddFlags(ctx.Flags())

	profileFlag := ctx.Flags().Get("profile")
	if profileFlag == nil {
		t.Fatalf("profile flag not found")
	}
	profileFlag.SetAssigned(true)
	profileFlag.SetValue("test-profile")

	c := NewContext(ctx)
	args := []string{"--profile", "test-profile", "integration-policy", "list", "--region", "cn-hangzhou"}
	result, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("RemoveFlagsForMainCli: %v", err)
	}

	for _, a := range result {
		if a == "--profile" || a == "test-profile" {
			t.Errorf("--profile and its value should be removed, got %v", result)
		}
	}

	if !contains(result, "--region") || !contains(result, "cn-hangzhou") {
		t.Errorf("--region should be preserved: %v", result)
	}
	if !contains(result, "integration-policy") || !contains(result, "list") {
		t.Errorf("subcommand args should be preserved: %v", result)
	}
}

func TestRemoveFlagsForMainCli_AssignedWithEquals(t *testing.T) {
	ctx, _, _ := newTestCtx()
	config.AddFlags(ctx.Flags())

	profileFlag := ctx.Flags().Get("profile")
	if profileFlag == nil {
		t.Fatalf("profile flag not found")
	}
	profileFlag.SetAssigned(true)
	profileFlag.SetValue("test-profile")

	endpointFlag := ctx.Flags().Get("endpoint")
	if endpointFlag == nil {
		t.Fatalf("endpoint flag not found")
	}
	endpointFlag.SetAssigned(true)
	endpointFlag.SetValue("https://cms.example.com")

	c := NewContext(ctx)
	args := []string{
		"-p:test-profile",
		"integration-policy",
		"list",
		"--endpoint:https://cms.example.com",
		"--region=cn-hangzhou",
	}
	result, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("RemoveFlagsForMainCli: %v", err)
	}

	for _, a := range result {
		if strings.HasPrefix(a, "--profile") || strings.HasPrefix(a, "-p") || strings.HasPrefix(a, "--endpoint") {
			t.Fatalf("config flags with inline values should be removed, got %v", result)
		}
	}
	if !contains(result, "integration-policy") || !contains(result, "list") || !contains(result, "--region=cn-hangzhou") {
		t.Fatalf("downstream args should be preserved: %v", result)
	}
}

func TestRemoveFlagsForMainCli_StripsCliAIModeFlags(t *testing.T) {
	ctx, _, _ := newTestCtx()
	config.AddFlags(ctx.Flags())
	openapi.AddFlags(ctx.Flags())

	aiFlag := ctx.Flags().Get(openapi.CliAIModeFlagName)
	if aiFlag == nil {
		t.Fatalf("cli-ai-mode flag not found")
	}
	aiFlag.SetAssigned(true)

	noAIFlag := ctx.Flags().Get(openapi.CliNoAIModeFlagName)
	if noAIFlag == nil {
		t.Fatalf("no-cli-ai-mode flag not found")
	}
	noAIFlag.SetAssigned(true)

	c := NewContext(ctx)
	args := []string{
		"--" + openapi.CliAIModeFlagName,
		"--" + openapi.CliNoAIModeFlagName,
		"version",
	}
	result, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("RemoveFlagsForMainCli: %v", err)
	}

	if contains(result, "--"+openapi.CliAIModeFlagName) || contains(result, "--"+openapi.CliNoAIModeFlagName) {
		t.Fatalf("wrapper-only AI flags should be removed, got %v", result)
	}
	if !contains(result, "version") {
		t.Fatalf("downstream args should be preserved: %v", result)
	}
}

func TestRemoveFlagsForMainCli_NilFlags(t *testing.T) {
	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	args := []string{"version"}
	result, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(result) != 1 || result[0] != "version" {
		t.Errorf("args should pass through: %v", result)
	}
}

// --- GetLatestCms2Version ---

func TestGetLatestCms2Version_Success(t *testing.T) {
	saveAndRestore(t)
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("User-Agent") == "" {
			t.Fatalf("User-Agent should be set")
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("  v1.2.3\n")),
		}, nil
	}

	ver, err := GetLatestCms2Version()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ver != "v1.2.3" {
		t.Errorf("version mismatch: got %s, want v1.2.3", ver)
	}
}

func TestGetLatestCms2Version_NetworkError(t *testing.T) {
	saveAndRestore(t)
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network down")
	}
	_, err := GetLatestCms2Version()
	if err == nil || !strings.Contains(err.Error(), "network down") {
		t.Fatalf("expected network error, got %v", err)
	}
}

func TestGetLatestCms2Version_Non200(t *testing.T) {
	saveAndRestore(t)
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found"))}, nil
	}
	_, err := GetLatestCms2Version()
	if err == nil || !strings.Contains(err.Error(), "status code 404") {
		t.Fatalf("expected 404 error, got %v", err)
	}
}

func TestGetLatestCms2Version_Empty(t *testing.T) {
	saveAndRestore(t)
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("  \n")),
		}, nil
	}
	_, err := GetLatestCms2Version()
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty error, got %v", err)
	}
}

// --- GetLocalVersion / SaveLocalVersion ---

func TestGetLocalVersion(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()

	if err := c.GetLocalVersion(); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("expected not installed error, got %v", err)
	}

	writeExec(t, c.execFilePath)
	c.installed = true
	_ = os.WriteFile(c.versionFilePath, []byte("v1.0.0\n"), 0644)

	if err := c.GetLocalVersion(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if c.versionLocal != "v1.0.0" {
		t.Errorf("versionLocal: got %s, want v1.0.0", c.versionLocal)
	}
}

// --- Execute ---

func TestExecute_Success(t *testing.T) {
	saveAndRestore(t)
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "exit 0")
	}

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.execFilePath = "/any/path"
	c.envMap = map[string]string{"ALIYUN_CMS_CLI_ACCESS_KEY_ID": "ak"}

	if err := c.Execute([]string{"version"}); err != nil {
		t.Fatalf("Execute should succeed: %v", err)
	}
}

func TestExecute_ExitCode(t *testing.T) {
	saveAndRestore(t)
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "exit 42")
	}

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.execFilePath = "/any/path"
	c.envMap = map[string]string{}

	err := c.Execute([]string{"version"})
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

func TestExecute_EnvNoConflict(t *testing.T) {
	saveAndRestore(t)

	t.Setenv("ALIYUN_CMS_CLI_ACCESS_KEY_ID", "old-ak")

	var capturedEnv []string
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command("bash", "-c", "exit 0")
		return cmd
	}

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	c.execFilePath = "/any/path"
	c.envMap = map[string]string{
		"ALIYUN_CMS_CLI_ACCESS_KEY_ID": "new-ak",
	}

	envs := filterEnv(os.Environ(), c.envMap)
	for k, v := range c.envMap {
		envs = append(envs, k+"="+v)
	}
	capturedEnv = envs

	akCount := 0
	for _, item := range capturedEnv {
		if strings.HasPrefix(item, "ALIYUN_CMS_CLI_ACCESS_KEY_ID=") {
			akCount++
			if !strings.Contains(item, "new-ak") {
				t.Errorf("expected new-ak, got %s", item)
			}
		}
	}
	if akCount != 1 {
		t.Errorf("ALIYUN_CMS_CLI_ACCESS_KEY_ID should appear exactly once, got %d", akCount)
	}
}

// --- Run integration ---

func TestRun_FullFlow(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }

	writeExec(t, filepath.Join(tmpDir, "aliyuncms"))
	_ = os.WriteFile(filepath.Join(tmpDir, ".cms2_version_check"),
		[]byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)

	configPath := filepath.Join(t.TempDir(), "config.json")
	_ = os.WriteFile(configPath, []byte(`{
		"current":"default",
		"profiles":[{"name":"default","mode":"AK","access_key_id":"test-ak","access_key_secret":"test-sk","region_id":"cn-hangzhou"}]
	}`), 0644)

	ctx, _, _ := newTestCtx()
	config.AddFlags(ctx.Flags())
	cpFlag := ctx.Flags().Get(config.ConfigurePathFlagName)
	cpFlag.SetAssigned(true)
	cpFlag.SetValue(configPath)

	var capturedArgs []string
	var capturedEnv []string
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		capturedArgs = args
		cmd := exec.Command("bash", "-c", "exit 0")
		return cmd
	}

	origExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		capturedArgs = args
		cmd := origExec(name, args...)
		return cmd
	}

	c := NewContext(ctx)
	if err := c.Run([]string{"integration-policy", "list"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	_ = capturedArgs
	_ = capturedEnv

	if c.envMap["ALIYUN_CMS_CLI_ACCESS_KEY_ID"] != "test-ak" {
		t.Errorf("access key mismatch: %s", c.envMap["ALIYUN_CMS_CLI_ACCESS_KEY_ID"])
	}
	if c.envMap["ALIYUN_CMS_CLI_ACCESS_KEY_SECRET"] != "test-sk" {
		t.Errorf("secret key mismatch: %s", c.envMap["ALIYUN_CMS_CLI_ACCESS_KEY_SECRET"])
	}
	if c.envMap["ALIYUN_CMS_CLI_REGION"] != "cn-hangzhou" {
		t.Errorf("region mismatch: %s", c.envMap["ALIYUN_CMS_CLI_REGION"])
	}
}

func TestRun_UnsupportedPlatform(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "plan9" }
	runtimeGOARCHFunc = func() string { return "amd64" }

	ctx, _, _ := newTestCtx()
	c := NewContext(ctx)
	err := c.Run([]string{"version"})
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected unsupported error, got %v", err)
	}
}

func TestRun_NotInstalled_Warning(t *testing.T) {
	saveAndRestore(t)
	tmpDir := t.TempDir()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }

	getLatestCms2VersionFunc = func() (string, error) { return "", fmt.Errorf("network error") }

	ctx, _, errOut := newTestCtx()
	c := NewContext(ctx)
	err := c.Run([]string{"version"})
	if err == nil {
		t.Fatalf("expected error when not installed and download fails")
	}
	_ = errOut
}

// --- flagStringValue ---

func TestFlagStringValue(t *testing.T) {
	if got := flagStringValue(nil, "region"); got != "" {
		t.Fatalf("nil ctx should return empty, got %s", got)
	}

	ctx, _, _ := newTestCtx()
	config.AddFlags(ctx.Flags())
	regionFlag := ctx.Flags().Get("region")
	if regionFlag == nil {
		t.Fatalf("region flag not found")
	}
	regionFlag.SetAssigned(true)
	regionFlag.SetValue(" cn-hangzhou ")
	if got := flagStringValue(ctx, "region"); got != "cn-hangzhou" {
		t.Fatalf("expected cn-hangzhou, got %s", got)
	}

	if got := flagStringValue(ctx, "nonexistent"); got != "" {
		t.Fatalf("missing flag should return empty, got %s", got)
	}
}

// --- fileExists ---

func TestFileExists(t *testing.T) {
	f := filepath.Join(t.TempDir(), "x.txt")
	if fileExists(f) {
		t.Fatalf("should false before create")
	}
	_ = os.WriteFile(f, []byte("1"), 0644)
	if !fileExists(f) {
		t.Fatalf("should true after create")
	}
}

// --- ExitError ---

func TestExitError(t *testing.T) {
	e := &ExitError{Code: 7}
	if e.Error() != "subprocess exited with code 7" {
		t.Errorf("Error() mismatch: %s", e.Error())
	}
	if e.ExitCode() != 7 {
		t.Errorf("ExitCode() mismatch: %d", e.ExitCode())
	}
}

// --- downloadFile ---

func TestDownloadFile_ErrorPaths(t *testing.T) {
	saveAndRestore(t)

	httpGetFunc = func(url string) (*http.Response, error) {
		return nil, fmt.Errorf("http get failed")
	}
	if err := downloadFile("http://x", filepath.Join(t.TempDir(), "f")); err == nil || !strings.Contains(err.Error(), "http get failed") {
		t.Fatalf("expected http error, got %v", err)
	}

	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("err"))}, nil
	}
	if err := downloadFile("http://x", filepath.Join(t.TempDir(), "f")); err == nil || !strings.Contains(err.Error(), "status code 500") {
		t.Fatalf("expected status code error, got %v", err)
	}

	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("data"))}, nil
	}
	if err := downloadFile("http://x", filepath.Join(t.TempDir(), "missing", "f")); err == nil || !strings.Contains(err.Error(), "failed to create file") {
		t.Fatalf("expected create file error, got %v", err)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
