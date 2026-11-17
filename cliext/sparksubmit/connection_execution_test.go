package sparksubmit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/require"
)

func TestSparkConnectionPropertiesUseSTSFlagsAndConfiguredWorkspace(t *testing.T) {
	c, _, _ := sparkLifecycleContext(t)
	profile := config.Profile{Mode: config.StsToken, AccessKeyId: "sts-id", AccessKeySecret: "sts-secret", StsToken: "session-token"}
	previousProfile := loadProfileFunc
	loadProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
		require.Same(t, c.originCtx, ctx)
		return profile, nil
	}
	t.Cleanup(func() { loadProfileFunc = previousProfile })
	config.AddFlags(c.originCtx.Flags())
	c.originCtx.Flags().Get(config.RegionFlagName).SetValue(" cn-shanghai ")
	c.originCtx.Flags().Get(config.RegionFlagName).SetAssigned(true)
	t.Setenv(EnvEndpoint, " https://spark.internal.example.invalid ")
	t.Setenv(EnvWorkspaceID, " workspace-from-env ")
	require.NoError(t, c.prepareConnectionProperties())
	path := filepath.Join(c.confDir, "connection.properties")
	props, err := readPropertiesFile(path)
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"accessKeyId": "sts-id", "accessKeySecret": "sts-secret", "securityToken": "session-token",
		"regionId": "cn-shanghai", "endpoint": "https://spark.internal.example.invalid", "workspaceId": "workspace-from-env",
	}, props)

	t.Setenv(EnvEndpoint, "")
	t.Setenv(EnvWorkspaceID, "")
	profile = config.Profile{Mode: config.AK, AccessKeyId: "configured-id", AccessKeySecret: "configured-secret"}
	c.originCtx.Flags().Get(config.RegionFlagName).SetValue(" ")
	unknown := cli.NewFlagSet()
	regionID := config.NewRegionIdFlag()
	regionID.SetValue(" cn-beijing ")
	regionID.SetAssigned(true)
	unknown.Add(regionID)
	c.originCtx.SetUnknownFlags(unknown)
	require.NoError(t, os.WriteFile(path, []byte("# retained user settings\n\nmalformed line\nworkspaceId = saved-workspace\nendpoint = custom.example.invalid\nqueue=dev=priority\n"), 0600))
	require.NoError(t, c.prepareConnectionProperties())
	props, err = readPropertiesFile(path)
	require.NoError(t, err)
	require.Equal(t, "cn-beijing", props["regionId"])
	require.Equal(t, "saved-workspace", props["workspaceId"])
	require.Equal(t, "custom.example.invalid", props["endpoint"])
	require.Equal(t, "dev=priority", props["queue"])
	require.Equal(t, "configured-id", props["accessKeyId"])
	require.Equal(t, "configured-secret", props["accessKeySecret"])
	require.NotContains(t, props, "securityToken")
}

func TestSparkConnectionPropertiesRejectProfileAndFilesystemFailures(t *testing.T) {
	for _, test := range []struct {
		name, want string
		profile    config.Profile
		loadErr    error
	}{
		{name: "profile failure", loadErr: errors.New("configuration unreadable"), want: "load profile: configuration unreadable"},
		{name: "empty access key", profile: config.Profile{Mode: config.AK}, want: "access key is empty"},
		{name: "missing region", profile: config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret"}, want: "region is empty"},
		{name: "unsupported credentials", profile: config.Profile{Mode: config.Anonymous}, want: "Anonymous profile mode is not supported"},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, _, _ := sparkLifecycleContext(t)
			previousProfile := loadProfileFunc
			loadProfileFunc = func(*cli.Context) (config.Profile, error) { return test.profile, test.loadErr }
			t.Cleanup(func() { loadProfileFunc = previousProfile })
			require.ErrorContains(t, c.prepareConnectionProperties(), test.want)
			require.NoDirExists(t, c.confDir)
		})
	}
	c, _, _ := sparkLifecycleContext(t)
	previousProfile := loadProfileFunc
	loadProfileFunc = func(*cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou"}, nil
	}
	t.Cleanup(func() { loadProfileFunc = previousProfile })
	require.NoError(t, os.WriteFile(c.installDir, []byte("parent is a file"), 0600))
	require.Error(t, c.prepareConnectionProperties())
	require.NoError(t, os.Remove(c.installDir))
	require.NoError(t, os.MkdirAll(filepath.Join(c.confDir, "connection.properties"), 0755))
	require.Error(t, c.prepareConnectionProperties(), "a directory cannot be read as connection properties")
	require.Empty(t, flagStringValue(nil, config.RegionFlagName))
}

type sparkExecutionReport struct {
	Arguments []string
	ConfDir   string
	Props     map[string]string
}

func TestSparkSubmitExecutionChild(t *testing.T) {
	mode := os.Getenv("ALIYUN_CLI_TEST_SPARK_PROCESS")
	if mode == "" {
		return
	}
	if mode == "failure" {
		fmt.Fprint(os.Stderr, "application rejected")
		os.Exit(9)
	}
	arguments := os.Args
	for index, value := range arguments {
		if value == "--" {
			arguments = arguments[index+1:]
			break
		}
	}
	confDir := os.Getenv("SPARK_CONF_DIR")
	props, err := readPropertiesFile(filepath.Join(confDir, "connection.properties"))
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(10)
	}
	_ = json.NewEncoder(os.Stdout).Encode(sparkExecutionReport{Arguments: arguments, ConfDir: confDir, Props: props})
	fmt.Fprint(os.Stderr, "spark diagnostic")
	os.Exit(0)
}

func TestSparkRunAndExecutionPassArgumentsAndConnectionProperties(t *testing.T) {
	c, stdout, stderr := sparkLifecycleContext(t)
	previousLook, previousProfile, previousCommand := lookPathFunc, loadProfileFunc, execCommandFunc
	t.Cleanup(func() {
		lookPathFunc, loadProfileFunc, execCommandFunc = previousLook, previousProfile, previousCommand
	})
	lookPathFunc = func(name string) (string, error) {
		require.Equal(t, "java", name)
		return filepath.Join(c.configPath, "java"), nil
	}
	profileCalls := 0
	loadProfileFunc = func(*cli.Context) (config.Profile, error) {
		profileCalls++
		return config.Profile{Mode: config.StsToken, AccessKeyId: "id", AccessKeySecret: "secret", StsToken: "token", RegionId: "cn-hangzhou"}, nil
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(c.execFilePath), 0755))
	require.NoError(t, os.WriteFile(c.execFilePath, []byte("cached tool"), 0755))
	t.Setenv(EnvNoUpdateCheck, "1")
	t.Setenv(EnvWorkspaceID, "workspace")
	t.Setenv("ALIYUN_CLI_TEST_SPARK_PROCESS", "success")
	var command *exec.Cmd
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		require.Equal(t, c.execFilePath, name)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		t.Cleanup(cancel)
		command = exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestSparkSubmitExecutionChild$", "--"}, args...)...)
		return command
	}
	args := []string{"--region=cn-hangzhou", "--name", "Daily job with spaces", "-p", "default", "--output=json", "--class", "example.Main", "oss://bucket/main.jar", "12"}
	wantArguments := []string{"--name", "Daily job with spaces", "--output=json", "--class", "example.Main", "oss://bucket/main.jar", "12"}
	if runtime.GOOS == "windows" {
		require.ErrorContains(t, NewSparkSubmitCommand().Run(c.originCtx, args), "spark-submit is not supported on Windows")
		require.Zero(t, profileCalls)
		require.Nil(t, command)
		require.NoError(t, c.prepareConnectionProperties())
		require.NoError(t, c.execute(c.removeFlagsForMainCli(args)))
	} else {
		require.NoError(t, NewSparkSubmitCommand().Run(c.originCtx, args))
	}
	var report sparkExecutionReport
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &report), stdout.String())
	require.Equal(t, wantArguments, report.Arguments)
	require.Equal(t, c.confDir, report.ConfDir)
	require.Equal(t, "id", report.Props["accessKeyId"])
	require.Equal(t, "secret", report.Props["accessKeySecret"])
	require.Equal(t, "token", report.Props["securityToken"])
	require.Equal(t, "workspace", report.Props["workspaceId"])
	require.Equal(t, "spark diagnostic", stderr.String())
	require.Same(t, os.Stdin, command.Stdin)

	stdout.Reset()
	stderr.Reset()
	customConf := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(customConf, "connection.properties"), []byte("workspaceId=custom\n"), 0600))
	t.Setenv("SPARK_CONF_DIR", customConf)
	require.NoError(t, c.execute([]string{"--status", "job-123"}))
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &report))
	require.Equal(t, customConf, report.ConfDir)
	require.Equal(t, "custom", report.Props["workspaceId"])
	require.Equal(t, []string{"--status", "job-123"}, report.Arguments)

	t.Setenv("ALIYUN_CLI_TEST_SPARK_PROCESS", "failure")
	err := c.execute([]string{"--kill", "job-123"})
	var status *ExitError
	require.ErrorAs(t, err, &status)
	require.Equal(t, 9, status.ExitCode())
	require.Equal(t, "subprocess exited with code 9", status.Error())
	execCommandFunc = func(string, ...string) *exec.Cmd {
		return exec.Command(filepath.Join(t.TempDir(), "missing-spark-submit"))
	}
	require.ErrorContains(t, c.execute([]string{"--status", "job-123"}), "execute "+c.execFilePath)
}

func TestSparkInitializationResolvesExternalBinAndRunPrerequisites(t *testing.T) {
	c, _, _ := sparkLifecycleContext(t)
	external := filepath.Join(t.TempDir(), "custom", "bin", "spark-submit")
	t.Setenv(EnvExecPath, " "+external+" ")
	c.initBasicInfo()
	require.Equal(t, external, c.execFilePath)
	require.Equal(t, filepath.Dir(filepath.Dir(external)), c.installDir)
	require.Equal(t, filepath.Join(c.installDir, "conf"), c.confDir)
	t.Setenv(EnvExecPath, "")
	previousLook := lookPathFunc
	lookPathFunc = func(string) (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { lookPathFunc = previousLook })
	err := c.Run([]string{"--version"})
	if runtime.GOOS == "windows" {
		require.ErrorContains(t, err, "not supported on Windows")
	} else {
		require.ErrorContains(t, err, "java not found in PATH")
	}
}
