package openapi

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

func newEndpointSuffixTestContext(t *testing.T) *cli.Context {
	t.Helper()
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	AddFlags(ctx.Flags())
	config.AddFlags(ctx.Flags())
	writeOssTestConfig(t, ctx)
	return ctx
}

func TestLegacyEndpointSuffixBeforeSigning(t *testing.T) {
	for _, source := range []string{"regional", "global", "vpc", "profile", "flag"} {
		t.Run(source, func(t *testing.T) {
			t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
			ctx := newEndpointSuffixTestContext(t)
			profile := &config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou"}
			product := &meta.Product{Code: "Ecs", Version: "2014-05-26"}
			host := source + ".aliyuncs.com"
			switch source {
			case "regional":
				product.RegionalEndpoints = map[string]string{profile.RegionId: host}
			case "global":
				product.GlobalEndpoint = host
			case "vpc":
				profile.EndpointType = "vpc"
				product.RegionalVpcEndpoints = map[string]string{profile.RegionId: host}
			case "profile":
				profile.Endpoint = host
			case "flag":
				config.EndpointFlag(ctx.Flags()).SetValue(host)
				config.EndpointFlag(ctx.Flags()).SetAssigned(true)
			}
			invoker := NewBasicInvoker(profile)
			require.NoError(t, invoker.Init(ctx, product))
			require.Equal(t, source+".aliyuncs.alibaba", invoker.request.Domain)
			calls := 0
			invoker.client.SetTransport(boundaryTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, source+".aliyuncs.alibaba", r.URL.Host)
				require.NotEmpty(t, r.URL.Query().Get("Signature"))
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			}))
			invoker.request.ApiName = "DescribeRegions"
			_, err := invoker.client.Send(invoker.request)
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			if source == "profile" {
				require.Equal(t, host, profile.Endpoint)
			}
		})
	}
}

func TestOpenAPIEndpointSuffixOnWire(t *testing.T) {
	for _, productCode := range []string{"demo", "sls"} {
		t.Run(productCode, func(t *testing.T) {
			t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
			ctx := newEndpointSuffixTestContext(t)
			profile := &config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou"}
			product := &meta.Product{Code: productCode, GlobalEndpoint: "demo.aliyuncs.com"}
			h := NewHttpContext(profile)
			require.NoError(t, h.Init(ctx, product))
			want := "demo.aliyuncs.alibaba"
			if productCode == "sls" {
				want = "cn-hangzhou.log.aliyuncs.alibaba"
			}
			calls := 0
			h.openapiClient.HttpClient = boundaryHTTPClient{RoundTripper: boundaryTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, want, r.URL.Host)
				require.Equal(t, want, r.Host)
				var authorization string
				for name, values := range r.Header {
					if strings.EqualFold(name, "authorization") {
						authorization = strings.Join(values, ",")
					}
				}
				require.NotEmpty(t, authorization)
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			})}
			h.openapiParams.SetAction("GetReport").SetVersion("2020-12-30").SetPathname("/").SetMethod("GET")
			_, err := httpContextExecuteFunc(h)
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			// Explicit request overrides are also rewritten before SDK signing.
			config.EndpointFlag(ctx.Flags()).SetValue("override.aliyuncs.com")
			config.EndpointFlag(ctx.Flags()).SetAssigned(true)
			require.NoError(t, h.Init(ctx, product))
			require.Equal(t, "override.aliyuncs.alibaba", tea.StringValue(h.openapiRequest.EndpointOverride))
		})
	}
}

func TestInvalidEndpointSuffixFailsBeforeCredentials(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "a/b")
	ctx := newEndpointSuffixTestContext(t)
	profile := &config.Profile{Mode: config.External, ProcessCommand: "must-not-run"}
	require.ErrorContains(t, NewBasicInvoker(profile).Init(ctx, &meta.Product{}), endpoint.EnvSuffix)
	require.ErrorContains(t, NewHttpContext(profile).Init(ctx, &meta.Product{}), endpoint.EnvSuffix)
}

func TestPricingEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	t.Setenv(estimateCostEndpointEnv, "")
	require.Equal(t, "cloudcontrol.aliyuncs.alibaba", estimateCostEndpoint())
	t.Setenv(estimateCostEndpointEnv, "custom.aliyuncs.com:8443")
	require.Equal(t, "custom.aliyuncs.alibaba:8443", estimateCostEndpoint())
}
