package openapi

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildProductForStyleUsesVersionIndexAndStyleDisplayNames(t *testing.T) {
	service := testMachineHelpService(t)

	camel, err := service.buildProductForStyle("demo", "", "camel")
	require.NoError(t, err)
	assert.Equal(t, "2026-01-01", camel.Product.SelectedVersion)
	assert.Equal(t, "camel", camel.Target.RequestedStyle)
	require.Len(t, camel.APIs, 2)
	assert.Equal(t, "CreateReport", camel.APIs[0].DisplayName)
	assert.Equal(t, "DescribeRegions", camel.APIs[1].DisplayName)
	var camelText bytes.Buffer
	require.NoError(t, renderCanonicalProductText(&camelText, camel, ""))
	assert.Contains(t, camelText.String(), "CreateReport")
	assert.NotContains(t, camelText.String(), "create-report")

	kebab, err := service.buildProductForStyle("demo", "", "kebab")
	require.NoError(t, err)
	assert.Equal(t, "2025-01-01", kebab.Product.SelectedVersion)
	assert.Equal(t, "kebab", kebab.Target.RequestedStyle)
	require.Len(t, kebab.APIs, 1)
	assert.Equal(t, "describe-regions", kebab.APIs[0].DisplayName)
}

func TestProductDefaultAllAndSearchProjectionSemantics(t *testing.T) {
	newDocument := func() *machineHelpProductDocument {
		apis := make([]machineHelpAPISummary, 0, 105)
		for index := 104; index >= 0; index-- {
			apis = append(apis, machineHelpAPISummary{
				Name:        fmt.Sprintf("DescribeItem%03d", index),
				CmdName:     fmt.Sprintf("describe-item-%03d", index),
				DisplayName: fmt.Sprintf("DescribeItem%03d", index),
				Title:       machineHelpLocalizedText{EN: "Item title"},
				Description: machineHelpLocalizedText{EN: "Complete item description"},
			})
		}
		return &machineHelpProductDocument{Product: machineHelpProduct{Code: "demo"}, APIs: apis}
	}

	defaultDocument := newDocument()
	applyProductHelpOptions(defaultDocument, helpOptions{}, true)
	wantShown := defaultHelpLogicalLineBudget - productDefaultHelpReservedLines
	assert.Len(t, defaultDocument.APIs, wantShown)
	assert.Equal(t, "DescribeItem000", defaultDocument.APIs[0].DisplayName)
	assert.Empty(t, defaultDocument.APIs[0].Title.EN)
	assert.Empty(t, defaultDocument.APIs[0].Description.EN)
	assert.Equal(t, HelpResult{Shown: wantShown, Total: 105, Truncated: true}, defaultDocument.Result)
	var defaultText bytes.Buffer
	require.NoError(t, renderCanonicalProductText(&defaultText, defaultDocument, ""))
	assert.Contains(t, defaultText.String(), fmt.Sprintf("...\nShowing %d of 105 APIs.", wantShown))

	allDocument := newDocument()
	applyProductHelpOptions(allDocument, helpOptions{All: true}, true)
	assert.Len(t, allDocument.APIs, 105)
	assert.Equal(t, "Complete item description", allDocument.APIs[0].Description.EN)
	assert.Equal(t, HelpResult{Shown: 105, Total: 105}, allDocument.Result)

	searchDocument := newDocument()
	applyProductHelpOptions(searchDocument, helpOptions{Search: "item"}, true)
	assert.Equal(t, "item", searchDocument.Query)
	assert.Len(t, searchDocument.APIs, helpSearchResultLimit)
	assert.Equal(t, "Complete item description", searchDocument.APIs[0].Description.EN)
	assert.Equal(t, HelpResult{Shown: 20, Total: 105, Truncated: true}, searchDocument.Result)
}

func TestRootSearchIncludesLocalCommandsAndProductsThenCapsGlobally(t *testing.T) {
	document := &machineHelpRootDocument{
		Commands: []machineHelpCommandSummary{{
			Group: "core", Path: []string{"aliyun", "configure"}, Name: "configure", Aliases: []string{"setup"},
			Description: machineHelpLocalizedText{EN: "Configure credentials"},
		}},
		Products: []machineHelpProductSummary{{Code: "ecs", Name: machineHelpLocalizedText{EN: "Elastic Compute Service"}}},
	}
	applyRootHelpOptions(document, helpOptions{Search: "configure"}, true)
	assert.Equal(t, "configure", document.Query)
	require.Len(t, document.Commands, 1)
	assert.Empty(t, document.Products)
	assert.Equal(t, HelpResult{Shown: 1, Total: 1}, document.Result)

	aliasDocument := &machineHelpRootDocument{Commands: []machineHelpCommandSummary{{
		Name: "configure", Aliases: []string{"setup"},
	}}}
	applyRootHelpOptions(aliasDocument, helpOptions{Search: "setup"}, true)
	require.Len(t, aliasDocument.Commands, 1)

	renderDocument := &machineHelpRootDocument{Commands: []machineHelpCommandSummary{
		{Group: "core", Path: []string{"aliyun", "configure"}, Name: "configure"},
		{Group: "utils", Path: []string{"aliyun", "utils", "mcp-proxy"}, Name: "mcp-proxy"},
	}}
	var rendered bytes.Buffer
	require.NoError(t, renderCanonicalRootText(&rendered, renderDocument, ""))
	assert.Contains(t, rendered.String(), "Core Commands:")
	assert.Contains(t, rendered.String(), "Utilities:")
	assert.Contains(t, rendered.String(), "utils mcp-proxy")

	large := &machineHelpRootDocument{Commands: []machineHelpCommandSummary{{Name: "item-command"}}}
	for index := 104; index >= 0; index-- {
		large.Products = append(large.Products, machineHelpProductSummary{Code: fmt.Sprintf("item-%03d", index)})
	}
	applyRootHelpOptions(large, helpOptions{Search: "item"}, true)
	assert.Len(t, large.Commands, 0)
	assert.Len(t, large.Products, 20)
	assert.Equal(t, "item-000", large.Products[0].Code)
	assert.Equal(t, HelpResult{Shown: 20, Total: 106, Truncated: true}, large.Result)
}

func TestActionDefaultKeepsAllRequiredAndBudgetedOptionalParameters(t *testing.T) {
	required := make([]machineHelpParameter, 0, 4)
	for index := 0; index < 4; index++ {
		required = append(required, machineHelpParameter{
			Name: fmt.Sprintf("required-%d", index), Required: true,
			Fields: []machineHelpParameter{{Name: "one"}, {Name: "two"}, {Name: "three"}},
		})
	}
	optional := make([]machineHelpParameter, 0, 30)
	for index := 0; index < 30; index++ {
		optional = append(optional, machineHelpParameter{
			Name:   fmt.Sprintf("optional-%02d", index),
			Fields: []machineHelpParameter{{Name: "one"}, {Name: "two"}, {Name: "three"}},
		})
	}
	document := &machineHelpAPIDocument{
		Product:            machineHelpProduct{APIStyle: "RPC"},
		API:                machineHelpAPI{Operation: machineHelpOperation{Method: "POST", Protocol: "https", URL: "/"}},
		ActiveParameterSet: "camel",
		ParameterSets:      machineHelpParameterSets{Camel: append(required, optional...)},
		GlobalParameters:   []machineHelpParameter{{Name: "region", Location: "global"}},
	}

	applyActionHelpOptions(document, helpOptions{}, true, true)
	parameters := activeMachineHelpParameters(document)
	require.NotEmpty(t, parameters)
	for index := range required {
		assert.Equal(t, required[index].Name, parameters[index].Name)
	}
	assert.Less(t, len(parameters), len(required)+len(optional))
	assert.Empty(t, document.GlobalParameters, "default Action Help does not repeat globals")
	assert.True(t, document.Result.Truncated)
	assert.Empty(t, document.API.Operation.Method)
	assert.Empty(t, document.API.Operation.URL)

	roa := &machineHelpAPIDocument{
		Product:            machineHelpProduct{APIStyle: "ROA"},
		API:                machineHelpAPI{Operation: machineHelpOperation{Method: "GET", URL: "/instances/{id}"}},
		ActiveParameterSet: "camel",
	}
	applyActionHelpOptions(roa, helpOptions{}, true, true)
	assert.Equal(t, "GET", roa.API.Operation.Method)
	assert.Equal(t, "/instances/{id}", roa.API.Operation.URL)
}

func TestMachineHelpResponseDocumentRetainsFullResponsesAndReachableComponents(t *testing.T) {
	service := testMachineHelpService(t)
	document, err := service.buildAPIResponse("demo", "CreateReport", "2026-01-01")
	require.NoError(t, err)

	require.NotEmpty(t, document.Responses)
	require.NotNil(t, document.Components)
	assert.Contains(t, document.Components.Schemas, "ReportList")
	assert.Contains(t, document.Components.Schemas, "Report")
	assert.NotContains(t, document.Components.Schemas, "Unused")
	assert.NotNil(t, document.OutputSchema, "selected success schema remains available for query guidance")
}

func TestMachineHelpPurposePrefersTitleAndDeduplicatesTextOnly(t *testing.T) {
	title := machineHelpLocalizedText{EN: "Short purpose", ZH: "简短用途"}
	description := machineHelpLocalizedText{EN: "Long description", ZH: "完整说明"}
	assert.Equal(t, "Short purpose", localizedMachineHelpPurpose(title, description))

	assert.Equal(t, "Long description", localizedMachineHelpPurpose(machineHelpLocalizedText{}, description))
	assert.Equal(t, "Short purpose", title.EN, "structured fields are not mutated by text fallback")

	document := &machineHelpAPIDocument{
		Target: machineHelpTarget{Path: []string{"aliyun", "demo", "DescribeThings"}},
		API: machineHelpAPI{
			Title: title, Description: description,
			Operation: machineHelpOperation{APIVersion: "2026-01-01"},
		},
	}
	var output bytes.Buffer
	require.NoError(t, renderCanonicalRequestText(&output, document))
	assert.Contains(t, output.String(), "Description: Short purpose")
	assert.Contains(t, output.String(), "Details: Long description")
}
