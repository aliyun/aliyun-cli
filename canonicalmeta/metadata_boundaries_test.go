package canonicalmeta

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseMetadataReportsMalformedSelectedResponse(t *testing.T) {
	for _, responses := range []string{`{"200":5}`, `{"200":{"content":{"application/json":5}}}`} {
		_, err := (&API{Responses: json.RawMessage(responses)}).ResponseSchema()
		var schemaErr *ResponseSchemaError
		require.ErrorAs(t, err, &schemaErr)
		require.Equal(t, "responses", schemaErr.Field)
		require.NotNil(t, schemaErr.Unwrap())
	}
	_, err := (&API{Responses: json.RawMessage("5")}).ResponseSection()
	var schemaErr *ResponseSchemaError
	require.ErrorAs(t, err, &schemaErr)
	require.Equal(t, "responses", schemaErr.Field)
	_, _, err = schemaReferences(json.RawMessage("{"))
	require.Error(t, err)
}

func TestResponseMetadataRetainsSharedComponentOnlyOnce(t *testing.T) {
	api := &API{
		Responses:  json.RawMessage(`{"200":{"schema":{"type":"object","properties":{"first":{"$ref":"#/components/schemas/Shared"},"second":{"$ref":"#/components/schemas/Shared"}}}}}`),
		Components: json.RawMessage(`{"schemas":{"Shared":{"type":"string"}}}`),
	}
	schema, err := api.ResponseSchema()
	require.NoError(t, err)
	require.Len(t, schema.Components, 1)
	require.Empty(t, schema.Warnings)
	section, err := api.ResponseSection()
	require.NoError(t, err)
	require.Len(t, section.Components, 1)
	require.Empty(t, section.Warnings)
}

func TestLegacyParameterProjectionDeduplicatesAndHonorsBodyOverrides(t *testing.T) {
	v1 := []V1Parameter{{Name: "Action", Type: "String", Required: true}, {Name: "Id", Position: "path", Type: "String", Required: true}, {Name: "Id", Position: "query", Type: "String", Required: true}}
	api := &API{V1Parameters: &v1, Parameters: []Parameter{{RawName: "Id", IsWildcard: true}}}
	projected := api.LegacyTopLevelParameters()
	require.Len(t, projected, 1)
	require.Equal(t, "Id", projected[0].LegacyName())
	require.True(t, projected[0].IsWildcard())
	var checked []string
	require.NoError(t, api.CheckLegacyRequiredParameters(func(name string) bool { checked = append(checked, name); return true }))
	require.Equal(t, []string{"Id"}, checked)
	body := []V1Parameter{{Name: "Shared", Position: "body", Type: "String", Required: true}, {Name: "NewBody", Position: "body", Type: "String"}}
	api = &API{Parameters: []Parameter{
		{RawName: "Action", Location: "query", Required: true},
		{RawName: "Shared", Location: "query", Required: true},
		{RawName: "Duplicate", Location: "query", Required: true},
		{RawName: "Duplicate", Location: "query", Required: true},
		{RawName: "HiddenBody", Location: "body", Required: true},
	}, V1BodyParameters: &body}
	projected = api.LegacyTopLevelParameters()
	var names []string
	for _, parameter := range projected {
		names = append(names, parameter.LegacyName())
		if parameter.LegacyName() == "Shared" {
			require.Equal(t, "Body", parameter.LegacyPosition())
		}
	}
	require.ElementsMatch(t, []string{"Shared", "NewBody", "Duplicate"}, names)
	checked = nil
	require.NoError(t, api.CheckLegacyRequiredParameters(func(name string) bool { checked = append(checked, name); return true }))
	require.Equal(t, []string{"Duplicate", "Shared"}, checked)
}

func TestLegacyRequiredBodyParametersAreCheckedOnce(t *testing.T) {
	body := []V1Parameter{{Name: "Id", Required: true}, {Name: "Id", Required: true}}
	api := &API{V1BodyParameters: &body}
	var checked []string
	require.NoError(t, api.CheckLegacyRequiredParameters(func(name string) bool {
		checked = append(checked, name)
		return true
	}))
	require.Equal(t, []string{"Id"}, checked)
	view := &LegacyParameterView{source: SourceCanonical, canonical: &Parameter{Location: "query"}, resolvedPosition: "Body"}
	require.Equal(t, "Body", view.LegacyPosition())
	require.Equal(t, "https", (&API{Protocol: "http|https"}).GetProtocol())
	require.Nil(t, (&API{Parameters: []Parameter{{RawName: "Id", Location: "query", Type: "string"}}}).FindLegacyParameter("Missing"))
}
