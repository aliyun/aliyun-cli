package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadInstanceIDRejectsUnreadableAndNonHexState(t *testing.T) {
	t.Run("directory_instead_of_file", func(t *testing.T) {
		dir := t.TempDir()
		path := GetInstanceIDFilePath(dir)
		require.NoError(t, os.Mkdir(path, 0700))
		for _, load := range []func(string) (string, error){LoadInstanceID, EnsureInstanceID} {
			id, err := load(dir)
			require.Error(t, err)
			require.Empty(t, id)
		}
		require.DirExists(t, path, "collection must not overwrite unreadable state")
	})
	t.Run("correct_shape_invalid_hex", func(t *testing.T) {
		dir := t.TempDir()
		const invalid = "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz"
		require.NoError(t, os.WriteFile(GetInstanceIDFilePath(dir), []byte(invalid), 0600))
		id, err := EnsureInstanceID(dir)
		require.ErrorContains(t, err, "purge")
		require.Empty(t, id)
		data, err := os.ReadFile(GetInstanceIDFilePath(dir))
		require.NoError(t, err)
		require.Equal(t, invalid, string(data))
	})
}

func TestPurgeReportsFailureToRemoveInstanceID(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Save(dir, &Config{Enabled: false}))
	path := GetInstanceIDFilePath(dir)
	require.NoError(t, os.Mkdir(path, 0700))
	child := filepath.Join(path, "keep")
	require.NoError(t, os.WriteFile(child, []byte("private contents"), 0600))
	err := Purge(dir)
	var pathErr *os.PathError
	require.ErrorAs(t, err, &pathErr)
	require.Equal(t, path, pathErr.Path)
	data, err := os.ReadFile(child)
	require.NoError(t, err)
	require.Equal(t, "private contents", string(data))
	cfg, err := Load(dir)
	require.NoError(t, err)
	require.False(t, cfg.Enabled, "a failed purge must preserve preferences")
}

func TestTelemetryPreferenceStorageErrors(t *testing.T) {
	t.Run("nil_uses_default", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, Save(dir, nil))
		cfg, err := Load(dir)
		require.NoError(t, err)
		require.Equal(t, DefaultConfig(), cfg)
	})
	t.Run("config_path_is_directory", func(t *testing.T) {
		dir := t.TempDir()
		path := GetConfigFilePath(dir)
		require.NoError(t, os.Mkdir(path, 0700))
		cfg, err := Load(dir)
		require.Error(t, err)
		require.Nil(t, cfg)
		require.Error(t, Save(dir, DefaultConfig()))
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, entries, 1, "failed save must remove its temporary file")
		require.Equal(t, ConfigFileName, entries[0].Name())
	})
	t.Run("parent_is_file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(path, []byte("keep"), 0600))
		require.Error(t, Save(path, DefaultConfig()))
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "keep", string(data))
	})
}

func TestEventCacheInvalidDestinationAndCleanup(t *testing.T) {
	cache := t.TempDir()
	// A slash in an event ID cannot create an unrequested nested directory.
	path, err := WriteEvent(cache, Event{EventID: "missing/child"})
	require.Error(t, err)
	require.Empty(t, path)
	entries, err := os.ReadDir(cache)
	require.NoError(t, err)
	require.Empty(t, entries)

	// A directory with an event-like name is never an expired event file.
	nested := filepath.Join(cache, "event-keep.ndjson")
	require.NoError(t, os.Mkdir(nested, 0700))
	child := filepath.Join(nested, "keep")
	require.NoError(t, os.WriteFile(child, []byte("keep"), 0600))
	old := time.Now().Add(-30 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(nested, old, old))
	cleanupExpiredEventFiles(cache, time.Now(), 24*time.Hour)
	require.FileExists(t, child)
	require.NotPanics(t, func() { cleanupExpiredEventFiles(filepath.Join(cache, "missing"), time.Now(), time.Hour) })

	// A missing parent must fail before replacement is attempted.
	require.Error(t, atomicWriteFile(filepath.Join(cache, "missing", "target"), []byte("data"), 0600))
	entries, err = os.ReadDir(cache)
	require.NoError(t, err)
	for _, entry := range entries {
		require.False(t, strings.HasSuffix(entry.Name(), ".tmp"))
	}
}
