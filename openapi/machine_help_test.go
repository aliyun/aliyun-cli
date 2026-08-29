package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMachineHelpService(t *testing.T) *machineHelpService {
	t.Helper()
	repo := canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	return newMachineHelpService(repo)
}

func testMachineHelpRootCommand() *cli.Command {
	root := &cli.Command{Name: "aliyun", Short: i18n.T("Alibaba Cloud CLI", "阿里云 CLI")}
	root.AddSubCommand(&cli.Command{Name: "plugin", Short: i18n.T("Manage plugins", "管理插件")})
	root.AddSubCommand(&cli.Command{Name: "configure", Short: i18n.T("Configure credentials", "配置凭证")})
	root.AddSubCommand(&cli.Command{Name: "internal", Hidden: true, Short: i18n.T("Internal", "内部")})
	return root
}

func TestMachineHelpRoot(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildRoot(testMachineHelpRootCommand())
	require.NoError(t, err)

	assert.Equal(t, machineHelpSchemaVersion, doc.SchemaVersion)
	assert.Equal(t, "root", doc.Kind)
	assert.Equal(t, []string{"aliyun"}, doc.Target.Path)
	require.Len(t, doc.Commands, 2)
	assert.Equal(t, "configure", doc.Commands[0].Name)
	assert.Equal(t, "plugin", doc.Commands[1].Name)
	require.Len(t, doc.Products, 1)
	assert.Equal(t, "demo", doc.Products[0].Code)
	assert.Equal(t, []string{"camel", "kebab"}, doc.Products[0].CommandStyles)
	assert.True(t, doc.Products[0].CanonicalHelp)
}

func TestMachineHelpRootJSONUsesExplicitGroups(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildRoot(testMachineHelpRootCommand())
	require.NoError(t, err)

	var output bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&output, doc, false))
	var raw map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &raw))
	assert.NotContains(t, raw, "commands")
	assert.Contains(t, raw, "coreCommands")
	assert.Contains(t, raw, "products")
}

func TestMachineHelpProduct(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildProduct("demo", "")
	require.NoError(t, err)

	assert.Equal(t, machineHelpSchemaVersion, doc.SchemaVersion)
	assert.Equal(t, "product", doc.Kind)
	assert.Equal(t, []string{"aliyun", "demo"}, doc.Target.Path)
	assert.Equal(t, "2026-01-01", doc.Product.LegacyDefaultVersion)
	assert.Equal(t, "2025-01-01", doc.Product.PluginDefaultVersion)
	assert.Equal(t, []string{"2025-01-01", "2026-01-01"}, doc.Product.SupportedVersions)
	assert.Equal(t, "2025-01-01", doc.Product.SelectedVersion)
	require.Len(t, doc.APIs, 1)
	// buildProduct renders kebab style: the PascalCase name must stay hidden.
	assert.Empty(t, doc.APIs[0].Name)
	assert.Equal(t, "describe-regions", doc.APIs[0].CmdName)
}

func TestMachineHelpProductExplicitVersion(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildProduct("DEMO", "2026-01-01")
	require.NoError(t, err)

	assert.Equal(t, "2026-01-01", doc.Product.SelectedVersion)
	require.Len(t, doc.APIs, 2)
	assert.Equal(t, "create-report", doc.APIs[0].CmdName)
	assert.Equal(t, "describe-regions", doc.APIs[1].CmdName)
}

func TestMachineHelpAPICamelAndKebabShareCanonicalIdentity(t *testing.T) {
	service := testMachineHelpService(t)
	camel, err := service.buildAPI("demo", "CreateReport", "2026-01-01")
	require.NoError(t, err)
	kebab, err := service.buildAPI("demo", "create-report", "2026-01-01")
	require.NoError(t, err)

	assert.Equal(t, machineHelpSchemaVersion, camel.SchemaVersion)
	assert.Equal(t, "api", camel.Kind)
	assert.Equal(t, helpSectionRequest, camel.Section)
	// Each document exposes only its own style's identifiers.
	assert.Equal(t, "CreateReport", camel.API.Name)
	assert.Empty(t, camel.API.CmdName)
	assert.Equal(t, "demo CreateReport", camel.API.CmdFullName)
	assert.Empty(t, kebab.API.Name)
	assert.Equal(t, "create-report", kebab.API.CmdName)
	assert.Equal(t, "demo create-report", kebab.API.CmdFullName)
	assert.Empty(t, kebab.API.Operation.Action)
	assert.Equal(t, "camel", camel.ActiveParameterSet)
	assert.Equal(t, "kebab", kebab.ActiveParameterSet)
	assert.Equal(t, []string{"aliyun", "demo", "CreateReport"}, camel.Target.Path)
	assert.Equal(t, []string{"aliyun", "demo", "create-report"}, kebab.Target.Path)

	assert.NotNil(t, findMachineHelpParameter(camel.ParameterSets.Camel, "--body"))
	assert.NotNil(t, findMachineHelpParameter(camel.ParameterSets.Camel, "--ReportId"))
	assert.NotNil(t, findMachineHelpParameter(kebab.ParameterSets.Kebab, "--report-id"))
	assert.NotNil(t, findMachineHelpParameter(kebab.ParameterSets.Kebab, "--workspace-id"))
	assert.Equal(t, "aliyun demo CreateReport ....", camel.Examples.Camel)
	assert.Equal(t, "aliyun demo create-report ...", camel.Examples.Kebab)
}

func TestMachineHelpJSONOmitsEmptyOptionalValues(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildAPI("demo", "CreateReport", "2026-01-01")
	require.NoError(t, err)

	var encoded bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&encoded, doc, false))
	assert.NotContains(t, encoded.String(), `"outputSchema"`)
	assert.NotContains(t, encoded.String(), `"pagination"`)
	assert.NotContains(t, encoded.String(), `"risk"`)
	assert.NotContains(t, encoded.String(), `"recovery"`)
	assert.NotContains(t, encoded.String(), `: null`)
	assert.NotContains(t, encoded.String(), `: ""`)
	assert.NotContains(t, encoded.String(), `: []`)
	assert.NotContains(t, encoded.String(), `: {}`)
	assert.Contains(t, encoded.String(), `"required": false`)
}

func TestMachineHelpJSONPreservesResponseSchemaValues(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"Token":{"type":"string","const":"","default":"","enum":[""]},"Counter":{"type":"integer","maximum":9223372036854775809}}}`)
	doc := &machineHelpAPIResponseDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Kind:          "api",
		Section:       helpSectionResponse,
		OutputSchema: &machineHelpOutputSchema{
			StatusCode:  "200",
			ContentType: "application/json",
			Schema:      schema,
			Components: &machineHelpComponents{Schemas: map[string]json.RawMessage{
				"Payload":  schema,
				"Anything": json.RawMessage(`{}`),
			}},
		},
		Components: &machineHelpComponents{Schemas: map[string]json.RawMessage{
			"TopLevelEmpty": json.RawMessage(`{}`),
		}},
	}

	var encoded bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&encoded, doc, false))
	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, encoded.Bytes()))
	output := compact.String()
	assert.Contains(t, output, `"const":""`)
	assert.Contains(t, output, `"default":""`)
	assert.Contains(t, output, `"enum":[""]`)
	assert.Contains(t, output, `"maximum":9223372036854775809`)
	assert.Contains(t, output, `"Anything":{}`)
	assert.Contains(t, output, `"TopLevelEmpty":{}`)
}

func TestMachineHelpAPIDefaultVersionFollowsCommandStyle(t *testing.T) {
	service := testMachineHelpService(t)
	camel, err := service.buildAPI("demo", "DescribeRegions", "")
	require.NoError(t, err)
	assert.Equal(t, "2026-01-01", camel.Product.SelectedVersion)

	// The plugin default is 2025-01-01. Its index resolves the kebab command
	// even though the API fixture itself intentionally does not exist there.
	_, err = service.buildAPI("demo", "describe-regions", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2025-01-01")
}

func TestMachineHelpAPIResponseUsesCanonicalSchemaAndReachableComponents(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildAPIResponse("demo", "CreateReport", "2026-01-01")
	require.NoError(t, err)

	assert.Equal(t, helpSectionResponse, doc.Section)
	assert.Equal(t, "camel", doc.Target.RequestedStyle)
	require.NotNil(t, doc.OutputSchema)
	assert.Equal(t, "200", doc.OutputSchema.StatusCode)
	assert.Equal(t, "application/json", doc.OutputSchema.ContentType)
	assert.JSONEq(t, `{"type":"object","properties":{"RequestId":{"type":"string","description":"The request ID."},"Reports":{"$ref":"#/components/schemas/ReportList"}}}`, string(doc.OutputSchema.Schema))
	require.NotNil(t, doc.OutputSchema.Components)
	assert.Contains(t, doc.OutputSchema.Components.Schemas, "ReportList")
	assert.Contains(t, doc.OutputSchema.Components.Schemas, "Report")
	assert.NotContains(t, doc.OutputSchema.Components.Schemas, "Unused")
	assert.Empty(t, doc.Notice)
}

func TestMachineHelpAPIResponseWithoutSchemaReturnsNotice(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildAPIResponse("demo", "DescribeRegions", "2026-01-01")
	require.NoError(t, err)

	assert.Nil(t, doc.OutputSchema)
	assert.Equal(t, "No response schema is available for this API.", doc.Notice)

	var encoded bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&encoded, doc, false))
	assert.NotContains(t, encoded.String(), `"outputSchema"`)
	assert.Contains(t, encoded.String(), `"notice": "No response schema is available for this API."`)
}

func TestMachineHelpRequestIncludesStylePreservingResponseQueryExample(t *testing.T) {
	service := testMachineHelpService(t)

	camel, err := service.buildAPI("demo", "CreateReport", "2026-01-01")
	require.NoError(t, err)
	require.NotNil(t, camel.ResponseQuery)
	assert.Equal(t, "Reports.Report[*].{ReportId:ReportId}", camel.ResponseQuery.Path)
	assert.Equal(t, "aliyun help demo CreateReport --api-version 2026-01-01 --cli-section response", camel.ResponseQuery.SchemaCommand)
	assert.Equal(t, "aliyun demo CreateReport --api-version 2026-01-01 --ReportId <value> --WorkspaceId <value> --cli-query 'Reports.Report[*].{ReportId:ReportId}'", camel.ResponseQuery.QueryCommand)

	kebab, err := service.buildAPI("demo", "create-report", "2026-01-01")
	require.NoError(t, err)
	require.NotNil(t, kebab.ResponseQuery)
	assert.Equal(t, "aliyun help demo create-report --api-version 2026-01-01 --cli-section response", kebab.ResponseQuery.SchemaCommand)
	assert.Equal(t, "aliyun demo create-report --api-version 2026-01-01 --report-id <value> --workspace-id <value> --cli-query 'Reports.Report[*].{ReportId:ReportId}'", kebab.ResponseQuery.QueryCommand)
}

func TestMachineHelpRequestSearchKeepsOnlyActiveParameterSetAndGlobals(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildAPI("demo", "create-report", "2026-01-01")
	require.NoError(t, err)
	doc.GlobalParameters = []machineHelpParameter{{
		Name: "header", Options: []string{"--header"}, Type: "string", Location: "global",
	}}

	applyRequestHelpOptions(doc, helpOptions{Search: "report-id"}, false)

	require.Len(t, doc.ParameterSets.Kebab, 1)
	assert.Equal(t, "report_id", doc.ParameterSets.Kebab[0].Name)
	assert.Empty(t, doc.ParameterSets.Camel)
	assert.Empty(t, doc.GlobalParameters)

	doc, err = service.buildAPI("demo", "create-report", "2026-01-01")
	require.NoError(t, err)
	doc.GlobalParameters = []machineHelpParameter{{
		Name: "header", Options: []string{"--header"}, Type: "string", Location: "global",
	}}
	applyRequestHelpOptions(doc, helpOptions{Search: "header"}, false)
	assert.Empty(t, doc.ParameterSets.Kebab)
	require.Len(t, doc.GlobalParameters, 1)
	assert.Equal(t, "header", doc.GlobalParameters[0].Name)
}

func TestMachineHelpAIRequestRemainsComplete(t *testing.T) {
	newDocument := func() *machineHelpAPIDocument {
		parameters := make([]machineHelpParameter, 0, 23)
		for index := 1; index <= 21; index++ {
			parameters = append(parameters, machineHelpParameter{Name: fmt.Sprintf("optional-%02d", index)})
		}
		parameters = append(parameters,
			machineHelpParameter{Name: "required-one", Required: true},
			machineHelpParameter{Name: "required-two", Required: true},
		)
		return &machineHelpAPIDocument{
			ActiveParameterSet: "camel",
			ParameterSets: machineHelpParameterSets{
				Camel: parameters,
				Kebab: []machineHelpParameter{{Name: "inactive-style"}},
			},
			GlobalParameters: []machineHelpParameter{
				{Name: "global-one"},
				{Name: "global-two"},
				{Name: "global-three"},
			},
		}
	}

	complete := newDocument()
	applyRequestHelpOptions(complete, helpOptions{}, true)
	require.Len(t, complete.ParameterSets.Camel, 23)
	assert.Equal(t, "optional-01", complete.ParameterSets.Camel[0].Name)
	assert.Empty(t, complete.ParameterSets.Kebab, "explicit sections must not expose the inactive style")
	assert.Empty(t, complete.GlobalParameters, "AI mode drops global CLI flags from request Help")
	assert.Nil(t, complete.Listing)

	legacy := newDocument()
	applyRequestHelpOptions(legacy, helpOptions{}, false)
	require.Len(t, legacy.ParameterSets.Camel, 23)
	assert.Equal(t, "optional-01", legacy.ParameterSets.Camel[0].Name)
	assert.Empty(t, legacy.ParameterSets.Kebab, "explicit sections must not expose the inactive style")
	assert.Len(t, legacy.GlobalParameters, 3)
	assert.Nil(t, legacy.Listing)
}

func TestMachineHelpResponseSearchProjectsMatchesAndFilteredQuery(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildAPIResponse("demo", "CreateReport", "2026-01-01")
	require.NoError(t, err)

	require.NoError(t, applyResponseHelpOptions(doc, helpOptions{Search: "report-id"}))
	assert.Equal(t, "report-id", doc.Query)
	assert.Empty(t, doc.Responses, "Search must not leak the unfiltered full response document")
	assert.Nil(t, doc.Components)
	assert.Equal(t, []string{"Reports.Report.ReportId"}, doc.Matches)
	require.NotNil(t, doc.OutputSchema)
	require.NotNil(t, doc.OutputSchema.Components)
	assert.Contains(t, doc.OutputSchema.Components.Schemas, "ReportList")
	assert.Contains(t, doc.OutputSchema.Components.Schemas, "Report")
	assert.NotContains(t, doc.OutputSchema.Components.Schemas, "Unused")

	doc.ResponseQuery = projectResponseQueryExample(
		helpResponseSchema(doc), "demo", "CreateReport", doc.Target.RequestedStyle, "2026-01-01",
	)
	require.NotNil(t, doc.ResponseQuery)
	assert.Equal(t, "Reports.Report[*].{ReportId:ReportId}", doc.ResponseQuery.Path)
}

func TestMachineHelpResponseSearchNoMatchReturnsClearNotice(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildAPIResponse("demo", "CreateReport", "2026-01-01")
	require.NoError(t, err)

	require.NoError(t, applyResponseHelpOptions(doc, helpOptions{Search: "does-not-exist"}))
	assert.Nil(t, doc.OutputSchema)
	assert.Equal(t, `No Help entries matched --help-search "does-not-exist".`, doc.Notice)
}

func TestMachineHelpNestedParameters(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildAPI("demo", "DescribeRegions", "2026-01-01")
	require.NoError(t, err)

	tags := findMachineHelpParameter(doc.ParameterSets.Kebab, "--tags")
	require.NotNil(t, tags)
	assert.Equal(t, "array", tags.Type)
	require.NotNil(t, tags.Element)
	assert.Equal(t, "object", tags.Element.Type)
	require.Len(t, tags.Element.Fields, 2)
	assert.Equal(t, "key", tags.Element.Fields[0].Name)
	assert.Empty(t, tags.Element.Fields[0].RawName, "kebab help must not expose PascalCase wire names")
	assert.Equal(t, "value", tags.Element.Fields[1].Name)
	assert.Empty(t, tags.Element.Fields[1].RawName)

	legacyTags := findMachineHelpParameter(doc.ParameterSets.Camel, "--Tags")
	require.NotNil(t, legacyTags)
	require.NotNil(t, legacyTags.Element)
	require.Len(t, legacyTags.Element.Fields, 2)
	assert.Equal(t, []string{"--Tags.#.Key"}, legacyTags.Element.Fields[0].Options)
}

func findMachineHelpParameter(parameters []machineHelpParameter, option string) *machineHelpParameter {
	for i := range parameters {
		for _, candidate := range parameters[i].Options {
			if candidate == option {
				return &parameters[i]
			}
		}
	}
	return nil
}

func TestCommandoHelpJSONUsesCanonicalMetadataWithoutLoadingPlugins(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	c.library.helpRepo = canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	root := testMachineHelpRootCommand()
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	MachineHelpFormatFlag(ctx.Flags()).SetAssigned(true)
	MachineHelpFormatFlag(ctx.Flags()).SetValue("json")
	VersionFlag(ctx.Flags()).SetAssigned(true)
	VersionFlag(ctx.Flags()).SetValue("2026-01-01")

	err := c.help(ctx, []string{"demo", "CreateReport"})
	require.NoError(t, err)
	assert.False(t, c.pluginLoaded)
	assert.Empty(t, stderr.String())

	var doc machineHelpAPIDocument
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc))
	assert.Equal(t, machineHelpSchemaVersion, doc.SchemaVersion)
	assert.Equal(t, "api", doc.Kind)
	assert.Equal(t, "CreateReport", doc.API.Name)
	assert.Empty(t, doc.GlobalParameters, "default Action Help does not repeat Root global flags")
}

func TestCommandoHelpJSONResponseSection(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	c.library.helpRepo = canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	root := testMachineHelpRootCommand()
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	MachineHelpFormatFlag(ctx.Flags()).SetAssigned(true)
	MachineHelpFormatFlag(ctx.Flags()).SetValue("json")
	CliHelpSectionFlag(ctx.Flags()).SetAssigned(true)
	CliHelpSectionFlag(ctx.Flags()).SetValue("response")
	VersionFlag(ctx.Flags()).SetAssigned(true)
	VersionFlag(ctx.Flags()).SetValue("2026-01-01")

	err := c.help(ctx, []string{"demo", "CreateReport"})
	require.NoError(t, err)
	assert.False(t, c.pluginLoaded)
	assert.Empty(t, stderr.String())

	var doc machineHelpAPIResponseDocument
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc))
	assert.Equal(t, helpSectionResponse, doc.Section)
	// The complete document carries the schema once, in responses.
	assert.NotNil(t, doc.Responses)
	assert.Nil(t, doc.OutputSchema)
	assert.NotContains(t, stdout.String(), `"parameterSets"`)
}

func TestCommandoHelpJSONRequestSearchReturnsOnlyActiveMatches(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	c.library.helpRepo = canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	root := testMachineHelpRootCommand()
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	MachineHelpFormatFlag(ctx.Flags()).SetAssigned(true)
	MachineHelpFormatFlag(ctx.Flags()).SetValue("json")
	CliHelpSearchFlag(ctx.Flags()).SetAssigned(true)
	CliHelpSearchFlag(ctx.Flags()).SetValue("workspace-id")
	VersionFlag(ctx.Flags()).SetAssigned(true)
	VersionFlag(ctx.Flags()).SetValue("2026-01-01")

	require.NoError(t, c.help(ctx, []string{"demo", "create-report"}))
	assert.Empty(t, stderr.String())

	var document map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &document))
	sets := document["parameterSets"].(map[string]any)
	assert.NotContains(t, sets, "camel")
	kebab := sets["kebab"].([]any)
	require.Len(t, kebab, 1)
	assert.Equal(t, "workspace_id", kebab[0].(map[string]any)["name"])
	assert.NotContains(t, document, "globalParameters")
	assert.Contains(t, document, "responseQueryExample")
}

func TestCommandoHelpJSONResponseSearchReturnsPathsAndMinimalSchema(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	c.library.helpRepo = canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	root := testMachineHelpRootCommand()
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	MachineHelpFormatFlag(ctx.Flags()).SetAssigned(true)
	MachineHelpFormatFlag(ctx.Flags()).SetValue("json")
	CliHelpSectionFlag(ctx.Flags()).SetAssigned(true)
	CliHelpSectionFlag(ctx.Flags()).SetValue("response")
	CliHelpSearchFlag(ctx.Flags()).SetAssigned(true)
	CliHelpSearchFlag(ctx.Flags()).SetValue("report-id")
	VersionFlag(ctx.Flags()).SetAssigned(true)
	VersionFlag(ctx.Flags()).SetValue("2026-01-01")

	require.NoError(t, c.help(ctx, []string{"demo", "CreateReport"}))
	assert.Empty(t, stderr.String())
	assert.Contains(t, stdout.String(), `"matches": [`)
	assert.Contains(t, stdout.String(), `"Reports.Report.ReportId"`)
	assert.Contains(t, stdout.String(), `"responseQueryExample"`)
	assert.NotContains(t, stdout.String(), `"Unused"`)
}

func TestCommandoHelpJSONAcceptsKebabAPIVersionFlag(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	c.library.helpRepo = canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	root := testMachineHelpRootCommand()
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	MachineHelpFormatFlag(ctx.Flags()).SetAssigned(true)
	MachineHelpFormatFlag(ctx.Flags()).SetValue("json")
	ctx.SetUnknownFlags(cli.NewFlagSet())
	apiVersion, err := ctx.UnknownFlags().AddByName("api-version")
	require.NoError(t, err)
	apiVersion.SetAssigned(true)
	apiVersion.SetValue("2026-01-01")

	err = c.help(ctx, []string{"demo", "create-report"})
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var doc machineHelpAPIDocument
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc))
	assert.Equal(t, "kebab", doc.ActiveParameterSet)
	assert.Equal(t, "2026-01-01", doc.Product.SelectedVersion)
}

func TestCommandoHelpJSONRejectsUnsupportedCLIOutputAsTypedOptionError(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	root := testMachineHelpRootCommand()
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	MachineHelpFormatFlag(ctx.Flags()).SetAssigned(true)
	MachineHelpFormatFlag(ctx.Flags()).SetValue("yaml")

	err := c.help(ctx, nil)
	require.Error(t, err)
	var optionErr *cli.HelpOptionError
	require.True(t, errors.As(err, &optionErr))
	assert.Equal(t, cli.HelpOptionInvalidOutput, optionErr.Code)
	assert.Equal(t, "yaml", optionErr.Value)
	assert.False(t, c.pluginLoaded)
}

func TestBuildAPIIncludesQueryOptions(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildAPI("demo", "CreateReport", "2026-01-01")
	require.NoError(t, err)

	require.Len(t, doc.QueryOptions, 3)
	section := doc.QueryOptions[0]
	assert.Equal(t, "--cli-section", section.Name)
	assert.True(t, section.HasDefault)
	assert.Equal(t, "request", section.Default)

	var encoded bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&encoded, doc, false))
	assert.Contains(t, encoded.String(), `"queryOptions"`)
	assert.Contains(t, encoded.String(), `"default": "request"`)
}

func TestMachineHelpRequestSearchDropsInvocationExample(t *testing.T) {
	service := testMachineHelpService(t)
	document, err := service.buildAPI("demo", "create-report", "2026-01-01")
	require.NoError(t, err)

	applyRequestHelpOptions(document, helpOptions{Search: "report-id"}, true)

	assert.Equal(t, "", document.Examples.Kebab)
	assert.Equal(t, "", document.Examples.Camel)
	require.NotEmpty(t, document.ParameterSets.Kebab)
	assert.Equal(t, "report_id", document.ParameterSets.Kebab[0].Name)
}

func TestMachineHelpAggregatesDocRequiredIntoRequired(t *testing.T) {
	parameter := canonicalmeta.Parameter{
		Name:        "user_name",
		RawName:     "UserName",
		Type:        "string",
		Required:    false,
		DocRequired: true,
		Location:    "query",
		Options:     []string{"--user-name"},
		Fields: []canonicalmeta.Field{{
			Name:        "token",
			RawName:     "Token",
			Type:        "string",
			Required:    false,
			DocRequired: true,
		}},
	}

	t.Run("kebab parameter and nested field", func(t *testing.T) {
		projected := projectCanonicalParameter(&parameter)
		assert.True(t, projected.Required, "doc_required must surface as required")
		require.Len(t, projected.Fields, 1)
		assert.True(t, projected.Fields[0].Required, "nested doc_required must surface as required")
	})

	t.Run("camel legacy view", func(t *testing.T) {
		view := canonicalmeta.NewCanonicalView(&parameter)
		projected := projectLegacyParameter(view, "")
		assert.True(t, projected.Required, "camel help must aggregate doc_required")
	})

	t.Run("protocol-required stays true without doc_required", func(t *testing.T) {
		plain := parameter
		plain.Required = true
		plain.DocRequired = false
		assert.True(t, projectCanonicalParameter(&plain).Required)
	})
}

func TestMachineHelpSearchDropsMetadataInAIMode(t *testing.T) {
	service := testMachineHelpService(t)
	document, err := service.buildAPI("demo", "create-report", "2026-01-01")
	require.NoError(t, err)

	applyRequestHelpOptions(document, helpOptions{Search: "report-id"}, true)

	var encoded bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&encoded, document, true))
	out := encoded.String()

	var raw map[string]any
	require.NoError(t, json.Unmarshal(encoded.Bytes(), &raw))
	assert.NotContains(t, raw, "api")
	assert.NotContains(t, raw, "product")
	assert.NotContains(t, raw, "queryOptions")
	assert.NotContains(t, raw, "responseQueryExample")
	assert.NotContains(t, raw, "examples")
	assert.Contains(t, raw, "parameterSets")
	assert.Contains(t, raw, "target")
	assert.Contains(t, out, "report_id")
}
