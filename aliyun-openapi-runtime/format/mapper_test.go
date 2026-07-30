package format

import (
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
