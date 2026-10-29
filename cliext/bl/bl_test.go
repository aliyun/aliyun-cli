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
		{goos: "windows", want: filepath.Join("root", prefixDirName, "bin", binName+".exe")},
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
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.configPath = t.TempDir
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

func TestRunWithManagedInstalledBinary(t *testing.T) {
	bridge, _ := binaryContext(t)
	installOld(t, bridge, "2.6.2")
	t.Setenv(noUpdateEnv, "1")
	bridge.command = func(string, ...string) *exec.Cmd { return successfulCommand() }
	if err := bridge.Run([]string{"help"}); err != nil {
		t.Fatalf("Run: %v", err)
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
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	t.Setenv(execPathEnv, os.Args[0])
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
					separator = ";"
				}
				if got["npm_config_prefix"] != "old-lower" || got["NPM_CONFIG_PREFIX"] != "old-upper" || got["PATH"] != binDir+separator+"old-bin" || got["BAILIAN_INSTALL_METHOD"] != "binary" || got["BAILIAN_SHARE_DIR"] != bridge.prefixPath || got["BAILIAN_BIN_DIR"] != binDir || got["ALIBABA_CLOUD_BL_COMPAT_MODE"] != "aliyun bl" || got["KEEP"] != "value" {
					t.Fatalf("managed environment missing: %v", got)
				}
				counts := make(map[string]int)
				for _, entry := range env {
					key, _, _ := strings.Cut(entry, "=")
					counts[strings.ToLower(key)]++
				}
				for _, key := range []string{"path", "bailian_install_method", "bailian_share_dir", "bailian_bin_dir", "alibaba_cloud_bl_compat_mode"} {
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
	if got["NO_UPDATE_NOTIFIER"] != "1" || got["BAILIAN_INSTALL_METHOD"] != "binary" || got["ALIBABA_CLOUD_BL_COMPAT_MODE"] != "aliyun bl" || got["PATH"] != filepath.Join("managed", "bin")+";old" {
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

func TestBinaryOverrideWithoutNodeOrNpm(t *testing.T) {
	t.Setenv(execPathEnv, os.Args[0])
	t.Setenv("PATH", "")
	t.Setenv("ALIBABA_CLOUD_BL_NODE_PATH", "/missing/node")
	t.Setenv("ALIBABA_CLOUD_BL_NPM_PATH", "/missing/npm")
	t.Setenv(versionEnv, "../../invalid")
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.configPath = t.TempDir
	bridge.goos = func() string { return "unsupported" }
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{}, nil }
	bridge.command = func(_ string, args ...string) *exec.Cmd {
		if !reflect.DeepEqual(args, []string{"help"}) {
			t.Fatalf("override performed a version or installation check: %v", args)
		}
		return successfulCommand()
	}
	if err := bridge.Run([]string{"help"}); err != nil {
		t.Fatalf("standalone override must not require Node/npm: %v", err)
	}
}

func TestBinaryManagedPathsAndEnvironment(t *testing.T) {
	t.Setenv(execPathEnv, "")
	ctx, _, _ := newTestContext()
	bridge := NewContext(ctx)
	bridge.configPath = func() string { return "root" }
	bridge.goos = func() string { return "windows" }
	if err := bridge.InitBasicInfo(); err != nil {
		t.Fatal(err)
	}
	wantPrefix := filepath.Join("root", "bl")
	if bridge.execFilePath != filepath.Join(wantPrefix, "bin", "bl.exe") {
		t.Errorf("binary install path = %s", bridge.execFilePath)
	}
	bridge.environ = func() []string {
		return []string{"npm_config_prefix=caller", "bailian_share_dir=outside", "bailian_bin_dir=outside", "Path=caller-bin"}
	}
	bridge.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{}, nil }
	env, err := bridge.BuildExecEnv()
	if err != nil {
		t.Fatal(err)
	}
	got := envMap(env)
	if got["BAILIAN_INSTALL_METHOD"] != "binary" || got["BAILIAN_SHARE_DIR"] != wantPrefix || got["BAILIAN_BIN_DIR"] != filepath.Join(wantPrefix, "bin") || got["npm_config_prefix"] != "caller" {
		t.Errorf("binary environment contract not met: %v", got)
	}
}
