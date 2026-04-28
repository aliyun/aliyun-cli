package agentbay

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write exec failed: %v", err)
	}
}

func newOriginCtx() (*cli.Context, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, errOut)
	return ctx, out, errOut
}

func newAgentBayConfigCtx() *cli.Context {
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	return ctx
}

func writeAgentBayConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func assignFlag(t *testing.T, ctx *cli.Context, name string, value string) {
	t.Helper()
	flag := ctx.Flags().Get(name)
	if flag == nil {
		t.Fatalf("flag %s not found", name)
	}
	flag.SetAssigned(true)
	flag.SetValue(value)
}

func TestLoadAgentBayProfileUsesConfigPathAndProfileFlag(t *testing.T) {
	configPath := writeAgentBayConfig(t, `{
		"current":"default",
		"profiles":[
			{"name":"default","mode":"AK","access_key_id":"default-ak","access_key_secret":"default-sk","region_id":"cn-hangzhou"},
			{"name":"target","mode":"StsToken","access_key_id":"target-ak","access_key_secret":"target-sk","sts_token":"target-token","region_id":"ap-southeast-1"}
		]
	}`)
	ctx := newAgentBayConfigCtx()
	assignFlag(t, ctx, config.ConfigurePathFlagName, configPath)
	assignFlag(t, ctx, config.ProfileFlagName, "target")

	profile, err := loadAgentBayProfile(ctx)
	if err != nil {
		t.Fatalf("loadAgentBayProfile: %v", err)
	}
	if profile.Name != "target" {
		t.Fatalf("profile name mismatch: %s", profile.Name)
	}
	if profile.AccessKeyId != "target-ak" || profile.AccessKeySecret != "target-sk" || profile.StsToken != "target-token" {
		t.Fatalf("profile credentials mismatch: %#v", profile)
	}
	if profile.RegionId != "ap-southeast-1" {
		t.Fatalf("profile region mismatch: %s", profile.RegionId)
	}
}

func TestLoadAgentBayProfileUsesProfileFromEnv(t *testing.T) {
	configPath := writeAgentBayConfig(t, `{
		"current":"default",
		"profiles":[
			{"name":"default","mode":"AK","access_key_id":"default-ak","access_key_secret":"default-sk","region_id":"cn-hangzhou"},
			{"name":"env-profile","mode":"AK","access_key_id":"env-ak","access_key_secret":"env-sk","region_id":"cn-shanghai"}
		]
	}`)
	t.Setenv("ALIBABA_CLOUD_PROFILE", "env-profile")
	ctx := newAgentBayConfigCtx()
	assignFlag(t, ctx, config.ConfigurePathFlagName, configPath)

	profile, err := loadAgentBayProfile(ctx)
	if err != nil {
		t.Fatalf("loadAgentBayProfile: %v", err)
	}
	if profile.Name != "env-profile" {
		t.Fatalf("profile name mismatch: %s", profile.Name)
	}
	if profile.AccessKeyId != "env-ak" || profile.RegionId != "cn-shanghai" {
		t.Fatalf("env profile mismatch: %#v", profile)
	}
}

func TestLoadAgentBayProfileOverwritesWithFlagsInConfigureMode(t *testing.T) {
	configPath := writeAgentBayConfig(t, `{
		"current":"default",
		"profiles":[
			{"name":"default","mode":"AK","access_key_id":"old-ak","access_key_secret":"old-sk","region_id":"cn-hangzhou"}
		]
	}`)
	ctx := newAgentBayConfigCtx()
	ctx.SetInConfigureMode(true)
	assignFlag(t, ctx, config.ConfigurePathFlagName, configPath)
	assignFlag(t, ctx, config.AccessKeyIdFlagName, "flag-ak")
	assignFlag(t, ctx, config.AccessKeySecretFlagName, "flag-sk")
	assignFlag(t, ctx, config.RegionFlagName, "ap-southeast-1")

	profile, err := loadAgentBayProfile(ctx)
	if err != nil {
		t.Fatalf("loadAgentBayProfile: %v", err)
	}
	if profile.AccessKeyId != "flag-ak" {
		t.Fatalf("access key should be overwritten: %s", profile.AccessKeyId)
	}
	if profile.AccessKeySecret != "flag-sk" {
		t.Fatalf("secret should be overwritten: %s", profile.AccessKeySecret)
	}
	if profile.RegionId != "ap-southeast-1" {
		t.Fatalf("region should be overwritten: %s", profile.RegionId)
	}
}

func TestLoadAgentBayProfileReturnsErrorForUnknownProfile(t *testing.T) {
	configPath := writeAgentBayConfig(t, `{
		"current":"default",
		"profiles":[
			{"name":"default","mode":"AK","access_key_id":"default-ak","access_key_secret":"default-sk","region_id":"cn-hangzhou"}
		]
	}`)
	ctx := newAgentBayConfigCtx()
	assignFlag(t, ctx, config.ConfigurePathFlagName, configPath)
	assignFlag(t, ctx, config.ProfileFlagName, "missing")

	_, err := loadAgentBayProfile(ctx)
	if err == nil || !strings.Contains(err.Error(), "unknown profile missing") {
		t.Fatalf("expected unknown profile error, got %v", err)
	}
}

func TestRun_NotInstalled_FreshInstallAndExecute(t *testing.T) {
	tmpDir := t.TempDir()
	origConfig := getConfigurePathFunc
	origGetLatest := getLatestAgentBayVersionFunc
	origDownload := downloadAndInstallFunc
	origLoadProfile := loadAgentBayProfileFunc
	origTimeNow := timeNowFunc
	origExec := execCommandFunc
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	t.Cleanup(func() {
		getConfigurePathFunc = origConfig
		getLatestAgentBayVersionFunc = origGetLatest
		downloadAndInstallFunc = origDownload
		loadAgentBayProfileFunc = origLoadProfile
		timeNowFunc = origTimeNow
		execCommandFunc = origExec
		runtimeGOOSFunc = origGOOS
		runtimeGOARCHFunc = origGOARCH
	})

	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	loadAgentBayProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.AK, AccessKeyId: "ak", AccessKeySecret: "sk"}, nil
	}
	getLatestAgentBayVersionFunc = func() (string, error) { return "1.2.3", nil }
	fixedNow := time.Unix(1700000000, 0)
	timeNowFunc = func() time.Time { return fixedNow }
	installCount := 0
	downloadAndInstallFunc = func(url, dest, exe string) error {
		installCount++
		if !strings.Contains(url, "/1.2.3/agentbay-1.2.3-linux-amd64.tar.gz") {
			t.Fatalf("unexpected url %s", url)
		}
		writeExecutable(t, exe, "#!/bin/sh\n")
		return nil
	}
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		if name != filepath.Join(tmpDir, "agentbay") {
			t.Fatalf("unexpected exec path %s", name)
		}
		if strings.Join(args, " ") != "agentbay image list" {
			t.Fatalf("unexpected args %v", args)
		}
		return exec.Command("bash", "-c", "exit 0")
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"agentbay", "image", "list"}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if installCount != 1 {
		t.Fatalf("expected install once, got %d", installCount)
	}
	data, err := os.ReadFile(filepath.Join(tmpDir, ".agentbay_version_check"))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	if strings.TrimSpace(string(data)) != fmt.Sprintf("%d", fixedNow.Unix()) {
		t.Fatalf("cache timestamp mismatch: %s", string(data))
	}
}

func TestRun_Installed_NoVersionCheckWithinTTL(t *testing.T) {
	tmpDir := t.TempDir()
	origConfig := getConfigurePathFunc
	origGetLatest := getLatestAgentBayVersionFunc
	origDownload := downloadAndInstallFunc
	origLoadProfile := loadAgentBayProfileFunc
	origExec := execCommandFunc
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	t.Cleanup(func() {
		getConfigurePathFunc = origConfig
		getLatestAgentBayVersionFunc = origGetLatest
		downloadAndInstallFunc = origDownload
		loadAgentBayProfileFunc = origLoadProfile
		execCommandFunc = origExec
		runtimeGOOSFunc = origGOOS
		runtimeGOARCHFunc = origGOARCH
	})

	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	loadAgentBayProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.AK, AccessKeyId: "ak", AccessKeySecret: "sk"}, nil
	}
	writeExecutable(t, filepath.Join(tmpDir, "agentbay"), "#!/bin/sh\n")
	_ = os.WriteFile(filepath.Join(tmpDir, ".agentbay_version_check"), []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)
	getCalls := 0
	getLatestAgentBayVersionFunc = func() (string, error) { getCalls++; return "1.2.3", nil }
	installCount := 0
	downloadAndInstallFunc = func(url, dest, exe string) error { installCount++; return nil }
	execCommandFunc = func(name string, args ...string) *exec.Cmd { return exec.Command("bash", "-c", "exit 0") }

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"agentbay", "image", "list"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if getCalls != 0 {
		t.Fatalf("expected no version check, got %d", getCalls)
	}
	if installCount != 0 {
		t.Fatalf("expected not install, got %d", installCount)
	}
}

func TestRun_Installed_UpdateWhenExpired(t *testing.T) {
	tmpDir := t.TempDir()
	origConfig := getConfigurePathFunc
	origGetLatest := getLatestAgentBayVersionFunc
	origDownload := downloadAndInstallFunc
	origLoadProfile := loadAgentBayProfileFunc
	origExec := execCommandFunc
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	t.Cleanup(func() {
		getConfigurePathFunc = origConfig
		getLatestAgentBayVersionFunc = origGetLatest
		downloadAndInstallFunc = origDownload
		loadAgentBayProfileFunc = origLoadProfile
		execCommandFunc = origExec
		runtimeGOOSFunc = origGOOS
		runtimeGOARCHFunc = origGOARCH
	})

	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "darwin" }
	runtimeGOARCHFunc = func() string { return "arm64" }
	loadAgentBayProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.AK, AccessKeyId: "ak", AccessKeySecret: "sk"}, nil
	}
	writeExecutable(t, filepath.Join(tmpDir, "agentbay"), "#!/bin/sh\n")
	old := time.Now().Unix() - int64(VersionCheckTTL) - 10
	_ = os.WriteFile(filepath.Join(tmpDir, ".agentbay_version_check"), []byte(fmt.Sprintf("%d", old)), 0644)
	getLatestAgentBayVersionFunc = func() (string, error) { return "1.0.1", nil }
	installCount := 0
	downloadAndInstallFunc = func(url, dest, exe string) error {
		installCount++
		if !strings.Contains(url, "/1.0.1/agentbay-1.0.1-darwin-arm64.tar.gz") {
			t.Fatalf("unexpected url %s", url)
		}
		return nil
	}
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		if len(args) == 1 && args[0] == "version" {
			return exec.Command("bash", "-c", "printf 'AgentBay CLI version 0.9.0\\nGit commit: x\\n'")
		}
		return exec.Command("bash", "-c", "exit 0")
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.Run([]string{"agentbay", "image", "list"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if installCount != 1 {
		t.Fatalf("expected install triggered, got %d", installCount)
	}
}

func TestBuildExecEnvPassesAgentBayCredentialsFromProfile(t *testing.T) {
	origLoadProfile := loadAgentBayProfileFunc
	t.Cleanup(func() { loadAgentBayProfileFunc = origLoadProfile })
	t.Setenv("AGENTBAY_ACCESS_KEY_ID", "old-ak")
	t.Setenv("AGENTBAY_ACCESS_KEY_SECRET", "old-sk")
	t.Setenv("AGENTBAY_ACCESS_KEY_SESSION_TOKEN", "old-token")
	t.Setenv("ALIBABA_CLOUD_REGION_ID", "cn-hangzhou")
	t.Setenv("ALIBABA_CLOUD_USER_AGENT", "Custom/1")
	t.Setenv("ALIBABA_CLOUD_CLI_AI_USER_AGENT", "AI/1")

	loadAgentBayProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		return config.Profile{
			Mode:            config.StsToken,
			AccessKeyId:     "profile-ak",
			AccessKeySecret: "profile-sk",
			StsToken:        "profile-token",
			RegionId:        "cn-shanghai",
		}, nil
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	env, err := c.BuildExecEnv(agentBayCLIOptions{})
	if err != nil {
		t.Fatalf("BuildExecEnv: %v", err)
	}
	envMap := envSliceToMap(env)
	if envMap["AGENTBAY_ACCESS_KEY_ID"] != "profile-ak" {
		t.Fatalf("AGENTBAY_ACCESS_KEY_ID mismatch: %s", envMap["AGENTBAY_ACCESS_KEY_ID"])
	}
	if envMap["AGENTBAY_ACCESS_KEY_SECRET"] != "profile-sk" {
		t.Fatalf("AGENTBAY_ACCESS_KEY_SECRET mismatch: %s", envMap["AGENTBAY_ACCESS_KEY_SECRET"])
	}
	if envMap["AGENTBAY_ACCESS_KEY_SESSION_TOKEN"] != "profile-token" {
		t.Fatalf("AGENTBAY_ACCESS_KEY_SESSION_TOKEN mismatch: %s", envMap["AGENTBAY_ACCESS_KEY_SESSION_TOKEN"])
	}
	if _, ok := envMap["ALIBABA_CLOUD_REGION_ID"]; ok {
		t.Fatalf("region should not be passed to agentbay")
	}
	if _, ok := envMap["ALIBABA_CLOUD_USER_AGENT"]; ok {
		t.Fatalf("custom user-agent should not be passed to agentbay")
	}
	if _, ok := envMap["ALIBABA_CLOUD_CLI_AI_USER_AGENT"]; ok {
		t.Fatalf("AI user-agent should not be passed to agentbay")
	}
	if envMap["AGENTBAY_CLI_ENDPOINT"] != "xiaoying.cn-shanghai.aliyuncs.com" {
		t.Fatalf("endpoint should be derived from profile region: %s", envMap["AGENTBAY_CLI_ENDPOINT"])
	}
}

func TestBuildExecEnvRemovesStaleAgentBaySessionTokenForAKProfile(t *testing.T) {
	origLoadProfile := loadAgentBayProfileFunc
	t.Cleanup(func() { loadAgentBayProfileFunc = origLoadProfile })
	t.Setenv("AGENTBAY_ACCESS_KEY_SESSION_TOKEN", "old-token")

	loadAgentBayProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.AK, AccessKeyId: "profile-ak", AccessKeySecret: "profile-sk"}, nil
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	env, err := c.BuildExecEnv(agentBayCLIOptions{})
	if err != nil {
		t.Fatalf("BuildExecEnv: %v", err)
	}
	envMap := envSliceToMap(env)
	if _, ok := envMap["AGENTBAY_ACCESS_KEY_SESSION_TOKEN"]; ok {
		t.Fatalf("stale AGENTBAY_ACCESS_KEY_SESSION_TOKEN should be removed for AK profile")
	}
}

func TestBuildExecEnvPassesOAuthProfileCredentials(t *testing.T) {
	origLoadProfile := loadAgentBayProfileFunc
	t.Cleanup(func() { loadAgentBayProfileFunc = origLoadProfile })

	loadAgentBayProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		return config.Profile{
			Mode:            config.OAuth,
			AccessKeyId:     "oauth-ak",
			AccessKeySecret: "oauth-sk",
			StsToken:        "oauth-token",
			StsExpiration:   time.Now().Unix() + 3600,
			RegionId:        "ap-southeast-1",
		}, nil
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	env, err := c.BuildExecEnv(agentBayCLIOptions{})
	if err != nil {
		t.Fatalf("BuildExecEnv: %v", err)
	}
	envMap := envSliceToMap(env)
	if envMap["AGENTBAY_ACCESS_KEY_ID"] != "oauth-ak" {
		t.Fatalf("oauth ak mismatch: %s", envMap["AGENTBAY_ACCESS_KEY_ID"])
	}
	if envMap["AGENTBAY_ACCESS_KEY_SECRET"] != "oauth-sk" {
		t.Fatalf("oauth sk mismatch: %s", envMap["AGENTBAY_ACCESS_KEY_SECRET"])
	}
	if envMap["AGENTBAY_ACCESS_KEY_SESSION_TOKEN"] != "oauth-token" {
		t.Fatalf("oauth token mismatch: %s", envMap["AGENTBAY_ACCESS_KEY_SESSION_TOKEN"])
	}
	if envMap["AGENTBAY_CLI_ENDPOINT"] != "xiaoying.ap-southeast-1.aliyuncs.com" {
		t.Fatalf("oauth endpoint mismatch: %s", envMap["AGENTBAY_CLI_ENDPOINT"])
	}
}

func TestBuildExecEnvEndpointOptionOverridesRegion(t *testing.T) {
	origLoadProfile := loadAgentBayProfileFunc
	t.Cleanup(func() { loadAgentBayProfileFunc = origLoadProfile })
	t.Setenv("AGENTBAY_CLI_ENDPOINT", "old.endpoint")

	loadAgentBayProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		return config.Profile{
			Mode:            config.AK,
			AccessKeyId:     "profile-ak",
			AccessKeySecret: "profile-sk",
			RegionId:        "cn-hangzhou",
		}, nil
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	env, err := c.BuildExecEnv(agentBayCLIOptions{endpoint: "custom.endpoint", region: "ap-southeast-1"})
	if err != nil {
		t.Fatalf("BuildExecEnv: %v", err)
	}
	envMap := envSliceToMap(env)
	if envMap["AGENTBAY_CLI_ENDPOINT"] != "custom.endpoint" {
		t.Fatalf("endpoint flag should win: %s", envMap["AGENTBAY_CLI_ENDPOINT"])
	}
}

func TestBuildExecEnvRegionOptionOverridesProfileRegion(t *testing.T) {
	origLoadProfile := loadAgentBayProfileFunc
	t.Cleanup(func() { loadAgentBayProfileFunc = origLoadProfile })

	loadAgentBayProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		return config.Profile{
			Mode:            config.AK,
			AccessKeyId:     "profile-ak",
			AccessKeySecret: "profile-sk",
			RegionId:        "cn-hangzhou",
		}, nil
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	env, err := c.BuildExecEnv(agentBayCLIOptions{region: "ap-southeast-1"})
	if err != nil {
		t.Fatalf("BuildExecEnv: %v", err)
	}
	envMap := envSliceToMap(env)
	if envMap["AGENTBAY_CLI_ENDPOINT"] != "xiaoying.ap-southeast-1.aliyuncs.com" {
		t.Fatalf("region endpoint mismatch: %s", envMap["AGENTBAY_CLI_ENDPOINT"])
	}
}

func TestBuildExecEnvErrorPaths(t *testing.T) {
	origLoadProfile := loadAgentBayProfileFunc
	t.Cleanup(func() { loadAgentBayProfileFunc = origLoadProfile })

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	loadAgentBayProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		return config.Profile{}, fmt.Errorf("load failed")
	}
	if _, err := c.BuildExecEnv(agentBayCLIOptions{}); err == nil || !strings.Contains(err.Error(), "load failed") {
		t.Fatalf("expected load error, got %v", err)
	}

	loadAgentBayProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.AK, AccessKeyId: "ak"}, nil
	}
	if _, err := c.BuildExecEnv(agentBayCLIOptions{}); err == nil || !strings.Contains(err.Error(), "AccessKeyId/AccessKeySecret") {
		t.Fatalf("expected credential error, got %v", err)
	}
}

func TestExtractAgentBayCLIOptionsRemovesCLIEndpointAndRegionFlags(t *testing.T) {
	args, options := extractAgentBayCLIOptions([]string{
		"session", "create",
		"--region", "ap-southeast-1",
		"--endpoint=custom.endpoint",
		"-e", "last.endpoint",
		"--name", "demo",
	})
	if strings.Join(args, " ") != "session create --name demo" {
		t.Fatalf("filtered args mismatch: %v", args)
	}
	if options.region != "ap-southeast-1" {
		t.Fatalf("region mismatch: %s", options.region)
	}
	if options.endpoint != "last.endpoint" {
		t.Fatalf("endpoint mismatch: %s", options.endpoint)
	}
}

func TestFlagStringValue(t *testing.T) {
	if got := flagStringValue(nil, config.RegionFlagName); got != "" {
		t.Fatalf("nil ctx should return empty, got %s", got)
	}

	ctx := newAgentBayConfigCtx()
	assignFlag(t, ctx, config.RegionFlagName, " cn-hangzhou ")
	if got := flagStringValue(ctx, config.RegionFlagName); got != "cn-hangzhou" {
		t.Fatalf("assigned flag mismatch: %s", got)
	}
	if got := flagStringValue(ctx, "missing"); got != "" {
		t.Fatalf("missing flag should return empty, got %s", got)
	}

	unknownFlags := cli.NewFlagSet()
	regionFlag, err := unknownFlags.AddByName(config.RegionFlagName)
	if err != nil {
		t.Fatalf("add unknown flag: %v", err)
	}
	regionFlag.SetAssigned(true)
	regionFlag.SetValue(" ap-southeast-1 ")
	ctx2, _, _ := newOriginCtx()
	ctx2.SetUnknownFlags(unknownFlags)
	if got := flagStringValue(ctx2, config.RegionFlagName); got != "ap-southeast-1" {
		t.Fatalf("unknown flag mismatch: %s", got)
	}
}

func envSliceToMap(env []string) map[string]string {
	result := map[string]string{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			result[key] = value
		}
	}
	return result
}

func TestNeedCheckVersionVariants(t *testing.T) {
	tmpDir := t.TempDir()
	origConfig := getConfigurePathFunc
	origNow := timeNowFunc
	t.Cleanup(func() {
		getConfigurePathFunc = origConfig
		timeNowFunc = origNow
	})
	getConfigurePathFunc = func() string { return tmpDir }
	fixedNow := time.Unix(1800000000, 0)
	timeNowFunc = func() time.Time { return fixedNow }

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.InitBasicInfo()
	if c.NeedCheckVersion() {
		t.Fatalf("not installed should return false")
	}
	writeExecutable(t, filepath.Join(tmpDir, "agentbay"), "#!/bin/sh\n")
	c.InitBasicInfo()
	if !c.NeedCheckVersion() {
		t.Fatalf("installed no cache => true")
	}
	_ = os.WriteFile(filepath.Join(tmpDir, ".agentbay_version_check"), []byte("abc"), 0644)
	if !c.NeedCheckVersion() {
		t.Fatalf("invalid content => true")
	}
	_ = os.WriteFile(filepath.Join(tmpDir, ".agentbay_version_check"), []byte(fmt.Sprintf("%d", fixedNow.Unix())), 0644)
	if c.NeedCheckVersion() {
		t.Fatalf("fresh cache => false")
	}
	_ = os.WriteFile(filepath.Join(tmpDir, ".agentbay_version_check"), []byte(fmt.Sprintf("%d", fixedNow.Unix()-int64(VersionCheckTTL)-5)), 0644)
	if !c.NeedCheckVersion() {
		t.Fatalf("expired => true")
	}
}

func TestGetLatestAgentBayVersionWithServer(t *testing.T) {
	origHTTP := httpDoFunc
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("User-Agent") == "" {
			t.Fatalf("User-Agent should be set")
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"channels":{"stable":{"latest_version":"0.2.1","versions":[{"version":"0.2.1","is_latest":true}]}}}`)),
		}, nil
	}
	defer func() { httpDoFunc = origHTTP }()
	ver, err := GetLatestAgentBayVersion()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ver != "0.2.1" {
		t.Fatalf("unexpected version %s", ver)
	}
}

func TestGetLatestAgentBayVersion_ParseError(t *testing.T) {
	origHTTP := httpDoFunc
	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("bad content")),
		}, nil
	}
	defer func() { httpDoFunc = origHTTP }()
	_, err := GetLatestAgentBayVersion()
	if err == nil || !strings.Contains(err.Error(), "parse version manifest") {
		t.Fatalf("expect parse error, got %v", err)
	}
}

func TestGetLatestAgentBayVersion_ErrorPaths(t *testing.T) {
	origHTTP := httpDoFunc
	t.Cleanup(func() { httpDoFunc = origHTTP })

	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network down")
	}
	if _, err := GetLatestAgentBayVersion(); err == nil || !strings.Contains(err.Error(), "network down") {
		t.Fatalf("expected http error, got %v", err)
	}

	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("unavailable"))}, nil
	}
	if _, err := GetLatestAgentBayVersion(); err == nil || !strings.Contains(err.Error(), "status code 503") {
		t.Fatalf("expected status error, got %v", err)
	}

	httpDoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"channels":{"stable":{"latest_version":""}}}`))}, nil
	}
	if _, err := GetLatestAgentBayVersion(); err == nil || !strings.Contains(err.Error(), "latest stable version") {
		t.Fatalf("expected missing stable version error, got %v", err)
	}
}

func TestGetLocalVersion_Success(t *testing.T) {
	origExec := execCommandFunc
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "printf 'AgentBay CLI version 5.6.7\\nGit commit: abc\\nBuild date: today\\n'")
	}
	defer func() { execCommandFunc = origExec }()
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.installed = true
	c.execFilePath = "/any/path/agentbay"
	if err := c.GetLocalVersion(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if c.versionLocal != "5.6.7" {
		t.Fatalf("versionLocal mismatch %s", c.versionLocal)
	}
}

func TestGetLocalVersionErrorPaths(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if err := c.GetLocalVersion(); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("expected not installed error, got %v", err)
	}

	origExec := execCommandFunc
	t.Cleanup(func() { execCommandFunc = origExec })
	c.installed = true
	c.execFilePath = "/any/path/agentbay"
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "exit 7")
	}
	if err := c.GetLocalVersion(); err == nil || !strings.Contains(err.Error(), "failed to execute") {
		t.Fatalf("expected execute error, got %v", err)
	}

	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "printf 'bad version output\\n'")
	}
	if err := c.GetLocalVersion(); err == nil || !strings.Contains(err.Error(), "failed to parse version") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestParseAgentBayVersion(t *testing.T) {
	ver, err := parseAgentBayVersion("AgentBay CLI version 0.2.2\nGit commit: abc\n")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ver != "0.2.2" {
		t.Fatalf("version mismatch: %s", ver)
	}
	if _, err := parseAgentBayVersion("Git commit: abc\nAgentBay CLI version 0.2.2\n"); err == nil {
		t.Fatalf("expected parse error when first line does not match")
	}
}

func TestCheckOsTypeAndArchVariants(t *testing.T) {
	origGOOS := runtimeGOOSFunc
	origGOARCH := runtimeGOARCHFunc
	defer func() { runtimeGOOSFunc = origGOOS; runtimeGOARCHFunc = origGOARCH }()
	tests := []struct {
		os      string
		arch    string
		support bool
		file    string
	}{
		{"linux", "amd64", true, "agentbay-1.2.3-linux-amd64.tar.gz"},
		{"darwin", "arm64", true, "agentbay-1.2.3-darwin-arm64.tar.gz"},
		{"windows", "arm64", true, "agentbay-1.2.3-windows-arm64.exe"},
		{"linux", "386", false, ""},
		{"windows", "386", false, ""},
	}
	for _, tc := range tests {
		runtimeGOOSFunc = func(val string) func() string { return func() string { return val } }(tc.os)
		runtimeGOARCHFunc = func(val string) func() string { return func() string { return val } }(tc.arch)
		ctx, _, _ := newOriginCtx()
		c := NewContext(ctx)
		c.CheckOsTypeAndArch()
		if c.osSupport != tc.support {
			t.Fatalf("expect support=%v for %s/%s got %v", tc.support, tc.os, tc.arch, c.osSupport)
		}
		if got := agentBayFileName("1.2.3", tc.os, tc.arch); got != tc.file {
			t.Fatalf("file mismatch got %s want %s", got, tc.file)
		}
	}
}

func TestInstallErrorPaths(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.versionRemote = "1.2.3"
	c.osType = "plan9"
	c.osArch = "amd64"
	if err := c.Install(); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected unsupported error, got %v", err)
	}

	origDownload := downloadAndInstallFunc
	t.Cleanup(func() { downloadAndInstallFunc = origDownload })
	c.osType = "linux"
	c.osArch = "amd64"
	c.execFilePath = filepath.Join(t.TempDir(), "agentbay")
	downloadAndInstallFunc = func(url string, destFile string, exeFilePath string) error {
		return fmt.Errorf("download failed")
	}
	if err := c.Install(); err == nil || !strings.Contains(err.Error(), "download failed") {
		t.Fatalf("expected download error, got %v", err)
	}
}

func TestDownloadAndInstallTarGz(t *testing.T) {
	tarPath := createAgentBayTarGz(t)
	origHTTP := httpGetFunc
	origGOOS := runtimeGOOSFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		fh, err := os.Open(tarPath)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Body: fh}, nil
	}
	runtimeGOOSFunc = func() string { return "linux" }
	defer func() { httpGetFunc = origHTTP; runtimeGOOSFunc = origGOOS }()

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "agentbay.tar.gz")
	exe := filepath.Join(tmp, "agentbay")
	if err := DownloadAndInstall("http://example/agentbay.tar.gz", dest, exe); err != nil {
		t.Fatalf("DownloadAndInstall: %v", err)
	}
	if !fileExists(exe) {
		t.Fatalf("exe not exist")
	}
}

func TestDownloadFileErrorPaths(t *testing.T) {
	origHTTP := httpGetFunc
	t.Cleanup(func() { httpGetFunc = origHTTP })

	httpGetFunc = func(url string) (*http.Response, error) {
		return nil, fmt.Errorf("http get failed")
	}
	if err := downloadFile("http://x", filepath.Join(t.TempDir(), "agentbay")); err == nil || !strings.Contains(err.Error(), "http get failed") {
		t.Fatalf("expected http get error, got %v", err)
	}

	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("data"))}, nil
	}
	err := downloadFile("http://x", filepath.Join(t.TempDir(), "missing", "agentbay"))
	if err == nil || !strings.Contains(err.Error(), "failed to create file") {
		t.Fatalf("expected create file error, got %v", err)
	}
}

func TestExtractAgentBayTarGzErrorPaths(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "missing.tar.gz")
	if err := extractAgentBayTarGz(missing, filepath.Join(tmp, "agentbay")); err == nil {
		t.Fatalf("expected missing source error")
	}

	invalid := filepath.Join(tmp, "invalid.tar.gz")
	if err := os.WriteFile(invalid, []byte("not gzip"), 0644); err != nil {
		t.Fatalf("write invalid gzip: %v", err)
	}
	if err := extractAgentBayTarGz(invalid, filepath.Join(tmp, "agentbay")); err == nil {
		t.Fatalf("expected gzip error")
	}

	other := createTarGzWithFile(t, "not-agentbay", "content")
	if err := extractAgentBayTarGz(other, filepath.Join(tmp, "agentbay")); err == nil || !strings.Contains(err.Error(), "not exist") {
		t.Fatalf("expected missing agentbay error, got %v", err)
	}
}

func TestExtractAgentBayTarGzExistingTargetRemoveError(t *testing.T) {
	src := createTarGzWithEntries(t, []tarEntry{
		{name: "docs", mode: 0755, isDir: true},
		{name: "bin/agentbay", mode: 0755, body: "#!/bin/sh\n"},
	})
	targetDir := filepath.Join(t.TempDir(), "agentbay")
	if err := os.Mkdir(targetDir, 0755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "child"), []byte("x"), 0644); err != nil {
		t.Fatalf("write child: %v", err)
	}

	err := extractAgentBayTarGz(src, targetDir)
	if err == nil || !strings.Contains(err.Error(), "failed to remove existing file") {
		t.Fatalf("expected remove existing target error, got %v", err)
	}
}

func TestDownloadAndInstallWindowsCopiesExe(t *testing.T) {
	origHTTP := httpGetFunc
	origGOOS := runtimeGOOSFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString("windows-agentbay-exe")),
		}, nil
	}
	runtimeGOOSFunc = func() string { return "windows" }
	defer func() { httpGetFunc = origHTTP; runtimeGOOSFunc = origGOOS }()

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "agentbay.exe.download")
	exe := filepath.Join(tmp, "agentbay.exe")
	if err := os.WriteFile(exe, []byte("old-exe"), 0644); err != nil {
		t.Fatalf("write old exe: %v", err)
	}
	if err := DownloadAndInstall("http://example/agentbay.exe", dest, exe); err != nil {
		t.Fatalf("DownloadAndInstall windows: %v", err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	if string(data) != "windows-agentbay-exe" {
		t.Fatalf("exe content mismatch: %s", string(data))
	}
	if fileExists(dest) {
		t.Fatalf("download temp file should be removed")
	}
}

func TestDownloadAndInstallExtractError(t *testing.T) {
	origHTTP := httpGetFunc
	origGOOS := runtimeGOOSFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString("not gzip")),
		}, nil
	}
	runtimeGOOSFunc = func() string { return "linux" }
	defer func() { httpGetFunc = origHTTP; runtimeGOOSFunc = origGOOS }()

	tmp := t.TempDir()
	err := DownloadAndInstall("http://example/bad.tar.gz", filepath.Join(tmp, "bad.tar.gz"), filepath.Join(tmp, "agentbay"))
	if err == nil {
		t.Fatalf("expected extract error")
	}
}

func TestDownloadAndInstall_ErrorStatus(t *testing.T) {
	origHTTP := httpGetFunc
	httpGetFunc = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString("err"))}, nil
	}
	defer func() { httpGetFunc = origHTTP }()
	tmp := t.TempDir()
	err := DownloadAndInstall("http://x", filepath.Join(tmp, "a.tar.gz"), filepath.Join(tmp, "agentbay"))
	if err == nil || !strings.Contains(err.Error(), "status code") {
		t.Fatalf("expect status code error, got %v", err)
	}
}

func createAgentBayTarGz(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	content := []byte("#!/bin/sh\n")
	header := &tar.Header{
		Name: "agentbay",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("write content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	path := filepath.Join(t.TempDir(), "agentbay.tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write tar.gz: %v", err)
	}
	return path
}

func createTarGzWithFile(t *testing.T, name string, body string) string {
	t.Helper()
	return createTarGzWithEntries(t, []tarEntry{{name: name, mode: 0644, body: body}})
}

type tarEntry struct {
	name  string
	mode  int64
	body  string
	isDir bool
}

func createTarGzWithEntries(t *testing.T, entries []tarEntry) string {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, entry := range entries {
		content := []byte(entry.body)
		header := &tar.Header{
			Name: entry.name,
			Mode: entry.mode,
			Size: int64(len(content)),
		}
		if entry.isDir {
			header.Typeflag = tar.TypeDir
			header.Size = 0
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if !entry.isDir {
			if _, err := tw.Write(content); err != nil {
				t.Fatalf("write content: %v", err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	path := filepath.Join(t.TempDir(), "custom.tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write tar.gz: %v", err)
	}
	return path
}

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

func TestRuntimePackageCompilesOnWindowsHook(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based exec tests are covered by hook behavior on non-windows")
	}
}
