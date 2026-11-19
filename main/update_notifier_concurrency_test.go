//go:build linux || darwin || windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/internal/updatecheck"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/updatenotifier"
	"github.com/stretchr/testify/require"
)

func TestUpdateWorkerNetworkDoesNotBlockForegroundPluginCommand(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".aliyun")
	root := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(root, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"current":"default","profiles":[{"name":"default","mode":"Anonymous","language":"en","region_id":"cn-hangzhou"}]}`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"plugins":{"aliyun-cli-updatefixture":{"name":"aliyun-cli-updatefixture","version":"1.0.0","type":"go","command":"updatefixture"}}}`), 0600))
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	var requests atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect && r.Host == "api.aliyun.com:443" && requests.Add(1) == 1 {
			close(started)
			select {
			case <-release:
			case <-r.Context().Done():
			}
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(proxy.Close)
	extra := []string{updatenotifier.DisableEnv + "=0", "HTTPS_PROXY=" + proxy.URL, "HTTP_PROXY=" + proxy.URL, "NO_PROXY="}
	scope, err := json.Marshal([]string{root, "https://aliyuncli.alicdn.com/plugins-v2", "cn", ""})
	require.NoError(t, err)
	sum := sha256.Sum256(scope)
	workerDone, foregroundDone := make(chan struct{}), make(chan struct{})
	worker := make(chan telemetryPermissionResult, 1)
	foreground := make(chan telemetryPermissionResult, 1)
	foregroundStarted := false
	t.Cleanup(func() {
		unblock()
		select {
		case <-workerDone:
		case <-time.After(5 * time.Second):
			t.Error("update worker did not stop")
		}
		if foregroundStarted {
			select {
			case <-foregroundDone:
			case <-time.After(5 * time.Second):
				t.Error("foreground plugin command did not stop")
			}
		}
	})
	go func() {
		defer close(workerDone)
		worker <- runTelemetryPermissionCommand(t, home, true, []string{updatecheck.WorkerCommand, "--config-dir", dir, "--scope", hex.EncodeToString(sum[:16])}, extra...)
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not start plugin metadata networking")
	}
	foregroundStarted = true
	go func() {
		defer close(foregroundDone)
		foreground <- runTelemetryPermissionCommand(t, home, true, []string{"plugin", "list-remote"}, extra...)
	}()
	select {
	case got := <-foreground:
		require.Equal(t, 1, got.exitCode)
		require.Empty(t, got.stdout)
		require.Contains(t, got.stderr, "Bad Gateway")
	case <-time.After(2 * time.Second):
		t.Fatal("foreground command waited for the held background network request")
	}
	unblock()
	select {
	case got := <-worker:
		require.Equal(t, 0, got.exitCode)
		require.Empty(t, got.stdout)
		require.Empty(t, got.stderr)
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not finish after release")
	}
}
