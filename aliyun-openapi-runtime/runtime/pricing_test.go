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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	credentialsv2 "github.com/aliyun/credentials-go/credentials"
)

type pricingTestCredential struct {
	model *credentialsv2.CredentialModel
	err   error
}

func (c pricingTestCredential) GetAccessKeyId() (*string, error) {
	if c.model == nil {
		return nil, c.err
	}
	return c.model.AccessKeyId, c.err
}

func (c pricingTestCredential) GetAccessKeySecret() (*string, error) {
	if c.model == nil {
		return nil, c.err
	}
	return c.model.AccessKeySecret, c.err
}

func (c pricingTestCredential) GetSecurityToken() (*string, error) {
	if c.model == nil {
		return nil, c.err
	}
	return c.model.SecurityToken, c.err
}

func (c pricingTestCredential) GetBearerToken() *string { return nil }
func (c pricingTestCredential) GetType() *string        { return nil }
func (c pricingTestCredential) GetCredential() (*credentialsv2.CredentialModel, error) {
	return c.model, c.err
}

func stringPointer(value string) *string { return &value }

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

func TestPriceModeEnabled(t *testing.T) {
	for _, test := range []struct {
		flag bool
		env  string
		want bool
	}{
		{flag: true, want: true},
		{env: "1", want: true},
		{env: "TRUE", want: true},
		{env: "false", want: false},
	} {
		t.Setenv(priceModeEnv, test.env)
		if got := PriceModeEnabled(test.flag); got != test.want {
			t.Fatalf("PriceModeEnabled(%v) with env %q = %v, want %v", test.flag, test.env, got, test.want)
		}
	}
}

func TestEstimateCostValidatesContext(t *testing.T) {
	if _, err := EstimateCost(nil, &AssembledRequest{}, nil); err == nil {
		t.Fatal("EstimateCost(nil context) succeeded")
	}
	if _, err := EstimateCost(&ExecContext{}, nil, nil); err == nil {
		t.Fatal("EstimateCost(nil request) succeeded")
	}
	if _, err := EstimateCost(&ExecContext{}, &AssembledRequest{}, nil); err == nil || !strings.Contains(err.Error(), "resolved credentials") {
		t.Fatalf("EstimateCost(no credential) error = %v", err)
	}
}

func pricingFixtureContext() *ExecContext {
	return &ExecContext{
		Credential: pricingTestCredential{model: &credentialsv2.CredentialModel{
			AccessKeyId: stringPointer("test-ak"), AccessKeySecret: stringPointer("test-secret"), SecurityToken: stringPointer("test-token"),
		}},
		API: &meta.API{ProductCode: "ecs"}, Region: "cn-hangzhou", Language: "zh", SkipSecureVerify: true,
	}
}

func pricingTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		t.Setenv(key, "")
	}

	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	t.Setenv(pricingEndpointEnv, server.Listener.Addr().String())
	t.Setenv(pricingHostEnv, "")
	t.Setenv(pricingPopCodeEnv, "")
	return server
}

func quoteResponder(t *testing.T, status int, body string) (*ExecContext, *AssembledRequest) {
	t.Helper()
	pricingTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	})
	return pricingFixtureContext(), &AssembledRequest{Action: "EnableBackupPlan", Version: "2017-09-08"}
}

func TestEstimateCostSignsAndPostsQuote(t *testing.T) {
	var calls atomic.Int32
	pricingTestServer(t, func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		if req.Method != "POST" || req.URL.Path != pricingQuotePath || req.Host != "pricing.test" {
			t.Errorf("request = %s %s host=%s", req.Method, req.URL, req.Host)
		}
		for _, header := range []string{"Date", "x-acs-signature-nonce", "x-acs-security-token", "x-acs-accesskey-id"} {
			if req.Header.Get(header) == "" {
				t.Errorf("missing header %s", header)
			}
		}
		if !strings.HasPrefix(req.Header.Get("Authorization"), "acs test-ak:") {
			t.Error("legacy ROA signature missing")
		}
		if req.Header.Get("x-acs-version") != pricingApiVersion || req.Header.Get("x-acs-action") != "GetApiPrice" {
			t.Error("wrong quote identity")
		}
		var body map[string]any
		decoder := json.NewDecoder(req.Body)
		decoder.UseNumber()
		if err := decoder.Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if _, present := body["responseVersion"]; present {
			t.Error("must not select response version")
		}
		if body["language"] != "zh" || body["popCode"] != "ecs" || body["apiName"] != "RunInstances" {
			t.Errorf("payload=%v", body)
		}
		params := body["parameters"].(map[string]any)
		if params["ClusterId"] != "c-123" || params["ID"] != json.Number("9007199254740993") {
			t.Errorf("lost parameters: %v", params)
		}
		if !reflect.DeepEqual(params["Nested"], map[string]any{"Enabled": true}) || !reflect.DeepEqual(params["List"], []any{json.Number("1"), json.Number("2")}) {
			t.Errorf("lost JSON structure: %v", params)
		}
		if !reflect.DeepEqual(params["PricingContext"], map[string]any{"Traffic": "100"}) {
			t.Errorf("lost pricing context: %v", params)
		}
		_, _ = io.WriteString(w, `{"price":{"success":true,"totalAmount":0.084},"requestId":"r"}`)
	})
	t.Setenv(pricingHostEnv, "pricing.test")
	ec := pricingFixtureContext()
	ec.Endpoint = "target.must.not.be.called"
	quote, err := EstimateCost(ec, &AssembledRequest{
		Action: "RunInstances", Version: "2014-05-26", Endpoint: ec.Endpoint,
		PathParams: map[string]string{"ClusterId": "c-123"},
		Body:       `{"ID":9007199254740993,"Nested":{"Enabled":true},"List":[1,2]}`,
	}, map[string]string{"Traffic": "100"})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 || !strings.Contains(quote, "\n") {
		t.Fatalf("calls=%d quote=%s", calls.Load(), quote)
	}
}

func TestBuildPricingParameters(t *testing.T) {
	t.Setenv(pricingPopCodeEnv, "override")
	for _, body := range []any{
		map[string]any{"RegionId": "body-region", "Nested": map[string]any{"x": true}, "List": []any{"a"}, "Empty": ""},
		`{"RegionId":"body-region","Nested":{"x":true},"List":["a"],"Empty":""}`,
		[]byte(`{"RegionId":"body-region","Nested":{"x":true},"List":["a"],"Empty":""}`),
	} {
		params, err := BuildPricingParameters(PricingParameterInput{
			Query: map[string]string{"Query": "q", "Collision": "query", "RegionId": "query-region"},
			Form:  map[string]string{"Form": "f", "Collision": "form"}, Path: map[string]string{"Path": "p", "Collision": "path"}, Body: body, Region: "fallback",
		})
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]any{"Query": "q", "Form": "f", "Path": "p", "Collision": "path", "RegionId": "body-region", "Nested": map[string]any{"x": true}, "List": []any{"a"}, "Empty": ""}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("parameters=%#v", params)
		}
	}
	got, err := buildPriceRequest(pricingFixtureContext(), &AssembledRequest{Action: "Run", Version: "v1", Query: map[string]string{"Amount": "1"}}, map[string]string{"Expr": "a==b"})
	if err != nil {
		t.Fatal(err)
	}
	if got.PopCode != "override" || got.Parameters["RegionId"] != "cn-hangzhou" || got.Parameters["Amount"] != "1" {
		t.Fatalf("request=%+v", got)
	}
	if got.Parameters["PricingContext"].(map[string]any)["Expr"] != "a==b" {
		t.Fatal("context changed")
	}
	for _, body := range []any{`null`, `[1]`, `1`, `{bad`, `{"x":1} {"x":2}`, 42} {
		_, err := BuildPricingParameters(PricingParameterInput{Body: body})
		if err == nil || !strings.Contains(err.Error(), "JSON object") {
			t.Fatalf("body=%v error=%v", body, err)
		}
	}
}

func TestPricingClientRetriesAndBusinessErrors(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		status, retries, calls int
		body                   string
		success                bool
	}{
		{"server retries", 500, 2, 3, `{"Code":"InternalError","Message":"try later","RequestId":"r"}`, false},
		{"retries disabled", 500, 0, 1, `{"Code":"InternalError","Message":"try later","RequestId":"r"}`, false},
		{"forbidden", 403, 2, 1, `{"Code":"Forbidden","Message":"denied","RequestId":"r"}`, false},
		{"throttled", 429, 2, 1, `{"Code":"Throttling","Message":"slow down","RequestId":"r"}`, false},
		{"free", 400, 2, 1, `{"Code":"PricingNotRequired","Message":"server free","RequestId":"r"}`, true},
		{"unsupported", 400, 2, 1, `{"Code":"PricingNotSupported","Message":"server unsupported","RequestId":"r"}`, true},
		{"direct backend", 400, 0, 1, `{"errorCode":"PricingNotSupported","errorMsg":"server unsupported","requestId":"r"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			pricingTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			ec := pricingFixtureContext()
			ec.RetryCount = tc.retries
			raw, err := EstimateCost(ec, &AssembledRequest{Action: "Run", Version: "v1"}, nil)
			if (err == nil) != tc.success || int(calls.Load()) != tc.calls {
				t.Fatalf("calls=%d error=%v raw=%s", calls.Load(), err, raw)
			}
			if tc.success {
				var before, after any
				_ = json.Unmarshal([]byte(tc.body), &before)
				_ = json.Unmarshal([]byte(raw), &after)
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("changed server envelope: %s", raw)
				}
			} else if !strings.Contains(err.Error(), "r") {
				t.Fatalf("missing diagnostics: %v", err)
			}
		})
	}
}

func TestPricingClientTimeoutAndTLS(t *testing.T) {
	for _, host := range []string{"", "pricing.test"} {
		t.Run("host="+host, func(t *testing.T) {
			pricingTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond)
				_, _ = io.WriteString(w, `{}`)
			})
			t.Setenv(pricingHostEnv, host)
			ec := pricingFixtureContext()
			ec.ReadTimeout = 20 * time.Millisecond
			ec.ConnectTimeout = 20 * time.Millisecond
			_, err := EstimateCost(ec, &AssembledRequest{Action: "Run", Version: "v1"}, nil)
			if err == nil {
				t.Fatal("read timeout was ignored")
			}
			ec.ReadTimeout = time.Second
			ec.ConnectTimeout = time.Second
			ec.SkipSecureVerify = false
			_, err = EstimateCost(ec, &AssembledRequest{Action: "Run", Version: "v1"}, nil)
			if err == nil || !strings.Contains(err.Error(), "certificate") {
				t.Fatalf("TLS validation error=%v", err)
			}
		})
	}
}

func TestEstimateCostSurfacesShortRead(t *testing.T) {
	for _, status := range []int{200, 400} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			pricingTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "1000")
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"Code":"PricingNotRequired"}`)
			})
			_, err := EstimateCost(pricingFixtureContext(), &AssembledRequest{Action: "Run", Version: "v1"}, nil)
			if err == nil {
				t.Fatal("short response must fail")
			}
		})
	}
}

func TestPricingClientValidation(t *testing.T) {
	if _, err := NewPricingClient(PricingClientOptions{}); err == nil {
		t.Fatal("nil credential accepted")
	}
	client, err := NewPricingClient(PricingClientOptions{Credential: pricingFixtureContext().Credential})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Quote(PricingRequest{Parameters: map[string]any{"bad": make(chan int)}}); err == nil {
		t.Fatal("unmarshalable payload accepted")
	}
	failure := errors.New("credential failed")
	client, err = NewPricingClient(PricingClientOptions{Credential: pricingTestCredential{err: failure}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Quote(PricingRequest{}); !errors.Is(err, failure) {
		t.Fatalf("credential error=%v", err)
	}
	t.Setenv(pricingEndpointEnv, "")
	if PricingEndpoint() != defaultPricingEndpoint {
		t.Fatal("wrong default endpoint")
	}
	if prettyJSON([]byte("not-json")) != "not-json" {
		t.Fatal("invalid response changed")
	}
}

type rotatingPricingCredential struct {
	pricingTestCredential
	count int
}

func (c *rotatingPricingCredential) GetCredential() (*credentialsv2.CredentialModel, error) {
	c.count++
	return &credentialsv2.CredentialModel{AccessKeyId: tea.String("ak"), AccessKeySecret: tea.String("secret"), SecurityToken: tea.String(fmt.Sprintf("token-%d", c.count))}, nil
}

func TestPricingClientRefreshesCredentials(t *testing.T) {
	var calls atomic.Int32
	pricingTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		want := fmt.Sprintf("token-%d", calls.Add(1))
		if r.Header.Get("x-acs-security-token") != want {
			t.Errorf("stale credential: %s", r.Header.Get("x-acs-security-token"))
		}
		_, _ = io.WriteString(w, `{"price":{"success":true}}`)
	})
	client, err := NewPricingClient(PricingClientOptions{Credential: &rotatingPricingCredential{}, SkipSecureVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := client.Quote(PricingRequest{}); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("missing quote requests")
	}
}
