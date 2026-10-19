package openapi

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	spi "github.com/alibabacloud-go/alibabacloud-gateway-spi/client"
	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	openapiTeaUtils "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/require"
)

type failingGateway struct {
	spi.ClientInterface
	cause error
	calls int
}

func (g *failingGateway) ModifyConfiguration(context *spi.InterceptorContext, _ *spi.AttributeMap) error {
	g.calls++
	if tea.StringValue(context.Request.Action) != "GetReport" {
		return errors.New("unexpected gateway action")
	}
	return g.cause
}

type boundaryHTTPClient struct{ http.RoundTripper }

func (c boundaryHTTPClient) Call(request *http.Request, _ *http.Transport) (*http.Response, error) {
	return c.RoundTrip(request)
}

func TestHttpContextUsesSDKTransportAndGatewayErrorRouting(t *testing.T) {
	calls := 0
	client, err := openapiClient.NewClient(&openapiClient.Config{Endpoint: tea.String("fixture.invalid"), Protocol: tea.String("http"), AccessKeyId: tea.String("id"), AccessKeySecret: tea.String("secret")})
	require.NoError(t, err)
	client.HttpClient = boundaryHTTPClient{RoundTripper: boundaryTransport(func(request *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "fixture.invalid", request.URL.Host)
		require.Equal(t, "/reports", request.URL.Path)
		require.Equal(t, http.MethodGet, request.Method)
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ReportId":"r-1"}`))}, nil
	})}
	h := &HttpContext{openapiClient: client, openapiRequest: &openapiutil.OpenApiRequest{Headers: map[string]*string{}}, openapiRuntime: &openapiTeaUtils.RuntimeOptions{}, openapiParams: &openapiClient.Params{
		Action: tea.String("GetReport"), Version: tea.String("v1"), Protocol: tea.String("http"), Pathname: tea.String("/reports"), Method: tea.String("GET"), AuthType: tea.String("Anonymous"), BodyType: tea.String("json"), ReqBodyType: tea.String("json"), Style: tea.String("ROA"),
	}}
	response, err := httpContextExecuteFunc(h)
	require.NoError(t, err)
	require.Equal(t, "r-1", response["body"].(map[string]any)["ReportId"])
	require.Equal(t, 1, calls)
	cause := errors.New("gateway configuration unavailable")
	gateway := &failingGateway{cause: cause}
	client.Spi = gateway
	_, err = httpContextExecuteFunc(h)
	require.ErrorIs(t, err, cause)
	require.Equal(t, 1, gateway.calls)
	require.Equal(t, 1, calls)
}

func TestLegacyClientRejectsUnsupportedCredentialsAndInvalidRegions(t *testing.T) {
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	writeOssTestConfig(t, ctx)
	for _, profile := range []config.Profile{
		{Mode: config.BearerToken},
		{Mode: config.StsToken, AccessKeyId: "id", AccessKeySecret: "secret", StsToken: "token"},
		{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "invalid region"},
	} {
		_, err := GetClient(&profile, ctx)
		require.Error(t, err)
	}
	profile := &config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "invalid region"}
	invoker := NewBasicInvoker(profile)
	require.ErrorContains(t, invoker.Init(ctx, &meta.Product{Code: "demo", Version: "v1", GlobalEndpoint: "fixture.invalid"}), "init client failed")
	profile.RegionId = "cn-hangzhou"
	UserAgentFlag(ctx.Flags()).SetAssigned(true)
	UserAgentFlag(ctx.Flags()).SetValue("custom/1.0")
	client, err := GetOpenapiClient(profile, ctx, &meta.Product{Code: "demo", GlobalEndpoint: "fixture.invalid"})
	require.NoError(t, err)
	require.Contains(t, tea.StringValue(client.UserAgent), "custom/1.0")
	profile.Mode, profile.ProcessCommand = config.External, filepath.Join(t.TempDir(), "missing-credential-helper")
	_, err = GetClient(profile, ctx)
	require.Error(t, err)
}
