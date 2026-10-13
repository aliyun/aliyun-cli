package openapi

import (
	"testing"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func TestLegacyBackendTimeoutPrecedence(t *testing.T) {
	for _, tc := range []struct {
		product, action string
		backend         int64
		user            int
		sse             bool
		want            time.Duration
	}{
		{"rds", "CreateDBInstance", 55000, 0, false, 65 * time.Second},
		{"rds", "CreateDBInstance", 55000, 2, false, 0},
		{"ecs", "CreateInstance", 80000, 0, false, 0},
		{"ecs", "DescribeInstances", 80000, 0, false, 0},
		{"rds", "CreateDBInstance", 0, 0, false, 0},
		{"rds", "CreateDBInstance", -1, 0, false, 0},
		{"rds", "CreateDBInstance", 55000, 0, true, 0},
	} {
		request := requests.NewCommonRequest()
		request.Product = tc.product
		invoker := &BasicInvoker{request: request, profile: &config.Profile{ReadTimeout: tc.user}}
		invoker.applyBackendTimeout(&canonicalmeta.API{Name: tc.action, BackendTimeoutMS: tc.backend,
			Operation: &canonicalmeta.Operation{IsSSE: tc.sse}})
		if got := request.GetReadTimeout(); got != tc.want {
			t.Errorf("%s/%s backend=%d user=%d sse=%v: got %v, want %v", tc.product, tc.action, tc.backend, tc.user, tc.sse, got, tc.want)
		}
	}
}

func TestBackendTimeoutDoesNotUseAnotherAPIVersion(t *testing.T) {
	request := requests.NewCommonRequest()
	request.Product, request.Version = "rds", "2025-01-01"
	invoker := &BasicInvoker{request: request, profile: &config.Profile{}}
	invoker.applyBackendTimeout(&canonicalmeta.API{Name: "CreateDBInstance", BackendTimeoutMS: 55000,
		Operation: &canonicalmeta.Operation{APIVersion: "2014-08-15"}})
	if request.GetReadTimeout() != 0 {
		t.Fatal("used a budget from another API version")
	}
}
