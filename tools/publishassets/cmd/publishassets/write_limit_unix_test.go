//go:build darwin || linux

package main

import "syscall"

func restoreWriteFailureLimit() error {
	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
		return err
	}
	limit.Cur = limit.Max
	return syscall.Setrlimit(syscall.RLIMIT_FSIZE, &limit)
}
