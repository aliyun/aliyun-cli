package updatenotifier

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestPreferenceAndEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(DisableEnv, "")
	require.True(t, Enabled(dir))
	require.NoError(t, Save(dir, false))
	require.False(t, Enabled(dir))
	t.Setenv(DisableEnv, "0")
	require.False(t, Enabled(dir))
	require.NoError(t, Save(dir, true))
	require.True(t, Enabled(dir))
	for _, value := range []string{"1", "TRUE", " yes "} {
		t.Setenv(DisableEnv, value)
		require.False(t, Enabled(dir))
	}
	t.Setenv(DisableEnv, "")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ConfigFileName), []byte("broken"), 0600))
	require.False(t, Enabled(dir))
}

func TestUnreadableOrUnwritablePreferencesFailSafely(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(DisableEnv, "0")
	require.NoError(t, os.Mkdir(filepath.Join(dir, ConfigFileName), 0700))
	_, err := Load(dir)
	require.Error(t, err)
	require.False(t, Enabled(dir))
	require.Error(t, Save(dir, true))
}
