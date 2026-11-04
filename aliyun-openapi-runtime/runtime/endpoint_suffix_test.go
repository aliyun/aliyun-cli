package runtime

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

type suffixHTTPClient struct{ http.RoundTripper }

func (c suffixHTTPClient) Call(r *http.Request, _ *http.Transport) (*http.Response, error) {
	return c.RoundTrip(r)
}

func TestEndpointSuffixAssemblyAndSigning(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	for _, source := range []string{"regional", "global", "vpc", "explicit"} {
		t.Run(source, func(t *testing.T) {
			host := source + ".aliyuncs.com"
			api := rpcAPI()
			api.Endpoints = meta.Endpoints{}
			ec := &ExecContext{API: api, Region: "cn-hangzhou", Credential: staticAKCredential(t)}
			switch source {
			case "regional":
				api.Endpoints.Public = map[string]string{ec.Region: host}
			case "global":
				api.Endpoints.Global = host
			case "vpc":
				ec.UseVPC = true
				api.Endpoints.VPC = map[string]string{ec.Region: host}
			case "explicit":
				ec.Endpoint = host
			}
			req, err := Assemble(ec)
			if err != nil {
				t.Fatal(err)
			}
			want := source + ".aliyuncs.alibaba"
			if req.Endpoint != want {
				t.Fatalf("endpoint = %q; want %q", req.Endpoint, want)
			}
			call, err := prepareCall(ec, req)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			call.client.HttpClient = suffixHTTPClient{roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host != want || r.Host != want || r.Header.Get("Authorization") == "" {
					t.Fatalf("unexpected signed request: host=%q Host=%q auth=%q", r.URL.Host, r.Host, r.Header.Get("Authorization"))
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			})}
			if _, err := call.client.CallApi(call.params, call.request, call.runtime); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("calls = %d", calls)
			}
			if ec.Endpoint != "" && ec.Endpoint != host {
				t.Fatal("assembly mutated endpoint override")
			}
		})
	}
	t.Setenv(endpoint.EnvSuffix, "a/b")
	if _, err := Assemble(&ExecContext{API: rpcAPI()}); err == nil || !strings.Contains(err.Error(), endpoint.EnvSuffix) {
		t.Fatalf("invalid suffix was not rejected: %v", err)
	}
}

func TestPricingEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	t.Setenv(pricingEndpointEnv, "")
	if got := pricingEndpoint(); got != "cloudcontrol.aliyuncs.alibaba" {
		t.Fatalf("endpoint = %q", got)
	}
	t.Setenv(pricingEndpointEnv, "custom.aliyuncs.com:8443")
	if got := pricingEndpoint(); got != "custom.aliyuncs.alibaba:8443" {
		t.Fatalf("endpoint = %q", got)
	}
}
