package telemetry

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInstanceIDStorageIsIndependentOfPreferences(t *testing.T) {
	for _, savePreference := range []bool{false, true} {
		t.Run(fmt.Sprint(savePreference), func(t *testing.T) {
			dir := t.TempDir()
			if savePreference {
				require.NoError(t, Save(dir, &Config{Enabled: false}))
			}
			before, _ := os.ReadFile(GetConfigFilePath(dir))
			id, err := EnsureInstanceID(dir)
			require.NoError(t, err)
			require.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, id)
			data, err := os.ReadFile(GetInstanceIDFilePath(dir))
			require.NoError(t, err)
			require.Equal(t, id+"\n", string(data))
			again, err := EnsureInstanceID(dir)
			require.NoError(t, err)
			require.Equal(t, id, again)
			after, err := os.ReadFile(GetConfigFilePath(dir))
			if savePreference {
				require.NoError(t, err)
				require.Equal(t, before, after)
				require.JSONEq(t, `{"enabled":false}`, string(after))
			} else {
				require.ErrorIs(t, err, os.ErrNotExist)
			}
		})
	}
}

func TestInstanceIDLockContentionSkipsInitialization(t *testing.T) {
	dir := t.TempDir()
	unlock, err := acquireFileLock(filepath.Join(dir, instanceIDLockName), 0)
	require.NoError(t, err)
	defer unlock()
	start := time.Now()
	id, err := EnsureInstanceID(dir)
	require.ErrorIs(t, err, ErrLockBusy)
	require.Empty(t, id)
	require.Less(t, time.Since(start), time.Second)
	require.NoFileExists(t, GetInstanceIDFilePath(dir))
	unlock()
	id, err = EnsureInstanceID(dir)
	require.NoError(t, err)

	// Existing installations need no lock or writes on the command path.
	unlock, err = acquireFileLock(filepath.Join(dir, instanceIDLockName), 0)
	require.NoError(t, err)
	defer unlock()
	again, err := EnsureInstanceID(dir)
	require.NoError(t, err)
	require.Equal(t, id, again)
}

func TestPurgePreservesPreferenceAndStableLock(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Save(dir, &Config{Enabled: false}))
	before, err := os.ReadFile(GetConfigFilePath(dir))
	require.NoError(t, err)
	oldID, err := EnsureInstanceID(dir)
	require.NoError(t, err)
	lockPath := filepath.Join(dir, instanceIDLockName)
	lockInfo, err := os.Stat(lockPath)
	require.NoError(t, err)
	for _, pipeline := range []string{PipelineInner, PipelinePublic} {
		_, err := WriteEvent(GetCacheDir(dir, pipeline), Event{EventID: pipeline})
		require.NoError(t, err)
	}
	require.DirExists(t, filepath.Join(dir, "telemetry-cache"))
	require.NoError(t, Purge(dir))
	require.NoFileExists(t, GetInstanceIDFilePath(dir))
	require.NoDirExists(t, filepath.Join(dir, CacheDirName))
	after, err := os.ReadFile(GetConfigFilePath(dir))
	require.NoError(t, err)
	require.Equal(t, before, after)
	afterLockInfo, err := os.Stat(lockPath)
	require.NoError(t, err)
	require.True(t, os.SameFile(lockInfo, afterLockInfo))
	newID, err := EnsureInstanceID(dir)
	require.NoError(t, err)
	require.NotEqual(t, oldID, newID)
}

func TestPurgeCannotResetIDWhileAnotherProcessHoldsLock(t *testing.T) {
	// The OS-lock tests separately exercise distinct processes. This test
	// verifies that purge uses the same lock as ID initialization.
	dir := t.TempDir()
	id, err := EnsureInstanceID(dir)
	require.NoError(t, err)
	unlock, err := acquireFileLock(filepath.Join(dir, instanceIDLockName), 0)
	require.NoError(t, err)
	defer unlock()
	require.ErrorIs(t, Purge(dir), ErrLockBusy)
	got, err := LoadInstanceID(dir)
	require.NoError(t, err)
	require.Equal(t, id, got)
	unlock()
	require.NoError(t, Purge(dir))
	require.NoFileExists(t, GetInstanceIDFilePath(dir))
	require.NoFileExists(t, GetConfigFilePath(dir))
}

func TestInvalidStoredTelemetryStateDoesNotEnableCollection(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(GetConfigFilePath(dir), []byte(`{"enabled":`), 0600))
	cfg, err := Load(dir)
	require.Error(t, err)
	require.True(t, Disabled(cfg))
	// An explicit setting can repair malformed preference JSON.
	require.NoError(t, Save(dir, &Config{Enabled: false}))
	_, err = Load(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(GetInstanceIDFilePath(dir), []byte("broken"), 0600))
	_, err = EnsureInstanceID(dir)
	require.Error(t, err)
	// Purge can reset a malformed ID without having to read or migrate it.
	require.NoError(t, Purge(dir))
	_, err = EnsureInstanceID(dir)
	require.NoError(t, err)
}

func TestEndRechecksDisabledPreference(t *testing.T) {
	for _, environmentDisabled := range []bool{false, true} {
		t.Run(fmt.Sprint(environmentDisabled), func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv(TelemetryDisableEnv, "")
			Begin(BeginInput{ConfigDir: dir, CommandToken: "version", CommandArgs: []string{"version"}})
			require.NotNil(t, activeExecution)
			if environmentDisabled {
				t.Setenv(TelemetryDisableEnv, "1")
			} else {
				require.NoError(t, Save(dir, &Config{Enabled: false}))
			}
			End(nil)
			require.NoFileExists(t, GetInstanceIDFilePath(dir))
			require.NoDirExists(t, filepath.Join(dir, CacheDirName))
		})
	}
}

func TestTelemetryManagementRoutesDoNotStartCollection(t *testing.T) {
	for _, args := range [][]string{
		{"configure", "telemetry"},
		{"configure", "telemetry", "show"},
		{"configure", "telemetry", "enable"},
		{"configure", "telemetry", "disable"},
		{"configure", "telemetry", "purge"},
		{"--profile", "foo", "configure", "telemetry", "purge"},
		{"configure", "--profile:foo", "telemetry", "purge"},
		{"configure", "--profile=foo", "telemetry", "purge"},
		{"configure", "--profile", "foo", "telemetry", "purge"},
		{"configure", "--insecure", "telemetry", "purge"},
		{"configure", "-h", "telemetry"},
		{"help", "configure", "telemetry"},
		{"configure", "help", "telemetry"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv(TelemetryDisableEnv, "")
			Begin(BeginInput{ConfigDir: dir, CommandToken: "configure", CommandArgs: args})
			require.Nil(t, activeExecution)
			End(nil)
			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Empty(t, entries)
		})
	}
	for _, args := range [][]string{
		{"configure"},
		{"configure", "set", "--profile", "telemetry"},
		{"configure", "--profile:telemetry", "set"},
		{"version"},
	} {
		require.False(t, isTelemetryManagementCommand(args), "%v", args)
	}
}

func TestInstanceIDConcurrentProcesses(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	type child struct {
		cmd    *exec.Cmd
		input  io.WriteCloser
		output bytes.Buffer
		stderr bytes.Buffer
	}
	children := make([]*child, 8)
	for i := range children {
		c := &child{cmd: exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInstanceIDProcessHelper$")}
		c.cmd.Env = append(os.Environ(), "ALIYUN_TEST_INSTANCE_ID_DIR="+dir)
		c.cmd.Stdout = &c.output
		c.cmd.Stderr = &c.stderr
		var err error
		c.input, err = c.cmd.StdinPipe()
		require.NoError(t, err)
		require.NoError(t, c.cmd.Start())
		children[i] = c
	}
	// Release all initialized subprocesses together, with no shell involved.
	for _, c := range children {
		_, err := io.WriteString(c.input, "start\n")
		require.NoError(t, err)
		require.NoError(t, c.input.Close())
	}
	var publishedIDs []string
	for _, c := range children {
		require.NoError(t, c.cmd.Wait(), "%s", c.stderr.String())
		result := strings.TrimSpace(c.output.String())
		if result != "busy" {
			publishedIDs = append(publishedIDs, result)
		}
	}
	require.NotEmpty(t, publishedIDs)
	id, err := LoadInstanceID(dir)
	require.NoError(t, err)
	require.NotEmpty(t, id)
	for _, result := range publishedIDs {
		require.Equal(t, id, result, "all successful initializers must use the persisted ID")
	}
	require.NoFileExists(t, GetConfigFilePath(dir))
}

func TestInstanceIDProcessHelper(t *testing.T) {
	dir := os.Getenv("ALIYUN_TEST_INSTANCE_ID_DIR")
	if dir == "" {
		return
	}
	if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	id, err := EnsureInstanceID(dir)
	if errors.Is(err, ErrLockBusy) {
		fmt.Print("busy")
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else {
		fmt.Print(id)
	}
	os.Exit(0)
}
