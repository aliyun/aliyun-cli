package updatecheck

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/cli/trust"
	"github.com/aliyun/aliyun-cli/v3/cli/upgrade"
	"github.com/aliyun/aliyun-cli/v3/internal/localstate"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/pluginsettings"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/updatenotifier"
	"github.com/stretchr/testify/require"
)

func TestPluginNoticesRequireMatchingOwnerAndValidCache(t *testing.T) {
	previous := cli.Version
	cli.Version = "3.5.3"
	t.Cleanup(func() { cli.Version = previous })
	for _, scenario := range []string{"update", "other_owner", "passthrough", "removed", "already_upgraded", "expired", "changed_proof", "legacy_proof", "timestamp_only_proof"} {
		t.Run(scenario, func(t *testing.T) {
			s := testService(t)
			require.NoError(t, os.MkdirAll(s.dir(), 0700))
			ctx := cli.NewCommandContext(io.Discard, io.Discard)
			ctx.SetUpdateOwner("meta", "example")
			manifest := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
				"example": {Name: "example", Version: "1.0.0", MinCliVersion: " \t3.6.0\n"},
			}}
			statePath := filepath.Join(t.TempDir(), "trust-state.json")
			state := []byte(`{"root_version":1,"keys_version":1}`)
			require.NoError(t, os.WriteFile(statePath, state, 0600))
			sum := sha256.Sum256(state)
			snapshot := cache{
				Plugins: &plugin.UpdateReport{Status: "complete", Plugins: []plugin.UpdateResult{{
					Name: "example", Current: "0.9.0", Latest: "2.0.0", Status: "update_available", Command: "aliyun plugin update --name example",
				}}},
				PluginProof: trust.CacheProof{SchemaVersion: 2, StatePath: statePath, StateHash: hex.EncodeToString(sum[:]), ValidUntil: s.Now().Add(time.Hour)},
			}
			switch scenario {
			case "other_owner":
				ctx.SetUpdateOwner("meta", "unrelated")
			case "passthrough":
				ctx.SetUpdateOwner("passthrough", "example")
			case "removed":
				delete(manifest.Plugins, "example")
			case "already_upgraded":
				p := manifest.Plugins["example"]
				p.Version = "2.0.0"
				manifest.Plugins["example"] = p
			case "expired":
				snapshot.PluginProof.ValidUntil = s.Now()
			case "changed_proof":
				require.NoError(t, os.WriteFile(statePath, []byte(`{"root_version":2}`), 0600))
			case "legacy_proof":
				snapshot.PluginProof.SchemaVersion = 0
			case "timestamp_only_proof":
				snapshot.PluginProof.SchemaVersion = 1
			}
			require.NoError(t, writeJSON(filepath.Join(s.Scope.Root, "manifest.json"), manifest))
			notices, complete := s.provider(ctx, snapshot)()
			defer complete(false)
			if scenario != "update" {
				require.Empty(t, notices)
				return
			}
			require.Len(t, notices, 1)
			require.Equal(t, "1.0.0", notices[0].Current, "use the installed version, not the older snapshot")
			require.Equal(t, "example", notices[0].Component)
			require.Equal(t, "2.0.0", notices[0].Latest)
			require.Contains(t, notices[0].Message, "Upgrade the CLI first")
		})
	}
}

func TestReminderHistoryOnlyCommitsRenderedTargets(t *testing.T) {
	s := testService(t)
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	snapshot := cache{CLI: &upgrade.UpdateResult{Target: "99.0.0"}, CLIValidUntil: s.Now().Add(time.Hour)}
	require.NoError(t, os.MkdirAll(s.dir(), 0700))
	provider := s.provider(ctx, snapshot)
	notices, complete := provider()
	require.Len(t, notices, 1)
	// Another caller must not race the in-progress reminder transaction.
	other, done := s.provider(ctx, snapshot)()
	require.Empty(t, other)
	done(true)
	complete(false)
	require.NoFileExists(t, s.path(".reminders.json"))
	notices, done = provider()
	require.Empty(t, notices, "one provider is consumed only once")
	done(true)
	notices, complete = s.provider(ctx, snapshot)()
	require.Len(t, notices, 1, "failed rendering must not postpone the next reminder")
	complete(true)
	snapshot.CLI.Target = "99.1.0"
	notices, complete = s.provider(ctx, snapshot)()
	require.Len(t, notices, 1, "a new target has its own reminder interval")
	complete(true)
	for _, history := range []string{"broken", "null"} {
		require.NoError(t, os.WriteFile(s.path(".reminders.json"), []byte(history), 0600))
		notices, complete = s.provider(ctx, snapshot)()
		if history == "broken" {
			require.Empty(t, notices, "invalid history must fail silently")
		} else {
			require.Len(t, notices, 1)
		}
		complete(false)
	}
	t.Setenv(updatenotifier.DisableEnv, "1")
	notices, done = s.provider(ctx, snapshot)()
	require.Empty(t, notices)
	done(true)
}

type offlineUpdateTransport func(*http.Request) (*http.Response, error)

func (f offlineUpdateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWorkerValidationAndExplicitCheckFailure(t *testing.T) {
	s := testService(t)
	for _, args := range [][]string{nil, {"--wrong", "dir", "--scope", "key"}, {"--config-dir", "relative", "--scope", "key"}} {
		require.Zero(t, RunWorker(args))
	}
	args := []string{"--config-dir", s.ConfigDir, "--scope", s.key()}
	require.Zero(t, RunWorker([]string{"--config-dir", s.ConfigDir, "--scope", "different-scope"}))
	// A recently checked cache makes the real worker return without networking.
	before := cache{AttemptAt: time.Now().UTC()}
	require.NoError(t, writeJSON(s.path(".json"), before))
	require.Zero(t, RunWorker(args))
	var after cache
	require.NoError(t, readJSON(s.path(".json"), &after))
	require.True(t, before.AttemptAt.Equal(after.AttemptAt))
	t.Setenv(updatenotifier.DisableEnv, "1")
	require.Zero(t, RunWorker(args))
	t.Setenv(updatenotifier.DisableEnv, "0")
	previous := http.DefaultTransport
	http.DefaultTransport = offlineUpdateTransport(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("offline fixture")
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	var out, stderr bytes.Buffer
	err := RunCheck(s.ConfigDir, cli.NewCommandContext(&out, &stderr))
	require.ErrorContains(t, err, "CLI version check failed")
	require.Contains(t, out.String(), "Alibaba Cloud CLI")
	require.Empty(t, stderr.String(), "the command framework renders the returned error")
}

func TestInvalidSettingsDisableAutomaticDiagnostics(t *testing.T) {
	s := testService(t)
	settingsPath := pluginsettings.GetConfigFilePath(s.ConfigDir)
	require.NoError(t, os.MkdirAll(settingsPath, 0700))
	_, err := New(s.ConfigDir)
	require.Error(t, err)
	var out, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&out, &stderr)
	require.NotPanics(t, func() { Begin(s.ConfigDir, ctx, []string{"ecs", "DescribeRegions"}) })
	require.Empty(t, out.String())
	require.Empty(t, stderr.String())
	require.Error(t, RunCheck(s.ConfigDir, ctx))
	require.NoError(t, os.Remove(settingsPath))
	t.Setenv("ALIBABA_CLOUD_ENDPOINT_SUFFIX", "invalid.example")
	_, err = New(s.ConfigDir)
	require.ErrorContains(t, err, "invalid endpoint suffix")
}

func TestCacheFailuresDoNotStartChecksOrLoseErrors(t *testing.T) {
	for _, scenario := range []string{"blocked_cache", "blocked_result", "busy_lock", "save_after_check"} {
		t.Run(scenario, func(t *testing.T) {
			s := testService(t)
			require.NoError(t, os.MkdirAll(s.ConfigDir, 0700))
			if scenario == "blocked_cache" {
				require.NoError(t, os.WriteFile(s.dir(), []byte("file"), 0600))
			} else {
				require.NoError(t, os.MkdirAll(s.dir(), 0700))
			}
			ctx := context.Background()
			if scenario == "blocked_result" {
				require.NoError(t, os.Mkdir(s.path(".json"), 0700))
			} else if scenario == "busy_lock" {
				unlock, err := localstate.Lock(nil, s.path(".check.lock"))
				require.NoError(t, err)
				defer unlock()
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			} else if scenario == "save_after_check" {
				original := s.CLI
				s.CLI = func(ctx context.Context, version string) (upgrade.UpdateResult, error) {
					if err := os.Remove(s.path(".json")); err != nil {
						return upgrade.UpdateResult{}, err
					}
					if err := os.Mkdir(s.path(".json"), 0700); err != nil {
						return upgrade.UpdateResult{}, err
					}
					return original(ctx, version)
				}
			}
			_, err := s.Check(ctx, true)
			require.Error(t, err)
			if scenario == "busy_lock" {
				require.ErrorIs(t, err, context.Canceled)
			} else if scenario == "blocked_cache" {
				require.ErrorContains(t, err, "cache is not writable")
			} else {
				require.ErrorContains(t, err, "cache cannot be saved")
			}
		})
	}
}

func TestStateParsingAndReportWriteFailures(t *testing.T) {
	s := testService(t)
	path := filepath.Join(t.TempDir(), "oversized.json")
	require.NoError(t, os.WriteFile(path, bytes.Repeat([]byte("x"), (2<<20)+1), 0600))
	var state cache
	require.ErrorContains(t, readJSON(path, &state), "too large")
	require.Error(t, readJSON(filepath.Dir(path), &state))
	require.Error(t, writeJSON(path, func() {}))
	ctx := cli.NewCommandContext(updateReportFailWriter{}, io.Discard)
	require.ErrorIs(t, s.runCheck(ctx), io.ErrClosedPipe)
}

type updateReportFailWriter struct{}

func (updateReportFailWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
