package plugin

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstalledPluginLookupAndExecutionFailures(t *testing.T) {
	t.Cleanup(setTestHomeDir(t, t.TempDir()))
	root := t.TempDir()
	t.Setenv(EnvPluginsDir, root)
	manager := &Manager{rootDir: root}
	pluginPath := filepath.Join(root, "demo")
	require.NoError(t, os.Mkdir(pluginPath, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(pluginPath, "demo"), []byte("invalid executable"), 0700))
	require.NoError(t, manager.saveLocalManifest(&LocalManifest{Plugins: map[string]LocalPlugin{"demo": {Name: "demo", Command: "demo", Version: "1.0.0", Path: pluginPath}}}))
	installed, name, err := IsPluginInstalled("demo")
	require.NoError(t, err)
	require.True(t, installed)
	require.Equal(t, "demo", name)
	found, err := ExecutePlugin("demo", nil, newTestContext())
	require.True(t, found)
	require.ErrorContains(t, err, "plugin execution failed")
	require.ErrorContains(t, ValidateLocalPluginCliVersion("", nil), "plugin is nil")
	require.NoError(t, ValidateLocalPluginCliVersion("", &LocalPlugin{Name: "demo", Version: "1.0.0"}))
	require.NoError(t, os.WriteFile(filepath.Join(pluginPath, "demo.exe"), []byte("Windows fixture"), 0600))
	resolved, err := resolvePluginBinaryPath(&LocalPlugin{Name: "demo", Path: pluginPath})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(pluginPath, "demo.exe"), resolved)
	blocked := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocked, []byte("keep"), 0600))
	t.Setenv(EnvPluginsDir, blocked)
	installed, _, err = IsPluginInstalled("demo")
	require.Error(t, err)
	require.False(t, installed)
	_, ok := InstalledPluginType("demo")
	require.False(t, ok)
	_, _, err = InstalledPluginPackageVersion("demo")
	require.Error(t, err)
	require.Error(t, ValidatePluginCliVersion("demo"))
	require.True(t, IsProfileRequiredForCommand("demo"))
	require.Error(t, manager.ApplySourceBaseOverride("https://"))
	require.False(t, matchPluginAlias(LocalPlugin{CommandAliases: []string{"demo"}}, " "))
	_, err = (&Manager{rootDir: filepath.Join(root, "invalid\x00")}).GetLocalManifest()
	require.Error(t, err)
}

func TestPluginExitStatusChild(t *testing.T) {
	switch os.Getenv("ALIYUN_CLI_TEST_PROCESS_ROLE") {
	case "driver":
		err := runPluginCommand(os.Args[0], []string{"-test.run=^TestPluginExitStatusChild$"}, io.Discard, io.Discard, append(os.Environ(), "ALIYUN_CLI_TEST_PROCESS_ROLE=worker"))
		if err != nil {
			os.Exit(90)
		}
		os.Exit(91)
	case "worker":
		os.Exit(7)
	}
}

func TestPluginProcessPreservesExitStatus(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=^TestPluginExitStatusChild$")
	command.Env = append(os.Environ(), "ALIYUN_CLI_TEST_PROCESS_ROLE=driver")
	output, err := command.CombinedOutput()
	var exitError *exec.ExitError
	require.ErrorAs(t, err, &exitError, string(output))
	require.Equal(t, 7, exitError.ExitCode())
}
