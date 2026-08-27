package openapi

import (
	"bytes"
	"encoding/json"
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
	assert.Equal(t, "DescribeRegions", doc.APIs[0].Name)
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
	assert.Equal(t, "CreateReport", camel.API.Name)
	assert.Equal(t, "CreateReport", kebab.API.Name)
	assert.Equal(t, "create-report", camel.API.CmdName)
	assert.Equal(t, "create-report", kebab.API.CmdName)
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
	require.NoError(t, encodeMachineHelpJSON(&encoded, doc))
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
	assert.JSONEq(t, `{"type":"object","properties":{"RequestId":{"type":"string","description_en":"The request ID.","description_zh":"请求 ID。"},"Reports":{"$ref":"#/components/schemas/ReportList"}}}`, string(doc.OutputSchema.Schema))
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
	require.NoError(t, encodeMachineHelpJSON(&encoded, doc))
	assert.NotContains(t, encoded.String(), `"outputSchema"`)
	assert.Contains(t, encoded.String(), `"notice": "No response schema is available for this API."`)
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
	assert.Equal(t, "Key", tags.Element.Fields[0].RawName)
	assert.Equal(t, "Value", tags.Element.Fields[1].RawName)

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
	assert.NotEmpty(t, doc.GlobalParameters)
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
	require.NotNil(t, doc.OutputSchema)
	assert.Equal(t, "200", doc.OutputSchema.StatusCode)
	assert.NotContains(t, stdout.String(), `"parameterSets"`)
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

func TestCommandoHelpJSONRejectsUnsupportedFormatStructurally(t *testing.T) {
	c, stdout, stderr := newTestCommando()
	root := testMachineHelpRootCommand()
	AddFlags(root.Flags())
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	MachineHelpFormatFlag(ctx.Flags()).SetAssigned(true)
	MachineHelpFormatFlag(ctx.Flags()).SetValue("yaml")

	err := c.help(ctx, nil)
	require.Error(t, err)
	structured, ok := err.(cli.StructuredError)
	require.True(t, ok)
	var rendered bytes.Buffer
	require.NoError(t, structured.RenderError(&rendered))
	assert.JSONEq(t, `{"schemaVersion":"v1","error":{"code":"INVALID_FORMAT","message":"unsupported help format \"yaml\"","target":["aliyun"],"suggestions":["use --help=json or help --format json"]}}`, rendered.String())
	assert.False(t, c.pluginLoaded)
}
