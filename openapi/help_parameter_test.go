package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParameterHelpJSONKeepsOnlyParameterContext(t *testing.T) {
	service := testMachineHelpService(t)
	action, err := service.buildAPI("demo", "create-report", "2026-01-01")
	require.NoError(t, err)
	document, err := buildParameterHelpDocument(action, "--report-id")
	require.NoError(t, err)
	attachMachineHelpAIModeHint(document)

	var output bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&output, document, false))
	var raw map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &raw))

	assert.Len(t, raw, 4)
	for _, key := range []string{"schemaVersion", "helpLevel", "parameter", "aiModeHint"} {
		assert.Contains(t, raw, key)
	}
	for _, key := range []string{"kind", "target", "product", "api", "section", "result", "query", "matches"} {
		assert.NotContains(t, raw, key)
	}
	assert.Equal(t, "2026-01-01", document.Target.APIVersion)
}

func TestParameterSearchJSONAddsOnlySearchProjection(t *testing.T) {
	service := testMachineHelpService(t)
	action, err := service.buildAPI("demo", "create-report", "2026-01-01")
	require.NoError(t, err)
	document, err := buildParameterHelpDocument(action, "--report-id")
	require.NoError(t, err)
	searchParameterHelpDocument(document, "report", false)

	var output bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&output, document, false))
	var raw map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &raw))

	for _, key := range []string{"query", "matches", "result"} {
		assert.Contains(t, raw, key)
	}
	for _, key := range []string{"product", "api", "section"} {
		assert.NotContains(t, raw, key)
	}
}

func TestProjectCanonicalParameterRetainsCompleteFiniteTreeAndConstraints(t *testing.T) {
	parameter := canonicalmeta.Parameter{
		Name:       "filters",
		RawName:    "Filters",
		Type:       "array",
		Location:   "query",
		ParamStyle: "json",
		Required:   true,
		Enum:       []string{"active", "inactive"},
		Pattern:    "^[a-z]+$",
		Minimum:    "1",
		Maximum:    "10",
		MinLength:  "1",
		MaxLength:  "16",
		Element: &canonicalmeta.TypeShape{
			Type:   "object",
			Format: "filter",
			Fields: []canonicalmeta.Field{{
				Name:      "resource_id",
				RawName:   "ResourceId",
				Type:      "string",
				Enum:      []string{"one", "two"},
				Pattern:   "^res-",
				MinLength: "4",
			}},
		},
	}

	projected := projectCanonicalParameter(&parameter)
	assert.Equal(t, []string{"active", "inactive"}, projected.Constraints.Enum)
	assert.Equal(t, "^[a-z]+$", projected.Constraints.Pattern)
	assert.Equal(t, "1", projected.Constraints.Minimum)
	assert.Equal(t, "10", projected.Constraints.Maximum)
	require.NotNil(t, projected.Element)
	assert.Equal(t, "filter", projected.Element.Format)
	require.Len(t, projected.Element.Fields, 1)
	assert.Equal(t, []string{"one", "two"}, projected.Element.Fields[0].Constraints.Enum)
	assert.Equal(t, "^res-", projected.Element.Fields[0].Constraints.Pattern)
	assert.Equal(t, "4", projected.Element.Fields[0].Constraints.MinLength)
}

func TestBuildParameterHelpDocumentUsesOnlyCurrentStyleTopLevelFlags(t *testing.T) {
	service := testMachineHelpService(t)
	action, err := service.buildAPI("demo", "create-report", "2026-01-01")
	require.NoError(t, err)
	action.GlobalParameters = []machineHelpParameter{{
		Name: "region", RawName: "region", Options: []string{"--region"}, Location: "global",
	}}

	document, err := buildParameterHelpDocument(action, "--report-id")
	require.NoError(t, err)
	assert.Equal(t, "parameter", document.Kind)
	assert.Equal(t, "report_id", document.Parameter.Name)
	assert.Equal(t, []string{"aliyun", "demo", "create-report", "--report-id"}, document.Target.Path)

	_, err = buildParameterHelpDocument(action, "--ReportId")
	assert.Error(t, err, "inactive camel flags must not resolve in kebab Help")

	global, err := buildParameterHelpDocument(action, "--region")
	require.NoError(t, err)
	assert.Equal(t, "global", global.Parameter.Location)
}

func TestSearchParameterHelpDocumentSearchesNestedFieldsAndCapsResults(t *testing.T) {
	fields := make([]machineHelpParameter, 0, 25)
	for index := 24; index >= 0; index-- {
		fields = append(fields, machineHelpParameter{
			Name:    fmt.Sprintf("Field%02d", index),
			RawName: fmt.Sprintf("Field%02d", index),
			Type:    "string",
		})
	}
	document := &machineHelpParameterDocument{Parameter: machineHelpParameter{
		Name: "config", RawName: "Config", Options: []string{"--config"}, Type: "object", Fields: fields,
	}}

	searchParameterHelpDocument(document, "field", false)
	assert.Equal(t, "field", document.Query)
	require.Len(t, document.Matches, helpSearchResultLimit)
	assert.Equal(t, "Config.Field00", document.Matches[0].Path)
	assert.Equal(t, "Config.Field19", document.Matches[19].Path)
	assert.Equal(t, &HelpResult{Shown: 20, Total: 25, Truncated: true}, document.Result)
	assert.Len(t, document.Parameter.Fields, 25, "L3 source tree remains complete")
	var output bytes.Buffer
	require.NoError(t, renderParameterHelpText(&output, document))
	assert.Contains(t, output.String(), "Matched fields:\n  Config.Field00")
	assert.Contains(t, output.String(), "Showing 20 of 25 matches.")
}

func TestActionParameterSearchIncludesExamplesAndConstraints(t *testing.T) {
	parameter := machineHelpParameter{
		Name:    "filter",
		RawName: "Filter",
		Example: "production-special-token",
		Constraints: machineHelpConstraints{
			Enum:    []string{"enabled-state"},
			Pattern: "resource-prefix",
		},
		Fields: []machineHelpParameter{{
			Name: "nested", RawName: "Nested", Example: "nested-example-token",
		}},
	}
	candidate := machineHelpParameterCandidate(parameter, "parameter")

	for _, query := range []string{"special-token", "enabled-state", "resource-prefix", "nested-example-token"} {
		t.Run(query, func(t *testing.T) {
			assert.Len(t, SearchHelpCandidates([]HelpSearchCandidate{candidate}, query), 1)
		})
	}
}

func TestRenderParameterHelpTextGoldenKeepsNestedTree(t *testing.T) {
	document := &machineHelpParameterDocument{Parameter: machineHelpParameter{
		Name:          "config",
		RawName:       "Config",
		Options:       []string{"--config"},
		Type:          "object",
		Location:      "query",
		Required:      true,
		Serialization: "json",
		Help:          machineHelpLocalizedText{EN: "The configuration."},
		Example:       `{"mode":"safe"}`,
		Fields: []machineHelpParameter{{
			Name: "mode", RawName: "Mode", Type: "string",
			Constraints: machineHelpConstraints{Enum: []string{"safe", "fast"}},
		}},
		Element: &machineHelpShape{
			Type: "string", Format: "uuid",
			Constraints: machineHelpConstraints{Pattern: "^res-", MinLength: "4"},
			Element:     &machineHelpShape{Type: "integer"},
		},
	}}

	var output bytes.Buffer
	require.NoError(t, renderParameterHelpText(&output, document))
	assert.Equal(t, ""+
		"Parameter: --config\n"+
		"  Raw name: Config\n"+
		"  Type: object\n"+
		"  Location: query\n"+
		"  Required: true\n"+
		"  Serialization: json\n"+
		"  Help: The configuration.\n"+
		"  Example: {\"mode\":\"safe\"}\n"+
		"  Fields:\n"+
		"    Config.Mode\n"+
		"      Raw name: Mode\n"+
		"      Type: string\n"+
		"      Required: false\n"+
		"      Enum: safe, fast\n"+
		"  Element:\n"+
		"    Type: string\n"+
		"    Format: uuid\n"+
		"    Pattern: ^res-\n"+
		"    Minimum length: 4\n"+
		"    Element:\n"+
		"      Type: integer\n", output.String())
}
