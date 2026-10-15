package meta

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestResponseSectionHandlesMissingAndMalformedDocuments(t *testing.T) {
	for _, api := range []*API{nil, {}, {Responses: rawJSON("null")}} {
		doc, err := api.ResponseSection()
		if err != nil || len(doc.Responses) != 0 {
			t.Fatalf("empty section=%#v,%v", doc, err)
		}
	}
	for _, raw := range []string{"{", `[]`} {
		_, err := (&API{Responses: rawJSON(raw)}).ResponseSection()
		var schemaError *ResponseSchemaError
		if !errors.As(err, &schemaError) || schemaError.Field != "responses" || !strings.Contains(err.Error(), "parse runtime response responses failed") {
			t.Fatalf("section error=%v", err)
		}
		if errors.Unwrap(schemaError) == nil {
			t.Fatal("schema error lost cause")
		}
	}
	for _, raw := range []string{`{"200":[]}`, `{"200":{"content":{"application/json":[]}}}`} {
		_, err := (&API{Responses: rawJSON(raw)}).ResponseSchema()
		if err == nil || !strings.Contains(err.Error(), "responses") {
			t.Fatalf("malformed response error=%v", err)
		}
	}
}

func TestResponseSectionDeduplicatesReachableReferencesAndWarnings(t *testing.T) {
	api := &API{Responses: rawJSON(`{"200":{"schema":{"allOf":[{"$ref":"#/components/schemas/Item/properties/id"},{"$ref":"#/components/schemas/Item"},{"$ref":"remote.json"},{"$ref":"remote.json"},{"$ref":"#/components/schemas/"}]}}}`), Components: rawJSON(`{"schemas":{"Item":{"type":"object","properties":{"id":{"type":"string"}}},"Unused":{"type":"string"}}}`)}
	doc, err := api.ResponseSection()
	if err != nil || len(doc.Components) != 1 || doc.Components["Item"] == nil || len(doc.Warnings) != 2 {
		t.Fatalf("reachable section=%#v,%v", doc, err)
	}
	api.Responses = rawJSON(`{"200":{"schema":{"$ref":"#/components/schemas/Missing"}}}`)
	api.Components = nil
	doc, err = api.ResponseSection()
	if err != nil || doc.Components != nil || len(doc.Warnings) != 1 {
		t.Fatalf("missing component section=%#v,%v", doc, err)
	}
	api.Components = rawJSON(`[]`)
	_, err = api.ResponseSection()
	if err == nil || !strings.Contains(err.Error(), "components") {
		t.Fatalf("invalid components error=%v", err)
	}
}

func TestResponseSchemaIgnoresInvalidStatusCodesAndReportsReferenceDecodeErrors(t *testing.T) {
	doc, err := (&API{Responses: rawJSON(`{"2xx":{"schema":{"title":"invalid"}},"default":{"schema":{"title":"fallback"}}}`)}).ResponseSchema()
	if err != nil || doc.StatusCode != "default" {
		t.Fatalf("success selection=%#v,%v", doc, err)
	}
	if _, _, err := schemaReferences(json.RawMessage("{")); err == nil {
		t.Fatal("invalid reference JSON accepted")
	}
}
