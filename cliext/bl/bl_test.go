package bl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func newTestContext() (*cli.Context, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := cli.NewCommandContext(stdout, stderr)
	config.AddFlags(ctx.Flags())
	return ctx, stdout, stderr
}

func helperCommand(exitCode int) *exec.Cmd {
	return exec.Command(os.Args[0], "-test.run=^TestHelperProcess$", "--", strconv.Itoa(exitCode))
}

func successfulCommand() *exec.Cmd {
	return helperCommand(0)
}

func TestHelperProcess(t *testing.T) {
	for i, arg := range os.Args {
		if arg != "--" || i+1 >= len(os.Args) {
			continue
		}
		code, err := strconv.Atoi(os.Args[i+1])
		if err != nil {
			os.Exit(2)
		}
		os.Exit(code)
	}
}

func TestNewBlCommandUsesRawArgs(t *testing.T) {
	cmd := NewBlCommand()
	if cmd.Name != "bl" {
		t.Fatalf("name = %q", cmd.Name)
	}
	if cmd.Usage != "aliyun bl <command> [args...]" {
		t.Fatalf("usage = %q", cmd.Usage)
	}
	if !cmd.RawArgs || !cmd.SkipDefaultHelp || !cmd.EnableUnknownFlag {
		t.Fatalf("raw delegation flags are not enabled: %#v", cmd)
	}
	if cmd.Run == nil {
		t.Fatal("Run is nil")
	}
}

func TestPrepareArgsStripsOnlyHostFlags(t *testing.T) {
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	args := []string{
		"help",
		"--profile", "prod",
		"--mode=RamRoleArn",
		"--ram-role-arn", "acs:ram::123:role/test",
		"--access-key-id", "bl-ak",
		"--access-key-secret=bl-sk",
		"--security-token", "bl-token",
		"--region", "cn-hangzhou",
		"--config", "team",
		"--endpoint", "example.test",
		"-e", "short.example.test",
		"--output", "json",
		"--unknown=value",
		"--", "--profile", "child-profile",
	}
	want := []string{
		"help",
		"--mode=RamRoleArn",
		"--access-key-id", "bl-ak",
		"--access-key-secret=bl-sk",
		"--security-token", "bl-token",
		"--region", "cn-hangzhou",
		"--config", "team",
		"--endpoint", "example.test",
		"-e", "short.example.test",
		"--output", "json",
		"--unknown=value",
		"--", "--profile", "child-profile",
	}
	got, err := bridge.PrepareArgs(args)
	if err != nil {
		t.Fatalf("PrepareArgs: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", want, got)
	}
	if got, _ := ctx.Flags().Get(config.ProfileFlagName).GetValue(); got != "prod" {
		t.Fatalf("profile = %q", got)
	}
	if ctx.Flags().Get(config.ModeFlagName).IsAssigned() {
		t.Fatal("child mode assigned as host authentication mode")
	}
}

func TestPrepareArgsSupportsProfileShorthandForms(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "separate", args: []string{"-p", "prod", "help"}},
		{name: "equals", args: []string{"-p=prod", "help"}},
		{name: "joined", args: []string{"-pprod", "help"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _, _ := newTestContext()
			got, err := NewContext(ctx).PrepareArgs(tc.args)
			if err != nil {
				t.Fatalf("PrepareArgs: %v", err)
			}
			if !reflect.DeepEqual(got, []string{"help"}) {
				t.Fatalf("forwarded args = %#v", got)
			}
			if value, _ := ctx.Flags().Get(config.ProfileFlagName).GetValue(); value != "prod" {
				t.Fatalf("profile = %q", value)
			}
		})
	}
}

func TestPrepareArgsRejectsMissingHostFlagValue(t *testing.T) {
	ctx, _, _ := newTestContext()
	_, err := NewContext(ctx).PrepareArgs([]string{"help", "--profile"})
	if err == nil || !strings.Contains(err.Error(), "--profile requires a value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildExecEnvReplacesAKAndRemovesStaleSTS(t *testing.T) {
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.environ = func() []string {
		return []string{
			"KEEP=value",
			"ALIBABA_CLOUD_ACCESS_KEY_ID=old-ak",
			"ALIBABA_CLOUD_ACCESS_KEY_SECRET=old-sk",
			"ALIBABA_CLOUD_SECURITY_TOKEN=old-token",
		}
	}
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.AK, AccessKeyId: "profile-ak", AccessKeySecret: "profile-sk"}, nil
	}

	env, err := bridge.BuildExecEnv()
	if err != nil {
		t.Fatalf("BuildExecEnv: %v", err)
	}
	got := envMap(env)
	if got["ALIBABA_CLOUD_ACCESS_KEY_ID"] != "profile-ak" || got["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] != "profile-sk" {
		t.Fatalf("profile credentials not injected: %#v", got)
	}
	if _, ok := got["ALIBABA_CLOUD_SECURITY_TOKEN"]; ok {
		t.Fatalf("stale STS token remains: %#v", got)
	}
	if got["KEEP"] != "value" {
		t.Fatalf("unrelated env missing: %#v", got)
	}
}

func TestBuildExecEnvInjectsSTSWithoutRegion(t *testing.T) {
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.environ = func() []string { return []string{"ALIBABA_CLOUD_REGION_ID=cn-hangzhou"} }
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Mode: config.StsToken, AccessKeyId: "ak", AccessKeySecret: "sk", StsToken: "token", RegionId: "cn-shanghai",
		}, nil
	}

	env, err := bridge.BuildExecEnv()
	if err != nil {
		t.Fatalf("BuildExecEnv: %v", err)
	}
	got := envMap(env)
	if got["ALIBABA_CLOUD_SECURITY_TOKEN"] != "token" {
		t.Fatalf("STS token not injected: %#v", got)
	}
	if got["ALIBABA_CLOUD_REGION_ID"] != "cn-hangzhou" {
		t.Fatalf("region must not be injected or replaced: %#v", got)
	}
}

func TestBuildExecEnvPreservesUserEnvForUnconfiguredProfile(t *testing.T) {
	base := []string{
		"ALIBABA_CLOUD_ACCESS_KEY_ID=user-ak",
		"ALIBABA_CLOUD_ACCESS_KEY_SECRET=user-sk",
		"ALIBABA_CLOUD_SECURITY_TOKEN=user-token",
	}
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.environ = func() []string { return append([]string(nil), base...) }
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{Name: "default"}, nil }

	got, err := bridge.BuildExecEnv()
	if err != nil {
		t.Fatalf("BuildExecEnv: %v", err)
	}
	for _, entry := range base {
		key, value, _ := strings.Cut(entry, "=")
		if envMap(got)[key] != value {
			t.Fatalf("inherited %s changed", key)
		}
	}
}

func TestBuildExecEnvRejectsIncompleteConfiguredCredentials(t *testing.T) {
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) {
		return config.Profile{Name: "broken", Mode: config.AK, AccessKeyId: "ak"}, nil
	}
	_, err := bridge.BuildExecEnv()
	if err == nil || !strings.Contains(err.Error(), "access key ID and secret must be configured together") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildExecEnvPropagatesProfileError(t *testing.T) {
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) {
		return config.Profile{}, fmt.Errorf("unknown profile missing")
	}
	_, err := bridge.BuildExecEnv()
	if err == nil || !strings.Contains(err.Error(), "unknown profile missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitBasicInfoUsesManagedPlatformPaths(t *testing.T) {
	for _, tc := range []struct {
		goos string
		want string
	}{
		{goos: "linux", want: filepath.Join("root", prefixDirName, "bin", binName)},
		{goos: "windows", want: filepath.Join("root", prefixDirName, binName+".cmd")},
	} {
		t.Run(tc.goos, func(t *testing.T) {
			t.Setenv(execPathEnv, "")
			ctx, _, _ := newTestContext()
			bridge := NewContext(ctx)
			bridge.configPath = func() string { return "root" }
			bridge.goos = func() string { return tc.goos }
			if err := bridge.InitBasicInfo(); err != nil {
				t.Fatalf("InitBasicInfo: %v", err)
			}
			if bridge.execFilePath != tc.want {
				t.Fatalf("exec path = %q, want %q", bridge.execFilePath, tc.want)
			}
		})
	}
}

func TestEnsureNodeAvailableEnforces1817(t *testing.T) {
	for _, tc := range []struct {
		name         string
		major, minor int
		wantErr      bool
	}{
		{name: "too old minor", major: 18, minor: 16, wantErr: true},
		{name: "minimum", major: 18, minor: 17},
		{name: "newer major", major: 20, minor: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(nodePathEnv, "")
			ctx, _, _ := newTestContext()
			bridge := NewContext(ctx)
			bridge.lookPath = func(string) (string, error) { return "/fake/node", nil }
			bridge.nodeVersion = func(string) (int, int, int, error) { return tc.major, tc.minor, 0, nil }
			err := bridge.EnsureNodeAvailable()
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateExecPathOverride(t *testing.T) {
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)

	missing := filepath.Join(t.TempDir(), "missing")
	bridge.execFilePath = missing
	if err := bridge.validateExecPathOverride(); err == nil || !strings.Contains(err.Error(), execPathEnv) {
		t.Fatalf("missing path error = %v", err)
	}

	dir := t.TempDir()
	bridge.execFilePath = dir
	if err := bridge.validateExecPathOverride(); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("directory error = %v", err)
	}

	if runtime.GOOS != "windows" {
		file := filepath.Join(t.TempDir(), "bl")
		if err := os.WriteFile(file, []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
		bridge.execFilePath = file
		bridge.goos = func() string { return "linux" }
		if err := bridge.validateExecPathOverride(); err == nil || !strings.Contains(err.Error(), "executable") {
			t.Fatalf("permission error = %v", err)
		}
	}
}

func TestNpmInstallUsesVersionRegistryAndPrefix(t *testing.T) {
	t.Setenv(versionEnv, "2.5.0")
	t.Setenv(registryEnv, "https://registry.example.test")
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.npmPath = "/fake/npm"
	bridge.prefixPath = "/managed/bl"
	var gotName string
	var gotArgs []string
	bridge.command = func(name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return successfulCommand()
	}
	if err := bridge.npmInstall(); err != nil {
		t.Fatalf("npmInstall: %v", err)
	}
	if gotName != "/fake/npm" {
		t.Fatalf("command = %q", gotName)
	}
	want := []string{
		"install", "-g", "--prefix", "/managed/bl", "--no-fund", "--no-audit", "--loglevel=warn",
		"--registry", "https://registry.example.test", "bailian-cli@2.5.0",
	}
	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("npm args mismatch\nwant: %#v\n got: %#v", want, gotArgs)
	}
}

func TestExecutePreservesArgsStdioCwdAndExitCode(t *testing.T) {
	ctx, stdout, stderr := newTestContext()
	bridge := NewContext(ctx)
	bridge.execFilePath = "/fake/bl"
	bridge.environ = func() []string { return []string{"KEEP=value"} }
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{}, nil }
	workingDir := t.TempDir()
	bridge.getwd = func() (string, error) { return workingDir, nil }
	var gotName string
	var gotArgs []string
	var gotCmd *exec.Cmd
	bridge.command = func(name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = append([]string(nil), args...)
		gotCmd = helperCommand(7)
		return gotCmd
	}

	err := bridge.Execute([]string{"auth", "login", "--console"})
	exitErr, ok := err.(*ExitError)
	if !ok || exitErr.Code != 7 {
		t.Fatalf("exit error = %#v", err)
	}
	if gotName != "/fake/bl" || !reflect.DeepEqual(gotArgs, []string{"auth", "login", "--console"}) {
		t.Fatalf("command = %q %#v", gotName, gotArgs)
	}
	if gotCmd.Stdout != stdout || gotCmd.Stderr != stderr || gotCmd.Stdin != os.Stdin || gotCmd.Dir != workingDir {
		t.Fatalf("stdio/cwd not preserved: %#v", gotCmd)
	}
}

func TestLoadBlProfileUsesRawConfigWithoutInheritedSTS(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	t.Setenv("ALIBABA_CLOUD_SECURITY_TOKEN", "inherited-token")
	configDir := filepath.Join(home, ".aliyun")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configJSON := `{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"ak","access_key_secret":"sk"}]}`
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(configJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newTestContext()
	profile, err := loadBlProfile(ctx)
	if err != nil {
		t.Fatalf("loadBlProfile: %v", err)
	}
	if profile.StsToken != "" {
		t.Fatalf("inherited token contaminated raw profile: %q", profile.StsToken)
	}
}

func TestLoadBlProfileAppliesExplicitHostCredentialFlags(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	ctx, _, _ := newTestContext()
	for name, value := range map[string]string{
		config.ModeFlagName:            "StsToken",
		config.AccessKeyIdFlagName:     "flag-ak",
		config.AccessKeySecretFlagName: "flag-sk",
		config.StsTokenFlagName:        "flag-token",
	} {
		flag := ctx.Flags().Get(name)
		flag.SetAssigned(true)
		flag.SetValue(value)
	}
	profile, err := loadBlProfile(ctx)
	if err != nil {
		t.Fatalf("loadBlProfile: %v", err)
	}
	if profile.Mode != config.StsToken || profile.AccessKeyId != "flag-ak" || profile.AccessKeySecret != "flag-sk" || profile.StsToken != "flag-token" {
		t.Fatalf("flag credentials not applied: %#v", profile)
	}
}

func TestBuildExecEnvResolvesCachedOAuthCredentials(t *testing.T) {
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.environ = func() []string { return nil }
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Mode: config.OAuth, AccessKeyId: "oauth-ak", AccessKeySecret: "oauth-sk", StsToken: "oauth-token",
			StsExpiration: time.Now().Unix() + 3600, RegionId: "cn-hangzhou",
		}, nil
	}
	env, err := bridge.BuildExecEnv()
	if err != nil {
		t.Fatalf("BuildExecEnv: %v", err)
	}
	got := envMap(env)
	if got["ALIBABA_CLOUD_ACCESS_KEY_ID"] != "oauth-ak" || got["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] != "oauth-sk" || got["ALIBABA_CLOUD_SECURITY_TOKEN"] != "oauth-token" {
		t.Fatalf("oauth credentials not resolved: %#v", got)
	}
}

func TestLoadBlProfileRecognizesModeFromExplicitHostCredentials(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	ctx, _, _ := newTestContext()
	for name, value := range map[string]string{
		config.AccessKeyIdFlagName:     "flag-ak",
		config.AccessKeySecretFlagName: "flag-sk",
	} {
		flag := ctx.Flags().Get(name)
		flag.SetAssigned(true)
		flag.SetValue(value)
	}
	profile, err := loadBlProfile(ctx)
	if err != nil {
		t.Fatalf("loadBlProfile: %v", err)
	}
	if profile.Mode != config.AK {
		t.Fatalf("mode = %q, want AK", profile.Mode)
	}
}

func TestRunWithExecOverride(t *testing.T) {
	t.Setenv(execPathEnv, os.Args[0])
	t.Setenv(nodePathEnv, "/fake/node")
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.configPath = t.TempDir
	bridge.nodeVersion = func(string) (int, int, int, error) { return 20, 0, 0, nil }
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{}, nil }
	bridge.getwd = func() (string, error) { return t.TempDir(), nil }
	var gotArgs []string
	bridge.command = func(_ string, args ...string) *exec.Cmd {
		gotArgs = append([]string(nil), args...)
		return successfulCommand()
	}

	if err := bridge.Run([]string{"--profile", "prod", "help"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !reflect.DeepEqual(gotArgs, []string{"help"}) {
		t.Fatalf("forwarded args = %#v", gotArgs)
	}
}

func TestRunWithManagedInstalledPackage(t *testing.T) {
	t.Setenv(execPathEnv, "")
	t.Setenv(nodePathEnv, "/fake/node")
	t.Setenv(npmPathEnv, os.Args[0])
	root := t.TempDir()
	execPath := filepath.Join(root, prefixDirName, "bin", binName)
	if err := os.MkdirAll(filepath.Dir(execPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath, []byte("installed"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.configPath = func() string { return root }
	bridge.nodeVersion = func(string) (int, int, int, error) { return 20, 0, 0, nil }
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{}, nil }
	bridge.getwd = func() (string, error) { return root, nil }
	bridge.command = func(string, ...string) *exec.Cmd { return successfulCommand() }

	if err := bridge.Run([]string{"help"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestEnsureNodeAvailableErrors(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		t.Setenv(nodePathEnv, "")
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.lookPath = func(string) (string, error) { return "", fmt.Errorf("not found") }
		if err := bridge.EnsureNodeAvailable(); err == nil || !strings.Contains(err.Error(), "node >= 18.17") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("version detection", func(t *testing.T) {
		t.Setenv(nodePathEnv, "/bad/node")
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.nodeVersion = func(string) (int, int, int, error) { return 0, 0, 0, fmt.Errorf("broken") }
		if err := bridge.EnsureNodeAvailable(); err == nil || !strings.Contains(err.Error(), "failed to detect node version") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestEnsureNpmAvailable(t *testing.T) {
	t.Run("override", func(t *testing.T) {
		t.Setenv(npmPathEnv, "/custom/npm")
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		if err := bridge.EnsureNpmAvailable(); err != nil {
			t.Fatalf("EnsureNpmAvailable: %v", err)
		}
		if bridge.npmPath != "/custom/npm" {
			t.Fatalf("npm path = %q", bridge.npmPath)
		}
	})

	t.Run("beside node", func(t *testing.T) {
		t.Setenv(npmPathEnv, "")
		dir := t.TempDir()
		npmPath := filepath.Join(dir, "npm")
		if err := os.WriteFile(npmPath, []byte("npm"), 0o644); err != nil {
			t.Fatal(err)
		}
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.nodePath = filepath.Join(dir, "node")
		bridge.goos = func() string { return "linux" }
		if err := bridge.EnsureNpmAvailable(); err != nil {
			t.Fatalf("EnsureNpmAvailable: %v", err)
		}
		if bridge.npmPath != npmPath {
			t.Fatalf("npm path = %q, want %q", bridge.npmPath, npmPath)
		}
	})

	t.Run("path fallback", func(t *testing.T) {
		t.Setenv(npmPathEnv, "")
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.nodePath = filepath.Join(t.TempDir(), "node")
		bridge.goos = func() string { return "linux" }
		bridge.lookPath = func(name string) (string, error) {
			if name != "npm" {
				t.Fatalf("lookup = %q", name)
			}
			return "/path/npm", nil
		}
		if err := bridge.EnsureNpmAvailable(); err != nil {
			t.Fatalf("EnsureNpmAvailable: %v", err)
		}
		if bridge.npmPath != "/path/npm" {
			t.Fatalf("npm path = %q", bridge.npmPath)
		}
	})

	t.Run("missing", func(t *testing.T) {
		t.Setenv(npmPathEnv, "")
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.nodePath = filepath.Join(t.TempDir(), "node")
		bridge.goos = func() string { return "linux" }
		bridge.lookPath = func(string) (string, error) { return "", fmt.Errorf("not found") }
		if err := bridge.EnsureNpmAvailable(); err == nil || !strings.Contains(err.Error(), npmPathEnv) {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestEnsurePrefixAndPackage(t *testing.T) {
	t.Run("already installed", func(t *testing.T) {
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.prefixPath = filepath.Join(t.TempDir(), "prefix")
		bridge.installed = true
		bridge.command = func(string, ...string) *exec.Cmd {
			t.Fatal("npm must not run")
			return nil
		}
		if err := bridge.EnsurePrefixAndPackage(); err != nil {
			t.Fatalf("EnsurePrefixAndPackage: %v", err)
		}
	})

	t.Run("installs package", func(t *testing.T) {
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.prefixPath = filepath.Join(t.TempDir(), "prefix")
		bridge.execFilePath = filepath.Join(bridge.prefixPath, "bin", binName)
		bridge.npmPath = "/fake/npm"
		bridge.command = func(string, ...string) *exec.Cmd {
			if err := os.MkdirAll(filepath.Dir(bridge.execFilePath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(bridge.execFilePath, []byte("installed"), 0o644); err != nil {
				t.Fatal(err)
			}
			return successfulCommand()
		}
		if err := bridge.EnsurePrefixAndPackage(); err != nil {
			t.Fatalf("EnsurePrefixAndPackage: %v", err)
		}
		if !bridge.installed {
			t.Fatal("package was not marked installed")
		}
	})

	t.Run("missing binary after install", func(t *testing.T) {
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.prefixPath = filepath.Join(t.TempDir(), "prefix")
		bridge.execFilePath = filepath.Join(bridge.prefixPath, "bin", binName)
		bridge.npmPath = "/fake/npm"
		bridge.command = func(string, ...string) *exec.Cmd { return successfulCommand() }
		if err := bridge.EnsurePrefixAndPackage(); err == nil || !strings.Contains(err.Error(), "binary not found") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("npm failure", func(t *testing.T) {
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.prefixPath = filepath.Join(t.TempDir(), "prefix")
		bridge.execFilePath = filepath.Join(bridge.prefixPath, "bin", binName)
		bridge.npmPath = "/fake/npm"
		bridge.command = func(string, ...string) *exec.Cmd { return helperCommand(4) }
		if err := bridge.EnsurePrefixAndPackage(); err == nil || !strings.Contains(err.Error(), "failed to install") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("prefix creation failure", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.prefixPath = filepath.Join(file, "prefix")
		if err := bridge.EnsurePrefixAndPackage(); err == nil || !strings.Contains(err.Error(), "failed to create prefix") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestGetNodeVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires a POSIX executable script")
	}
	for _, tc := range []struct {
		name    string
		script  string
		major   int
		minor   int
		patch   int
		wantErr string
	}{
		{name: "semantic version", script: "#!/bin/sh\nprintf 'v20.11.1\\n'\n", major: 20, minor: 11, patch: 1},
		{name: "missing patch", script: "#!/bin/sh\nprintf 'v18.17\\n'\n", major: 18, minor: 17},
		{name: "malformed", script: "#!/bin/sh\nprintf 'not-a-version\\n'\n", wantErr: "unexpected node --version output"},
		{name: "command failure", script: "#!/bin/sh\nexit 3\n", wantErr: "exit status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "node")
			if err := os.WriteFile(path, []byte(tc.script), 0o755); err != nil {
				t.Fatal(err)
			}
			major, minor, patch, err := getNodeVersion(path)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("getNodeVersion: %v", err)
			}
			if major != tc.major || minor != tc.minor || patch != tc.patch {
				t.Fatalf("version = %d.%d.%d", major, minor, patch)
			}
		})
	}
}

func TestBuildExecEnvAdditionalModes(t *testing.T) {
	t.Run("sts requires token", func(t *testing.T) {
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.loadProfile = func(*cli.Context) (config.Profile, error) {
			return config.Profile{Mode: config.StsToken, AccessKeyId: "ak", AccessKeySecret: "sk"}, nil
		}
		if _, err := bridge.BuildExecEnv(); err == nil || !strings.Contains(err.Error(), "security token") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("anonymous removes inherited credentials", func(t *testing.T) {
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.environ = func() []string {
			return []string{"KEEP=value", "ALIBABA_CLOUD_ACCESS_KEY_ID=ak", "ALIBABA_CLOUD_ACCESS_KEY_SECRET=sk", "ALIBABA_CLOUD_SECURITY_TOKEN=token"}
		}
		bridge.loadProfile = func(*cli.Context) (config.Profile, error) {
			return config.Profile{Mode: config.Anonymous}, nil
		}
		env, err := bridge.BuildExecEnv()
		if err != nil {
			t.Fatalf("BuildExecEnv: %v", err)
		}
		got := envMap(env)
		if got["KEEP"] != "value" || got["ALIBABA_CLOUD_ACCESS_KEY_ID"] != "" || got["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] != "" || got["ALIBABA_CLOUD_SECURITY_TOKEN"] != "" {
			t.Fatal("anonymous environment retained credentials or lost unrelated values")
		}
	})

	t.Run("credential provider setup error", func(t *testing.T) {
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.loadProfile = func(*cli.Context) (config.Profile, error) {
			return config.Profile{Name: "cloud-sso", Mode: config.CloudSSO}, nil
		}
		if _, err := bridge.BuildExecEnv(); err == nil || !strings.Contains(err.Error(), "can't get credential") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestExecuteSuccessAndErrors(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.execFilePath = "/fake/bl"
		bridge.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{}, nil }
		bridge.getwd = func() (string, error) { return t.TempDir(), nil }
		bridge.command = func(string, ...string) *exec.Cmd { return successfulCommand() }
		if err := bridge.Execute([]string{"help"}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})

	t.Run("working directory failure", func(t *testing.T) {
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{}, nil }
		bridge.getwd = func() (string, error) { return "", fmt.Errorf("cwd unavailable") }
		if err := bridge.Execute(nil); err == nil || !strings.Contains(err.Error(), "current working directory") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("command start failure", func(t *testing.T) {
		ctx, _, _ := newTestContext()
		bridge := NewContext(ctx)
		bridge.execFilePath = filepath.Join(t.TempDir(), "missing")
		bridge.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{}, nil }
		bridge.getwd = func() (string, error) { return t.TempDir(), nil }
		if err := bridge.Execute(nil); err == nil || !strings.Contains(err.Error(), "failed to execute") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("exit error accessors", func(t *testing.T) {
		err := &ExitError{Code: 9}
		if err.Error() != "subprocess exited with code 9" || err.ExitCode() != 9 {
			t.Fatalf("exit error = %q, code %d", err.Error(), err.ExitCode())
		}
	})
}

func TestNewBlCommandRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires a POSIX executable script")
	}
	nodePath := filepath.Join(t.TempDir(), "node")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\nprintf 'v20.0.0\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	t.Setenv(execPathEnv, os.Args[0])
	t.Setenv(nodePathEnv, nodePath)
	ctx, _, _ := newTestContext()
	cmd := NewBlCommand()

	if err := cmd.Run(ctx, []string{"-test.run=^TestHelperProcess$", "--", "0"}); err != nil {
		t.Fatalf("Run success: %v", err)
	}

	cli.DisableExitCode()
	defer cli.EnableExitCode()
	if err := cmd.Run(ctx, []string{"-test.run=^TestHelperProcess$", "--", "7"}); err != nil {
		t.Fatalf("Run exit: %v", err)
	}
}

func TestPrepareArgsPreservesChildOwnedFlags(t *testing.T) {
	for _, args := range [][]string{
		{"chat", "--mode", "stream", "--language", "python", "--profile", "demo"},
		{"--mode=stream", "chat", "--language=python", "--profile=demo"},
		{"chat", "--mode=", "--language", "", "--profile", "demo"},
		{"chat", "--mode", "--profile", "demo"},
		{"chat", "--language", "--profile", "demo"},
		{"chat", "--mode=first", "--mode=second", "--read-timeout", "30", "--profile", "demo"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			ctx, _, _ := newTestContext()
			before := append([]string(nil), args...)
			got, err := NewContext(ctx).PrepareArgs(args)
			if err != nil {
				t.Fatal(err)
			}
			want := args[:len(args)-2]
			if strings.HasPrefix(args[len(args)-1], "--profile=") {
				want = args[:len(args)-1]
			}
			if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(args, before) {
				t.Fatalf("want %q, got %q; input changed: %v", want, got, !reflect.DeepEqual(args, before))
			}
			if ctx.Flags().Get(config.ModeFlagName).IsAssigned() || ctx.Flags().Get(config.LanguageFlagName).IsAssigned() {
				t.Fatal("child mode/language assigned as host flags")
			}
		})
	}
}

func TestLoadBlProfileIgnoresPersistentProfile(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		flags   map[string]string
		mode    config.AuthenticateMode
	}{
		{name: "wrong stored account", content: `{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"wrong-ak","access_key_secret":"wrong-sk"}]}`},
		{name: "broken selected file", content: `not json`, flags: map[string]string{config.ProfileFlagName: "missing"}},
		{name: "explicit AK", content: `not json`, flags: map[string]string{config.AccessKeyIdFlagName: "explicit-ak", config.AccessKeySecretFlagName: "explicit-sk"}, mode: config.AK},
		{name: "explicit STS", content: `not json`, flags: map[string]string{config.AccessKeyIdFlagName: "explicit-ak", config.AccessKeySecretFlagName: "explicit-sk", config.StsTokenFlagName: "explicit-token"}, mode: config.StsToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("HOMEDRIVE", "")
			t.Setenv("HOMEPATH", "")
			t.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")
			path := filepath.Join(home, "selected.json")
			if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, _, _ := newTestContext()
			bridge := NewContext(ctx)
			if err := bridge.assignFlag(config.ConfigurePathFlagName, path); err != nil {
				t.Fatal(err)
			}
			for name, value := range tc.flags {
				if err := bridge.assignFlag(name, value); err != nil {
					t.Fatal(err)
				}
			}
			profile, err := loadBlProfile(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if profile.Mode != tc.mode || profile.AccessKeyId != tc.flags[config.AccessKeyIdFlagName] || profile.AccessKeySecret != tc.flags[config.AccessKeySecretFlagName] || profile.StsToken != tc.flags[config.StsTokenFlagName] {
				t.Fatal("profile did not contain only explicit credentials")
			}
			base := []string{"KEEP=value", "ALIBABA_CLOUD_ACCESS_KEY_ID=inherited-ak", "ALIBABA_CLOUD_ACCESS_KEY_SECRET=inherited-sk", "ALIBABA_CLOUD_SECURITY_TOKEN=inherited-token"}
			bridge.environ = func() []string { return base }
			env, err := bridge.BuildExecEnv()
			if err != nil {
				t.Fatal(err)
			}
			got := envMap(env)
			if tc.mode == "" {
				for _, entry := range base {
					key, value, _ := strings.Cut(entry, "=")
					if got[key] != value {
						t.Fatalf("inherited %s changed", key)
					}
				}
			} else if got["ALIBABA_CLOUD_ACCESS_KEY_ID"] != "explicit-ak" || got["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] != "explicit-sk" || got["ALIBABA_CLOUD_SECURITY_TOKEN"] != tc.flags[config.StsTokenFlagName] {
				t.Fatal("explicit credentials not mapped")
			}
		})
	}
}

func TestBuildExecEnvManagedEnvironment(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		for _, mode := range []config.AuthenticateMode{"", config.AK} {
			t.Run(goos+"/"+string(mode), func(t *testing.T) {
				ctx, _, _ := newTestContext()
				bridge := NewContext(ctx)
				bridge.prefixPath = filepath.Join(t.TempDir(), "managed prefix")
				bridge.goos = func() string { return goos }
				base := []string{"KEEP=value", "npm_config_prefix=old-lower", "NPM_CONFIG_PREFIX=old-upper", "PATH=old-bin", "BAILIAN_INSTALL_METHOD=other", "ALIBABA_CLOUD_BL_COMPAT_MODE=bl"}
				if goos == "windows" {
					base[3] = "Path=old-bin"
				}
				bridge.environ = func() []string { return base }
				bridge.loadProfile = func(*cli.Context) (config.Profile, error) {
					if mode == "" {
						return config.Profile{}, nil
					}
					return config.Profile{Mode: mode, AccessKeyId: "ak", AccessKeySecret: "sk"}, nil
				}
				env, err := bridge.BuildExecEnv()
				if err != nil {
					t.Fatal(err)
				}
				got := envMap(env)
				binDir, separator := filepath.Join(bridge.prefixPath, "bin"), ":"
				if goos == "windows" {
					binDir, separator = bridge.prefixPath, ";"
				}
				if got["npm_config_prefix"] != bridge.prefixPath || got["PATH"] != binDir+separator+"old-bin" || got["BAILIAN_INSTALL_METHOD"] != "npm" || got["ALIBABA_CLOUD_BL_COMPAT_MODE"] != "aliyun bl" || got["KEEP"] != "value" {
					t.Fatalf("managed environment missing: %v", got)
				}
				if _, ok := got["NPM_CONFIG_PREFIX"]; ok {
					t.Fatal("uppercase npm prefix remains")
				}
				counts := make(map[string]int)
				for _, entry := range env {
					key, _, _ := strings.Cut(entry, "=")
					counts[strings.ToLower(key)]++
				}
				for _, key := range []string{"npm_config_prefix", "path", "bailian_install_method", "alibaba_cloud_bl_compat_mode"} {
					if counts[key] != 1 {
						t.Fatalf("%s count = %d", key, counts[key])
					}
				}
				if base[1] != "npm_config_prefix=old-lower" {
					t.Fatal("input environment changed")
				}
			})
		}
	}
}

func TestBuildExecEnvUpdateNotifier(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  string
	}{
		{value: "1", want: "1"},
		{value: "0", want: "user-setting"},
		{value: "", want: "user-setting"},
	} {
		t.Run(tc.value, func(t *testing.T) {
			ctx, _, _ := newTestContext()
			bridge := NewContext(ctx)
			bridge.environ = func() []string {
				return []string{"ALIBABA_CLOUD_BL_NO_UPDATE_CHECK=" + tc.value, "NO_UPDATE_NOTIFIER=user-setting"}
			}
			bridge.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{}, nil }
			env, err := bridge.BuildExecEnv()
			if err != nil {
				t.Fatal(err)
			}
			if envMap(env)["NO_UPDATE_NOTIFIER"] != tc.want {
				t.Fatal("update notifier not mapped")
			}
		})
	}
}

func TestBuildExecEnvWindowsCaseInsensitiveControls(t *testing.T) {
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.goos = func() string { return "windows" }
	bridge.prefixPath = "managed"
	bridge.environ = func() []string {
		return []string{"alibaba_cloud_bl_no_update_check=1", "No_Update_Notifier=0", "bailian_install_method=other", "alibaba_cloud_bl_compat_mode=bl", "pAtH=old"}
	}
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{}, nil }
	env, err := bridge.BuildExecEnv()
	if err != nil {
		t.Fatal(err)
	}
	got := envMap(env)
	if got["NO_UPDATE_NOTIFIER"] != "1" || got["BAILIAN_INSTALL_METHOD"] != "npm" || got["ALIBABA_CLOUD_BL_COMPAT_MODE"] != "aliyun bl" || got["PATH"] != "managed;old" {
		t.Fatalf("windows controls not replaced: %v", got)
	}
	counts := make(map[string]int)
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		counts[strings.ToLower(key)]++
	}
	for _, key := range []string{"no_update_notifier", "bailian_install_method", "alibaba_cloud_bl_compat_mode", "path"} {
		if counts[key] != 1 {
			t.Fatalf("duplicate %s", key)
		}
	}
}

func envMap(env []string) map[string]string {
	result := make(map[string]string, len(env))
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			result[key] = value
		}
	}
	return result
}
