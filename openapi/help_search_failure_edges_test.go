package openapi

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/require"
)

func isolateSearchEdgeHelp(t *testing.T, ctx *cli.Context) {
	t.Helper()
	config.AddFlags(ctx.Flags())
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(filepath.Join(t.TempDir(), "config.json"))
	t.Setenv(originalProductHelpEnv, "")
	t.Setenv(baselineProductHelpEnv, "")
	previous := i18n.GetLanguage()
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	i18n.SetLanguage("en")
}

func TestSearchEdgeProductHelpPropagatesProviderFailures(t *testing.T) {
	for _, provider := range []string{"metadata plugin", "Go plugin", "baseline"} {
		t.Run(provider, func(t *testing.T) {
			c, ctx, output, _ := newCanonicalHelpTestContext(t)
			isolateSearchEdgeHelp(t, ctx)
			repo, err := meta.MockLoadRepository([]meta.Product{{Code: "searchedge", Version: "v1", ApiStyle: "rpc"}})
			require.NoError(t, err)
			c.library.builtinRepo = repo
			if provider != "baseline" {
				kind := plugin.PluginTypeMeta
				if provider == "Go plugin" {
					kind = plugin.PluginTypeGo
				}
				c.localManifest.Plugins["aliyun-cli-searchedge"] = plugin.LocalPlugin{Name: "aliyun-cli-searchedge", Command: "searchedge", Type: kind}
			} else {
				t.Setenv(baselineProductHelpEnv, "true")
			}
			previousRender, previousExecute, previousAvailable := productHelpRender, helpDelegateExecute, productHelpAvailable
			t.Cleanup(func() {
				productHelpRender, helpDelegateExecute, productHelpAvailable = previousRender, previousExecute, previousAvailable
			})
			cause := errors.New("help provider unavailable")
			calls := 0
			productHelpAvailable = func(product string) bool { return product == "searchedge" }
			productHelpRender = func(context *cli.Context, product string) error {
				require.NotEqual(t, "Go plugin", provider)
				require.Same(t, ctx, context)
				require.Equal(t, "searchedge", product)
				calls++
				return cause
			}
			helpDelegateExecute = func(product string, args []string, context *cli.Context) (bool, error) {
				require.Equal(t, "Go plugin", provider)
				require.Equal(t, "searchedge", product)
				require.Equal(t, []string{"searchedge", "--help"}, args)
				require.Same(t, ctx, context)
				calls++
				return true, cause
			}
			require.ErrorIs(t, c.printProductUsage(ctx, "searchedge"), cause)
			require.Equal(t, 1, calls)
			require.NotContains(t, output.String(), "Available Api List")
		})
	}
}

func TestSearchEdgeLegacyProductHelpPreservesMetadataAnnotations(t *testing.T) {
	for _, style := range []string{"rpc", "restful"} {
		t.Run(style, func(t *testing.T) {
			c, ctx, output, _ := newCanonicalHelpTestContext(t)
			isolateSearchEdgeHelp(t, ctx)
			repo, err := meta.MockLoadRepository([]meta.Product{{Code: "searchedge", Version: "v1", ApiStyle: style, ApiNames: []string{"Missing", "OldEndpoint", "PublicEndpoint", "Undocumented"}}})
			require.NoError(t, err)
			c.library.builtinRepo = repo
			canonical := newFakeCanonicalRepo()
			canonical.AddAPI("searchedge", "v1", &canonicalmeta.API{Name: "OldEndpoint", Method: "GET", PathPattern: "/old", DescriptionEn: "old endpoint", Deprecated: true})
			canonical.AddAPI("searchedge", "v1", &canonicalmeta.API{Name: "PublicEndpoint", Method: "GET", PathPattern: "/public", DescriptionEn: "public endpoint", Security: []string{"Anonymous"}})
			canonical.AddAPI("searchedge", "v1", &canonicalmeta.API{Name: "Undocumented", Method: "POST", PathPattern: "/undocumented"})
			c.library.canonicalRepo = canonical
			previous := productHelpAvailable
			t.Cleanup(func() { productHelpAvailable = previous })
			productHelpAvailable = func(string) bool { return false }
			require.NoError(t, c.printProductUsage(ctx, "searchedge"))
			require.Contains(t, output.String(), "[Deprecated]old endpoint")
			require.Contains(t, output.String(), "Undocumented")
			if style == "restful" {
				require.NotContains(t, output.String(), "Missing")
				require.Contains(t, output.String(), "GET /old")
				require.Contains(t, output.String(), "POST /undocumented")
			} else {
				require.Contains(t, output.String(), "[Anonymous]public endpoint")
				require.Contains(t, output.String(), "Missing")
			}
		})
	}
}

func TestSearchEdgeMissingPluginProductsOfferExactInstallCommand(t *testing.T) {
	c, ctx, output, _ := newCanonicalHelpTestContext(t)
	isolateSearchEdgeHelp(t, ctx)
	repo, err := meta.MockLoadRepository(nil)
	require.NoError(t, err)
	c.library.builtinRepo = repo
	c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{{Name: "aliyun-cli-searchedge", ProductCode: "searchedge"}}}
	previous := runtimeTryHelp
	t.Cleanup(func() { runtimeTryHelp = previous })
	runtimeTryHelp = func(*cli.Context, []string) (bool, error) { return false, nil }
	require.ErrorContains(t, c.printApiUsage(ctx, "searchedge", "ListItems"), "aliyun plugin install --names aliyun-cli-searchedge")
	require.Empty(t, output.String())
	err = c.invalidBaselineCommandError("searchedge", "bad-command")
	require.ErrorContains(t, err, "available kebab-case commands")
	require.NotContains(t, err.Error(), "traditional PascalCase")
	printProductHelpSwitchHint(ctx, "searchedge; echo invalid", productHelpTraditionalToKebab)
	require.Empty(t, output.String())
}

func TestSearchEdgeAPIHelpDistinguishesLegacyMetadataAndPluginRecovery(t *testing.T) {
	c, ctx, output, _ := newCanonicalHelpTestContext(t)
	isolateSearchEdgeHelp(t, ctx)
	t.Setenv(plugin.EnvPluginsDir, t.TempDir())
	repo, err := meta.MockLoadRepository([]meta.Product{{Code: "searchedge", Version: "v1", ApiStyle: "restful", ApiNames: []string{"ListItems"}}})
	require.NoError(t, err)
	c.library.builtinRepo = repo
	canonical := newFakeCanonicalRepo()
	canonical.AddAPI("searchedge", "v1", &canonicalmeta.API{Name: "ListItems", Method: "GET", PathPattern: "/items"})
	c.library.canonicalRepo = canonical
	previous := runtimeTryHelp
	t.Cleanup(func() { runtimeTryHelp = previous })
	runtimeTryHelp = func(*cli.Context, []string) (bool, error) { return false, nil }
	require.NoError(t, c.printApiUsage(ctx, "searchedge", "ListItems"))
	require.Contains(t, output.String(), "Method:      GET")
	require.Contains(t, output.String(), "PathPattern: /items")
	output.Reset()
	c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{{Name: "aliyun-cli-searchedge", ProductCode: "searchedge"}}}
	err = c.printApiUsage(ctx, "searchedge", "unknown-command")
	require.ErrorContains(t, err, "not a valid built-in command")
	require.ErrorContains(t, err, "aliyun plugin install --names aliyun-cli-searchedge")
	c.localManifest.Plugins["aliyun-cli-searchedge"] = plugin.LocalPlugin{Name: "aliyun-cli-searchedge", Command: "searchedge", Type: plugin.PluginTypeGo, Version: "1.2.3"}
	var unified *InvalidUnifiedApiError
	require.ErrorAs(t, c.printApiUsage(ctx, "searchedge", "MissingAction"), &unified)
	require.Equal(t, "MissingAction", unified.Name)
	require.Equal(t, "searchedge", unified.product.Code)
	require.Equal(t, "1.2.3", unified.lPlugin.Version)
	require.Empty(t, output.String())
}

func TestSearchEdgeResponseSearchMergesDuplicateCompositionPaths(t *testing.T) {
	document := HelpResponseSchema{Schema: json.RawMessage(`{"properties":{"Report":{"oneOf":[{"properties":{"Name":{"type":"string","description":"report name"}}},{"properties":{"Name":{"type":"string"}}}]}}}`)}
	result, err := SearchResponseSchema(document, "report name", false)
	require.NoError(t, err)
	require.Equal(t, []string{"Report.Name"}, result.Paths)
	require.Equal(t, HelpResult{Shown: 1, Total: 1}, result.Result)
	require.JSONEq(t, string(document.Schema), string(result.Schema))
	tied, err := SearchResponseSchema(HelpResponseSchema{Schema: json.RawMessage(`{"properties":{"ReportName":{"type":"string"},"report-name":{"type":"integer"}}}`)}, "report_name", false)
	require.NoError(t, err)
	require.Equal(t, []string{"ReportName", "report-name"}, tied.Paths)
}

func TestSearchEdgeResponseSearchKeepsBooleanItemsAndMissingReferences(t *testing.T) {
	document := HelpResponseSchema{Schema: json.RawMessage(`{"properties":{"Items":{"type":"array","items":{"allOf":[{"$ref":"#/components/schemas/Entry"},{"$ref":"#/components/schemas/Missing"}]}},"Other":{"type":"string"}}}`), Components: map[string]json.RawMessage{
		"Entry":    json.RawMessage(`{"required":["Id"],"allOf":[{"$ref":"#/components/schemas/Identity"},{"$ref":"#/components/schemas/Identity"}]}`),
		"Identity": json.RawMessage(`{"type":"object","properties":{"Id":{"type":"string"}}}`),
	}}
	result, err := SearchResponseSchema(document, "items", false)
	require.NoError(t, err)
	require.Equal(t, []string{"Items"}, result.Paths)
	require.Len(t, result.Components, 2)
	require.NotContains(t, result.Components, "Missing")
	for name, raw := range document.Components {
		require.JSONEq(t, string(raw), string(result.Components[name]))
	}
	result, err = SearchResponseSchema(HelpResponseSchema{Schema: json.RawMessage(`{"properties":{"Items":{"type":"array","items":true},"Status":{"type":"object","required":["Code"]}}}`)}, "status", false)
	require.NoError(t, err)
	require.JSONEq(t, `{"properties":{"Status":{"type":"object","required":["Code"]}}}`, string(result.Schema))
	result, err = SearchResponseSchema(HelpResponseSchema{Schema: json.RawMessage(`{"properties":{"Items":{"type":"array","items":true}}}`)}, "items", false)
	require.NoError(t, err)
	require.JSONEq(t, `{"properties":{"Items":{"type":"array","items":true}}}`, string(result.Schema))
}

func TestSearchEdgeResponseSearchRemovesUnmatchedCompositionAndRejectsInvalidSchema(t *testing.T) {
	document := HelpResponseSchema{Schema: json.RawMessage(`{"type":"object","allOf":[{"properties":{"Name":{"type":"string"}}}],"properties":{"Status":{"type":"string"}},"required":["Name"]}`)}
	result, err := SearchResponseSchema(document, "status", false)
	require.NoError(t, err)
	require.Equal(t, []string{"Status"}, result.Paths)
	require.JSONEq(t, `{"type":"object","properties":{"Status":{"type":"string"}}}`, string(result.Schema))
	validation, err := ValidateResponseSchemaSearch(HelpResponseSchema{Schema: json.RawMessage("{")}, "status")
	require.ErrorContains(t, err, "parse response schema: invalid JSON")
	require.Equal(t, HelpSearchValidation{}, validation)
}

func TestSearchEdgeQueryRejectsUnsafeIdentityAfterSelectingArray(t *testing.T) {
	document := HelpResponseSchema{Schema: json.RawMessage(`{"properties":{"Items":{"type":"array","items":{"type":"string"}}}}`)}
	path, err := SelectResponseArrayPath(document, "ListItems", "")
	require.NoError(t, err)
	require.Equal(t, "Items", path)
	for _, context := range []ResponseQueryContext{
		{API: "", Product: "demo", Style: ResponseCommandStyleCamel},
		{API: "---", Product: "demo", Style: ResponseCommandStyleCamel},
		{API: "___", Product: "demo", Style: ResponseCommandStyleKebab},
		{API: "ListItems", Product: "demo", Style: "unknown"},
		{API: "ListItems", Product: "demo;rm", Style: ResponseCommandStyleCamel},
		{API: "ListItems", Product: "demo", APIVersion: "v1;pwd", Style: ResponseCommandStyleCamel},
	} {
		context.Document = document
		example, err := BuildResponseQueryExample(context)
		require.NoError(t, err)
		require.Nil(t, example, "%+v", context)
	}
}

func TestSearchEdgeQueryQuotesApostrophesAndResolvesArrayAliases(t *testing.T) {
	document := HelpResponseSchema{Schema: json.RawMessage(`{"properties":{"Container":{"$ref":"#/components/schemas/Scalar"},"Owner'sItems":{"anyOf":[{"type":"string"},{"$ref":"#/components/schemas/ItemArray"}]}}}`), Components: map[string]json.RawMessage{
		"Scalar":    json.RawMessage(`{"type":"string"}`),
		"ItemArray": json.RawMessage(`{"type":"array","items":{"type":"string"}}`),
	}}
	example, err := BuildResponseQueryExample(ResponseQueryContext{Document: document, Product: "demo", API: "ListItems", Style: ResponseCommandStylePascal})
	require.NoError(t, err)
	require.NotNil(t, example)
	require.Equal(t, `"Owner'sItems"`, example.Path)
	require.Equal(t, `aliyun demo ListItems --cli-query '"Owner'"'"'sItems"'`, example.QueryCommand)
	require.Equal(t, "aliyun help demo ListItems --cli-section response", example.SchemaCommand)
}
