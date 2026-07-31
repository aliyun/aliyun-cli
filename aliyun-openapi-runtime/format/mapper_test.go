package format

import (
	"reflect"
	"testing"

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
