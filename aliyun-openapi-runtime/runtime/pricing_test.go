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
	"errors"
	"testing"
)

func TestIsAPIDryRunRequested(t *testing.T) {
	if IsAPIDryRunRequested(nil) {
		t.Fatal("nil should be false")
	}
	if IsAPIDryRunRequested(&AssembledRequest{Query: map[string]string{}}) {
		t.Fatal("missing DryRun should be false")
	}
	for _, v := range []string{"true", "True", "TRUE", "1"} {
		req := &AssembledRequest{Query: map[string]string{"DryRun": v}}
		if !IsAPIDryRunRequested(req) {
			t.Fatalf("DryRun=%q should be requested", v)
		}
	}
	req := &AssembledRequest{Query: map[string]string{"DryRun": "false"}}
	if IsAPIDryRunRequested(req) {
		t.Fatal("DryRun=false should not be requested")
	}
}

func TestIsDryRunPassError(t *testing.T) {
	if IsDryRunPassError(nil) {
		t.Fatal("nil should be false")
	}
	if IsDryRunPassError(errors.New("something else")) {
		t.Fatal("unrelated error should be false")
	}
	if !IsDryRunPassError(errors.New(`SDKError:\n   StatusCode: 400\n   Code: DryRunOperation\n   Message: Request validation has been passed with DryRun flag set`)) {
		t.Fatal("Code: DryRunOperation should match")
	}
	if !IsDryRunPassError(errors.New(`{"Code":"DryRunOperation","Message":"ok"}`)) {
		t.Fatal("JSON Code should match")
	}
}

func TestStripAPIDryRun(t *testing.T) {
	StripAPIDryRun(nil)
	req := &AssembledRequest{Query: map[string]string{"DryRun": "true", "RegionId": "cn-hangzhou"}}
	StripAPIDryRun(req)
	if _, ok := req.Query["DryRun"]; ok {
		t.Fatal("DryRun should be removed")
	}
	if req.Query["RegionId"] != "cn-hangzhou" {
		t.Fatal("other query keys must remain")
	}
}

func TestBuildPriceRequestIncludesPricingContext(t *testing.T) {
	t.Setenv(pricingPopCodeEnv, "ecs")
	ec := &ExecContext{Region: "cn-hangzhou"}
	req := &AssembledRequest{
		Action:  "RunInstances",
		Version: "2014-05-26",
		Query:   map[string]string{"Amount": "1"},
		Body:    map[string]any{"SpotStrategy": "SpotAsPriceGo"},
	}

	got := buildPriceRequest(ec, req, map[string]string{"ZoneId": "cn-hangzhou-i"})
	if got.PopCode != "ecs" || got.PopVersion != req.Version || got.ApiName != req.Action {
		t.Fatalf("unexpected price request identity: %#v", got)
	}
	if got.Parameters["Amount"] != "1" || got.Parameters["SpotStrategy"] != "SpotAsPriceGo" || got.Parameters["RegionId"] != "cn-hangzhou" {
		t.Fatalf("unexpected price request parameters: %#v", got.Parameters)
	}
	context, ok := got.Parameters["PricingContext"].(map[string]interface{})
	if !ok || context["ZoneId"] != "cn-hangzhou-i" {
		t.Fatalf("unexpected pricing context: %#v", got.Parameters["PricingContext"])
	}
}
