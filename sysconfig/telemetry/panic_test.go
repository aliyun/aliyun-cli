package telemetry

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

// A regression must not turn an uploader invocation of this test binary into
// a recursive run of the test suite. No test uploader may make real requests.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "__telemetry-upload" {
		os.Exit(99)
	}
	os.Exit(m.Run())
}

func TestCollectionPanicDuringBeginDiscardsExecution(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(TelemetryDisableEnv, "")
	resetExecutionState()
	originalPattern := safePluginNamePattern
	t.Cleanup(func() {
		safePluginNamePattern = originalPattern
		resetExecutionState()
	})
	// A valid Begin first makes stale-state cleanup observable.
	Begin(BeginInput{ConfigDir: dir, CommandToken: "version", CommandArgs: []string{"version"}})
	require.NotNil(t, activeExecution)
	safePluginNamePattern = nil
	Begin(BeginInput{
		ConfigDir: dir, CommandToken: "plugin",
		CommandArgs: []string{"plugin", "install", "--name", "panic-test"},
	})
	// executionEnded becomes true only when the injected nil-regexp panic was
	// recovered; an early return before the injection cannot satisfy this.
	require.Nil(t, activeExecution)
	require.True(t, executionEnded)
	End(nil)
	EndWithExitCode(0)
	require.NoFileExists(t, GetInstanceIDFilePath(dir))
	require.NoDirExists(t, filepath.Join(dir, CacheDirName))

	safePluginNamePattern = originalPattern
	Begin(BeginInput{ConfigDir: dir, CommandToken: "version", CommandArgs: []string{"version"}})
	require.NotNil(t, activeExecution, "a later command may collect normally")
	require.False(t, executionEnded)
	// Stop before a successful flush could launch an uploader.
	t.Setenv(TelemetryDisableEnv, "1")
	End(nil)
}

func TestCollectionPanicDuringEndIsNotRetried(t *testing.T) {
	for _, test := range []struct {
		name string
		end  func()
	}{
		{"success", func() { End(nil) }},
		{"business-error", func() { End(errors.New("business failure")) }},
		{"exit-code", func() { EndWithExitCode(1) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := collectionPanicConfigDir(t)
			t.Setenv(TelemetryDisableEnv, "")
			calls := 0
			originalTTY := stdinIsTerminal
			stdinIsTerminal = func(int) bool {
				calls++
				panic("injected collection panic")
			}
			t.Cleanup(func() {
				stdinIsTerminal = originalTTY
				resetExecutionState()
			})
			Begin(BeginInput{ConfigDir: dir, CommandToken: "version", CommandArgs: []string{"version"}})
			require.NotNil(t, activeExecution)
			test.end()
			require.Equal(t, 1, calls, "event construction must reach the injected panic")
			require.Nil(t, activeExecution)
			require.True(t, executionEnded)
			End(nil)
			EndWithExitCode(0)
			require.Equal(t, 1, calls, "a discarded execution must never be retried")
			require.NoDirExists(t, filepath.Join(dir, CacheDirName))
		})
	}
}

func TestCollectionPanicPreservesCLIResult(t *testing.T) {
	for _, result := range []string{"success", "business-error", "plugin-exit"} {
		t.Run(result, func(t *testing.T) {
			baseline := runCollectionPanicCLI(t, result, true)
			withPanic := runCollectionPanicCLI(t, result, false)
			require.Equal(t, baseline, withPanic, "collection panic must not change command output or actual exit status")
			require.Equal(t, "business stdout\n", baseline.stdout)
			switch result {
			case "success":
				require.Equal(t, 0, baseline.exitCode)
				require.Equal(t, "business stderr\n", baseline.stderr)
			case "plugin-exit":
				require.Equal(t, 7, baseline.exitCode)
				require.Equal(t, "business stderr\n", baseline.stderr)
			case "business-error":
				require.Equal(t, 1, baseline.exitCode)
				require.Contains(t, baseline.stderr, "business stderr\n")
				require.Contains(t, baseline.stderr, "ERROR: business failure")
			}
		})
	}
}

type collectionPanicCLIResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func runCollectionPanicCLI(t *testing.T, result string, disabled bool) collectionPanicCLIResult {
	t.Helper()
	dir := collectionPanicConfigDir(t)
	marker := filepath.Join(dir, "collection-panic-reached")
	disable := ""
	if disabled {
		disable = "1"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCollectionPanicHelperProcess$")
	cmd.Env = append(os.Environ(),
		"ALIYUN_TEST_COLLECTION_PANIC_RESULT="+result,
		"ALIYUN_TEST_COLLECTION_PANIC_DIR="+dir,
		"ALIYUN_TEST_COLLECTION_PANIC_MARKER="+marker,
		TelemetryDisableEnv+"="+disable,
		"HOME="+dir, "USERPROFILE="+dir, "NO_COLOR=1",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	require.NoError(t, ctx.Err(), "child timed out: %s", stderr.String())
	if err != nil {
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr, "child launch failed")
	}
	if disabled {
		require.NoFileExists(t, marker, "disabled baseline must not enter collection")
	} else {
		data, err := os.ReadFile(marker)
		require.NoError(t, err, "enabled child must reach the injected panic")
		require.Equal(t, "reached\n", string(data))
	}
	require.NoDirExists(t, filepath.Join(dir, CacheDirName))
	return collectionPanicCLIResult{stdout.String(), stderr.String(), cmd.ProcessState.ExitCode()}
}

func TestCollectionPanicHelperProcess(t *testing.T) {
	result := os.Getenv("ALIYUN_TEST_COLLECTION_PANIC_RESULT")
	if result == "" {
		return
	}
	dir := os.Getenv("ALIYUN_TEST_COLLECTION_PANIC_DIR")
	stdinIsTerminal = func(int) bool {
		if err := os.WriteFile(os.Getenv("ALIYUN_TEST_COLLECTION_PANIC_MARKER"), []byte("reached\n"), 0600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(98)
		}
		panic("injected collection panic")
	}
	cli.EnableExitCode()
	root := &cli.Command{
		Name: "aliyun",
		BeforeExecute: func(_ *cli.Context, args []string) {
			Begin(BeginInput{ConfigDir: dir, CommandToken: "version", CommandArgs: args})
		},
		AfterExecute: HookEnd,
	}
	root.AddSubCommand(&cli.Command{Name: "version", Run: func(ctx *cli.Context, _ []string) error {
		fmt.Fprintln(ctx.Stdout(), "business stdout")
		fmt.Fprintln(ctx.Stderr(), "business stderr")
		if result == "plugin-exit" {
			// Model a plugin that forwards its exit code before Execute can
			// reach AfterExecute; this must preserve the real process status.
			EndWithExitCode(7)
			os.Exit(7)
		}
		if result == "business-error" {
			return errors.New("business failure")
		}
		return nil
	}})
	ctx := cli.NewCommandContext(os.Stdout, os.Stderr)
	ctx.EnterCommand(root)
	root.Execute(ctx, []string{"version"})
	os.Exit(0) // Do not add the testing framework's PASS output to the CLI result.
}

func TestCollectionGuardDoesNotRecoverBusinessPanic(t *testing.T) {
	dir := collectionPanicConfigDir(t)
	t.Setenv(TelemetryDisableEnv, "")
	t.Cleanup(resetExecutionState)
	root := &cli.Command{
		Name: "aliyun",
		BeforeExecute: func(_ *cli.Context, args []string) {
			Begin(BeginInput{ConfigDir: dir, CommandToken: "version", CommandArgs: args})
		},
		AfterExecute: HookEnd,
		Run:          func(*cli.Context, []string) error { panic("business panic") },
	}
	var stdout, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &stderr)
	ctx.EnterCommand(root)
	require.PanicsWithValue(t, "business panic", func() { root.Execute(ctx, nil) })
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
	require.NoDirExists(t, filepath.Join(dir, CacheDirName))
}

func TestCollectionGuardDoesNotHidePurgeError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ordinary-file")
	require.NoError(t, os.WriteFile(path, []byte("keep"), 0600))
	err := Purge(path)
	var pathErr *os.PathError
	require.ErrorAs(t, err, &pathErr, "explicit management operations must still report filesystem errors")
	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.Equal(t, "keep", string(data))
}

func collectionPanicConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(GetConfigFilePath(dir), []byte(`{"enabled":true}`), 0600))
	// Bypass first-use storage so injection cannot be masked by a lock or an
	// unrelated ID initialization failure.
	require.NoError(t, os.WriteFile(GetInstanceIDFilePath(dir), []byte("c4160461-19cf-4ec9-bc28-b2430c1ad90e\n"), 0600))
	return dir
}
