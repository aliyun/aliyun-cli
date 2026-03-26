package aimode

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectiveUserAgent(t *testing.T) {
	assert.Equal(t, DefaultUserAgent, EffectiveUserAgent(nil))
	assert.Equal(t, DefaultUserAgent, EffectiveUserAgent(DefaultConfig()))
	assert.Equal(t, "CustomAgent/1", EffectiveUserAgent(&Config{UserAgent: "CustomAgent/1"}))
}

func TestRequestUserAgentSuffix(t *testing.T) {
	assert.Equal(t, "", RequestUserAgentSuffix(nil))
	assert.Equal(t, "", RequestUserAgentSuffix(&Config{Enabled: false}))
	s := RequestUserAgentSuffix(&Config{Enabled: true, UserAgent: ""})
	assert.Contains(t, s, UserAgentEnabledMarker)
	assert.Contains(t, s, DefaultUserAgent)
	assert.Equal(t, UserAgentEnabledMarker+" CustomAgent/1", RequestUserAgentSuffix(&Config{Enabled: true, UserAgent: "CustomAgent/1"}))
}

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c, err := Load(dir)
	require.NoError(t, err)
	assert.False(t, c.Enabled)

	c.Enabled = true
	c.UserAgent = "my-agent"
	require.NoError(t, Save(dir, c))

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.True(t, loaded.Enabled)
	assert.Equal(t, "my-agent", loaded.UserAgent)
	assert.Equal(t, filepath.Join(dir, ConfigFileName), GetConfigFilePath(dir))
}

func TestMergeUserAgentIntoPluginEnvs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Save(dir, &Config{Enabled: true, UserAgent: "my-ai-ua"}))

	envs := map[string]string{"ALIBABA_CLOUD_REGION_ID": "cn-hangzhou"}
	MergeUserAgentIntoPluginEnvs(dir, envs, false, false)
	assert.Equal(t, "1", envs[EnvAIMode])
	assert.Contains(t, envs[EnvAIUserAgent], UserAgentEnabledMarker)
	assert.Contains(t, envs[EnvAIUserAgent], "my-ai-ua")
	assert.NotContains(t, envs, "ALIBABA_CLOUD_USER_AGENT")

	t.Run("disabledNoInject", func(t *testing.T) {
		offDir := t.TempDir()
		require.NoError(t, Save(offDir, &Config{Enabled: false}))
		e := map[string]string{"FOO": "bar"}
		MergeUserAgentIntoPluginEnvs(offDir, e, false, false)
		assert.NotContains(t, e, EnvAIMode)
		assert.NotContains(t, e, EnvAIUserAgent)
	})

	t.Run("forceOnWhenGlobalOff", func(t *testing.T) {
		offDir := t.TempDir()
		require.NoError(t, Save(offDir, &Config{Enabled: false, UserAgent: "inline"}))
		e := map[string]string{}
		MergeUserAgentIntoPluginEnvs(offDir, e, true, false)
		assert.Equal(t, "1", e[EnvAIMode])
		assert.Contains(t, e[EnvAIUserAgent], "inline")
	})

	t.Run("forceOffClears", func(t *testing.T) {
		onDir := t.TempDir()
		require.NoError(t, Save(onDir, &Config{Enabled: true}))
		e := map[string]string{EnvAIMode: "1", EnvAIUserAgent: "x"}
		MergeUserAgentIntoPluginEnvs(onDir, e, false, true)
		assert.Equal(t, "", e[EnvAIMode])
		assert.Equal(t, "", e[EnvAIUserAgent])
	})
}

func TestMergeIntoOssutilConfigPayload(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Save(dir, &Config{Enabled: true, UserAgent: "my-ai-ua"}))
	m := map[string]any{"region_id": "cn-hangzhou"}
	MergeIntoOssutilConfigPayload(dir, m, false, false)
	assert.Equal(t, "1", m[OssutilConfigKeyAIMode])
	assert.Contains(t, m[OssutilConfigKeyAIUserAgent], UserAgentEnabledMarker)
	assert.Contains(t, m[OssutilConfigKeyAIUserAgent], "my-ai-ua")

	offDir := t.TempDir()
	require.NoError(t, Save(offDir, &Config{Enabled: false}))
	m2 := map[string]any{"x": 1}
	MergeIntoOssutilConfigPayload(offDir, m2, false, false)
	assert.NotContains(t, m2, OssutilConfigKeyAIMode)

	m3 := map[string]any{}
	MergeIntoOssutilConfigPayload(offDir, m3, true, false)
	assert.Equal(t, "1", m3[OssutilConfigKeyAIMode])
}

func TestRequestUserAgentSuffixForCommand(t *testing.T) {
	cfg := &Config{Enabled: true, UserAgent: "s"}
	assert.Equal(t, "", RequestUserAgentSuffixForCommand(cfg, false, true))
	assert.Contains(t, RequestUserAgentSuffixForCommand(&Config{Enabled: false, UserAgent: "s"}, true, false), "s")
	assert.Equal(t, "", RequestUserAgentSuffixForCommand(&Config{Enabled: false}, false, false))
}
