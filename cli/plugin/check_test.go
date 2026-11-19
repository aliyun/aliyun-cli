package plugin

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCheckBothPluginKindsAndPlatforms(t *testing.T) {
	local := &LocalManifest{Plugins: map[string]LocalPlugin{
		"aliyun-cli-go": {Version: "1.0.0"}, "aliyun-cli-meta": {Version: "2.0.0", Type: PluginTypeMeta}, "local-only": {Version: "1.0.0"},
	}}
	index := &Index{Plugins: []PluginInfo{
		{Name: "aliyun-cli-go", Versions: map[string]VersionInfo{"1.1.0": {Platforms: map[string]PlatformInfo{GetCurrentPlatform(): {}}}, "2.0.0-beta.1": {}}},
		{Name: "aliyun-cli-meta", Versions: map[string]VersionInfo{"2.1.0": {Metadata: &VersionMetadata{MinCliVersion: "9.0.0"}, Platforms: map[string]PlatformInfo{PluginPlatformAny: {}}}}},
	}}
	r := CompareUpdates(local, index, "3.5.3")
	require.Len(t, r.Plugins, 3)
	require.Equal(t, "1.1.0", r.Plugins[0].Latest)
	require.True(t, r.Plugins[0].UpdateAvailable)
	require.Equal(t, PluginTypeMeta, r.Plugins[1].Type)
	require.True(t, r.Plugins[1].RequiresCLIUpgrade)
	require.Equal(t, "not_in_source", r.Plugins[2].Status)
	index.Plugins[0].Versions["1.2.0"] = VersionInfo{Platforms: map[string]PlatformInfo{"unavailable-platform": {}}}
	r = CompareUpdates(local, index, "3.5.3")
	require.False(t, r.Plugins[0].UpdateAvailable)
	require.Equal(t, "unsupported_platform", r.Plugins[0].Status)
}

func TestCheckUpdatesNormalizesMinimumVersions(t *testing.T) {
	local := &LocalManifest{Plugins: map[string]LocalPlugin{
		"example": {Version: "1.0.0", MinCliVersion: " \t3.4.0\n"},
	}}
	index := &Index{Plugins: []PluginInfo{{Name: "example", Versions: map[string]VersionInfo{
		"1.1.0": {Metadata: &VersionMetadata{MinCliVersion: " \t3.6.0\n"}, Platforms: map[string]PlatformInfo{PluginPlatformAny: {}}},
	}}}}
	r := CompareUpdates(local, index, "3.5.3")
	require.Len(t, r.Plugins, 1)
	require.Equal(t, "3.4.0", r.Plugins[0].InstalledMinCliVersion)
	require.Equal(t, "3.6.0", r.Plugins[0].MinCliVersion)
	require.True(t, r.Plugins[0].UpdateAvailable)
	require.True(t, r.Plugins[0].RequiresCLIUpgrade)
	// Reading/comparing the installed manifest must not rewrite its contents.
	require.Equal(t, " \t3.4.0\n", local.Plugins["example"].MinCliVersion)
}
