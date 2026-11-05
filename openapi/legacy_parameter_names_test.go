package openapi

import (
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/require"
)

func TestLegacyParameterFlagValidatorUsesFixedSchemaNames(t *testing.T) {
	api := &canonicalmeta.API{Parameters: []canonicalmeta.Parameter{
		{RawName: "Name", Type: "string", Location: "query"},
		{RawName: "Ids", Type: "array", Location: "query", Element: &canonicalmeta.TypeShape{Type: "string"}},
		{RawName: "Tags", Type: "array", Location: "query", Element: &canonicalmeta.TypeShape{Type: "object", Fields: []canonicalmeta.Field{
			{RawName: "Key", Type: "string"}, {RawName: "Value", Type: "string"}, {RawName: "Nested", Type: "array"},
		}}},
		{RawName: "Labels", Type: "map", Location: "query"},
		{RawName: "JsonItems", Type: "array", ParamStyle: "json", Location: "query"},
	}}
	validator := legacyParameterFlagValidator(api)
	for _, name := range []string{"--Name", "--Name-FILE", "--Ids", "--Ids.0", "--Ids.0002", "--Ids.1-FILE", "--Tags", "--Tags.1.Key", "--Tags.12.Value-FILE", "--Tags.2.Nested.3", "--Labels", "--JsonItems"} {
		require.True(t, validator(name), name)
	}
	for _, name := range []string{
		"--name", "-Name", "Name", "--Name-private", "--Name.1", "--Name-FILE-FILE",
		"--Ids.", "--Ids.private-customer", "--Ids.-1", "--Ids.+1", "--Ids.１２", "--Ids.1.private-customer",
		"--Tags.1", "--Tags.private-customer.Key", "--Tags.1.private-customer", "--Tags.1.Key.private-customer", "--Tags.1.Nested.private-customer",
		"--Labels.private-customer", "--JsonItems.1", "--private-customer-token",
	} {
		require.False(t, validator(name), name)
	}
	// The execution lookup deliberately retains its permissive compatibility
	// semantics; observing names must not tighten the actual command parser.
	require.NotNil(t, api.FindLegacyParameter("Tags.private-customer.Key"))
	require.NotNil(t, api.FindLegacyParameter("Ids.private-customer"))
	require.Nil(t, legacyParameterFlagValidator(nil))

	v1 := []canonicalmeta.V1Parameter{{Name: "Groups", Type: "array", SubParameters: []canonicalmeta.V1Parameter{{
		Name: "Items", Type: "array", SubParameters: []canonicalmeta.V1Parameter{{Name: "Name", Type: "string"}},
	}}}}
	validator = legacyParameterFlagValidator(&canonicalmeta.API{V1Parameters: &v1})
	require.True(t, validator("--Groups.1.Items.2.Name"))
	require.False(t, validator("--Groups.1.Items.private-customer.Name"))
	require.False(t, validator("--Groups.private-customer.Items.2.Name"))
}

func TestLegacyParameterNamesAreBoundBeforeCredentialFailure(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(map[bool]string{false: "credential failure", true: "invalid parameter"}[unknown], func(t *testing.T) {
			c, ctx, _, _ := newCanonicalHelpTestContext(t)
			config.AddFlags(ctx.Flags())
			ctx.SetUnknownFlags(cli.NewFlagSet())
			config.ConfigurePathFlag(ctx.Flags()).SetValue(filepath.Join(t.TempDir(), "missing-config.json"))
			args := []string{"demo", "CreateReport", "--version", "2026-01-01", "--ReportId=private-report", "--profile", "missing-profile"}
			if unknown {
				args = append(args, "--private-customer-token", "private")
			}
			ctx.SetInvocationArgs(args)
			positionals, err := cli.NewParser(args, ctx).ReadAll()
			require.NoError(t, err)
			require.Equal(t, []string{"--version", "--profile"}, ctx.ParsedFlagNames(-1), "dynamic names await the selected API")
			err = c.main(ctx, positionals)
			if unknown {
				var invalidParameter *InvalidParameterError
				require.ErrorAs(t, err, &invalidParameter)
			} else {
				var credentials *credentialConfigurationError
				require.ErrorAs(t, err, &credentials)
			}
			require.Equal(t, []string{"--version", "--ReportId", "--profile"}, ctx.ParsedFlagNames(-1))
		})
	}
}

func TestLegacyParameterNamesBoundAtActualAPISelection(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	ctx.SetUnknownFlags(cli.NewFlagSet())
	args := []string{"demo", "CreateReport", "--ReportId=private", "--private-customer-token", "private"}
	ctx.SetInvocationArgs(args)
	_, err := cli.NewParser(args, ctx).ReadAll()
	require.NoError(t, err)
	api := &canonicalmeta.API{Parameters: []canonicalmeta.Parameter{{RawName: "ReportId", Type: "string", Location: "query"}}}
	c.CheckApiParamWithBuildInArgs(ctx, api)
	require.Equal(t, []string{"--ReportId"}, ctx.ParsedFlagNames(-1))

	// Runtime routing preflight must not lend a legacy schema to a Go plugin
	// or runtime route selected after validation.
	ctx.SetInvocationArgs([]string{"demo", "create-report", "--version", "2026-01-01", "--ReportId=private"})
	ctx.SetUnknownFlags(cli.NewFlagSet())
	_, err = cli.NewParser(ctx.InvocationArgs(), ctx).ReadAll()
	require.NoError(t, err)
	require.NoError(t, c.validateCanonicalRuntimeCommand([]string{"demo", "create-report"}, ctx))
	require.Equal(t, []string{"--version"}, ctx.ParsedFlagNames(-1))
}
