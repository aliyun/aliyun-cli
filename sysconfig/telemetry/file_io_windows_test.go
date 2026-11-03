//go:build windows

package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

const windowsReadTestInstanceID = "c4160461-19cf-4ec9-bc28-b2430c1ad90e"

func holdWindowsInstanceIDHandle(t *testing.T, path string, access, sharing uint32) func() {
	t.Helper()
	name, err := windows.UTF16PtrFromString(path)
	require.NoError(t, err)
	handle, err := windows.CreateFile(name, access, sharing, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	require.NoError(t, err)
	var once sync.Once
	closeHandle := func() {
		once.Do(func() { require.NoError(t, windows.CloseHandle(handle)) })
	}
	t.Cleanup(closeHandle)
	return closeHandle
}

func TestInstanceIDReadWhileDeleteHandleIsOpen(t *testing.T) {
	dir := t.TempDir()
	path := GetInstanceIDFilePath(dir)
	require.NoError(t, os.WriteFile(path, []byte(windowsReadTestInstanceID+"\n"), 0600))
	holdWindowsInstanceIDHandle(t, path, windows.DELETE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE)

	// MoveFileEx can hold DELETE access while the newly published name is
	// already visible. An ordinary Go reader does not share that access.
	_, err := os.ReadFile(path)
	require.ErrorIs(t, err, windows.ERROR_SHARING_VIOLATION,
		"the fixture must reproduce the old reader's sharing conflict")

	for _, load := range []func(string) (string, error){LoadInstanceID, EnsureInstanceID} {
		id, err := load(dir)
		require.NoError(t, err)
		require.Equal(t, windowsReadTestInstanceID, id)
	}
	require.NoFileExists(t, filepath.Join(dir, instanceIDLockName),
		"reading a published ID must not start initialization")
}

func TestInstanceIDExclusiveHandleIsBusyAndPreservesID(t *testing.T) {
	dir := t.TempDir()
	path := GetInstanceIDFilePath(dir)
	contents := []byte(windowsReadTestInstanceID + "\n")
	require.NoError(t, os.WriteFile(path, contents, 0600))
	closeHandle := holdWindowsInstanceIDHandle(t, path, windows.GENERIC_READ, 0)

	for _, load := range []func(string) (string, error){LoadInstanceID, EnsureInstanceID} {
		id, err := load(dir)
		require.Empty(t, id)
		require.ErrorIs(t, err, ErrLockBusy)
		require.ErrorIs(t, err, windows.ERROR_SHARING_VIOLATION)
		require.False(t, os.IsNotExist(err), "a busy ID must never be treated as absent")
		var pathErr *os.PathError
		require.ErrorAs(t, err, &pathErr)
		require.Equal(t, path, pathErr.Path)
	}
	require.NoFileExists(t, filepath.Join(dir, instanceIDLockName),
		"busy readers must not try to generate a replacement ID")
	closeHandle()
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, contents, got)
	id, err := EnsureInstanceID(dir)
	require.NoError(t, err)
	require.Equal(t, windowsReadTestInstanceID, id)
}

func TestInstanceIDReadLongWindowsPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(),
		strings.Repeat("segment", 12), strings.Repeat("segment", 12), strings.Repeat("segment", 12))
	path := GetInstanceIDFilePath(dir)
	require.Greater(t, len(path), 260)
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, os.WriteFile(path, []byte(windowsReadTestInstanceID+"\n"), 0600))
	extendedDir := dir
	if !strings.HasPrefix(dir, `\\?\`) {
		if strings.HasPrefix(dir, `\\`) {
			extendedDir = `\\?\UNC\` + dir[2:]
		} else {
			extendedDir = `\\?\` + dir
		}
	}
	for _, test := range []struct{ name, dir string }{
		{"absolute", dir},
		{"extended", extendedDir},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, load := range []func(string) (string, error){LoadInstanceID, EnsureInstanceID} {
				id, err := load(test.dir)
				require.NoError(t, err)
				require.Equal(t, windowsReadTestInstanceID, id)
			}
		})
	}
}

func TestInstanceIDReadShortRelativePathFromLongWorkingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(),
		strings.Repeat("segment", 12), strings.Repeat("segment", 12), strings.Repeat("segment", 12))
	require.Greater(t, len(dir), 260)
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, os.WriteFile(GetInstanceIDFilePath(dir), []byte(windowsReadTestInstanceID+"\n"), 0600))
	// Chdir's cleanup is registered after TempDir's cleanup, restoring the
	// original directory before removing the long directory tree.
	t.Chdir(dir)
	require.Less(t, len(GetInstanceIDFilePath(".")), 248)
	for _, load := range []func(string) (string, error){LoadInstanceID, EnsureInstanceID} {
		id, err := load(".")
		require.NoError(t, err)
		require.Equal(t, windowsReadTestInstanceID, id)
	}
}

func TestInstanceIDWindowsMissingAndDirectoryErrorsRemainDistinct(t *testing.T) {
	dir := t.TempDir()
	path := GetInstanceIDFilePath(dir)
	id, err := LoadInstanceID(dir)
	require.NoError(t, err)
	require.Empty(t, id)
	require.NoFileExists(t, path, "loading an absent ID must not create one")

	require.NoError(t, os.Mkdir(path, 0700))
	for _, load := range []func(string) (string, error){LoadInstanceID, EnsureInstanceID} {
		_, err := load(dir)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrLockBusy, "directory read failure is not sharing contention")
		require.False(t, os.IsNotExist(err))
	}
}
