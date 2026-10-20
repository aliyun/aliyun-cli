package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/pluginsettings"
	"github.com/stretchr/testify/require"
)

type managerBoundaryTransport func(*http.Request) (*http.Response, error)

func (f managerBoundaryTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestManagerUnreadableSettingsAndMalformedCaches(t *testing.T) {
	home := t.TempDir()
	t.Cleanup(setTestHomeDir(t, home))
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".aliyun", pluginsettings.ConfigFileName), 0700))
	manager, err := NewManager()
	require.NoError(t, err)
	require.Empty(t, manager.sourceBase)
	cache := filepath.Join(t.TempDir(), "cache.json")
	require.NoError(t, os.WriteFile(cache, []byte("{"), 0600))
	var result Index
	hit, stale := manager.readCache(cache, time.Hour, &result)
	require.False(t, hit)
	require.False(t, stale)
	hit, stale = manager.readCache(filepath.Dir(cache), time.Hour, &result)
	require.False(t, hit)
	require.False(t, stale)
	_, err = httpGet("://bad-url", time.Second)
	require.Error(t, err)
	require.False(t, matchPluginInfoAlias(PluginInfo{CommandAliases: []string{"demo"}}, " "))
	_, err = resolvePluginInstallDir("", "demo")
	require.ErrorContains(t, err, "root directory is empty")
	_, err = validateInstalledPluginPath(t.TempDir(), "../escape", "path")
	require.Error(t, err)
	_, err = validateInstalledPluginPath(t.TempDir(), "demo", "")
	require.ErrorContains(t, err, "installed plugin path is empty")
	matched, missing := collectPluginsForProductCodes(nil, []string{"demo"})
	require.Nil(t, matched)
	require.Equal(t, []string{"demo"}, missing)
	require.False(t, pluginMatchesProductCode(nil, "demo"))
	require.False(t, pluginMatchesProductCode(&PluginInfo{}, " "))
	require.Nil(t, pluginInfos(nil))
}

func TestManagerInvalidLocalManifestsAndPackageReferences(t *testing.T) {
	manager := &Manager{rootDir: t.TempDir()}
	ctx := newTestContext()
	for _, ref := range []string{"", " ", filepath.Join(t.TempDir(), "missing.zip"), t.TempDir()} {
		require.Error(t, manager.InstallFromPackage(ctx, ref))
	}
	require.Error(t, manager.installFromRemotePackageURL(ctx, "https://example.invalid/%zz.zip"))
	manifestPath := filepath.Join(manager.rootDir, "manifest.json")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`{}`), 0600))
	manifest, err := manager.GetLocalManifest()
	require.NoError(t, err)
	require.NotNil(t, manifest.Plugins)
	require.NoError(t, os.WriteFile(manifestPath, []byte("{"), 0600))
	require.Error(t, manager.savePluginToManifest("demo", "1.0.0", "path", &PluginManifest{}))
	require.Error(t, manager.savePluginToManifest("../escape", "1.0.0", "path", &PluginManifest{}))
	require.Error(t, manager.UpdateAll(ctx, false))
	require.Error(t, manager.installListedPlugins(ctx, []*PluginInfo{{Name: "demo"}}, false))
	_, err = manager.loadAndValidatePluginManifest(t.TempDir(), "../escape")
	require.ErrorContains(t, err, "invalid expected plugin name")
	for _, raw := range []string{`{"name":"../escape","version":"1.0.0"}`, `{"name":"demo","version":"1.0.0","type":"unsupported"}`} {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(raw), 0600))
		_, err = manager.loadAndValidatePluginManifest(dir, "demo")
		require.Error(t, err)
		_, err = readPluginManifestFromDir(dir)
		require.Error(t, err)
	}
	blocked := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocked, []byte("file"), 0600))
	require.Error(t, manager.extractPlugin("unused", filepath.Join(blocked, "extract"), "archive.zip"))
}

func TestPluginUpdateSummariesWithUnavailableVersions(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	index := Index{Plugins: []PluginInfo{{Name: "demo", Versions: map[string]VersionInfo{}}}}
	body, err := json.Marshal(index)
	require.NoError(t, err)
	http.DefaultTransport = managerBoundaryTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	})
	manager := &Manager{rootDir: t.TempDir(), indexURL: "https://plugins.example.invalid/index.json"}
	require.NoError(t, manager.saveLocalManifest(&LocalManifest{Plugins: map[string]LocalPlugin{"demo": {Name: "demo", Version: "1.0.0"}}}))
	var out bytes.Buffer
	ctx := cli.NewCommandContext(&out, io.Discard)
	require.ErrorContains(t, manager.UpdateAll(ctx, false), "1 plugin(s) failed")
	require.Contains(t, out.String(), "no versions available")
	require.Contains(t, out.String(), "Failed: 1")
	manager.indexURL = "https://plugins.example.invalid/unavailable"
	http.DefaultTransport = managerBoundaryTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })
	manager.skipPluginIndexCacheForCLI = true
	_, err = manager.GetCommandIndex()
	require.Error(t, err)
}
