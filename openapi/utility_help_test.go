package openapi

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func utilityHelpRoot() *cli.Command {
	root := &cli.Command{Name: "aliyun"}
	utils := &cli.Command{
		Name: "utils", Short: i18n.T("Local utilities", "本地工具"),
		Usage: "utils <command>", Sample: "aliyun utils doctor",
	}
	utils.AddSubCommand(&cli.Command{Name: "doctor", Short: i18n.T("Diagnose the CLI", "诊断 CLI")})
	utils.AddSubCommand(&cli.Command{Name: "hidden", Hidden: true, Short: i18n.T("Hidden", "隐藏")})
	utils.Flags().Add(&cli.Flag{
		Name: "verbose", Shorthand: 'v', Aliases: []string{"debug"}, Category: "output",
		Short: i18n.T("Show diagnostic details", "显示诊断详情"),
	})
	utils.Flags().Add(&cli.Flag{Name: "internal", Hidden: true, Short: i18n.T("Internal", "内部")})
	root.AddSubCommand(utils)
	return root
}

func TestBuildUtilityHelpDocumentValidatesTargetsAndProjectsVisibleEntries(t *testing.T) {
	root := utilityHelpRoot()
	for _, target := range [][]string{nil, {"plugin"}, {"utils", "doctor", "extra"}} {
		if _, err := buildUtilityHelpDocument(root, target); err == nil {
			t.Fatalf("target %#v unexpectedly succeeded", target)
		}
	}
	if _, err := buildUtilityHelpDocument(nil, []string{"utils"}); err == nil {
		t.Fatal("nil root unexpectedly succeeded")
	}
	if _, err := buildUtilityHelpDocument(&cli.Command{Name: "aliyun"}, []string{"utils"}); err == nil {
		t.Fatal("missing utils command unexpectedly succeeded")
	}
	if _, err := buildUtilityHelpDocument(root, []string{"utils", "missing"}); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("unknown utility error = %v", err)
	}

	document, err := buildUtilityHelpDocument(root, []string{"utils"})
	require.NoError(t, err)
	assert.Equal(t, "utility", document.Kind)
	assert.Equal(t, []string{"aliyun", "utils"}, document.Target.Path)
	assert.Equal(t, "utils <command>", document.Usage)
	assert.Equal(t, "aliyun utils doctor", document.Sample)
	require.Len(t, document.Commands, 1)
	assert.Equal(t, "doctor", document.Commands[0].Name)
	require.Len(t, document.Flags, 1)
	assert.Equal(t, "--verbose", document.Flags[0].Name)
	assert.Equal(t, "-v", document.Flags[0].Shorthand)
	assert.Equal(t, []string{"debug"}, document.Flags[0].Aliases)
	assert.Equal(t, HelpResult{Shown: 2, Total: 2}, document.Result)

	child, err := buildUtilityHelpDocument(root, []string{"utils", "doctor"})
	require.NoError(t, err)
	assert.Equal(t, "utils doctor", child.Name)
	assert.Equal(t, []string{"aliyun", "utils", "doctor"}, child.Target.Path)
}

func TestUtilityHelpSearchProjectsCommandsFlagsAndNoMatches(t *testing.T) {
	build := func() *machineHelpUtilityDocument {
		document, err := buildUtilityHelpDocument(utilityHelpRoot(), []string{"utils"})
		require.NoError(t, err)
		return document
	}

	command := build()
	applyUtilityHelpOptions(command, cli.HelpOptions{Operation: cli.HelpOperationSearch, SearchQuery: "diagnose"})
	assert.Equal(t, "diagnose", command.Query)
	assert.Len(t, command.Commands, 1)
	assert.Empty(t, command.Flags)
	assert.Equal(t, HelpResult{Shown: 1, Total: 1}, command.Result)

	flag := build()
	applyUtilityHelpOptions(flag, cli.HelpOptions{Operation: cli.HelpOperationSearch, SearchQuery: "debug"})
	assert.Empty(t, flag.Commands)
	require.Len(t, flag.Flags, 1)
	assert.Equal(t, "--verbose", flag.Flags[0].Name)

	missing := build()
	applyUtilityHelpOptions(missing, cli.HelpOptions{Operation: cli.HelpOperationSearch, SearchQuery: "absent"})
	assert.Empty(t, missing.Commands)
	assert.Empty(t, missing.Flags)
	assert.Equal(t, HelpResult{}, missing.Result)

	unchanged := build()
	applyUtilityHelpOptions(unchanged, cli.HelpOptions{})
	assert.Len(t, unchanged.Commands, 1)
	applyUtilityHelpOptions(nil, cli.HelpOptions{Operation: cli.HelpOperationSearch, SearchQuery: "doctor"})
}

func TestRenderUtilityHelpTextAndJSON(t *testing.T) {
	previousLanguage := i18n.GetLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	root := utilityHelpRoot()
	commando, _, _ := newTestCommando()

	t.Run("text", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		ctx := cli.NewCommandContext(&stdout, &stderr)
		ctx.EnterCommand(root)
		require.NoError(t, commando.renderUtilityHelp(ctx, []string{"utils"}, cli.HelpOptions{}, false))
		for _, want := range []string{
			"Local utilities", "Usage:", "aliyun utils <command>", "Commands:", "doctor",
			"Flags:", "--verbose, -v", "Example:", "aliyun utils doctor", cli.AIModeEnableCommand,
		} {
			assert.Contains(t, stdout.String(), want)
		}
		assert.Empty(t, stderr.String())
	})

	t.Run("search miss text", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		ctx := cli.NewCommandContext(&stdout, &stderr)
		ctx.EnterCommand(root)
		require.NoError(t, commando.renderUtilityHelp(ctx, []string{"utils"}, cli.HelpOptions{
			Operation: cli.HelpOperationSearch, SearchQuery: "absent",
		}, false))
		assert.Contains(t, stdout.String(), `No Help entries matched --help-search "absent".`)
	})

	t.Run("json", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		ctx := cli.NewCommandContext(&stdout, &stderr)
		ctx.EnterCommand(root)
		require.NoError(t, commando.renderUtilityHelp(ctx, []string{"utils"}, cli.HelpOptions{
			Output: cli.HelpOutputJSON,
		}, false))
		var document map[string]any
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &document))
		assert.Equal(t, "utility", document["kind"])
		assert.Contains(t, document, "aiModeHint")
	})

	t.Run("AI JSON", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		ctx := cli.NewCommandContext(&stdout, &stderr)
		ctx.EnterCommand(root)
		require.NoError(t, commando.renderUtilityHelp(ctx, []string{"utils"}, cli.HelpOptions{}, true))
		assert.Equal(t, 1, strings.Count(stdout.String(), "\n"))
		assert.NotContains(t, stdout.String(), "aiModeHint")
	})
}
