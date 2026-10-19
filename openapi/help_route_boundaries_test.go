package openapi

import (
	"io"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/require"
)

func TestHelpRouteRejectsInvalidUtilityControls(t *testing.T) {
	for _, args := range [][]string{
		{"utils", "--help", "--cli-output", "xml"},
		{"utils", "--help", "--cli-section", "response"},
		{"demo", "CreateReport", "--help-search", ""},
	} {
		c, ctx, _, _ := newCanonicalHelpTestContext(t)
		handled, err := c.beforeParseHelpRoute(ctx, args)
		require.True(t, handled)
		require.Error(t, err)
	}
	parameter, ambiguous := rawParameterHelpTarget([]string{"demo", "CreateReport", "--Name", "assigned", "--Id", "--help"})
	require.False(t, ambiguous)
	require.Equal(t, "--Id", parameter)
	flag, version := rawExplicitVersion([]string{"--api-version=2026-01-01"})
	require.Equal(t, VersionFlagAPI, flag)
	require.Equal(t, "2026-01-01", version)
}

func TestInstalledPluginHelpPreservesRemovedOptionsAndReportsMissingExecutable(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	c.localManifest = &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{"aliyun-cli-fixture": {Command: "fixture", ProductCode: "fixture", Type: plugin.PluginTypeGo}}}
	previous := goPluginHelpDispatch
	t.Cleanup(func() { goPluginHelpDispatch = previous })
	calls := 0
	goPluginHelpDispatch = func(product string, args []string, context *cli.Context) (bool, error) {
		calls++
		require.Equal(t, "fixture", product)
		require.Same(t, ctx, context)
		require.Contains(t, args, "--help")
		return false, nil
	}
	handled, err := c.beforeParseHelpRoute(ctx, []string{"fixture", "list", "--cli-all"})
	require.False(t, handled)
	require.NoError(t, err)
	require.Zero(t, calls)
	handled, err = c.beforeParseHelpRoute(ctx, []string{"fixture", "list", "--cli-all", "--help"})
	require.True(t, handled)
	require.ErrorContains(t, err, "plugin aliyun-cli-fixture not found")
	require.Equal(t, 1, calls)
	handled, err = c.delegateInstalledPluginHelp(ctx, []string{"help", "fixture"})
	require.True(t, handled)
	require.Error(t, err)
	require.Equal(t, 2, calls)
	ctx.Command().AddSubCommand(&cli.Command{Name: "fixture"})
	handled, err = c.delegateInstalledPluginHelp(ctx, []string{"fixture", "--help"})
	require.False(t, handled)
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.False(t, (&Commando{}).hostOwnsLegacyHelpCommand([]string{"fixture", "GetInfo"}))
}

func TestHostHelpHandlesRequestSectionsParameterSearchAndLanguage(t *testing.T) {
	for _, parameter := range []bool{false, true} {
		c, ctx, output, _ := newCanonicalHelpTestContext(t)
		target := HelpTarget{Level: HelpLevelAction, Product: "demo", Action: "CreateReport", CommandStyle: CommandStyleCamel, Operation: HelpOperationDefault, Section: HelpSectionRequest, SectionExplicit: true, Output: HelpOutputText}
		if parameter {
			target.Level, target.Parameter, target.SectionExplicit, target.Operation, target.SearchQuery = HelpLevelParameter, "--ReportId", false, HelpOperationSearch, "report"
		}
		require.NoError(t, c.renderHostHelpTarget(ctx, target, false))
		require.Contains(t, output.String(), "ReportId")
	}
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	previous := i18n.GetLanguage()
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	config.AddFlags(ctx.Flags())
	config.LanguageFlag(ctx.Flags()).SetAssigned(true)
	config.LanguageFlag(ctx.Flags()).SetValue("zh")
	c.applyHostHelpLanguage(ctx)
	require.Equal(t, "zh", i18n.GetLanguage())
	require.Equal(t, "en", rawHelpLanguage([]string{"--language=en"}))
	target := HelpTarget{Level: HelpLevelProduct, Product: "absent", Output: HelpOutputJSON}
	var structured *machineHelpError
	require.ErrorAs(t, c.renderHostHelpTarget(ctx, target, false), &structured)
	require.Error(t, c.renderHostHelpTarget(ctx, HelpTarget{Level: "invalid"}, false))
	CliHelpAllFlag(ctx.Flags()).SetAssigned(true)
	parsed, _, err := c.resolveParsedHelpTarget(ctx, []string{"demo"})
	require.NoError(t, err)
	require.Equal(t, HelpOperationAll, parsed.Operation)
	writerContext := cli.NewCommandContext(&endpointFailWriter{}, io.Discard)
	writerContext.EnterCommand(ctx.Command())
	require.Error(t, c.renderHostHelpTarget(writerContext, HelpTarget{Level: HelpLevelProduct, Product: "demo", Output: HelpOutputText}, false))
}
