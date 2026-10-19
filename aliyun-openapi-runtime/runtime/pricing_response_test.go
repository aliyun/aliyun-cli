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
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestPricingResponseClassification(t *testing.T) {
	for _, test := range []struct {
		name, body      string
		failed, invalid bool
	}{
		{"priced", `{"success":true,"code":"Success","message":"ok","requestId":"r","price":{"success":true}}`, false, false},
		{"free", `{"success":true,"code":"PricingNotRequired","message":"free","requestId":"r","price":null}`, false, false},
		{"unsupported", `{"success":false,"code":"PricingNotSupported","message":"unsupported","requestId":"r","price":null}`, true, false},
		{"top level controls failure", `{"success":false,"code":"UpstreamError","message":"failed","requestId":"r","price":{"success":true}}`, true, false},
		{"legacy price", `{"price":{"success":true},"requestId":"r"}`, false, false},
		{"legacy failed price", `{"price":{"success":false,"errorCode":"InvalidParameter"},"requestId":"r"}`, true, false},
		{"legacy free", `{"Code":"PricingNotRequired","Message":"free","RequestId":"r"}`, false, false},
		{"legacy unsupported", `{"Code":"PricingNotSupported","Message":"unsupported","RequestId":"r"}`, true, false},
		{"missing success", `{"code":"Success","message":"ok","requestId":"r","price":{}}`, false, true},
		{"missing request id", `{"success":true,"code":"Success","message":"ok","price":{}}`, false, true},
		{"missing price", `{"success":true,"code":"Success","message":"ok","requestId":"r"}`, false, true},
		{"wrong success type", `{"success":"false","code":"Success","message":"ok","requestId":"r","price":null}`, false, true},
		{"null message", `{"success":true,"code":"Success","message":null,"requestId":"r","price":null}`, false, true},
		{"wrong price type", `{"success":true,"code":"Success","message":"ok","requestId":"r","price":42}`, false, true},
		{"unknown shape", `{}`, false, true},
		{"not json", `<html>bad gateway</html>`, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := PricingResponseError(test.body)
			var business *PricingBusinessError
			if errors.As(err, &business) != test.failed {
				t.Fatalf("business result = %v", err)
			}
			if (err != nil) != (test.failed || test.invalid) {
				t.Fatalf("result = %v", err)
			}
		})
	}
}

func TestEstimateCostPreservesBusinessEnvelopes(t *testing.T) {
	for _, test := range []struct {
		body   string
		status int
	}{
		{`{"success":true,"code":"Success","message":"quoted","requestId":"r","price":{"success":true,"totalAmount":0.084},"newField":"preserve me"}`, 200},
		{`{"success":true,"code":"PricingNotRequired","message":"服务端免费","requestId":"r","price":null}`, 200},
		{`{"success":false,"code":"PricingNotSupported","message":"服务端不支持","requestId":"r","price":null}`, 200},
		{`{"Code":"PricingNotRequired","Message":"legacy free","RequestId":"r"}`, http.StatusBadRequest},
		{`{"Code":"PricingNotSupported","Message":"legacy unsupported","RequestId":"r"}`, http.StatusBadRequest},
	} {
		ec, req := quoteResponder(t, test.status, test.body)
		out, err := EstimateCost(ec, req, nil)
		if err != nil {
			t.Fatal(err)
		}
		var before, after interface{}
		_ = json.Unmarshal([]byte(test.body), &before)
		_ = json.Unmarshal([]byte(out), &after)
		left, _ := json.Marshal(before)
		right, _ := json.Marshal(after)
		if string(left) != string(right) {
			t.Fatalf("response changed: %s", out)
		}
	}
}
