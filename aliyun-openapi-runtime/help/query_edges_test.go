package help

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResponseQueriesSelectNestedResourceAndInheritedFields(t *testing.T) {
	context := ResponseQueryContext{Product: "demo", API: "ListPolicies", APIVersion: "v1", RequiredFlags: []string{"--region", "unsafe value"}, Document: ResponseDocumentation{
		Schema:     json.RawMessage(`{"type":"object","properties":{"Groups":{"type":"array","items":{"type":"object","properties":{"Policies":{"type":"array","items":{"allOf":[{"$ref":"#/components/schemas/Policy"}]}}}}}}}`),
		Components: map[string]json.RawMessage{"Policy": json.RawMessage(`{"properties":{"PolicyId":{"type":"string"},"PolicyName":{"type":"string"},"Status":{"type":"string"},"Name":{"type":"string"},"Weight":{"type":"number"}}}`)},
	}}
	example, err := BuildResponseQueryExample(context)
	if err != nil || example == nil {
		t.Fatalf("query=%#v,%v", example, err)
	}
	if want := "Groups[].Policies[*].{PolicyId:PolicyId,PolicyName:PolicyName,Status:Status}"; example.Path != want {
		t.Fatalf("query path=%q,want %q", example.Path, want)
	}
	if !strings.Contains(example.QueryCommand, "--region <value>") || strings.Contains(example.QueryCommand, "unsafe value") {
		t.Fatalf("required flags=%q", example.QueryCommand)
	}
	if !strings.Contains(example.SchemaCommand, "--api-version v1") {
		t.Fatalf("schema command=%q", example.SchemaCommand)
	}
}

func TestResponseQueryArraySelectionAndQuotedIdentifiers(t *testing.T) {
	for _, tc := range []struct{ name, schema, api, want string }{
		{"pagination collection", `{"properties":{"Other":{"type":"array"},"Page":{"properties":{"NextToken":{"type":"string"},"Records":{"type":"array"}}}}}`, "ListWidgets", "Page.Records"},
		{"resource collection", `{"properties":{"Other":{"type":"array"},"Policies":{"type":"array"}}}`, "ListPolicies", "Policies"},
		{"common collection", `{"properties":{"Other":{"type":"array"},"Records":{"type":"array"}}}`, "GetUnknown", "Records"},
		{"deterministic fallback", `{"properties":{"Alpha":{"type":"array"},"Beta":{"type":"array"}}}`, "GetUnknown", "Alpha"},
		{"quoted collection", `{"properties":{"odd collection":{"type":"array","items":{"properties":{"display name":{"type":"string"}}}}}}`, "GetUnknown", `"odd collection"[*].{"display name":"display name"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			example, err := BuildResponseQueryExample(ResponseQueryContext{Product: "demo", API: tc.api, Document: ResponseDocumentation{Schema: json.RawMessage(tc.schema)}})
			if err != nil || example == nil || example.Path != tc.want {
				t.Fatalf("query=%#v,%v,want %q", example, err, tc.want)
			}
		})
	}
}

func TestResponseQueriesRejectMalformedSchemasAndUnsafeCommandTokens(t *testing.T) {
	for _, raw := range []string{"{", "[]"} {
		_, err := BuildResponseQueryExample(ResponseQueryContext{Document: ResponseDocumentation{Schema: json.RawMessage(raw)}})
		if err == nil {
			t.Fatalf("invalid root %q accepted", raw)
		}
	}
	_, err := BuildResponseQueryExample(ResponseQueryContext{Document: ResponseDocumentation{Schema: json.RawMessage(`{}`), Components: map[string]json.RawMessage{"Broken": json.RawMessage("{")}}})
	if err == nil {
		t.Fatal("invalid component accepted")
	}
	for _, raw := range []string{`null`, `{"type":"array","items":{}}`} {
		example, err := BuildResponseQueryExample(ResponseQueryContext{Document: ResponseDocumentation{Schema: json.RawMessage(raw)}})
		if err != nil || example != nil {
			t.Fatalf("root without object array query=%#v,%v", example, err)
		}
	}
	for _, tc := range []struct{ product, api, version string }{{"", "Get", ""}, {"bad product", "Get", ""}, {"demo", "Get", "bad;version"}} {
		example, err := BuildResponseQueryExample(ResponseQueryContext{Product: tc.product, API: tc.api, APIVersion: tc.version, Document: ResponseDocumentation{Schema: json.RawMessage(`{"properties":{"Items":{"type":"array"}}}`)}})
		if err != nil || example != nil {
			t.Fatalf("unsafe command query=%#v,%v", example, err)
		}
	}
	if tokens := resourceTokens("List"); len(tokens) != 0 {
		t.Fatalf("verb-only resource=%v", tokens)
	}
}
