package openapi

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectOriginalRequestHelpTextPreservesLayoutWhileCappingAndSearching(t *testing.T) {
	var source strings.Builder
	source.WriteString("Description: original renderer\n\nParameters:\n")
	for index := 1; index <= 23; index++ {
		fmt.Fprintf(&source, "  --parameter-%02d              string, help %02d\n", index, index)
		if index == 2 {
			source.WriteString("                              continuation stays aligned\n")
		}
	}
	source.WriteString("\nGlobal Parameters:\n  --region                      string, original global help\n\nExamples:\n  aliyun demo list-items\n")

	t.Run("AI mode caps only API parameters", func(t *testing.T) {
		projected, listing := projectOriginalRequestHelpText(source.String(), helpOptions{}, true)

		require.NotNil(t, listing)
		assert.Equal(t, 20, listing.Shown)
		assert.Equal(t, 23, listing.Total)
		assert.Contains(t, projected, "Description: original renderer")
		assert.Contains(t, projected, "--parameter-20")
		assert.NotContains(t, projected, "--parameter-21")
		assert.Contains(t, projected, "--region                      string, original global help")
		assert.Contains(t, projected, "Examples:\n  aliyun demo list-items")
		assert.Contains(t, projected, "Showing 20 of 23 parameters.")
	})

	t.Run("search keeps original matching blocks and all globals", func(t *testing.T) {
		projected, listing := projectOriginalRequestHelpText(source.String(), helpOptions{Search: "parameter-02"}, true)

		assert.Nil(t, listing)
		assert.Contains(t, projected, "--parameter-02")
		assert.Contains(t, projected, "continuation stays aligned")
		assert.NotContains(t, projected, "--parameter-01")
		assert.NotContains(t, projected, "--parameter-03")
		assert.Contains(t, projected, "--region                      string, original global help")
		assert.NotContains(t, projected, "Showing ")
	})
}

func TestValidateRecoverySearchUsesRealCanonicalHelpProvider(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)

	tests := []struct {
		name    string
		request RecoverySearchRequest
		want    bool
	}{
		{name: "root product", request: RecoverySearchRequest{Keyword: "演示"}, want: true},
		{
			name:    "Pascal API uses legacy default version",
			request: RecoverySearchRequest{Product: "demo", Style: "pascal", Keyword: "report"},
			want:    true,
		},
		{
			name:    "kebab API uses plugin default version",
			request: RecoverySearchRequest{Product: "demo", Style: "kebab", Keyword: "regions"},
			want:    true,
		},
		{
			name:    "request parameter",
			request: RecoverySearchRequest{Product: "demo", API: "create-report", Version: "2026-01-01", Section: "request", Keyword: "workspace-id"},
			want:    true,
		},
		{
			name:    "global parameter",
			request: RecoverySearchRequest{Product: "demo", API: "create-report", Version: "2026-01-01", Section: "request", Keyword: "header"},
			want:    true,
		},
		{
			name:    "response field",
			request: RecoverySearchRequest{Product: "demo", API: "CreateReport", Version: "2026-01-01", Section: "response", Keyword: "report-id"},
			want:    true,
		},
		{
			name:    "no result",
			request: RecoverySearchRequest{Product: "demo", API: "CreateReport", Version: "2026-01-01", Section: "response", Keyword: "not-present"},
			want:    false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, c.validateRecoverySearch(ctx, test.request))
		})
	}
}

func TestValidateRecoverySearchAllowsMetaPluginProvider(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	c.localManifest = &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-demo": {Name: "aliyun-cli-demo", Type: plugin.PluginTypeMeta},
	}}

	// Metadata plugins are served by host Machine Help, so search validation
	// must keep working instead of refusing the provider.
	assert.True(t, c.validateRecoverySearch(ctx, RecoverySearchRequest{
		Product: "demo", Style: "pascal", Keyword: "report",
	}))
}

func TestValidateRecoverySearchRefusesGoPluginTextProvider(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	c.localManifest = &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-demo": {Name: "aliyun-cli-demo", Type: plugin.PluginTypeGo},
	}}

	assert.False(t, c.validateRecoverySearch(ctx, RecoverySearchRequest{
		Product: "demo", Style: "pascal", Keyword: "report",
	}))
}

func TestMachineHelpAIModeHintFollowsEffectiveMode(t *testing.T) {
	for _, test := range []struct {
		name string
		env  string
		want bool
	}{
		{name: "off", env: "0", want: true},
		{name: "on", env: "1", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
			t.Setenv(aimode.EnvAIMode, test.env)
			c.library.helpRepo = canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
			MachineHelpFormatFlag(ctx.Flags()).SetAssigned(true)
			MachineHelpFormatFlag(ctx.Flags()).SetValue("json")

			assert.NoError(t, c.help(ctx, nil))
			assert.Equal(t, test.want, containsJSONField(stdout.Bytes(), "aiModeHint"))
		})
	}
}

func containsJSONField(data []byte, field string) bool {
	return strings.Contains(string(data), `"`+field+`"`)
}

func TestRenderMachineHelpParametersIndentsMultilineHelp(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "100")
	parameters := []machineHelpParameter{
		{
			Name:    "page-size",
			Options: []string{"--page-size"},
			Type:    "int",
			Help:    machineHelpLocalizedText{EN: "The page size."},
		},
		{
			Name:     "accept-language",
			Options:  []string{"--accept-language"},
			Type:     "string",
			Required: true,
			Help: machineHelpLocalizedText{
				EN: "The language of the response. Valid values:\n\nzh-CN: Chinese.\nen-US (default): English",
			},
		},
	}
	var out strings.Builder
	require.NoError(t, renderMachineHelpParameters(&out, parameters))

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	require.Len(t, lines, 7)
	inline := fmt.Sprintf("  %-30s %s  %s", "--page-size", "int", "The page size.")
	assert.Equal(t, inline, lines[0])
	assert.Equal(t, fmt.Sprintf("  %-30s %s", "--accept-language", "string"), lines[1])
	indent := strings.Repeat(" ", 2+machineHelpParameterNameWidth+1)
	assert.Equal(t, indent+"The language of the response.", lines[2])
	assert.Equal(t, indent+"Valid values:", lines[3])
	assert.Empty(t, lines[4])
	assert.Equal(t, indent+"zh-CN: Chinese.", lines[5])
	assert.Equal(t, indent+"en-US (default): English", lines[6])
}

func TestWrapMachineHelpTextPreservesParagraphsMarkersAndURLs(t *testing.T) {
	lines := wrapMachineHelpText("first paragraph\n\nsecond paragraph that is long enough to require wrapping at the configured width\n\n\nthird", 30)
	require.NotEmpty(t, lines)
	assert.Equal(t, "first paragraph", lines[0])
	assert.Empty(t, lines[1])
	assert.Equal(t, "third", lines[len(lines)-1])
	for _, line := range lines {
		if line != "" {
			assert.LessOrEqual(t, len([]rune(line)), 30)
		}
	}
	assert.Equal(t, []string{"Description before marker.", "Valid values:", "one, two"},
		wrapMachineHelpText("Description before marker. Valid values: one, two", 40))
	assert.Equal(t, []string{"Valid values:", "one, two"}, wrapMachineHelpText("Valid values: one, two", 40))
	url := "https://example.com/a/very/long/path/that/must/not/break"
	assert.Equal(t, []string{"See", url, "for details."}, wrapMachineHelpText("See "+url+" for details.", 20))
	assert.Equal(t, []string{"请使用", url, "接口查看详情。"}, wrapMachineHelpText("请使用"+url+"接口查看详情。", 20))
}

func TestAnnotatePluginProvenanceMarksMetaPluginProducts(t *testing.T) {
	c, _, _ := newTestCommando()
	c.localManifest = &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-demo": {Name: "aliyun-cli-demo", Version: "1.2.3", Type: plugin.PluginTypeMeta},
	}}

	productDoc := &machineHelpProductDocument{}
	c.annotatePluginProvenance(productDoc, "demo")
	assert.Equal(t, "aliyun-cli-demo", productDoc.Product.Plugin)
	assert.Equal(t, "1.2.3", productDoc.Product.PluginVersion)

	apiDoc := &machineHelpAPIDocument{}
	c.annotatePluginProvenance(apiDoc, "DEMO")
	assert.Equal(t, "aliyun-cli-demo", apiDoc.Product.Plugin)

	responseDoc := &machineHelpAPIResponseDocument{}
	c.annotatePluginProvenance(responseDoc, "demo")
	assert.Equal(t, "aliyun-cli-demo", responseDoc.Provider)
}

func TestAnnotatePluginProvenanceSkipsGoPluginsAndUnknownProducts(t *testing.T) {
	c, _, _ := newTestCommando()
	c.localManifest = &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-demo": {Name: "aliyun-cli-demo", Type: plugin.PluginTypeGo},
	}}

	doc := &machineHelpProductDocument{}
	c.annotatePluginProvenance(doc, "demo")
	assert.Empty(t, doc.Product.Plugin)

	other := &machineHelpProductDocument{}
	c.annotatePluginProvenance(other, "not-installed")
	assert.Empty(t, other.Product.Plugin)
}

func TestRenderCanonicalRequestTextShowsQueryOptions(t *testing.T) {
	document := &machineHelpAPIDocument{
		ActiveParameterSet: "camel",
		Target:             machineHelpTarget{Path: []string{"aliyun", "ecs", "DescribeSpotPriceHistory"}},
		API:                machineHelpAPI{Operation: machineHelpOperation{APIVersion: "2014-05-26"}},
		QueryOptions:       buildMachineHelpQueryOptions(),
		Examples:           machineHelpExamples{Camel: "aliyun ecs DescribeSpotPriceHistory --RegionId cn-hangzhou"},
		ResponseQuery: &machineHelpQueryExample{
			Path:          "SpotPrices.SpotPriceType",
			SchemaCommand: "aliyun help ecs DescribeSpotPriceHistory --cli-section response",
			QueryCommand:  "aliyun ecs DescribeSpotPriceHistory --cli-query 'SpotPrices.SpotPriceType'",
		},
	}
	var out strings.Builder
	require.NoError(t, renderCanonicalRequestText(&out, document))

	rendered := out.String()
	assert.Contains(t, rendered, "\nQuery Options:\n")
	assert.Contains(t, rendered, "--cli-section                  string (optional), default: request")
	assert.Contains(t, rendered, "--cli-query                    string (optional)")
	assert.Contains(t, rendered, "--help-search                  string (optional)")
	assert.Contains(t, rendered, "Response aggregation example (JMESPath: SpotPrices.SpotPriceType):")
	assert.Contains(t, rendered, "1. Inspect the response structure to pick the fields you need:")
	assert.Contains(t, rendered, "2. Then get only those fields, at any level of the response, in one call:")
	assert.Contains(t, rendered, "  aliyun help ecs DescribeSpotPriceHistory --cli-section response")
	assert.Contains(t, rendered, "  aliyun ecs DescribeSpotPriceHistory --cli-query 'SpotPrices.SpotPriceType'")
	assert.NotContains(t, rendered, "Response query example (")
	// Query Options precedes Example
	assert.Less(t, strings.Index(rendered, "Query Options:"), strings.Index(rendered, "Example:"))
}

func TestRenderCanonicalRequestTextQueryOptionsWithoutSchema(t *testing.T) {
	document := &machineHelpAPIDocument{
		ActiveParameterSet: "kebab",
		Target:             machineHelpTarget{Path: []string{"aliyun", "ecs", "describe-regions"}},
		API:                machineHelpAPI{Operation: machineHelpOperation{APIVersion: "2014-05-26"}},
		QueryOptions:       buildMachineHelpQueryOptions(),
	}
	var out strings.Builder
	require.NoError(t, renderCanonicalRequestText(&out, document))

	assert.Contains(t, out.String(), "\nQuery Options:\n")
	assert.NotContains(t, out.String(), "complex array")
}

func TestValidateRecoverySearchMirrorsAIModeGlobals(t *testing.T) {
	// Non-AI replay keeps global CLI flags searchable...
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	assert.True(t, c.validateRecoverySearch(ctx, RecoverySearchRequest{
		Product: "demo", API: "create-report", Version: "2026-01-01", Section: "request", Keyword: "header",
	}))

	// ...while AI mode drops them, so a global-only keyword must not validate
	// against help that would render empty.
	c2, ctx2, _, _ := newCanonicalHelpTestContext(t)
	t.Setenv(aimode.EnvAIMode, "1")
	assert.False(t, c2.validateRecoverySearch(ctx2, RecoverySearchRequest{
		Product: "demo", API: "create-report", Version: "2026-01-01", Section: "request", Keyword: "header",
	}))
	// API parameters remain searchable in AI mode.
	assert.True(t, c2.validateRecoverySearch(ctx2, RecoverySearchRequest{
		Product: "demo", API: "create-report", Version: "2026-01-01", Section: "request", Keyword: "workspace-id",
	}))
}

func productDocWithDeprecatedAPIs() *machineHelpProductDocument {
	return &machineHelpProductDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Kind:          "product",
		Product:       machineHelpProduct{Code: "demo"},
		APIs: []machineHelpAPISummary{
			{Name: "AlphaActive"},
			{Name: "BravoLegacy", Deprecated: true},
			{Name: "CharlieActive"},
			{Name: "DeltaLegacy", Deprecated: true},
			{Name: "EchoActive"},
		},
	}
}

func TestProductHelpDefaultHidesDeprecatedAPIs(t *testing.T) {
	for _, aiMode := range []bool{false, true} {
		document := productDocWithDeprecatedAPIs()
		applyProductHelpOptions(document, helpOptions{}, aiMode)

		names := make([]string, 0, len(document.APIs))
		for _, api := range document.APIs {
			names = append(names, api.Name)
			assert.False(t, api.Deprecated, "default listing must not contain deprecated APIs (aiMode=%v)", aiMode)
		}
		assert.Equal(t, []string{"AlphaActive", "CharlieActive", "EchoActive"}, names)
		assert.Equal(t, 3, document.Result.Total, "total reflects the supported surface")
		assert.False(t, document.Result.Truncated)
	}
}

func TestProductHelpAllMovesDeprecatedAPIsToEnd(t *testing.T) {
	document := productDocWithDeprecatedAPIs()
	applyProductHelpOptions(document, helpOptions{All: true}, true)

	names := make([]string, 0, len(document.APIs))
	for _, api := range document.APIs {
		names = append(names, api.Name)
	}
	assert.Equal(t,
		[]string{"AlphaActive", "CharlieActive", "EchoActive", "BravoLegacy", "DeltaLegacy"},
		names, "supported APIs lead, deprecated APIs trail")
	assert.Equal(t, 5, document.Result.Total)
}

func TestProductHelpSearchStillFindsDeprecatedAPIs(t *testing.T) {
	document := productDocWithDeprecatedAPIs()
	applyProductHelpOptions(document, helpOptions{Search: "bravo"}, true)

	require.Len(t, document.APIs, 1)
	assert.Equal(t, "BravoLegacy", document.APIs[0].Name)
	assert.True(t, document.APIs[0].Deprecated, "explicit search surfaces the deprecated flag")
}

func TestProductHelpOmittedDeprecatedCount(t *testing.T) {
	defaultDocument := productDocWithDeprecatedAPIs()
	applyProductHelpOptions(defaultDocument, helpOptions{}, true)
	assert.Equal(t, 3, defaultDocument.Result.Total)
	assert.Equal(t, 2, defaultDocument.Result.OmittedDeprecated, "default view reports how many deprecated APIs it hides")

	allDocument := productDocWithDeprecatedAPIs()
	applyProductHelpOptions(allDocument, helpOptions{All: true}, true)
	assert.Equal(t, 5, allDocument.Result.Total)
	assert.Equal(t, 0, allDocument.Result.OmittedDeprecated, "show-all hides nothing")

	var text bytes.Buffer
	require.NoError(t, renderCanonicalProductText(&text, defaultDocument, ""))
	assert.Contains(t, text.String(), "Omitting 2 deprecated APIs; use --help-all to include them.")
}
