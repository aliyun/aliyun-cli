package lib

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMonitorConcurrentFailureState(t *testing.T) {
	var rm RMMonitor
	rm.init()
	var cp CPMonitor
	cp.init(operationTypePut)
	cause := errors.New("scan failed")
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Go(func() {
			for j := 0; j < 100; j++ {
				rm.updateScanNum(1)
				rm.setScanError(cause)
				rm.updateRemovedBucket("bucket")
				rm.setOP(1)
				rm.getOP()
				cp.updateScanNum(1)
				cp.setScanError(cause)
			}
		})
	}
	workers.Wait()
	require.EqualValues(t, 800, rm.totalObjectNum)
	require.EqualValues(t, 800, cp.totalNum)
	require.ErrorIs(t, rm.seekAheadError, cause)
	require.ErrorIs(t, cp.seekAheadError, cause)
	require.True(t, rm.seekAheadEnd)
	require.True(t, cp.seekAheadEnd)
	require.Equal(t, "bucket", rm.removedBucket)
}

func TestFailureManifestPreservesWriteErrors(t *testing.T) {
	m, err := newFailureManifest(filepath.Join(t.TempDir(), "failures.jsonl"))
	require.NoError(t, err)
	require.NoError(t, m.file.Close())
	m.write(failureItem{Type: "failed_item"})
	first := m.err
	require.Error(t, first)
	m.write(failureItem{Type: "failed_item"})
	require.Equal(t, first, m.err)
	cause := errors.New("transfer failed")
	err = m.finish(cause)
	require.ErrorIs(t, err, cause)
	require.ErrorIs(t, err, first)
}
func TestLogWriterRedactsAndReportsWriteFailure(t *testing.T) {
	old := activeMachine
	activeMachine = &machineInvocation{secrets: []string{"secret"}}
	defer func() { activeMachine = old }()
	path := filepath.Join(t.TempDir(), "log")
	f, err := os.Create(path)
	require.NoError(t, err)
	w := redactingLogWriter{f}
	input := []byte("credential=secret")
	n, err := w.Write(input)
	require.NoError(t, err)
	require.Equal(t, len(input), n)
	require.NoError(t, f.Close())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(data), "secret")
	require.Contains(t, string(data), "[REDACTED]")
	n, err = w.Write(input)
	require.Error(t, err)
	require.Zero(t, n)
	cause := errors.New("private")
	sanitized := &sanitizedOSSError{cause, "safe"}
	require.Equal(t, "safe", sanitized.Error())
	require.ErrorIs(t, sanitized, cause)
}

func TestDiagnosticRedactsCompactCredentialFlags(t *testing.T) {
	for _, arg := range []string{"-iid", "-k=secret", "-ttoken"} {
		got := redactCommandLineArgs([]string{arg})
		require.NotEqual(t, arg, got[0])
		require.Contains(t, got[0], "REDACTED")
	}
	require.Equal(t, "https://example.invalid/?public=yes", redactSignedURL("https://example.invalid/?public=yes"))
	require.Equal(t, "https://%invalid/?Signature=secret", redactSignedURL("https://%invalid/?Signature=secret"))
}
