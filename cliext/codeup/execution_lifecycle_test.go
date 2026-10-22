package codeup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

type codeupExecutionReport struct {
	Arguments          []string
	IntegrationMode    string
	IntegrationEntries int
	InheritedValue     string
}

func TestCodeupExecutionChild(t *testing.T) {
	mode := os.Getenv("ALIYUN_CLI_TEST_CODEUP_PROCESS")
	if mode == "" {
		return
	}
	if mode == "failure" {
		fmt.Fprint(os.Stderr, "migration rejected")
		os.Exit(7)
	}
	arguments := os.Args
	for index, value := range arguments {
		if value == "--" {
			arguments = arguments[index+1:]
			break
		}
	}
	entries := 0
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "ALIBABA_CLOUD_CODEUP_INTEGRATION_MODE=") {
			entries++
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(codeupExecutionReport{
		Arguments: arguments, IntegrationMode: os.Getenv("ALIBABA_CLOUD_CODEUP_INTEGRATION_MODE"),
		IntegrationEntries: entries, InheritedValue: os.Getenv("ALIYUN_CLI_TEST_CODEUP_INHERITED"),
	})
	fmt.Fprint(os.Stderr, "migration diagnostic")
	os.Exit(0)
}

func codeupLifecyclePlatform(t *testing.T, directory string, platform string) {
	t.Helper()
	previousPath, previousOS, previousArch := getConfigurePathFunc, runtimeGOOSFunc, runtimeGOARCHFunc
	getConfigurePathFunc = func() string { return directory }
	runtimeGOOSFunc = func() string { return platform }
	runtimeGOARCHFunc = func() string { return "amd64" }
	t.Cleanup(func() {
		getConfigurePathFunc, runtimeGOOSFunc, runtimeGOARCHFunc = previousPath, previousOS, previousArch
	})
}

func codeupLifecycleCommand(t *testing.T, name *string, arguments *[]string) **exec.Cmd {
	t.Helper()
	previous := execCommandFunc
	var command *exec.Cmd
	execCommandFunc = func(executable string, args ...string) *exec.Cmd {
		*name = executable
		*arguments = append([]string(nil), args...)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		t.Cleanup(cancel)
		childArgs := append([]string{"-test.run=^TestCodeupExecutionChild$", "--"}, args...)
		command = exec.CommandContext(ctx, os.Args[0], childArgs...)
		return command
	}
	t.Cleanup(func() { execCommandFunc = previous })
	return &command
}

func TestCodeupCommandInstallsThenExecutesWithFilteredArguments(t *testing.T) {
	directory := t.TempDir()
	codeupLifecyclePlatform(t, directory, "linux")
	downloadDirectory := t.TempDir()
	t.Setenv("TMPDIR", downloadDirectory)
	t.Setenv("TMP", downloadDirectory)
	t.Setenv("TEMP", downloadDirectory)
	t.Setenv("ALIYUN_CLI_TEST_CODEUP_PROCESS", "success")
	t.Setenv("ALIYUN_CLI_TEST_CODEUP_INHERITED", "preserved")
	t.Setenv("ALIBABA_CLOUD_CODEUP_INTEGRATION_MODE", "stale mode")
	staleArchive := filepath.Join(os.TempDir(), "codeup-cli_download.zip")
	require.Equal(t, downloadDirectory, os.TempDir())
	require.NoError(t, os.WriteFile(staleArchive, []byte("incomplete previous download"), 0600))
	previousDownload := downloadAndExtractFunc
	downloads := 0
	downloadAndExtractFunc = func(url, archive, executable string) error {
		downloads++
		require.Equal(t, codeupCliBaseUrl+"codeup-cli-Linux-64.zip", url)
		require.Equal(t, staleArchive, archive)
		require.NoFileExists(t, archive)
		require.Equal(t, filepath.Join(directory, "codeup-cli"), executable)
		return os.WriteFile(executable, []byte("installed binary"), 0755)
	}
	t.Cleanup(func() { downloadAndExtractFunc = previousDownload })
	var executable string
	var arguments []string
	command := codeupLifecycleCommand(t, &executable, &arguments)
	var stdout, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &stderr)
	args := []string{"--profile=production", "import", "--read-timeout:30", "-p", "ignored", "--yes", "--repository", "team/repo with spaces", "--retry-timeout", "4", "--quiet", "--token=literal"}
	require.NoError(t, NewCodeupCliCommand().Run(ctx, args))
	require.Equal(t, 1, downloads)
	require.Equal(t, filepath.Join(directory, "codeup-cli"), executable)
	wantArguments := []string{"import", "--repository", "team/repo with spaces", "--token=literal"}
	require.Equal(t, wantArguments, arguments)
	var report codeupExecutionReport
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &report), stdout.String())
	require.Equal(t, wantArguments, report.Arguments)
	require.Equal(t, "aliyun codeup-cli", report.IntegrationMode)
	require.Equal(t, 1, report.IntegrationEntries)
	require.Equal(t, "preserved", report.InheritedValue)
	require.Equal(t, "migration diagnostic", stderr.String())
	require.Same(t, os.Stdin, (*command).Stdin)
}

func TestCodeupRunReusesWindowsInstallationAndReportsProcessFailure(t *testing.T) {
	directory := t.TempDir()
	codeupLifecyclePlatform(t, directory, "windows")
	executable := filepath.Join(directory, "codeup-cli.exe")
	require.NoError(t, os.WriteFile(executable, []byte("existing installation"), 0755))
	previousDownload := downloadAndExtractFunc
	downloads := 0
	downloadAndExtractFunc = func(string, string, string) error {
		downloads++
		return errors.New("unexpected installation")
	}
	t.Cleanup(func() { downloadAndExtractFunc = previousDownload })
	t.Setenv("ALIYUN_CLI_TEST_CODEUP_PROCESS", "failure")
	var commandName string
	var arguments []string
	codeupLifecycleCommand(t, &commandName, &arguments)
	var stdout, stderr bytes.Buffer
	options := NewContext(cli.NewCommandContext(&stdout, &stderr))
	err := options.Run([]string{"status", "--quiet", "-q", "-p=default", "--profile"})
	require.ErrorContains(t, err, "failed to execute "+executable)
	require.Equal(t, "migration rejected", stderr.String())
	require.Empty(t, stdout.String())
	require.Equal(t, []string{"status"}, arguments)
	require.Equal(t, executable, commandName)
	require.Zero(t, downloads)
	require.True(t, options.installed)
}

func TestCodeupRunRejectsUnsupportedPlatformAndInstallationFailure(t *testing.T) {
	directory := t.TempDir()
	codeupLifecyclePlatform(t, directory, "plan9")
	previousDownload, previousCommand := downloadAndExtractFunc, execCommandFunc
	downloads, executions := 0, 0
	downloadAndExtractFunc = func(string, string, string) error {
		downloads++
		return errors.New("download interrupted")
	}
	execCommandFunc = func(string, ...string) *exec.Cmd {
		executions++
		return nil
	}
	t.Cleanup(func() { downloadAndExtractFunc, execCommandFunc = previousDownload, previousCommand })
	var stdout, stderr bytes.Buffer
	options := NewContext(cli.NewCommandContext(&stdout, &stderr))
	require.ErrorContains(t, options.Run([]string{"status"}), "your os type plan9 and arch amd64 is not supported")
	require.Zero(t, downloads)
	runtimeGOOSFunc = func() string { return "linux" }
	downloadDirectory := t.TempDir()
	t.Setenv("TMPDIR", downloadDirectory)
	t.Setenv("TMP", downloadDirectory)
	t.Setenv("TEMP", downloadDirectory)
	err := options.Run([]string{"status"})
	require.ErrorContains(t, err, "failed to download and extract codeup-cli from "+codeupCliBaseUrl+"codeup-cli-Linux-64.zip")
	require.ErrorContains(t, err, "download interrupted")
	require.False(t, options.installed)
	require.Equal(t, 1, downloads)
	require.Zero(t, executions)
}
