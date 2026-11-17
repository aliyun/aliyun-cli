package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseQueryRecognizesArraysThroughCompositionsAndReferences(t *testing.T) {
	document, err := parseHelpResponseSchema(HelpResponseSchema{Schema: json.RawMessage(`{"properties":{"Items":{"allOf":[{"$ref":"#/components/schemas/Results"}]}}}`), Components: map[string]json.RawMessage{
		"Results": json.RawMessage(`{"type":"array","items":{"type":"string"}}`),
	}})
	require.NoError(t, err)
	array := document.root.properties[0].node
	require.True(t, responseNodeIsArray(document, array, map[string]bool{}))
	require.False(t, responseNodeIsArray(document, nil, map[string]bool{}))
	for _, schema := range []string{`{"type":"array"}`, `{"type":"array","items":{"$ref":"#/components/schemas/Missing"}}`, `{"$ref":"#/components/schemas/Loop"}`} {
		input := HelpResponseSchema{Schema: json.RawMessage(schema), Components: map[string]json.RawMessage{"Loop": json.RawMessage(`{"$ref":"#/components/schemas/Loop"}`)}}
		doc, err := parseHelpResponseSchema(input)
		require.NoError(t, err)
		require.Empty(t, responseArrayProjectionFields(doc, doc.root, "ListItems"))
	}
	require.Empty(t, responseArrayProjectionFields(nil, array, "ListItems"))
	require.Empty(t, responseArrayItemName(document, nil))
	require.False(t, responseSchemaPropertyIsScalar(document, nil))
	require.Empty(t, responseArrayItemProperties(document, nil))
	collector := responseArrayCollector{document: document}
	collector.walk(nil, nil, map[string]bool{})
	require.Empty(t, collector.paths)
}

func TestResponseQueryRefusesUnsafeCommandsAndUnknownStyles(t *testing.T) {
	input := HelpResponseSchema{Schema: json.RawMessage(`{"type":"array","items":{"type":"object","properties":{"Id":{"type":"string"}}}}`)}
	for _, context := range []ResponseQueryContext{
		{API: "", Style: ResponseCommandStyleCamel},
		{API: "---", Style: ResponseCommandStyleCamel},
		{API: "___", Style: ResponseCommandStyleKebab},
		{API: "ListItems", Style: "unsupported"},
		{API: "ListItems", Style: ResponseCommandStyleCamel, Product: "demo;rm"},
	} {
		context.Document = input
		result, err := BuildResponseQueryExample(context)
		require.NoError(t, err)
		require.Nil(t, result)
	}
	require.Equal(t, 0, responseProjectionFieldScore("---", []string{"item"}))
	require.False(t, responseArrayMatchesAPIResource(responseArrayPath{rawPath: "Items"}, "List"))
	require.Equal(t, "", firstRuneUpper(""))
	for plural, singular := range map[string]string{"branches": "branch", "bushes": "bush", "boxes": "box", "classes": "class"} {
		require.Equal(t, singular, singularResponseToken(plural))
	}
}

func TestResponseSearchKeepsRequiredAndReferenceBoundaries(t *testing.T) {
	properties := []responseSchemaProperty{{name: "Id"}, {name: "Name"}}
	require.Nil(t, filterResponseRequired(json.RawMessage(`["Id"]`), properties, nil))
	require.Nil(t, filterResponseRequired(json.RawMessage(`["Name"]`), properties, map[int]bool{0: true}))
	require.Equal(t, json.RawMessage(`true`), filterResponseRequired(json.RawMessage(`true`), properties, map[int]bool{0: true}))
	for _, ref := range []string{"#/components/schemas/", "#/components/schemas/has/path"} {
		_, ok := localResponseComponentName(ref)
		require.False(t, ok)
	}
	require.Empty(t, SearchHelpCandidates([]HelpSearchCandidate{{Name: "Id"}}, "---"))
	_, err := parseHelpResponseSchema(HelpResponseSchema{Components: map[string]json.RawMessage{"Invalid": json.RawMessage("{")}})
	require.ErrorContains(t, err, "response component")
}
