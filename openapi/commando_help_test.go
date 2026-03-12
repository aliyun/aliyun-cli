package openapi

import (
	"bytes"
	"os"
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
