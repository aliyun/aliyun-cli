package pluginsettings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	c, err := Load(dir)
	assert.NoError(t, err)
	assert.Equal(t, "", c.SourceBase)
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	err := Save(dir, &PluginSettings{SourceBase: "https://mirror.example.com/plugins"})
	assert.NoError(t, err)

	c, err := Load(dir)
	assert.NoError(t, err)
	assert.Equal(t, "https://mirror.example.com/plugins", c.SourceBase)
}

func TestEffectiveSourceBase_EnvOverrides(t *testing.T) {
	dir := t.TempDir()
	_ = Save(dir, &PluginSettings{SourceBase: "https://from-file.example/plugins"})
	t.Setenv(EnvSourceBase, "https://from-env.example/plugins")
	c, err := Load(dir)
	assert.NoError(t, err)
	assert.Equal(t, "https://from-env.example/plugins", EffectiveSourceBase(c))
}

func TestEffectiveSourceBase_FromFile(t *testing.T) {
	t.Setenv(EnvSourceBase, "")
	dir := t.TempDir()
	assert.NoError(t, Save(dir, &PluginSettings{SourceBase: "  https://x.example/plugins/  "}))
	c, err := Load(dir)
	assert.NoError(t, err)
	assert.Equal(t, "https://x.example/plugins", EffectiveSourceBase(c))
}

func TestGetConfigFilePath(t *testing.T) {
	p := GetConfigFilePath("/tmp/cfg")
	assert.Equal(t, filepath.Join("/tmp/cfg", ConfigFileName), p)
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(dir, ConfigFileName), []byte("{"), 0600))
	c, err := Load(dir)
	assert.NoError(t, err)
	assert.Equal(t, "", c.SourceBase)
}
