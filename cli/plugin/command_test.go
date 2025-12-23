package plugin

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
)

// setTestHomeDir sets the test home directory for cross-platform testing.
// Returns a cleanup function that restores the original environment variables.
func setTestHomeDir(t *testing.T, testHome string) func() {
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")

	os.Setenv("HOME", testHome)
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", testHome)
	}

	return func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
		if runtime.GOOS == "windows" {
			if originalUserProfile != "" {
				os.Setenv("USERPROFILE", originalUserProfile)
			} else {
				os.Unsetenv("USERPROFILE")
			}
		}
	}
}

func TestNewPluginCommand(t *testing.T) {
	cmd := NewPluginCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "plugin", cmd.Name)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Usage)

	assert.NotNil(t, cmd.GetSubCommand("list"), "Should have list subcommand")
	assert.NotNil(t, cmd.GetSubCommand("install"), "Should have install subcommand")
	assert.NotNil(t, cmd.GetSubCommand("install-all"), "Should have install-all subcommand")
	assert.NotNil(t, cmd.GetSubCommand("uninstall"), "Should have uninstall subcommand")
	assert.NotNil(t, cmd.GetSubCommand("update"), "Should have update subcommand")
}

func TestNewPluginCommand_Run(t *testing.T) {
	cmd := NewPluginCommand()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	err := cmd.Run(ctx, []string{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "command missing")
}

func TestNewListCommand(t *testing.T) {
	cmd := newListCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Name)
	assert.NotEmpty(t, cmd.Short)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := t.TempDir()
	os.Setenv("HOME", testHome)

	err := cmd.Run(ctx, []string{})
	assert.NoError(t, err)

	// Even with no plugins, header should be printed
	output := stdout.String()
	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Version")
	assert.Contains(t, output, "Description")
}

func TestNewListCommand_WithPlugins(t *testing.T) {
	cmd := newListCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
	os.MkdirAll(filepath.Dir(manifestPath), 0755)

	pluginPath := filepath.Join(testHome, ".aliyun", "plugins", "aliyun-cli-fc")
	manifest := LocalManifest{
		Plugins: map[string]LocalPlugin{
			"aliyun-cli-fc": {
				Name:        "aliyun-cli-fc",
				Version:     "1.0.0",
				Description: "FC plugin",
				Path:        pluginPath,
			},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	assert.NoError(t, err)
	os.WriteFile(manifestPath, manifestJSON, 0644)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	err = cmd.Run(ctx, []string{})
	assert.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Version")
	assert.Contains(t, output, "Description")
	assert.Contains(t, output, "aliyun-cli-fc")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "FC plugin")
}

func TestNewInstallCommand(t *testing.T) {
	cmd := newInstallCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "install", cmd.Name)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Usage)

	flags := cmd.Flags()
	nameFlag := flags.Get("name")
	assert.NotNil(t, nameFlag)
	assert.True(t, nameFlag.Required)

	versionFlag := flags.Get("version")
	assert.NotNil(t, versionFlag)
	assert.False(t, versionFlag.Required)
}

func TestNewInstallCommand_Run(t *testing.T) {
	cmd := newInstallCommand()

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := t.TempDir()
	os.Setenv("HOME", testHome)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
}

func TestNewInstallCommand_Run_WithNameFlag(t *testing.T) {
	cmd := newInstallCommand()

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := t.TempDir()
	os.Setenv("HOME", testHome)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	nameFlag := ctx.Flags().Get("name")
	assert.NotNil(t, nameFlag)
	nameFlag.SetAssigned(true)
	nameFlag.SetValue("nonexistent-plugin-xyz-123")

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-plugin-xyz-123 not found")
}

func TestNewInstallCommand_Run_WithNameAndVersionFlags(t *testing.T) {
	cmd := newInstallCommand()

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := t.TempDir()
	os.Setenv("HOME", testHome)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	nameFlag := ctx.Flags().Get("name")
	assert.NotNil(t, nameFlag)
	nameFlag.SetAssigned(true)
	nameFlag.SetValue("nonexistent-plugin-xyz-123")

	versionFlag := ctx.Flags().Get("version")
	assert.NotNil(t, versionFlag)
	versionFlag.SetAssigned(true)
	versionFlag.SetValue("1.0.0")

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-plugin-xyz-123 not found")
}

func TestNewInstallCommand_Run_WithVersionFlagOnly(t *testing.T) {
	cmd := newInstallCommand()

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := t.TempDir()
	os.Setenv("HOME", testHome)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	versionFlag := ctx.Flags().Get("version")
	assert.NotNil(t, versionFlag)
	versionFlag.SetAssigned(true)
	versionFlag.SetValue("1.0.0")

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name flag is required")
}

func TestNewInstallCommand_Run_FlagValueAssignment(t *testing.T) {
	cmd := newInstallCommand()

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := t.TempDir()
	os.Setenv("HOME", testHome)

	t.Run("Name flag value assignment", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)

		nameFlag := ctx.Flags().Get("name")
		nameFlag.SetAssigned(true)
		nameFlag.SetValue("test-plugin-name")

		v, ok := ctx.Flags().GetValue("name")
		assert.True(t, ok, "name flag should be retrievable")
		assert.Equal(t, "test-plugin-name", v, "name flag value should match")
	})

	t.Run("Version flag value assignment", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)

		versionFlag := ctx.Flags().Get("version")
		versionFlag.SetAssigned(true)
		versionFlag.SetValue("2.0.0")

		v, ok := ctx.Flags().GetValue("version")
		assert.True(t, ok, "version flag should be retrievable")
		assert.Equal(t, "2.0.0", v, "version flag value should match")
	})

	t.Run("Both flags value assignment", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)

		nameFlag := ctx.Flags().Get("name")
		nameFlag.SetAssigned(true)
		nameFlag.SetValue("another-plugin")

		versionFlag := ctx.Flags().Get("version")
		versionFlag.SetAssigned(true)
		versionFlag.SetValue("3.0.0")

		name, nameOk := ctx.Flags().GetValue("name")
		assert.True(t, nameOk)
		assert.Equal(t, "another-plugin", name)

		version, versionOk := ctx.Flags().GetValue("version")
		assert.True(t, versionOk)
		assert.Equal(t, "3.0.0", version)
	})

}

func TestNewInstallAllCommand(t *testing.T) {
	cmd := newInstallAllCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "install-all", cmd.Name)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Usage)
}

func TestNewUninstallCommand(t *testing.T) {
	cmd := newUninstallCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "uninstall", cmd.Name)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Usage)

	flags := cmd.Flags()
	nameFlag := flags.Get("name")
	assert.NotNil(t, nameFlag)
	assert.True(t, nameFlag.Required)
}

func TestNewUpdateCommand(t *testing.T) {
	cmd := newUpdateCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "update", cmd.Name)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Usage)

	flags := cmd.Flags()
	nameFlag := flags.Get("name")
	assert.NotNil(t, nameFlag)
	assert.False(t, nameFlag.Required) // name is optional for update
}

func TestNewUpdateCommand_Run_WithoutName(t *testing.T) {
	cmd := newUpdateCommand()

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := t.TempDir()
	os.Setenv("HOME", testHome)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	err := cmd.Run(ctx, []string{})
	assert.NoError(t, err)
}

func TestNewUpdateCommand_Run_WithName(t *testing.T) {
	cmd := newUpdateCommand()

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := t.TempDir()
	os.Setenv("HOME", testHome)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	nameFlag := ctx.Flags().Get("name")
	if nameFlag != nil {
		nameFlag.SetAssigned(true)
		nameFlag.SetValue("abc")
	}

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "abc not installed")
}
