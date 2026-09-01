package help

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestRuntimeHelpJSONUsesPublicHelpLevelEnvelope(t *testing.T) {
	product := &meta.Product{Code: "demo", Versions: []string{"v1"}}
	index := &meta.APIIndex{ProductCode: "demo", Version: "v1", Entries: map[string]meta.APIIndexEntry{}}
	api := &meta.API{
		Name: "CreateThing", CmdName: "create-thing", ProductCode: "demo", Version: "v1",
		Parameters: []meta.Parameter{{Name: "name", RawName: "Name", Options: []string{"--name"}, Type: meta.TypeString}},
	}
	responseDocumentation := &ResponseDocumentation{
		Responses: json.RawMessage(`{"200":{"description":"OK"}}`),
		Schema:    json.RawMessage(`{"type":"object"}`),
	}
	parameter, err := BuildAPIParameterDocument(product, api, "name", HelpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	response, err := BuildAPIResponseDocument(api, responseDocumentation, HelpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	documents := []struct {
		name      string
		helpLevel string
		document  any
	}{
		{name: "product", helpLevel: "product", document: BuildProductDocument(product, index, HelpOptions{})},
		{name: "action", helpLevel: "api", document: BuildActionDocument(product, api, responseDocumentation, HelpOptions{})},
		{name: "request", helpLevel: "api", document: BuildRequestDocument(product, api, responseDocumentation, HelpOptions{})},
		{name: "response", helpLevel: "api", document: response},
		{name: "parameter", helpLevel: "parameter", document: parameter},
	}

	provenance := &MetadataProvenance{Kind: "user", PluginName: "aliyun-cli-demo", PluginVersion: "1.2.3"}
	for _, test := range documents {
		t.Run(test.name, func(t *testing.T) {
			switch document := test.document.(type) {
			case *ProductDocument:
				document.Provenance = provenance
			case *ActionDocument:
				document.Provenance = provenance
			case *RequestDocument:
				document.Provenance = provenance
			case *APIResponseDocument:
				document.Provenance = provenance
			case *APIParameterDocument:
				document.Provenance = provenance
			}
			var output bytes.Buffer
			if err := Render(&output, test.document, HelpOptions{Format: FormatJSON, AIMode: true}); err != nil {
				t.Fatal(err)
			}
			var envelope map[string]any
			if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if got := envelope["helpLevel"]; got != test.helpLevel {
				t.Fatalf("helpLevel = %#v, want %q; JSON: %s", got, test.helpLevel, output.String())
			}
			if _, exists := envelope["kind"]; exists {
				t.Fatalf("kind leaked into public JSON: %s", output.String())
			}
			if _, exists := envelope["target"]; exists {
				t.Fatalf("target leaked into public JSON: %s", output.String())
			}
			if _, exists := envelope["provenance"]; exists {
				t.Fatalf("provenance leaked into public JSON: %s", output.String())
			}
			if test.name == "request" && envelope["section"] != "request" {
				t.Fatalf("request section = %#v", envelope["section"])
			}
			if test.name == "response" && envelope["section"] != "response" {
				t.Fatalf("response section = %#v", envelope["section"])
			}
		})
	}
}

func TestRuntimeHelpJSONKeepsNextCommandsWhileHidingRoutingTargets(t *testing.T) {
	product := &meta.Product{Code: "demo", Versions: []string{"v1"}}
	entries := make(map[string]meta.APIIndexEntry, 25)
	parameters := make([]meta.Parameter, 0, 25)
	fields := make([]meta.Parameter, 0, 25)
	properties := make(map[string]any, 25)
	for i := 0; i < 25; i++ {
		apiName := fmt.Sprintf("PluginField%02d", i)
		command := fmt.Sprintf("plugin-field-%02d", i)
		entries[apiName] = meta.APIIndexEntry{APIName: apiName, CmdName: command}
		parameter := meta.Parameter{
			Name: command, RawName: apiName, Options: []string{"--" + command}, Type: meta.TypeString,
		}
		parameters = append(parameters, parameter)
		fields = append(fields, parameter)
		properties[apiName] = map[string]any{"type": "string"}
	}
	index := &meta.APIIndex{ProductCode: "demo", Version: "v1", Entries: entries}
	api := &meta.API{
		Name: "ListPlugins", CmdName: "list-plugins", ProductCode: "demo", Version: "v1",
		Parameters: parameters,
	}
	responseSchema, err := json.Marshal(map[string]any{"type": "object", "properties": properties})
	if err != nil {
		t.Fatal(err)
	}
	responseDocumentation := &ResponseDocumentation{Schema: responseSchema, StatusCode: "200"}
	parameterAPI := &meta.API{
		Name: "ListPlugins", CmdName: "list-plugins", ProductCode: "demo", Version: "v1",
		Parameters: []meta.Parameter{{
			Name: "filters", RawName: "Filters", Options: []string{"--filters"}, Type: meta.TypeObject,
			Fields: fields,
		}},
	}
	parameter, err := BuildAPIParameterDocument(product, parameterAPI, "filters", HelpOptions{
		Search: "plugin", RequestedVersion: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := BuildAPIResponseDocument(api, responseDocumentation, HelpOptions{
		Search: "plugin", RequestedVersion: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	documents := []struct {
		name      string
		document  any
		searchAll string
	}{
		{
			name: "product",
			document: BuildProductDocument(product, index, HelpOptions{
				Search: "plugin", RequestedVersion: "v1",
			}),
			searchAll: "aliyun demo --api-version v1 --help-search plugin --help-all",
		},
		{
			name: "action",
			document: BuildActionDocument(product, api, responseDocumentation, HelpOptions{
				Search: "plugin", RequestedVersion: "v1",
			}),
			searchAll: "aliyun demo list-plugins --api-version v1 --help-search plugin --help-all",
		},
		{
			name:      "parameter",
			document:  parameter,
			searchAll: "aliyun demo list-plugins --api-version v1 --filters --help-search plugin --help-all",
		},
		{
			name:      "response",
			document:  response,
			searchAll: "aliyun help demo list-plugins --api-version v1 --cli-section response --help-search plugin --help-all",
		},
	}

	for _, test := range documents {
		t.Run(test.name, func(t *testing.T) {
			var target Target
			var next *Next
			switch document := test.document.(type) {
			case *ProductDocument:
				target, next = document.Target, document.Next
			case *ActionDocument:
				target, next = document.Target, document.Next
			case *APIParameterDocument:
				target, next = document.Target, document.Next
			case *APIResponseDocument:
				target, next = document.Target, document.Next
			}
			if target.Product != "demo" || next == nil || next.SearchAll != test.searchAll {
				t.Fatalf("internal route/next = target:%#v next:%#v", target, next)
			}

			var output bytes.Buffer
			if err := Render(&output, test.document, HelpOptions{Format: FormatJSON, AIMode: true}); err != nil {
				t.Fatal(err)
			}
			var envelope map[string]any
			if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if _, exists := envelope["target"]; exists {
				t.Fatalf("target leaked into public JSON: %s", output.String())
			}
			nextEnvelope, ok := envelope["next"].(map[string]any)
			if !ok || nextEnvelope["searchAll"] != test.searchAll {
				t.Fatalf("next command was not preserved: %s", output.String())
			}
		})
	}
}

func TestRuntimeHelpJSONPreservesBusinessPluginAndProviderSchemaFields(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"plugin":{"type":"string"},"provider":{"type":"string"}}}`)
	documents := []*APIResponseDocument{
		{
			SchemaVersion: SchemaVersion, Kind: "api", Section: SectionResponse,
			Responses:  json.RawMessage(`{"200":{"schema":{"type":"object","properties":{"plugin":{"type":"string"},"provider":{"type":"string"}}}}}`),
			Components: map[string]json.RawMessage{"Business": schema},
		},
		{
			SchemaVersion: SchemaVersion, Kind: "api", Section: SectionResponse,
			OutputSchema: &OutputSchema{Schema: schema, Components: map[string]json.RawMessage{"Business": schema}},
		},
	}
	for _, document := range documents {
		var output bytes.Buffer
		if err := Render(&output, document, HelpOptions{Format: FormatJSON, AIMode: true}); err != nil {
			t.Fatal(err)
		}
		if bytes.Count(output.Bytes(), []byte(`"plugin"`)) != 2 || bytes.Count(output.Bytes(), []byte(`"provider"`)) != 2 {
			t.Fatalf("business schema fields were removed: %s", output.String())
		}
	}
}

func TestRuntimeHelpJSONPreservesBusinessFieldsThroughResponseSearchProjection(t *testing.T) {
	schema := json.RawMessage(`{
		"type":"object",
		"required":["Records","Choice"],
		"properties":{
			"Records":{"type":"array","items":{"$ref":"#/components/schemas/Record"}},
			"Choice":{"allOf":[{"$ref":"#/components/schemas/Record"}],"oneOf":[{"$ref":"#/components/schemas/Record"}],"anyOf":[{"$ref":"#/components/schemas/Record"}]}
		}
	}`)
	components := map[string]json.RawMessage{
		"Record": json.RawMessage(`{
			"type":"object",
			"required":["plugin","provider"],
			"properties":{
				"plugin":{"type":"string"},
				"provider":{"type":"string"},
				"nested":{"$ref":"#/components/schemas/Nested"}
			}
		}`),
		"Nested": json.RawMessage(`{"type":"object","properties":{"plugin":{"type":"string"},"provider":{"type":"string"}}}`),
	}

	full, err := SearchResponseSchema(ResponseSchema{Schema: schema, Components: components}, "records", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := full.Components["Record"]; !ok {
		t.Fatalf("Record component missing: %#v", full.Components)
	}
	if _, ok := full.Components["Nested"]; !ok {
		t.Fatalf("Nested component missing: %#v", full.Components)
	}
	var record map[string]any
	if err := json.Unmarshal(full.Components["Record"], &record); err != nil {
		t.Fatal(err)
	}
	properties, ok := record["properties"].(map[string]any)
	if !ok || properties["plugin"] == nil || properties["provider"] == nil {
		t.Fatalf("business fields changed in projected component: %s", full.Components["Record"])
	}

	api := &meta.API{Name: "ListPlugins", CmdName: "list-plugins", ProductCode: "demo", Version: "v1"}
	documentation := &ResponseDocumentation{Schema: schema, Components: components, StatusCode: "200"}
	document, err := BuildAPIResponseDocument(api, documentation, HelpOptions{Search: "provider", All: true})
	if err != nil {
		t.Fatal(err)
	}
	if document.Kind != "api" || document.Section != SectionResponse || len(document.Matches) == 0 {
		t.Fatalf("response projection = %#v", document)
	}
	var output bytes.Buffer
	if err := Render(&output, document, HelpOptions{Format: FormatJSON, AIMode: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"helpLevel":"api"`) ||
		!strings.Contains(output.String(), `"section":"response"`) ||
		bytes.Count(output.Bytes(), []byte(`"provider"`)) == 0 {
		t.Fatalf("projected response contract lost business fields: %s", output.String())
	}
}

func TestRuntimeHelpJSONReportsInvalidRawSchemaInEveryResponseContainer(t *testing.T) {
	tests := []struct {
		name     string
		document *APIResponseDocument
	}{
		{name: "responses", document: &APIResponseDocument{Responses: json.RawMessage(`{`)}},
		{name: "components", document: &APIResponseDocument{Components: map[string]json.RawMessage{"Broken": json.RawMessage(`{`)}}},
		{name: "output schema", document: &APIResponseDocument{OutputSchema: &OutputSchema{Schema: json.RawMessage(`{`)}}},
		{name: "output components", document: &APIResponseDocument{OutputSchema: &OutputSchema{
			Schema:     json.RawMessage(`{"type":"object"}`),
			Components: map[string]json.RawMessage{"Broken": json.RawMessage(`{`)},
		}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := Render(&output, test.document, HelpOptions{Format: FormatJSON}); err == nil {
				t.Fatalf("invalid %s JSON was accepted", test.name)
			}
		})
	}

	api := &meta.API{Name: "ListPlugins", CmdName: "list-plugins", ProductCode: "demo", Version: "v1"}
	for _, documentation := range []*ResponseDocumentation{
		{Schema: json.RawMessage(`{`)},
		{Schema: json.RawMessage(`{"type":"object"}`), Components: map[string]json.RawMessage{"Broken": json.RawMessage(`{`)}},
	} {
		if _, err := BuildAPIResponseDocument(api, documentation, HelpOptions{Search: "plugin"}); err == nil {
			t.Fatal("response search accepted invalid raw schema")
		}
	}
}
