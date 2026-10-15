package aimode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFilesystemFailuresAndDefaultMerges(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(GetConfigFilePath(dir), 0700))
	_, err := Load(dir)
	require.Error(t, err)
	blocked := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocked, []byte("blocked"), 0600))
	require.Error(t, Save(filepath.Join(blocked, "child"), nil))
	require.Error(t, Save(t.TempDir(), &AiConfig{PluginSpecialOSSUTIL: make(chan int)}))
	good := t.TempDir()
	require.NoError(t, Save(good, nil))
	cfg, err := Load(good)
	require.NoError(t, err)
	require.Equal(t, DefaultAiConfig(), cfg)
	require.NoError(t, os.WriteFile(GetConfigFilePath(good), []byte("{"), 0600))
	cfg, err = Load(good)
	require.NoError(t, err)
	require.Equal(t, DefaultAiConfig(), cfg)
	MergeAiModeIntoOssutilPayload(dir, nil, false, false)
	MergeAiModeIntoOssutilPayload("", map[string]any{}, false, false)
	values := map[string]any{}
	MergeAiModeIntoOssutilPayload(dir, values, true, false)
	require.Equal(t, "1", values[OssutilConfigKeyAIMode])
	MergeUserAgentIntoPluginEnvs(dir, nil, false, false)
	envs := map[string]string{}
	MergeUserAgentIntoPluginEnvs(dir, envs, true, false)
	require.Equal(t, "1", envs[EnvAIMode])
	require.Contains(t, envs[EnvAIUserAgent], UserAgentEnabledMarker)
}
