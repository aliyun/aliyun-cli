package pbmeta

import (
	"strconv"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/source/format"
	"google.golang.org/protobuf/proto"
)

func TestBackendTimeoutSurvivesProtobufDecoding(t *testing.T) {
	for _, timeout := range []int64{0, 12001, 57000, 62000} {
		t.Run(strconv.FormatInt(timeout, 10), func(t *testing.T) {
			original := &CommandDefinition{Name: "CreateDBInstance", ProductCode: "Rds", TimeoutMs: timeout,
				Operation: &Operation{Action: "CreateDBInstance", ApiVersion: "2014-08-15", Method: "POST", Protocol: "HTTPS", ApiStyle: "RPC"}}
			data, err := proto.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			var decoded CommandDefinition
			if err := proto.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.GetTimeoutMs() != timeout {
				t.Fatalf("protobuf timeout = %d, want %d", decoded.GetTimeoutMs(), timeout)
			}
			definition, err := toCanonical(&decoded)
			if err != nil {
				t.Fatal(err)
			}
			api, err := format.DecodeCommandDefinition(definition, "test")
			if err != nil {
				t.Fatal(err)
			}
			if api.TimeoutMS != timeout {
				t.Fatalf("metadata timeout = %d, want %d", api.TimeoutMS, timeout)
			}
		})
	}
}
