package format

import (
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
