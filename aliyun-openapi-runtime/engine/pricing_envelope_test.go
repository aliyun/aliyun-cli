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
package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
	credentials "github.com/aliyun/credentials-go/credentials"
)

func TestEstimateCostEnvelopeAndFilteredExitStatus(t *testing.T) {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		t.Setenv(key, "")
	}
	t.Setenv("ALIBABA_CLOUD_PRICING_HOST", "")
	credential, err := credentials.NewCredential(&credentials.Config{
		Type: tea.String("access_key"), AccessKeyId: tea.String("fixture-ak"), AccessKeySecret: tea.String("fixture-secret"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ec := &runtime.ExecContext{
		Credential: credential, SkipSecureVerify: true, Language: "zh", Args: map[string]any{},
		API: &meta.API{Name: "Quote", ProductCode: "demo", Version: "v1", Style: meta.StyleRPC},
	}
	for _, tc := range []struct {
		name, body string
		status     int
		failed     bool
	}{
		{"quoted", `{"success":true,"code":"Success","message":"server message","requestId":"r","price":{"success":true,"totalAmount":0.084}}`, 200, false},
		{"free", `{"success":true,"code":"PricingNotRequired","message":"server message","requestId":"r"}`, 200, false},
		{"unsupported", `{"success":false,"code":"PricingNotSupported","message":"server message","requestId":"r","price":null}`, 200, true},
		{"legacy free", `{"Code":"PricingNotRequired","Message":"server message","RequestId":"r"}`, 400, false},
		{"legacy unsupported", `{"Code":"PricingNotSupported","Message":"server message","RequestId":"r"}`, 400, true},
		{"failed price", `{"price":{"success":false,"errorCode":"UpstreamFailure","errorMessage":"server message"},"requestId":"r"}`, 200, true},
	} {
		for _, mode := range []string{"full", "query", "quiet", "table"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					calls.Add(1)
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Error(err)
						return
					}
					if _, present := payload["responseVersion"]; present {
						t.Error("quote selected response version")
					}
					if payload["language"] != "zh" || req.URL.Path != "/api/v1/price/quote" {
						t.Errorf("unexpected quote request: %v", payload)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.status)
					_, _ = io.WriteString(w, tc.body)
				}))
				t.Cleanup(server.Close)
				t.Setenv("ALIBABA_CLOUD_PRICING_ENDPOINT", server.Listener.Addr().String())
				executor := &responseExecutor{}
				engine := newBuilderCoverageEngine(executor)
				res := &argparser.Result{}
				if mode == "query" {
					res.Reserved.CliQuery = "requestId || RequestId"
				}
				if mode == "table" {
					res.Reserved.OutputTable = &argparser.OutputTableConfig{Cols: []string{"requestId", "RequestId"}}
				}
				res.Reserved.Quiet = mode == "quiet"
				var out bytes.Buffer
				err := engine.executeEstimateCost(&out, ec, res)
				var business *runtime.PricingBusinessError
				if errors.As(err, &business) != tc.failed || (err != nil) != tc.failed {
					t.Fatalf("result=%v", err)
				}
				if tc.failed && business.ExitCode() != 3 {
					t.Fatal("failed quote must exit 3")
				}
				if executor.calls != 0 || calls.Load() != 1 {
					t.Fatalf("target calls=%d quote calls=%d", executor.calls, calls.Load())
				}
				switch mode {
				case "quiet":
					if out.Len() != 0 {
						t.Fatalf("quiet output=%s", &out)
					}
				case "query":
					if strings.TrimSpace(out.String()) != `"r"` {
						t.Fatalf("query output=%s", &out)
					}
				case "table":
					if !strings.Contains(out.String(), "r") {
						t.Fatalf("table output=%s", &out)
					}
				default:
					var before, after any
					_ = json.Unmarshal([]byte(tc.body), &before)
					err = json.Unmarshal(out.Bytes(), &after)
					if err != nil || !reflect.DeepEqual(before, after) {
						t.Fatalf("changed envelope: %s", &out)
					}
				}
			})
		}
	}
}

func TestEstimateCostExplicitAPIDryRun(t *testing.T) {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		t.Setenv(key, "")
	}
	t.Setenv("ALIBABA_CLOUD_PRICING_HOST", "")
	credential, err := credentials.NewCredential(&credentials.Config{Type: tea.String("access_key"), AccessKeyId: tea.String("ak"), AccessKeySecret: tea.String("secret")})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		precheck error
		quotes   int
	}{
		{"pass", errors.New("Code: DryRunOperation"), 1},
		{"rejected", errors.New("invalid target parameters"), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var payload struct {
					Parameters map[string]any `json:"parameters"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
					return
				}
				if _, present := payload.Parameters["DryRun"]; present {
					t.Error("precheck flag leaked into quote")
				}
				_, _ = io.WriteString(w, `{"price":{"success":true}}`)
			}))
			defer server.Close()
			t.Setenv("ALIBABA_CLOUD_PRICING_ENDPOINT", server.Listener.Addr().String())
			executor := &responseExecutor{err: tc.precheck}
			engine := newBuilderCoverageEngine(executor)
			ec := &runtime.ExecContext{Credential: credential, SkipSecureVerify: true, API: &meta.API{Name: "Run", ProductCode: "demo", Version: "v1", Style: meta.StyleRPC}, ExtraQuery: map[string]string{"DryRun": "true"}}
			err := engine.executeEstimateCost(io.Discard, ec, &argparser.Result{})
			if (err == nil) != (tc.quotes == 1) || executor.calls != 1 || int(calls.Load()) != tc.quotes {
				t.Fatalf("error=%v prechecks=%d quotes=%d", err, executor.calls, calls.Load())
			}
		})
	}
}
