//go:build !linux && !darwin && !windows

package localstate

import "os"

func tryLock(*os.File) error        { return ErrBusy }
func unlock(*os.File) error         { return nil }
func replace(src, dst string) error { return os.Rename(src, dst) }
