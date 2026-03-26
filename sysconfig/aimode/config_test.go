package aimode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectiveUserAgent(t *testing.T) {
	assert.Equal(t, DefaultUserAgent, EffectiveUserAgent(nil))
	assert.Equal(t, DefaultUserAgent, EffectiveUserAgent(DefaultAiConfig()))
	assert.Equal(t, "CustomAgent/1", EffectiveUserAgent(&AiConfig{UserAgent: "CustomAgent/1"}))
}

func TestRequestUserAgentSuffix(t *testing.T) {
	assert.Equal(t, "", RequestUserAgentSuffix(nil))
	assert.Equal(t, "", RequestUserAgentSuffix(&AiConfig{Enabled: false}))
	s := RequestUserAgentSuffix(&AiConfig{Enabled: true, UserAgent: ""})
	assert.Contains(t, s, UserAgentEnabledMarker)
	assert.Contains(t, s, DefaultUserAgent)
	assert.Equal(t, UserAgentEnabledMarker+" CustomAgent/1", RequestUserAgentSuffix(&AiConfig{Enabled: true, UserAgent: "CustomAgent/1"}))
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
	assert.Equal(t, filepath.Join(dir, AiConfigFileName), GetConfigFilePath(dir))
}

func TestLoad_OssutilStringJSONDecodesToAny(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, AiConfigFileName)
	// ossutil written as a JSON string (escaped object) — common when piping string values
	require.NoError(t, os.WriteFile(path, []byte(`{"enabled":false,"ossutil":"{\"k\":\"v\",\"n\":1}"}`), 0600))
	c, err := Load(dir)
	require.NoError(t, err)
	m, ok := c.PluginSpecialOSSUTIL.(map[string]any)
	require.True(t, ok, "got %T", c.PluginSpecialOSSUTIL)
	assert.Equal(t, "v", m["k"])
	assert.Equal(t, float64(1), m["n"])
}

func TestNormalizePluginSpecialOSSUTIL(t *testing.T) {
	assert.Nil(t, normalizePluginSpecialOSSUTIL(nil))
	assert.Nil(t, normalizePluginSpecialOSSUTIL("   "))
	m := map[string]any{"a": 1}
	assert.Equal(t, m, normalizePluginSpecialOSSUTIL(m))
	assert.Equal(t, float64(42), normalizePluginSpecialOSSUTIL(float64(42)))
	// invalid JSON string stays as-is
	raw := "not-json"
	assert.Equal(t, raw, normalizePluginSpecialOSSUTIL(raw))
}

func TestMergeUserAgentIntoPluginEnvs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Save(dir, &AiConfig{Enabled: true, UserAgent: "my-ai-ua"}))

	envs := map[string]string{"ALIBABA_CLOUD_REGION_ID": "cn-hangzhou"}
	MergeUserAgentIntoPluginEnvs(dir, envs, false, false)
	assert.Equal(t, "1", envs[EnvAIMode])
	assert.Contains(t, envs[EnvAIUserAgent], UserAgentEnabledMarker)
	assert.Contains(t, envs[EnvAIUserAgent], "my-ai-ua")
	assert.NotContains(t, envs, "ALIBABA_CLOUD_USER_AGENT")

	t.Run("disabledNoInject", func(t *testing.T) {
		offDir := t.TempDir()
		require.NoError(t, Save(offDir, &AiConfig{Enabled: false}))
		e := map[string]string{"FOO": "bar"}
		MergeUserAgentIntoPluginEnvs(offDir, e, false, false)
		assert.NotContains(t, e, EnvAIMode)
		assert.NotContains(t, e, EnvAIUserAgent)
	})

	t.Run("forceOnWhenGlobalOff", func(t *testing.T) {
		offDir := t.TempDir()
		require.NoError(t, Save(offDir, &AiConfig{Enabled: false, UserAgent: "inline"}))
		e := map[string]string{}
		MergeUserAgentIntoPluginEnvs(offDir, e, true, false)
		assert.Equal(t, "1", e[EnvAIMode])
		assert.Contains(t, e[EnvAIUserAgent], "inline")
	})

	t.Run("forceOffClears", func(t *testing.T) {
		onDir := t.TempDir()
		require.NoError(t, Save(onDir, &AiConfig{Enabled: true}))
		e := map[string]string{EnvAIMode: "1", EnvAIUserAgent: "x"}
		MergeUserAgentIntoPluginEnvs(onDir, e, false, true)
		assert.Equal(t, "", e[EnvAIMode])
		assert.Equal(t, "", e[EnvAIUserAgent])
	})
}

func TestMergeAiModeIntoOssutilPayload(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Save(dir, &AiConfig{Enabled: true, UserAgent: "my-ai-ua"}))
	m := map[string]any{"region_id": "cn-hangzhou"}
	MergeAiModeIntoOssutilPayload(dir, m, false, false)
	assert.Equal(t, "1", m[OssutilConfigKeyAIMode])
	assert.Contains(t, m[OssutilConfigKeyAIUserAgent], UserAgentEnabledMarker)
	assert.Contains(t, m[OssutilConfigKeyAIUserAgent], "my-ai-ua")

	offDir := t.TempDir()
	require.NoError(t, Save(offDir, &AiConfig{Enabled: false}))
	m2 := map[string]any{"x": 1}
	MergeAiModeIntoOssutilPayload(offDir, m2, false, false)
	assert.NotContains(t, m2, OssutilConfigKeyAIMode)

	m3 := map[string]any{}
	MergeAiModeIntoOssutilPayload(offDir, m3, true, false)
	assert.Equal(t, "1", m3[OssutilConfigKeyAIMode])

	pluginDir := t.TempDir()
	require.NoError(t, Save(pluginDir, &AiConfig{PluginSpecialOSSUTIL: map[string]any{"k": "v"}}))
	mp := map[string]any{"region_id": "r"}
	MergeAiModeIntoOssutilPayload(pluginDir, mp, false, false)
	raw, ok := mp[OssutilConfigAIModeOssutilKey].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "v", raw["k"])

	bothDir := t.TempDir()
	require.NoError(t, Save(bothDir, &AiConfig{Enabled: true, UserAgent: "x", PluginSpecialOSSUTIL: map[string]any{"o": 1}}))
	mb := map[string]any{}
	MergeAiModeIntoOssutilPayload(bothDir, mb, false, false)
	assert.Equal(t, "1", mb[OssutilConfigKeyAIMode])
	blob, ok := mb[OssutilConfigAIModeOssutilKey].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), blob["o"]) // JSON number decode
}

func TestRequestUserAgentSuffixForCommand(t *testing.T) {
	cfg := &AiConfig{Enabled: true, UserAgent: "s"}
	assert.Equal(t, "", RequestUserAgentSuffixForCommand(cfg, false, true))
	assert.Contains(t, RequestUserAgentSuffixForCommand(&AiConfig{Enabled: false, UserAgent: "s"}, true, false), "s")
	assert.Equal(t, "", RequestUserAgentSuffixForCommand(&AiConfig{Enabled: false}, false, false))
}
