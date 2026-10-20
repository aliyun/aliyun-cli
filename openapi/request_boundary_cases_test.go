package openapi

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/require"
)

func TestLegacyRequestPreparationPreservesBodyAndParameterLocations(t *testing.T) {
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	ctx.SetUnknownFlags(cli.NewFlagSet())
	path := filepath.Join(t.TempDir(), "body.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"name":"report"}`), 0600))
	BodyFileFlag(ctx.Flags()).SetAssigned(true)
	BodyFileFlag(ctx.Flags()).SetValue(path)
	InsecureFlag(ctx.Flags()).SetAssigned(true)
	InsecureFlag(ctx.Flags()).SetValue("true")
	api := &canonicalmeta.API{Name: "CreateReport", Protocol: "HTTPS", Method: "POST", Parameters: []canonicalmeta.Parameter{
		{Name: "Query", RawName: "Query", Location: "query", Type: "string"},
		{Name: "Form", RawName: "Form", Location: "body", Type: "string"},
		{Name: "Host", RawName: "Host", Location: "host", Type: "string"},
	}}
	for _, name := range []string{"Query", "Form", "Host"} {
		f, _ := ctx.UnknownFlags().AddByName(name)
		f.SetAssigned(true)
		f.SetValue(name + "-value")
	}
	inv := &RestfulInvoker{BasicInvoker: &BasicInvoker{request: requests.NewCommonRequest()}, api: api, method: "POST", path: "/reports"}
	require.NoError(t, inv.Prepare(ctx))
	require.Equal(t, "Query-value", inv.request.QueryParams["Query"])
	require.Equal(t, "Form-value", inv.request.FormParams["Form"])
	require.NotContains(t, inv.request.QueryParams, "Host")
	require.JSONEq(t, `{"name":"report"}`, string(inv.request.Content))
	require.Equal(t, "http", inv.request.Scheme)
	api.Parameters[2].Location = "cookie"
	require.ErrorContains(t, inv.Prepare(ctx), "unknown parameter position")

	api.Parameters = []canonicalmeta.Parameter{{Name: "body", RawName: "body", Location: "body", Type: "string"}, {Name: "Action", RawName: "Action", Location: "query", Type: "string", Required: true}}
	ctx.SetUnknownFlags(cli.NewFlagSet())
	BodyFlag(ctx.Flags()).SetAssigned(true)
	BodyFlag(ctx.Flags()).SetValue(`{"enabled":true}`)
	rpc := &RpcInvoker{BasicInvoker: &BasicInvoker{request: requests.NewCommonRequest()}, api: api}
	require.NoError(t, rpc.Prepare(ctx))
	require.Equal(t, "http", rpc.request.Scheme)
	require.JSONEq(t, `{"enabled":true}`, string(rpc.request.Content))
	MethodFlag(ctx.Flags()).SetAssigned(true)
	MethodFlag(ctx.Flags()).SetValue("DELETE")
	require.ErrorContains(t, rpc.Prepare(ctx), "not supported")
	force := &ForceRpcInvoker{BasicInvoker: &BasicInvoker{request: requests.NewCommonRequest()}, method: "CreateReport"}
	require.ErrorContains(t, force.Prepare(ctx), "not supported")
	MethodFlag(ctx.Flags()).SetAssigned(true)
	MethodFlag(ctx.Flags()).SetValue("GET")
	require.NoError(t, force.Prepare(ctx))
	require.Equal(t, "http", force.request.Scheme)
	fileFlag := &cli.Flag{Name: "Payload-FILE"}
	fileFlag.SetAssigned(true)
	fileFlag.SetValue(path)
	replaceValueWithFile(fileFlag)
	value, _ := fileFlag.GetValue()
	require.JSONEq(t, `{"name":"report"}`, value)
}

func TestQuoteCollectionRejectsMalformedBodiesBeforeCallingService(t *testing.T) {
	for _, route := range []string{"legacy", "triple", "openapi", "oss"} {
		t.Run(route, func(t *testing.T) {
			c, ctx, _, _ := newCanonicalHelpTestContext(t)
			config.AddFlags(ctx.Flags())
			writeOssTestConfig(t, ctx)
			profile, err := config.LoadProfileWithContext(ctx)
			require.NoError(t, err)
			c.profile = profile
			config.SkipSecureVerify(ctx.Flags()).SetAssigned(true)
			calls := make(chan struct{}, 1)
			servePricingFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls <- struct{}{}
				http.Error(w, "quote transport must not be reached", http.StatusBadRequest)
			}))
			ctx.SetUnknownFlags(cli.NewFlagSet())
			BodyFlag(ctx.Flags()).SetAssigned(true)
			BodyFlag(ctx.Flags()).SetValue("not-json")
			request := requests.NewCommonRequest()
			request.ApiName = "CreateReport"
			request.SetContent([]byte("not-json"))
			want := "requires the request body to be a JSON object"
			switch route {
			case "legacy":
				err = c.processEstimateCost(ctx, &RpcInvoker{BasicInvoker: &BasicInvoker{request: request}})
			case "triple":
				err = c.processEstimateCostByTriple(ctx, &meta.Product{Code: "demo"}, "2026-01-01", "CreateReport")
			case "openapi":
				want = "cannot read the request body of type chan int"
				err = c.processEstimateCostOpenapi(ctx, &OpenapiContext{HttpContext: &HttpContext{profile: &c.profile, openapiRequest: &openapiutil.OpenApiRequest{Body: make(chan int)}}, api: &canonicalmeta.API{Name: "CreateReport"}})
			case "oss":
				want = "invalid --estimate-cost-context `invalid`"
				EstimateCostContextFlag(ctx.Flags()).SetValues([]string{"invalid"})
				EstimateCostContextFlag(ctx.Flags()).SetAssigned(true)
				err = c.processOssEstimateCost(ctx, "CreateBucket")
			}
			require.ErrorContains(t, err, want)
			require.Empty(t, calls, "invalid pricing parameters must fail before transport")
		})
	}
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	ctx.SetUnknownFlags(cli.NewFlagSet())
	unassigned, _ := ctx.UnknownFlags().AddByName("Unused")
	unassigned.SetValue("ignored")
	multi, _ := ctx.UnknownFlags().AddByName("Tag")
	multi.SetAssigned(true)
	multi.SetValues([]string{"one", "two"})
	parameters, err := buildEstimateCostParametersFromFlags(ctx, &config.Profile{})
	require.NoError(t, err)
	require.Equal(t, []string{"one", "two"}, parameters["Tag"])
	require.NotContains(t, parameters, "Unused")
	badBody := filepath.Join(t.TempDir(), "invalid.json")
	require.NoError(t, os.WriteFile(badBody, []byte("[]"), 0600))
	BodyFileFlag(ctx.Flags()).SetAssigned(true)
	BodyFileFlag(ctx.Flags()).SetValue(badBody)
	_, err = buildEstimateCostParametersFromFlags(ctx, &config.Profile{})
	require.ErrorContains(t, err, "JSON object")
}

func TestResponseLocalizationRejectsInvalidSchemaAndPreservesNonText(t *testing.T) {
	broken := json.RawMessage("{")
	components := &machineHelpComponents{Schemas: map[string]json.RawMessage{"Broken": broken}}
	for _, doc := range []machineHelpAPIResponseDocument{
		{Responses: broken}, {Components: components},
		{OutputSchema: &machineHelpOutputSchema{Schema: broken}},
		{OutputSchema: &machineHelpOutputSchema{Schema: json.RawMessage(`{}`), Components: components}},
	} {
		_, err := json.Marshal(doc)
		require.Error(t, err)
	}
	for _, raw := range []string{`{"description_en":12}`, `{"description":{},"description_zh":"text"}`} {
		value, err := localizeMachineHelpRawJSON(json.RawMessage(raw))
		require.NoError(t, err)
		require.JSONEq(t, raw, string(value))
	}
	node := map[string]any{"description_en": "English", "description_zh": "中文"}
	projectMachineHelpLocalizedJSONText(node, "description", "zh")
	require.Equal(t, map[string]any{"description": "中文"}, node)
	require.Equal(t, broken, localizeHelpJSON(broken, "en"))
	require.Nil(t, localizeHelpComponents(nil, "en"))
	require.JSONEq(t, `{"example":"","pattern":""}`, string(localizeHelpJSON(json.RawMessage(`{"description":"","example":"","pattern":""}`), "en")))
	require.Error(t, encodeMachineHelpJSON(io.Discard, make(chan int), false))
}

func TestDryRunEndpointAndWildcardSanitization(t *testing.T) {
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	require.False(t, clientDryRunAssigned(nil))
	require.Equal(t, "redacted", sanitizedDryRunPathname("/wildcard", "/secret", map[string]string{"Path": "/secret"}, map[string]string{"Path": "redacted"}))
	require.Equal(t, "/plain", sanitizedDryRunPathname("/pattern", "/plain", nil, nil))
	h := &OpenapiContext{HttpContext: &HttpContext{profile: &config.Profile{Endpoint: "configured.test"}}}
	require.Equal(t, "configured.test", dryRunOpenapiEndpoint(ctx, h))
	h.profile.Endpoint = ""
	h.product = &meta.Product{Code: "demo", GlobalEndpoint: "global.test"}
	require.Equal(t, "global.test", dryRunOpenapiEndpoint(ctx, h))
	h.product = nil
	require.Empty(t, dryRunOpenapiEndpoint(ctx, h))
	r := &RestfulInvoker{BasicInvoker: &BasicInvoker{request: requests.NewCommonRequest()}, api: &canonicalmeta.API{Name: "ListReports"}}
	require.Equal(t, "ListReports", buildDryRunInvokeMeta(nil, r).API)
}
