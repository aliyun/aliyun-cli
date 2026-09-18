package lib

import (
	"errors"
	"fmt"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOSSSafetyPolicyBeforeExecution(t *testing.T) {
	clearEndpointTestEnv(t)
	for _, tc := range []struct {
		name, enabled, rule, approval string
		allowed, confirmation         bool
	}{
		{name: "deny force", enabled: "true", rule: "oss:*=deny"},
		{name: "deny yes", enabled: "true", rule: "oss:rm=deny", approval: "--yes"},
		{name: "invalid environment", enabled: "garbage", rule: "oss:*=deny"},
		{name: "confirm force", enabled: "true", rule: "oss:rm=confirm", confirmation: true},
		{name: "confirm yes", enabled: "true", rule: "oss:rm=confirm", approval: "--yes", allowed: true},
		{name: "confirm shorthand", enabled: "true", rule: "oss:rm=confirm", approval: "-y", allowed: true},
		{name: "allow", enabled: "true", rule: "oss:rm=allow", allowed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ALIBABA_CLOUD_SAFETY_POLICY_ENABLED", tc.enabled)
			t.Setenv("ALIBABA_CLOUD_SAFETY_POLICY_RULES", tc.rule)
			t.Setenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM", "false")
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) }))
			defer server.Close()
			args := []string{"oss://bucket/key", "--force", "--cli-ai-mode", "--cli-non-interactive"}
			if tc.approval != "" {
				args = append(args, tc.approval)
			}
			_, err := phaseCInvoke(t, removeCommand.command, server.URL, args...)
			if tc.allowed {
				require.NoError(t, err)
				require.EqualValues(t, 2, calls.Load())
			} else {
				require.Error(t, err)
				require.Zero(t, calls.Load())
				require.Equal(t, tc.confirmation, errors.Is(err, errConfirmationRequired))
			}
		})
	}
}

func TestMachineVersionListRetryBudget(t *testing.T) {
	clearEndpointTestEnv(t)
	oldSleep := retrySleep
	retrySleep = func(time.Duration) {}
	t.Cleanup(func() { retrySleep = oldSleep })
	for _, format := range []string{"json", "jsonl"} {
		for _, tc := range []struct {
			name                       string
			status, failures, attempts int
			success                    bool
		}{
			{"transient", 503, 1, 2, true}, {"permanent", 403, 1, 1, false}, {"exhausted", 503, 3, 2, false},
		} {
			t.Run(format+"/"+tc.name, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if int(calls.Add(1)) <= tc.failures {
						w.WriteHeader(tc.status)
						fmt.Fprint(w, `<Error><Code>TestError</Code><Message>failed</Message></Error>`)
						return
					}
					io.WriteString(w, `<ListVersionsResult><Version><Key>key</Key><VersionId>version</VersionId><Size>1</Size></Version></ListVersionsResult>`)
				}))
				defer server.Close()
				out, err := phaseCInvoke(t, listCommand.command, server.URL, "oss://bucket", "--all-versions", "--cli-output="+format, "--retry-count=2")
				require.EqualValues(t, tc.attempts, calls.Load())
				if tc.success {
					require.NoError(t, err)
					require.Contains(t, out, "version")
				} else {
					require.Error(t, err)
				}
			})
		}
	}
}

func TestRMMonitorConcurrentProgress(t *testing.T) {
	var monitor RMMonitor
	monitor.init()
	monitor.setOP(allType)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for i := 0; i < 1000; i++ {
			monitor.updateScanNum(1)
			monitor.updateScanUploadIdNum(1)
			monitor.updateObjectNum(1)
			monitor.updateUploadIdNum(1)
		}
		monitor.setScanEnd()
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 1000; i++ {
			monitor.progressBar(false, normalExit)
		}
	}()
	workers.Wait()
	require.Contains(t, monitor.progressBar(true, normalExit), "1000 objects")
}

func TestOSSPolicyPrecedesCredentialLoading(t *testing.T) {
	clearEndpointTestEnv(t)
	dir := t.TempDir()
	policyPath := safety.GetPolicyFilePath(dir)
	require.NoError(t, os.WriteFile(policyPath, []byte(`{"enabled":true,"rules":[{"pattern":"oss:rm","action":"deny"}]}`), 0600))
	t.Setenv("ALIBABA_CLOUD_SAFETY_POLICY_ENABLED", "true")
	t.Setenv("ALIBABA_CLOUD_SAFETY_POLICY_RULES", "")
	require.NoError(t, os.Unsetenv("ALIBABA_CLOUD_SAFETY_POLICY_RULES"))
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte("broken credential configuration"), 0600))
	for _, mode := range []string{"--force", "--cli-validate", "--cli-plan"} {
		ctx := cli.NewCommandContext(io.Discard, io.Discard)
		ctx.SetCommand(NewCommandBridge(removeCommand.command))
		err := parseAndRunCommandFromCli(ctx, []string{"oss://bucket/key", "--config-path", path, mode, "--cli-ai-mode"}, &removeCommand.command)
		require.ErrorContains(t, err, "blocked by safety policy")
	}
}
