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
	assert.Equal(t, SiteCN, c.Site)
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	err := Save(dir, &PluginSettings{SourceBase: "https://mirror.example.com/plugins", Site: SiteIntl})
	assert.NoError(t, err)

	c, err := Load(dir)
	assert.NoError(t, err)
	assert.Equal(t, "https://mirror.example.com/plugins", c.SourceBase)
	assert.Equal(t, SiteIntl, c.Site)
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

func TestEffectiveSite_DefaultFileAndEnv(t *testing.T) {
	t.Setenv(EnvSite, "")
	assert.Equal(t, SiteCN, EffectiveSite(nil))
	assert.Equal(t, SiteIntl, EffectiveSite(&PluginSettings{Site: SiteIntl}))

	t.Setenv(EnvSite, SiteCN)
	assert.Equal(t, SiteCN, EffectiveSite(&PluginSettings{Site: SiteIntl}))
}

func TestSiteValidation(t *testing.T) {
	assert.True(t, IsValidSite(" CN "))
	assert.True(t, IsValidSite("intl"))
	assert.False(t, IsValidSite("global"))
	assert.Equal(t, SiteCN, NormalizeSite("invalid"))
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(dir, ConfigFileName), []byte("{"), 0600))
	c, err := Load(dir)
	assert.NoError(t, err)
	assert.Equal(t, "", c.SourceBase)
	assert.Equal(t, SiteCN, c.Site)
}
