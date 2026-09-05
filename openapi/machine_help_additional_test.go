package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMachineHelpErrorContractAndUnavailableCause(t *testing.T) {
	err := newMachineHelpError("UNKNOWN", "unknown target", []string{"aliyun", "demo"}, []string{"inspect help"})
	assert.Equal(t, "unknown target", err.Error())
	assert.Nil(t, err.Unwrap())
	assert.Equal(t, 2, err.ExitCode())

	var output bytes.Buffer
	require.NoError(t, err.RenderError(&output))
	assert.JSONEq(t, `{"schemaVersion":"v1","error":{"code":"UNKNOWN","message":"unknown target","target":["aliyun","demo"],"suggestions":["inspect help"]}}`, output.String())

	cause := errors.New("repository failed")
	unavailable := newMachineHelpUnavailableError([]string{"aliyun"}, cause)
	assert.ErrorIs(t, unavailable, cause)
	assert.Contains(t, unavailable.Error(), "repository failed")
	assert.Error(t, unavailable.RenderError(alwaysFailWriter{}))
}

func TestMachineHelpLocalizedTextUnmarshalForms(t *testing.T) {
	previous := i18n.GetLanguage()
	t.Cleanup(func() { i18n.SetLanguage(previous) })

	i18n.SetLanguage("en")
	var english machineHelpLocalizedText
	require.NoError(t, json.Unmarshal([]byte(`"English"`), &english))
	assert.Equal(t, "English", english.EN)

	i18n.SetLanguage("zh")
	var chinese machineHelpLocalizedText
	require.NoError(t, json.Unmarshal([]byte(`"中文"`), &chinese))
	assert.Equal(t, "中文", chinese.ZH)

	var bilingual machineHelpLocalizedText
	require.NoError(t, json.Unmarshal([]byte(`{"en":"English","zh":"中文"}`), &bilingual))
	assert.Equal(t, machineHelpLocalizedText{EN: "English", ZH: "中文"}, bilingual)
	assert.Error(t, json.Unmarshal([]byte(`42`), &bilingual))
}

func TestPrintMachineHelpRejectsInvalidRequests(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	assertMachineHelpCode := func(err error, code string) {
		t.Helper()
		var structured *machineHelpError
		require.ErrorAs(t, err, &structured)
		assert.Equal(t, code, structured.document.Error.Code)
	}

	assertMachineHelpCode(c.printMachineHelp(ctx, nil, "yaml", helpOptions{}), "INVALID_FORMAT")
	assertMachineHelpCode(c.printMachineHelp(ctx, []string{"demo", "CreateReport", "extra"}, "json", helpOptions{}), "INVALID_TARGET")

	missing := &Commando{library: &Library{}}
	assertMachineHelpCode(missing.printMachineHelp(ctx, nil, "json", helpOptions{}), "MACHINE_HELP_UNAVAILABLE")
}

func TestPrintMachineHelpAllTargetLevels(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		options helpOptions
		kind    string
	}{
		{"root", nil, helpOptions{}, "root"},
		{"product", []string{"demo"}, helpOptions{}, "product"},
		{"request", []string{"demo", "CreateReport"}, helpOptions{}, "api"},
		{"response", []string{"demo", "CreateReport"}, helpOptions{Section: helpSectionResponse, Search: "report-id"}, "api"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
			require.NoError(t, c.printMachineHelp(ctx, test.args, "json", test.options))
			assert.Contains(t, stdout.String(), `"helpLevel": "`+test.kind+`"`)
		})
	}
}

func TestMachineHelpJSONPreparationAndPruning(t *testing.T) {
	var nilDocument *machineHelpRootDocument
	nilDocument.prepareJSONGroups()

	document := &machineHelpRootDocument{Commands: []machineHelpCommandSummary{
		{Name: "configure"},
		{Group: "utils", Name: "utils tool", Aliases: []string{"tool"}},
		{Group: "extension", Name: "completion"},
	}}
	document.prepareJSONGroups()
	require.Len(t, document.CoreCommands, 1)
	require.Len(t, document.Utilities, 1)
	require.Len(t, document.Extensions, 1)
	assert.Nil(t, document.Utilities[0].Aliases)

	searchDocument := &machineHelpRootDocument{Query: "demo", Commands: []machineHelpCommandSummary{{Name: "configure"}}}
	searchDocument.prepareJSONGroups()
	assert.Nil(t, searchDocument.CoreCommands)

	cleaned, keep := pruneMachineHelpEmpty(map[string]any{
		"empty": "", "false": false, "zero": json.Number("0"),
		"array": []any{"", nil, map[string]any{"value": "kept"}},
	})
	assert.True(t, keep)
	encoded, err := json.Marshal(cleaned)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"false":false`)
	assert.NotContains(t, string(encoded), `"empty"`)
	assert.Contains(t, string(encoded), `"array":["",{"value":"kept"}]`)

	_, keep = pruneMachineHelpEmpty([]any{})
	assert.False(t, keep)
	_, keep = pruneMachineHelpEmpty(map[string]any{})
	assert.False(t, keep)
	_, keep = pruneMachineHelpEmpty(nil)
	assert.False(t, keep)
}

func TestPrintMachineHelpEncodingFailureIsStructured(t *testing.T) {
	c, ctx, _, stderr := newCanonicalHelpTestContext(t)
	ctx = cli.NewCommandContext(alwaysFailWriter{}, stderr)
	ctx.EnterCommand(testMachineHelpRootCommand())
	err := c.printMachineHelp(ctx, nil, "json", helpOptions{})
	var structured *machineHelpError
	require.ErrorAs(t, err, &structured)
	assert.Equal(t, "MACHINE_HELP_UNAVAILABLE", structured.document.Error.Code)
	assert.True(t, strings.Contains(structured.Error(), "write failed"))
}

func TestMachineHelpOperationAndVersionSelectionEdgeCases(t *testing.T) {
	assert.Equal(t, machineHelpOperation{}, projectMachineHelpOperation(nil))
	legacy := projectMachineHelpOperation(&canonicalmeta.API{Method: "GET", Protocol: "HTTPS", PathPattern: "/things"})
	assert.Equal(t, "GET", legacy.Method)
	assert.Equal(t, "HTTPS", legacy.Protocol)
	assert.Equal(t, "/things", legacy.URL)

	full := projectMachineHelpOperation(&canonicalmeta.API{Operation: &canonicalmeta.Operation{
		Action: "CreateThing", APIStyle: "ROA", APIVersion: "v1", Method: "POST",
		Protocol: "HTTPS", URL: "/things", IsSSE: true, ReqBodyType: "json",
		ContentType: "application/json", HasWildcardPath: true,
	}})
	assert.Equal(t, "CreateThing", full.Action)
	assert.True(t, full.IsSSE)
	assert.True(t, full.HasWildcardPath)

	product := canonicalmeta.ProductEntry{Code: "demo", PluginDefaultVersion: "v2", Version: "v1"}
	version, err := selectProductVersion(product, []string{"v1", "v2"}, "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1", version)
	_, err = selectProductVersion(product, []string{"v1", "v2"}, "v3")
	assert.Error(t, err)
	version, err = selectProductVersion(product, []string{"v1", "v2"}, "")
	require.NoError(t, err)
	assert.Equal(t, "v2", version)
	product.PluginDefaultVersion = ""
	version, err = selectProductVersion(product, []string{"v1"}, "")
	require.NoError(t, err)
	assert.Equal(t, "v1", version)
	product.Version = ""
	version, err = selectProductVersion(product, []string{"v0"}, "")
	require.NoError(t, err)
	assert.Equal(t, "v0", version)
	_, err = selectProductVersion(product, nil, "")
	assert.Error(t, err)
}

func TestUnknownRootOptionCoversLongShortAndKnownFlags(t *testing.T) {
	assert.Empty(t, unknownRootOption(nil, []string{"--bad"}))
	ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	root := &cli.Command{Name: "aliyun"}
	root.Flags().Add(&cli.Flag{Name: "known", Shorthand: 'k'})
	ctx.EnterCommand(root)

	assert.Empty(t, unknownRootOption(ctx, []string{"--", "value", "--known", "-k"}))
	assert.Equal(t, "--help=all", unknownRootOption(ctx, []string{"--help=all"}))
	assert.Equal(t, "--missing", unknownRootOption(ctx, []string{"--missing=value"}))
	assert.Equal(t, "-x", unknownRootOption(ctx, []string{"-x"}))
}
