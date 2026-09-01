package openapi

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type alwaysFailWriter struct{}

func (alwaysFailWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

type machineHelpFDWriter struct {
	bytes.Buffer
	fd uintptr
}

func (w *machineHelpFDWriter) Fd() uintptr { return w.fd }

func TestRenderCanonicalRootTextAllSectionsAndSearch(t *testing.T) {
	document := &machineHelpRootDocument{
		Version:    "9.9.9",
		QuickStart: []string{"aliyun configure", "aliyun ecs DescribeRegions"},
		Commands: []machineHelpCommandSummary{
			{Name: "configure", Path: []string{"aliyun", "configure"}, Description: machineHelpLocalizedText{EN: "Configure credentials"}},
			{Group: "utils", Name: "mcp-proxy", Path: []string{"aliyun", "utils", "mcp-proxy"}, Description: machineHelpLocalizedText{EN: "Run MCP proxy"}},
			{Group: "extension", Name: "completion", Path: []string{"aliyun", "completion"}, Description: machineHelpLocalizedText{EN: "Generate completion"}},
		},
		GlobalFlags: []machineHelpFlagSummary{{Name: "--profile", Shorthand: "-p", Description: machineHelpLocalizedText{EN: "Profile name"}}},
		Products:    []machineHelpProductSummary{{Code: "ecs", Name: machineHelpLocalizedText{EN: "Elastic Compute Service"}}},
		Listing:     &machineHelpListing{Shown: 1, Total: 2, Hint: "list hint"},
		Result:      HelpResult{Shown: 3, Total: 9, Truncated: true},
		Next:        &HelpNext{Search: "aliyun --help-search <keyword>", SearchAll: "aliyun --help-search ecs --help-all", operation: HelpOperationSearch},
	}
	var out strings.Builder
	require.NoError(t, renderCanonicalRootText(&out, document, ""))
	rendered := out.String()
	for _, expected := range []string{"Version 9.9.9", "Quick Start:", "Core Commands:", "Utilities:", "Global Flags:", "Extensions:", "Products:", "Showing 1 of 2 products", "Try another keyword:", "Show all matches:"} {
		assert.Contains(t, rendered, expected)
	}

	search := &machineHelpRootDocument{
		Version: "9.9.9",
		Matches: []machineHelpRootMatch{{Kind: "product", Name: "ecs", Command: "aliyun ecs --help", Description: machineHelpLocalizedText{EN: "Elastic Compute Service"}}},
		Result:  HelpResult{Shown: 1, Total: 2, Truncated: true},
	}
	out.Reset()
	require.NoError(t, renderCanonicalRootText(&out, search, "compute"))
	assert.Contains(t, out.String(), "Matches:")
	assert.Contains(t, out.String(), "aliyun ecs --help")
	assert.Contains(t, out.String(), "Use a more specific --help-search")

	out.Reset()
	require.NoError(t, renderCanonicalRootText(&out, &machineHelpRootDocument{Version: "9.9.9"}, "absent"))
	assert.Contains(t, out.String(), `No Help entries matched --help-search "absent".`)
	assert.Error(t, renderCanonicalRootText(&out, nil, ""))
	assert.Error(t, renderCanonicalRootText(alwaysFailWriter{}, document, ""))
}

func TestRenderCanonicalProductTextFullAndEmpty(t *testing.T) {
	document := &machineHelpProductDocument{
		Product: machineHelpProduct{
			Code: "ecs", Name: machineHelpLocalizedText{EN: "Elastic Compute Service"}, SelectedVersion: "2014-05-26",
			Plugin: "aliyun-cli-ecs", PluginVersion: "1.2.3",
		},
		APIs: []machineHelpAPISummary{
			{DisplayName: "CreateInstance", Title: machineHelpLocalizedText{EN: "Create"}, Description: machineHelpLocalizedText{EN: "Create an instance"}},
			{Name: "DescribeInstances", CmdName: "describe-instances", Description: machineHelpLocalizedText{EN: "List instances"}},
		},
		Listing: &machineHelpListing{Shown: 2, Total: 4, Hint: "use search"},
		Result:  HelpResult{Shown: 2, Total: 5, Truncated: true, OmittedDeprecated: 2},
		Next:    &HelpNext{ShowAll: "aliyun ecs --help-all"},
	}
	var out strings.Builder
	require.NoError(t, renderCanonicalProductText(&out, document, ""))
	rendered := out.String()
	for _, expected := range []string{"Product: ecs", "Available API List:", "Create — Create an instance", "describe-instances", "Omitting 2 deprecated APIs", "Showing 2 of 4 APIs", "Show all:"} {
		assert.Contains(t, rendered, expected)
	}
	assert.NotContains(t, rendered, "Provided by plugin")

	out.Reset()
	require.NoError(t, renderCanonicalProductText(&out, &machineHelpProductDocument{Product: machineHelpProduct{Code: "ecs"}}, "absent"))
	assert.Contains(t, out.String(), `No Help entries matched --help-search "absent".`)
	assert.Error(t, renderCanonicalProductText(&out, nil, ""))
	assert.Error(t, renderCanonicalProductText(alwaysFailWriter{}, document, ""))
}

func TestRenderCanonicalRequestSearchAndFullDocuments(t *testing.T) {
	parameter := machineHelpParameter{Name: "instance-id", Type: "string", Required: true, Options: []string{"--InstanceId"}, Help: machineHelpLocalizedText{EN: "Instance ID"}}
	optional := machineHelpParameter{Name: "page-size", Type: "integer", Options: []string{"--PageSize"}, Help: machineHelpLocalizedText{EN: "Page size"}}
	document := &machineHelpAPIDocument{
		Target:             machineHelpTarget{Path: []string{"aliyun", "ecs", "DescribeInstances"}},
		Product:            machineHelpProduct{Plugin: "aliyun-cli-ecs"},
		ActiveParameterSet: "camel",
		API: machineHelpAPI{
			Title:       machineHelpLocalizedText{EN: "Describe instances"},
			Description: machineHelpLocalizedText{EN: "Returns detailed instance information"},
			Operation:   machineHelpOperation{APIVersion: "2014-05-26"},
		},
		ParameterSets:    machineHelpParameterSets{Camel: []machineHelpParameter{optional, parameter}},
		GlobalParameters: []machineHelpParameter{{Name: "region", Type: "string", Options: []string{"--region"}, Help: machineHelpLocalizedText{EN: "Region"}}},
		Listing:          &machineHelpListing{Shown: 1, Total: 2, Hint: "use all"},
		Result:           HelpResult{Shown: 1, Total: 2, Truncated: true},
		Next:             &HelpNext{Search: "aliyun ecs DescribeInstances --help-search id", operation: HelpOperationAll},
		QueryOptions:     []machineHelpQueryOption{{Name: "--cli-query", Type: "string", Required: true, Help: machineHelpLocalizedText{EN: "JMESPath expression"}}},
		ResponseQuery:    &machineHelpQueryExample{Path: "Instances.Instance", SchemaCommand: "schema command", QueryCommand: "query command"},
		Examples:         machineHelpExamples{Camel: "aliyun ecs DescribeInstances --InstanceId i-1"},
	}

	var out strings.Builder
	require.NoError(t, renderCanonicalRequestSearchText(&out, document, "instance"))
	assert.NotContains(t, out.String(), "Provided by plugin")
	assert.Contains(t, out.String(), "Required Parameters:")
	assert.Contains(t, out.String(), "Optional Parameters:")
	assert.Less(t, strings.Index(out.String(), "--InstanceId"), strings.Index(out.String(), "--PageSize"))
	assert.NotContains(t, out.String(), "(required)")
	assert.NotContains(t, out.String(), "(optional)")
	assert.Contains(t, out.String(), "Global Parameters:")

	out.Reset()
	require.NoError(t, renderCanonicalRequestText(&out, document))
	rendered := out.String()
	for _, expected := range []string{"Description: Describe instances", "Details: Returns detailed instance information", "API Version: 2014-05-26", "Usage:", "Showing 1 of 2 parameters", "Query Options:", "JMESPath expression", "Response aggregation example", "Example:"} {
		assert.Contains(t, rendered, expected)
	}
	assert.Contains(t, rendered, "Global Parameters:")
	assert.Less(t, strings.Index(rendered, "Example:"), strings.Index(rendered, "Search this Help:"))
	assert.True(t, strings.HasSuffix(rendered, "Search this Help:\n  aliyun ecs DescribeInstances --help-search id [--cli-output json]\n"))

	out.Reset()
	require.NoError(t, renderCanonicalRequestSearchText(&out, &machineHelpAPIDocument{}, "absent"))
	assert.Contains(t, out.String(), `No Help entries matched --help-search "absent".`)
	assert.Error(t, renderCanonicalRequestSearchText(&out, nil, ""))
	assert.Error(t, renderCanonicalRequestText(&out, nil))
	assert.Error(t, renderCanonicalRequestSearchText(alwaysFailWriter{}, document, "instance"))
	assert.Error(t, renderCanonicalRequestText(alwaysFailWriter{}, document))
}

func TestHelpProjectionSmallHelpers(t *testing.T) {
	assert.Nil(t, projectMachineHelpListing(nil))
	assert.Equal(t, &machineHelpListing{Shown: 2, Total: 3, Hint: "hint"}, projectMachineHelpListing(&HelpListingMetadata{Shown: 2, Total: 3, Hint: "hint"}))
	assert.Equal(t, ResponseCommandStyleKebab, responseCommandStyle("kebab"))
	assert.Equal(t, ResponseCommandStylePascal, responseCommandStyle("camel"))
	assert.Nil(t, activeMachineHelpParameters(nil))

	matches := []HelpSearchMatch{
		{Candidate: HelpSearchCandidate{Value: "one"}},
		{Candidate: HelpSearchCandidate{Value: 2}},
	}
	assert.Equal(t, []string{"one"}, helpSearchValues[string](matches))

	assert.Equal(t, "plain", stripHelpANSI("\x1b[31mplain\x1b[0m"))
	assert.Equal(t, "before\x1b[", stripHelpANSI("before\x1b["))
	assert.Equal(t, "plugin", machineHelpPluginProvider(machineHelpProduct{Plugin: "plugin"}))
	assert.Empty(t, machineHelpPluginProvider(machineHelpProduct{}))

	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "120")
	assert.Equal(t, 120, machineHelpMaxLineLength(&bytes.Buffer{}))
	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "invalid")
	assert.Equal(t, 80, machineHelpMaxLineLength(&bytes.Buffer{}))
	assert.Nil(t, wrapMachineHelpText("  ", 8))
	assert.Equal(t, []string{"short"}, wrapMachineHelpLine("short", 80))
	assert.Equal(t, 12, len([]rune(wrapMachineHelpText("abcdefghijklmnopqrst", 1)[0])))

	var output bytes.Buffer
	require.NoError(t, renderTextListing(&output, "items", nil))
	require.NoError(t, renderHelpProjectionResult(&output, "items", HelpResult{}, nil))
	require.NoError(t, renderRequestQueryExampleText(&output, nil))
}

func TestMachineHelpMaxLineLengthUsesTerminalWidthWithCap(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "")
	originalIsTerminal, originalGetSize := machineHelpIsTerminal, machineHelpGetSize
	machineHelpIsTerminal = func(fd int) bool { return fd == 42 }
	machineHelpGetSize = func(fd int) (int, int, error) { return 160, 40, nil }
	t.Cleanup(func() {
		machineHelpIsTerminal, machineHelpGetSize = originalIsTerminal, originalGetSize
	})
	assert.Equal(t, 120, machineHelpMaxLineLength(&machineHelpFDWriter{fd: 42}))
	assert.Equal(t, 80, machineHelpMaxLineLength(&bytes.Buffer{}))

	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "60")
	assert.Equal(t, 60, machineHelpMaxLineLength(&machineHelpFDWriter{fd: 42}))
}

func TestCanonicalTextRenderersPropagateFailuresFromLaterWrites(t *testing.T) {
	root := &machineHelpRootDocument{
		Version:    "9.9.9",
		QuickStart: []string{"aliyun configure", "aliyun ecs DescribeRegions"},
		Commands: []machineHelpCommandSummary{
			{Name: "configure", Description: machineHelpLocalizedText{EN: "Configure credentials"}},
			{Group: "utils", Name: "mcp-proxy", Description: machineHelpLocalizedText{EN: "MCP proxy"}},
			{Group: "extension", Name: "completion", Description: machineHelpLocalizedText{EN: "Completion"}},
		},
		GlobalFlags: []machineHelpFlagSummary{{Name: "--profile", Shorthand: "-p", Description: machineHelpLocalizedText{EN: "Profile"}}},
		Products:    []machineHelpProductSummary{{Code: "ecs", Name: machineHelpLocalizedText{EN: "Elastic Compute Service"}}},
		Listing:     &machineHelpListing{Shown: 1, Total: 2, Hint: "hint"},
		Result:      HelpResult{Shown: 1, Total: 2, Truncated: true},
		Next:        &HelpNext{ShowAll: "aliyun --help-all", Search: "aliyun --help-search ecs", SearchAll: "aliyun --help-search ecs --help-all"},
	}
	exerciseWriterFailures(t, func(w *failAfterWrites) error { return renderCanonicalRootText(w, root, "") })

	rootSearch := &machineHelpRootDocument{
		Version: "9.9.9",
		Matches: []machineHelpRootMatch{
			{Kind: "product", Name: "ecs", Command: "aliyun ecs", Description: machineHelpLocalizedText{EN: "Compute"}},
			{Kind: "command", Name: "configure", Command: "aliyun configure", Description: machineHelpLocalizedText{EN: "Configure"}},
		},
		Result: HelpResult{Shown: 2, Total: 3, Truncated: true},
	}
	exerciseWriterFailures(t, func(w *failAfterWrites) error { return renderCanonicalRootText(w, rootSearch, "ec") })

	product := &machineHelpProductDocument{
		Product: machineHelpProduct{Code: "ecs", Name: machineHelpLocalizedText{EN: "Elastic Compute Service"}, SelectedVersion: "v1", Plugin: "plugin", PluginVersion: "1.0"},
		APIs: []machineHelpAPISummary{
			{DisplayName: "CreateInstance", Description: machineHelpLocalizedText{EN: "Create"}},
			{DisplayName: "DescribeInstances", Description: machineHelpLocalizedText{EN: "Describe"}},
		},
		Listing: &machineHelpListing{Shown: 2, Total: 3, Hint: "hint"},
		Result:  HelpResult{Shown: 2, Total: 4, Truncated: true, OmittedDeprecated: 1},
		Next:    &HelpNext{ShowAll: "aliyun ecs --help-all", Search: "aliyun ecs --help-search instance"},
	}
	exerciseWriterFailures(t, func(w *failAfterWrites) error { return renderCanonicalProductText(w, product, "") })

	parameter := machineHelpParameter{
		Name: "instance-id", RawName: "InstanceId", Options: []string{"--InstanceId"},
		Type: "string", Location: "query", Required: true,
		Help: machineHelpLocalizedText{EN: "Instance ID"}, Example: "i-1",
	}
	request := &machineHelpAPIDocument{
		Target:             machineHelpTarget{Path: []string{"aliyun", "ecs", "DescribeInstances"}},
		Product:            machineHelpProduct{Plugin: "plugin"},
		ActiveParameterSet: "camel",
		API: machineHelpAPI{
			Title: machineHelpLocalizedText{EN: "Describe instances"}, Description: machineHelpLocalizedText{EN: "Details"},
			Operation: machineHelpOperation{APIVersion: "v1"},
		},
		ParameterSets:    machineHelpParameterSets{Camel: []machineHelpParameter{parameter}},
		GlobalParameters: []machineHelpParameter{{Name: "region", Options: []string{"--region"}, Type: "string", Help: machineHelpLocalizedText{EN: "Region"}}},
		QueryOptions:     []machineHelpQueryOption{{Name: "--cli-query", Type: "string", Help: machineHelpLocalizedText{EN: "Query"}}},
		ResponseQuery:    &machineHelpQueryExample{Path: "Instances", SchemaCommand: "schema", QueryCommand: "query"},
		Examples:         machineHelpExamples{Camel: "aliyun ecs DescribeInstances"},
		Listing:          &machineHelpListing{Shown: 1, Total: 2, Hint: "hint"},
		Result:           HelpResult{Shown: 1, Total: 2, Truncated: true},
		Next:             &HelpNext{ShowAll: "aliyun ecs DescribeInstances --help-all", Search: "aliyun ecs DescribeInstances --help-search id"},
	}
	exerciseWriterFailures(t, func(w *failAfterWrites) error { return renderCanonicalRequestSearchText(w, request, "id") })
	exerciseWriterFailures(t, func(w *failAfterWrites) error { return renderCanonicalRequestText(w, request) })
}
