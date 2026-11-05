//go:build linux || darwin || windows

package telemetry

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestTelemetryFileLockHelperProcess(t *testing.T) {
	if os.Getenv("ALIYUN_TELEMETRY_LOCK_TEST_CHILD") != "1" {
		return
	}
	unlock, err := acquireFileLock(os.Getenv("ALIYUN_TELEMETRY_LOCK_TEST_PATH"), 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer unlock()
	fmt.Fprintln(os.Stdout, "locked")
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestTelemetryFileLockAcrossProcesses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "telemetry.lock")
	cmd := exec.Command(os.Args[0], "-test.run=^TestTelemetryFileLockHelperProcess$")
	cmd.Env = append(os.Environ(), "ALIYUN_TELEMETRY_LOCK_TEST_CHILD=1", "ALIYUN_TELEMETRY_LOCK_TEST_PATH="+path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	t.Cleanup(func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	ready := make(chan bool, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		ready <- scanner.Scan() && scanner.Text() == "locked"
	}()
	select {
	case ok := <-ready:
		if !ok {
			_ = stdin.Close()
			_ = cmd.Wait()
			waited = true
			t.Fatalf("lock holder did not start: %s", stderr.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("lock holder did not become ready")
	}
	for _, timeout := range []time.Duration{0, 20 * time.Millisecond} {
		unlock, err := acquireFileLock(path, timeout)
		if unlock != nil {
			unlock()
			t.Fatal("acquired a lock held by another process")
		}
		if !errors.Is(err, ErrLockBusy) {
			t.Fatalf("expected ErrLockBusy, got %v", err)
		}
	}
	// Kill bypasses Go defers: releasing this lock must be the OS's work.
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	waited = true
	unlock, err := acquireFileLock(path, 2*time.Second)
	if err != nil {
		t.Fatalf("lock was not released after holder exit: %v", err)
	}
	unlock()
	unlock() // Repeated cleanup must not close or unlock a reused descriptor.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stable lock file must remain: %v", err)
	}
	unlock, err = acquireFileLock(path, 0)
	if err != nil {
		t.Fatalf("lock was not released: %v", err)
	}
	unlock()
}

func TestTelemetryFileLockWaitAndOpenError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "telemetry.lock")
	unlock, err := acquireFileLock(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	timer := time.AfterFunc(20*time.Millisecond, unlock)
	defer timer.Stop()
	next, err := acquireFileLock(path, time.Second)
	if err != nil {
		t.Fatalf("waiting acquisition failed: %v", err)
	}
	next()
	next, err = acquireFileLock(filepath.Join(path, "missing-parent", "lock"), 0)
	if next != nil || err == nil || errors.Is(err, ErrLockBusy) {
		t.Fatalf("open error must be distinct from contention: %v", err)
	}
}

func TestTelemetryAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "instance-id")
	for _, data := range []string{"first", "replacement content", ""} {
		if err := atomicWriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != data {
			t.Fatalf("unexpected contents %q, error %v", got, err)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("unexpected file permissions: %v, %v", info, err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "instance-id" {
		t.Fatalf("temporary files remained: %v, %v", entries, err)
	}
}

func TestTelemetryAtomicWriteFailurePreservesTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "instance-id")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	cause := errors.New("replacement failed")
	err := atomicWriteFileWithReplace(path, []byte("replacement"), 0600, func(source, target string) error {
		if filepath.Dir(source) != dir || target != path {
			t.Fatalf("replacement paths were not in the target directory")
		}
		got, err := os.ReadFile(source)
		if err != nil || string(got) != "replacement" {
			t.Fatalf("replacement file was incomplete: %q, %v", got, err)
		}
		return cause
	})
	if !errors.Is(err, cause) {
		t.Fatalf("expected replacement failure, got %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "original" {
		t.Fatalf("failed replacement damaged target: %q, %v", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("failed replacement left temporary files: %v, %v", entries, err)
	}
}
