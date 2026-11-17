package plugin

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManagerRecoveryRefusesHomeRelativePackagesWithoutHome(t *testing.T) {
	for _, name := range []string{"HOME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH"} {
		t.Setenv(name, "")
	}
	path, err := expandPluginSourcePath("~/package.tgz")
	require.ErrorContains(t, err, "resolve home directory")
	require.Empty(t, path)
}

func TestManagerRecoveryRejectsPackageWithoutManifestAndRemovesStaging(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "missing-manifest.tgz")
	require.NoError(t, os.WriteFile(archive, createTestPluginArchiveWithoutManifest(t), 0600))
	manager := &Manager{rootDir: t.TempDir()}
	require.ErrorContains(t, manager.InstallFromLocalFile(newTestContext(), archive), "invalid plugin package: manifest.json not found")
	entries, err := os.ReadDir(manager.rootDir)
	require.NoError(t, err)
	require.Empty(t, entries)
	require.FileExists(t, archive)
}

func TestManagerRecoveryCustomInstallPreservesBothFailureReasons(t *testing.T) {
	var requests atomic.Int32
	manager := &Manager{
		rootDir:     t.TempDir(),
		indexURL:    "https://packages.example.invalid/index.json",
		canonicalFS: canonicalProductsFS(`{"products":[{"code":"Known","distribution":"go"},{"code":"Missing","distribution":"go"}]}`),
		httpTransport: managerBoundaryTransport(func(request *http.Request) (*http.Response, error) {
			requests.Add(1)
			require.Equal(t, "/index.json", request.URL.Path)
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewBufferString(`{"plugins":[{"name":"aliyun-cli-known","productCode":"Known","versions":{}}]}`))}, nil
		}),
	}
	ctx := newTestContext()
	err := manager.InstallCustom(ctx, false)
	require.ErrorContains(t, err, "1 plugin(s) failed to install")
	require.ErrorContains(t, err, "1 custom product(s) had no plugin in the index")
	require.EqualValues(t, 1, requests.Load())
	require.Contains(t, ctx.Stdout().(*bytes.Buffer).String(), "no versions available for plugin aliyun-cli-known")
	require.Contains(t, ctx.Stdout().(*bytes.Buffer).String(), "No plugin found in index for product Missing")
	require.NoDirExists(t, filepath.Join(manager.rootDir, "aliyun-cli-known"))
}
