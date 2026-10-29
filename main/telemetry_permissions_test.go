//go:build linux || darwin || windows

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

const telemetryPermissionChildEnv = "ALIYUN_TEST_TELEMETRY_PERMISSION_CHILD"

// Run the real CLI entrypoint in child processes, retaining os.Exit behavior
// without adding a production hook. Other package tests run normally.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "__telemetry-upload" {
		// Any test that starts an upload child must neither send telemetry nor
		// recursively run the test suite, even without the helper marker.
		os.Exit(99)
	}
	if os.Getenv(telemetryPermissionChildEnv) == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestMainTelemetryWritePermissionFailurePreservesCommand(t *testing.T) {
	for _, scenario := range []string{
		"writable_control",
		"cannot_create_lock",
		"cannot_open_lock_for_writing",
		"cannot_publish_instance_id",
		"cannot_create_cache_directory",
		"cannot_create_event_file",
	} {
		for _, command := range []struct {
			name    string
			args    []string
			success bool
		}{
			{"success", []string{"version"}, true},
			{"command_error", []string{"version", "--permission-test-unknown-flag"}, false},
			{"structured_error", []string{"--cli-ai-mode", "version", "--permission-test-unknown-flag"}, false},
		} {
			t.Run(scenario+"/"+command.name, func(t *testing.T) {
				home := t.TempDir()
				dir := filepath.Join(home, ".aliyun")
				require.NoError(t, os.Mkdir(dir, 0700))
				const profile = `{"current":"default","profiles":[{"name":"default","mode":"AK","region_id":"cn-hangzhou","language":"en"}]}`
				const preference = `{"enabled":true}`
				const instanceID = "d69b3a04-b81c-47e7-a882-504e5865352d\n"
				require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(profile), 0600))
				require.NoError(t, os.WriteFile(telemetry.GetConfigFilePath(dir), []byte(preference), 0600))
				lockPath := filepath.Join(dir, "telemetry-instance-id.lock")
				cacheDir := telemetry.GetPublicCacheDir(dir)
				hasID := scenario == "cannot_create_cache_directory" || scenario == "cannot_create_event_file"
				hasLock := scenario == "cannot_open_lock_for_writing" || scenario == "cannot_publish_instance_id"
				if hasID {
					require.NoError(t, os.WriteFile(telemetry.GetInstanceIDFilePath(dir), []byte(instanceID), 0600))
				}
				if hasLock {
					require.NoError(t, os.WriteFile(lockPath, []byte("stable lock"), 0600))
				}

				deniedPath := dir
				if scenario == "cannot_open_lock_for_writing" {
					deniedPath = lockPath
				} else if scenario == "cannot_create_event_file" {
					require.NoError(t, os.MkdirAll(cacheDir, 0700))
					deniedPath = cacheDir
				}
				if scenario != "writable_control" {
					denyTelemetryWrites(t, deniedPath)
					verifyTelemetryWriteDenied(t, deniedPath)
				}
				if scenario == "cannot_publish_instance_id" {
					// Acquiring the existing lock must still work: this case
					// exercises failure to create the ID's temporary file.
					f, err := os.OpenFile(lockPath, os.O_RDWR, 0)
					require.NoError(t, err)
					require.NoError(t, f.Close())
				}

				// Compare the exact same command and filesystem with telemetry
				// disabled versus enabled. No cloud API or credentials are used.
				baseline := runTelemetryPermissionCommand(t, home, true, command.args)
				if command.success {
					require.Equal(t, 0, baseline.exitCode)
					require.Equal(t, cli.Version+"\n", baseline.stdout)
					require.Empty(t, baseline.stderr)
				} else {
					require.Greater(t, baseline.exitCode, 0)
					require.Empty(t, baseline.stdout)
					require.Contains(t, baseline.stderr, "permission-test-unknown-flag")
				}
				if scenario == "writable_control" {
					require.NoFileExists(t, telemetry.GetInstanceIDFilePath(dir))
					require.NoDirExists(t, filepath.Join(dir, telemetry.CacheDirName))
				}
				got := runTelemetryPermissionCommand(t, home, false, command.args)
				require.Equal(t, baseline.exitCode, got.exitCode, "telemetry changed the command exit status")
				require.Equal(t, baseline.stdout, got.stdout, "telemetry changed stdout")
				require.Equal(t, baseline.stderr, got.stderr, "telemetry changed stderr")

				assertTelemetryPermissionFile(t, filepath.Join(dir, "config.json"), profile)
				assertTelemetryPermissionFile(t, telemetry.GetConfigFilePath(dir), preference)
				if scenario == "writable_control" {
					// Prove each command actually collects telemetry in this
					// harness, so globally skipping collection cannot false-pass
					// all permission cases. TestMain prevents cloud upload.
					require.FileExists(t, telemetry.GetInstanceIDFilePath(dir))
					entries, err := os.ReadDir(cacheDir)
					require.NoError(t, err)
					require.Len(t, entries, 1)
					data, err := os.ReadFile(filepath.Join(cacheDir, entries[0].Name()))
					require.NoError(t, err)
					var event telemetry.Event
					require.NoError(t, json.Unmarshal(data, &event))
					require.NotEmpty(t, event.EventID)
					require.NotEmpty(t, event.InstanceID)
					if command.success {
						require.Equal(t, "Success", event.Result)
					} else {
						require.NotEqual(t, "Success", event.Result)
					}
					return
				}
				if hasID {
					assertTelemetryPermissionFile(t, telemetry.GetInstanceIDFilePath(dir), instanceID)
				} else {
					require.NoFileExists(t, telemetry.GetInstanceIDFilePath(dir))
				}
				if hasLock {
					assertTelemetryPermissionFile(t, lockPath, "stable lock")
				} else {
					require.NoFileExists(t, lockPath)
				}
				if scenario == "cannot_create_event_file" {
					entries, err := os.ReadDir(cacheDir)
					require.NoError(t, err)
					require.Empty(t, entries, "failed telemetry must leave no event file")
				} else {
					require.NoDirExists(t, filepath.Join(dir, telemetry.CacheDirName))
				}
			})
		}
	}
}

func TestMainTelemetryMissingUncreatableConfigDirectoryPreservesStartupFailure(t *testing.T) {
	for _, command := range []struct {
		name string
		args []string
	}{
		{"version", []string{"version"}},
		{"invalid_flag", []string{"version", "--permission-test-unknown-flag"}},
		{"ai_mode", []string{"--cli-ai-mode", "version", "--permission-test-unknown-flag"}},
	} {
		t.Run(command.name, func(t *testing.T) {
			home := t.TempDir()
			dir := filepath.Join(home, ".aliyun")
			denyTelemetryWrites(t, home)
			require.NoDirExists(t, dir)

			// Verify directory creation itself is denied, including on Windows
			// where creating a file and a subdirectory use different ACL bits.
			err := os.Mkdir(dir, 0755)
			if err == nil {
				require.NoError(t, os.Remove(dir))
				t.Skip("current user or filesystem bypasses directory creation restrictions")
			}
			require.ErrorIs(t, err, os.ErrPermission)
			var pathErr *os.PathError
			require.ErrorAs(t, err, &pathErr)

			// GetConfigPath currently panics before telemetry.Begin. Preserve
			// this existing failure in both modes, including the real exit code.
			// Suppress runtime stack addresses (which vary across processes) so
			// the actual diagnostic can be compared byte-for-byte without recovery.
			baseline := runTelemetryPermissionCommand(t, home, true, command.args, "GOTRACEBACK=none")
			require.Equal(t, 2, baseline.exitCode)
			require.Empty(t, baseline.stdout)
			require.Contains(t, baseline.stderr, "panic: mkdir ")
			require.Contains(t, baseline.stderr, ".aliyun")
			require.Contains(t, baseline.stderr, pathErr.Err.Error())
			require.NoDirExists(t, dir)

			got := runTelemetryPermissionCommand(t, home, false, command.args, "GOTRACEBACK=none")
			require.Equal(t, baseline.exitCode, got.exitCode, "telemetry changed the startup exit status")
			require.Equal(t, baseline.stdout, got.stdout, "telemetry changed stdout")
			require.Equal(t, baseline.stderr, got.stderr, "telemetry changed stderr")
			require.NoDirExists(t, dir)
			entries, err := os.ReadDir(home)
			require.NoError(t, err)
			require.Empty(t, entries, "failed startup must leave no config or telemetry files")
		})
	}
}

func verifyTelemetryWriteDenied(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	var f *os.File
	if info.IsDir() {
		f, err = os.CreateTemp(path, ".write-probe-*")
	} else {
		f, err = os.OpenFile(path, os.O_RDWR, 0)
	}
	if err == nil {
		require.NoError(t, f.Close())
		if info.IsDir() {
			require.NoError(t, os.Remove(f.Name()))
		}
		t.Skip("current user or filesystem bypasses write restrictions; cannot exercise permission denial")
	}
	require.ErrorIs(t, err, os.ErrPermission, "expected a real permission failure")
}

func assertTelemetryPermissionFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, string(got))
}

type telemetryPermissionResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func runTelemetryPermissionCommand(t *testing.T, home string, disabled bool, args []string, extraEnv ...string) telemetryPermissionResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], args...)
	disableValue := "0"
	if disabled {
		disableValue = "1"
	}
	// Use a minimal environment so credentials, completion, agent detection,
	// mock settings, and the developer's real config cannot affect the child.
	cmd.Env = []string{
		"HOME=" + home, "USERPROFILE=" + home, "HOMEDRIVE=", "HOMEPATH=",
		telemetryPermissionChildEnv + "=1",
		telemetry.TelemetryDisableEnv + "=" + disableValue,
		"ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT=http://127.0.0.1:1",
		sysmock.EnvMockEnabled + "=false",
		"ALIBABA_CLOUD_CLI_AI_MODE=0", "ALIBABA_CLOUD_CLI_AGENT_INTEGRATION=0",
		"NO_COLOR=1", "LANG=C",
	}
	cmd.Env = append(cmd.Env, extraEnv...)
	for _, name := range []string{"PATH", "SYSTEMROOT", "TMPDIR", "TMP", "TEMP"} {
		if value, ok := os.LookupEnv(name); ok {
			cmd.Env = append(cmd.Env, name+"="+value)
		}
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	require.NoError(t, ctx.Err(), "CLI timed out: %s", stderr.String())
	if err != nil {
		var exitErr *exec.ExitError
		require.True(t, errors.As(err, &exitErr), "could not run CLI child: %v", err)
	}
	return telemetryPermissionResult{stdout.String(), stderr.String(), cmd.ProcessState.ExitCode()}
}
