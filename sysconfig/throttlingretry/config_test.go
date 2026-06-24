package throttlingretry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaultWhenMissing(t *testing.T) {
	got, err := Load(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, got.Enabled)
	assert.Zero(t, got.MaxAttempts)
	assert.Zero(t, got.MaxDelayMS)
}

func TestSaveLoadAndMergeIntoPluginEnvs(t *testing.T) {
	dir := t.TempDir()
	enabled := false
	require.NoError(t, Save(dir, &Config{
		Enabled:     &enabled,
		MaxAttempts: 5,
		MaxDelayMS:  2000,
	}))

	envs := map[string]string{}
	MergeIntoPluginEnvs(dir, envs)

	assert.Equal(t, "false", envs[EnvEnabled])
	assert.Equal(t, "5", envs[EnvMaxAttempts])
	assert.Equal(t, "2000", envs[EnvMaxDelayMS])
}

func TestMergeFromEnvOverridesFile(t *testing.T) {
	enabled := false
	t.Setenv(EnvEnabled, "true")
	t.Setenv(EnvMaxAttempts, "7")
	t.Setenv(EnvMaxDelayMS, "3000")

	got := MergeFromEnv(&Config{
		Enabled:     &enabled,
		MaxAttempts: 5,
		MaxDelayMS:  2000,
	})

	require.NotNil(t, got.Enabled)
	assert.True(t, *got.Enabled)
	assert.Equal(t, 7, got.MaxAttempts)
	assert.Equal(t, int64(3000), got.MaxDelayMS)
}

func TestMergeIntoPluginEnvsNoop(t *testing.T) {
	MergeIntoPluginEnvs("", map[string]string{"a": "b"})
	MergeIntoPluginEnvs(t.TempDir(), nil)
}
