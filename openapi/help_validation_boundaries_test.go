package openapi

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/require"
)

func TestRootHelpRejectsInconsistentRegistrations(t *testing.T) {
	for _, name := range []string{"nil root", "nil catalog", "nil description", "empty command path", "invalid command group", "duplicate command", "missing command", "hidden command", "unclassified utility", "duplicate flag", "invalid flag visibility", "missing flag", "hidden flag"} {
		t.Run(name, func(t *testing.T) {
			root := &cli.Command{Name: "aliyun", Short: i18n.T("CLI", "CLI")}
			root.AddSubCommand(&cli.Command{Name: "configure"})
			root.Flags().Add(&cli.Flag{Name: "region"})
			catalog := &canonicalmeta.ProductsIndex{}
			commands := []RootCommandSpec{{Path: []string{"configure"}, Group: RootGroupCore}}
			flags := []RootFlagSpec{{Name: "region", Visibility: RootVisibilityDefault}}
			var want string
			switch name {
			case "nil root":
				root, want = nil, "root command is nil"
			case "nil catalog":
				catalog, want = nil, "catalog is nil"
			case "nil description":
				root.Short, want = nil, "description is nil"
			case "empty command path":
				commands[0].Path, want = nil, "empty path"
			case "invalid command group":
				commands[0].Group, want = RootGroupProduct, "invalid Help group"
			case "duplicate command":
				commands, want = append(commands, commands[0]), "duplicated"
			case "missing command":
				commands[0].Path, want = []string{"absent"}, "not registered"
			case "hidden command":
				root.GetSubCommand("configure").Hidden, want = true, "hidden root command"
			case "unclassified utility":
				utils := &cli.Command{Name: "utils"}
				utils.AddSubCommand(&cli.Command{Name: "inspect"})
				root.AddSubCommand(utils)
				commands = append(commands, RootCommandSpec{Path: []string{"utils"}, Group: RootGroupCore})
				want = "visible utility command"
			case "duplicate flag":
				flags, want = append(flags, flags[0]), "duplicated"
			case "invalid flag visibility":
				flags[0].Visibility, want = "invalid", "invalid Help visibility"
			case "missing flag":
				flags[0].Name, want = "absent", "not registered"
			case "hidden flag":
				root.Flags().Get("region").Hidden, want = true, "hidden root flag"
			}
			_, err := BuildRootHelpInput(root, catalog, commands, flags)
			require.ErrorContains(t, err, want)
		})
	}
}

func TestOrderedResponseObjectRejectsMalformedAndTrailingValues(t *testing.T) {
	for _, input := range []string{"{", `{"x":}`, `{"x"}`, `{"x":1`, `{"x":1} {}`, `{"x":1} invalid`} {
		t.Run(input, func(t *testing.T) {
			_, _, err := decodeOrderedRawObject(json.RawMessage(input))
			require.Error(t, err)
		})
	}
	fields, object, err := decodeOrderedRawObject(json.RawMessage(`{"second":2,"first":1}`))
	require.NoError(t, err)
	require.True(t, object)
	require.Equal(t, []responseSchemaRawField{{name: "second", raw: json.RawMessage("2")}, {name: "first", raw: json.RawMessage("1")}}, fields)
}

func TestResponseSchemaSearchHandlesEmptyAndNonObjectSchemas(t *testing.T) {
	for _, input := range []HelpResponseSchema{{}, {Schema: json.RawMessage("null")}, {Schema: json.RawMessage("true")}, {Schema: json.RawMessage(`{"properties":false,"items":[],"allOf":false}`)}} {
		result, err := SearchResponseSchema(input, "identifier", false)
		require.NoError(t, err)
		require.Empty(t, result.Paths)
	}
	result, err := SearchResponseSchema(HelpResponseSchema{Schema: json.RawMessage(`{"type":"object"}`)}, " ", false)
	require.NoError(t, err)
	require.Empty(t, result.Paths)
	_, err = SearchResponseSchema(HelpResponseSchema{Schema: json.RawMessage("{")}, "id", false)
	require.ErrorContains(t, err, "parse response schema")
	_, err = SearchResponseSchema(HelpResponseSchema{Components: map[string]json.RawMessage{"Broken": json.RawMessage("{")}}, "id", false)
	require.ErrorContains(t, err, `component "Broken"`)
}

func TestUtilityHelpPropagatesFailuresFromEverySection(t *testing.T) {
	document, err := buildUtilityHelpDocument(utilityHelpRoot(), []string{"utils"})
	require.NoError(t, err)
	exerciseWriterFailures(t, func(w *failAfterWrites) error {
		return renderUtilityHelpText(cli.NewCommandContext(w, io.Discard), document)
	})
	for _, writes := range []int{0, 1} {
		ctx := cli.NewCommandContext(&endpointFailWriter{remaining: writes}, io.Discard)
		require.ErrorIs(t, renderUtilityHelpText(ctx, &machineHelpUtilityDocument{Query: "absent"}), io.ErrClosedPipe)
	}
}

func TestQueryOptionsPropagateMultilineWriteFailures(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "80")
	options := []machineHelpQueryOption{{Name: "--cli-query", Type: "string", Required: true, HasDefault: true, Default: "*", Help: machineHelpLocalizedText{EN: "A description which is long enough to require multiple output lines and preserve both paragraphs.\n\nThe second paragraph describes the query result."}}}
	exerciseWriterFailures(t, func(w *failAfterWrites) error {
		return renderMachineHelpQueryOptions(w, options)
	})
}
