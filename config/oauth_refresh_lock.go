package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Keep the sidecar inode after unlocking: removing it permits two owners to
// lock different inodes. The OS releases the lock if the owner exits or crashes.
func lockOAuthConfiguration(path string) (func(), error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return nil, err
	}
	path = filepath.Join(parent, filepath.Base(absolute))
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("OAuth refresh requires the resolved configuration path, not a symlink")
	}
	lockPath := path + ".oauth.lock"
	if info, err := os.Lstat(lockPath); err == nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("OAuth lock is not a regular file")
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (func(), error) { file.Close(); return nil, err }
	// Check again after open to reject an exchanged path or a symlink.
	opened, err := file.Stat()
	if err != nil {
		return fail(err)
	}
	named, err := os.Lstat(lockPath)
	if err != nil {
		return fail(err)
	}
	if !named.Mode().IsRegular() || !os.SameFile(opened, named) {
		return fail(fmt.Errorf("OAuth lock changed while opening"))
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		acquired, err := tryOAuthFileLock(file)
		if err != nil {
			return fail(err)
		}
		if acquired {
			return func() { _ = releaseOAuthFileLock(file); _ = file.Close() }, nil
		}
		if time.Now().After(deadline) {
			return fail(fmt.Errorf("timed out waiting for OAuth refresh lock"))
		}
		time.Sleep(20 * time.Millisecond)
	}
}
