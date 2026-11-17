//go:build windows

package telemetry

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

func tryLockFile(f *os.File) error {
	err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, &windows.Overlapped{})
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return ErrLockBusy
	}
	return err
}

func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &windows.Overlapped{})
}

// readInstanceIDFile shares delete access as well as reads and writes. Windows
// requires this even for a new reader while MoveFileEx holds a DELETE handle;
// os.ReadFile's default sharing mode does not allow that concurrent open.
func readInstanceIDFile(path string) ([]byte, error) {
	// Resolve before checking length: even a short relative filename can live
	// beneath a working directory longer than Windows' legacy path limit.
	name, err := filepath.Abs(path)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	if len(name) >= 248 && !strings.HasPrefix(name, `\\?\`) && !strings.HasPrefix(name, `\\.\`) {
		// Match os file access for long paths, including UNC shares. Extended
		// paths must be absolute and normalized before adding the prefix.
		if strings.HasPrefix(name, `\\`) {
			name = `\\?\UNC\` + name[2:]
		} else {
			name = `\\?\` + name
		}
	}
	namep, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	handle, err := windows.CreateFile(namep, windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return nil, instanceIDReadError(&os.PathError{Op: "open", Path: path, Err: err})
	}
	f := os.NewFile(uintptr(handle), path)
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, instanceIDReadError(err)
	}
	return data, nil
}

func instanceIDReadError(err error) error {
	if errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		// Open may fail on sharing conflicts; ReadFile may instead fail on an
		// exclusive byte-range lock. Neither means the persisted ID is absent.
		return fmt.Errorf("%w: %w", ErrLockBusy, err)
	}
	return err
}

func replaceFile(source, target string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	// Both paths are on the same volume. Do not allow a copy/delete fallback or
	// remove the target first. Filesystem and open-handle restrictions may fail it.
	for attempt := 0; ; attempt++ {
		err := windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
		if !errors.Is(err, windows.ERROR_SHARING_VIOLATION) || attempt == 4 {
			return err
		}
		// Readers without FILE_SHARE_DELETE can briefly prevent replacement.
		time.Sleep(5 * time.Millisecond)
	}
}
