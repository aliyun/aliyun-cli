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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	credentialsv2 "github.com/aliyun/credentials-go/credentials"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func staticAKCredential(t *testing.T) credentialsv2.Credential {
	t.Helper()
	conf := new(credentialsv2.Config).
		SetType("access_key").
		SetAccessKeyId("LTAI-test-id").
		SetAccessKeySecret("test-secret")
	cred, err := credentialsv2.NewCredential(conf)
	if err != nil {
		t.Fatalf("build credential: %v", err)
	}
	return cred
}

// TestSendAgainstMockServer exercises the REAL send path end-to-end
// (assemble -> sign -> HTTP -> decode) without hitting Alibaba Cloud.
// A local httptest server stands in for the OpenAPI gateway: it echoes
// nothing back except a body carrying a large integer, letting us
// assert that the request actually left the client AND that response
// decoding preserves int64 precision via UseNumber.
func TestSendAgainstMockServer(t *testing.T) {
	var gotQuery url.Values
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		// 2^53+1 would lose precision through float64.
		_, _ = w.Write([]byte(`{"RequestId":"req-1","Total":9007199254740993}`))
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")

	api := &meta.API{
		Name:        "DescribeThing",
		Version:     "2024-04-02",
		Method:      "POST",
		Style:       meta.StyleRPC,
		Protocol:    "HTTP", // talk plain HTTP to httptest
		ProductCode: "acc",
		Parameters: []meta.Parameter{
			{Name: "region_id", RawName: "RegionId", Type: meta.TypeString, Position: meta.PosQuery},
		},
	}
	ec := &ExecContext{
		API:        api,
		Region:     "cn-hangzhou",
		Endpoint:   host,
		Credential: staticAKCredential(t),
		Args:       map[string]any{"RegionId": "cn-hangzhou"},
	}

	resp, err := NewExecutor().Execute(context.Background(), ec)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	// Request actually reached the server, signed, with our param.
	if gotAuth == "" || !strings.Contains(gotAuth, "ACS") {
		t.Errorf("expected signed Authorization header, got %q", gotAuth)
	}
	if gotQuery.Get("RegionId") != "cn-hangzhou" {
		t.Errorf("server did not receive RegionId; query=%v", gotQuery)
	}

	// Response decoded with UseNumber: the big integer survived intact.
	m, ok := resp.Parsed.(map[string]any)
	if !ok {
		t.Fatalf("parsed body not an object: %T", resp.Parsed)
	}
	num, ok := m["Total"].(json.Number)
	if !ok {
		t.Fatalf("Total not json.Number: %T (%v)", m["Total"], m["Total"])
	}
	if num.String() != "9007199254740993" {
		t.Fatalf("precision lost: %s", num.String())
	}
}

// TestSendROAAgainstMockServer exercises the ROA path against a
// mock gateway: the path placeholder must be substituted, the request
// must actually reach the templated path with the query attached, and
// AssembledRequest.Style "ROA" is forwarded to the SDK as-is.
func TestSendROAAgainstMockServer(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"triggers":[{"name":"t1"}]}`))
	}))
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")

	api := &meta.API{
		Name:        "ListTriggers",
		Version:     "2023-03-30",
		Method:      "GET",
		Style:       meta.StyleROA,
		Protocol:    "HTTP",
		URL:         "/2023-03-30/functions/{functionName}/triggers",
		ProductCode: "fc",
		Parameters: []meta.Parameter{
			{Name: "function_name", RawName: "functionName", Type: meta.TypeString, Position: meta.PosPath, Required: true},
			{Name: "prefix", RawName: "prefix", Type: meta.TypeString, Position: meta.PosQuery},
		},
	}
	ec := &ExecContext{
		API:        api,
		Region:     "cn-hangzhou",
		Endpoint:   host,
		Credential: staticAKCredential(t),
		Args:       map[string]any{"functionName": "my-func", "prefix": "web"},
	}

	resp, err := NewExecutor().Execute(context.Background(), ec)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	// Path placeholder substituted AND the SDK routed it as ROA (so
	// the templated pathname is used as the request path).
	if gotPath != "/2023-03-30/functions/my-func/triggers" {
		t.Fatalf("server path = %q, want the substituted ROA path", gotPath)
	}
	if gotQuery.Get("prefix") != "web" {
		t.Errorf("query prefix = %q", gotQuery.Get("prefix"))
	}
}

func TestSendDirectAnyBodyAgainstMockServer(t *testing.T) {
	tests := []struct {
		name string
		body any
		want string
	}{
		{name: "object", body: map[string]any{"enabled": true, "large": json.Number("9007199254740993")}, want: `{"enabled":true,"large":9007199254740993}`},
		{name: "null", body: nil, want: `null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotBody, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			}))
			defer srv.Close()

			api := &meta.API{
				Name: "UpdateThing", Version: "2020-01-01", Method: "POST", Style: meta.StyleROA,
				Protocol: "HTTP", URL: "/things", ProductCode: "demo",
				Parameters: []meta.Parameter{{
					Name: "body", RawName: "body", Type: meta.TypeAny, Position: meta.PosBody,
				}},
			}
			_, err := NewExecutor().Execute(context.Background(), &ExecContext{
				API: api, Endpoint: strings.TrimPrefix(srv.URL, "http://"), Credential: staticAKCredential(t),
				Args: map[string]any{"body": tt.body},
			})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if string(gotBody) != tt.want {
				t.Fatalf("wire body = %q, want %q", gotBody, tt.want)
			}
		})
	}
}

// TestDarabonbaStyleMapping locks the model->SDK style vocabulary.
func TestDarabonbaStyleMapping(t *testing.T) {
	cases := map[string]string{
		"RESTful": "ROA", "ROA": "ROA", "restful": "ROA",
		"RPC": "RPC", "": "RPC",
	}
	for in, want := range cases {
		if got := darabonbaStyle(in); got != want {
			t.Errorf("darabonbaStyle(%q) = %q, want %q", in, got, want)
		}
	}
}
