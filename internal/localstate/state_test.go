package localstate

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLockAndAtomicReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	require.NoError(t, Write(path, []byte("first")))
	require.NoError(t, Write(path, []byte("second")))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "second", string(data))
	unlock, err := Lock(nil, filepath.Join(dir, "state.lock"))
	require.NoError(t, err)
	_, err = Lock(nil, filepath.Join(dir, "state.lock"))
	require.ErrorIs(t, err, ErrBusy)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = Lock(ctx, filepath.Join(dir, "state.lock"))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	unlock()
	unlock()
	again, err := Lock(nil, filepath.Join(dir, "state.lock"))
	require.NoError(t, err)
	again()
}

func TestInvalidStatePathsReturnErrorsAndCleanTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(blocked, []byte("keep"), 0600))
	require.Error(t, Write(filepath.Join(blocked, "state.json"), []byte("new")))
	unlock, err := Lock(nil, filepath.Join(blocked, "state.lock"))
	require.Error(t, err)
	require.Nil(t, unlock)
	destination := filepath.Join(dir, "state.json")
	require.NoError(t, os.Mkdir(destination, 0700))
	child := filepath.Join(destination, "existing")
	require.NoError(t, os.WriteFile(child, []byte("original"), 0600))
	require.Error(t, Write(destination, []byte("replacement")))
	data, err := os.ReadFile(child)
	require.NoError(t, err)
	require.Equal(t, "original", string(data), "a failed replacement must preserve existing state")
	leftovers, err := filepath.Glob(filepath.Join(dir, ".state-*.tmp"))
	require.NoError(t, err)
	require.Empty(t, leftovers)
}
