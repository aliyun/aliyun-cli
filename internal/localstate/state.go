// Package localstate provides atomic files and process locks for local CLI state.
package localstate

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var ErrBusy = errors.New("local state is busy")

// Lock never removes the lock file. A nil context makes one nonblocking attempt.
func Lock(ctx context.Context, path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	for {
		err = tryLock(f)
		if err == nil {
			var once sync.Once
			return func() { once.Do(func() { _ = unlock(f); _ = f.Close() }) }, nil
		}
		if !errors.Is(err, ErrBusy) || ctx == nil {
			_ = f.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func Write(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".state-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err = f.Chmod(0600); err != nil {
		return err
	}
	n, err := f.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return replace(f.Name(), path)
}
