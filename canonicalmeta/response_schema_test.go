package canonicalmeta

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIResponseMetadataStaysRawUntilProjected(t *testing.T) {
	var api API
	err := json.Unmarshal([]byte(`{
		"name": "DescribeThings",
		"responses": {"200": "not-yet-decoded"},
		"components": {"schemas": []}
	}`), &api)
	require.NoError(t, err)
	require.JSONEq(t, `{"200":"not-yet-decoded"}`, string(api.Responses))
	require.JSONEq(t, `{"schemas":[]}`, string(api.Components))
}

func TestResponseSchemaSelectsSuccessStatus(t *testing.T) {
	tests := []struct {
		name       string
		responses  string
		wantStatus string
		wantSchema string
	}{
		{
			name: "200 before other success and default",
			responses: `{
				"201":{"schema":{"title":"created"}},
				"200":{"schema":{"title":"ok"}},
				"default":{"schema":{"title":"fallback"}}
			}`,
			wantStatus: "200",
			wantSchema: `{"title":"ok"}`,
		},
		{
			name: "smallest numeric 2xx",
			responses: `{
				"299":{"schema":{"title":"last"}},
				"204":{"schema":{"title":"empty"}},
				"201":{"schema":{"title":"first"}}
			}`,
			wantStatus: "201",
			wantSchema: `{"title":"first"}`,
		},
		{
			name:       "default fallback",
			responses:  `{"400":{"schema":{"title":"bad"}},"default":{"schema":{"title":"fallback"}}}`,
			wantStatus: "default",
			wantSchema: `{"title":"fallback"}`,
		},
		{
			name:       "error responses are not response help",
			responses:  `{"400":{"schema":{"title":"bad"}},"503":{"schema":{"title":"down"}}}`,
			wantStatus: "",
			wantSchema: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := (&API{Responses: rawJSON(tt.responses)}).ResponseSchema()
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, doc.StatusCode)
			if tt.wantSchema == "" {
				require.False(t, doc.HasSchema())
				return
			}
			require.True(t, doc.HasSchema())
			require.JSONEq(t, tt.wantSchema, string(doc.Schema))
		})
	}
}

func TestResponseSchemaSelectsContentType(t *testing.T) {
	tests := []struct {
		name            string
		response        string
		wantContentType string
		wantSchema      string
	}{
		{
			name: "application json first",
			response: `{"content":{
				"text/plain":{"schema":{"title":"text"}},
				"application/problem+json":{"schema":{"title":"problem"}},
				"application/json":{"schema":{"title":"json"}}
			}}`,
			wantContentType: "application/json",
			wantSchema:      `{"title":"json"}`,
		},
		{
			name: "json suffix before other media and sorted stably",
			response: `{"content":{
				"text/plain":{"schema":{"title":"text"}},
				"application/z+json":{"schema":{"title":"z"}},
				"application/a+json":{"schema":{"title":"a"}}
			}}`,
			wantContentType: "application/a+json",
			wantSchema:      `{"title":"a"}`,
		},
		{
			name: "other media type sorted stably",
			response: `{"content":{
				"text/plain":{"schema":{"title":"text"}},
				"application/xml":{"schema":{"title":"xml"}}
			}}`,
			wantContentType: "application/xml",
			wantSchema:      `{"title":"xml"}`,
		},
		{
			name: "media without schema is skipped",
			response: `{"content":{
				"application/json":{"example":{"ok":true}},
				"application/problem+json":{"schema":{"title":"problem"}}
			}}`,
			wantContentType: "application/problem+json",
			wantSchema:      `{"title":"problem"}`,
		},
		{
			name:            "legacy direct schema",
			response:        `{"content":{"application/json":{"example":{}}},"schema":{"type":"array"}}`,
			wantContentType: "",
			wantSchema:      `{"type":"array"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := (&API{Responses: rawJSON(`{"200":` + tt.response + `}`)}).ResponseSchema()
			require.NoError(t, err)
			require.True(t, doc.HasSchema())
			require.Equal(t, "200", doc.StatusCode)
			require.Equal(t, tt.wantContentType, doc.ContentType)
			require.JSONEq(t, tt.wantSchema, string(doc.Schema))
		})
	}
}

func TestResponseSchemaReportsNoSchemaWithoutFallingThrough(t *testing.T) {
	tests := []struct {
		name       string
		responses  json.RawMessage
		wantStatus string
	}{
		{name: "responses absent"},
		{name: "responses empty", responses: rawJSON(`{}`)},
		{name: "no success response", responses: rawJSON(`{"404":{"schema":{"type":"object"}}}`)},
		{
			name:       "selected response only has headers",
			responses:  rawJSON(`{"200":{"headers":{"x-request-id":{"schema":{"type":"string"}}}},"201":{"schema":{"type":"object"}}}`),
			wantStatus: "200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := (&API{Responses: tt.responses}).ResponseSchema()
			require.NoError(t, err)
			require.False(t, doc.HasSchema())
			require.Equal(t, tt.wantStatus, doc.StatusCode)
			require.Empty(t, doc.ContentType)
			require.Empty(t, doc.Schema)
			require.Empty(t, doc.Components)
		})
	}
}

func TestResponseSchemaIncludesOnlyReachableComponents(t *testing.T) {
	api := &API{
		Responses: rawJSON(`{"200":{"content":{"application/json":{"schema":{
			"type":"object",
			"properties":{"result":{"$ref":"#/components/schemas/Result"}}
		}}}}}`),
		Components: rawJSON(`{"schemas":{
			"Result":{"type":"object","properties":{"item":{"$ref":"#/components/schemas/Item"}}},
			"Item":{"type":"object","properties":{"id":{"type":"string"}}},
			"Unreachable":{"type":"object"}
		}}`),
	}

	doc, err := api.ResponseSchema()
	require.NoError(t, err)
	require.True(t, doc.HasSchema())
	require.Len(t, doc.Components, 2)
	require.Contains(t, doc.Components, "Result")
	require.Contains(t, doc.Components, "Item")
	require.NotContains(t, doc.Components, "Unreachable")
	require.JSONEq(t, `{"type":"object","properties":{"result":{"$ref":"#/components/schemas/Result"}}}`, string(doc.Schema))
	require.JSONEq(t, `{"type":"object","properties":{"item":{"$ref":"#/components/schemas/Item"}}}`, string(doc.Components["Result"]))
	require.Empty(t, doc.Warnings)
}

func TestResponseSchemaProtectsCyclicRefs(t *testing.T) {
	tests := []struct {
		name       string
		root       string
		components string
		wantNames  []string
	}{
		{
			name:       "self reference",
			root:       `{"$ref":"#/components/schemas/Node"}`,
			components: `{"schemas":{"Node":{"type":"object","properties":{"next":{"$ref":"#/components/schemas/Node"}}}}}`,
			wantNames:  []string{"Node"},
		},
		{
			name: "bidirectional cycle",
			root: `{"$ref":"#/components/schemas/A"}`,
			components: `{"schemas":{
				"A":{"properties":{"b":{"$ref":"#/components/schemas/B"}}},
				"B":{"properties":{"a":{"$ref":"#/components/schemas/A"}}}
			}}`,
			wantNames: []string{"A", "B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &API{
				Responses:  rawJSON(`{"200":{"schema":` + tt.root + `}}`),
				Components: rawJSON(tt.components),
			}
			doc, err := api.ResponseSchema()
			require.NoError(t, err)
			require.Len(t, doc.Components, len(tt.wantNames))
			for _, name := range tt.wantNames {
				require.Contains(t, doc.Components, name)
			}
			require.Contains(t, strings.Join(doc.Warnings, "\n"), "cyclic")
		})
	}
}

func TestResponseSchemaWarnsForMissingAndNonLocalRefs(t *testing.T) {
	api := &API{
		Responses: rawJSON(`{"200":{"schema":{"allOf":[
			{"$ref":"#/components/schemas/Present"},
			{"$ref":"#/components/schemas/Missing"},
			{"$ref":"https://example.com/schema.json"}
		]}}}`),
		Components: rawJSON(`{"schemas":{"Present":{"type":"object"},"Unused":{"type":"object"}}}`),
	}

	doc, err := api.ResponseSchema()
	require.NoError(t, err)
	require.Len(t, doc.Components, 1)
	require.Contains(t, doc.Components, "Present")
	require.NotContains(t, doc.Components, "Missing")
	warnings := strings.Join(doc.Warnings, "\n")
	require.Contains(t, warnings, "#/components/schemas/Missing")
	require.Contains(t, warnings, "https://example.com/schema.json")
}

func TestResponseSchemaReturnsTypedErrorsForMalformedMetadata(t *testing.T) {
	tests := []struct {
		name      string
		api       *API
		wantField string
	}{
		{
			name:      "responses json",
			api:       &API{Responses: rawJSON(`{"200":`)},
			wantField: "responses",
		},
		{
			name:      "selected response shape",
			api:       &API{Responses: rawJSON(`{"200":"not-an-object"}`)},
			wantField: "responses",
		},
		{
			name: "components shape for referenced schema",
			api: &API{
				Responses:  rawJSON(`{"200":{"schema":{"$ref":"#/components/schemas/Result"}}}`),
				Components: rawJSON(`{"schemas":[]}`),
			},
			wantField: "components",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.api.ResponseSchema()
			require.Error(t, err)
			var metaErr *ResponseSchemaError
			require.True(t, errors.As(err, &metaErr))
			require.Equal(t, tt.wantField, metaErr.Field)
			require.Error(t, metaErr.Unwrap())
		})
	}
}

func TestResponseSchemaDoesNotDecodeUnneededComponents(t *testing.T) {
	api := &API{
		Responses:  rawJSON(`{"200":{"schema":{"type":"object","properties":{"ok":{"type":"boolean"}}}}}`),
		Components: rawJSON(`{"schemas":`),
	}

	doc, err := api.ResponseSchema()
	require.NoError(t, err)
	require.True(t, doc.HasSchema())
	require.Empty(t, doc.Components)
	require.Empty(t, doc.Warnings)
}

func rawJSON(value string) json.RawMessage {
	return json.RawMessage(value)
}
