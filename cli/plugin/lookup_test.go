package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupLocalPluginForTelemetry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := t.TempDir()
	t.Setenv(EnvPluginsDir, dir)
	manifest := LocalManifest{Plugins: map[string]LocalPlugin{
		"aliyun-cli-demo":   {Name: "aliyun-cli-demo", Command: "demo", CommandAliases: []string{"alias-demo"}, Inner: true, Version: "1.2.3"},
		"aliyun-cli-public": {Name: "aliyun-cli-public", Command: "public"},
	}}
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600))
	for _, command := range []string{"demo", "alias-demo", "aliyun-cli-demo"} {
		local, err := LookupLocalPlugin(command)
		require.NoError(t, err)
		require.Equal(t, manifest.Plugins["aliyun-cli-demo"], *local)
		require.True(t, IsInnerPluginCommand(command))
	}
	for _, command := range []string{"public", "not-installed"} {
		require.False(t, IsInnerPluginCommand(command))
	}
	local, err := LookupLocalPlugin("not-installed")
	require.NoError(t, err)
	require.Nil(t, local)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{"), 0600))
	local, err = LookupLocalPlugin("demo")
	require.ErrorContains(t, err, "failed to read local plugin manifest")
	require.Nil(t, local)
	require.False(t, IsInnerPluginCommand("demo"))

	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, []byte("keep"), 0600))
	t.Setenv(EnvPluginsDir, file)
	local, err = LookupLocalPlugin("demo")
	require.Error(t, err)
	require.Nil(t, local)
	require.False(t, IsInnerPluginCommand("demo"))
}
