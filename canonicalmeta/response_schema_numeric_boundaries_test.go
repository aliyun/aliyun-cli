package canonicalmeta

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseMetadataRejectsOutOfRangeNumbersWithSourceErrors(t *testing.T) {
	for _, tc := range []struct {
		name, responses, components, field string
	}{
		{
			name:      "response default",
			responses: `{"200":{"schema":{"type":"number","default":1e1000}}}`,
			field:     "responses",
		},
		{
			name:       "component default",
			responses:  `{"200":{"schema":{"$ref":"#/components/schemas/Root"}}}`,
			components: `{"schemas":{"Root":{"type":"number","default":1e1000}}}`,
			field:      "components",
		},
		{
			name:       "transitive component example",
			responses:  `{"200":{"schema":{"$ref":"#/components/schemas/Root"}}}`,
			components: `{"schemas":{"Root":{"properties":{"next":{"$ref":"#/components/schemas/Leaf"}}},"Leaf":{"type":"number","example":1e1000}}}`,
			field:      "components",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.True(t, json.Valid([]byte(tc.responses)))
			if tc.components != "" {
				require.True(t, json.Valid([]byte(tc.components)))
			}
			api := &API{Responses: rawJSON(tc.responses), Components: rawJSON(tc.components)}
			for _, section := range []bool{false, true} {
				var err error
				if section {
					var document ResponseSectionDocument
					document, err = api.ResponseSection()
					require.Equal(t, ResponseSectionDocument{}, document)
				} else {
					var document ResponseSchemaDocument
					document, err = api.ResponseSchema()
					require.Equal(t, ResponseSchemaDocument{}, document)
				}
				var schemaErr *ResponseSchemaError
				require.ErrorAs(t, err, &schemaErr)
				require.Equal(t, tc.field, schemaErr.Field)
				var numberErr *json.UnmarshalTypeError
				require.ErrorAs(t, schemaErr.Unwrap(), &numberErr)
				require.Equal(t, "float64", numberErr.Type.String())
				require.Contains(t, numberErr.Value, "1e1000")
			}
		})
	}
}

func TestResponseMetadataWarnsWhenReferencedComponentsAreAbsent(t *testing.T) {
	for _, components := range []json.RawMessage{nil, rawJSON("null"), rawJSON(" ")} {
		api := &API{
			Responses:  rawJSON(`{"200":{"schema":{"$ref":"#/components/schemas/Missing"}}}`),
			Components: components,
		}
		schema, err := api.ResponseSchema()
		require.NoError(t, err)
		require.True(t, schema.HasSchema())
		require.Empty(t, schema.Components)
		require.Equal(t, []string{`schema reference "#/components/schemas/Missing" was not found in components.schemas`}, schema.Warnings)
		section, err := api.ResponseSection()
		require.NoError(t, err)
		require.JSONEq(t, string(api.Responses), string(section.Responses))
		require.Empty(t, section.Components)
		require.Equal(t, schema.Warnings, section.Warnings)
	}
}

func TestResponseSchemaUsesDefaultForNonnumericSuccessStatus(t *testing.T) {
	api := &API{Responses: rawJSON(`{"2XX":{"schema":{"title":"wildcard"}},"default":{"schema":{"title":"fallback"}}}`)}
	document, err := api.ResponseSchema()
	require.NoError(t, err)
	require.Equal(t, "default", document.StatusCode)
	require.JSONEq(t, `{"title":"fallback"}`, string(document.Schema))
}

func TestLegacyProtocolAndRepeatedParameterPrefixes(t *testing.T) {
	api := &API{Protocol: "HTTPS", Parameters: []Parameter{{RawName: "Tags", Location: "query", Type: "array", Element: &TypeShape{Type: "string"}}}}
	require.Equal(t, "https", api.GetProtocol())
	require.Nil(t, api.FindLegacyParameter("TagsExtra"))
	require.NotNil(t, api.FindLegacyParameter("Tags.1"))
}
