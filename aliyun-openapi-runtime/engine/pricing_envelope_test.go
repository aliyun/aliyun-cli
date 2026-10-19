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
	"strings"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
	credentials "github.com/aliyun/credentials-go/credentials"
)

type quoteTransport func(*http.Request) (*http.Response, error)

func (f quoteTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestEstimateCostEnvelopeAndFilteredExitStatus(t *testing.T) {
	credential, err := credentials.NewCredential(&credentials.Config{
		Type: tea.String("access_key"), AccessKeyId: tea.String("fixture-ak"), AccessKeySecret: tea.String("fixture-secret"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ec := &runtime.ExecContext{
		Credential: credential, Language: "zh", Args: map[string]any{},
		API: &meta.API{Name: "Quote", ProductCode: "demo", Version: "v1", Style: meta.StyleRPC},
	}
	engine := newBuilderCoverageEngine(&responseExecutor{})
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	for _, code := range []string{"Success", "PricingNotRequired", "PricingNotSupported"} {
		for _, mode := range []string{"full", "query", "quiet"} {
			t.Run(code+"/"+mode, func(t *testing.T) {
				failed := code == "PricingNotSupported"
				envelope := map[string]any{"success": !failed, "code": code, "message": "server message", "requestId": "r"}
				if code == "Success" {
					envelope["price"] = map[string]any{"success": true, "totalAmount": 0.084}
				}
				body, _ := json.Marshal(envelope)
				http.DefaultTransport = quoteTransport(func(req *http.Request) (*http.Response, error) {
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					if payload["responseVersion"] != "2" || payload["language"] != "zh" || req.URL.Path != "/api/v1/price/quote" {
						t.Fatalf("unexpected quote request: %v", payload)
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body)), Request: req}, nil
				})
				res := &argparser.Result{}
				if mode == "query" {
					res.Reserved.CliQuery = "requestId"
				}
				res.Reserved.Quiet = mode == "quiet"
				var out bytes.Buffer
				err := engine.executeEstimateCost(&out, ec, res)
				var business *runtime.PricingBusinessError
				if errors.As(err, &business) != failed || (err != nil) != failed {
					t.Fatalf("result = %v", err)
				}
				if failed && business.ExitCode() != 3 {
					t.Fatal("failed quote must exit 3")
				}
				switch mode {
				case "quiet":
					if out.Len() != 0 {
						t.Fatalf("quiet output: %s", &out)
					}
				case "query":
					if strings.TrimSpace(out.String()) != `"r"` {
						t.Fatalf("query output: %s", &out)
					}
				default:
					var compact bytes.Buffer
					if err := json.Compact(&compact, out.Bytes()); err != nil || !bytes.Equal(compact.Bytes(), body) {
						t.Fatalf("changed envelope: %s", &out)
					}
				}
			})
		}
	}
}
