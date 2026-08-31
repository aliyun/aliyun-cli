package format

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
)

func TestSchemaToAPIDetectsSSE(t *testing.T) {
	tests := []struct {
		name string
		def  *schema.CommandDefinition
		want bool
	}{
		{
			name: "operation flag",
			def: &schema.CommandDefinition{Operation: &schema.OperationConfig{
				Protocol: "HTTPS",
				IsSSE:    true,
			}},
			want: true,
		},
		{
			name: "ordinary HTTPS",
			def: &schema.CommandDefinition{Operation: &schema.OperationConfig{
				Protocol: "HTTPS",
			}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := schemaToAPI(tt.def).IsSSE; got != tt.want {
				t.Fatalf("IsSSE = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSchemaToAPIMapsRequestBodyWireMetadata(t *testing.T) {
	definition := &schema.CommandDefinition{Operation: &schema.OperationConfig{
		ReqBodyType: "formData",
		ContentType: "application/x-www-form-urlencoded",
	}}
	api := schemaToAPI(definition)
	if got, want := []string{api.ReqBodyType, api.ContentType}, []string{"formData", "application/x-www-form-urlencoded"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("request body metadata = %v, want %v", got, want)
	}
}

func TestSchemaToAPIMapsResponseMetadata(t *testing.T) {
	definition := &schema.CommandDefinition{
		Responses:  json.RawMessage(`{"200":{"description":"OK"}}`),
		Components: json.RawMessage(`{"schemas":{"Thing":{"type":"object"}}}`),
	}
	api := schemaToAPI(definition)
	if !reflect.DeepEqual(api.Responses, definition.Responses) ||
		!reflect.DeepEqual(api.Components, definition.Components) {
		t.Fatalf("response metadata = %s, %s", api.Responses, api.Components)
	}

	definition.Responses[0] = '['
	definition.Components[0] = '['
	if api.Responses[0] != '{' || api.Components[0] != '{' {
		t.Fatal("response metadata must be copied")
	}
}

func TestSchemaToAPIMapsWildcardPathMetadata(t *testing.T) {
	definition := &schema.CommandDefinition{
		Operation: &schema.OperationConfig{HasWildcardPath: true},
		Parameters: []schema.ArgumentDefinition{{
			Name: "request_path", RawName: "requestPath", Location: "path", IsWildcard: true,
		}},
	}
	api := schemaToAPI(definition)
	if !api.HasWildcardPath || len(api.Parameters) != 1 || !api.Parameters[0].IsWildcard {
		t.Fatalf("wildcard metadata was not mapped: %#v", api)
	}
}

func TestMapArgumentMapsRecursiveCompositeMetadata(t *testing.T) {
	arg := schema.ArgumentDefinition{
		Type: "map",
		Value: &schema.TypeShape{
			Type: "array",
			Element: &schema.TypeShape{
				Type: "object",
				Fields: []schema.ArgumentDefinition{{
					Name: "enabled", RawName: "Enabled", Type: "boolean",
				}},
			},
		},
	}

	got := mapArgument(&arg)
	if got.ValueType == nil || got.ValueType.Type != meta.TypeArray ||
		got.ValueType.ItemType == nil || got.ValueType.ItemType.Type != meta.TypeObject ||
		len(got.ValueType.ItemType.Fields) != 1 ||
		got.ValueType.ItemType.Fields[0].Type != meta.TypeBoolean {
		t.Fatalf("recursive map shape was not mapped: %#v", got)
	}
}

func TestMapArgumentSupportsArbitraryRecursiveContainerDepth(t *testing.T) {
	arg := schema.ArgumentDefinition{
		Type: "array",
		Element: &schema.TypeShape{Type: "map", Value: &schema.TypeShape{
			Type: "array", Element: &schema.TypeShape{Type: "string"},
		}},
	}

	got := mapArgument(&arg)
	if got.ItemType == nil || got.ItemType.ValueType == nil ||
		got.ItemType.ValueType.ItemType == nil ||
		got.ItemType.ValueType.ItemType.Type != meta.TypeString {
		t.Fatalf("deep recursive shape was not mapped: %#v", got)
	}
}

func TestMapArgumentMapsRecursiveConstraints(t *testing.T) {
	arg := schema.ArgumentDefinition{
		Name: "items", Type: "array", Format: "ignored-top",
		Enum: []string{"ignored-for-container"},
		Element: &schema.TypeShape{
			Type: "object", Format: "ignored-element",
			Fields: []schema.ArgumentDefinition{{
				Name: "score", RawName: "Score", Type: "float",
				Enum: []string{"1.5", "2.5"}, Minimum: "1.5", Maximum: "2.5",
			}, {
				Name: "label", RawName: "Label", Type: "string",
				Pattern: "^[a-z]+$",
			}},
		},
	}

	got := mapArgument(&arg)
	if !reflect.DeepEqual(got.Enum, []string{"ignored-for-container"}) ||
		got.ItemType == nil || len(got.ItemType.Fields) != 2 {
		t.Fatalf("recursive constraints were not mapped: %#v", got)
	}
	score := got.ItemType.Fields[0]
	if !reflect.DeepEqual(score.Enum, []string{"1.5", "2.5"}) ||
		score.Minimum != "1.5" || score.Maximum != "2.5" {
		t.Fatalf("numeric constraints = %#v", score)
	}
	if got.ItemType.Fields[1].Pattern != "^[a-z]+$" {
		t.Fatalf("pattern constraint = %#v", got.ItemType.Fields[1])
	}
}

func TestMapArgumentMapsJSONDocRequired(t *testing.T) {
	var arg schema.ArgumentDefinition
	if err := json.Unmarshal([]byte(`{
		"name":"config","raw_name":"Config","type":"object",
		"doc_required":true,"min_length":"1","max_length":"8",
		"fields":[{
			"name":"token","raw_name":"Token","type":"string",
			"doc_required":true,"min_length":"2","max_length":"4"
		}]
	}`), &arg); err != nil {
		t.Fatal(err)
	}

	got := mapArgument(&arg)
	if !got.DocRequired || got.MinLength != "1" || got.MaxLength != "8" ||
		len(got.Fields) != 1 || !got.Fields[0].DocRequired ||
		got.Fields[0].MinLength != "2" || got.Fields[0].MaxLength != "4" {
		t.Fatalf("JSON docRequired metadata was not mapped recursively: %#v", got)
	}
}

func TestProductEntryToProductDefaultsAndCopies(t *testing.T) {
	entry := &schema.ProductEntry{
		PluginDefaultVersion: "v2", Version: "v1", Versions: []string{"v1", "v2"},
		Name: map[string]string{"zh": "演示", "en": "Demo"}, GlobalEndpoint: "global.example",
		RegionalEndpoints: map[string]string{"cn-a": "public.example"}, RegionalVPCEndpoints: map[string]string{"cn-a": "vpc.example"},
	}
	product := ProductEntryToProduct(entry, "demo")
	if product.Code != "demo" || product.DefaultVersion != "v2" || product.Description.ZH != "演示" || product.Description.EN != "Demo" ||
		product.Endpoints.Global != "global.example" || product.Endpoints.Public["cn-a"] != "public.example" || product.Endpoints.VPC["cn-a"] != "vpc.example" {
		t.Fatalf("product = %#v", product)
	}
	product.Versions[0] = "changed"
	if entry.Versions[0] != "v1" {
		t.Fatal("Versions was not copied")
	}

	if got := ProductEntryToProduct(&schema.ProductEntry{Version: "legacy"}, "x").DefaultVersion; got != "legacy" {
		t.Fatalf("legacy default = %q", got)
	}
	if got := ProductEntryToProduct(&schema.ProductEntry{Versions: []string{"2020-01-01", "2022-01-01"}}, "x").DefaultVersion; got != "2022-01-01" {
		t.Fatalf("chosen default = %q", got)
	}
	if mapLookup(nil, "en") != "" || mapLookup(map[string]string{"en": "name"}, "en") != "name" {
		t.Fatal("mapLookup returned an unexpected value")
	}
}

func TestMapperScalarTablesAndExamples(t *testing.T) {
	typeTests := map[string]meta.DataType{
		"": meta.TypeString, " string ": meta.TypeString, "int": meta.TypeInteger, "int32": meta.TypeInteger, "integer": meta.TypeInteger,
		"long": meta.TypeLong, "int64": meta.TypeLong, "float": meta.TypeFloat, "double": meta.TypeFloat, "number": meta.TypeFloat,
		"bool": meta.TypeBoolean, "boolean": meta.TypeBoolean, "object": meta.TypeObject, "struct": meta.TypeObject,
		"array": meta.TypeArray, "list": meta.TypeArray, "repeatlist": meta.TypeArray, "map": meta.TypeMap, "any": meta.TypeAny, "unknown": meta.TypeAny,
	}
	for input, want := range typeTests {
		if got := mapType(input); got != want {
			t.Errorf("mapType(%q) = %v, want %v", input, got, want)
		}
	}
	positionTests := map[string]meta.Position{
		"": meta.PosQuery, "query": meta.PosQuery, "body": meta.PosBody, "header": meta.PosHeader, "path": meta.PosPath,
		"formdata": meta.PosFormData, "host": meta.PosHost, "unknown": meta.PosQuery,
	}
	for input, want := range positionTests {
		if got := mapPosition(input); got != want {
			t.Errorf("mapPosition(%q) = %v, want %v", input, got, want)
		}
	}
	for input, want := range map[string]meta.APIStyle{"": meta.StyleRPC, " RPC ": meta.StyleRPC, "roa": meta.StyleROA, "RESTFUL": meta.StyleROA, "unknown": meta.StyleRPC} {
		if got := mapStyle(input); got != want {
			t.Errorf("mapStyle(%q) = %v, want %v", input, got, want)
		}
	}
	if got := exampleList(&schema.CommandDefinition{}); got != nil {
		t.Fatalf("empty examples = %#v", got)
	}
	if got := exampleList(&schema.CommandDefinition{CamelExample: "camel"}); !reflect.DeepEqual(got, []string{"camel"}) {
		t.Fatalf("camel examples = %#v", got)
	}
	if got := exampleList(&schema.CommandDefinition{KebabExample: "kebab", CamelExample: "camel"}); !reflect.DeepEqual(got, []string{"kebab"}) {
		t.Fatalf("preferred examples = %#v", got)
	}
	if mapTypeShape(nil) != nil || mapArguments(nil) != nil {
		t.Fatal("nil mapping should remain nil")
	}
}

func TestSchemaToAPIMapsOperationAndArgumentMetadata(t *testing.T) {
	def := &schema.CommandDefinition{
		Name: "Action", CmdName: "action", CmdFullName: "action-full",
		TitleZH: "标题", TitleEN: "title", DescriptionZH: "中", DescriptionEN: "en",
		Deprecated: true, MultiVersion: true,
		Operation: &schema.OperationConfig{
			APIVersion: "v1", Method: "get", URL: "/items/*", APIStyle: "ROA",
			Protocol: "HTTPS", HasWildcardPath: true,
		},
		Parameters: []schema.ArgumentDefinition{{
			Name: "name", RawName: "Name", Type: "string", Location: "header", Required: true, DirectBody: true,
			Options: []string{"--name"}, HelpZH: "名称", HelpEN: "name", Example: "demo", ParamStyle: "json",
		}},
	}
	api := schemaToAPI(def)
	if api.Name != "Action" || api.CmdFullName != "action-full" || api.Version != "v1" ||
		api.Method != "GET" || api.Style != meta.StyleROA || api.URL != "/items/*" ||
		api.Protocol != "HTTPS" || !api.Deprecated || !api.MultiVersion || !api.HasWildcardPath {
		t.Fatalf("API = %#v", api)
	}
	if api.Title.ZH != "标题" || api.Title.EN != "title" {
		t.Fatalf("API title = %#v", api.Title)
	}
	p := api.Parameters[0]
	if p.Position != meta.PosHeader || !p.Required || !p.DirectBody || p.Description.EN != "name" || p.Example != "demo" || p.ParamStyle != "json" {
		t.Fatalf("parameter = %#v", p)
	}
	withoutOperation := schemaToAPI(&schema.CommandDefinition{Name: "NoOp"})
	if withoutOperation.Name != "NoOp" || withoutOperation.Version != "" {
		t.Fatalf("API without operation = %#v", withoutOperation)
	}
}
