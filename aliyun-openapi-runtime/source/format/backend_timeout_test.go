package format

import (
	"encoding/json"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/schema"
)

func TestBackendTimeoutSurvivesJSONDecoding(t *testing.T) {
	var definition schema.CommandDefinition
	if err := json.Unmarshal([]byte(`{"name":"CreateDBInstance","timeout_ms":55000,"operation":{"action":"CreateDBInstance","api_version":"2014-08-15","method":"POST"}}`), &definition); err != nil {
		t.Fatal(err)
	}
	if got := schemaToAPI(&definition).TimeoutMS; got != 55000 {
		t.Fatalf("backend timeout = %d, want 55000", got)
	}
}
