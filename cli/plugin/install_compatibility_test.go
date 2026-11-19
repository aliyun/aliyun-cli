package plugin

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func TestInstallAndUpgradeRejectIncompatibleCLIBeforeDownloading(t *testing.T) {
	previous := cli.Version
	cli.Version = "3.5.4"
	t.Cleanup(func() { cli.Version = previous })
	for _, operation := range []string{"install_explicit", "install_latest", "reinstall", "upgrade"} {
		for _, minimum := range []string{"3.6.0", " \t3.6.0\n"} {
			t.Run(operation+"/"+minimum, func(t *testing.T) {
				const name = "aliyun-cli-minimum"
				root := t.TempDir()
				archive := createTestPluginArchive(t, name, "2.0.0", "minimum")
				checksum, err := calculateSHA256FromBytes(archive)
				require.NoError(t, err)
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					_, _ = w.Write(archive)
				}))
				defer server.Close()
				index := &Index{Plugins: []PluginInfo{{Name: name, Versions: map[string]VersionInfo{
					"2.0.0": {Metadata: &VersionMetadata{MinCliVersion: minimum}, Platforms: map[string]PlatformInfo{
						GetCurrentPlatform(): {URL: server.URL + "/plugin.tar.gz", Checksum: checksum},
					}},
				}}}}
				fetcher := usePluginTrustIndex(t, index)
				manager := &Manager{rootDir: root, sourceBase: server.URL, trustClient: fetcher}
				manifestPath := filepath.Join(root, "manifest.json")
				pluginDir := filepath.Join(root, name)
				var originalManifest []byte
				if operation == "reinstall" || operation == "upgrade" {
					require.NoError(t, os.MkdirAll(pluginDir, 0700))
					require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "binary"), []byte("existing working plugin"), 0700))
					require.NoError(t, manager.saveLocalManifest(&LocalManifest{Plugins: map[string]LocalPlugin{
						name: {Name: name, Version: "1.0.0", Command: "minimum", Path: pluginDir, MinCliVersion: "3.5.0"},
					}}))
					originalManifest, err = os.ReadFile(manifestPath)
					require.NoError(t, err)
				}
				ctx := newTestContext()
				switch operation {
				case "install_explicit", "reinstall":
					err = manager.Install(ctx, name, "2.0.0", false)
				case "install_latest":
					err = manager.Install(ctx, name, "", false)
				case "upgrade":
					err = manager.Upgrade(ctx, name, false)
				}
				require.EqualValues(t, 0, requests.Load(), "reject incompatible versions before downloading or replacing anything")
				require.ErrorContains(t, err, "requires CLI version 3.6.0 or higher, but you have 3.5.4")
				require.ErrorContains(t, err, "aliyun upgrade")
				require.Equal(t, 1, fetcher.calls)
				if originalManifest == nil {
					require.NoFileExists(t, manifestPath)
					require.NoDirExists(t, pluginDir)
				} else {
					manifest, err := os.ReadFile(manifestPath)
					require.NoError(t, err)
					require.Equal(t, originalManifest, manifest)
					binary, err := os.ReadFile(filepath.Join(pluginDir, "binary"))
					require.NoError(t, err)
					require.Equal(t, "existing working plugin", string(binary))
				}
			})
		}
	}
}
