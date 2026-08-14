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

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestAssembleAppliesTransportHeadersAndRPCCallContext(t *testing.T) {
	ec := &ExecContext{
		API:  rpcAPI(),
		Args: map[string]any{},
		ExtraHeaders: map[string]string{
			"traceparent": "user-value",
		},
		Transport: TransportOptions{
			Headers: map[string]string{
				"traceparent": "host-value",
				"baggage":     "tenant=\r\ntest",
				"bad header":  "ignored",
			},
			CallContext: CallContextOptions{
				SourceIP:        "192.0.2.1",
				SecureTransport: "true",
			},
		},
	}
	req, err := Assemble(ec)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if got := req.Headers["traceparent"]; got != "host-value" {
		t.Fatalf("traceparent = %q, want host-value", got)
	}
	if got := req.Headers["baggage"]; got != "tenant=test" {
		t.Fatalf("baggage = %q", got)
	}
	if _, exists := req.Headers["bad header"]; exists {
		t.Fatal("invalid host-managed header was injected")
	}
	if got := req.Query["SourceIp"]; got != "192.0.2.1" {
		t.Fatalf("SourceIp = %q", got)
	}
	if got := req.Query["SecureTransport"]; got != "true" {
		t.Fatalf("SecureTransport = %q", got)
	}
}

func TestAssembleAppliesROACallContextWithoutOverwriting(t *testing.T) {
	api := &meta.API{
		Name: "GetThing", Version: "2024-01-01", Style: meta.StyleROA,
		ProductCode: "demo",
	}
	ec := &ExecContext{
		API:          api,
		Args:         map[string]any{},
		ExtraHeaders: map[string]string{"x-acs-source-ip": "explicit"},
		Transport: TransportOptions{CallContext: CallContextOptions{
			SourceIP:        "192.0.2.1",
			SecureTransport: "yes",
		}},
	}
	req, err := Assemble(ec)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if got := req.Headers["x-acs-source-ip"]; got != "explicit" {
		t.Fatalf("source IP was overwritten: %q", got)
	}
	if got := req.Headers["x-acs-secure-transport"]; got != "yes" {
		t.Fatalf("secure transport = %q", got)
	}
}

func TestAssembleSkipsCallContextProducts(t *testing.T) {
	for _, product := range []string{"sls", "PDS", "custom"} {
		t.Run(product, func(t *testing.T) {
			api := rpcAPI()
			api.ProductCode = product
			req, err := Assemble(&ExecContext{
				API:  api,
				Args: map[string]any{},
				Transport: TransportOptions{CallContext: CallContextOptions{
					SourceIP:     "192.0.2.1",
					SkipProducts: []string{"CUSTOM"},
				}},
			})
			if err != nil {
				t.Fatalf("Assemble: %v", err)
			}
			if _, exists := req.Query["SourceIp"]; exists {
				t.Fatalf("SourceIp injected for skipped product %q", product)
			}
		})
	}
}
