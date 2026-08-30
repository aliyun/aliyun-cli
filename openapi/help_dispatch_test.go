package openapi

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
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
			if test.pluginType == plugin.PluginTypeMeta {
				// Metadata plugins are served by host Machine Help: the
				// invocation must NOT be forwarded to the plugin process, and
				// host option validation applies exactly like the baseline path.
				assert.True(t, handled)
				assert.EqualError(t, err, "--help-all conflicts with --help")
				assert.Empty(t, received, "metadata-plugin Help must not delegate to the engine text path")
			} else {
				require.NoError(t, err)
				assert.True(t, handled)
				assert.Equal(t, input, received, "plugin owns even host-invalid Help option combinations")
			}
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

func TestHelpFlagRendersResponseSectionEquivalentToPrefix(t *testing.T) {
	for _, invocation := range [][]string{
		{"demo", "create-report", "--help", "--cli-section", "response"},
		{"help", "demo", "create-report", "--cli-section", "response"},
	} {
		t.Run(strings.Join(invocation, " "), func(t *testing.T) {
			t.Setenv(aimode.EnvAIMode, "0")
			c, stdout, stderr := newTestCommando()
			c.library.helpRepo = canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
			c.localLoaded = true
			c.localManifest = &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{}}
			root := testMachineHelpRootCommand()
			AddFlags(root.Flags())
			ctx := cli.NewCommandContext(stdout, stderr)
			ctx.EnterCommand(root)
			CliHelpSectionFlag(ctx.Flags()).SetAssigned(true)
			CliHelpSectionFlag(ctx.Flags()).SetValue("response")
			VersionFlag(ctx.Flags()).SetAssigned(true)
			VersionFlag(ctx.Flags()).SetValue("2026-01-01")
			ctx.SetInvocationArgs(invocation)

			// Production routing passes only the positional target to Help;
			// flags were already consumed by the parser.
			target, _, err := c.resolveParsedHelpTarget(ctx, []string{"demo", "create-report"})
			require.NoError(t, err)
			require.NoError(t, c.renderHostHelpTarget(ctx, target, false))
			assert.Contains(t, stdout.String(), "Responses:")
			assert.Contains(t, stdout.String(), `"200": {`)
		})
	}
}

func TestHostOwnsLegacyHelpCommand(t *testing.T) {
	c, _, _, _ := newCanonicalHelpTestContext(t)
	assert.True(t, c.hostOwnsLegacyHelpCommand([]string{"sts", "GetCallerIdentity", "--help"}), "PascalCase API names belong to the host legacy chain")
	assert.True(t, c.hostOwnsLegacyHelpCommand([]string{"help", "ecs", "DescribeRegions"}), "the help prefix keeps the same target semantics")
	assert.False(t, c.hostOwnsLegacyHelpCommand([]string{"sts", "get-caller-identity"}), "kebab commands belong to the plugin chain")
	assert.False(t, c.hostOwnsLegacyHelpCommand([]string{"cs", "GET", "/clusters"}), "HTTP verbs stay with the plugin (method+path exceeds the host Help model)")
	assert.False(t, c.hostOwnsLegacyHelpCommand([]string{"cs", "get", "/clusters"}), "HTTP verbs are case-insensitive")
	assert.False(t, c.hostOwnsLegacyHelpCommand([]string{"sts", "--help"}), "product-level Help keeps plugin ownership")
	assert.False(t, c.hostOwnsLegacyHelpCommand([]string{"demo", "CreateReport", "--help"}), "products unknown to the host stay plugin-owned")
}

func TestGoPluginHelpDelegationKeepsPascalCaseWithHost(t *testing.T) {
	newGoPluginCommando := func(t *testing.T) (*Commando, *cli.Context, *bytes.Buffer, *bytes.Buffer, *bool) {
		t.Helper()
		t.Setenv(aimode.EnvAIMode, "0")
		c, stdout, stderr := newTestCommando()
		c.localLoaded = true
		c.localManifest = &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
			"aliyun-cli-sts": {Name: "aliyun-cli-sts", Version: "1.0.0", Type: plugin.PluginTypeGo},
		}}
		dispatched := false
		original := goPluginHelpDispatch
		goPluginHelpDispatch = func(_ string, _ []string, _ *cli.Context) (bool, error) {
			dispatched = true
			return true, nil
		}
		t.Cleanup(func() { goPluginHelpDispatch = original })
		root := testMachineHelpRootCommand()
		AddFlags(root.Flags())
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(root)
		return c, ctx, stdout, stderr, &dispatched
	}

	t.Run("pascal case api renders from host", func(t *testing.T) {
		c, ctx, stdout, _, dispatched := newGoPluginCommando(t)
		ctx.SetInvocationArgs([]string{"sts", "GetCallerIdentity", "--help"})
		require.NoError(t, c.help(ctx, []string{"sts", "GetCallerIdentity"}))
		assert.False(t, *dispatched, "PascalCase Help must resolve against the same owner that executes it (host legacy chain)")
		assert.Contains(t, stdout.String(), "GetCallerIdentity")
		assert.Contains(t, stdout.String(), "sts")
	})

	t.Run("kebab command still delegates", func(t *testing.T) {
		c, ctx, _, _, dispatched := newGoPluginCommando(t)
		delegated, err := c.delegateInstalledPluginHelp(ctx, []string{"sts", "get-caller-identity", "--help"})
		require.NoError(t, err)
		assert.True(t, delegated)
		assert.True(t, *dispatched)
	})

	t.Run("product level help still delegates", func(t *testing.T) {
		c, ctx, _, _, dispatched := newGoPluginCommando(t)
		delegated, err := c.delegateInstalledPluginHelp(ctx, []string{"sts", "--help"})
		require.NoError(t, err)
		assert.True(t, delegated)
		assert.True(t, *dispatched)
	})

	t.Run("http verb still delegates", func(t *testing.T) {
		c, ctx, _, _, dispatched := newGoPluginCommando(t)
		delegated, err := c.delegateInstalledPluginHelp(ctx, []string{"sts", "GET", "--help"})
		require.NoError(t, err)
		assert.True(t, delegated)
		assert.True(t, *dispatched)
	})
}
