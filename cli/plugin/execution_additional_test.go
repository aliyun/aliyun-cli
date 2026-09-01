package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstalledPluginTypeAndPublicVersionValidation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvPluginsDir, dir)
	manifest := LocalManifest{Plugins: map[string]LocalPlugin{
		"aliyun-cli-go-demo": {
			Name: "aliyun-cli-go-demo", Version: "1.0.0", ProductCode: "go-demo", Command: "go-demo",
		},
		"aliyun-cli-meta-demo": {
			Name: "aliyun-cli-meta-demo", Version: "1.0.0", ProductCode: "meta-demo", Command: "meta-demo", Type: PluginTypeMeta,
		},
	}}
	encoded, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), encoded, 0600))

	pluginType, ok := InstalledPluginType("go-demo")
	assert.True(t, ok)
	assert.Equal(t, PluginTypeGo, pluginType)
	pluginType, ok = InstalledPluginType("meta-demo")
	assert.True(t, ok)
	assert.Equal(t, PluginTypeMeta, pluginType)
	_, ok = InstalledPluginType("missing")
	assert.False(t, ok)

	require.NoError(t, ValidatePluginCliVersion("go-demo"))
	assert.Error(t, ValidatePluginCliVersion("missing"))
	metaPlugin := manifest.Plugins["aliyun-cli-meta-demo"]
	goPlugin := manifest.Plugins["aliyun-cli-go-demo"]
	assert.True(t, metaPlugin.IsMeta())
	assert.False(t, goPlugin.IsMeta())
	assert.False(t, (*LocalPlugin)(nil).IsMeta())
}
