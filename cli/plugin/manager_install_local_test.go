package plugin

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromoteExtractedPlugin_rename(t *testing.T) {
	root := t.TempDir()
	tmpExtract := filepath.Join(root, "_extract_me")
	require.NoError(t, os.MkdirAll(tmpExtract, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpExtract, "manifest.json"), []byte("{}"), 0644))

	mgr := &Manager{rootDir: root}
	promotion, err := mgr.promoteExtractedPlugin(tmpExtract, "myplugin")
	require.NoError(t, err)
	promotion.commit()

	want := filepath.Join(root, "myplugin")
	assert.Equal(t, want, promotion.finalPath)
	_, err = os.Stat(filepath.Join(want, "manifest.json"))
	require.NoError(t, err)
	_, err = os.Stat(tmpExtract)
	assert.True(t, os.IsNotExist(err), "temp extract dir should be gone after rename")
}

func TestPromoteExtractedPlugin_rollbackRestoresExistingDirectory(t *testing.T) {
	root := t.TempDir()
	finalDir := filepath.Join(root, "myplugin")
	require.NoError(t, os.MkdirAll(finalDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(finalDir, "version"), []byte("old"), 0644))

	tmpExtract := filepath.Join(root, "_extract_me")
	require.NoError(t, os.MkdirAll(tmpExtract, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpExtract, "version"), []byte("new"), 0644))

	mgr := &Manager{rootDir: root}
	promotion, err := mgr.promoteExtractedPlugin(tmpExtract, "myplugin")
	require.NoError(t, err)
	require.NoError(t, promotion.rollback())

	version, err := os.ReadFile(filepath.Join(finalDir, "version"))
	require.NoError(t, err)
	assert.Equal(t, "old", string(version))
}

func TestValidatePluginNameRejectsPathValues(t *testing.T) {
	for _, name := range []string{
		"aliyun-cli-ecs",
		"plugin_name.v2",
		"插件名称",
	} {
		t.Run("valid_"+name, func(t *testing.T) {
			require.NoError(t, validatePluginName(name))
		})
	}

	for _, name := range []string{
		"", "   ", ".", "..", "../outside", `..\outside`,
		"nested/plugin", `nested\plugin`, "/absolute", `C:\absolute`, "C:relative", "bad\nname",
	} {
		t.Run(fmt.Sprintf("invalid_%q", name), func(t *testing.T) {
			require.Error(t, validatePluginName(name))
		})
	}
}

func TestPromoteExtractedPluginRejectsEscapingName(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "plugins")
	require.NoError(t, os.MkdirAll(root, 0755))
	outside := filepath.Join(parent, "outside")
	require.NoError(t, os.MkdirAll(outside, 0755))
	sentinel := filepath.Join(outside, "keep")
	require.NoError(t, os.WriteFile(sentinel, []byte("keep"), 0644))

	tmpExtract := filepath.Join(t.TempDir(), "extract")
	require.NoError(t, os.MkdirAll(tmpExtract, 0755))
	mgr := &Manager{rootDir: root}
	_, err := mgr.promoteExtractedPlugin(tmpExtract, "../outside")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid plugin name")
	assert.FileExists(t, sentinel)
	assert.DirExists(t, tmpExtract)
}

func TestManager_InstallFromLocalFile(t *testing.T) {
	t.Run("Success from tar.gz", func(t *testing.T) {
		pluginRoot := t.TempDir()
		mgr := &Manager{rootDir: pluginRoot}

		archiveBody := createTestPluginArchive(t, "local-test-plugin", "2.1.0", "local")
		archivePath := filepath.Join(t.TempDir(), "plugin.tar.gz")
		assert.NoError(t, os.WriteFile(archivePath, archiveBody, 0644))

		ctx := newTestContext()
		err := mgr.InstallFromLocalFile(ctx, archivePath)
		assert.NoError(t, err)

		manifest, err := mgr.GetLocalManifest()
		assert.NoError(t, err)
		p, ok := manifest.Plugins["local-test-plugin"]
		assert.True(t, ok)
		assert.Equal(t, "2.1.0", p.Version)
		assert.Contains(t, ctx.Stdout().(*bytes.Buffer).String(), "Installing plugin from")
	})

	t.Run("Unsupported extension", func(t *testing.T) {
		pluginRoot := t.TempDir()
		mgr := &Manager{rootDir: pluginRoot}
		badPath := filepath.Join(t.TempDir(), "plugin.txt")
		assert.NoError(t, os.WriteFile(badPath, []byte("x"), 0644))

		ctx := newTestContext()
		err := mgr.InstallFromLocalFile(ctx, badPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported package format")
	})

	t.Run("Overwrite prints note when same plugin name exists", func(t *testing.T) {
		pluginRoot := t.TempDir()
		mgr := &Manager{rootDir: pluginRoot}
		ctx := newTestContext()

		v1 := createTestPluginArchive(t, "overwrite-test", "1.0.0", "x")
		p1 := filepath.Join(t.TempDir(), "overwrite-a.tgz")
		require.NoError(t, os.WriteFile(p1, v1, 0644))
		require.NoError(t, mgr.InstallFromLocalFile(ctx, p1))

		v2 := createTestPluginArchive(t, "overwrite-test", "2.0.0", "x")
		p2 := filepath.Join(t.TempDir(), "overwrite-b.tgz")
		require.NoError(t, os.WriteFile(p2, v2, 0644))
		require.NoError(t, mgr.InstallFromLocalFile(ctx, p2))

		out := ctx.Stdout().(*bytes.Buffer).String()
		assert.Contains(t, out, `plugin "overwrite-test" is already installed`)
		assert.Contains(t, out, "continuing will replace it with version 2.0.0")
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
	assert.Contains(t, err.Error(), "--name/--names cannot be used together with --package")
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

	type requestDetails struct {
		username string
		password string
		token    string
	}
	requests := make(chan requestDetails, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, _ := r.BasicAuth()
		requests <- requestDetails{
			username: username,
			password: password,
			token:    r.URL.Query().Get("token"),
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(archiveBody)
	}))
	defer srv.Close()
	archiveURL := strings.Replace(srv.URL, "http://", "http://log-user:log-password@", 1) +
		"/pkgs/remote-url-plugin/7.8.9/plugin.tgz?token=query-secret&signature=signature-secret#fragment-secret"

	ctx := newTestContext()
	err := mgr.InstallFromPackage(ctx, archiveURL)
	require.NoError(t, err)
	request := <-requests
	assert.Equal(t, "log-user", request.username)
	assert.Equal(t, "log-password", request.password)
	assert.Equal(t, "query-secret", request.token)

	manifest, err := mgr.GetLocalManifest()
	require.NoError(t, err)
	p, ok := manifest.Plugins["remote-url-plugin"]
	require.True(t, ok)
	assert.Equal(t, "7.8.9", p.Version)
	out := ctx.Stdout().(*bytes.Buffer).String()
	assert.Contains(t, out, "Downloading plugin package from")
	assert.Contains(t, out, "Installing plugin from")
	assert.Contains(t, out, srv.URL+"/pkgs/remote-url-plugin/7.8.9/plugin.tgz")
	assert.NotContains(t, out, "log-user")
	assert.NotContains(t, out, "log-password")
	assert.NotContains(t, out, "query-secret")
	assert.NotContains(t, out, "signature-secret")
	assert.NotContains(t, out, "fragment-secret")
}

func TestManager_InstallFromPackage_RemoteURLBadSuffix(t *testing.T) {
	mgr := &Manager{rootDir: t.TempDir()}
	ctx := newTestContext()
	err := mgr.InstallFromPackage(ctx, "https://example.com/plugins/nosuffix")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package URL path must end")
}

func TestManager_installFromRemotePackageURL_invalidSchemeAndPath(t *testing.T) {
	mgr := &Manager{rootDir: t.TempDir()}
	ctx := newTestContext()

	err := mgr.installFromRemotePackageURL(ctx, "ftp://example.com/pkg.tgz")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package URL must use http or https")

	err = mgr.installFromRemotePackageURL(ctx, "https://example.com/not-an-archive")
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
		rawURL := strings.Replace(base, "http://", "http://error-user:error-password@", 1) +
			"/plugin.tgz?token=error-token#fragment-secret"
		err := mgr.installFromRemotePackageURL(ctx, rawURL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "download plugin package from "+base+"/plugin.tgz:")
		assert.NotContains(t, err.Error(), "error-user")
		assert.NotContains(t, err.Error(), "error-password")
		assert.NotContains(t, err.Error(), "error-token")
		assert.NotContains(t, err.Error(), "fragment-secret")
	})

	t.Run("non-OK status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()
		err := mgr.installFromRemotePackageURL(ctx, srv.URL+"/missing.tgz")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "download plugin package: status 404")
	})
}

func TestInstallFromPackageFile_saveLocalManifestFailsRollsBack(t *testing.T) {
	root := t.TempDir()
	mgr := &Manager{rootDir: root}
	ctx := newTestContext()

	v1 := createTestPluginArchive(t, "save-fail-pl", "1.0.0", "x")
	v1Path := filepath.Join(t.TempDir(), "v1.tgz")
	require.NoError(t, os.WriteFile(v1Path, v1, 0644))
	require.NoError(t, mgr.installFromPackageFile(ctx, v1Path, v1Path))

	mgr.saveLocalManifestHook = func(*LocalManifest) error {
		return fmt.Errorf("injected manifest commit failure")
	}
	v2 := createTestPluginArchive(t, "save-fail-pl", "2.0.0", "x")
	archivePath := filepath.Join(t.TempDir(), "save-fail.tgz")
	require.NoError(t, os.WriteFile(archivePath, v2, 0644))

	err := mgr.installFromPackageFile(ctx, archivePath, archivePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected manifest commit failure")

	mgr.saveLocalManifestHook = nil
	localManifest, err := mgr.GetLocalManifest()
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", localManifest.Plugins["save-fail-pl"].Version)

	packageManifest, err := readPluginManifestFromDir(filepath.Join(root, "save-fail-pl"))
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", packageManifest.Version)
}

func TestFindInstalledPluginInManifest(t *testing.T) {
	m := &LocalManifest{Plugins: map[string]LocalPlugin{
		"aliyun-cli-x":        {Name: "aliyun-cli-x"},
		"aliyun-cli-hologram": {Name: "aliyun-cli-hologram", Command: "hologram", CommandAliases: []string{"hologres"}},
	}}
	n, lp, ok := FindInstalledPluginInManifest(m, "aliyun-cli-x")
	require.True(t, ok)
	assert.Equal(t, "aliyun-cli-x", n)
	assert.Equal(t, "aliyun-cli-x", lp.Name)

	n, _, ok = FindInstalledPluginInManifest(m, "x")
	require.True(t, ok)
	assert.Equal(t, "aliyun-cli-x", n)

	// alias matches the same plugin as the main command
	n, _, ok = FindInstalledPluginInManifest(m, "hologres")
	require.True(t, ok)
	assert.Equal(t, "aliyun-cli-hologram", n)

	// alias matching is case-insensitive, mirroring matchPluginName
	n, _, ok = FindInstalledPluginInManifest(m, "HOLOGRES")
	require.True(t, ok)
	assert.Equal(t, "aliyun-cli-hologram", n)

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
	t.Run("path name", func(t *testing.T) {
		d := t.TempDir()
		raw := `{"name":"../outside","version":"1.0.0"}`
		require.NoError(t, os.WriteFile(filepath.Join(d, "manifest.json"), []byte(raw), 0644))
		_, err := readPluginManifestFromDir(d)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "single path-safe component")
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
