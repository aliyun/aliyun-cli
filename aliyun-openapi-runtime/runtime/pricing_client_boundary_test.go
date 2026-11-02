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
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPricingClientListPreservesPaginationTokens(t *testing.T) {
	const token = "next+a/b=?&"
	const body = `{"supportedApis":[{"popCode":"ecs","apiName":"RunInstances"}],"nextToken":"next+a/b=?&"}`
	var calls atomic.Int32
	pricingTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		page := calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != pricingListPath || r.Header.Get("x-acs-action") != "ListSupportedPricingApis" {
			t.Errorf("unexpected list request: %s %s", r.Method, r.URL)
		}
		if page == 1 && r.URL.RawQuery != "" || page == 2 && r.URL.Query().Get("nextToken") != token {
			t.Errorf("page %d query = %q", page, r.URL.RawQuery)
		}
		if r.Header.Get("x-acs-version") != pricingApiVersion || !strings.HasPrefix(r.Header.Get("Authorization"), "acs test-ak:") {
			t.Error("list request lost the signed pricing-service identity")
		}
		_, _ = io.WriteString(w, body)
	})
	client, err := NewPricingClient(PricingClientOptions{Credential: pricingFixtureContext().Credential, SkipSecureVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for _, next := range []string{"", token} {
		got, err := client.ListSupportedAPIs(next)
		if err != nil || got != body {
			t.Fatalf("list response = %q, error = %v", got, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("list requests = %d, want 2", calls.Load())
	}
}

func TestPricingClientRejectsNoContentResponses(t *testing.T) {
	pricingTestServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	client, err := NewPricingClient(PricingClientOptions{Credential: pricingFixtureContext().Credential, SkipSecureVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for _, operation := range []string{"quote", "list"} {
		t.Run(operation, func(t *testing.T) {
			var body string
			var err error
			if operation == "quote" {
				body, err = client.Quote(PricingRequest{})
			} else {
				body, err = client.ListSupportedAPIs("")
			}
			if err == nil || !strings.Contains(err.Error(), "no response body") || body != "" {
				t.Fatalf("empty HTTP 204 must fail: body=%q error=%v", body, err)
			}
		})
	}
}

type pricingTestHTTPClient struct {
	call func(*http.Request, *http.Transport) (*http.Response, error)
}

func (c pricingTestHTTPClient) Call(r *http.Request, tr *http.Transport) (*http.Response, error) {
	return c.call(r, tr)
}

func TestPricingClientRetriesOnlyRecoverableTransportErrors(t *testing.T) {
	transient := &url.Error{Op: "Post", URL: "https://pricing.invalid", Err: io.ErrUnexpectedEOF}
	for _, tc := range []struct {
		name  string
		err   error
		retry bool
	}{
		{"transient connection", transient, true},
		{"unknown authority", &url.Error{Err: x509.UnknownAuthorityError{}}, false},
		{"expired certificate", &url.Error{Err: x509.CertificateInvalidError{Cert: &x509.Certificate{}, Reason: x509.Expired}}, false},
		{"hostname mismatch", &url.Error{Err: x509.HostnameError{Certificate: &x509.Certificate{}, Host: "pricing.invalid"}}, false},
		{"non-transport failure", errors.New("request preparation failed"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewPricingClient(PricingClientOptions{Credential: pricingFixtureContext().Credential, RetryCount: 2})
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			calls := 0
			client.client.HttpClient = pricingTestHTTPClient{call: func(r *http.Request, tr *http.Transport) (*http.Response, error) {
				calls++
				if calls == 1 {
					return nil, tc.err
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"price":{"success":true}}`))}, nil
			}}
			body, err := client.Quote(PricingRequest{})
			if tc.retry {
				if err != nil || calls != 2 || body != `{"price":{"success":true}}` {
					t.Fatalf("retry did not recover: calls=%d body=%q err=%v", calls, body, err)
				}
			} else if !errors.Is(err, tc.err) || calls != 1 || body != "" {
				t.Fatalf("permanent failure must be preserved without retry: calls=%d body=%q err=%v", calls, body, err)
			}
		})
	}
}

func TestEstimateCostRejectsMalformedBodyBeforeSending(t *testing.T) {
	var calls atomic.Int32
	pricingTestServer(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
	for _, body := range []any{`null`, `[1]`, `{"x":1} {"x":2}`, make(chan int)} {
		out, err := EstimateCost(pricingFixtureContext(), &AssembledRequest{Action: "RunInstances", Version: "2014-05-26", Body: body}, nil)
		if err == nil || !strings.Contains(err.Error(), "JSON object") || out != "" {
			t.Fatalf("malformed body must fail before quote: output=%q error=%v", out, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("malformed bodies sent %d requests", calls.Load())
	}
}

func TestPricingEmptyBodyRetainsPreparedParameters(t *testing.T) {
	for _, body := range []any{nil, []byte(nil), []byte{}, ""} {
		got, err := BuildPricingParameters(PricingParameterInput{Query: map[string]string{"RegionId": "cn-test", "": "ignored", "Empty": ""}, Body: body, Region: "fallback"})
		if err != nil || len(got) != 1 || got["RegionId"] != "cn-test" {
			t.Fatalf("empty body changed prepared parameters: %#v, error=%v", got, err)
		}
	}
}
