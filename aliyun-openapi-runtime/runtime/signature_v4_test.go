// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"testing"

	slsgateway "github.com/alibabacloud-go/alibabacloud-gateway-sls/client"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestPrepareCallEnablesSignatureV4(t *testing.T) {
	ec := &ExecContext{
		API: &meta.API{
			ProductCode: "Ecs",
			Name:        "DescribeInstances",
			Version:     "2014-05-26",
		},
		Credential:       staticAKCredential(t),
		SignatureVersion: SignatureVersionV4,
		Region:           "cn-hangzhou",
	}
	req := &AssembledRequest{
		Action:   "DescribeInstances",
		Version:  "2014-05-26",
		Method:   "POST",
		Protocol: "HTTPS",
		Style:    "RPC",
		Endpoint: "ecs.cn-hangzhou.aliyuncs.com",
	}
	call, err := prepareCall(ec, req)
	if err != nil {
		t.Fatalf("prepareCall: %v", err)
	}
	if got := tea.StringValue(call.client.SignatureVersion); got != SignatureVersionV4 {
		t.Fatalf("SignatureVersion = %q, want %q", got, SignatureVersionV4)
	}
	if got := tea.StringValue(call.client.ProductId); got != "Ecs" {
		t.Fatalf("ProductId = %q, want Ecs", got)
	}
	if call.client.Spi == nil {
		t.Fatal("expected POP gateway SPI to be installed")
	}
}

func TestPrepareCallSignatureV3IsDefault(t *testing.T) {
	ec := &ExecContext{
		API: &meta.API{
			ProductCode: "Ecs",
			Name:        "DescribeInstances",
			Version:     "2014-05-26",
		},
		Credential:       staticAKCredential(t),
		SignatureVersion: SignatureVersionV3,
		Region:           "cn-hangzhou",
	}
	req := &AssembledRequest{
		Action:   "DescribeInstances",
		Version:  "2014-05-26",
		Method:   "POST",
		Protocol: "HTTPS",
		Style:    "RPC",
		Endpoint: "ecs.cn-hangzhou.aliyuncs.com",
	}
	call, err := prepareCall(ec, req)
	if err != nil {
		t.Fatalf("prepareCall: %v", err)
	}
	if call.client.SignatureVersion != nil && tea.StringValue(call.client.SignatureVersion) != "" {
		t.Fatalf("unexpected SignatureVersion %q", tea.StringValue(call.client.SignatureVersion))
	}
	if call.client.SignatureAlgorithm != nil && tea.StringValue(call.client.SignatureAlgorithm) != "" {
		t.Fatalf("unexpected SignatureAlgorithm %q", tea.StringValue(call.client.SignatureAlgorithm))
	}
	if call.client.Spi != nil {
		t.Fatalf("v3 must not install gateway, got %T", call.client.Spi)
	}
}

func TestPrepareCallSignatureV2SetsAlgorithm(t *testing.T) {
	ec := &ExecContext{
		API: &meta.API{
			ProductCode: "Ecs",
			Name:        "DescribeInstances",
			Version:     "2014-05-26",
		},
		Credential:       staticAKCredential(t),
		SignatureVersion: SignatureVersionV2,
		Region:           "cn-hangzhou",
	}
	req := &AssembledRequest{
		Action:   "DescribeInstances",
		Version:  "2014-05-26",
		Method:   "POST",
		Protocol: "HTTPS",
		Style:    "RPC",
		Endpoint: "ecs.cn-hangzhou.aliyuncs.com",
	}
	call, err := prepareCall(ec, req)
	if err != nil {
		t.Fatalf("prepareCall: %v", err)
	}
	if tea.StringValue(call.client.SignatureAlgorithm) != "v2" {
		t.Fatalf("SignatureAlgorithm = %q, want v2", tea.StringValue(call.client.SignatureAlgorithm))
	}
	if call.client.SignatureVersion != nil && tea.StringValue(call.client.SignatureVersion) != "" {
		t.Fatalf("v2 must not set SignatureVersion, got %q", tea.StringValue(call.client.SignatureVersion))
	}
	if call.client.Spi != nil {
		t.Fatalf("v2 must not install gateway, got %T", call.client.Spi)
	}
}

func TestPrepareCallRejectsUnsupportedSignatureVersion(t *testing.T) {
	ec := &ExecContext{
		API: &meta.API{
			ProductCode: "Ecs",
			Name:        "DescribeInstances",
			Version:     "2014-05-26",
		},
		Credential:       staticAKCredential(t),
		SignatureVersion: "v1",
		Region:           "cn-hangzhou",
	}
	req := &AssembledRequest{
		Action:   "DescribeInstances",
		Version:  "2014-05-26",
		Method:   "POST",
		Protocol: "HTTPS",
		Style:    "RPC",
		Endpoint: "ecs.cn-hangzhou.aliyuncs.com",
	}
	_, err := prepareCall(ec, req)
	if err == nil {
		t.Fatal("expected unsupported signature_version error")
	}
}

func TestBuildExecContextCopiesSignatureVersion(t *testing.T) {
	// Imported via engine package would create a cycle; verify Settings field
	// round-trips through StaticHost which buildExecContext reads.
	host := StaticHost{SettingsVal: Settings{SignatureVersion: SignatureVersionV4}}
	if got := host.Settings().SignatureVersion; got != SignatureVersionV4 {
		t.Fatalf("Settings.SignatureVersion = %q, want %q", got, SignatureVersionV4)
	}
}

func TestPrepareCallSignatureV4RequiresProductId(t *testing.T) {
	ec := &ExecContext{
		API: &meta.API{
			ProductCode: "",
			Name:        "DescribeInstances",
			Version:     "2014-05-26",
		},
		Credential:       staticAKCredential(t),
		SignatureVersion: SignatureVersionV4,
		Region:           "cn-hangzhou",
	}
	req := &AssembledRequest{
		Action:   "DescribeInstances",
		Version:  "2014-05-26",
		Method:   "POST",
		Protocol: "HTTPS",
		Style:    "RPC",
		Endpoint: "ecs.cn-hangzhou.aliyuncs.com",
	}
	_, err := prepareCall(ec, req)
	if err == nil {
		t.Fatal("expected product id error")
	}
}

func TestPrepareCallSlsSignatureV4KeepsSlsGateway(t *testing.T) {
	ec := &ExecContext{
		API: &meta.API{
			ProductCode: "Sls",
			Name:        "GetProject",
			Version:     "2020-12-30",
		},
		Credential:       staticAKCredential(t),
		SignatureVersion: SignatureVersionV4,
		Region:           "cn-hangzhou",
	}
	req := &AssembledRequest{
		Action:   "GetProject",
		Version:  "2020-12-30",
		Method:   "GET",
		Protocol: "HTTPS",
		Style:    "ROA",
		Endpoint: "cn-hangzhou.log.aliyuncs.com",
	}
	call, err := prepareCall(ec, req)
	if err != nil {
		t.Fatalf("prepareCall: %v", err)
	}
	if got := tea.StringValue(call.client.SignatureVersion); got != SignatureVersionV4 {
		t.Fatalf("SignatureVersion = %q, want %q", got, SignatureVersionV4)
	}
	if tea.StringValue(call.client.ProductId) != "" {
		t.Fatalf("SLS V4 must not set POP ProductId, got %q", tea.StringValue(call.client.ProductId))
	}
	if _, ok := call.client.Spi.(*slsgateway.Client); !ok {
		t.Fatalf("SLS V4 must keep slsgateway.Client, got %T", call.client.Spi)
	}
}

func TestPrepareCallSignatureV4ProductFallback(t *testing.T) {
	ec := &ExecContext{
		API: &meta.API{
			ProductCode: "",
			Name:        "DescribeInstances",
			Version:     "2014-05-26",
		},
		Product:          &meta.Product{Code: "Ecs"},
		Credential:       staticAKCredential(t),
		SignatureVersion: SignatureVersionV4,
		Region:           "cn-hangzhou",
	}
	req := &AssembledRequest{
		Action:   "DescribeInstances",
		Version:  "2014-05-26",
		Method:   "POST",
		Protocol: "HTTPS",
		Style:    "RPC",
		Endpoint: "ecs.cn-hangzhou.aliyuncs.com",
	}
	call, err := prepareCall(ec, req)
	if err != nil {
		t.Fatalf("prepareCall: %v", err)
	}
	if got := tea.StringValue(call.client.ProductId); got != "Ecs" {
		t.Fatalf("ProductId = %q, want Ecs", got)
	}
}

func TestPrepareCallSignatureV4EmptyMeansLegacy(t *testing.T) {
	ec := &ExecContext{
		API: &meta.API{
			ProductCode: "Ecs",
			Name:        "DescribeInstances",
			Version:     "2014-05-26",
		},
		Credential: staticAKCredential(t),
		Region:     "cn-hangzhou",
	}
	req := &AssembledRequest{
		Action:   "DescribeInstances",
		Version:  "2014-05-26",
		Method:   "POST",
		Protocol: "HTTPS",
		Style:    "RPC",
		Endpoint: "ecs.cn-hangzhou.aliyuncs.com",
	}
	call, err := prepareCall(ec, req)
	if err != nil {
		t.Fatalf("prepareCall: %v", err)
	}
	if call.client.SignatureVersion != nil && tea.StringValue(call.client.SignatureVersion) != "" {
		t.Fatalf("unexpected SignatureVersion %q", tea.StringValue(call.client.SignatureVersion))
	}
}

func TestExecuteSSERejectsSignatureV2AndV4(t *testing.T) {
	for _, version := range []string{SignatureVersionV2, SignatureVersionV4} {
		err := NewExecutor().ExecuteSSE(&ExecContext{
			API: &meta.API{
				ProductCode: "Ecs",
				Name:        "DescribeInstances",
				Version:     "2014-05-26",
				IsSSE:       true,
			},
			Credential:       staticAKCredential(t),
			SignatureVersion: version,
			Region:           "cn-hangzhou",
		}, nil)
		if err == nil || err.Error() != "runtime: SSE API does not support signature version v2 or v4" {
			t.Fatalf("ExecuteSSE(signature=%s) error = %v", version, err)
		}
	}
}

func TestPrepareCallLegacySignatureOverridesV4(t *testing.T) {
	ec := &ExecContext{
		API: &meta.API{
			ProductCode: "fnf",
			Name:        "ListFlows",
			Version:     "2019-03-15",
		},
		Credential:       staticAKCredential(t),
		SignatureVersion: SignatureVersionV4,
		Region:           "cn-hangzhou",
	}
	req := &AssembledRequest{
		Action:   "ListFlows",
		Version:  "2019-03-15",
		Method:   "GET",
		Protocol: "HTTPS",
		Style:    "RPC",
		Endpoint: "fnf.cn-hangzhou.aliyuncs.com",
	}
	call, err := prepareCall(ec, req)
	if err != nil {
		t.Fatalf("prepareCall: %v", err)
	}
	if tea.StringValue(call.client.SignatureAlgorithm) != "v2" {
		t.Fatalf("SignatureAlgorithm = %q, want v2", tea.StringValue(call.client.SignatureAlgorithm))
	}
	if call.client.SignatureVersion != nil && tea.StringValue(call.client.SignatureVersion) != "" {
		t.Fatalf("legacy product must not set SignatureVersion, got %q", tea.StringValue(call.client.SignatureVersion))
	}
	if call.client.Spi != nil {
		t.Fatalf("legacy product must not install POP/SLS gateway, got %T", call.client.Spi)
	}
}
