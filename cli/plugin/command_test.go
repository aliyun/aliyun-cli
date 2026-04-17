package plugin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	assert.NotNil(t, cmd.GetSubCommand("show"), "Should have show subcommand")
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

func TestNewListRemoteCommand(t *testing.T) {
	cmd := newListRemoteCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "list-remote", cmd.Name)
	assert.NotEmpty(t, cmd.Short)
	assert.NotNil(t, cmd.Flags().Get("source-base"))
}

func TestNewListRemoteCommand_Run(t *testing.T) {
	cmd := newListRemoteCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
	os.MkdirAll(filepath.Dir(manifestPath), 0755)
	manifest := LocalManifest{Plugins: map[string]LocalPlugin{}}
	manifestJSON, _ := json.Marshal(manifest)
	os.WriteFile(manifestPath, manifestJSON, 0644)

	index := &Index{
		Plugins: []PluginInfo{
			{
				Name:        "aliyun-cli-fc",
				Description: "FC plugin",
				Versions:    map[string]VersionInfo{"1.0.0": {}},
			},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	}))
	defer server.Close()

	// Override the index URL via cache: write the index to the cache file
	cacheFile := filepath.Join(testHome, ".aliyun", "plugins", indexCacheFile)
	indexJSON, _ := json.Marshal(index)
	os.WriteFile(cacheFile, indexJSON, 0644)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	err := cmd.Run(ctx, []string{})
	assert.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "aliyun-cli-fc")
	assert.Contains(t, output, "Not installed")
}

func TestDisplayRemotePlugins_Empty(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)

	index := &Index{}
	manifest := &LocalManifest{Plugins: map[string]LocalPlugin{}}

	err := displayRemotePlugins(ctx, index, manifest)
	assert.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Total plugins available: 0")
	assert.Contains(t, output, "No plugins available")
}

func TestDisplayRemotePlugins_MixedStatus(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)

	index := &Index{
		Plugins: []PluginInfo{
			{
				Name:        "aliyun-cli-fc",
				Description: "FC plugin",
				Versions:    map[string]VersionInfo{"1.2.0": {}},
			},
			{
				Name:        "aliyun-cli-oss",
				Description: "OSS plugin",
				Versions:    map[string]VersionInfo{"2.0.0": {}},
			},
		},
	}
	manifest := &LocalManifest{
		Plugins: map[string]LocalPlugin{
			"aliyun-cli-fc": {
				Name:    "aliyun-cli-fc",
				Version: "1.0.0",
			},
		},
	}

	err := displayRemotePlugins(ctx, index, manifest)
	assert.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Total plugins available: 2")
	assert.Contains(t, output, "aliyun-cli-fc")
	assert.Contains(t, output, "aliyun-cli-oss")
	assert.Contains(t, output, "Installed")
	assert.Contains(t, output, "Not installed")
	assert.Contains(t, output, "1.0.0")
}

func TestDisplayRemotePlugins_SortOrder(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)

	index := &Index{
		Plugins: []PluginInfo{
			{Name: "aliyun-cli-zzz", Description: "Z plugin", Versions: map[string]VersionInfo{"1.0.0": {}}},
			{Name: "aliyun-cli-aaa", Description: "A plugin", Versions: map[string]VersionInfo{"1.0.0": {}}},
			{Name: "aliyun-cli-bbb", Description: "B plugin", Versions: map[string]VersionInfo{"1.0.0": {}}},
		},
	}
	manifest := &LocalManifest{
		Plugins: map[string]LocalPlugin{
			"aliyun-cli-zzz": {Name: "aliyun-cli-zzz", Version: "1.0.0"},
		},
	}

	err := displayRemotePlugins(ctx, index, manifest)
	assert.NoError(t, err)

	output := stdout.String()
	// Installed plugins should come first, then sorted alphabetically
	zzzIdx := strings.Index(output, "aliyun-cli-zzz")
	aaaIdx := strings.Index(output, "aliyun-cli-aaa")
	bbbIdx := strings.Index(output, "aliyun-cli-bbb")
	assert.Less(t, zzzIdx, aaaIdx, "installed plugin zzz should appear before uninstalled aaa")
	assert.Less(t, aaaIdx, bbbIdx, "uninstalled plugins should be sorted alphabetically")
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

	packageFlag := flags.Get("package")
	assert.NotNil(t, packageFlag)
	assert.False(t, packageFlag.Required)

	sourceBaseFlag := flags.Get("source-base")
	assert.NotNil(t, sourceBaseFlag)
	assert.False(t, sourceBaseFlag.Required)
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
	namesFlag.SetValues([]string{"nonexistent-plugin-xyz-123"})

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-plugin-xyz-123 not found")
}

func TestNewInstallCommand_Run_WithNamesAndVersionFlags(t *testing.T) {
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

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

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
	assert.Contains(t, err.Error(), "either --names or --package is required")
}

func TestNewInstallCommand_Run_WithNamesAndEnablePreFlags(t *testing.T) {
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
	namesFlag.SetValues([]string{"nonexistent-plugin-xyz-123"})

	enablePreFlag := ctx.Flags().Get("enable-pre")
	assert.NotNil(t, enablePreFlag)
	enablePreFlag.SetAssigned(true)

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-plugin-xyz-123 not found")
}

func TestNewInstallCommand_Run_FlagValueAssignment(t *testing.T) {
	cmd := newInstallCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

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

	t.Run("Enable pre flag value assignment", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)

		enablePreFlag := ctx.Flags().Get("enable-pre")
		enablePreFlag.SetAssigned(true)

		enablePreFlag2 := ctx.Flags().Get("enable-pre")
		assert.NotNil(t, enablePreFlag2) // enable-pre flag should be retrievable
		assert.True(t, enablePreFlag2.IsAssigned(), "enable-pre flag should be assigned")
	})

}

func TestNewInstallAllCommand(t *testing.T) {
	cmd := newInstallAllCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "install-all", cmd.Name)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Usage)
	assert.NotNil(t, cmd.Flags().Get("source-base"))
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

func TestNewShowCommand(t *testing.T) {
	cmd := newShowCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "show", cmd.Name)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Usage)

	flags := cmd.Flags()
	nameFlag := flags.Get("name")
	assert.NotNil(t, nameFlag)
	assert.False(t, nameFlag.Required)
}

func TestNewShowCommand_Run_MissingName(t *testing.T) {
	cmd := newShowCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--name")
}

func TestNewShowCommand_Run_NotInstalled(t *testing.T) {
	cmd := newShowCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
	assert.NoError(t, os.MkdirAll(filepath.Dir(manifestPath), 0755))
	manifest := LocalManifest{Plugins: map[string]LocalPlugin{}}
	manifestJSON, err := json.Marshal(manifest)
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(manifestPath, manifestJSON, 0644))

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	nameFlag := ctx.Flags().Get("name")
	assert.NotNil(t, nameFlag)
	nameFlag.SetAssigned(true)
	nameFlag.SetValue("missing-plugin")

	err = cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestNewShowCommand_Run_Success(t *testing.T) {
	cmd := newShowCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
	assert.NoError(t, os.MkdirAll(filepath.Dir(manifestPath), 0755))
	pluginPath := filepath.Join(testHome, ".aliyun", "plugins", "aliyun-cli-demo")
	assert.NoError(t, os.MkdirAll(pluginPath, 0755))

	pkgManifest := map[string]interface{}{
		"name":        "aliyun-cli-demo",
		"version":     "2.0.0",
		"productCode": "demo-product",
		"apiVersions": map[string]interface{}{
			"default": "2017-06-13",
			"supported": []string{
				"2019-08-16",
				"2017-06-13",
			},
			"versionInfo": map[string]interface{}{
				"2017-06-13": map[string]interface{}{
					"deprecated": false, "recommended": true, "description": "stable line",
				},
				"2019-08-16": map[string]interface{}{
					"deprecated": false, "recommended": false, "description": "newer line",
				},
			},
		},
	}
	pkgJSON, err := json.Marshal(pkgManifest)
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(filepath.Join(pluginPath, "manifest.json"), pkgJSON, 0644))

	manifest := LocalManifest{
		Plugins: map[string]LocalPlugin{
			"aliyun-cli-demo": {
				Name:             "aliyun-cli-demo",
				Version:          "2.0.0",
				Path:             pluginPath,
				ProductCode:      "demo-product",
				Command:          "demo",
				ShortDescription: "short",
				Description:      "full description",
				Inner:            true,
			},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(manifestPath, manifestJSON, 0644))

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	nameFlag := ctx.Flags().Get("name")
	assert.NotNil(t, nameFlag)
	nameFlag.SetAssigned(true)
	nameFlag.SetValue("aliyun-cli-demo")

	err = cmd.Run(ctx, []string{})
	assert.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Name:\taliyun-cli-demo")
	assert.Contains(t, out, "Version:\t2.0.0")
	assert.Contains(t, out, "Product code:\tdemo-product\n")
	assert.Contains(t, out, "Short description:\tshort")
	assert.Contains(t, out, "Description:\tfull description")
	assert.Contains(t, out, "API default:\t2017-06-13\n")
	assert.Contains(t, out, "API supported:\t2019-08-16, 2017-06-13\n")
	assert.Contains(t, out, "Inner:\ttrue")
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

	assert.NotNil(t, flags.Get("source-base"))
}

func TestNewManagerWithOptionalSourceBase(t *testing.T) {
	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	t.Run("no override when source-base unset", func(t *testing.T) {
		cmd := newUpdateCommand()
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)
		mgr, err := newManagerWithOptionalSourceBase(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, mgr)
	})

	t.Run("valid override applies trim and trailing slash", func(t *testing.T) {
		cmd := newUpdateCommand()
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)
		f := ctx.Flags().Get("source-base")
		assert.NotNil(t, f)
		f.SetAssigned(true)
		f.SetValue("  https://mirror.example/plugins/  ")
		mgr, err := newManagerWithOptionalSourceBase(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, mgr)
		assert.Equal(t, "https://mirror.example/plugins", mgr.sourceBase)
	})

	t.Run("invalid scheme returns ApplySourceBaseOverride error", func(t *testing.T) {
		cmd := newUpdateCommand()
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)
		f := ctx.Flags().Get("source-base")
		f.SetAssigned(true)
		f.SetValue("ftp://mirror.example/plugins")
		mgr, err := newManagerWithOptionalSourceBase(ctx)
		assert.Error(t, err)
		assert.Nil(t, mgr)
		assert.Contains(t, err.Error(), "source-base must start with http:// or https://")
	})

	t.Run("whitespace only value returns error", func(t *testing.T) {
		cmd := newUpdateCommand()
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)
		f := ctx.Flags().Get("source-base")
		f.SetAssigned(true)
		f.SetValue("   \t  ")
		mgr, err := newManagerWithOptionalSourceBase(ctx)
		assert.Error(t, err)
		assert.Nil(t, mgr)
		assert.Contains(t, err.Error(), "source-base must not be empty")
	})
}

func TestNewUpdateCommand_Run_WithoutName(t *testing.T) {
	cmd := newUpdateCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	err := cmd.Run(ctx, []string{})
	assert.NoError(t, err)
}

func TestNewUpdateCommand_Run_WithName(t *testing.T) {
	cmd := newUpdateCommand()

	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

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

func TestDisplaySearchResults(t *testing.T) {
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

		results := map[string][]string{
			"aliyun-cli-ecs": {"ecs"},
		}

		// Setup mock index for test
		index := &Index{
			Plugins: []PluginInfo{
				{
					Name:        "aliyun-cli-ecs",
					Description: "ECS plugin for testing",
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
		mgr.indexURL = server.URL

		err = displaySearchResults(ctx, mgr, "ecs", results)
		assert.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "Plugin")
		assert.NotContains(t, output, "Command(s)")
		assert.Contains(t, output, "Status")
		assert.Contains(t, output, "Latest Version")
		assert.Contains(t, output, "Preview")
		assert.Contains(t, output, "Local Version")
		assert.Contains(t, output, "Description")

		assert.Contains(t, output, "aliyun-cli-ecs")
		assert.Contains(t, output, "Installed")
		assert.Contains(t, output, "1.0.0")
	})

	t.Run("Display not installed plugin", func(t *testing.T) {
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

		mgr, err := NewManager()
		assert.NoError(t, err)
		mgr.rootDir = filepath.Join(testHome, ".aliyun", "plugins")

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		results := map[string][]string{
			"aliyun-cli-fc": {"fc"},
		}

		// Setup mock index for test
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
		mgr.indexURL = server.URL

		err = displaySearchResults(ctx, mgr, "fc", results)
		assert.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "aliyun-cli-fc")
		assert.Contains(t, output, "Not installed")
		assert.Contains(t, output, "1.0.0")
		assert.Contains(t, output, "FC plugin")
	})

	t.Run("Display multiple plugins", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		// Create local manifest with one installed
		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		localManifest := LocalManifest{
			Plugins: map[string]LocalPlugin{
				"aliyun-cli-ecs": {
					Name:    "aliyun-cli-ecs",
					Version: "1.0.0",
				},
			},
		}
		manifestJSON, err := json.Marshal(localManifest)
		assert.NoError(t, err)
		os.WriteFile(manifestPath, manifestJSON, 0644)

		mgr, err := NewManager()
		assert.NoError(t, err)
		mgr.rootDir = filepath.Join(testHome, ".aliyun", "plugins")

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		results := map[string][]string{
			"aliyun-cli-ecs": {"ecs"},
			"aliyun-cli-ecr": {"ecr"},
		}

		// Setup mock index for test
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
					Name:        "aliyun-cli-ecr",
					Description: "ECR plugin",
					Versions: map[string]VersionInfo{
						"2.0.0": {},
					},
				},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(index)
		}))
		defer server.Close()
		mgr.indexURL = server.URL

		err = displaySearchResults(ctx, mgr, "ec", results)
		assert.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "aliyun-cli-ecs")
		assert.Contains(t, output, "Installed")
		// assert.NotContains(t, output, "ecr") // Cannot assert this as 'ecr' is part of 'aliyun-cli-ecr'
		assert.Contains(t, output, "aliyun-cli-ecr")
		assert.Contains(t, output, "Not installed")
		assert.Contains(t, output, "2.0.0")
	})
}

func TestRunSearch(t *testing.T) {
	t.Run("No args", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err := runSearch(ctx, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command name is required")
	})

	t.Run("Empty command name", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err := runSearch(ctx, []string{""})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command name cannot be empty")
	})

	t.Run("No match found", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		pluginsDir := filepath.Join(testHome, ".aliyun", "plugins")
		os.MkdirAll(pluginsDir, 0755)

		commandIndex := map[string]string{
			"ecs": "aliyun-cli-ecs",
		}
		commandIndexJSON, _ := json.Marshal(commandIndex)
		os.WriteFile(filepath.Join(pluginsDir, commandCacheFile), commandIndexJSON, 0644)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err := runSearch(ctx, []string{"nonexistent-xyz"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no plugin found for command prefix")
	})

	t.Run("Match found", func(t *testing.T) {
		testHome := t.TempDir()
		cleanup := setTestHomeDir(t, testHome)
		defer cleanup()

		pluginsDir := filepath.Join(testHome, ".aliyun", "plugins")
		os.MkdirAll(pluginsDir, 0755)

		// Prepare command index cache
		commandIndex := map[string]string{
			"ecs":              "aliyun-cli-ecs",
			"ecs list-regions": "aliyun-cli-ecs",
			"fc":               "aliyun-cli-fc",
		}
		commandIndexJSON, _ := json.Marshal(commandIndex)
		os.WriteFile(filepath.Join(pluginsDir, commandCacheFile), commandIndexJSON, 0644)

		// Prepare plugin index cache
		pluginIndex := &Index{
			Plugins: []PluginInfo{
				{
					Name:        "aliyun-cli-ecs",
					Description: "ECS plugin",
					Versions:    map[string]VersionInfo{"1.0.0": {}},
				},
			},
		}
		pluginIndexJSON, _ := json.Marshal(pluginIndex)
		os.WriteFile(filepath.Join(pluginsDir, indexCacheFile), pluginIndexJSON, 0644)

		// Prepare local manifest (no plugins installed)
		manifest := LocalManifest{Plugins: map[string]LocalPlugin{}}
		manifestJSON, _ := json.Marshal(manifest)
		os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), manifestJSON, 0644)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		err := runSearch(ctx, []string{"ecs"})
		assert.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "aliyun-cli-ecs")
		assert.Contains(t, output, "Not installed")
	})
}

func TestNewSearchCommand_Integration(t *testing.T) {
	cmd := newSearchCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "search", cmd.Name)
	assert.NotNil(t, cmd.Flags().Get("source-base"))

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	// No args should error
	err := cmd.Run(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command name is required")
}
