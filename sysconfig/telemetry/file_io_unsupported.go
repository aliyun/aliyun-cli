//go:build !linux && !darwin && !windows

package telemetry

import (
	"errors"
	"os"
)

var errUnsupportedFileIO = errors.New("telemetry file locking and replacement are unsupported on this platform")

func tryLockFile(_ *os.File) error { return errUnsupportedFileIO }

func unlockFile(_ *os.File) error { return errUnsupportedFileIO }

func replaceFile(_, _ string) error { return errUnsupportedFileIO }

func readInstanceIDFile(path string) ([]byte, error) { return os.ReadFile(path) }
