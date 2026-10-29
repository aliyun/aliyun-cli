//go:build windows

package telemetry

import (
	"errors"
	"os"
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
