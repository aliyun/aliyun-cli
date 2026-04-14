package plugin

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
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
		assert.Contains(t, err.Error(), "unsupported archive format")
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

	sourceFlag := ctx.Flags().Get("source")
	assert.NotNil(t, sourceFlag)
	sourceFlag.SetAssigned(true)
	sourceFlag.SetValue("/tmp/x.zip")

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--names cannot be used together with --source")
}

func TestNewInstallCommand_Run_WithSourceFlagSuccess(t *testing.T) {
	cmd := newInstallCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	archiveBody := createTestPluginArchive(t, "cli-local-cmd-test", "1.2.3", "x")
	archivePath := filepath.Join(t.TempDir(), "plugin-local-cmd.tar.gz")
	assert.NoError(t, os.WriteFile(archivePath, archiveBody, 0644))

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	sourceFlag := ctx.Flags().Get("source")
	assert.NotNil(t, sourceFlag)
	sourceFlag.SetAssigned(true)
	sourceFlag.SetValue(archivePath)

	err := cmd.Run(ctx, []string{})
	assert.NoError(t, err)

	mgr, err := NewManager()
	assert.NoError(t, err)
	manifest, err := mgr.GetLocalManifest()
	assert.NoError(t, err)
	p, ok := manifest.Plugins["cli-local-cmd-test"]
	assert.True(t, ok)
	assert.Equal(t, "1.2.3", p.Version)
	assert.Contains(t, stdout.String(), "Installing plugin from")
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
