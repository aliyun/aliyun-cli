package canonicalmeta

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseSectionKeepsAllResponsesAndReachableComponents(t *testing.T) {
	api := &API{
		Responses: rawJSON(`{
			"200":{"schema":{"$ref":"#/components/schemas/Result"}},
			"400":{"schema":{"$ref":"#/components/schemas/Error"}}
		}`),
		Components: rawJSON(`{"schemas":{
			"Result":{"properties":{"item":{"$ref":"#/components/schemas/Item"}}},
			"Item":{"properties":{"parent":{"$ref":"#/components/schemas/Result"}}},
			"Error":{"type":"object"},
			"Unused":{"type":"object"}
		}}`),
	}

	document, err := api.ResponseSection()
	require.NoError(t, err)
	require.JSONEq(t, string(api.Responses), string(document.Responses))
	assert.Len(t, document.Components, 3)
	assert.Contains(t, document.Components, "Result")
	assert.Contains(t, document.Components, "Item")
	assert.Contains(t, document.Components, "Error")
	assert.NotContains(t, document.Components, "Unused")
	assert.Contains(t, strings.Join(document.Warnings, "\n"), "cyclic")
}

func TestResponseSectionEmptyInlineAndWarningCases(t *testing.T) {
	for _, api := range []*API{nil, {}, {Responses: rawJSON(`null`)}} {
		document, err := api.ResponseSection()
		require.NoError(t, err)
		assert.Empty(t, document.Responses)
	}

	inline := &API{Responses: rawJSON(`{"200":{"schema":{"type":"string"}}}`), Components: rawJSON(`{broken`)}
	document, err := inline.ResponseSection()
	require.NoError(t, err, "components stay lazy when responses have no references")
	assert.NotEmpty(t, document.Responses)
	assert.Empty(t, document.Components)

	warnings := &API{
		Responses: rawJSON(`{"200":{"schema":{"allOf":[
			{"$ref":"#/components/schemas/Missing"},
			{"$ref":"https://example.com/remote.json"},
			{"$ref":"#/components/schemas/Missing"}
		]}}}`),
		Components: rawJSON(`{"schemas":{}}`),
	}
	document, err = warnings.ResponseSection()
	require.NoError(t, err)
	joined := strings.Join(document.Warnings, "\n")
	assert.Contains(t, joined, "Missing")
	assert.Contains(t, joined, "non-local")
	assert.Equal(t, 2, len(document.Warnings), "duplicate warnings should be collapsed")
}

func TestResponseSectionReturnsTypedErrorsForMalformedMetadata(t *testing.T) {
	tests := []struct {
		name      string
		api       *API
		wantField string
	}{
		{name: "responses object", api: &API{Responses: rawJSON(`{broken`)}, wantField: "responses"},
		{name: "response traversal", api: &API{Responses: rawJSON(`{"200":{"schema":`)}, wantField: "responses"},
		{
			name: "components object",
			api: &API{
				Responses:  rawJSON(`{"200":{"schema":{"$ref":"#/components/schemas/Result"}}}`),
				Components: rawJSON(`{"schemas":[]}`),
			},
			wantField: "components",
		},
		{
			name: "component traversal",
			api: &API{
				Responses:  rawJSON(`{"200":{"schema":{"$ref":"#/components/schemas/Result"}}}`),
				Components: rawJSON(`{"schemas":{"Result":{"properties":`),
			},
			wantField: "components",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.api.ResponseSection()
			require.Error(t, err)
			var typed *ResponseSchemaError
			require.True(t, errors.As(err, &typed))
			assert.Equal(t, tc.wantField, typed.Field)
			assert.Contains(t, typed.Error(), "parse canonical response "+tc.wantField+" failed")
			assert.Error(t, typed.Unwrap())
		})
	}
}

func TestResponseSchemaPointerAndWarningEdgeCases(t *testing.T) {
	assert.Equal(t, "A/B~C", func() string {
		value, ok := localComponentName("#/components/schemas/A~1B~0C/path")
		require.True(t, ok)
		return value
	}())
	_, ok := localComponentName("#/components/schemas/")
	assert.False(t, ok)
	_, ok = localComponentName("https://example.com/schema")
	assert.False(t, ok)

	warnings := newWarningCollector()
	warnings.add("same")
	warnings.add("same")
	warnings.add("")
	assert.Equal(t, []string{"same", ""}, warnings.values())
}
