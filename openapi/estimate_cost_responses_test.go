package openapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateCostEntryPointsUseQuoteResponseAndPropagateOutputErrors(t *testing.T) {
	for _, route := range []string{"oss bridge", "direct oss", "triple", "openapi", "legacy"} {
		for _, outputError := range []bool{false, true} {
			t.Run(route+map[bool]string{false: "/success", true: "/invalid query"}[outputError], func(t *testing.T) {
				received := make(chan estimateCostRequest, 1)
				servePricingFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, estimateCostQuotePath, r.URL.Path)
					var payload estimateCostRequest
					if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload)) {
						http.Error(w, "invalid quote request", http.StatusBadRequest)
						return
					}
					received <- payload
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{"Success":true,"price":{"amount":42}}`)
				}))
				var output bytes.Buffer
				ctx := cli.NewCommandContext(&output, io.Discard)
				cmd := &cli.Command{Name: "quote", EnableUnknownFlag: true}
				config.AddFlags(cmd.Flags())
				AddFlags(cmd.Flags())
				ctx.EnterCommand(cmd)
				ctx.SetUnknownFlags(cli.NewFlagSet())
				writeOssTestConfig(t, ctx)
				config.SkipSecureVerify(ctx.Flags()).SetAssigned(true)
				EstimateCostContextFlag(ctx.Flags()).SetAssigned(true)
				EstimateCostContextFlag(ctx.Flags()).SetValues([]string{"EstimatedStorageGB=1"})
				if outputError {
					QueryFlag(ctx.Flags()).SetAssigned(true)
					QueryFlag(ctx.Flags()).SetValue("[")
				}
				profile, err := config.LoadProfileWithContext(ctx)
				require.NoError(t, err)
				c := NewCommando(&output, profile)
				repo, err := meta.MockLoadRepository(nil)
				require.NoError(t, err)
				c.library.builtinRepo = repo
				switch route {
				case "legacy":
					request := requests.NewCommonRequest()
					request.Product, request.Version, request.ApiName, request.RegionId = "Oss", ossDefaultPopVersion, "CreateBucket", profile.RegionId
					err = c.processEstimateCost(ctx, &RpcInvoker{BasicInvoker: &BasicInvoker{request: request}})
				case "oss bridge":
					err = c.processOssEstimateCost(ctx, "CreateBucket")
				case "direct oss":
					err = EstimateOssCost(ctx, "CreateBucket", nil, map[string]interface{}{"EstimatedStorageGB": 1})
				case "triple":
					err = c.processEstimateCostByTriple(ctx, &meta.Product{Code: "Oss"}, ossDefaultPopVersion, "CreateBucket")
				case "openapi":
					oc := &OpenapiContext{HttpContext: &HttpContext{profile: &profile, product: &meta.Product{Code: "Oss", Version: ossDefaultPopVersion}}, api: &canonicalmeta.API{Name: "CreateBucket"}}
					err = c.processEstimateCostOpenapi(ctx, oc)
				}
				if outputError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.Contains(t, output.String(), "42")
				}
				select {
				case payload := <-received:
					require.Equal(t, "CreateBucket", payload.ApiName)
					require.Equal(t, "Oss", payload.PopCode)
					require.Equal(t, ossDefaultPopVersion, payload.PopVersion)
					require.Equal(t, profile.RegionId, payload.Parameters["RegionId"])
				default:
					t.Fatal("quote endpoint did not receive the request")
				}
			})
		}
	}
}

func TestEstimateCostRejectsCredentialsAndUnencodableParameters(t *testing.T) {
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	config.AddFlags(ctx.Flags())
	AddFlags(ctx.Flags())
	_, err := invokeEstimateCost(ctx, &config.Profile{Mode: config.Anonymous}, "Oss", "2026-01-01", "CreateBucket", nil)
	require.ErrorContains(t, err, "init estimate-cost client")
	profile := &config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou"}
	_, err = invokeEstimateCost(ctx, profile, "Oss", "2026-01-01", "CreateBucket", map[string]interface{}{"invalid": make(chan int)})
	var unsupported *json.UnsupportedTypeError
	require.ErrorAs(t, err, &unsupported)
	_, err = invokeListSupportedPricingApis(ctx, &config.Profile{Mode: config.Anonymous})
	require.ErrorContains(t, err, "init")
}

func servePricingFixture(t *testing.T, handler http.Handler) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	// The SDK keeps endpoint domains and ports separate; tunnel the test hostname to the local TLS server.
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, http.MethodConnect, r.Method) {
			http.Error(w, "CONNECT required", http.StatusMethodNotAllowed)
			return
		}
		upstream, err := net.Dial("tcp", server.Listener.Addr().String())
		if !assert.NoError(t, err) {
			http.Error(w, "quote server unavailable", http.StatusBadGateway)
			return
		}
		defer upstream.Close()
		conn, buffered, err := w.(http.Hijacker).Hijack()
		if !assert.NoError(t, err) {
			return
		}
		defer conn.Close()
		io.WriteString(conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
		go io.Copy(upstream, buffered)
		io.Copy(conn, upstream)
	}))
	t.Cleanup(proxy.Close)
	t.Setenv(estimateCostEndpointEnv, "quote.test.invalid")
	t.Setenv("HTTPS_PROXY", proxy.URL)
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")
}
