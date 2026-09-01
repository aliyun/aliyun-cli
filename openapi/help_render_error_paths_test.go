package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failAfterWrites struct {
	remaining int
}

func (w *failAfterWrites) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, errors.New("forced write failure")
	}
	w.remaining--
	return len(p), nil
}

func exerciseWriterFailures(t *testing.T, render func(*failAfterWrites) error) {
	t.Helper()
	failures := 0
	succeeded := false
	for writes := 0; writes < 500; writes++ {
		err := render(&failAfterWrites{remaining: writes})
		if err != nil {
			failures++
			continue
		}
		succeeded = true
		break
	}
	assert.True(t, succeeded, "renderer should eventually complete when enough writes are allowed")
	assert.GreaterOrEqual(t, failures, 3, "renderer should propagate failures from multiple write sites")
}

func completeParameterHelpForRendering() machineHelpParameter {
	constraints := machineHelpConstraints{
		Enum: []string{"one", "two"}, Pattern: "^value$",
		Minimum: "1", Maximum: "9", MinLength: "1", MaxLength: "16",
	}
	leaf := machineHelpParameter{
		Name: "child", RawName: "Child", Options: []string{"--child"},
		Type: "string", Location: "body", Required: true,
		Serialization: "json", Help: machineHelpLocalizedText{EN: "child help"},
		Example: "value", Constraints: constraints,
	}
	shape := &machineHelpShape{
		Type: "object", Format: "map", Constraints: constraints,
		Fields:  []machineHelpParameter{leaf},
		Element: &machineHelpShape{Type: "string", Constraints: constraints},
		Value:   &machineHelpShape{Type: "integer", Format: "int64"},
	}
	return machineHelpParameter{
		Name: "config", RawName: "Config", Options: []string{"--config"},
		Type: "object", Location: "query", Required: true,
		Serialization: "json", Help: machineHelpLocalizedText{EN: "config help"},
		Example: "{}", Constraints: constraints, Fields: []machineHelpParameter{leaf},
		Element: shape, Value: shape,
	}
}

func TestParameterHelpRenderersPropagateEveryWriterFailure(t *testing.T) {
	parameter := completeParameterHelpForRendering()

	t.Run("default document", func(t *testing.T) {
		document := &machineHelpParameterDocument{
			Parameter: parameter,
			Result:    &HelpResult{Shown: 1, Total: 2, Truncated: true},
		}
		exerciseWriterFailures(t, func(w *failAfterWrites) error {
			return renderParameterHelpText(w, document)
		})
	})

	t.Run("search document", func(t *testing.T) {
		document := &machineHelpParameterDocument{
			Query: "child", Parameter: parameter,
			Matches: []machineHelpParameterMatch{{Path: "Config.Child", Parameter: parameter}},
			Result:  &HelpResult{Shown: 1, Total: 1},
		}
		exerciseWriterFailures(t, func(w *failAfterWrites) error {
			return renderParameterHelpText(w, document)
		})
	})

	t.Run("no search matches", func(t *testing.T) {
		var output bytes.Buffer
		require.NoError(t, renderParameterHelpText(&output, &machineHelpParameterDocument{Query: "absent"}))
		assert.Contains(t, output.String(), "No Help entries matched")
	})

	assert.Error(t, renderParameterHelpText(&bytes.Buffer{}, nil))
	assert.NoError(t, renderMachineHelpShapeNode(&bytes.Buffer{}, nil, nil, 0))
}

func TestParameterHelpBuilderAndWalkerEdgeCases(t *testing.T) {
	assert.Error(t, func() error { _, err := buildParameterHelpDocument(nil, "--id"); return err }())
	assert.Error(t, func() error { _, err := buildParameterHelpDocument(&machineHelpAPIDocument{}, " "); return err }())

	action := &machineHelpAPIDocument{
		Target: machineHelpTarget{Path: []string{"aliyun", "demo", "create"}},
		API:    machineHelpAPI{Operation: machineHelpOperation{APIVersion: "2026-01-01"}},
		ParameterSets: machineHelpParameterSets{Camel: []machineHelpParameter{
			{Options: []string{"--same"}}, {Options: []string{"--same"}},
		}},
		ActiveParameterSet: "camel",
	}
	_, err := buildParameterHelpDocument(action, "--same")
	assert.ErrorContains(t, err, "ambiguous")
	_, err = buildParameterHelpDocument(action, "--missing")
	assert.ErrorContains(t, err, "unknown parameter")

	visited := make([]string, 0)
	walkMachineHelpShape(&machineHelpShape{
		Fields:  []machineHelpParameter{{Name: "field"}},
		Element: &machineHelpShape{Fields: []machineHelpParameter{{RawName: "ElementField"}}},
		Value:   &machineHelpShape{Fields: []machineHelpParameter{{Name: "value_field"}}},
	}, []string{"Root"}, func(path string, _ machineHelpParameter) {
		visited = append(visited, path)
	})
	assert.ElementsMatch(t, []string{"Root.field", "Root.ElementField", "Root.value_field"}, visited)
	walkMachineHelpShape(nil, nil, func(string, machineHelpParameter) { t.Fatal("nil shape must not visit") })

	assert.Equal(t, "--a", helpParameterDisplayName(machineHelpParameter{Options: []string{"--z", "--a"}}))
	assert.Equal(t, "name", helpParameterDisplayName(machineHelpParameter{Name: "name", RawName: "raw"}))
	assert.Equal(t, "raw", helpParameterDisplayName(machineHelpParameter{RawName: "raw"}))
}

func completeResponseHelpDocument(useResponses bool) *machineHelpAPIResponseDocument {
	components := &machineHelpComponents{Schemas: map[string]json.RawMessage{
		"Item": json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}}}`),
	}}
	document := &machineHelpAPIResponseDocument{
		Provider: "aliyun-cli-demo", Components: components,
		Warnings:      []string{"first warning", "second warning"},
		ResponseQuery: &machineHelpQueryExample{QueryCommand: "aliyun demo list --cli-query Items"},
		Notice:        "notice", Result: HelpResult{Shown: 1, Total: 2, Truncated: true},
		Next: &HelpNext{ShowAll: "aliyun demo --help-all"},
	}
	if useResponses {
		document.Responses = json.RawMessage(`{"200":{"description":"ok","schema":{"$ref":"#/components/schemas/Item"}}}`)
		return document
	}
	document.Matches = []string{"Items", "Items.Id"}
	document.OutputSchema = &machineHelpOutputSchema{
		StatusCode: "200", ContentType: "application/json",
		Schema:     json.RawMessage(`{"type":"array","items":{"$ref":"#/components/schemas/Item"}}`),
		Components: components,
	}
	return document
}

func TestResponseHelpRenderersPropagateWriterAndJSONErrors(t *testing.T) {
	for _, useResponses := range []bool{true, false} {
		document := completeResponseHelpDocument(useResponses)
		t.Run(map[bool]string{true: "responses", false: "selected schema"}[useResponses], func(t *testing.T) {
			exerciseWriterFailures(t, func(w *failAfterWrites) error {
				return renderResponseHelpText(w, document)
			})
		})
	}

	assert.Error(t, renderResponseHelpText(&bytes.Buffer{}, nil))
	assert.Error(t, renderResponseHelpText(&bytes.Buffer{}, &machineHelpAPIResponseDocument{
		Responses: json.RawMessage(`{broken`),
	}))
	assert.Error(t, renderResponseHelpText(&bytes.Buffer{}, &machineHelpAPIResponseDocument{
		Matches:      []string{"Items"},
		OutputSchema: &machineHelpOutputSchema{StatusCode: "200", Schema: json.RawMessage(`{broken`)},
	}))
	assert.Error(t, writeIndentedJSON(&bytes.Buffer{}, json.RawMessage(`{broken`)))
}

func TestResponseHelpSmallHelpers(t *testing.T) {
	assert.Equal(t, []string{"one", "two", "three"}, mergeMachineHelpWarnings(
		[]string{"", "one", "two"}, []string{"two", "three"},
	))
	assert.Empty(t, mergeMachineHelpWarnings())
	assert.Equal(t, HelpResult{}, completeMachineHelpJSONResult(json.RawMessage(`[]`)))
	assert.Equal(t, HelpResult{Shown: 2, Total: 2}, completeMachineHelpJSONResult(json.RawMessage(`{"a":1,"b":2}`)))

	var output bytes.Buffer
	require.NoError(t, renderMachineHelpResponseWarnings(&output, nil))
	require.NoError(t, renderMachineHelpResponseQuery(&output, nil))
	require.NoError(t, renderMachineHelpComponents(&output, nil))
	require.NoError(t, renderMachineHelpComponents(&output, &machineHelpComponents{}))
}
