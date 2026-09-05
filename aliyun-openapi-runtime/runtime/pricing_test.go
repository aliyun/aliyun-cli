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
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	credentialsv2 "github.com/aliyun/credentials-go/credentials"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

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

func TestEstimateCostSignsAndPostsQuote(t *testing.T) {
	oldClient := priceHTTPClient
	t.Cleanup(func() { priceHTTPClient = oldClient })

	priceHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != pricingQuotePath {
			t.Errorf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.Host != "pricing.test" {
			t.Errorf("Host = %q", req.Host)
		}
		for _, header := range []string{"Authorization", "Content-MD5", "Date", "x-acs-signature-nonce", "x-acs-security-token", "x-acs-accesskey-id"} {
			if req.Header.Get(header) == "" {
				t.Errorf("missing header %s", header)
			}
		}
		if !strings.HasPrefix(req.Header.Get("Authorization"), "acs test-ak:") {
			t.Errorf("Authorization = %q", req.Header.Get("Authorization"))
		}
		var body priceRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.PopCode != "ecs" || body.PopVersion != "v1" || body.ApiName != "RunInstances" {
			t.Errorf("request body = %#v", body)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"Price":1,"Currency":"CNY"}`)),
			Request:    req,
		}, nil
	})}
	t.Setenv(pricingEndpointEnv, "pricing.example.com")
	t.Setenv(pricingHostEnv, "pricing.test")

	credential := pricingTestCredential{model: &credentialsv2.CredentialModel{
		AccessKeyId:     stringPointer("test-ak"),
		AccessKeySecret: stringPointer("test-secret"),
		SecurityToken:   stringPointer("test-token"),
	}}
	quote, err := EstimateCost(
		&ExecContext{Credential: credential, API: &meta.API{ProductCode: "ecs"}},
		&AssembledRequest{Action: "RunInstances", Version: "v1", Query: map[string]string{"Amount": "1"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(quote, "\n") || !strings.Contains(quote, `"Currency": "CNY"`) {
		t.Fatalf("quote = %q, want pretty JSON", quote)
	}
}

func TestPriceRequestHelpers(t *testing.T) {
	t.Setenv(pricingPopCodeEnv, "")
	ec := &ExecContext{API: &meta.API{ProductCode: "ecs"}, Region: "cn-test"}
	req := &AssembledRequest{
		Action: "Quote", Version: "v1",
		Query: map[string]string{"RegionId": "explicit"},
		Body: map[string]any{
			"String": "value", "Bool": true, "Float": 1.5, "Int": 2,
			"Int64": int64(3), "Number": json.Number("4"), "Ignored": []string{"x"},
		},
	}
	got := buildPriceRequest(ec, req, nil)
	if got.PopCode != "ecs" || got.Parameters["RegionId"] != "explicit" || got.Parameters["Bool"] != "true" || got.Parameters["Ignored"] != nil {
		t.Fatalf("buildPriceRequest() = %#v", got)
	}

	for value, want := range map[any]string{
		"text": "text", true: "true", float64(1.25): "1.25", int(2): "2", int64(3): "3", json.Number("4.5"): "4.5",
	} {
		if actual, ok := priceScalarString(value); !ok || actual != want {
			t.Fatalf("priceScalarString(%#v) = %q, %v", value, actual, ok)
		}
	}
	if _, ok := priceScalarString([]string{"x"}); ok {
		t.Fatal("priceScalarString accepted composite value")
	}

	t.Setenv(pricingEndpointEnv, "custom.example.com")
	if pricingEndpoint() != "custom.example.com" {
		t.Fatalf("pricingEndpoint() = %q", pricingEndpoint())
	}
	t.Setenv(pricingEndpointEnv, "")
	if pricingEndpoint() != defaultPricingEndpoint {
		t.Fatalf("default pricingEndpoint() = %q", pricingEndpoint())
	}
	if prettyJSON([]byte("not-json")) != "not-json" {
		t.Fatal("prettyJSON should preserve invalid JSON")
	}
	if got := md5Base64([]byte("abc")); got != "kAFQmDzST7DWlj99KOF/cg==" {
		t.Fatalf("md5Base64() = %q", got)
	}
	if _, err := base64.StdEncoding.DecodeString(hmacSHA1Base64("key", "data")); err != nil {
		t.Fatalf("hmacSHA1Base64() returned invalid base64: %v", err)
	}
	if got := randomHex(8); len(got) != 16 {
		t.Fatalf("randomHex(8) length = %d", len(got))
	}
}

func TestPostPriceQuoteSignedFailures(t *testing.T) {
	request := &priceRequest{Parameters: map[string]interface{}{}}
	credentialFailure := pricingTestCredential{err: errors.New("credential failed")}
	if _, err := postPriceQuoteSigned(credentialFailure, "example.com", request); err == nil || !strings.Contains(err.Error(), "get credential") {
		t.Fatalf("credential error = %v", err)
	}

	credential := pricingTestCredential{model: &credentialsv2.CredentialModel{
		AccessKeyId: stringPointer("ak"), AccessKeySecret: stringPointer("secret"),
	}}
	request.Parameters["bad"] = make(chan int)
	if _, err := postPriceQuoteSigned(credential, "example.com", request); err == nil {
		t.Fatal("unmarshalable request succeeded")
	}
	delete(request.Parameters, "bad")
	if _, err := postPriceQuoteSigned(credential, "%", request); err == nil {
		t.Fatal("invalid endpoint succeeded")
	}

	oldClient := priceHTTPClient
	t.Cleanup(func() { priceHTTPClient = oldClient })
	priceHTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("transport failed")
	})}
	if _, err := postPriceQuoteSigned(credential, "example.com", request); err == nil || !strings.Contains(err.Error(), "call pricing service") {
		t.Fatalf("transport error = %v", err)
	}
}

func TestBuildROAStringToSignAndPricingErrors(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.com/path", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("x-acs-z", "last")
	req.Header.Set("x-acs-a", "first")
	req.Header.Set("ignored", "value")
	got := buildROAStringToSign(req, "/path", "md5", "application/json", "date")
	if !strings.Contains(got, "x-acs-a:first\nx-acs-z:last") || strings.Contains(got, "ignored") {
		t.Fatalf("string to sign = %q", got)
	}

	pr := &priceRequest{PopCode: "ecs", PopVersion: "v1", ApiName: "Run"}
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"unsupported", `{"Code":"PricingNotSupported","RequestId":"req-1"}`, "no pricing information for ecs/v1/Run"},
		{"detailed", `{"Code":"BadRequest","Message":"bad input","RequestId":"req-2","Recommend":"fix it"}`, "BadRequest — bad input"},
		{"code only", `{"Code":"BadRequest"}`, "BadRequest"},
		{"plain", "service unavailable", "HTTP 503: service unavailable"},
		{"truncated", strings.Repeat("x", 300), "…"},
	}
	for _, test := range tests {
		err := parsePricingHTTPError(pr, 503, []byte(test.raw))
		if !strings.Contains(err.Error(), test.want) {
			t.Fatalf("%s: error = %q, want substring %q", test.name, err, test.want)
		}
	}
}
