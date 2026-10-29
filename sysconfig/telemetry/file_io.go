package telemetry

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrLockBusy means another process held the lock until the wait budget expired.
var ErrLockBusy = errors.New("telemetry file lock is busy")

// acquireFileLock locks a stable file. Its parent directory must already exist;
// callers must never remove or replace the lock file, including during purge.
// A nonpositive timeout makes a single nonblocking attempt.
func acquireFileLock(path string, timeout time.Duration) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		err = tryLockFile(f)
		if err == nil {
			var once sync.Once
			return func() {
				once.Do(func() {
					_ = unlockFile(f)
					_ = f.Close()
				})
			}, nil
		}
		if !errors.Is(err, ErrLockBusy) {
			_ = f.Close()
			return nil, fmt.Errorf("lock %s: %w", path, err)
		}
		remaining := time.Until(deadline)
		if timeout <= 0 || remaining <= 0 {
			_ = f.Close()
			return nil, fmt.Errorf("lock %s: %w", path, ErrLockBusy)
		}
		time.Sleep(min(remaining, 5*time.Millisecond))
		if !time.Now().Before(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("lock %s: %w", path, ErrLockBusy)
		}
	}
}

// atomicWriteFile publishes a complete file using a same-directory replacement.
// It does not create the parent directory or serialize concurrent writers.
func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	return atomicWriteFileWithReplace(path, data, mode, replaceFile)
}

func atomicWriteFileWithReplace(path string, data []byte, mode os.FileMode, replace func(string, string) error) error {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return err
	}
	tempPath := f.Name()
	published := false
	defer func() {
		_ = f.Close()
		if !published {
			_ = os.Remove(tempPath)
		}
	}()
	if err := f.Chmod(mode.Perm()); err != nil {
		return err
	}
	n, err := f.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := replace(tempPath, path); err != nil {
		return err
	}
	published = true
	return nil
}
