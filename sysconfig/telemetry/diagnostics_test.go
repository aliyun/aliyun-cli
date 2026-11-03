package telemetry

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const diagnosticTestEventID = "1f8b85cd-e427-493b-a663-601a1a8c9df4"

func readDiagnosticEntries(t *testing.T, path string) []map[string]any {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	var entries []map[string]any
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &entry), "every append must remain a complete JSON record")
		entries = append(entries, entry)
	}
	require.NoError(t, scanner.Err())
	return entries
}

func TestDiagnosticRecordContainsOnlySafeFields(t *testing.T) {
	dir := t.TempDir()
	record := diagnosticRecord{
		Stage: "upload", Status: "failed", Reason: "http_error", ErrorKind: "network_error",
		EventID: diagnosticTestEventID, Pipeline: "inner", HTTPStatus: 503, DurationMs: 42,
	}
	writeDiagnostic(dir, record)
	path := filepath.Join(dir, "logs", diagnosticLogName)
	entries := readDiagnosticEntries(t, path)
	require.Len(t, entries, 1)
	entry := entries[0]
	stamp, err := time.Parse(time.RFC3339Nano, entry["timestamp"].(string))
	require.NoError(t, err)
	_, offset := stamp.Zone()
	require.Zero(t, offset)
	require.True(t, strings.HasSuffix(entry["timestamp"].(string), "Z"))
	require.Equal(t, map[string]any{
		"timestamp": entry["timestamp"], "pid": float64(os.Getpid()),
		"stage": "upload", "status": "failed", "reason": "http_error", "errorKind": "network_error",
		"eventId": diagnosticTestEventID, "pipeline": "inner", "httpStatus": float64(503), "durationMs": float64(42),
	}, entry)
	if runtime.GOOS != "windows" {
		for _, p := range []string{path, path + ".lock"} {
			info, err := os.Stat(p)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0600), info.Mode().Perm())
		}
		info, err := os.Stat(filepath.Dir(path))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0700), info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(string(data), "\n"))

	secret := "oss://private-bucket/secret?accessKey=private-value\nforged-log"
	writeDiagnostic(dir, diagnosticRecord{
		Stage: "upload", Status: "failed", Reason: secret, ErrorKind: secret,
		EventID: secret, Pipeline: secret, HTTPStatus: 999, DurationMs: -1,
	})
	entries = readDiagnosticEntries(t, path)
	require.Len(t, entries, 2)
	require.Equal(t, map[string]any{
		"timestamp": entries[1]["timestamp"], "pid": float64(os.Getpid()), "stage": "upload", "status": "failed",
	}, entries[1])
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(data), "private")
	for _, record := range []diagnosticRecord{
		{Stage: secret, Status: "failed"}, {Stage: "upload", Status: secret}, {},
	} {
		writeDiagnostic(dir, record)
	}
	require.Len(t, readDiagnosticEntries(t, path), 2, "unknown stages and statuses must be discarded")
}

func TestDiagnosticAllowedEnumsAndIdentifiers(t *testing.T) {
	for _, stage := range []string{"collection", "event_build", "cache_write", "upload_spawn", "cache_read", "cache_remove", "upload"} {
		for _, status := range []string{"started", "succeeded", "failed", "skipped"} {
			record := diagnosticRecord{Stage: stage, Status: status}
			require.True(t, sanitizeDiagnostic(&record))
			require.Equal(t, stage, record.Stage)
			require.Equal(t, status, record.Status)
		}
	}
	for _, reason := range []string{"management_command", "invalid_config", "disabled", "scope_inactive", "instance_id_failed", "panic", "marshal_failed", "invalid_cache_path", "cache_empty", "invalid_event", "pipeline_mismatch", "executable_unavailable", "process_start_failed", "request_invalid", "http_error", "missing_arguments", "invalid_pipeline"} {
		record := diagnosticRecord{Stage: "collection", Status: "skipped", Reason: reason}
		require.True(t, sanitizeDiagnostic(&record))
		require.Equal(t, reason, record.Reason)
	}
	for _, kind := range []string{"permission_denied", "not_found", "lock_busy", "timeout", "network_error", "invalid_json", "io_error", "other"} {
		record := diagnosticRecord{Stage: "collection", Status: "failed", ErrorKind: kind}
		require.True(t, sanitizeDiagnostic(&record))
		require.Equal(t, kind, record.ErrorKind)
	}
	for _, id := range []string{diagnosticTestEventID, strings.ToUpper(diagnosticTestEventID)} {
		require.True(t, validDiagnosticEventID(id))
	}
	for _, id := range []string{"", "secret", strings.ReplaceAll(diagnosticTestEventID, "-", ""), "1f8b85cd_e427-493b-a663-601a1a8c9df4", "zf8b85cd-e427-493b-a663-601a1a8c9df4"} {
		require.False(t, validDiagnosticEventID(id))
	}
	for _, status := range []int{100, 599} {
		record := diagnosticRecord{Stage: "upload", Status: "succeeded", HTTPStatus: status, Pipeline: "public"}
		require.True(t, sanitizeDiagnostic(&record))
		require.Equal(t, status, record.HTTPStatus)
		require.Equal(t, "public", record.Pipeline)
	}
}

func TestDiagnosticRotationRetainsFiveBackups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "logs", diagnosticLogName)
	const limit = 550
	for i := 0; i < 50; i++ {
		writeDiagnosticWithLimits(dir, diagnosticRecord{
			Stage: "upload", Status: "succeeded", EventID: fmt.Sprintf("%08x-e427-493b-a663-601a1a8c9df4", i),
		}, limit, diagnosticBackups)
	}
	files, err := filepath.Glob(path + "*")
	require.NoError(t, err)
	require.Len(t, files, 7, "one current file, five backups, and the stable lock")
	lastID := ""
	for i := diagnosticBackups; i >= 0; i-- {
		p := path
		if i > 0 {
			p += "." + strconv.Itoa(i)
		}
		info, err := os.Stat(p)
		require.NoError(t, err)
		require.LessOrEqual(t, info.Size(), int64(limit))
		entries := readDiagnosticEntries(t, p)
		require.NotEmpty(t, entries)
		for _, entry := range entries {
			id := entry["eventId"].(string)
			require.Greater(t, id, lastID, "rotation must preserve event ordering")
			lastID = id
		}
		if runtime.GOOS != "windows" {
			require.Equal(t, os.FileMode(0600), info.Mode().Perm())
		}
	}
	require.Equal(t, "00000031-e427-493b-a663-601a1a8c9df4", lastID)
	require.NoFileExists(t, path+".6")
}

func TestDiagnosticConcurrentAppendsRemainComplete(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			writeDiagnostic(dir, diagnosticRecord{
				Stage: "cache_write", Status: "succeeded", EventID: fmt.Sprintf("%08x-e427-493b-a663-601a1a8c9df4", i),
			})
		}(i)
	}
	wg.Wait()
	entries := readDiagnosticEntries(t, filepath.Join(dir, "logs", diagnosticLogName))
	require.Len(t, entries, 80)
	seen := make(map[any]bool)
	for _, entry := range entries {
		require.False(t, seen[entry["eventId"]])
		seen[entry["eventId"]] = true
	}
}

func TestDiagnosticHelperProcess(t *testing.T) {
	if os.Getenv("ALIYUN_TELEMETRY_DIAGNOSTIC_CHILD") != "1" {
		return
	}
	for i := 0; i < 8; i++ {
		writeDiagnostic(os.Getenv("ALIYUN_TELEMETRY_DIAGNOSTIC_DIR"), diagnosticRecord{Stage: "upload", Status: "succeeded"})
		time.Sleep(2 * time.Millisecond)
	}
}

func TestDiagnosticConcurrentProcesses(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("cross-process locking is supported on Linux, macOS, and Windows")
	}
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	commands := make([]*exec.Cmd, 2)
	for i := range commands {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDiagnosticHelperProcess$")
		cmd.Env = append(os.Environ(), "ALIYUN_TELEMETRY_DIAGNOSTIC_CHILD=1", "ALIYUN_TELEMETRY_DIAGNOSTIC_DIR="+dir)
		require.NoError(t, cmd.Start())
		commands[i] = cmd
		t.Cleanup(func() { _ = cmd.Process.Kill() })
	}
	for _, cmd := range commands {
		require.NoError(t, cmd.Wait())
	}
	entries := readDiagnosticEntries(t, filepath.Join(dir, "logs", diagnosticLogName))
	counts := make(map[int]int)
	for _, entry := range entries {
		counts[int(entry["pid"].(float64))]++
	}
	for _, cmd := range commands {
		require.Positive(t, counts[cmd.Process.Pid], "each writer must append parseable records")
		require.LessOrEqual(t, counts[cmd.Process.Pid], 8)
	}
}

func TestDiagnosticBusyLockIsBestEffort(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("file locking unsupported")
	}
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	require.NoError(t, os.Mkdir(logDir, 0700))
	path := filepath.Join(logDir, diagnosticLogName)
	unlock, err := acquireFileLock(path+".lock", 0)
	require.NoError(t, err)
	defer unlock()
	start := time.Now()
	writeDiagnostic(dir, diagnosticRecord{Stage: "collection", Status: "started"})
	require.Less(t, time.Since(start), time.Second)
	require.NoFileExists(t, path)
	unlock()
	writeDiagnostic(dir, diagnosticRecord{Stage: "collection", Status: "started"})
	require.Len(t, readDiagnosticEntries(t, path), 1)
}

func TestDiagnosticUnsafeFilesAreIgnored(t *testing.T) {
	for _, name := range []string{"logs", "logs/telemetry.log", "logs/telemetry.log.lock", "logs/telemetry.log.1"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if name == "logs" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "logs"), []byte("existing"), 0600))
			} else {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, name), 0700))
			}
			require.NotPanics(t, func() { writeDiagnostic(dir, diagnosticRecord{Stage: "collection", Status: "started"}) })
			if name == "logs" {
				data, err := os.ReadFile(filepath.Join(dir, "logs"))
				require.NoError(t, err)
				require.Equal(t, "existing", string(data))
			} else {
				require.DirExists(t, filepath.Join(dir, name))
			}
		})
	}
	for _, limits := range []struct {
		size    int64
		backups int
	}{{0, 5}, {100, 0}, {100, 6}, {1, 5}} {
		dir := t.TempDir()
		writeDiagnosticWithLimits(dir, diagnosticRecord{Stage: "collection", Status: "started"}, limits.size, limits.backups)
		require.NoDirExists(t, filepath.Join(dir, "logs"))
	}
	// An unresolved config path must never turn into a relative ./logs path.
	writeDiagnostic("", diagnosticRecord{Stage: "collection", Status: "started"})
}

func TestDiagnosticDoesNotFollowSymlinks(t *testing.T) {
	for _, name := range []string{"logs", "logs/telemetry.log", "logs/telemetry.log.lock", "logs/telemetry.log.1"} {
		t.Run(name, func(t *testing.T) {
			dir, outside := t.TempDir(), t.TempDir()
			target := filepath.Join(outside, "protected")
			if name == "logs" {
				target = outside
			} else {
				require.NoError(t, os.Mkdir(filepath.Join(dir, "logs"), 0700))
				require.NoError(t, os.WriteFile(target, []byte("protected"), 0600))
			}
			if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			require.NotPanics(t, func() { writeDiagnostic(dir, diagnosticRecord{Stage: "collection", Status: "started"}) })
			if name == "logs" {
				entries, err := os.ReadDir(outside)
				require.NoError(t, err)
				require.Empty(t, entries)
			} else {
				data, err := os.ReadFile(target)
				require.NoError(t, err)
				require.Equal(t, "protected", string(data))
			}
		})
	}
}

func TestDiagnosticUnwritableDirectoryIsSilent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not deny directory writes on Windows")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })
	probe, err := os.CreateTemp(dir, "permission-probe-*")
	if err == nil {
		_ = probe.Close()
		_ = os.Remove(probe.Name())
		t.Skip("current user or filesystem permits writes despite chmod")
	}
	require.ErrorIs(t, err, os.ErrPermission)
	require.NotPanics(t, func() { writeDiagnostic(dir, diagnosticRecord{Stage: "collection", Status: "started"}) })
	require.NoDirExists(t, filepath.Join(dir, "logs"))
}

type diagnosticNoMessageError struct{}

func (diagnosticNoMessageError) Error() string { panic("error text must not be read") }

func TestDiagnosticErrorKindDoesNotExposeErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		kind string
	}{
		{"none", nil, ""},
		{"permission", fmt.Errorf("private-path: %w", os.ErrPermission), "permission_denied"},
		{"missing", &os.PathError{Op: "open", Path: "secret", Err: os.ErrNotExist}, "not_found"},
		{"lock", fmt.Errorf("private-path: %w", ErrLockBusy), "lock_busy"},
		{"deadline", context.DeadlineExceeded, "timeout"},
		{"network-timeout", &net.DNSError{Err: "private-host", IsTimeout: true}, "timeout"},
		{"network", &net.OpError{Op: "private-operation", Err: errors.New("private-data")}, "network_error"},
		{"json-syntax", &json.SyntaxError{Offset: 2}, "invalid_json"},
		{"json-type", &json.UnmarshalTypeError{Value: "private-value"}, "invalid_json"},
		{"path", &os.PathError{Op: "private-op", Path: "secret", Err: errors.New("disk failure")}, "io_error"},
		{"link", &os.LinkError{Op: "private-op", Old: "secret", New: "secret", Err: errors.New("failure")}, "io_error"},
		{"short-write", io.ErrShortWrite, "io_error"},
		{"unexpected-eof", io.ErrUnexpectedEOF, "io_error"},
		{"unknown-without-text", diagnosticNoMessageError{}, "other"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.kind, diagnosticErrorKind(test.err))
		})
	}
}
