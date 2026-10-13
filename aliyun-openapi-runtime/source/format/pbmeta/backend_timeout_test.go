package pbmeta

import (
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/source/format"
	"google.golang.org/protobuf/proto"
)

func TestBackendTimeoutSurvivesProtobufDecoding(t *testing.T) {
	original := &CommandDefinition{Name: "CreateDBInstance", ProductCode: "Rds", BackendTimeoutMs: 55000,
		Operation: &Operation{Action: "CreateDBInstance", ApiVersion: "2014-08-15", Method: "POST", Protocol: "HTTPS", ApiStyle: "RPC"}}
	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded CommandDefinition
	if err := proto.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	definition, err := toCanonical(&decoded)
	if err != nil {
		t.Fatal(err)
	}
	api, err := format.DecodeCommandDefinition(definition, "test")
	if err != nil {
		t.Fatal(err)
	}
	if api.BackendTimeoutMS != 55000 {
		t.Fatalf("backend timeout = %d, want 55000", api.BackendTimeoutMS)
	}
}
