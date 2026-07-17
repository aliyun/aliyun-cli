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
//  tryDelegatePluginHelp (commando_help.go) — gating + step-by-step coverage
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

// Test_tryDelegatePluginHelp_RefusesUppercaseSubcommand exercises Gate-2:
// any uppercase character in args[1] flags the input as a legacy-mode
// (typically PascalCase OpenAPI APIName) shape, so the helper falls
// through to the historical "too many arguments" wording rather than
// inventing a plugin-flavoured diagnostic that would point at the
// wrong problem. Examples drawn from real OpenAPI invocations.
func Test_tryDelegatePluginHelp_RefusesUppercaseSubcommand(t *testing.T) {
	c, ctx := newTryDelegateHarness(t)
	// Even with a fully populated index we must NOT treat these as
	// plugin commands — the gate is purely syntactic on args[1].
	c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{
		{Name: "aliyun-cli-hologram", ProductCode: "hologram"},
		{Name: "aliyun-cli-ecs", ProductCode: "ecs"}, // hypothetical plugin homonym
	}}
	helpDelegateIsInstalled = func(string) (bool, string, error) {
		t.Fatal("Gate-2 must short-circuit before any I/O")
		return false, "", nil
	}

	cases := [][]string{
		{"ecs", "DescribeRegions", "extra"}, // canonical PascalCase API
		{"hologram", "DescribeFoo", "bar"},  // PascalCase even on a real plugin name
		{"vpc", "describeVpcs", "x"},        // camelCase — first letter lower but Vpcs uppercased
		{"fc-open", "InvokeFunction", "y"},  // hyphenated product + PascalCase API
		{"hologram", "Config", "set"},       // user accidentally TitleCase'd a plugin sub-command
	}
	for _, args := range cases {
		t.Run(args[0]+"/"+args[1], func(t *testing.T) {
			delegated, err := c.tryDelegatePluginHelp(ctx, args)
			assert.False(t, delegated, "uppercase args[1] must fall through (legacy shape)")
			assert.NoError(t, err)
		})
	}
}

// newTryDelegateHarness wires a Commando + Context with the framework
// state tryDelegatePluginHelp needs (cmd flags + i18n short text), and
// also stashes/restores the package-level test seams so each subtest
// can swap helpDelegateIsInstalled / helpDelegateExecute freely. The
// seam wiring is the whole reason these tests live at help-test scope:
// production code calls plugin.IsPluginInstalled / plugin.ExecutePlugin
// directly, which would otherwise need a real plugin binary on disk.
func newTryDelegateHarness(t *testing.T) (*Commando, *cli.Context) {
	t.Helper()
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
	return c, ctx
}

// Test_tryDelegatePluginHelp_PluginPath covers every step the helper can
// take after the gates pass (Gate-1 HTTP method, Gate-2 uppercase in
// args[1]). Layout mirrors the decision tree in commando_help.go:
//
//	Step 2 — index hit + installed:    forward (success / exit-err / vanished)
//	Step 2 — index hit + not installed: install guidance
//	Step 2 — index hit + manifest err: treated as not installed → install guidance
//	Step 2 — index hit case-insensitive
//	Step 3 — index miss + installed:   forward (dev side-load)
//	Step 4 — neither plugin nor product: InvalidProductOrPluginError + fuzzy
//	Step 4 — index fetch failed:       fetch-note + InvalidProductOrPluginError
//
// Helpers are toggled per-subtest so each path is exercised in isolation.
func Test_tryDelegatePluginHelp_PluginPath(t *testing.T) {
	t.Run("step-2 index hit + installed → forward succeeds", func(t *testing.T) {
		c, ctx := newTryDelegateHarness(t)
		c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-hologram", ProductCode: "hologram"},
		}}

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
		// getPluginArgsForHelp rebuilds argv from os.Args; the test
		// binary's argv doesn't contain "hologram", so the helper falls
		// back to `[productCode, "--help"]`. The shape is what matters:
		// the plugin name leads and --help is present.
		assert.Equal(t, []string{"hologram", "--help"}, capturedArgs)
	})

	t.Run("step-2 index hit + installed → plugin exit error propagates verbatim", func(t *testing.T) {
		c, ctx := newTryDelegateHarness(t)
		c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-hologram", ProductCode: "hologram"},
		}}
		pluginErr := fmt.Errorf("plugin exited with status 2")
		helpDelegateIsInstalled = func(string) (bool, string, error) { return true, "x", nil }
		helpDelegateExecute = func(string, []string, *cli.Context) (bool, error) {
			return true, pluginErr
		}
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"hologram", "config", "set"})
		assert.True(t, delegated)
		assert.Equal(t, pluginErr, err, "plugin's exit error must reach the user verbatim")
	})

	t.Run("step-2 index hit + binary vanished → reinstall guidance (no fall-through)", func(t *testing.T) {
		// Manifest says installed but ExecutePlugin returns ok=false
		// (binary missing). New behaviour: surface a one-line reinstall
		// hint instead of the legacy `too many arguments` fall-through,
		// which would have left the user stranded.
		c, ctx := newTryDelegateHarness(t)
		c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-hologram", ProductCode: "hologram"},
		}}
		helpDelegateIsInstalled = func(string) (bool, string, error) { return true, "aliyun-cli-hologram", nil }
		helpDelegateExecute = func(string, []string, *cli.Context) (bool, error) {
			return false, nil // simulate manifest-says-installed-but-binary-missing
		}
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"hologram", "config", "set"})
		assert.True(t, delegated, "vanished-binary case must produce a final answer, not fall through")
		assert.Error(t, err)
		// Hallmark phrases of the new copy — locked in so a future
		// refactor that drifts the wording trips a test instead of
		// silently changing user-visible diagnostics.
		assert.Contains(t, err.Error(), "looks like a plugin command",
			"reinstall guidance must share the plugin-cmd preamble with install guidance")
		assert.Contains(t, err.Error(), "registered as installed")
		assert.Contains(t, err.Error(), "binary cannot be located")
		assert.Contains(t, err.Error(), "to reinstall it and try again",
			"the reinstall variant must end with the reinstall-and-retry prompt")
		assert.Contains(t, err.Error(), "aliyun plugin install --name aliyun-cli-hologram",
			"reinstall hint must reference the canonical plugin name")
		assert.Contains(t, err.Error(), "'aliyun hologram config set'",
			"error must echo the user's full logical command, not just the plugin code")
	})

	t.Run("step-2 index hit + not installed → install guidance", func(t *testing.T) {
		c, ctx := newTryDelegateHarness(t)
		c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-hologram", ProductCode: "hologram"},
			{Name: "aliyun-cli-pds", ProductCode: "pds"},
		}}
		helpDelegateIsInstalled = func(string) (bool, string, error) { return false, "", nil }
		helpDelegateExecute = func(string, []string, *cli.Context) (bool, error) {
			t.Fatal("ExecutePlugin must not be called when plugin not installed")
			return false, nil
		}
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"hologram", "dt", "list"})
		assert.True(t, delegated, "guided error must short-circuit the legacy 'too many arguments'")
		assert.Error(t, err)
		// Hallmark phrases of the new copy — see the matching block in
		// the binary-vanished subtest for rationale.
		assert.Contains(t, err.Error(), "looks like a plugin command",
			"install guidance must lead with the plugin-cmd preamble")
		assert.Contains(t, err.Error(), "is not installed",
			"diagnose the install state explicitly")
		assert.Contains(t, err.Error(), "to install it and try again",
			"the install variant must end with the install-and-retry prompt")
		assert.Contains(t, err.Error(), "aliyun plugin install --name aliyun-cli-hologram",
			"the error must point the user at the exact install command")
		assert.Contains(t, err.Error(), "plugin 'hologram'",
			"the error must name the offending plugin")
		assert.Contains(t, err.Error(), "'aliyun hologram dt list'",
			"error must echo the user's full logical command for long chains")
	})

	t.Run("step-2 index hit + manifest unreadable → treated as not installed", func(t *testing.T) {
		// Corrupted manifest used to silently fall through to the
		// legacy "too many arguments" — masking both the real bug and
		// the install guidance. New behaviour: index says it's a known
		// plugin, so guide the user to (re)install regardless of the
		// manifest hiccup; reinstall would fix a corrupted manifest too.
		c, ctx := newTryDelegateHarness(t)
		c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-hologram", ProductCode: "hologram"},
		}}
		helpDelegateIsInstalled = func(string) (bool, string, error) {
			return false, "", fmt.Errorf("manifest corrupted")
		}
		helpDelegateExecute = func(string, []string, *cli.Context) (bool, error) {
			t.Fatal("ExecutePlugin must not be called when manifest unreadable")
			return false, nil
		}
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"hologram", "config", "set"})
		assert.True(t, delegated, "manifest errors must NOT silently fall through any more")
		assert.Error(t, err)
		// Manifest-error path must produce the SAME diagnostic as the
		// genuine "not installed" path — re-running install repairs
		// both, so users shouldn't have to distinguish them.
		assert.Contains(t, err.Error(), "is not installed")
		assert.Contains(t, err.Error(), "to install it and try again")
		assert.Contains(t, err.Error(), "aliyun plugin install --name aliyun-cli-hologram")
		assert.Contains(t, err.Error(), "'aliyun hologram config set'")
	})

	t.Run("step-2 index hit case-insensitive → canonical Name in guidance", func(t *testing.T) {
		// `aliyun help Hologram dt list` — user typed mixed case; the
		// index stores canonical lowercase. EqualFold must still find
		// a match so the install guidance uses the canonical Name from
		// the index (consistent with printProductUsage's matching).
		c, ctx := newTryDelegateHarness(t)
		c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-hologram", ProductCode: "hologram"},
		}}
		helpDelegateIsInstalled = func(string) (bool, string, error) { return false, "", nil }
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"Hologram", "dt", "list"})
		assert.True(t, delegated)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "aliyun-cli-hologram")
	})

	t.Run("step-3 index miss + installed → forward (dev side-load)", func(t *testing.T) {
		// Internal plugin distributed before it hits the public
		// catalog. Index doesn't know it but the binary is on disk.
		// We must still forward, otherwise these users would get
		// "'myplugin' is not a valid product" — a regression vs the
		// old fall-through behaviour.
		c, ctx := newTryDelegateHarness(t)
		c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{
			// Plugins exist but none matches args[0]
			{Name: "aliyun-cli-pds", ProductCode: "pds"},
		}}
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
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"sideloaded", "sub", "cmd"})
		assert.True(t, delegated, "side-loaded plugins must still get to render their own help")
		assert.NoError(t, err)
		assert.Equal(t, "sideloaded", capturedName)
		assert.Equal(t, []string{"sideloaded", "--help"}, capturedArgs)
	})

	t.Run("step-4 unknown args[0] + remote index has no match → InvalidProductOrPluginError", func(t *testing.T) {
		// Remote index loaded successfully but doesn't know args[0]
		// and the plugin isn't installed locally. Step-4 surfaces it
		// as an unknown product / plugin (with typo suggestions
		// populated from the remote index for the SuggestibleError
		// protocol).
		c, ctx := newTryDelegateHarness(t)
		c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-pds", ProductCode: "pds"},
			{Name: "aliyun-cli-fc", ProductCode: "fc"},
		}}
		helpDelegateIsInstalled = func(string) (bool, string, error) { return false, "", nil }

		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"zzzunknownproduct", "foo", "bar"})
		assert.True(t, delegated)
		var ipErr *InvalidProductOrPluginError
		assert.ErrorAs(t, err, &ipErr)
		assert.Equal(t, "zzzunknownproduct", ipErr.Code)
		// Hint must surface the OpenAPI legacy form so users who
		// landed here via mistyped APIName (e.g. `aliyun ecs
		// describeregions extra`) see the right syntax instead of
		// just "not a valid product".
		assert.NotEmpty(t, ipErr.Hint, "step-4 must always set Hint so users understand WHY they landed on the diagnostic")
		assert.Contains(t, err.Error(), "OpenAPI built-in call",
			"hint must explain the OpenAPI alternative")
		assert.Contains(t, err.Error(), "PascalCase",
			"hint must call out APIName casing convention — that's the typical lesson here")
		assert.Contains(t, err.Error(), "DescribeRegions",
			"a concrete PascalCase example anchors the explanation")
	})

	t.Run("step-4 unknown args[0] + no remote index → InvalidProductOrPluginError", func(t *testing.T) {
		// pluginIndex stays nil (default), nothing installed. Step-4
		// still kicks in: never fall through to "too many arguments"
		// once we've decided the shape is plugin-cmd.
		c, ctx := newTryDelegateHarness(t)
		c.pluginIndex = nil
		c.pluginIndexErr = nil
		helpDelegateIsInstalled = func(string) (bool, string, error) { return false, "", nil }

		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"zzzunknown", "foo", "bar"})
		assert.True(t, delegated, "step-4 must short-circuit the legacy 'too many arguments'")
		var ipErr *InvalidProductOrPluginError
		assert.ErrorAs(t, err, &ipErr)
		assert.Equal(t, "zzzunknown", ipErr.Code)
		// Hint is independent of the remote index — it only depends
		// on having reached step-4. Lock that in.
		assert.Contains(t, err.Error(), "OpenAPI built-in call")
	})

	t.Run("step-4 index fetch failed → fetch-note printed + InvalidProductOrPluginError", func(t *testing.T) {
		// Offline / catalog-load failure. We still produce a final
		// answer, plus we surface a fetch-failure note so the user
		// understands why fuzzy suggestions might be sparse.
		c, ctx := newTryDelegateHarness(t)
		c.pluginIndex = nil
		c.pluginIndexErr = fmt.Errorf("network unreachable")
		helpDelegateIsInstalled = func(string) (bool, string, error) { return false, "", nil }

		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"unknown", "x", "y"})
		assert.True(t, delegated)
		var ipErr *InvalidProductOrPluginError
		assert.ErrorAs(t, err, &ipErr)
		assert.Contains(t, err.Error(), "OpenAPI built-in call",
			"hint must be present even when the remote catalog couldn't load")
		// printPluginIndexLoadFailureNote writes to ctx.Stderr; assert
		// the side-effect happened so users actually get the heads-up.
		stderrBuf := ctx.Stderr().(*bytes.Buffer)
		assert.NotEmpty(t, stderrBuf.String(),
			"fetch-failure note must be visible to the user when the index can't be loaded")
	})

	t.Run("PascalCase args[1] never reaches step-1 (regression guard)", func(t *testing.T) {
		// Defensive duplicate of Test_tryDelegatePluginHelp_RefusesUppercaseSubcommand,
		// kept here so anyone reading PluginPath in isolation sees that
		// `ecs DescribeRegions extra` does NOT reach step-1, even when
		// the index, library, and local manifest are all populated.
		// Without this guard a future refactor could regress Gate-2 and
		// silently misroute legacy OpenAPI shapes into plugin diagnostics.
		c, ctx := newTryDelegateHarness(t)
		c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-ecs", ProductCode: "ecs"}, // tempt step-1
		}}
		helpDelegateIsInstalled = func(string) (bool, string, error) {
			t.Fatal("plugin-cmd predicate must reject PascalCase before any I/O")
			return false, "", nil
		}
		delegated, err := c.tryDelegatePluginHelp(ctx, []string{"ecs", "DescribeRegions", "extra"})
		assert.False(t, delegated, "PascalCase args[1] must fall through; legacy 'too many arguments' is the right error here")
		assert.NoError(t, err)
	})
}
