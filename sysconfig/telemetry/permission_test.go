//go:build linux || darwin

package telemetry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	permissionTestConfig = "{\"enabled\":true}\n"
	permissionTestID     = "d69b3a04-b81c-47e7-a882-504e5865352d"
)

func newPermissionFixture(t *testing.T, withID bool) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(GetConfigFilePath(dir), []byte(permissionTestConfig), 0600))
	if withID {
		require.NoError(t, os.WriteFile(GetInstanceIDFilePath(dir), []byte(permissionTestID+"\n"), 0600))
	}
	return dir
}

func assertPermissionFixtureUnchanged(t *testing.T, dir string, withID bool) {
	t.Helper()
	data, err := os.ReadFile(GetConfigFilePath(dir))
	require.NoError(t, err)
	require.Equal(t, permissionTestConfig, string(data))
	if withID {
		data, err = os.ReadFile(GetInstanceIDFilePath(dir))
		require.NoError(t, err)
		require.Equal(t, permissionTestID+"\n", string(data))
	} else {
		require.NoFileExists(t, GetInstanceIDFilePath(dir))
	}
}

func makePermissionDirectoryReadOnly(t *testing.T, dir string) {
	t.Helper()
	info, err := os.Stat(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := os.Chmod(dir, info.Mode().Perm()); err != nil {
			t.Errorf("restore directory permissions: %v", err)
		}
	})
	require.NoError(t, os.Chmod(dir, 0500))

	// Verify the effective permissions instead of trusting mode bits: root or
	// a filesystem that ignores chmod must not make these tests pass falsely.
	f, err := os.CreateTemp(dir, ".permission-probe-*")
	if err == nil {
		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(f.Name()))
		t.Skip("current user or filesystem permits writes despite chmod; permission denial cannot be tested")
	}
	require.ErrorIs(t, err, os.ErrPermission, "permission probe failed for an unrelated reason")
}

func TestEnsureInstanceIDWithUnwritableDirectory(t *testing.T) {
	for _, existingLock := range []bool{false, true} {
		name := "lock_creation_denied"
		if existingLock {
			name = "id_temp_creation_denied"
		}
		t.Run(name, func(t *testing.T) {
			dir := newPermissionFixture(t, false)
			lockPath := filepath.Join(dir, instanceIDLockName)
			if existingLock {
				require.NoError(t, os.WriteFile(lockPath, []byte("stable lock"), 0600))
			}
			makePermissionDirectoryReadOnly(t, dir)
			if existingLock {
				// The lock remains writable; only publishing the missing ID fails.
				f, err := os.OpenFile(lockPath, os.O_RDWR, 0)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}

			id, err := EnsureInstanceID(dir)
			require.ErrorIs(t, err, os.ErrPermission)
			require.Empty(t, id)
			var pathErr *os.PathError
			require.ErrorAs(t, err, &pathErr)
			if existingLock {
				require.Equal(t, dir, filepath.Dir(pathErr.Path))
				require.Contains(t, filepath.Base(pathErr.Path), "."+InstanceIDFileName+"-")
				data, readErr := os.ReadFile(lockPath)
				require.NoError(t, readErr)
				require.Equal(t, "stable lock", string(data))
			} else {
				require.Equal(t, lockPath, pathErr.Path)
				require.NoFileExists(t, lockPath)
			}
			assertPermissionFixtureUnchanged(t, dir, false)
			require.NoDirExists(t, filepath.Join(dir, CacheDirName))
		})
	}
}

func TestEnsureInstanceIDReadsExistingReadOnlyID(t *testing.T) {
	dir := newPermissionFixture(t, true)
	idPath := GetInstanceIDFilePath(dir)
	require.NoError(t, os.Chmod(idPath, 0400))
	t.Cleanup(func() { require.NoError(t, os.Chmod(idPath, 0600)) })
	makePermissionDirectoryReadOnly(t, dir)

	id, err := EnsureInstanceID(dir)
	require.NoError(t, err)
	require.Equal(t, permissionTestID, id)
	assertPermissionFixtureUnchanged(t, dir, true)
	require.NoFileExists(t, filepath.Join(dir, instanceIDLockName))
	require.NoDirExists(t, filepath.Join(dir, CacheDirName))
}

func TestWriteEventWithUnwritableStorage(t *testing.T) {
	for _, name := range []string{"cache_root_creation_denied", PipelineInner, PipelinePublic} {
		t.Run(name, func(t *testing.T) {
			dir := newPermissionFixture(t, true)
			pipeline := name
			missingCacheRoot := name == "cache_root_creation_denied"
			if missingCacheRoot {
				pipeline = PipelinePublic
			}
			cacheDir := GetCacheDir(dir, pipeline)
			if missingCacheRoot {
				makePermissionDirectoryReadOnly(t, dir)
			} else {
				require.NoError(t, os.MkdirAll(cacheDir, 0700))
				makePermissionDirectoryReadOnly(t, cacheDir)
			}

			path, err := WriteEvent(cacheDir, Event{EventID: "permission-test", InstanceID: permissionTestID})
			require.ErrorIs(t, err, os.ErrPermission)
			require.Empty(t, path)
			assertPermissionFixtureUnchanged(t, dir, true)
			if missingCacheRoot {
				require.NoDirExists(t, filepath.Join(dir, CacheDirName))
			} else {
				entries, readErr := os.ReadDir(cacheDir)
				require.NoError(t, readErr)
				require.Empty(t, entries, "failed writes must leave no event file")
			}
		})
	}
}
