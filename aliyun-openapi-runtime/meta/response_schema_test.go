// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package meta

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestAPIResponseMetadataStaysRawUntilProjected(t *testing.T) {
	var api API
	err := json.Unmarshal([]byte(`{
		"name": "DescribeThings",
		"responses": {"200": "not-yet-decoded"},
		"components": {"schemas": []}
	}`), &api)
	requireNoError(t, err)
	requireJSONEqual(t, `{"200":"not-yet-decoded"}`, api.Responses)
	requireJSONEqual(t, `{"schemas":[]}`, api.Components)
}

func TestResponseSchemaSelectsSuccessStatus(t *testing.T) {
	tests := []struct {
		name, responses, wantStatus, wantSchema string
	}{
		{"200 before other success and default", `{
			"201":{"schema":{"title":"created"}},
			"200":{"schema":{"title":"ok"}},
			"default":{"schema":{"title":"fallback"}}
		}`, "200", `{"title":"ok"}`},
		{"smallest numeric 2xx", `{
			"299":{"schema":{"title":"last"}},
			"204":{"schema":{"title":"empty"}},
			"201":{"schema":{"title":"first"}}
		}`, "201", `{"title":"first"}`},
		{"default fallback", `{"400":{"schema":{"title":"bad"}},"default":{"schema":{"title":"fallback"}}}`, "default", `{"title":"fallback"}`},
		{"error responses are not response help", `{"400":{"schema":{"title":"bad"}},"503":{"schema":{"title":"down"}}}`, "", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := (&API{Responses: rawJSON(test.responses)}).ResponseSchema()
			requireNoError(t, err)
			if document.StatusCode != test.wantStatus {
				t.Fatalf("StatusCode = %q, want %q", document.StatusCode, test.wantStatus)
			}
			if test.wantSchema == "" {
				if document.HasSchema() {
					t.Fatal("HasSchema() = true, want false")
				}
				return
			}
			if !document.HasSchema() {
				t.Fatal("HasSchema() = false, want true")
			}
			requireJSONEqual(t, test.wantSchema, document.Schema)
		})
	}
}

func TestResponseSchemaSelectsContentType(t *testing.T) {
	tests := []struct {
		name, response, wantContentType, wantSchema string
	}{
		{"application json first", `{"content":{
			"text/plain":{"schema":{"title":"text"}},
			"application/problem+json":{"schema":{"title":"problem"}},
			"application/json":{"schema":{"title":"json"}}
		}}`, "application/json", `{"title":"json"}`},
		{"json suffix before other media and sorted stably", `{"content":{
			"text/plain":{"schema":{"title":"text"}},
			"application/z+json":{"schema":{"title":"z"}},
			"application/a+json":{"schema":{"title":"a"}}
		}}`, "application/a+json", `{"title":"a"}`},
		{"other media type sorted stably", `{"content":{
			"text/plain":{"schema":{"title":"text"}},
			"application/xml":{"schema":{"title":"xml"}}
		}}`, "application/xml", `{"title":"xml"}`},
		{"media without schema is skipped", `{"content":{
			"application/json":{"example":{"ok":true}},
			"application/problem+json":{"schema":{"title":"problem"}}
		}}`, "application/problem+json", `{"title":"problem"}`},
		{"legacy direct schema", `{"content":{"application/json":{"example":{}}},"schema":{"type":"array"}}`, "", `{"type":"array"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := (&API{Responses: rawJSON(`{"200":` + test.response + `}`)}).ResponseSchema()
			requireNoError(t, err)
			if !document.HasSchema() {
				t.Fatal("HasSchema() = false, want true")
			}
			if document.StatusCode != "200" || document.ContentType != test.wantContentType {
				t.Fatalf("selection = (%q, %q), want (%q, %q)", document.StatusCode, document.ContentType, "200", test.wantContentType)
			}
			requireJSONEqual(t, test.wantSchema, document.Schema)
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := (&API{Responses: test.responses}).ResponseSchema()
			requireNoError(t, err)
			if document.HasSchema() || document.StatusCode != test.wantStatus ||
				document.ContentType != "" || len(document.Schema) != 0 ||
				len(document.Components) != 0 {
				t.Fatalf("unexpected no-schema document: %#v", document)
			}
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

	document, err := api.ResponseSchema()
	requireNoError(t, err)
	if !document.HasSchema() || len(document.Components) != 2 {
		t.Fatalf("unexpected document: %#v", document)
	}
	requireMapContains(t, document.Components, "Result", "Item")
	if _, ok := document.Components["Unreachable"]; ok {
		t.Fatal("unreachable component was included")
	}
	requireJSONEqual(t, `{"type":"object","properties":{"result":{"$ref":"#/components/schemas/Result"}}}`, document.Schema)
	requireJSONEqual(t, `{"type":"object","properties":{"item":{"$ref":"#/components/schemas/Item"}}}`, document.Components["Result"])
	if len(document.Warnings) != 0 {
		t.Fatalf("Warnings = %v, want empty", document.Warnings)
	}
}

func TestResponseSchemaProtectsCyclicRefs(t *testing.T) {
	tests := []struct {
		name, root, components string
		wantNames              []string
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &API{
				Responses:  rawJSON(`{"200":{"schema":` + test.root + `}}`),
				Components: rawJSON(test.components),
			}
			document, err := api.ResponseSchema()
			requireNoError(t, err)
			if len(document.Components) != len(test.wantNames) {
				t.Fatalf("Components = %v, want %v", mapKeys(document.Components), test.wantNames)
			}
			requireMapContains(t, document.Components, test.wantNames...)
			if !strings.Contains(strings.Join(document.Warnings, "\n"), "cyclic") {
				t.Fatalf("Warnings = %v, want cyclic warning", document.Warnings)
			}
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

	document, err := api.ResponseSchema()
	requireNoError(t, err)
	if len(document.Components) != 1 {
		t.Fatalf("Components = %v, want Present only", mapKeys(document.Components))
	}
	requireMapContains(t, document.Components, "Present")
	if _, ok := document.Components["Missing"]; ok {
		t.Fatal("missing component was included")
	}
	warnings := strings.Join(document.Warnings, "\n")
	for _, expected := range []string{"#/components/schemas/Missing", "https://example.com/schema.json"} {
		if !strings.Contains(warnings, expected) {
			t.Fatalf("Warnings = %q, want %q", warnings, expected)
		}
	}
}

func TestResponseSchemaReturnsTypedErrorsForMalformedMetadata(t *testing.T) {
	tests := []struct {
		name, wantField string
		api             *API
	}{
		{"responses json", "responses", &API{Responses: rawJSON(`{"200":`)}},
		{"selected response shape", "responses", &API{Responses: rawJSON(`{"200":"not-an-object"}`)}},
		{
			name:      "components shape for referenced schema",
			api:       &API{Responses: rawJSON(`{"200":{"schema":{"$ref":"#/components/schemas/Result"}}}`), Components: rawJSON(`{"schemas":[]}`)},
			wantField: "components",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.api.ResponseSchema()
			if err == nil {
				t.Fatal("ResponseSchema() error = nil")
			}
			var metadataError *ResponseSchemaError
			if !errors.As(err, &metadataError) {
				t.Fatalf("error type = %T, want *ResponseSchemaError", err)
			}
			if metadataError.Field != test.wantField || metadataError.Unwrap() == nil {
				t.Fatalf("metadata error = %#v, want field %q and cause", metadataError, test.wantField)
			}
		})
	}
}

func TestResponseSchemaDoesNotDecodeUnneededComponents(t *testing.T) {
	api := &API{
		Responses:  rawJSON(`{"200":{"schema":{"type":"object","properties":{"ok":{"type":"boolean"}}}}}`),
		Components: rawJSON(`{"schemas":`),
	}

	document, err := api.ResponseSchema()
	requireNoError(t, err)
	if !document.HasSchema() || len(document.Components) != 0 || len(document.Warnings) != 0 {
		t.Fatalf("unexpected inline-schema document: %#v", document)
	}
}

func TestResponseSectionPreservesAllResponsesAndOnlyReachableComponents(t *testing.T) {
	api := &API{
		Responses: rawJSON(`{
			"200":{"schema":{"$ref":"#/components/schemas/Result"}},
			"400":{"schema":{"$ref":"#/components/schemas/Error"}}
		}`),
		Components: rawJSON(`{"schemas":{
			"Result":{"properties":{"item":{"$ref":"#/components/schemas/Item"}}},
			"Item":{"properties":{"parent":{"$ref":"#/components/schemas/Result"}}},
			"Error":{"properties":{"code":{"type":"string"}}},
			"Unused":{"type":"object"}
		}}`),
	}

	document, err := api.ResponseSection()
	requireNoError(t, err)
	requireJSONEqual(t, string(api.Responses), document.Responses)
	if got, want := mapKeys(document.Components), []string{"Error", "Item", "Result"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Components = %v, want %v", got, want)
	}
	if _, ok := document.Components["Unused"]; ok {
		t.Fatal("unreachable component was included")
	}
	const cyclicWarning = `cyclic schema reference "#/components/schemas/Result" was preserved`
	if !containsString(document.Warnings, cyclicWarning) {
		t.Fatalf("Warnings = %v, want %q", document.Warnings, cyclicWarning)
	}
}

func rawJSON(value string) json.RawMessage {
	return json.RawMessage(value)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireJSONEqual(t *testing.T, want string, got json.RawMessage) {
	t.Helper()
	var wantValue, gotValue any
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("invalid expected JSON: %v", err)
	}
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("invalid actual JSON: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("JSON = %s, want %s", got, want)
	}
}

func requireMapContains(t *testing.T, values map[string]json.RawMessage, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, ok := values[name]; !ok {
			t.Fatalf("Components = %v, missing %q", mapKeys(values), name)
		}
	}
}

func mapKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sortStrings(keys)
	return keys
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
