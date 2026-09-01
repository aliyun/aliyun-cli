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

func helpNextJSON(t *testing.T, next *HelpNext) map[string]any {
	t.Helper()
	require.NotNil(t, next)
	raw, err := json.Marshal(next)
	require.NoError(t, err)
	var value map[string]any
	require.NoError(t, json.Unmarshal(raw, &value))
	return value
}

func TestProgressiveHelpNextAcrossSevenLevels(t *testing.T) {
	targets := []struct {
		name   string
		target HelpTarget
	}{
		{name: "root", target: HelpTarget{Level: HelpLevelRoot}},
		{name: "product", target: HelpTarget{Level: HelpLevelProduct, Product: "demo", CommandStyle: CommandStyleCamel}},
		{name: "api", target: HelpTarget{Level: HelpLevelAction, Product: "demo", Action: "CreateReport", CommandStyle: CommandStyleCamel}},
		{name: "parameter", target: HelpTarget{Level: HelpLevelParameter, Product: "demo", Action: "CreateReport", Parameter: "ReportFilter", CommandStyle: CommandStyleCamel}},
		{name: "utility", target: HelpTarget{Level: HelpLevelUtility, Utility: "doctor"}},
		{name: "request section", target: HelpTarget{Level: HelpLevelAction, Product: "demo", Action: "CreateReport", CommandStyle: CommandStyleCamel, Section: HelpSectionRequest, SectionExplicit: true}},
		{name: "response section", target: HelpTarget{Level: HelpLevelAction, Product: "demo", Action: "CreateReport", CommandStyle: CommandStyleCamel, Section: HelpSectionResponse, SectionExplicit: true}},
	}

	for _, test := range targets {
		t.Run(test.name+" search", func(t *testing.T) {
			target := test.target
			target.Operation = HelpOperationSearch
			target.SearchQuery = "report id"
			next := helpNextJSON(t, buildHelpNext(target, false))
			assert.NotEmpty(t, next["search"])
			assert.NotEmpty(t, next["searchAll"])
			assert.NotContains(t, next, "showAll")
			if test.target.SectionExplicit {
				assert.Contains(t, next["search"], "aliyun help demo CreateReport")
				assert.Contains(t, next["search"], "--cli-section "+string(test.target.Section))
			}
		})

		t.Run(test.name+" search all", func(t *testing.T) {
			target := test.target
			target.Operation = HelpOperationSearch
			target.SearchQuery = "report id"
			target.SearchAll = true
			next := helpNextJSON(t, buildHelpNext(target, false))
			assert.NotEmpty(t, next["search"])
			assert.NotContains(t, next, "searchAll", "Search-All must not advertise itself")
			assert.NotContains(t, next, "showAll")
		})
	}

	for _, test := range targets[:5] {
		t.Run(test.name+" all", func(t *testing.T) {
			target := test.target
			target.Operation = HelpOperationAll
			next := helpNextJSON(t, buildHelpNext(target, false))
			assert.NotEmpty(t, next["search"])
			assert.NotContains(t, next, "showAll")
			assert.NotContains(t, next, "searchAll")
		})
	}
}

func TestProgressiveHelpSearchAddsOnlyUniqueExactChild(t *testing.T) {
	t.Run("root exact product keeps fuzzy sibling", func(t *testing.T) {
		document := &machineHelpRootDocument{
			Commands: []machineHelpCommandSummary{{Name: "configure"}},
			Products: []machineHelpProductSummary{{Code: "demo"}, {Code: "demo-extra"}},
		}
		applyRootHelpOptions(document, helpOptions{Search: "DE_MO"}, false)
		target := HelpTarget{Level: HelpLevelRoot, Operation: HelpOperationSearch, SearchQuery: "DE_MO"}
		setRootHelpNext(document, target, false)

		assert.Equal(t, 2, document.Result.Total, "fuzzy siblings remain visible")
		next := helpNextJSON(t, document.Next)
		assert.Equal(t, "aliyun demo --help-search <keyword>", next["childSearch"])
	})

	t.Run("root ambiguous exact does not descend", func(t *testing.T) {
		document := &machineHelpRootDocument{
			Commands: []machineHelpCommandSummary{{Name: "demo"}},
			Products: []machineHelpProductSummary{{Code: "demo"}},
		}
		applyRootHelpOptions(document, helpOptions{Search: "demo"}, false)
		target := HelpTarget{Level: HelpLevelRoot, Operation: HelpOperationSearch, SearchQuery: "demo"}
		setRootHelpNext(document, target, false)

		next := helpNextJSON(t, document.Next)
		assert.NotContains(t, next, "childSearch")
	})

	t.Run("product exact api preserves kebab version and JSON", func(t *testing.T) {
		document := &machineHelpProductDocument{APIs: []machineHelpAPISummary{
			{Name: "CreateReport", CmdName: "create-report", DisplayName: "create-report"},
			{Name: "CreateReportDetail", CmdName: "create-report-detail", DisplayName: "create-report-detail"},
		}}
		applyProductHelpOptions(document, helpOptions{Search: "CREATE_REPORT"}, false)
		target := HelpTarget{
			Level: HelpLevelProduct, Product: "demo", CommandStyle: CommandStyleKebab,
			VersionFlag: VersionFlagAPI, Version: "2026-01-01",
			Operation: HelpOperationSearch, SearchQuery: "CREATE_REPORT", Output: HelpOutputJSON,
		}
		setProductHelpNext(document, target, false)

		assert.Equal(t, 2, document.Result.Total)
		next := helpNextJSON(t, document.Next)
		assert.Equal(t,
			"aliyun demo create-report --api-version 2026-01-01 --help-search <keyword> --cli-output json",
			next["childSearch"])
	})

	t.Run("api exact composite descends but scalar and fuzzy do not", func(t *testing.T) {
		build := func(query string, composite bool) (*machineHelpAPIDocument, HelpTarget) {
			parameter := machineHelpParameter{
				Name: "ReportFilter", RawName: "ReportFilter", Options: []string{"--ReportFilter"}, Type: "object",
			}
			if composite {
				parameter.Fields = []machineHelpParameter{{Name: "Status", RawName: "Status", Type: "string"}}
			}
			document := &machineHelpAPIDocument{
				ActiveParameterSet: "camel",
				ParameterSets: machineHelpParameterSets{Camel: []machineHelpParameter{
					parameter,
					{Name: "ReportFilterExtra", RawName: "ReportFilterExtra", Options: []string{"--ReportFilterExtra"}, Type: "string"},
				}},
			}
			applyRequestHelpOptions(document, helpOptions{Search: query}, false)
			target := HelpTarget{
				Level: HelpLevelAction, Product: "demo", Action: "CreateReport", CommandStyle: CommandStyleCamel,
				VersionFlag: VersionFlagLegacy, Version: "2026-01-01",
				Section: HelpSectionRequest, SectionExplicit: true,
				Operation: HelpOperationSearch, SearchQuery: query,
			}
			setActionHelpNext(document, target, false)
			return document, target
		}

		composite, _ := build("report_filter", true)
		assert.Equal(t, 2, composite.Result.Total)
		assert.Equal(t,
			"aliyun demo CreateReport --version 2026-01-01 --ReportFilter --help-search <keyword>",
			helpNextJSON(t, composite.Next)["childSearch"])

		scalar, _ := build("report_filter", false)
		assert.NotContains(t, helpNextJSON(t, scalar.Next), "childSearch")

		fuzzy, _ := build("report", true)
		assert.NotContains(t, helpNextJSON(t, fuzzy.Next), "childSearch")
	})
}

func TestHelpHintTextLabelsAndOptionalJSON(t *testing.T) {
	previous := i18n.GetLanguage()
	t.Cleanup(func() { i18n.SetLanguage(previous) })

	t.Run("all English", func(t *testing.T) {
		i18n.SetLanguage("en")
		next := buildHelpNext(HelpTarget{Level: HelpLevelRoot, Operation: HelpOperationAll}, false)
		var output bytes.Buffer
		require.NoError(t, renderHelpProjectionResult(&output, "entries", HelpResult{}, next))
		assert.Equal(t, "\nSearch this Help:\n  aliyun --help-search <keyword> [--cli-output json]\n", output.String())
	})

	t.Run("search Chinese", func(t *testing.T) {
		i18n.SetLanguage("zh")
		next := buildHelpNext(HelpTarget{Level: HelpLevelRoot, Operation: HelpOperationSearch, SearchQuery: "ecs"}, false)
		var output bytes.Buffer
		require.NoError(t, renderHelpProjectionResult(&output, "entries", HelpResult{Shown: 1, Total: 1}, next))
		assert.Contains(t, output.String(), "更换关键词重新搜索：\n  aliyun --help-search <keyword> [--cli-output json]")
		assert.Contains(t, output.String(), "查看全部匹配：\n  aliyun --help-search ecs --help-all [--cli-output json]")
		assert.Contains(t, output.String(), "[--cli-output json]\n\n查看全部匹配：")
	})

	t.Run("explicit JSON stays executable", func(t *testing.T) {
		next := buildHelpNext(HelpTarget{
			Level: HelpLevelProduct, Product: "demo", CommandStyle: CommandStyleCamel,
			Operation: HelpOperationSearch, SearchQuery: "report", Output: HelpOutputJSON,
		}, false)
		encoded := helpNextJSON(t, next)
		assert.True(t, strings.HasSuffix(encoded["search"].(string), "--cli-output json"))
		assert.True(t, strings.HasSuffix(encoded["searchAll"].(string), "--cli-output json"))

		ai := helpNextJSON(t, buildHelpNext(HelpTarget{
			Level: HelpLevelProduct, Product: "demo", CommandStyle: CommandStyleCamel,
			Operation: HelpOperationSearch, SearchQuery: "report", Output: HelpOutputJSON,
		}, true))
		assert.NotContains(t, ai["search"], "--cli-output json")
		assert.NotContains(t, ai["searchAll"], "--cli-output json")
	})
}

func TestDefaultTextHelpAddsMachineReadableCommandBeforeAIModeHint(t *testing.T) {
	c, ctx, stdout, stderr := newCanonicalHelpTestContext(t)
	target := HelpTarget{Level: HelpLevelRoot, Operation: HelpOperationDefault, Output: HelpOutputText}
	require.NoError(t, c.renderHostHelpTarget(ctx, target, false))
	assert.Empty(t, stderr.String())

	output := stdout.String()
	machine := "For machine-readable Help, run:\n  aliyun --help --cli-output json"
	ai := "For AI agents, run:"
	assert.Contains(t, output, machine)
	assert.Contains(t, output, ai)
	assert.Less(t, strings.Index(output, machine), strings.Index(output, ai))
}

func TestHelpHintHelperEdgeContracts(t *testing.T) {
	t.Run("parameter navigation guards and eligibility", func(t *testing.T) {
		target := HelpTarget{Level: HelpLevelParameter, Product: "demo", Action: "CreateReport", Parameter: "Config"}
		setParameterHelpNext(nil, target, false)

		document := &machineHelpParameterDocument{}
		setParameterHelpNext(document, target, false)
		assert.Nil(t, document.Next)

		target.Operation = HelpOperationAll
		setParameterHelpNext(document, target, false)
		require.NotNil(t, document.Next)

		document.Next = nil
		document.Result = &HelpResult{Truncated: true}
		target.Operation = HelpOperationDefault
		setParameterHelpNext(document, target, false)
		require.NotNil(t, document.Next)
	})

	t.Run("utility target preserves parsed options", func(t *testing.T) {
		target := utilityHelpTarget([]string{"utils", "mcp-proxy"}, cli.HelpOptions{
			Operation: HelpOperationSearch, SearchQuery: "oauth", SearchAll: true, Output: cli.HelpOutputJSON,
		})
		assert.Equal(t, HelpLevelUtility, target.Level)
		assert.Equal(t, "mcp-proxy", target.Utility)
		assert.Equal(t, HelpOperationSearch, target.Operation)
		assert.Equal(t, "oauth", target.SearchQuery)
		assert.True(t, target.SearchAll)
		assert.Equal(t, HelpOutputJSON, target.Output)

		root := utilityHelpTarget([]string{"utils"}, cli.HelpOptions{})
		assert.Empty(t, root.Utility)
		assert.Equal(t, HelpOperationDefault, root.Operation)
	})

	t.Run("parameter shape and preferred option recursion", func(t *testing.T) {
		assert.Equal(t, "Config", preferredHelpParameterOption(machineHelpParameter{Options: []string{"--Config"}}))
		assert.Equal(t, "Raw", preferredHelpParameterOption(machineHelpParameter{RawName: "Raw", Name: "name"}))
		assert.Equal(t, "name", preferredHelpParameterOption(machineHelpParameter{Name: "name"}))

		assert.False(t, machineHelpShapeHasFields(nil))
		assert.True(t, machineHelpShapeHasFields(&machineHelpShape{Fields: []machineHelpParameter{{Name: "field"}}}))
		assert.True(t, machineHelpShapeHasFields(&machineHelpShape{Element: &machineHelpShape{Fields: []machineHelpParameter{{Name: "field"}}}}))
		assert.True(t, machineHelpShapeHasFields(&machineHelpShape{Value: &machineHelpShape{Fields: []machineHelpParameter{{Name: "field"}}}}))
		assert.False(t, machineHelpParameterHasNestedFields(machineHelpParameter{}))
		assert.True(t, machineHelpParameterHasNestedFields(machineHelpParameter{Value: &machineHelpShape{Fields: []machineHelpParameter{{Name: "field"}}}}))
	})

	t.Run("invalid child navigation stays absent", func(t *testing.T) {
		next := &HelpNext{}
		parent := HelpTarget{Level: HelpLevelAction, Operation: HelpOperationSearch}
		attachHelpChildSearch(next, parent, false, helpHintExact{count: 1, kind: helpHintChildAPI, action: "CreateReport"})
		assert.Empty(t, next.ChildSearch, "invalid action target must not emit a broken command")

		attachHelpChildSearch(next, parent, false, helpHintExact{count: 1, kind: helpHintChildKind(255)})
		assert.Empty(t, next.ChildSearch)
	})

	t.Run("footer labels and writer errors", func(t *testing.T) {
		previous := i18n.GetLanguage()
		i18n.SetLanguage("en")
		t.Cleanup(func() { i18n.SetLanguage(previous) })

		assert.NoError(t, renderHelpHintFooter(&bytes.Buffer{}, nil, true))
		assert.Empty(t, helpHintTextCommand("  "))
		assert.NoError(t, renderHelpHintFooter(&bytes.Buffer{}, &HelpNext{}, false))

		for _, test := range []struct {
			kind helpHintChildKind
			name string
			want string
		}{
			{helpHintChildProduct, "demo", "Search APIs in demo"},
			{helpHintChildAPI, "CreateReport", "Search parameters in CreateReport"},
			{helpHintChildParameter, "Config", "Search nested fields in Config"},
		} {
			var output bytes.Buffer
			next := &HelpNext{
				operation: HelpOperationSearch, Search: "search", SearchAll: "search-all",
				ChildSearch: "child-search", childKind: test.kind, childName: test.name,
			}
			require.NoError(t, renderHelpHintFooter(&output, next, false))
			assert.Contains(t, output.String(), test.want)
		}

		next := &HelpNext{
			operation: HelpOperationSearch, Search: "search", SearchAll: "search-all",
			ChildSearch: "child-search", childKind: helpHintChildProduct, childName: "demo",
		}
		exerciseWriterFailures(t, func(w *failAfterWrites) error {
			return renderHelpHintFooter(w, next, true)
		})
	})

	t.Run("machine readable hint propagates target and writer errors", func(t *testing.T) {
		assert.Error(t, renderMachineReadableHelpHint(&bytes.Buffer{}, HelpTarget{Level: HelpLevel("invalid")}))
		assert.Error(t, renderMachineReadableHelpHint(alwaysFailWriter{}, HelpTarget{Level: HelpLevelRoot}))

		commando := &Commando{}
		defaultContext := cli.NewCommandContext(alwaysFailWriter{}, &bytes.Buffer{})
		assert.Error(t, commando.finishCanonicalTextHelp(defaultContext, false, HelpTarget{
			Level: HelpLevelRoot, Operation: HelpOperationDefault,
		}))

		searchContext := cli.NewCommandContext(alwaysFailWriter{}, &bytes.Buffer{})
		assert.Error(t, commando.finishCanonicalTextHelp(searchContext, false, HelpTarget{
			Level: HelpLevelRoot, Operation: HelpOperationSearch,
		}))
	})

	t.Run("localized labels use one language with fallback", func(t *testing.T) {
		previous := i18n.GetLanguage()
		t.Cleanup(func() { i18n.SetLanguage(previous) })

		i18n.SetLanguage("zh")
		assert.Equal(t, "中文", machineHelpLabel("English", "中文"))
		assert.Equal(t, "中文描述", localizedMachineHelpText(machineHelpLocalizedText{EN: "English description", ZH: "中文描述"}))

		i18n.SetLanguage("en")
		assert.Equal(t, "English", machineHelpLabel("English", "中文"))
		assert.Equal(t, "English description", localizedMachineHelpText(machineHelpLocalizedText{EN: "English description", ZH: "中文描述"}))
		assert.Equal(t, "仅中文", localizedMachineHelpText(machineHelpLocalizedText{ZH: "仅中文"}))
	})
}
