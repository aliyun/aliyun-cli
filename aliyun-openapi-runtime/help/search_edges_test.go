package help

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestResponseSearchDescriptionsAndRequiredProjection(t *testing.T) {
	input := ResponseSchema{Schema: json.RawMessage(`{"type":"object","required":["Other",7],"properties":{"Other":{"type":"string"},"Selected":{"type":"string","description_zh":"区域标识"}}}`)}
	result, err := SearchResponseSchema(input, "区域", false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Paths, []string{"Selected"}) {
		t.Fatalf("paths = %v", result.Paths)
	}
	var schema map[string]any
	if err := json.Unmarshal(result.Schema, &schema); err != nil {
		t.Fatal(err)
	}
	if _, exists := schema["required"]; exists {
		t.Fatalf("unselected required properties retained: %s", result.Schema)
	}
	if strings.Contains(string(result.Schema), "Other") {
		t.Fatalf("unselected property retained: %s", result.Schema)
	}
	for _, keyword := range []string{"", "--"} {
		empty, err := SearchResponseSchema(ResponseSchema{Schema: json.RawMessage(`invalid`)}, keyword, false)
		if err != nil || len(empty.Paths) != 0 {
			t.Fatalf("empty query = %#v, %v", empty, err)
		}
	}
	if result, err := SearchResponseSchema(ResponseSchema{}, "anything", false); err != nil || len(result.Paths) != 0 {
		t.Fatalf("empty schema = %#v, %v", result, err)
	}
}

func TestSearchAcrossDescriptionsAndContainerElements(t *testing.T) {
	apiMatches := searchAPIs([]APISummary{{Name: "CreateThing", Title: LocalizedText{EN: "Provision a regional resource"}}}, "regional")
	if len(apiMatches) != 1 || apiMatches[0].Name != "CreateThing" {
		t.Fatalf("description API matches = %#v", apiMatches)
	}
	for _, parameter := range []Parameter{
		{Name: "items", Element: &Parameter{Name: "region"}},
		{Name: "labels", Value: &Parameter{Name: "region"}},
	} {
		matches := searchParameters([]Parameter{parameter}, "region")
		if len(matches) != 1 || matches[0].Name != parameter.Name {
			t.Fatalf("container match = %#v", matches)
		}
	}
	for _, ref := range []string{"#/components/schemas/", "#/components/schemas/A/B", "remote.json"} {
		if name, ok := componentName(ref); ok || name != "" {
			t.Fatalf("invalid reference accepted: %q -> %q", ref, name)
		}
	}
	if got := reachableComponentRaw(json.RawMessage(`invalid`), nil); got != nil {
		t.Fatalf("invalid schema components = %v", got)
	}
	components := map[string]json.RawMessage{"Region": json.RawMessage(`{"type":"string"}`)}
	got := reachableComponentRaw(json.RawMessage(`{"allOf":[{"$ref":"#/components/schemas/Region"}]}`), components)
	if !reflect.DeepEqual(got, components) {
		t.Fatalf("composed components = %v", got)
	}
}
