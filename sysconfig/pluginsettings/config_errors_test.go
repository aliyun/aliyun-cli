package pluginsettings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSettingsDefaultsAndFilesystemErrors(t *testing.T) {
	t.Setenv(EnvSourceBase, "")
	require.Empty(t, EffectiveSourceBase(nil))
	dir := t.TempDir()
	require.NoError(t, Save(dir, nil))
	got, err := Load(dir)
	require.NoError(t, err)
	require.Empty(t, got.SourceBase)
	blocked := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(blocked, nil, 0600))
	require.Error(t, Save(filepath.Join(blocked, "child"), nil))
	require.NoError(t, os.Remove(GetConfigFilePath(dir)))
	require.NoError(t, os.Mkdir(GetConfigFilePath(dir), 0700))
	_, err = Load(dir)
	require.Error(t, err)
	require.Error(t, Save(dir, nil))
}
