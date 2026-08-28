package openapi

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeforeParseHelpRouteDelegatesInstalledPluginBeforeHostValidation(t *testing.T) {
	tests := []struct {
		name       string
		pluginType string
	}{
		{name: "go", pluginType: plugin.PluginTypeGo},
		{name: "meta", pluginType: plugin.PluginTypeMeta},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, ctx, _, _ := newCanonicalHelpTestContext(t)
			c.localManifest = &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
				"aliyun-cli-demo": {Name: "aliyun-cli-demo", Version: "1.0.0", Type: test.pluginType},
			}}
			input := []string{"demo", "CreateReport", "--help", "--help-all", "--cli-output", "json"}
			var received []string
			if test.pluginType == plugin.PluginTypeMeta {
				original := metaPluginHelpDispatch
				metaPluginHelpDispatch = func(_ *cli.Context, args []string) error {
					received = append([]string(nil), args...)
					return nil
				}
				t.Cleanup(func() { metaPluginHelpDispatch = original })
			} else {
				original := goPluginHelpDispatch
				goPluginHelpDispatch = func(_ string, args []string, _ *cli.Context) (bool, error) {
					received = append([]string(nil), args...)
					return true, nil
				}
				t.Cleanup(func() { goPluginHelpDispatch = original })
			}

			handled, err := c.beforeParseHelpRoute(ctx, input)
			require.NoError(t, err)
			assert.True(t, handled)
			assert.Equal(t, input, received, "plugin owns even host-invalid Help option combinations")
			assert.Equal(t, []string{"demo", "CreateReport", "--help", "--help-all", "--cli-output", "json"}, input)
		})
	}
}

func TestBeforeParseHelpRouteRendersCanonicalL3WithoutParsingAValue(t *testing.T) {
	c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
	args := []string{"demo", "CreateReport", "--ReportId", "--help", "--version", "2026-01-01"}
	handled, err := c.beforeParseHelpRoute(ctx, args)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, stdout.String(), "--ReportId")
	assert.Contains(t, stdout.String(), "string")
	assert.NotContains(t, stdout.String(), "--WorkspaceId")
}

func TestBeforeParseHelpRouteRejectsAmbiguousUnassignedL3Parameters(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	handled, err := c.beforeParseHelpRoute(ctx, []string{
		"demo", "CreateReport", "--ReportId", "--WorkspaceId", "--help",
	})
	assert.True(t, handled)
	var invalidOptions *InvalidOptionCombinationError
	require.True(t, errors.As(err, &invalidOptions))
	assert.Contains(t, err.Error(), "parameter Help target is ambiguous")
}

func TestResolveParsedHelpTargetTypesSectionAllConflict(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	CliHelpSectionFlag(ctx.Flags()).SetAssigned(true)
	CliHelpSectionFlag(ctx.Flags()).SetValue("request")
	CliHelpAllFlag(ctx.Flags()).SetAssigned(true)
	ctx.SetInvocationArgs([]string{
		"help", "demo", "CreateReport", "--cli-section", "request", "--help-all",
	})
	_, _, err := c.resolveParsedHelpTarget(ctx, []string{"demo", "CreateReport"})
	var invalidOptions *InvalidOptionCombinationError
	require.True(t, errors.As(err, &invalidOptions))
	assert.Equal(t, []string{"--cli-section", "--help-all"}, invalidOptions.Options)
}

func TestBeforeParseHelpRouteRejectsRemovedHostHelpFlags(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	for _, args := range [][]string{
		{"--cli-all"}, {"--cli-search", "ecs"}, {"--help=json"}, {"--help-json"},
		{"help", "--format", "json"}, {"demo", "--cli-search", "report"}, {"demo", "--help=json"},
	} {
		handled, err := c.beforeParseHelpRoute(ctx, args)
		assert.True(t, handled)
		var invalid *cli.InvalidFlagError
		assert.True(t, errors.As(err, &invalid), "args=%v err=%T %v", args, err, err)
	}
}

func TestBeforeParseHelpRouteRejectsUnknownRootFlag(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	handled, err := c.beforeParseHelpRoute(ctx, []string{"--regoin", "cn-hangzhou"})
	assert.True(t, handled)
	var invalid *cli.InvalidFlagError
	require.True(t, errors.As(err, &invalid))
	assert.Equal(t, "--regoin", invalid.Flag)
}

func TestBeforeParseHelpRouteRejectsUnknownProductFlag(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	handled, err := c.beforeParseHelpRoute(ctx, []string{"demo", "--hekp"})

	assert.True(t, handled)
	var invalid *cli.InvalidFlagError
	require.True(t, errors.As(err, &invalid))
	assert.Equal(t, "--hekp", invalid.Flag)
}

func TestValidateCanonicalAPICommandPrecedesProfileResolution(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	ctx.SetInvocationArgs([]string{"demo", "MissingAPI", "--version", "2026-01-01"})
	err := c.validateCanonicalAPICommand([]string{"demo", "MissingAPI"}, ctx)
	var invalid *InvalidApiError
	require.True(t, errors.As(err, &invalid))
	assert.Equal(t, "MissingAPI", invalid.Name)

	ctx.SetInvocationArgs([]string{"demo", "CreateReport", "--version", "2026-01-01"})
	require.NoError(t, c.validateCanonicalAPICommand([]string{"demo", "CreateReport"}, ctx))

	ctx.SetUnknownFlags(cli.NewFlagSet())
	unknown, addErr := ctx.UnknownFlags().AddByName("ReprotId")
	require.NoError(t, addErr)
	unknown.SetAssigned(true)
	unknown.SetValue("r-1")
	err = c.validateCanonicalAPICommand([]string{"demo", "CreateReport"}, ctx)
	var invalidParameter *InvalidParameterError
	require.True(t, errors.As(err, &invalidParameter))
	assert.Equal(t, "ReprotId", invalidParameter.Name)
}

func TestLowercaseCanonicalCommandIsValidatedBeforeRuntimeProfileResolution(t *testing.T) {
	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	t.Cleanup(cleanup)
	require.NoError(t, os.MkdirAll(filepath.Join(testHome, ".aliyun", "plugins"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(testHome, ".aliyun", "plugins", "manifest.json"),
		[]byte(`{"plugins":{}}`),
		0644,
	))

	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	ctx.SetInvocationArgs([]string{"demo", "get-caller"})
	originalArgs := os.Args
	os.Args = []string{"aliyun", "demo", "get-caller"}
	t.Cleanup(func() { os.Args = originalArgs })

	runtimeCalled := false
	originalDispatch := runtimeTryDispatch
	runtimeTryDispatch = func(_ *cli.Context, _ []string) (bool, error) {
		runtimeCalled = true
		return true, errors.New("profile default: failed to resolve credentials")
	}
	t.Cleanup(func() { runtimeTryDispatch = originalDispatch })

	err := c.main(ctx, []string{"demo", "get-caller"})
	var invalid *InvalidApiError
	require.True(t, errors.As(err, &invalid), "err=%T %v", err, err)
	assert.Equal(t, "get-caller", invalid.Name)
	assert.False(t, runtimeCalled, "local command identity must be checked before runtime/profile setup")
}

func TestBundledSTSUnknownCommandIsLocallyIdentified(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	root := testMachineHelpRootCommand()
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	ctx.SetInvocationArgs([]string{"sts", "get"})
	assert.False(t, EstimateCostFlag(ctx.Flags()).IsAssigned())
	assert.False(t, ForceFlag(ctx.Flags()).IsAssigned())

	err := c.validateCanonicalRuntimeCommand([]string{"sts", "get"}, ctx)
	var invalid *InvalidApiError
	require.True(t, errors.As(err, &invalid), "err=%T %v", err, err)
	assert.Equal(t, "get", invalid.Name)
	require.NoError(t, c.validateCanonicalAPICommand([]string{"sts", "GET"}, ctx), "uppercase REST method remains exempt")
}
