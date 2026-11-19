package updatecheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/cli/upgrade"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/updatenotifier"
	"github.com/stretchr/testify/require"
)

func testService(t *testing.T) *Service {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("ALIBABA_CLOUD_CLI_PLUGINS_DIR", filepath.Join(home, ".aliyun", "plugins"))
	t.Setenv("ALIBABA_CLOUD_CLI_PLUGIN_SOURCE_BASE", "")
	t.Setenv("ALIBABA_CLOUD_CLI_PLUGIN_SITE", "cn")
	t.Setenv("ALIBABA_CLOUD_ENDPOINT_SUFFIX", "")
	t.Setenv(updatenotifier.DisableEnv, "0")
	s, err := New(filepath.Join(home, ".aliyun"))
	require.NoError(t, err)
	now := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return now }
	s.CLI = func(context.Context, string) (upgrade.UpdateResult, error) {
		return upgrade.UpdateResult{Current: cli.GetVersion(), Latest: "99.0.0", Target: "99.0.0", UpdateAvailable: true, Status: "update_available", Command: "aliyun upgrade"}, nil
	}
	s.Plugins = func(context.Context, string, plugin.UpdateScope) (plugin.UpdateReport, error) {
		return plugin.UpdateReport{Status: "complete", Fresh: true, Plugins: []plugin.UpdateResult{}}, nil
	}
	return s
}

func TestConcurrentChecksAndExplicitBypass(t *testing.T) {
	s := testService(t)
	var requests atomic.Int32
	original := s.CLI
	s.CLI = func(ctx context.Context, version string) (upgrade.UpdateResult, error) {
		requests.Add(1)
		return original(ctx, version)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.Check(context.Background(), false); require.NoError(t, err) }()
	}
	wg.Wait()
	require.EqualValues(t, 1, requests.Load())
	_, err := s.Check(context.Background(), true)
	require.NoError(t, err)
	require.EqualValues(t, 2, requests.Load())
	previous := s.Now()
	s.Now = func() time.Time { return previous.Add(24 * time.Hour) }
	_, err = s.Check(context.Background(), false)
	require.NoError(t, err)
	require.EqualValues(t, 3, requests.Load())
}

func TestFailedCheckThrottlesAndPreservesLastSuccess(t *testing.T) {
	s := testService(t)
	_, err := s.Check(context.Background(), true)
	require.NoError(t, err)
	var calls int
	s.CLI = func(context.Context, string) (upgrade.UpdateResult, error) {
		calls++
		return upgrade.UpdateResult{Status: "failed"}, errors.New("offline")
	}
	r, err := s.Check(context.Background(), true)
	require.Error(t, err)
	require.Equal(t, "partial", r.Status)
	_, err = s.Check(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	var state cache
	require.NoError(t, readJSON(s.path(".json"), &state))
	require.Equal(t, "99.0.0", state.CLI.Target)
}

func TestCorruptSnapshotCannotProducePartialNotices(t *testing.T) {
	s := testService(t)
	require.NoError(t, os.MkdirAll(s.dir(), 0700))
	body, _ := json.Marshal(map[string]any{"attempt_at": 42, "cli": map[string]any{"target": "99.0.0"}, "cli_valid_until": s.Now().Add(resultLifetime)})
	require.NoError(t, os.WriteFile(s.path(".json"), body, 0600))
	var out, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&out, &stderr)
	Begin(s.ConfigDir, ctx, []string{"version"})
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	cmd := &cli.Command{Name: "aliyun", Run: func(*cli.Context, []string) error { return errors.New("original failure") }}
	cmd.Execute(ctx, nil)
	require.Contains(t, stderr.String(), "original failure")
	require.NotContains(t, stderr.String(), "aliyun upgrade")
}

func TestPreviousCacheFieldsAreIgnored(t *testing.T) {
	s := testService(t)
	legacy := map[string]any{
		"cli":             map[string]any{"target": "99.0.0", "minimum_supported": "100.0.0", "support_policy_status": "verified"},
		"cli_valid_until": s.Now().Add(resultLifetime),
		"policy_proof":    map[string]any{"valid_until": s.Now().Add(time.Hour)},
	}
	require.NoError(t, writeJSON(s.path(".json"), legacy))
	var snapshot cache
	require.NoError(t, readJSON(s.path(".json"), &snapshot))
	var out bytes.Buffer
	ctx := cli.NewCommandContext(&out, &out)
	notices, complete := s.provider(ctx, snapshot)()
	require.Len(t, notices, 1)
	require.Equal(t, "update", notices[0].Severity)
	require.Equal(t, "99.0.0", notices[0].Latest)
	complete(false)
	body, err := json.Marshal(snapshot)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &fields))
	require.NotContains(t, fields, "policy_proof")
	require.NoError(t, json.Unmarshal(fields["cli"], &fields))
	require.NotContains(t, fields, "minimum_supported")
	require.NotContains(t, fields, "support_policy_status")
}

func TestSuccessIsSilentAndErrorsRemainOneJSON(t *testing.T) {
	s := testService(t)
	_, err := s.Check(context.Background(), true)
	require.NoError(t, err)
	var state cache
	require.NoError(t, readJSON(s.path(".json"), &state))
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	for _, structured := range []bool{false, true} {
		var out, stderr bytes.Buffer
		ctx := cli.NewCommandContext(&out, &stderr)
		ctx.SetUpdateNoticeProvider(s.provider(ctx, state))
		cmd := &cli.Command{Name: "aliyun", Run: func(ctx *cli.Context, _ []string) error { fmt.Fprintln(ctx.Stdout(), "business result"); return nil }}
		cmd.Execute(ctx, nil)
		require.Equal(t, "business result\n", out.String())
		require.Empty(t, stderr.String())
		require.NoFileExists(t, s.path(".reminders.json"))
		cause := errors.New("business failed")
		cmd.Run = func(*cli.Context, []string) error { return cause }
		if structured {
			cmd.NormalizeError = func(*cli.Context, []string, error) error {
				return cli.NewAgentError(cli.AgentErrorEnvelope{Message: "business failed", ErrorCode: "OriginalCode", RequestId: "original-id", Recovery: cli.AgentErrorRecovery{Action: "retry", Hint: "original recovery"}}, cause).WithExitCode(7).WithSchemaVersion()
			}
		}
		cmd.Execute(ctx, nil)
		if structured {
			var envelope cli.AgentErrorEnvelope
			require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope))
			require.Equal(t, "OriginalCode", envelope.ErrorCode)
			require.Equal(t, "original-id", envelope.RequestId)
			require.Len(t, envelope.UpdateNotices, 1)
		} else {
			require.Contains(t, stderr.String(), "ERROR: business failed")
			require.Contains(t, stderr.String(), "aliyun upgrade")
		}
		// A fresh invocation of the same target within 72h has no notice.
		notices, done := s.provider(ctx, state)()
		require.Empty(t, notices)
		done(true)
		require.NoError(t, os.Remove(s.path(".reminders.json")))
	}
}

func TestReminderExpiryLocalUpgradeAndScopeChange(t *testing.T) {
	s := testService(t)
	require.NoError(t, os.MkdirAll(s.dir(), 0700))
	var out bytes.Buffer
	ctx := cli.NewCommandContext(&out, &out)
	state := cache{CLI: &upgrade.UpdateResult{Target: "99.0.0"}, CLIValidUntil: s.Now().Add(7 * 24 * time.Hour)}
	n, done := s.provider(ctx, state)()
	require.Len(t, n, 1)
	done(true)
	n, done = s.provider(ctx, state)()
	require.Empty(t, n)
	done(true)
	before := s.Now()
	s.Now = func() time.Time { return before.Add(72 * time.Hour) }
	n, done = s.provider(ctx, state)()
	require.Len(t, n, 1)
	done(true)
	oldVersion := cli.Version
	cli.Version = "99.0.0"
	t.Cleanup(func() { cli.Version = oldVersion })
	n, done = s.provider(ctx, state)()
	require.Empty(t, n)
	done(true)
	cli.Version = oldVersion
	t.Setenv("ALIBABA_CLOUD_ENDPOINT_SUFFIX", "aliyuncs.alibaba")
	n, done = s.provider(ctx, state)()
	require.Empty(t, n)
	done(true)
}

func TestScheduleLeaseAndDisabledBegin(t *testing.T) {
	s := testService(t)
	var starts int
	s.Spawn = func(_ string, args, env []string) error {
		starts++
		require.Equal(t, WorkerCommand, args[0])
		return errors.New("cannot spawn")
	}
	for range 5 {
		s.schedule()
	}
	require.Equal(t, 1, starts)
	t.Setenv(updatenotifier.DisableEnv, "true")
	var out bytes.Buffer
	Begin(s.ConfigDir, cli.NewCommandContext(&out, &out), []string{"version"})
	require.Empty(t, out.String())
	require.True(t, Skip([]string{"--profile", "test", "upgrade", "--check"}))
	require.False(t, Skip([]string{"ecs", "DescribeInstances", "--Name", "plugin"}))
}
