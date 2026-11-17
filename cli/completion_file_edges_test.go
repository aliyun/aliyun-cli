package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompletionFilesPreserveLongLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shellrc")
	longLine := strings.Repeat("existing configuration ", 500)
	require.NoError(t, os.WriteFile(path, []byte(longLine+"\nremove this\nretain this\n"), 0600))
	require.True(t, lineInFile(path, longLine))
	require.False(t, lineInFile(path, "missing"))
	require.NoError(t, removeFromFile(path, "remove this"))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, longLine+"\nretain this\n", string(data))
}

func TestCompletionFileErrorsLeaveSourceIntact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shellrc")
	require.NoError(t, os.WriteFile(path, []byte("preserve\n"), 0600))
	require.NoError(t, os.Mkdir(path+".bck", 0700))
	require.Error(t, removeFromFile(path, "preserve"))
	require.Error(t, copyFile(filepath.Join(dir, "missing"), filepath.Join(dir, "copy")))
	require.Error(t, copyFile(path, dir))
	require.Error(t, appendToFile(filepath.Join(dir, "missing"), "content"))
	require.False(t, lineInFile(dir, "not a file"))
	_, err := removeContentToTempFile(filepath.Join(dir, "missing"), "content")
	require.Error(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "preserve\n", string(data))
}
