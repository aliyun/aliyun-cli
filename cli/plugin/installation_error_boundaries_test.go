package plugin

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func TestPackageDownloadHelpersEnforceStatusAndLimits(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	for _, tc := range []struct {
		name      string
		status    int
		body      string
		wantError string
	}{
		{"valid", 200, "archive", ""}, {"unavailable", 503, "failure", "download failed: 503"}, {"oversized", 200, "archive too large", "exceeds limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			http.DefaultTransport = managerBoundaryTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(bytes.NewBufferString(tc.body)), Header: make(http.Header), ContentLength: int64(len(tc.body))}, nil
			})
			path := filepath.Join(t.TempDir(), "package.zip")
			if tc.wantError == "" {
				require.NoError(t, downloadFile("https://packages.example.invalid/archive.zip", path))
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, tc.body, string(data))
			} else {
				require.ErrorContains(t, downloadFileWithLimit("https://packages.example.invalid/archive.zip", path, 10), tc.wantError)
				require.NoFileExists(t, path)
			}
		})
	}
	require.Error(t, downloadFile("://invalid", filepath.Join(t.TempDir(), "package")))
}

func TestInstallLocalZipAndManifestCommitFailure(t *testing.T) {
	manager := &Manager{rootDir: t.TempDir()}
	archive := filepath.Join(t.TempDir(), "package.zip")
	require.NoError(t, createTestZip(archive, []testFile{{name: "manifest.json", content: `{"name":"demo","version":"1.0.0","command":"demo"}`}, {name: "demo", content: "binary fixture"}}))
	var output bytes.Buffer
	ctx := cli.NewCommandContext(&output, io.Discard)
	require.NoError(t, manager.InstallFromLocalFile(ctx, archive))
	manifest, err := manager.GetLocalManifest()
	require.NoError(t, err)
	require.Equal(t, "1.0.0", manifest.Plugins["demo"].Version)
	require.FileExists(t, filepath.Join(manager.rootDir, "demo", "demo"))
	manifest.Plugins["demo"] = LocalPlugin{Name: "demo", Path: filepath.Join(manager.rootDir, "demo")}
	require.NoError(t, manager.saveLocalManifest(manifest))
	manager.printOverwriteIfPluginInstalled(ctx, "demo", "2.0.0")
	require.Contains(t, output.String(), "version unknown")
	cause := errors.New("manifest disk full")
	manager.saveLocalManifestHook = func(*LocalManifest) error { return cause }
	require.ErrorIs(t, manager.Uninstall(ctx, "demo"), cause)
	require.NoDirExists(t, filepath.Join(manager.rootDir, "demo"))
	unchanged, err := manager.GetLocalManifest()
	require.NoError(t, err)
	require.Contains(t, unchanged.Plugins, "demo")
	manager = &Manager{rootDir: t.TempDir(), saveLocalManifestHook: func(*LocalManifest) error { return cause }}
	require.ErrorIs(t, manager.InstallFromLocalFile(ctx, archive), cause)
	require.NoDirExists(t, filepath.Join(manager.rootDir, "demo"))
	blocked := filepath.Join(t.TempDir(), "blocked")
	require.NoError(t, os.WriteFile(blocked, []byte("keep"), 0600))
	manager = &Manager{rootDir: blocked}
	require.Error(t, manager.InstallFromLocalFile(ctx, archive))
	_, err = calculateSHA256(t.TempDir())
	require.Error(t, err)
}

func TestRemotePackageDownloadStorageFailures(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	manager := &Manager{rootDir: t.TempDir()}
	ctx := newTestContext()
	http.DefaultTransport = managerBoundaryTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(archiveBrokenReader{})}, nil
	})
	require.ErrorContains(t, manager.installFromRemotePackageURL(ctx, "https://packages.example.invalid/package.zip"), "write plugin package")
	missing := filepath.Join(t.TempDir(), "missing")
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, missing)
	}
	require.Error(t, manager.installFromRemotePackageURL(ctx, "https://packages.example.invalid/package.zip"))
}

func TestPluginCommandsReportIndexAndLocalManifestFailures(t *testing.T) {
	t.Setenv(EnvPluginsDir, t.TempDir())
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	root := os.Getenv(EnvPluginsDir)
	http.DefaultTransport = managerBoundaryTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("index offline") })
	ctx := newTestContext()
	command := newListRemoteCommand()
	ctx.EnterCommand(command)
	require.ErrorContains(t, command.Run(ctx, nil), "failed to fetch remote plugin index")
	require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), []byte("{"), 0600))
	http.DefaultTransport = managerBoundaryTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewBufferString(`{"plugins":[]}`))}, nil
	})
	command = newListRemoteCommand()
	ctx = newTestContext()
	ctx.EnterCommand(command)
	require.Error(t, command.Run(ctx, nil))
	http.DefaultTransport = managerBoundaryTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("index offline") })
	command = newListCommand()
	ctx = newTestContext()
	ctx.EnterCommand(command)
	require.Error(t, command.Run(ctx, nil))
	manager := &Manager{rootDir: root}
	manager.skipPluginIndexCacheForCLI = true
	manager.printOverwriteIfPluginInstalled(ctx, "demo", "1.0.0")
	require.ErrorContains(t, manager.InstallFromLocalFile(ctx, ""), "source path is empty")
	require.Error(t, displaySearchResults(ctx, manager, "", map[string][]string{"demo": {"run"}}))
	command = newSearchCommand()
	ctx = newTestContext()
	ctx.EnterCommand(command)
	require.Error(t, command.Run(ctx, []string{"run"}))
	require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"plugins":{"demo":{"name":"demo","version":"1.0.0","description":"local description"}}}`), 0600))
	var out, stderr bytes.Buffer
	ctx = cli.NewCommandContext(&out, &stderr)
	require.NoError(t, displaySearchResults(ctx, manager, "", map[string][]string{"demo": {"run"}}))
	require.Contains(t, out.String(), "local description")
	require.Contains(t, stderr.String(), "Warning:")
	http.DefaultTransport = managerBoundaryTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewBufferString(`{"plugins":[{"name":"demo","versions":{"2.0.0-beta":{}}}]}`))}, nil
	})
	manager.skipPluginIndexCacheForCLI = true
	out.Reset()
	require.NoError(t, displaySearchResults(ctx, manager, "", map[string][]string{"demo": {"run"}}))
	require.Contains(t, out.String(), "2.0.0-beta")
	require.Contains(t, out.String(), "Yes")
	out.Reset()
	require.NoError(t, displayRemotePlugins(ctx, &Index{Plugins: []PluginInfo{{Name: "empty"}}}, &LocalManifest{}))
	require.Contains(t, out.String(), "N/A")
	require.False(t, apiVersionsPresent(nil))
	dir := t.TempDir()
	_, err := readPluginAPIVersionsFromDir(dir)
	require.Error(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{"), 0600))
	_, err = readPluginAPIVersionsFromDir(dir)
	require.Error(t, err)
	for _, raw := range []string{"{", `{"metadata":[]}`, `{"linux-amd64":[]}`} {
		var version VersionInfo
		require.Error(t, json.Unmarshal([]byte(raw), &version))
	}
	_, err = validateInstallArgs([]string{"", ""}, "")
	require.Error(t, err)
}

func TestUpgradePrereleaseAndMultipleInstallFailures(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	t.Setenv(EnvPluginsDir, t.TempDir())
	http.DefaultTransport = managerBoundaryTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewBufferString(`{"plugins":[{"name":"demo","versions":{"2.0.0-beta":{}}},{"name":"empty","versions":{}}]}`))}, nil
	})
	manager := &Manager{rootDir: os.Getenv(EnvPluginsDir)}
	require.NoError(t, manager.saveLocalManifest(&LocalManifest{Plugins: map[string]LocalPlugin{"demo": {Name: "demo", Version: "1.0.0"}, "empty": {Name: "empty", Version: "1.0.0"}}}))
	var out bytes.Buffer
	ctx := cli.NewCommandContext(&out, io.Discard)
	require.Error(t, manager.Upgrade(ctx, "demo", true))
	require.Contains(t, out.String(), "pre-release")
	require.Error(t, manager.Upgrade(ctx, "empty", false))
	out.Reset()
	require.Error(t, manager.UpdateAll(ctx, true))
	require.Contains(t, out.String(), "pre-release")
	require.Error(t, executeInstall(ctx, []string{"empty", "missing"}, "", "", false, ""))
	command := newUpdateCommand()
	ctx = newTestContext()
	ctx.EnterCommand(command)
	ctx.Flags().Get("enable-pre").SetAssigned(true)
	ctx.Flags().Get("name").SetValue("demo")
	ctx.Flags().Get("name").SetAssigned(true)
	require.Error(t, command.Run(ctx, nil))
}

func TestZipRejectsUnrepresentableDeclaredSize(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	_, err := writer.CreateRaw(&zip.FileHeader{Name: "huge", Method: zip.Store, UncompressedSize64: 1 << 63})
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	path := filepath.Join(t.TempDir(), "huge.zip")
	require.NoError(t, os.WriteFile(path, archive.Bytes(), 0600))
	require.ErrorContains(t, unzip(path, t.TempDir()), "unsupported size")
}

func TestPluginStagingFailuresPreserveInputPackage(t *testing.T) {
	archive := createTestPluginArchive(t, "demo", "1.0.0", "demo")
	checksum, err := calculateSHA256FromBytes(archive)
	require.NoError(t, err)
	platform := &PlatformInfo{URL: "https://packages.example.invalid/demo.tgz", Checksum: checksum}
	t.Run("temporary directory unavailable", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing")
		for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
			t.Setenv(name, missing)
		}
		_, err := (&Manager{}).downloadAndVerifyPlugin(newTestContext(), platform, "demo", "1.0.0")
		require.Error(t, err)
	})
	blocked := filepath.Join(t.TempDir(), "plugins")
	require.NoError(t, os.WriteFile(blocked, []byte("preserve"), 0600))
	manager := &Manager{rootDir: blocked, httpTransport: managerBoundaryTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(archive)), Header: make(http.Header)}, nil
	})}
	plugin := &PluginInfo{Name: "demo", Versions: map[string]VersionInfo{"1.0.0": {Platforms: map[string]PlatformInfo{GetCurrentPlatform(): *platform}}}}
	require.ErrorContains(t, manager.installPlugin(newTestContext(), plugin, "1.0.0", false, false), "create plugin staging directory")
	data, err := os.ReadFile(blocked)
	require.NoError(t, err)
	require.Equal(t, "preserve", string(data))
}

func TestPluginPathsReportRemovedWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	removed := t.TempDir()
	t.Chdir(removed)
	if err := os.Remove(removed); err != nil {
		require.Equal(t, "windows", runtime.GOOS)
		path, err := expandPluginSourcePath("package.zip")
		require.NoError(t, err)
		require.Equal(t, filepath.Join(removed, "package.zip"), path)
		return
	}
	workingDirectory, workingError := os.Getwd()
	if workingError == nil {
		path, err := expandPluginSourcePath("package.zip")
		require.NoError(t, err)
		require.Equal(t, filepath.Join(workingDirectory, "package.zip"), path)
		return
	}
	_, err := expandPluginSourcePath("package.zip")
	require.ErrorContains(t, err, "resolve source path")
	_, err = resolvePluginInstallDir("plugins", "demo")
	require.ErrorContains(t, err, "resolve plugin root directory")
	_, err = validateInstalledPluginPath(root, "demo", "relative-path")
	require.ErrorContains(t, err, "resolve installed plugin path")
}
