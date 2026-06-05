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
	"github.com/aliyun/aliyun-cli/v3/i18n"
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

func TestPrintPluginIndexLoadFailureNote_NoError(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)

	c.printPluginIndexLoadFailureNote(ctx)

	assert.Empty(t, stderr.String())
}

func TestPrintPluginIndexLoadFailureNote_WithError_English(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	prevLang := i18n.GetLanguage()
	t.Cleanup(func() { i18n.SetLanguage(prevLang) })
	i18n.SetLanguage("en")

	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)
	c.pluginIndexErr = fmt.Errorf("remote catalog unavailable")

	c.printPluginIndexLoadFailureNote(ctx)

	out := stderr.String()
	assert.Contains(t, out, "Note: Could not load the remote plugin catalog")
	assert.Contains(t, out, "remote catalog unavailable")
}

func TestPrintPluginIndexLoadFailureNote_WithError_Chinese(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	prevLang := i18n.GetLanguage()
	t.Cleanup(func() { i18n.SetLanguage(prevLang) })
	i18n.SetLanguage("zh")

	c, stdout, stderr := newTestCommando()
	ctx := newTestContext(stdout, stderr)
	c.pluginIndexErr = fmt.Errorf("网络错误")

	c.printPluginIndexLoadFailureNote(ctx)

	out := stderr.String()
	assert.Contains(t, out, "提示：未能加载远程插件目录")
	assert.Contains(t, out, "网络错误")
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

// ─────────────────────────────────────────────────────────────────────────────
//  tryDelegatePluginHelp (commando_help.go) — gating + tier-by-tier coverage
// ─────────────────────────────────────────────────────────────────────────────

func Test_tryDelegatePluginHelp_RefusesHTTPMethod(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}

	profile := config.Profile{Language: "en", Mode: "AK", AccessKeyId: "x", AccessKeySecret: "y", RegionId: "cn-hangzhou"}
	c := NewCommando(w, profile)

	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		t.Run("HTTP method "+method, func(t *testing.T) {
			delegated, err := c.tryDelegatePluginHelp(ctx, []string{"ecs", method, "/path"})
			assert.False(t, delegated, "RESTful shape must not be delegated to plugin")
			assert.NoError(t, err)
		})
	}
}

// Test_tryDelegatePluginHelp_PluginPath covers every tier reached after the
// HTTP-method gate passes:
//
//	tier-0: plugin installed → ExecutePlugin (success / vanished-binary / exit-error)
//	tier-1: not installed + remote index match → install guidance
//	tier-2: not installed + builtin product → InvalidUnifiedApiError or fall-through
//	tier-3: not installed + neither plugin nor product → InvalidProductOrPluginError
//
// `helpDelegateIsInstalled` / `helpDelegateExecute` are package-level test
// seams declared in commando_help.go next to tryDelegatePluginHelp;
// production code outside the helper still calls plugin.* directly.
func Test_tryDelegatePluginHelp_PluginPath(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}

	profile := config.Profile{Language: "en", Mode: "AK", AccessKeyId: "x", AccessKeySecret: "y", RegionId: "cn-hangzhou"}
	c := NewCommando(w, profile)

	origIsInstalled := helpDelegateIsInstalled
	origExecute := helpDelegateExecute
	t.Cleanup(func() {
		helpDelegateIsInstalled = origIsInstalled
		helpDelegateExecute = origExecute
	})

	t.Run("IsInstalled error → fall through", func(t *testing.T) {
		helpDelegateIsInstalled = func(string) (bool, string, error) {
			return false, "", fmt.Errorf("manifest corrupted")
		}
		helpDelegateExecute = func(string, []string, *cli.Context) (bool, error) {
			t.Fatal("ExecutePlugin must not be called when IsInstalled errors")
			return false, nil
		}
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"hologram", "config", "set"})
		assert.False(t, delegated)
		assert.NoError(t, err, "inspection errors must never bubble up at help time")
	})

	t.Run("not installed + no remote index → tier-3 InvalidProductOrPluginError", func(t *testing.T) {
		// pluginIndex stays nil (default), args[0] not a built-in product.
		// Tier-3 diagnostic kicks in: report it as an unknown product /
		// plugin rather than falling through to "too many arguments",
		// which would point at the wrong problem.
		helpDelegateIsInstalled = func(string) (bool, string, error) { return false, "", nil }
		helpDelegateExecute = func(string, []string, *cli.Context) (bool, error) {
			t.Fatal("ExecutePlugin must not be called when plugin not installed")
			return false, nil
		}
		c.pluginIndex = nil
		c.pluginIndexErr = nil
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"zzzunknown", "foo", "bar"})
		assert.True(t, delegated, "tier-3 must short-circuit the legacy 'too many arguments'")
		var ipErr *InvalidProductOrPluginError
		assert.ErrorAs(t, err, &ipErr)
		assert.Equal(t, "zzzunknown", ipErr.Code)
	})

	t.Run("not installed + remote index has match → tier-1 install guidance", func(t *testing.T) {
		// User typed `aliyun hologram dt list --help` but hologram plugin
		// was never installed. The remote catalog knows it → we MUST tell
		// them how to install instead of dumping "too many arguments: N".
		// This mirrors printProductUsage's behaviour for `aliyun hologram --help`.
		helpDelegateIsInstalled = func(string) (bool, string, error) { return false, "", nil }
		helpDelegateExecute = func(string, []string, *cli.Context) (bool, error) {
			t.Fatal("ExecutePlugin must not be called when plugin not installed")
			return false, nil
		}
		c.pluginIndex = &plugin.Index{
			Plugins: []plugin.PluginInfo{
				{Name: "aliyun-cli-hologram", ProductCode: "hologram"},
				{Name: "aliyun-cli-pds", ProductCode: "pds"},
			},
		}
		c.pluginIndexErr = nil
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"hologram", "dt", "list"})
		assert.True(t, delegated, "guided error must short-circuit the legacy 'too many arguments'")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "aliyun plugin install --names aliyun-cli-hologram",
			"the error must point the user at the exact install command")
		assert.Contains(t, err.Error(), "'hologram'",
			"the error must name the offending command")
	})

	t.Run("not installed + remote match is case-insensitive", func(t *testing.T) {
		// `aliyun help Hologram dt list` — user typed mixed case; the index
		// stores canonical lowercase. EqualFold must still find a match so
		// the install guidance is consistent with how printProductUsage matches.
		helpDelegateIsInstalled = func(string) (bool, string, error) { return false, "", nil }
		c.pluginIndex = &plugin.Index{
			Plugins: []plugin.PluginInfo{{Name: "aliyun-cli-hologram", ProductCode: "hologram"}},
		}
		c.pluginIndexErr = nil
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"Hologram", "dt", "list"})
		assert.True(t, delegated)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "aliyun-cli-hologram")
	})

	t.Run("not installed + remote index has no match → tier-3 InvalidProductOrPluginError", func(t *testing.T) {
		// Remote index loaded successfully but doesn't know args[0], and
		// args[0] isn't a built-in product either. Tier-3 surfaces it as
		// an unknown product / plugin (with typo suggestions populated
		// from the remote index for the SuggestibleError protocol).
		helpDelegateIsInstalled = func(string) (bool, string, error) { return false, "", nil }
		c.pluginIndex = &plugin.Index{
			Plugins: []plugin.PluginInfo{
				{Name: "aliyun-cli-pds", ProductCode: "pds"},
				{Name: "aliyun-cli-fc", ProductCode: "fc"},
			},
		}
		c.pluginIndexErr = nil
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"zzzunknownproduct", "foo", "bar"})
		assert.True(t, delegated)
		var ipErr *InvalidProductOrPluginError
		assert.ErrorAs(t, err, &ipErr)
		assert.Equal(t, "zzzunknownproduct", ipErr.Code)
	})

	t.Run("tier-2: builtin product + unknown API → InvalidUnifiedApiError with suggestions", func(t *testing.T) {
		// args[0]=ecs is a real OpenAPI built-in product, args[1] is a
		// typo of a real API. Surface the API-name problem (with typo
		// suggestions from product.ApiNames) instead of the misleading
		// "too many arguments", which would imply the wrong fix.
		c.library.builtinRepo = meta.LoadRepository()
		helpDelegateIsInstalled = func(string) (bool, string, error) { return false, "", nil }
		c.pluginIndex = nil // bypass tier-1
		c.pluginIndexErr = nil
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"ecs", "DescribeXyzTypo", "extra"})
		assert.True(t, delegated, "tier-2 must short-circuit so user sees the actionable API error")
		var apiErr *InvalidUnifiedApiError
		assert.ErrorAs(t, err, &apiErr)
		assert.Equal(t, "DescribeXyzTypo", apiErr.Name)
		assert.Contains(t, apiErr.Error(), "ecs",
			"error message must reference the product so the user knows where to look")
	})

	t.Run("tier-2: builtin product + valid API but extra args → fall through to legacy", func(t *testing.T) {
		// args[0]=ecs + a valid API (DescribeRegions is universally
		// present) + extra junk. This IS a structural arg-count problem
		// (API help is two-levels deep), so we deliberately fall back to
		// the legacy "too many arguments" wording rather than fabricating
		// a more elaborate error.
		c.library.builtinRepo = meta.LoadRepository()
		helpDelegateIsInstalled = func(string) (bool, string, error) { return false, "", nil }
		c.pluginIndex = nil
		c.pluginIndexErr = nil
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"ecs", "DescribeRegions", "extra"})
		assert.False(t, delegated, "valid product+API must fall through; len>2 here is genuinely structural")
		assert.NoError(t, err)
	})

	t.Run("installed + execute succeeds → delegated", func(t *testing.T) {
		var capturedName string
		var capturedArgs []string
		helpDelegateIsInstalled = func(name string) (bool, string, error) {
			return true, "aliyun-cli-" + name, nil
		}
		helpDelegateExecute = func(name string, args []string, _ *cli.Context) (bool, error) {
			capturedName = name
			capturedArgs = args
			return true, nil
		}
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"hologram", "config", "set"})
		assert.True(t, delegated, "must report delegation so caller skips the legacy error")
		assert.NoError(t, err)
		assert.Equal(t, "hologram", capturedName)
		// getPluginArgsForHelp rebuilds argv from os.Args; the test binary's
		// argv doesn't contain "hologram", so the helper returns its
		// no-match fallback `[productCode, "--help"]`. The shape is what
		// matters: the plugin name leads and --help is present.
		assert.Equal(t, []string{"hologram", "--help"}, capturedArgs)
	})

	t.Run("installed + execute returns ok=false (binary vanished) → fall through", func(t *testing.T) {
		helpDelegateIsInstalled = func(string) (bool, string, error) {
			return true, "aliyun-cli-hologram", nil
		}
		helpDelegateExecute = func(string, []string, *cli.Context) (bool, error) {
			return false, nil // simulate manifest-says-installed-but-binary-missing
		}
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"hologram", "config", "set"})
		assert.False(t, delegated, "ok=false must surface as fall-through, not as delegated")
		assert.NoError(t, err)
	})

	t.Run("installed + plugin exits with error → delegated, error propagates", func(t *testing.T) {
		pluginErr := fmt.Errorf("plugin exited with status 2")
		helpDelegateIsInstalled = func(string) (bool, string, error) { return true, "x", nil }
		helpDelegateExecute = func(string, []string, *cli.Context) (bool, error) {
			return true, pluginErr
		}
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"hologram", "config", "set"})
		assert.True(t, delegated)
		assert.Equal(t, pluginErr, err, "plugin's exit error must reach the user verbatim")
	})
}
