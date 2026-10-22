//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package config

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

func tryOAuthFileLock(f *os.File) (bool, error) {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EINTR) {
		return false, nil
	}
	return err == nil, err
}
func releaseOAuthFileLock(f *os.File) error { return unix.Flock(int(f.Fd()), unix.LOCK_UN) }
