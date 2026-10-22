//go:build !darwin && !linux && !freebsd && !openbsd && !netbsd && !dragonfly && !windows

package config

import (
	"fmt"
	"os"
)

func tryOAuthFileLock(*os.File) (bool, error) {
	return false, fmt.Errorf("OAuth refresh locking is unsupported on this platform")
}
func releaseOAuthFileLock(*os.File) error { return nil }
