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
		backend int64
		user    time.Duration
		sse     bool
		want    int
	}{
		{55000, 0, false, 65000},
		{80000, 0, false, 90000},
		{55000, 2 * time.Second, false, 2000},
		{0, 0, false, 0}, {-1, 0, false, 0},
		{55000, 0, true, 0},
	} {
		call, err := prepareCall(&ExecContext{Credential: credential, ReadTimeout: tc.user,
			API: &meta.API{ProductCode: "ecs", BackendTimeoutMS: tc.backend, IsSSE: tc.sse}},
			&AssembledRequest{Endpoint: "example.com", Protocol: "HTTPS", Action: "CreateInstance", Method: "POST"})
		if err != nil {
			t.Fatal(err)
		}
		got := 0
		if call.client.ReadTimeout != nil {
			got = *call.client.ReadTimeout
		}
		if got != tc.want {
			t.Errorf("backend=%d user=%v sse=%v: got %d, want %d", tc.backend, tc.user, tc.sse, got, tc.want)
		}
	}
}
