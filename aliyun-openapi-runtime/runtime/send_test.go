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
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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

	resp, err := NewExecutor().Execute(ec)
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

func TestSendProductSignatureAlgorithm(t *testing.T) {
	form := map[string]string{"FlowName": "test-flow", "Input": `{"key":"a+b&中文"}`}
	query := map[string]string{"Limit": "10", "NextToken": "next+&中文"}
	tests := []struct {
		name, product, action, method, token string
		legacy                               bool
		query, form                          map[string]string
	}{
		{name: "fnf POST", product: "fnf", action: "StartSyncExecution", method: "POST", legacy: true, form: form},
		{name: "fnf GET", product: "fnf", action: "ListFlows", method: "GET", legacy: true, query: query},
		{name: "mixed-case FNF", product: "FnF", action: "StartSyncExecution", method: "POST", legacy: true, query: query, form: form},
		{name: "default ACS3", product: "demo", action: "SubmitForm", method: "POST", query: query, form: form},
		{name: "similar product is ACS3", product: "fnf-other", action: "StartSyncExecution", method: "POST", form: form},
		{name: "FNF STS", product: "fnf", action: "StartSyncExecution", method: "POST", legacy: true, token: "fake-sts-token", form: form},
		{name: "ACS3 STS", product: "demo", action: "SubmitForm", method: "POST", token: "fake-sts-token", form: form},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotQuery, gotForm url.Values
			var gotHeader http.Header
			var gotMethod string
			var captureErr error
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotHeader, gotQuery = r.Method, r.Header.Clone(), r.URL.Query()
				var body []byte
				body, captureErr = io.ReadAll(r.Body)
				if captureErr == nil {
					gotForm, captureErr = url.ParseQuery(string(body))
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"RequestId":"test","Total":9007199254740993}`))
			}))
			defer srv.Close()

			api := &meta.API{
				Name: tt.action, ProductCode: tt.product, Version: "2019-03-15", Method: tt.method,
				Style: meta.StyleRPC, Protocol: "HTTP",
			}
			args := map[string]any{}
			for name, value := range tt.query {
				api.Parameters = append(api.Parameters, meta.Parameter{Name: name, RawName: name, Type: meta.TypeString, Position: meta.PosQuery})
				args[name] = value
			}
			for name, value := range tt.form {
				api.Parameters = append(api.Parameters, meta.Parameter{Name: name, RawName: name, Type: meta.TypeString, Position: meta.PosFormData})
				args[name] = value
				api.ReqBodyType = "formData"
			}
			credential := staticAKCredential(t)
			if tt.token != "" {
				var err error
				credential, err = credentialsv2.NewCredential(new(credentialsv2.Config).
					SetType("sts").SetAccessKeyId("LTAI-test-id").SetAccessKeySecret("test-secret").SetSecurityToken(tt.token))
				if err != nil {
					t.Fatal(err)
				}
			}
			response, err := NewExecutor().Execute(&ExecContext{
				API: api, Args: args, Credential: credential, Endpoint: strings.TrimPrefix(srv.URL, "http://"),
			})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if captureErr != nil {
				t.Fatal(captureErr)
			}
			if response.StatusCode != 200 || response.Parsed.(map[string]any)["Total"] != json.Number("9007199254740993") {
				t.Fatal("response status or integer precision changed")
			}
			if gotMethod != tt.method {
				t.Fatalf("method = %q, want %q", gotMethod, tt.method)
			}
			for name, value := range tt.query {
				if gotQuery.Get(name) != value || gotForm.Has(name) {
					t.Errorf("query parameter %s changed value or location", name)
				}
			}
			for name, value := range tt.form {
				if gotForm.Get(name) != value || gotQuery.Has(name) {
					t.Errorf("form parameter %s changed value or location", name)
				}
			}
			if len(gotForm) != len(tt.form) {
				t.Fatalf("form has %d fields, want %d", len(gotForm), len(tt.form))
			}
			if len(tt.form) > 0 && gotHeader.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Fatalf("Content-Type = %q", gotHeader.Get("Content-Type"))
			}
			if !tt.legacy {
				if !strings.HasPrefix(gotHeader.Get("Authorization"), "ACS3-HMAC-SHA256 ") || len(gotQuery) != len(tt.query) {
					t.Fatal("ACS3 request switched to legacy signing")
				}
				if gotHeader.Get("x-acs-security-token") != tt.token {
					t.Fatal("ACS3 security token changed")
				}
				return
			}
			if gotHeader.Get("Authorization") != "" || gotHeader.Get("x-acs-content-sha256") != "" {
				t.Fatal("legacy RPC request contains ACS3 signing headers")
			}
			for name, want := range map[string]string{
				"Action": tt.action, "Version": api.Version, "Format": "json", "AccessKeyId": "LTAI-test-id",
				"SignatureMethod": "HMAC-SHA1", "SignatureVersion": "1.0", "SecurityToken": tt.token,
			} {
				if gotQuery.Get(name) != want {
					t.Errorf("legacy RPC public query parameter %s changed", name)
				}
			}
			if gotQuery.Get("Timestamp") == "" || gotQuery.Get("SignatureNonce") == "" || gotHeader.Get("x-acs-security-token") != "" {
				t.Fatal("legacy RPC public signing fields are missing or misplaced")
			}
			// Verify the SDK signs both query and form parameters, including STS and escaped input.
			signature := gotQuery.Get("Signature")
			gotQuery.Del("Signature")
			for name, values := range gotForm {
				gotQuery[name] = values
			}
			canonical := strings.ReplaceAll(gotQuery.Encode(), "+", "%20")
			mac := hmac.New(sha1.New, []byte("test-secret&"))
			_, _ = mac.Write([]byte(tt.method + "&%2F&" + url.QueryEscape(canonical)))
			if signature != base64.StdEncoding.EncodeToString(mac.Sum(nil)) {
				t.Fatal("legacy RPC signature does not cover query and form parameters")
			}
		})
	}
}

func TestExecuteSSERejectsLegacySignature(t *testing.T) {
	for _, isSSE := range []bool{false, true} {
		err := NewExecutor().ExecuteSSE(&ExecContext{
			API: &meta.API{ProductCode: "FnF", IsSSE: isSSE},
		}, nil)
		if err == nil || err.Error() != "runtime: SSE does not support signature algorithm v2" {
			t.Fatalf("ExecuteSSE(IsSSE=%t) error = %v", isSSE, err)
		}
	}
}

func TestSendRetriesThrottlingResponse(t *testing.T) {
	requests := 0
	var retryAttempts, retryDelay string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("x-acs-retry-after", "0")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"Code":"Throttling","Message":"slow down","RequestId":"req-1"}`))
			return
		}
		retryAttempts = r.Header.Get("x-acs-retry-attempts")
		retryDelay = r.Header.Get("x-acs-retry-delay")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"RequestId":"req-2"}`))
	}))
	defer srv.Close()

	originalSleep := throttlingRetrySleep
	throttlingRetrySleep = func(time.Duration) {}
	t.Cleanup(func() { throttlingRetrySleep = originalSleep })

	api := &meta.API{
		Name: "DescribeThing", Version: "2024-04-02", Method: "POST",
		Style: meta.StyleRPC, Protocol: "HTTP", ProductCode: "acc",
	}
	_, err := NewExecutor().Execute(&ExecContext{
		API:        api,
		Endpoint:   strings.TrimPrefix(srv.URL, "http://"),
		Credential: staticAKCredential(t),
		Args:       map[string]any{},
		Transport: TransportOptions{ThrottlingRetry: ThrottlingRetryOptions{
			MaxAttempts: 1,
		}},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if retryAttempts != "1" || retryDelay != "0" {
		t.Fatalf("retry headers attempts=%q delay=%q", retryAttempts, retryDelay)
	}
}

func TestSendFormDataLetsSDKSetContentType(t *testing.T) {
	var gotContentType string
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotForm, _ = url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	api := &meta.API{
		Name: "SubmitForm", Version: "2024-01-01", Method: "POST", Style: meta.StyleRPC,
		Protocol: "HTTP", ProductCode: "demo", ReqBodyType: "formData",
		Parameters: []meta.Parameter{
			{Name: "field", RawName: "Field", Type: meta.TypeString, Position: meta.PosFormData},
			{Name: "body", RawName: "body", Type: meta.TypeArray, Position: meta.PosFormData, ParamStyle: "json",
				ItemType: &meta.Parameter{Type: meta.TypeObject}},
		},
	}
	ec := &ExecContext{
		API: api, Endpoint: strings.TrimPrefix(srv.URL, "http://"), Credential: staticAKCredential(t),
		Args: map[string]any{
			"Field": "two words",
			"body":  []any{map[string]any{"ReferenceId": "01"}},
		},
	}

	assembled, err := Assemble(ec)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if _, exists := assembled.Headers["content-type"]; exists {
		t.Fatalf("runtime must leave formData content-type to the SDK: %#v", assembled.Headers)
	}
	if _, err := NewExecutor().Execute(ec); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("wire Content-Type = %q", gotContentType)
	}
	if gotForm.Get("Field") != "two words" {
		t.Fatalf("wire Field = %q; form=%v", gotForm.Get("Field"), gotForm)
	}
	if gotForm.Get("body") != `[{"ReferenceId":"01"}]` {
		t.Fatalf("wire body field = %q; form=%v", gotForm.Get("body"), gotForm)
	}
	if gotForm.Get("body.1.ReferenceId") != "" {
		t.Fatalf("json-style form field was flattened: %v", gotForm)
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

	resp, err := NewExecutor().Execute(ec)
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

// TestSendROAEncodedPathAgainstMockServer verifies that the Go-plugin-compatible
// pathname produced by Assemble is preserved on the actual HTTP request URI.
func TestSendROAEncodedPathAgainstMockServer(t *testing.T) {
	var gotRequestURI string
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestURI = r.RequestURI
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	api := &meta.API{
		Name: "GetThing", Version: "2024-01-01", Method: "GET", Style: meta.StyleROA,
		Protocol: "HTTP", URL: "/things/{thingId}:inspect", ProductCode: "demo",
		Parameters: []meta.Parameter{
			{Name: "thing_id", RawName: "thingId", Type: meta.TypeString, Position: meta.PosPath, Required: true},
		},
	}
	_, err := NewExecutor().Execute(&ExecContext{
		API: api, Endpoint: strings.TrimPrefix(srv.URL, "http://"), Credential: staticAKCredential(t),
		Args: map[string]any{"thingId": "thing-123"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "/things/thing-123%3Ainspect"; gotRequestURI != want {
		t.Fatalf("request URI = %q, want %q", gotRequestURI, want)
	}
	if gotAuth == "" {
		t.Fatal("request was not signed")
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
					Name: "body", RawName: "body", Type: meta.TypeAny, Position: meta.PosBody, DirectBody: true,
				}},
			}
			_, err := NewExecutor().Execute(&ExecContext{
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
