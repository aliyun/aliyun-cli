package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

func newTestCommando() (*Commando, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	profile := config.Profile{
		Language: "en",
		RegionId: "cn-hangzhou",
	}
	c := NewCommando(stdout, profile)
	return c, stdout, stderr
}

func newTestContext(w, stderr *bytes.Buffer) *cli.Context {
	return cli.NewCommandContext(w, stderr)
}

func TestPrintProductUsage_BuiltinProduct(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)
	c.library.builtinRepo = getRepository()

	err := c.printProductUsage(ctx, "ecs")
	assert.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "ecs")
	assert.Contains(t, output, "Available Api List")
}

func TestPrintProductUsage_UnknownProduct_NoPlugin(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)
	c.library.builtinRepo = getRepository()
	c.pluginIndex = &plugin.Index{}

	err := c.printProductUsage(ctx, "nonexistent")
	assert.Error(t, err)
	assert.IsType(t, &InvalidProductOrPluginError{}, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestPrintProductUsage_PluginAvailableNotInstalled(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)
	c.library.builtinRepo = getRepository()
	c.pluginIndex = &plugin.Index{
		Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-ecs", ProductCode: "ecs"},
		},
	}
	c.localManifest = &plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{},
	}

	err := c.printProductUsage(ctx, "ecs")
	assert.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "[Suggestion]")
	assert.Contains(t, output, "aliyun-cli-ecs")
	assert.Contains(t, output, "Available Api List")
}

func TestPrintProductUsage_NonBuiltinProduct_PluginNotInstalled(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)

	repo, _ := meta.MockLoadRepository([]meta.Product{})
	c.library.builtinRepo = repo
	c.pluginIndex = &plugin.Index{
		Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-fc", ProductCode: "fc"},
		},
	}
	c.localManifest = &plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{},
	}

	err := c.printProductUsage(ctx, "fc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "aliyun plugin install --names aliyun-cli-fc")
}

func TestPrintProductUsage_NonBuiltinProduct_NoPlugin(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)

	repo, _ := meta.MockLoadRepository([]meta.Product{})
	c.library.builtinRepo = repo
	c.pluginIndex = &plugin.Index{}

	err := c.printProductUsage(ctx, "nonexistent")
	assert.Error(t, err)
	assert.IsType(t, &InvalidProductOrPluginError{}, err)
}

func TestPrintProductUsage_RpcProduct(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)

	repo, _ := meta.MockLoadRepository([]meta.Product{
		{Code: "Ecs", Version: "2014-05-26", ApiStyle: "rpc", ApiNames: []string{"DescribeInstances"}},
	})
	c.library.builtinRepo = repo

	err := c.printProductUsage(ctx, "Ecs")
	assert.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "aliyun ecs <ApiName>")
}

func TestPrintProductUsage_RestfulProduct(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)

	repo, _ := meta.MockLoadRepository([]meta.Product{
		{Code: "CS", Version: "2015-12-15", ApiStyle: "restful", ApiNames: []string{}},
	})
	c.library.builtinRepo = repo

	err := c.printProductUsage(ctx, "CS")
	assert.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "[GET|PUT|POST|DELETE]")
}

func TestPrintApiUsage_BuiltinApi(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)
	c.library.builtinRepo = getRepository()

	err := c.printApiUsage(ctx, "ecs", "DescribeRegions")
	assert.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Product:")
	assert.Contains(t, output, "Parameters:")
}

func TestPrintApiUsage_UnknownApi_NoPlugin(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)
	c.library.builtinRepo = getRepository()

	err := c.printApiUsage(ctx, "ecs", "NonExistentApi")
	assert.Error(t, err)
	assert.IsType(t, &InvalidApiError{}, err)
	assert.Contains(t, err.Error(), "NonExistentApi")
}

func TestPrintApiUsage_UnknownProduct_NoPlugin(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)
	c.library.builtinRepo = getRepository()
	c.pluginIndex = &plugin.Index{}

	err := c.printApiUsage(ctx, "nonexistent", "SomeApi")
	assert.Error(t, err)
	assert.IsType(t, &InvalidProductOrPluginError{}, err)
}

func TestPrintApiUsage_UnknownApi_PluginAvailableNotInstalled_Lowercase(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)
	c.library.builtinRepo = getRepository()
	c.pluginIndex = &plugin.Index{
		Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-ecs", ProductCode: "ecs"},
		},
	}
	c.localManifest = &plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{},
	}

	// Lowercase API name triggers shouldTryPlugin=true, uninstalled plugin -> install hint
	err := c.printApiUsage(ctx, "ecs", "describeinstances")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "aliyun-cli-ecs")
	assert.Contains(t, err.Error(), "aliyun plugin install --names")
}

func TestPrintApiUsage_UnknownApi_PluginHint_NonLowercase(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)
	c.library.builtinRepo = getRepository()
	c.pluginIndex = &plugin.Index{
		Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-ecs", ProductCode: "ecs"},
		},
	}
	c.localManifest = &plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{},
	}

	// Non-lowercase (e.g. "BadApiName") -> prints suggestion, then returns InvalidApiError
	err := c.printApiUsage(ctx, "ecs", "BadApiName")
	assert.Error(t, err)
	assert.IsType(t, &InvalidApiError{}, err)

	output := stdout.String()
	assert.Contains(t, output, "[Suggestion]")
	assert.Contains(t, output, "aliyun-cli-ecs")
}

func TestPrintApiUsage_NonBuiltinProduct_PluginNotInstalled(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)

	repo, _ := meta.MockLoadRepository([]meta.Product{})
	c.library.builtinRepo = repo
	c.pluginIndex = &plugin.Index{
		Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-fc", ProductCode: "fc"},
		},
	}
	c.localManifest = &plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{},
	}

	err := c.printApiUsage(ctx, "fc", "somecmd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "aliyun plugin install --names aliyun-cli-fc")
}

func TestGetPluginArgsForHelp(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	t.Run("NoMatch - fallback", func(t *testing.T) {
		os.Args = []string{"aliyun", "other"}
		args := getPluginArgsForHelp("nonexistent-product-xyz")
		assert.Equal(t, []string{"nonexistent-product-xyz", "--help"}, args)
	})

	t.Run("Match - appends help", func(t *testing.T) {
		os.Args = []string{"aliyun", "ecs"}
		args := getPluginArgsForHelp("ecs")
		assert.Equal(t, []string{"ecs", "--help"}, args)
	})

	t.Run("Match case insensitive", func(t *testing.T) {
		os.Args = []string{"aliyun", "ECS", "--region", "cn-hangzhou"}
		args := getPluginArgsForHelp("ecs")
		assert.Equal(t, []string{"ECS", "--region", "cn-hangzhou", "--help"}, args)
	})

	t.Run("Match - already has --help", func(t *testing.T) {
		os.Args = []string{"aliyun", "ecs", "--help"}
		args := getPluginArgsForHelp("ecs")
		assert.Equal(t, []string{"ecs", "--help"}, args)
	})

	t.Run("Match - already has -h", func(t *testing.T) {
		os.Args = []string{"aliyun", "ecs", "-h"}
		args := getPluginArgsForHelp("ecs")
		assert.Equal(t, []string{"ecs", "-h"}, args)
	})

	t.Run("Match with extra flags", func(t *testing.T) {
		os.Args = []string{"aliyun", "ecs", "--api-version", "2014-05-26"}
		args := getPluginArgsForHelp("ecs")
		assert.Equal(t, []string{"ecs", "--api-version", "2014-05-26", "--help"}, args)
	})
}

func setupInstalledPlugin(t *testing.T, pluginName string) {
	t.Helper()
	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	t.Cleanup(cleanup)

	pluginsDir := filepath.Join(testHome, ".aliyun", "plugins")
	os.MkdirAll(pluginsDir, 0755)

	manifest := plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{
			pluginName: {
				Name:    pluginName,
				Version: "1.0.0",
				Path:    filepath.Join(pluginsDir, pluginName),
			},
		},
	}
	data, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), data, 0644)
}

func TestPrintProductUsage_NonBuiltinProduct_PluginInstalled(t *testing.T) {
	setupInstalledPlugin(t, "aliyun-cli-fc")

	c, w, stderr := newTestCommando()
	ctx := newTestContext(w, stderr)

	repo, _ := meta.MockLoadRepository([]meta.Product{})
	c.library.builtinRepo = repo
	c.pluginIndex = &plugin.Index{
		Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-fc", ProductCode: "fc"},
		},
	}
	c.localManifest = &plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{
			"aliyun-cli-fc": {Name: "aliyun-cli-fc", Version: "1.0.0"},
		},
	}

	err := c.printProductUsage(ctx, "fc")
	assert.NoError(t, err)

	output := w.String()
	assert.Contains(t, output, "Product 'fc' is provided by plugin 'aliyun-cli-fc'")
}

func TestPrintProductUsage_BuiltinProduct_PluginInstalled(t *testing.T) {
	setupInstalledPlugin(t, "aliyun-cli-ecs")
	t.Setenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP", "")

	c, w, stderr := newTestCommando()
	ctx := newTestContext(w, stderr)
	c.library.builtinRepo = getRepository()
	c.pluginIndex = &plugin.Index{
		Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-ecs", ProductCode: "ecs"},
		},
	}
	c.localManifest = &plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{
			"aliyun-cli-ecs": {Name: "aliyun-cli-ecs", Version: "1.0.0"},
		},
	}

	err := c.printProductUsage(ctx, "ecs")
	assert.NoError(t, err)

	output := w.String()
	assert.Contains(t, output, "Note: The help information for product 'ecs' is provided by the installed plugin 'aliyun-cli-ecs'")
	assert.Contains(t, output, "ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP=true")
}

func TestPrintApiUsage_NonBuiltinProduct_PluginInstalled(t *testing.T) {
	setupInstalledPlugin(t, "aliyun-cli-fc")

	c, w, stderr := newTestCommando()
	ctx := newTestContext(w, stderr)

	repo, _ := meta.MockLoadRepository([]meta.Product{})
	c.library.builtinRepo = repo
	c.pluginIndex = &plugin.Index{
		Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-fc", ProductCode: "fc"},
		},
	}
	c.localManifest = &plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{
			"aliyun-cli-fc": {Name: "aliyun-cli-fc", Version: "1.0.0"},
		},
	}

	err := c.printApiUsage(ctx, "fc", "deploy")
	assert.NoError(t, err)

	output := w.String()
	assert.Contains(t, output, "Command 'fc deploy' is provided by plugin 'aliyun-cli-fc'")
}

func setupLocalPluginShellBinary(t *testing.T, pluginName string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("plugin fixture uses a Unix shell shebang")
	}
	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	t.Cleanup(cleanup)

	pluginsDir := filepath.Join(testHome, ".aliyun", "plugins")
	pluginDir := filepath.Join(pluginsDir, pluginName)
	assert.NoError(t, os.MkdirAll(pluginDir, 0755))

	binPath := filepath.Join(pluginDir, pluginName)
	script := "#!/bin/sh\necho HELP_FROM_LOCAL_PLUGIN\nexit 0\n"
	assert.NoError(t, os.WriteFile(binPath, []byte(script), 0755))

	manifest := plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{
			pluginName: {
				Name:    pluginName,
				Version: "1.0.0",
				Path:    pluginDir,
			},
		},
	}
	data, err := json.Marshal(manifest)
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), data, 0644))
}

func TestPrintProductUsage_LocalPluginNotInRemoteIndex(t *testing.T) {
	setupLocalPluginShellBinary(t, "aliyun-cli-localonly")

	c, w, stderr := newTestCommando()
	ctx := newTestContext(w, stderr)

	repo, _ := meta.MockLoadRepository([]meta.Product{})
	c.library.builtinRepo = repo
	c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{}}

	mgr, err := plugin.NewManager()
	assert.NoError(t, err)
	lm, err := mgr.GetLocalManifest()
	assert.NoError(t, err)
	c.localManifest = lm

	err = c.printProductUsage(ctx, "localonly")
	assert.NoError(t, err)
	out := w.String()
	assert.Contains(t, out, "Product 'localonly' is provided by plugin 'aliyun-cli-localonly'")
	assert.Contains(t, out, "HELP_FROM_LOCAL_PLUGIN")
}

func TestPrintApiUsage_LocalPluginNotInRemoteIndex(t *testing.T) {
	setupLocalPluginShellBinary(t, "aliyun-cli-localonly2")

	c, w, stderr := newTestCommando()
	ctx := newTestContext(w, stderr)

	repo, _ := meta.MockLoadRepository([]meta.Product{})
	c.library.builtinRepo = repo
	c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{}}

	mgr, err := plugin.NewManager()
	assert.NoError(t, err)
	lm, err := mgr.GetLocalManifest()
	assert.NoError(t, err)
	c.localManifest = lm

	err = c.printApiUsage(ctx, "localonly2", "SomeApi")
	assert.NoError(t, err)
	assert.Contains(t, w.String(), "Command 'localonly2 SomeApi' is provided by plugin 'aliyun-cli-localonly2'")
}

func TestPrintProducts_PluginIndexLoadHint(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)
	c.library.builtinRepo = getRepository()
	c.pluginLoaded = true
	c.pluginIndexErr = fmt.Errorf("failed to fetch plugin index")
	c.pluginIndex = nil

	c.printProducts(ctx)

	assert.Contains(t, stderr.String(), "plugin catalog")
}

func TestPrintHelpContextHints_AiModeConfigEnabled(t *testing.T) {
	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	t.Cleanup(cleanup)

	confDir := filepath.Join(testHome, ".aliyun")
	assert.NoError(t, os.MkdirAll(confDir, 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(confDir, "ai-mode.json"), []byte(`{"enabled":true}`), 0600))

	c, w, stderr := newTestCommando()
	cmd := &cli.Command{}
	AddFlags(cmd.Flags())
	ctx := cli.NewCommandContext(w, stderr)
	ctx.EnterCommand(cmd)

	c.pluginLoaded = true
	c.printHelpContextHints(ctx)

	out := stderr.String()
	assert.Contains(t, out, "configure ai-mode")
	assert.Contains(t, out, "disable")
}
