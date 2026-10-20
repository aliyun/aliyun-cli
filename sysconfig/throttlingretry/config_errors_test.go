package throttlingretry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetryDefaultsMalformedSettingsAndFilesystemErrors(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Save(dir, nil))
	for _, content := range []string{"invalid", `{"max_attempts":-1,"max_delay_ms":-2}`} {
		require.NoError(t, os.WriteFile(GetConfigFilePath(dir), []byte(content), 0600))
		got, err := Load(dir)
		require.NoError(t, err)
		require.Zero(t, got.MaxAttempts)
		require.Zero(t, got.MaxDelayMS)
	}
	blocked := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(blocked, nil, 0600))
	require.Error(t, Save(filepath.Join(blocked, "child"), nil))
	require.NoError(t, os.Remove(GetConfigFilePath(dir)))
	require.NoError(t, os.Mkdir(GetConfigFilePath(dir), 0700))
	_, err := LoadEffective(dir)
	require.Error(t, err)
	require.Error(t, Save(dir, nil))
	envs := map[string]string{"keep": "value"}
	MergeIntoPluginEnvs(dir, envs)
	require.Equal(t, map[string]string{"keep": "value"}, envs)
	t.Setenv(EnvEnabled, "invalid")
	t.Setenv(EnvMaxAttempts, "invalid")
	t.Setenv(EnvMaxDelayMS, "invalid")
	require.Equal(t, Default(), MergeFromEnv(nil))
}
