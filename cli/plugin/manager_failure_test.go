package plugin

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPluginFilesystemFailures(t *testing.T) {
	root := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(root, []byte("preserve"), 0600))
	m := &Manager{rootDir: root}
	require.Error(t, m.saveLocalManifest(&LocalManifest{}))
	_, err := m.newPluginStagingDir("staging-*")
	require.Error(t, err)
	m.rootDir = t.TempDir()
	_, err = m.promoteExtractedPlugin(filepath.Join(m.rootDir, "missing"), "plugin")
	require.ErrorContains(t, err, "replace final path")
	cause := errors.New("transport failed")
	require.ErrorIs(t, safePluginDownloadError(cause, "https://example.invalid"), cause)
	require.Equal(t, "<unknown>", safePluginPackageURL(nil))
}

func TestIndexInstallManifestFailureRestoresOldPlugin(t *testing.T) {
	root := t.TempDir()
	m := &Manager{rootDir: root}
	ctx := newTestContext()
	v1 := createTestPluginArchive(t, "rollback-plugin", "1.0.0", "rollback")
	path := filepath.Join(t.TempDir(), "v1.tgz")
	require.NoError(t, os.WriteFile(path, v1, 0600))
	require.NoError(t, m.installFromPackageFile(ctx, path, path))
	v2 := createTestPluginArchive(t, "rollback-plugin", "2.0.0", "rollback")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(v2) }))
	defer server.Close()
	info := &PluginInfo{Name: "rollback-plugin", Versions: map[string]VersionInfo{"2.0.0": {Platforms: map[string]PlatformInfo{runtime.GOOS + "-" + runtime.GOARCH: {URL: server.URL + "/v2.tgz", Checksum: fmt.Sprintf("%x", sha256.Sum256(v2))}}}}}
	cause := errors.New("manifest commit failed")
	m.saveLocalManifestHook = func(*LocalManifest) error { return cause }
	require.ErrorIs(t, m.installPlugin(ctx, info, "2.0.0", false, false), cause)
	installed, err := readPluginManifestFromDir(filepath.Join(root, "rollback-plugin"))
	require.NoError(t, err)
	require.Equal(t, "1.0.0", installed.Version)
	manifest, err := m.GetLocalManifest()
	require.NoError(t, err)
	require.Equal(t, "1.0.0", manifest.Plugins["rollback-plugin"].Version)
}
