package plugin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
)

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

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

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
	namesFlag := flags.Get("names")
	assert.NotNil(t, namesFlag)
	assert.False(t, namesFlag.Required)

	versionFlag := flags.Get("version")
	assert.NotNil(t, versionFlag)
	assert.False(t, versionFlag.Required)
}

func TestNewInstallCommand_Run(t *testing.T) {
	cmd := newInstallCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
}

func TestNewInstallCommand_Run_WithNamesFlag(t *testing.T) {
	cmd := newInstallCommand()

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := t.TempDir()
	os.Setenv("HOME", testHome)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	namesFlag := ctx.Flags().Get("names")
	assert.NotNil(t, namesFlag)
	namesFlag.SetAssigned(true)
	namesFlag.SetValues([]string{"nonexistent-plugin-xyz-123"})

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-plugin-xyz-123 not found")
}

func TestNewInstallCommand_Run_WithNamesAndVersionFlags(t *testing.T) {
	cmd := newInstallCommand()

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := t.TempDir()
	os.Setenv("HOME", testHome)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	namesFlag := ctx.Flags().Get("names")
	assert.NotNil(t, namesFlag)
	namesFlag.SetAssigned(true)
	namesFlag.SetValues([]string{"nonexistent-plugin-xyz-123"})

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
	assert.Contains(t, err.Error(), "names flag is required")
}

func TestNewInstallCommand_Run_FlagValueAssignment(t *testing.T) {
	cmd := newInstallCommand()

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := t.TempDir()
	os.Setenv("HOME", testHome)

	t.Run("Names flag value assignment", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)

		namesFlag := ctx.Flags().Get("names")
		namesFlag.SetAssigned(true)
		namesFlag.SetValues([]string{"test-plugin-name"})

		namesFlag2 := ctx.Flags().Get("names")
		values := namesFlag2.GetValues()
		assert.NotNil(t, values, "names flag should be retrievable")
		assert.Len(t, values, 1, "names flag should have one value")
		assert.Equal(t, "test-plugin-name", values[0], "names flag value should match")
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

		namesFlag := ctx.Flags().Get("names")
		namesFlag.SetAssigned(true)
		namesFlag.SetValues([]string{"another-plugin"})

		versionFlag := ctx.Flags().Get("version")
		versionFlag.SetAssigned(true)
		versionFlag.SetValue("3.0.0")

		namesFlag2 := ctx.Flags().Get("names")
		values := namesFlag2.GetValues()
		assert.NotNil(t, values)
		assert.Len(t, values, 1)
		assert.Equal(t, "another-plugin", values[0])

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
	assert.False(t, nameFlag.Required)
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
	assert.Contains(t, err.Error(), "plugin abc not found in local manifest")
}

func TestDisplayRemotePlugins(t *testing.T) {
	t.Run("Empty plugin list", func(t *testing.T) {
		index := &Index{
			Plugins: []PluginInfo{},
		}
		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{},
		}

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err := displayRemotePlugins(ctx, index, localManifest)
		assert.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "Total plugins available: 0")
		assert.Contains(t, output, "No plugins available in remote index")
	})

	t.Run("Display plugins with status and preview", func(t *testing.T) {
		index := &Index{
			Plugins: []PluginInfo{
				{
					Name:        "aliyun-cli-ecs",
					Description: "ECS plugin",
					Versions: map[string]VersionInfo{
						"1.0.0": {},
					},
				},
				{
					Name:        "aliyun-cli-fc",
					Description: "FC plugin",
					Versions: map[string]VersionInfo{
						"2.0.0-beta": {},
						"1.0.0":      {},
					},
				},
				{
					Name:        "aliyun-cli-oss",
					Description: "OSS plugin",
					Versions: map[string]VersionInfo{
						"3.0.0": {},
					},
				},
			},
		}
		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"aliyun-cli-ecs": {
					Name:    "aliyun-cli-ecs",
					Version: "1.0.0",
				},
			},
		}

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err := displayRemotePlugins(ctx, index, localManifest)
		assert.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "Total plugins available: 3")
		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "Latest Version")
		assert.Contains(t, output, "Preview")
		assert.Contains(t, output, "Status")
		assert.Contains(t, output, "Local Version")
		assert.Contains(t, output, "Description")

		// Check plugin data
		assert.Contains(t, output, "aliyun-cli-ecs")
		assert.Contains(t, output, "1.0.0")
		assert.Contains(t, output, "Installed")
		assert.Contains(t, output, "ECS plugin")

		assert.Contains(t, output, "aliyun-cli-fc")
		assert.Contains(t, output, "2.0.0-beta") // Latest version (pre-release)
		assert.Contains(t, output, "FC plugin")

		assert.Contains(t, output, "aliyun-cli-oss")
		assert.Contains(t, output, "3.0.0")
		assert.Contains(t, output, "Not installed")
		assert.Contains(t, output, "OSS plugin")
	})

	t.Run("Check preview column for pre-release", func(t *testing.T) {
		index := &Index{
			Plugins: []PluginInfo{
				{
					Name:        "aliyun-cli-stable",
					Description: "Stable plugin",
					Versions: map[string]VersionInfo{
						"1.0.0": {},
					},
				},
				{
					Name:        "aliyun-cli-preview",
					Description: "Preview plugin",
					Versions: map[string]VersionInfo{
						"1.0.0-alpha": {},
					},
				},
			},
		}
		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{},
		}

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err := displayRemotePlugins(ctx, index, localManifest)
		assert.NoError(t, err)

		lines := bytes.Split(stdout.Bytes(), []byte("\n"))

		var stableLine, previewLine string
		for _, line := range lines {
			lineStr := string(line)
			if bytes.Contains(line, []byte("aliyun-cli-stable")) {
				stableLine = lineStr
			}
			if bytes.Contains(line, []byte("aliyun-cli-preview")) {
				previewLine = lineStr
			}
		}

		assert.NotEmpty(t, stableLine, "Should find stable plugin line")
		assert.NotEmpty(t, previewLine, "Should find preview plugin line")

		output := stdout.String()
		assert.Contains(t, output, "Yes") // Preview plugin has "Yes"
		assert.Contains(t, output, "No")  // Stable plugin has "No"
	})

	t.Run("Installed plugins appear first", func(t *testing.T) {
		index := &Index{
			Plugins: []PluginInfo{
				{
					Name:        "aliyun-cli-zzz",
					Description: "ZZZ plugin (not installed)",
					Versions: map[string]VersionInfo{
						"1.0.0": {},
					},
				},
				{
					Name:        "aliyun-cli-aaa",
					Description: "AAA plugin (installed)",
					Versions: map[string]VersionInfo{
						"1.0.0": {},
					},
				},
			},
		}
		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"aliyun-cli-aaa": {
					Name:    "aliyun-cli-aaa",
					Version: "1.0.0",
				},
			},
		}

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err := displayRemotePlugins(ctx, index, localManifest)
		assert.NoError(t, err)

		lines := bytes.Split(stdout.Bytes(), []byte("\n"))

		// Find data lines (skip header lines)
		var dataLines []string
		for i, line := range lines {
			lineStr := string(line)
			if i > 2 && bytes.Contains(line, []byte("aliyun-cli-")) {
				dataLines = append(dataLines, lineStr)
			}
		}

		// The installed plugin (aaa) should appear before not-installed (zzz)
		assert.True(t, len(dataLines) >= 2, "Should have at least 2 data lines")

		var aaaIndex, zzzIndex int = -1, -1
		for i, line := range dataLines {
			if bytes.Contains([]byte(line), []byte("aliyun-cli-aaa")) {
				aaaIndex = i
			}
			if bytes.Contains([]byte(line), []byte("aliyun-cli-zzz")) {
				zzzIndex = i
			}
		}

		assert.True(t, aaaIndex >= 0, "Should find aaa plugin")
		assert.True(t, zzzIndex >= 0, "Should find zzz plugin")
		assert.True(t, aaaIndex < zzzIndex, "Installed plugin (aaa) should appear before not-installed (zzz)")
	})
}

func TestDisplaySearchResult(t *testing.T) {
	t.Run("Display installed plugin", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		mgr, err := NewManager()
		assert.NoError(t, err)

		// Create local manifest
		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)

		localManifest := LocalManifest{
			Plugins: map[string]LocalPlugin{
				"aliyun-cli-ecs": {
					Name:        "aliyun-cli-ecs",
					Version:     "1.0.0",
					Description: "ECS plugin for testing",
					Path:        filepath.Join(testHome, ".aliyun", "plugins", "aliyun-cli-ecs"),
				},
			},
		}
		manifestJSON, err := json.Marshal(localManifest)
		assert.NoError(t, err)
		os.WriteFile(manifestPath, manifestJSON, 0644)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err = displaySearchResult(ctx, mgr, "ecs", "aliyun-cli-ecs")
		assert.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "Command: ecs")
		assert.Contains(t, output, "Plugin: aliyun-cli-ecs")
		assert.Contains(t, output, "Status: Installed")
		assert.Contains(t, output, "Local Version: 1.0.0")
		assert.Contains(t, output, "Description: ECS plugin for testing")
	})

	t.Run("Display not installed plugin with stable version", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		// Create empty local manifest
		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		emptyManifest := LocalManifest{Plugins: map[string]LocalPlugin{}}
		manifestJSON, err := json.Marshal(emptyManifest)
		assert.NoError(t, err)
		os.WriteFile(manifestPath, manifestJSON, 0644)

		// Create a mock HTTP server for the plugin index
		index := &Index{
			Plugins: []PluginInfo{
				{
					Name:        "aliyun-cli-fc",
					Description: "FC plugin",
					Versions: map[string]VersionInfo{
						"1.0.0": {},
					},
				},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(index)
		}))
		defer server.Close()

		mgr := &Manager{rootDir: filepath.Join(testHome, ".aliyun"), indexURL: server.URL}

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err = displaySearchResult(ctx, mgr, "fc", "aliyun-cli-fc")
		assert.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "Command: fc")
		assert.Contains(t, output, "Plugin: aliyun-cli-fc")
		assert.Contains(t, output, "Status: Not installed")
		assert.Contains(t, output, "Latest Version: 1.0.0")
		assert.Contains(t, output, "Description: FC plugin")
		assert.Contains(t, output, "To install: aliyun plugin install --names aliyun-cli-fc")
		assert.NotContains(t, output, "--enable-pre") // Should not suggest --enable-pre for stable version
	})

	t.Run("Display not installed plugin with pre-release version", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		// Create empty local manifest
		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		emptyManifest := LocalManifest{Plugins: map[string]LocalPlugin{}}
		manifestJSON, err := json.Marshal(emptyManifest)
		assert.NoError(t, err)
		os.WriteFile(manifestPath, manifestJSON, 0644)

		// Create a mock HTTP server for the plugin index with pre-release version
		index := &Index{
			Plugins: []PluginInfo{
				{
					Name:        "aliyun-cli-preview",
					Description: "Preview plugin",
					Versions: map[string]VersionInfo{
						"2.0.0-beta": {},
					},
				},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(index)
		}))
		defer server.Close()

		mgr := &Manager{rootDir: filepath.Join(testHome, ".aliyun"), indexURL: server.URL}

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err = displaySearchResult(ctx, mgr, "preview", "aliyun-cli-preview")
		assert.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "Command: preview")
		assert.Contains(t, output, "Plugin: aliyun-cli-preview")
		assert.Contains(t, output, "Status: Not installed")
		assert.Contains(t, output, "Latest Version: 2.0.0-beta")
		assert.Contains(t, output, "Note: This is a pre-release version")
		assert.Contains(t, output, "Description: Preview plugin")
		assert.Contains(t, output, "To install: aliyun plugin install --names aliyun-cli-preview --enable-pre")
	})

	t.Run("Plugin not in index", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		// Create empty local manifest
		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		emptyManifest := LocalManifest{Plugins: map[string]LocalPlugin{}}
		manifestJSON, err := json.Marshal(emptyManifest)
		assert.NoError(t, err)
		os.WriteFile(manifestPath, manifestJSON, 0644)

		// Create a mock HTTP server with empty index
		index := &Index{
			Plugins: []PluginInfo{},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(index)
		}))
		defer server.Close()

		mgr := &Manager{rootDir: filepath.Join(testHome, ".aliyun"), indexURL: server.URL}

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err = displaySearchResult(ctx, mgr, "nonexistent", "aliyun-cli-nonexistent")
		assert.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "Command: nonexistent")
		assert.Contains(t, output, "Plugin: aliyun-cli-nonexistent")
		assert.Contains(t, output, "Status: Not installed")
		// Should not show version or install instructions since plugin not in index
		assert.NotContains(t, output, "Latest Version:")
		assert.NotContains(t, output, "To install:")
	})

	t.Run("GetIndex error - still displays basic info", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		// Create empty local manifest
		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		emptyManifest := LocalManifest{Plugins: map[string]LocalPlugin{}}
		manifestJSON, err := json.Marshal(emptyManifest)
		assert.NoError(t, err)
		os.WriteFile(manifestPath, manifestJSON, 0644)

		// Use invalid URL to trigger error
		mgr := &Manager{rootDir: filepath.Join(testHome, ".aliyun"), indexURL: "http://localhost:12345"}

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err = displaySearchResult(ctx, mgr, "test", "aliyun-cli-test")
		assert.NoError(t, err) // Function should not return error even if GetIndex fails

		output := stdout.String()
		assert.Contains(t, output, "Command: test")
		assert.Contains(t, output, "Plugin: aliyun-cli-test")
		assert.Contains(t, output, "Status: Not installed")
		// Should not show remote info since GetIndex failed
		assert.NotContains(t, output, "Latest Version:")
	})
}
