package runtime

import (
	"testing"
	"time"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	credentialsv2 "github.com/aliyun/credentials-go/credentials"
)

func TestPrepareCallBackendTimeoutPrecedence(t *testing.T) {
	credential := pricingTestCredential{model: &credentialsv2.CredentialModel{
		AccessKeyId: stringPointer("ak"), AccessKeySecret: stringPointer("secret"),
	}}
	for _, tc := range []struct {
		metadata int64
		user     time.Duration
		sse      bool
		want     int
	}{
		{57000, 0, false, 57000},
		{62000, 0, false, 62000},
		{12001, 0, false, 12001},
		{57000, 2 * time.Second, false, 2000},
		{62000, 120 * time.Second, false, 120000},
		{0, 0, false, 0}, {-1, 0, false, 0},
		{57000, 0, true, 0},
	} {
		call, err := prepareCall(&ExecContext{Credential: credential, ReadTimeout: tc.user,
			API: &meta.API{ProductCode: "ecs", TimeoutMS: tc.metadata, IsSSE: tc.sse}},
			&AssembledRequest{Endpoint: "example.com", Protocol: "HTTPS", Action: "CreateInstance", Method: "POST"})
		if err != nil {
			t.Fatal(err)
		}
		got := 0
		if call.client.ReadTimeout != nil {
			got = *call.client.ReadTimeout
		}
		if got != tc.want {
			t.Errorf("metadata=%d user=%v sse=%v: got %d, want %d", tc.metadata, tc.user, tc.sse, got, tc.want)
		}
	}
}
