package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustResponseSchemaNode(t *testing.T, raw string) *responseSchemaNode {
	t.Helper()
	node, err := parseResponseSchemaNode(json.RawMessage(raw))
	require.NoError(t, err)
	return node
}

func TestResponseSchemaSearchStatePathOrderingAndDeduplication(t *testing.T) {
	state := newResponseSchemaSearchState(&responseSchemaDocument{}, newHelpSearchText("item"))
	state.addPath("", HelpSearchTextContains)
	state.addPath("Items.Zeta", HelpSearchTextContains)
	state.addPath("Items.Alpha", HelpSearchTextContains)
	state.addPath("Items.Zeta", HelpSearchExactName)
	state.addPath("Items.Beta", HelpSearchNameTokenPrefix)

	assert.Equal(t, []string{"Items.Zeta", "Items.Beta", "Items.Alpha"}, state.sortedPaths())
	assert.Len(t, state.sortedPathMatches(), 3)
	assert.Nil(t, state.keepFor(nil))
	require.NotNil(t, state.keepFor(mustResponseSchemaNode(t, `{"type":"string"}`)))
}

func TestResponseSchemaSearchStateMarksArraysCompositionsAndComponents(t *testing.T) {
	componentA := mustResponseSchemaNode(t, `{"type":"object","properties":{"b":{"$ref":"#/components/schemas/B"}}}`)
	componentB := mustResponseSchemaNode(t, `{"type":"object","properties":{"a":{"$ref":"#/components/schemas/A"}}}`)
	document := &responseSchemaDocument{components: map[string]*responseSchemaNode{"A": componentA, "B": componentB}}
	state := newResponseSchemaSearchState(document, newHelpSearchText("value"))

	state.markFullComponent("")
	state.markFullComponent("Missing")
	state.markFullComponent("A")
	assert.True(t, state.fullComponents["A"])
	assert.True(t, state.fullComponents["B"])
	state.markFullComponent("A")

	composition := mustResponseSchemaNode(t, `{"allOf":[{"type":"string"}]}`)
	state.markNodeSelf(composition, map[string]bool{})
	assert.True(t, state.fullNodes[composition])

	array := mustResponseSchemaNode(t, `{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"}}}}`)
	state.markNodeSelf(array, map[string]bool{})
	assert.True(t, state.keep[array].items)
	assert.True(t, state.fullNodes[array.items])

	ref := mustResponseSchemaNode(t, `{"$ref":"#/components/schemas/A"}`)
	state.markNodeSelf(ref, map[string]bool{})
	assert.NotNil(t, state.keep[componentA])
	state.markNodeSelf(ref, map[string]bool{"A": true})
	state.markNodeSelf(nil, nil)
	state.markFullNode(nil)
	state.markFullNode(componentA)
}

func TestResponseSchemaNodeParsingEdgeShapes(t *testing.T) {
	primitive, err := parseResponseSchemaNode(json.RawMessage(`true`))
	require.NoError(t, err)
	assert.Empty(t, primitive.fields)

	node, err := parseResponseSchemaNode(json.RawMessage(`{
		"title":123,
		"description":"useful",
		"properties":[],
		"items":"not-a-schema",
		"allOf":{"type":"string"},
		"oneOf":[{"type":"string"}],
		"anyOf":[true]
	}`))
	require.NoError(t, err)
	assert.Equal(t, []string{"useful"}, node.describes)
	assert.Empty(t, node.properties)
	assert.Nil(t, node.items)
	assert.Len(t, node.compositions["oneOf"], 1)
	assert.Len(t, node.compositions["anyOf"], 1)

	_, err = parseResponseSchemaNode(json.RawMessage(`{broken`))
	assert.ErrorContains(t, err, "invalid JSON")
	fields, object, err := decodeOrderedRawObject(json.RawMessage(`[]`))
	require.NoError(t, err)
	assert.False(t, object)
	assert.Nil(t, fields)

	assert.False(t, responseSchemaNodeIsArray(nil))
	assert.True(t, responseSchemaNodeIsArray(&responseSchemaNode{items: &responseSchemaNode{}}))
	assert.Equal(t, []string{"A", "B"}, collectLocalResponseRefs(json.RawMessage(`{
		"one":{"$ref":"#/components/schemas/B"},
		"two":{"$ref":"#/components/schemas/A"},
		"duplicate":{"$ref":"#/components/schemas/A"}
	}`)))
	assert.Nil(t, collectLocalResponseRefs(json.RawMessage(`{broken`)))
}

func TestMachineHelpLogicalLineCountingNestedShapes(t *testing.T) {
	shape := &machineHelpShape{
		Fields: []machineHelpParameter{{
			Name:   "field",
			Fields: []machineHelpParameter{{Name: "nested"}},
		}},
		Element: &machineHelpShape{Fields: []machineHelpParameter{{Name: "element"}}},
		Value:   &machineHelpShape{Fields: []machineHelpParameter{{Name: "value"}}},
	}
	parameter := machineHelpParameter{
		Fields:  []machineHelpParameter{{Name: "direct"}},
		Element: shape,
		Value:   shape,
	}
	assert.Equal(t, 10, machineHelpParameterLogicalLines(parameter))
	assert.Equal(t, 0, machineHelpShapeLogicalLines(nil))
}
