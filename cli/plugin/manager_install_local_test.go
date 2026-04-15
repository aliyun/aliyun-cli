package plugin

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyDirTree(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")

	require.NoError(t, os.MkdirAll(filepath.Join(src, "nested"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "b.txt"), []byte("beta"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("alpha"), 0644))

	require.NoError(t, copyDirTree(src, dst))

	b, err := os.ReadFile(filepath.Join(dst, "nested", "b.txt"))
	require.NoError(t, err)
	assert.Equal(t, "beta", string(b))
	a, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "alpha", string(a))
}

func TestCopyDirTree_emptyTree(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "empty-out")
	require.NoError(t, os.MkdirAll(filepath.Join(src, "child"), 0755))

	require.NoError(t, copyDirTree(src, dst))

	st, err := os.Stat(filepath.Join(dst, "child"))
	require.NoError(t, err)
	assert.True(t, st.IsDir())
}

func TestCopyDirTree_preservesFilePerm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits differ on Windows")
	}
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "perm-out")
	p := filepath.Join(src, "secret")
	require.NoError(t, os.WriteFile(p, []byte("x"), 0600))

	require.NoError(t, copyDirTree(src, dst))

	st, err := os.Stat(filepath.Join(dst, "secret"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), st.Mode().Perm())
}

func TestPromoteExtractedPlugin_rename(t *testing.T) {
	root := t.TempDir()
	tmpExtract := filepath.Join(root, "_extract_me")
	require.NoError(t, os.MkdirAll(tmpExtract, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpExtract, "manifest.json"), []byte("{}"), 0644))

	mgr := &Manager{rootDir: root}
	finalDir, err := mgr.promoteExtractedPlugin(tmpExtract, "myplugin")
	require.NoError(t, err)

	want := filepath.Join(root, "myplugin")
	assert.Equal(t, want, finalDir)
	_, err = os.Stat(filepath.Join(want, "manifest.json"))
	require.NoError(t, err)
	_, err = os.Stat(tmpExtract)
	assert.True(t, os.IsNotExist(err), "temp extract dir should be gone after rename")
}

func TestManager_InstallFromLocalFile(t *testing.T) {
	t.Run("Success from tar.gz", func(t *testing.T) {
		pluginRoot := t.TempDir()
		mgr := &Manager{rootDir: pluginRoot}

		archiveBody := createTestPluginArchive(t, "local-test-plugin", "2.1.0", "local")
		archivePath := filepath.Join(t.TempDir(), "plugin.tar.gz")
		assert.NoError(t, os.WriteFile(archivePath, archiveBody, 0644))

		ctx := newTestContext()
		err := mgr.InstallFromLocalFile(ctx, archivePath, "")
		assert.NoError(t, err)

		manifest, err := mgr.GetLocalManifest()
		assert.NoError(t, err)
		p, ok := manifest.Plugins["local-test-plugin"]
		assert.True(t, ok)
		assert.Equal(t, "2.1.0", p.Version)
		assert.Contains(t, ctx.Stdout().(*bytes.Buffer).String(), "Installing plugin from")
	})

	t.Run("Version flag must match manifest", func(t *testing.T) {
		pluginRoot := t.TempDir()
		mgr := &Manager{rootDir: pluginRoot}

		archiveBody := createTestPluginArchive(t, "local-test-plugin", "2.1.0", "local")
		archivePath := filepath.Join(t.TempDir(), "plugin.tar.gz")
		assert.NoError(t, os.WriteFile(archivePath, archiveBody, 0644))

		ctx := newTestContext()
		err := mgr.InstallFromLocalFile(ctx, archivePath, "9.9.9")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not match package manifest version")
	})

	t.Run("Unsupported extension", func(t *testing.T) {
		pluginRoot := t.TempDir()
		mgr := &Manager{rootDir: pluginRoot}
		badPath := filepath.Join(t.TempDir(), "plugin.txt")
		assert.NoError(t, os.WriteFile(badPath, []byte("x"), 0644))

		ctx := newTestContext()
		err := mgr.InstallFromLocalFile(ctx, badPath, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported package format")
	})
}

func TestNewInstallCommand_Run_NamesAndSourceConflict(t *testing.T) {
	cmd := newInstallCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	namesFlag := ctx.Flags().Get("names")
	assert.NotNil(t, namesFlag)
	namesFlag.SetAssigned(true)
	namesFlag.SetValues([]string{"some-plugin"})

	pkgFlag := ctx.Flags().Get("package")
	assert.NotNil(t, pkgFlag)
	pkgFlag.SetAssigned(true)
	pkgFlag.SetValue("/tmp/x.zip")

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--names cannot be used together with --package")
}

func TestNewInstallCommand_Run_WithPackageFlagSuccess(t *testing.T) {
	cmd := newInstallCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	archiveBody := createTestPluginArchive(t, "cli-package-flag-test", "4.5.6", "x")
	archivePath := filepath.Join(t.TempDir(), "plugin-package-cmd.tar.gz")
	assert.NoError(t, os.WriteFile(archivePath, archiveBody, 0644))

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	pkgFlag := ctx.Flags().Get("package")
	assert.NotNil(t, pkgFlag)
	pkgFlag.SetAssigned(true)
	pkgFlag.SetValue(archivePath)

	err := cmd.Run(ctx, []string{})
	assert.NoError(t, err)

	mgr, err := NewManager()
	assert.NoError(t, err)
	manifest, err := mgr.GetLocalManifest()
	assert.NoError(t, err)
	p, ok := manifest.Plugins["cli-package-flag-test"]
	assert.True(t, ok)
	assert.Equal(t, "4.5.6", p.Version)
	assert.Contains(t, stdout.String(), "Installing plugin from")
}

func TestManager_InstallFromPackage_RemoteURL(t *testing.T) {
	pluginRoot := t.TempDir()
	mgr := &Manager{rootDir: pluginRoot}
	archiveBody := createTestPluginArchive(t, "remote-url-plugin", "7.8.9", "x")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(archiveBody)
	}))
	defer srv.Close()
	archiveURL := srv.URL + "/pkgs/remote-url-plugin/7.8.9/plugin.tgz"

	ctx := newTestContext()
	err := mgr.InstallFromPackage(ctx, archiveURL, "")
	require.NoError(t, err)

	manifest, err := mgr.GetLocalManifest()
	require.NoError(t, err)
	p, ok := manifest.Plugins["remote-url-plugin"]
	require.True(t, ok)
	assert.Equal(t, "7.8.9", p.Version)
	out := ctx.Stdout().(*bytes.Buffer).String()
	assert.Contains(t, out, "Downloading plugin package from")
	assert.Contains(t, out, "Installing plugin from")
}

func TestManager_InstallFromPackage_RemoteURLBadSuffix(t *testing.T) {
	mgr := &Manager{rootDir: t.TempDir()}
	ctx := newTestContext()
	err := mgr.InstallFromPackage(ctx, "https://example.com/plugins/nosuffix", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package URL path must end")
}

func TestManager_installFromRemotePackageURL_invalidSchemeAndPath(t *testing.T) {
	mgr := &Manager{rootDir: t.TempDir()}
	ctx := newTestContext()

	err := mgr.installFromRemotePackageURL(ctx, "ftp://example.com/pkg.tgz", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package URL must use http or https")

	err = mgr.installFromRemotePackageURL(ctx, "https://example.com/not-an-archive", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package URL path must end with .zip, .tar.gz, or .tgz")
}

func TestManager_installFromRemotePackageURL_downloadErrors(t *testing.T) {
	mgr := &Manager{rootDir: t.TempDir()}
	ctx := newTestContext()

	t.Run("http get fails", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}))
		base := srv.URL
		srv.Close()
		err := mgr.installFromRemotePackageURL(ctx, base+"/plugin.tgz", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "download plugin package:")
	})

	t.Run("non-OK status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()
		err := mgr.installFromRemotePackageURL(ctx, srv.URL+"/missing.tgz", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "download plugin package: status 404")
	})
}

func TestInstallFromPackageFile_saveLocalManifestFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only manifest.json via chmod is not reliable on Windows")
	}
	root := t.TempDir()
	mani := filepath.Join(root, "manifest.json")
	require.NoError(t, os.WriteFile(mani, []byte(`{"plugins":{}}`), 0644))
	require.NoError(t, os.Chmod(mani, 0444))
	defer func() { _ = os.Chmod(mani, 0644) }()

	mgr := &Manager{rootDir: root}
	archiveBody := createTestPluginArchive(t, "save-fail-pl", "2.0.0", "x")
	archivePath := filepath.Join(t.TempDir(), "save-fail.tgz")
	require.NoError(t, os.WriteFile(archivePath, archiveBody, 0644))

	ctx := newTestContext()
	err := mgr.installFromPackageFile(ctx, archivePath, "", archivePath)
	require.Error(t, err)
	lowered := strings.ToLower(err.Error())
	require.True(t,
		strings.Contains(lowered, "permission denied") ||
			strings.Contains(lowered, "operation not permitted"),
		"unexpected error from saveLocalManifest: %v", err,
	)
}

func TestFindInstalledPluginInManifest(t *testing.T) {
	m := &LocalManifest{Plugins: map[string]LocalPlugin{
		"aliyun-cli-x": {Name: "aliyun-cli-x"},
	}}
	n, lp, ok := FindInstalledPluginInManifest(m, "aliyun-cli-x")
	require.True(t, ok)
	assert.Equal(t, "aliyun-cli-x", n)
	assert.Equal(t, "aliyun-cli-x", lp.Name)

	n, _, ok = FindInstalledPluginInManifest(m, "x")
	require.True(t, ok)
	assert.Equal(t, "aliyun-cli-x", n)

	_, _, ok = FindInstalledPluginInManifest(m, "nosuch")
	assert.False(t, ok)
	_, _, ok = FindInstalledPluginInManifest(nil, "x")
	assert.False(t, ok)
}

func TestIsRemotePluginPackageRef(t *testing.T) {
	cases := []struct {
		ref  string
		want bool
	}{
		{"/local/path/plugin.tgz", false},
		{"file:///tmp/a.tgz", false},
		{"ftp://example.com/a.tgz", false},
		{"https://example.com/pkgs/foo.tgz", true},
		{"HTTP://EXAMPLE.COM/X.ZIP", true},
		{"https://example.com/nosuffix", false},
		{"https://example.com/foo.tar.gz", true},
		{"  https://mirror.example/x.tgz  ", true},
		{"https:///x.tgz", false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q", tc.ref), func(t *testing.T) {
			assert.Equal(t, tc.want, isRemotePluginPackageRef(tc.ref), "ref=%q", tc.ref)
		})
	}
}

func TestReadPluginManifestFromDir(t *testing.T) {
	t.Run("missing manifest", func(t *testing.T) {
		d := t.TempDir()
		_, err := readPluginManifestFromDir(d)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugin package: manifest.json not found")
	})
	t.Run("invalid json", func(t *testing.T) {
		d := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(d, "manifest.json"), []byte("{"), 0644))
		_, err := readPluginManifestFromDir(d)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugin manifest")
	})
	t.Run("empty name", func(t *testing.T) {
		d := t.TempDir()
		raw := `{"name":"  \t  ","version":"1.0.0"}`
		require.NoError(t, os.WriteFile(filepath.Join(d, "manifest.json"), []byte(raw), 0644))
		_, err := readPluginManifestFromDir(d)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugin manifest: name is empty")
	})
	t.Run("empty version", func(t *testing.T) {
		d := t.TempDir()
		raw := `{"name":"aliyun-cli-test","version":"  "}`
		require.NoError(t, os.WriteFile(filepath.Join(d, "manifest.json"), []byte(raw), 0644))
		_, err := readPluginManifestFromDir(d)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugin manifest: version is empty")
	})
	t.Run("missing name field", func(t *testing.T) {
		d := t.TempDir()
		raw := `{"version":"1.0.0"}`
		require.NoError(t, os.WriteFile(filepath.Join(d, "manifest.json"), []byte(raw), 0644))
		_, err := readPluginManifestFromDir(d)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugin manifest: name is empty")
	})
	t.Run("missing version field", func(t *testing.T) {
		d := t.TempDir()
		raw := `{"name":"aliyun-cli-test"}`
		require.NoError(t, os.WriteFile(filepath.Join(d, "manifest.json"), []byte(raw), 0644))
		_, err := readPluginManifestFromDir(d)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugin manifest: version is empty")
	})
	t.Run("success", func(t *testing.T) {
		d := t.TempDir()
		raw := `{"name":"aliyun-cli-test","version":"1.0.0","command":"test"}`
		require.NoError(t, os.WriteFile(filepath.Join(d, "manifest.json"), []byte(raw), 0644))
		pm, err := readPluginManifestFromDir(d)
		require.NoError(t, err)
		assert.Equal(t, "aliyun-cli-test", pm.Name)
		assert.Equal(t, "1.0.0", pm.Version)
	})
}

func TestExpandPluginSourcePath(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, err := expandPluginSourcePath("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
	t.Run("whitespace_only", func(t *testing.T) {
		_, err := expandPluginSourcePath("   \t  ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
	t.Run("tilde_prefix", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		wantFile := filepath.Join(testHome, "my-plugin.tgz")
		require.NoError(t, os.WriteFile(wantFile, []byte("x"), 0644))

		got, err := expandPluginSourcePath("~/my-plugin.tgz")
		require.NoError(t, err)
		want, err := filepath.Abs(wantFile)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
	t.Run("trim_then_abs", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "a.zip")
		require.NoError(t, os.WriteFile(p, []byte("z"), 0644))
		want, err := filepath.Abs(p)
		require.NoError(t, err)
		got, err := expandPluginSourcePath("  " + p + "  ")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}
