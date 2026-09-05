package openapi

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setMachineHelpTestLanguage(t *testing.T, lang string) {
	t.Helper()
	previous := i18n.GetLanguage()
	i18n.SetLanguage(lang)
	t.Cleanup(func() { i18n.SetLanguage(previous) })
}

func TestEncodeMachineHelpJSONCompactEmitsSingleLine(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildProduct("demo", "")
	require.NoError(t, err)

	var compact bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&compact, doc, true))
	require.NoError(t, json.Unmarshal(compact.Bytes(), &map[string]any{}))
	assert.NotContains(t, compact.String(), "\n  ")
	assert.Equal(t, 1, strings.Count(compact.String(), "\n"), "compact output keeps only the trailing newline")

	var pretty bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&pretty, doc, false))
	assert.Contains(t, pretty.String(), "\n  ")
}

func TestMachineHelpJSONCollapsesLocalizedTextPerLanguage(t *testing.T) {
	doc := &machineHelpProductDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Kind:          "product",
		Product: machineHelpProduct{
			Code:            "demo",
			Name:            machineHelpLocalizedText{EN: "Demo Service", ZH: "演示服务"},
			SelectedVersion: "2026-01-01",
		},
		APIs: []machineHelpAPISummary{
			{DisplayName: "CreateThing", Title: machineHelpLocalizedText{EN: "Create", ZH: "创建"}},
			{DisplayName: "DeleteThing"},
		},
		Result: HelpResult{Shown: 2, Total: 2},
	}

	encode := func() map[string]any {
		var output bytes.Buffer
		require.NoError(t, encodeMachineHelpJSON(&output, doc, true))
		var raw map[string]any
		require.NoError(t, json.Unmarshal(output.Bytes(), &raw))
		return raw
	}

	setMachineHelpTestLanguage(t, "en")
	raw := encode()
	assert.Equal(t, "Demo Service", raw["product"].(map[string]any)["name"])
	assert.Equal(t, "Create", raw["apis"].([]any)[0].(map[string]any)["title"])
	assert.NotContains(t, outputLocalizedKeys(raw), "en")

	setMachineHelpTestLanguage(t, "zh")
	raw = encode()
	assert.Equal(t, "演示服务", raw["product"].(map[string]any)["name"])
	assert.Equal(t, "创建", raw["apis"].([]any)[0].(map[string]any)["title"])
	assert.NotContains(t, outputLocalizedKeys(raw), "zh")
}

func outputLocalizedKeys(value any) []string {
	var keys []string
	collectLocalizedKeys(value, &keys)
	return keys
}

func collectLocalizedKeys(value any, keys *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if key == "en" || key == "zh" {
				*keys = append(*keys, key)
			}
			collectLocalizedKeys(item, keys)
		}
	case []any:
		for _, item := range typed {
			collectLocalizedKeys(item, keys)
		}
	}
}

func TestMachineHelpJSONOmitsDefaultAnnotationsAndRepeatedIdentity(t *testing.T) {
	doc := &machineHelpProductDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Kind:          "product",
		Product:       machineHelpProduct{Code: "demo", SelectedVersion: "2026-01-01"},
		APIs: []machineHelpAPISummary{
			{Name: "CreateThing", DisplayName: "CreateThing"},
			{Name: "DeleteThing", DisplayName: "DeleteThing", Deprecated: true},
		},
		Result: HelpResult{Shown: 2, Total: 2, Truncated: false},
	}

	var output bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&output, doc, false))
	assert.NotContains(t, output.String(), "displayName")

	var raw map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &raw))
	first := raw["apis"].([]any)[0].(map[string]any)
	second := raw["apis"].([]any)[1].(map[string]any)
	assert.NotContains(t, first, "deprecated")
	assert.Equal(t, true, second["deprecated"])
	result := raw["result"].(map[string]any)
	assert.Equal(t, false, result["truncated"], "decision-relevant booleans stay explicit")
}

func TestMachineHelpSearchJSONOmitsProductContext(t *testing.T) {
	product := machineHelpProduct{
		Code:                 "demo",
		Name:                 machineHelpLocalizedText{EN: "Demo Service"},
		APIStyle:             "rpc",
		LegacyDefaultVersion: "2026-01-01",
		SelectedVersion:      "2026-01-01",
	}
	documents := []any{
		&machineHelpProductDocument{
			SchemaVersion: machineHelpSchemaVersion,
			Kind:          "product",
			Query:         "report",
			Product:       product,
			APIs:          []machineHelpAPISummary{{DisplayName: "CreateReport"}},
			Result:        HelpResult{Shown: 1, Total: 1},
		},
		&machineHelpAPIDocument{
			SchemaVersion:      machineHelpSchemaVersion,
			Kind:               "api",
			Section:            helpSectionRequest,
			Query:              "report-id",
			Product:            product,
			ActiveParameterSet: "camel",
			ParameterSets: machineHelpParameterSets{Camel: []machineHelpParameter{{
				Name: "ReportId", Type: "string",
			}}},
			Result: HelpResult{Shown: 1, Total: 1},
		},
	}

	for _, document := range documents {
		var output bytes.Buffer
		require.NoError(t, encodeMachineHelpJSON(&output, document, false))
		var raw map[string]any
		require.NoError(t, json.Unmarshal(output.Bytes(), &raw))
		assert.NotContains(t, raw, "product")
	}
}

func TestMachineHelpJSONKeepsProductContextOutsideSearch(t *testing.T) {
	documents := []any{
		&machineHelpProductDocument{
			SchemaVersion: machineHelpSchemaVersion,
			Kind:          "product",
			Product:       machineHelpProduct{Code: "demo"},
		},
		&machineHelpAPIDocument{
			SchemaVersion: machineHelpSchemaVersion,
			Kind:          "api",
			Section:       helpSectionRequest,
			Product:       machineHelpProduct{Code: "demo"},
		},
	}

	for _, document := range documents {
		var output bytes.Buffer
		require.NoError(t, encodeMachineHelpJSON(&output, document, false))
		var raw map[string]any
		require.NoError(t, json.Unmarshal(output.Bytes(), &raw))
		assert.Contains(t, raw, "product")
	}
}

func TestMachineHelpJSONOmitsDerivableOptionsAndRawName(t *testing.T) {
	doc := &machineHelpAPIDocument{
		SchemaVersion:      machineHelpSchemaVersion,
		Kind:               "api",
		Section:            helpSectionRequest,
		ActiveParameterSet: "camel",
		ParameterSets: machineHelpParameterSets{Camel: []machineHelpParameter{
			{
				Name:     "RegionId",
				RawName:  "RegionId",
				Options:  []string{"--RegionId"},
				Type:     "string",
				Location: "query",
				Required: true,
				Help:     machineHelpLocalizedText{EN: "Region ID"},
			},
			{
				Name:     "Tags",
				Options:  []string{"--Tags"},
				Type:     "array",
				Location: "query",
				Required: false,
				Fields: []machineHelpParameter{{
					Name:    "Key",
					Options: []string{"--Tags.#.Key"},
					Type:    "string",
					Help:    machineHelpLocalizedText{EN: "Tag key"},
				}},
			},
		}},
	}

	var output bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&output, doc, false))
	assert.NotContains(t, output.String(), "rawName")

	var raw map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &raw))
	parameters := raw["parameterSets"].(map[string]any)["camel"].([]any)
	region := parameters[0].(map[string]any)
	assert.NotContains(t, region, "options", "options repeating --+name is derivable")
	tags := parameters[1].(map[string]any)
	assert.NotContains(t, tags, "options")
	nested := tags["fields"].([]any)[0].(map[string]any)
	assert.Equal(t, []any{"--Tags.#.Key"}, nested["options"], "nested flag paths are not derivable and stay")
}

func TestMachineHelpJSONKeepsKebabOptionsWhenNameSpellingDiffers(t *testing.T) {
	doc := &machineHelpAPIDocument{
		SchemaVersion:      machineHelpSchemaVersion,
		Kind:               "api",
		Section:            helpSectionRequest,
		ActiveParameterSet: "kebab",
		ParameterSets: machineHelpParameterSets{Kebab: []machineHelpParameter{
			{Name: "report_id", Options: []string{"--report-id"}, Type: "string", Required: true},
		}},
	}

	var output bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&output, doc, true))
	var raw map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &raw))
	parameter := raw["parameterSets"].(map[string]any)["kebab"].([]any)[0].(map[string]any)
	assert.Equal(t, []any{"--report-id"}, parameter["options"], "snake_case names cannot derive the kebab flag")
}

func TestMachineHelpJSONLeavesResponseSchemaUserValuesIntact(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","default":{"en":"hello","zh":"你好"},"description_en":"greeting","description_zh":"问候"}`)
	doc := &machineHelpAPIResponseDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Kind:          "api",
		Section:       helpSectionResponse,
		OutputSchema: &machineHelpOutputSchema{
			StatusCode:  "200",
			ContentType: "application/json",
			Schema:      localizeHelpJSON(schema, "en"),
		},
	}

	var output bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&output, doc, false))
	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, output.Bytes()))
	assert.Contains(t, compact.String(), `"default":{"en":"hello","zh":"你好"}`)
}

func TestCommandoHelpJSONAIModeEmitsCompactOutput(t *testing.T) {
	t.Setenv(aimode.EnvAIMode, "1")
	c, stdout, stderr := newTestCommando()
	c.library.helpRepo = canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	c.library.baselineHelpRepo = c.library.helpRepo
	root := testMachineHelpRootCommand()
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	VersionFlag(ctx.Flags()).SetAssigned(true)
	VersionFlag(ctx.Flags()).SetValue("2026-01-01")

	require.NoError(t, c.help(ctx, []string{"demo", "CreateReport"}))
	assert.Empty(t, stderr.String())

	var doc machineHelpAPIDocument
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc))
	assert.NotContains(t, stdout.String(), "\n  ", "AI mode help is compact JSON")
	assert.NotContains(t, stdout.String(), "aiModeHint")
}
