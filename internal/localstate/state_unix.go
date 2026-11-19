//go:build linux || darwin

package localstate

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

func tryLock(f *os.File) error {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
		return ErrBusy
	}
	return err
}
func unlock(f *os.File) error       { return unix.Flock(int(f.Fd()), unix.LOCK_UN) }
func replace(src, dst string) error { return os.Rename(src, dst) }
