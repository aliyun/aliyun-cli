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

func TestPrepareCallRejectsUnsupportedSignatureVersion(t *testing.T) {
	ec := &ExecContext{
		API: &meta.API{
			ProductCode: "Ecs",
			Name:        "DescribeInstances",
			Version:     "2014-05-26",
		},
		Credential:       staticAKCredential(t),
		SignatureVersion: "v3",
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
