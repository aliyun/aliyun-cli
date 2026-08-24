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
package openapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
)

func TestBuildEstimateCostParameters(t *testing.T) {
	req := requests.NewCommonRequest()
	req.QueryParams["InstanceType"] = "ecs.g6.large"
	req.QueryParams["SystemDisk.Category"] = "cloud_essd"
	req.QueryParams["empty"] = ""
	req.FormParams["Description"] = "from-form"
	req.PathParams["ClusterId"] = "c-123"
	req.RegionId = "cn-hangzhou"
	req.SetContent([]byte(`{"Period": 1, "AutoRenew": true}`))

	parameters, err := buildEstimateCostParameters(req)
	assert.Nil(t, err)
	assert.Equal(t, "ecs.g6.large", parameters["InstanceType"])
	assert.Equal(t, "cloud_essd", parameters["SystemDisk.Category"])
	assert.Equal(t, "from-form", parameters["Description"])
	assert.Equal(t, "c-123", parameters["ClusterId"])
	assert.Equal(t, "cn-hangzhou", parameters["RegionId"])
	assert.Equal(t, float64(1), parameters["Period"])
	assert.Equal(t, true, parameters["AutoRenew"])
	_, ok := parameters["empty"]
	assert.False(t, ok, "empty-string param should be dropped, otherwise GetApiPrice sees noise")
}

func TestBuildEstimateCostParametersRegionNotOverridden(t *testing.T) {
	// RegionId in the body / query wins over the request RegionId fallback —
	// otherwise users that explicitly switch region via --RegionId would get
	// quoted against their default profile region by surprise.
	req := requests.NewCommonRequest()
	req.QueryParams["RegionId"] = "cn-beijing"
	req.RegionId = "cn-hangzhou"

	parameters, err := buildEstimateCostParameters(req)
	assert.Nil(t, err)
	assert.Equal(t, "cn-beijing", parameters["RegionId"])
}

func TestBuildEstimateCostParametersBadBody(t *testing.T) {
	// --body that isn't a JSON object can't be merged into pricing parameters —
	// fail with an actionable tip rather than silently dropping the body.
	req := requests.NewCommonRequest()
	req.SetContent([]byte(`not-json`))

	_, err := buildEstimateCostParameters(req)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "JSON object")
}

func TestResolveEstimateCostApiName(t *testing.T) {
	// RPC style: api name already on the request
	rpc := &RpcInvoker{BasicInvoker: &BasicInvoker{request: requests.NewCommonRequest()}}
	rpc.request.ApiName = "RunInstances"
	name, err := resolveEstimateCostApiName(nil, rpc)
	assert.Nil(t, err)
	assert.Equal(t, "RunInstances", name)

	// RESTful style with resolved api metadata
	restful := &RestfulInvoker{
		BasicInvoker: &BasicInvoker{request: requests.NewCommonRequest()},
		api:          &meta.Api{Name: "DescribeClusters"},
	}
	name, err = resolveEstimateCostApiName(nil, restful)
	assert.Nil(t, err)
	assert.Equal(t, "DescribeClusters", name)

	// RESTful style without api metadata and no library match -> clear error
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())
	library := NewLibrary(ctx.Stdout(), "en")
	bare := &RestfulInvoker{
		BasicInvoker: &BasicInvoker{request: requests.NewCommonRequest()},
		method:       "GET",
		path:         "/no/such/path",
	}
	_, err = resolveEstimateCostApiName(library, bare)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot resolve the api name")
}

// quoteServer stands up a TLS stub of the quote endpoint, points
// --estimate-cost at it and relaxes cert verification (self-signed), so the
// full request path — signing, transport, status handling — is exercised
// rather than the body classifier alone. Returns the ready command context.
func quoteServer(t *testing.T, status int, body string) (*cli.Context, *bytes.Buffer, *config.Profile) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv(estimateCostEndpointEnv, srv.Listener.Addr().String())

	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, w)
	cmd := &cli.Command{EnableUnknownFlag: true}
	AddFlags(cmd.Flags())
	config.AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.SetUnknownFlags(cli.NewFlagSet())
	config.SkipSecureVerify(ctx.Flags()).SetAssigned(true)

	profile := config.NewProfile("estimate-cost-test")
	profile.Mode = "AK"
	profile.AccessKeyId = "test-ak"
	profile.AccessKeySecret = "test-secret"
	profile.RegionId = "cn-hangzhou"
	return ctx, w, &profile
}

func TestInvokeEstimateCostCostIrrelevantSucceeds(t *testing.T) {
	// The whole point of the change: a confirmed-free API arrives as a 4xx,
	// and the call must still succeed so the command exits 0.
	ctx, _, profile := quoteServer(t, http.StatusBadRequest,
		`{"RequestId":"req-pnr","Code":"PricingNotRequired","Message":"该 OpenAPI 已确认为费用无关 API，调用不产生费用，无需询价。","HostId":"h"}`)

	out, err := invokeEstimateCost(ctx, profile, "hbr", "2017-09-08", "EnableBackupPlan", map[string]interface{}{})
	assert.Nil(t, err)

	var got costIrrelevantQuote
	assert.Nil(t, json.Unmarshal([]byte(out), &got))
	assert.True(t, got.CostIrrelevant)
	assert.Equal(t, "hbr", got.PopCode)
	assert.Equal(t, "2017-09-08", got.PopVersion)
	assert.Equal(t, "EnableBackupPlan", got.ApiName)
	assert.Equal(t, costIrrelevantMessage, got.Message)

	// And it must not trip the exit-code contract on the way out.
	assert.Nil(t, estimateCostBusinessError(out))
}

func TestInvokeEstimateCostNotSupportedStillFails(t *testing.T) {
	// The sibling code must keep failing — "cannot be quoted" is a real error.
	ctx, _, profile := quoteServer(t, http.StatusNotFound,
		`{"RequestId":"req-pns","Code":"PricingNotSupported","Message":"no pricing","HostId":"h"}`)

	out, err := invokeEstimateCost(ctx, profile, "Ecs", "2014-05-26", "DescribeInstances", map[string]interface{}{})
	assert.Equal(t, "", out)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no pricing information for Ecs/2014-05-26/DescribeInstances")
}

func TestInvokeEstimateCostSuccessBodyPassesThrough(t *testing.T) {
	// A normal 2xx quote is untouched by the cost-irrelevant handling.
	ctx, _, profile := quoteServer(t, http.StatusOK, `{"price":{"originalAmount":0.178},"requestId":"req-ok"}`)

	out, err := invokeEstimateCost(ctx, profile, "Ecs", "2014-05-26", "RunInstances", map[string]interface{}{})
	assert.Nil(t, err)
	assert.Contains(t, out, "originalAmount")
	assert.NotContains(t, out, "costIrrelevant")
}

func TestProcessEstimateCostByTripleCostIrrelevantPrintsAndSucceeds(t *testing.T) {
	// End-to-end through the by-triple entry point: the document is printed
	// and no error is returned, which is what makes the command exit 0.
	ctx, w, profile := quoteServer(t, http.StatusBadRequest,
		`{"RequestId":"req-pnr","Code":"PricingNotRequired","Message":"...","HostId":"h"}`)
	command := NewCommando(w, *profile)

	product := &meta.Product{Code: "tablestore", Version: "2020-12-09"}
	err := command.processEstimateCostByTriple(ctx, product, "2020-12-09", "CreateAgentStorage")
	assert.Nil(t, err)
	assert.Contains(t, w.String(), `"costIrrelevant": true`)
	assert.Contains(t, w.String(), costIrrelevantMessage)
}

func TestTranslateEstimateCostErrorPassthrough(t *testing.T) {
	// Non-server errors fall through unchanged so callers see the original
	// network/transport error instead of a misleading "pricing rejected" tip.
	plain := assert.AnError
	assert.Equal(t, plain, translateEstimateCostError("Ecs", "2014-05-26", "RunInstances", plain))
}

func TestTranslateEstimateCostErrorPricingNotSupported(t *testing.T) {
	// "The quote cannot be produced" case — must be turned into the friendly
	// hint, not a raw error string with the upstream Code embedded.
	body := `{"RequestId":"req-pns","Code":"PricingNotSupported","Message":"no pricing","HostId":"host"}`
	serverErr := sdkerrors.NewServerError(404, body, "")
	err := translateEstimateCostError("Ecs", "2014-05-26", "DescribeRegions", serverErr)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no pricing information for Ecs/2014-05-26/DescribeRegions")
	tip, _ := err.(cli.ErrorWithTip)
	assert.NotNil(t, tip)
	assert.Contains(t, tip.GetTip(""), "cannot be quoted")
	// The tip must not let a reader conclude the call is free — that is the
	// exact conflation PricingNotRequired exists to remove, and reading it the
	// wrong way is a billing-relevant mistake.
	assert.Contains(t, tip.GetTip(""), "does not mean the call is free")
	assert.NotContains(t, tip.GetTip(""), "either incurs no cost")
}

func TestTranslateEstimateCostErrorLeavesPricingNotRequiredAlone(t *testing.T) {
	// PricingNotRequired is intercepted in invokeEstimateCost and never
	// reaches the error translator; if it ever did, it must not be dressed up
	// as one of the failure cases.
	body := `{"RequestId":"req-pnr","Code":"PricingNotRequired","Message":"free","HostId":"host"}`
	serverErr := sdkerrors.NewServerError(400, body, "")
	err := translateEstimateCostError("hbr", "2017-09-08", "EnableBackupPlan", serverErr)
	assert.Equal(t, serverErr, err)
}

// TestCostIrrelevantQuoteMarshals guards the assumption that lets
// newCostIrrelevantQuote ignore the json.Marshal error: the document is a bool
// and four strings, which cannot fail to marshal. If someone later adds a field
// that can (a channel, a func, a failing MarshalJSON), this test breaks loudly
// at build time — better than a runtime branch that could silently report a
// free API as unquotable, which is a billing-relevant wrong answer.
func TestCostIrrelevantQuoteMarshals(t *testing.T) {
	out := newCostIrrelevantQuote("Ecs", "2014-05-26", "RunInstances")
	assert.JSONEq(t, `{
		"costIrrelevant": true,
		"popCode": "Ecs",
		"popVersion": "2014-05-26",
		"apiName": "RunInstances",
		"message": "`+costIrrelevantMessage+`"
	}`, out)
}

func TestNewCostIrrelevantQuoteCarriesTripleAndEnglishMessage(t *testing.T) {
	out := newCostIrrelevantQuote("hbr", "2017-09-08", "EnableBackupPlan")

	var got costIrrelevantQuote
	assert.Nil(t, json.Unmarshal([]byte(out), &got))
	assert.True(t, got.CostIrrelevant)
	assert.Equal(t, "hbr", got.PopCode)
	assert.Equal(t, "2017-09-08", got.PopVersion)
	assert.Equal(t, "EnableBackupPlan", got.ApiName)
	assert.Equal(t, costIrrelevantMessage, got.Message)
	// CLI output is read internationally and by scripts, so the wording is
	// the CLI's own English text, never the server's localized message.
	assert.NotContains(t, got.Message, "无需询价")
}

func TestCostIrrelevantFromErrorRecognisesPricingNotRequired(t *testing.T) {
	body := `{"RequestId":"req-pnr","Code":"PricingNotRequired","Message":"该 OpenAPI 已确认为费用无关 API，调用不产生费用，无需询价。","HostId":"host"}`
	serverErr := sdkerrors.NewServerError(400, body, "")

	out, isCostIrrelevant := costIrrelevantFromError("hbr", "2017-09-08", "EnableBackupPlan", serverErr)
	assert.True(t, isCostIrrelevant)

	var got costIrrelevantQuote
	assert.Nil(t, json.Unmarshal([]byte(out), &got))
	assert.True(t, got.CostIrrelevant)
	assert.Equal(t, "EnableBackupPlan", got.ApiName)
	// Server sent a Chinese message; the CLI reports its own English wording.
	assert.Equal(t, costIrrelevantMessage, got.Message)
}

func TestCostIrrelevantFromErrorPassesOtherErrorsThrough(t *testing.T) {
	// Anything that is not PricingNotRequired must stay on the error path —
	// especially PricingNotSupported, which means the opposite thing.
	notSupported := sdkerrors.NewServerError(404,
		`{"RequestId":"r","Code":"PricingNotSupported","Message":"no pricing","HostId":"h"}`, "")
	_, isCostIrrelevant := costIrrelevantFromError("Ecs", "2014-05-26", "DescribeRegions", notSupported)
	assert.False(t, isCostIrrelevant)

	_, isCostIrrelevant = costIrrelevantFromError("Ecs", "2014-05-26", "RunInstances", assert.AnError)
	assert.False(t, isCostIrrelevant)
}

func TestCostIrrelevantQuoteIsNotABusinessFailure(t *testing.T) {
	// The exit-code contract: a confirmed-free answer must NOT fail the
	// process. Scripts and agents gate on `$?`, and aborting on a free
	// operation is precisely the behaviour this change removes.
	out := newCostIrrelevantQuote("hbr", "2017-09-08", "EnableBackupPlan")
	assert.Nil(t, estimateCostBusinessError(out))
}

func TestTranslateEstimateCostErrorInvalidParameter(t *testing.T) {
	// Parameter-side rejection (bad popCode/version, missing required field)
	// gets a "check parameters" tip; raw error keeps its detail (the upstream
	// SDK.ServerError formatted body) so users can see what the server
	// objected to. Tip is delivered via cli.ErrorWithTip.GetTip, not the
	// Error() string itself — Error() preserves the wrapped error verbatim.
	body := `{"RequestId":"req-ip","Code":"InvalidParameter","Message":"bad popVersion","HostId":"host"}`
	serverErr := sdkerrors.NewServerError(400, body, "")
	err := translateEstimateCostError("Ecs", "2014-05-26", "RunInstances", serverErr)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "InvalidParameter")
	tip, _ := err.(cli.ErrorWithTip)
	assert.NotNil(t, tip)
	assert.Contains(t, tip.GetTip(""), "check them against the target API")
}

func TestTranslateEstimateCostErrorUnknownServerCode(t *testing.T) {
	// Server errors with codes we don't special-case (Throttling, Forbidden,
	// random new ones) fall through unchanged — better the user sees the
	// upstream code+message than a vague "pricing failed" wrapper.
	body := `{"RequestId":"req-th","Code":"Throttling.User","Message":"slow down","HostId":"host"}`
	serverErr := sdkerrors.NewServerError(429, body, "")
	got := translateEstimateCostError("Ecs", "2014-05-26", "RunInstances", serverErr)
	assert.Equal(t, serverErr, got)
}

func TestPrintEstimateCostResultQuietSkipsOutput(t *testing.T) {
	// `--quiet` short-circuits before any rendering — otherwise --estimate-cost
	// combined with -q (e.g. agent piping to /dev/null) would still print the
	// quote, surprising callers.
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	QuietFlag(ctx.Flags()).SetAssigned(true)
	defer QuietFlag(ctx.Flags()).SetAssigned(false)

	err := printEstimateCostResult(ctx, `{"price":{"calculatedAmount":42}}`)
	assert.Nil(t, err)
	assert.Empty(t, w.String(), "no output should be written under --quiet")
}

func TestPrintEstimateCostResultPlainJSON(t *testing.T) {
	// Default path (no quiet/query/output) just sorts and prints the JSON.
	// Anchor it so a future refactor of the output pipeline can't silently
	// drop the response.
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	err := printEstimateCostResult(ctx, `{"price":{"calculatedAmount":42},"requestId":"req-1"}`)
	assert.Nil(t, err)
	assert.Contains(t, w.String(), "calculatedAmount")
	assert.Contains(t, w.String(), "42")
	assert.Contains(t, w.String(), "req-1")
}

func TestPrintEstimateCostResultWithCliQuery(t *testing.T) {
	// --cli-query JMESPath filter must apply on top of the estimate JSON. If
	// this branch silently failed, agents piping `.price.calculatedAmount`
	// through --cli-query would see the full envelope instead of just the
	// number they asked for.
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	QueryFlag(ctx.Flags()).SetAssigned(true)
	QueryFlag(ctx.Flags()).SetValue("price.calculatedAmount")
	defer func() {
		QueryFlag(ctx.Flags()).SetAssigned(false)
	}()

	err := printEstimateCostResult(ctx, `{"price":{"calculatedAmount":42},"requestId":"req-1"}`)
	assert.Nil(t, err)
	// Output should be just the number (or string form of it), not the full envelope.
	assert.Contains(t, w.String(), "42")
	assert.NotContains(t, w.String(), "req-1")
}

func TestProcessInvokeEstimateCostFlag(t *testing.T) {
	// Endpoint pointed at an unresolvable host: the flow must reach the
	// estimate-cost client call (proving interception) and fail on DNS, not
	// invoke the target API. If the EstimateCostFlag check were missing or
	// wrong, the call would go to ecs.cn-hangzhou.aliyuncs.com instead.
	t.Setenv(estimateCostEndpointEnv, "estimate-cost.test.invalid")

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	profile := config.NewProfile("test-estimate-cost")
	profile.Mode = "AK"
	profile.AccessKeyId = "test-ak"
	profile.AccessKeySecret = "test-secret"
	profile.RegionId = "cn-hangzhou"
	command := NewCommando(w, profile)

	EstimateCostFlag(ctx.Flags()).SetAssigned(true)
	defer EstimateCostFlag(ctx.Flags()).SetAssigned(false)

	err := command.processInvoke(ctx, "ecs", "DescribeRegions", "")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "estimate-cost.test.invalid")
}

func TestListSupportedPricingApisEndpointOverride(t *testing.T) {
	// Endpoint env override must take effect; otherwise users can't target
	// pre/staging gateways from the main repo side (same expectation as the
	// plugin runtime — the two share the env name on purpose).
	t.Setenv(estimateCostEndpointEnv, "pricing.test.example")
	assert.Equal(t, "pricing.test.example", estimateCostEndpoint())

	t.Setenv(estimateCostEndpointEnv, "")
	assert.Equal(t, defaultEstimateCostHost, estimateCostEndpoint())
}

func TestMainEstimateCostMissingProductOrApi(t *testing.T) {
	// `aliyun --estimate-cost` (no product/api) and `aliyun ecs --estimate-cost`
	// (product only) must fail loud with an actionable example, otherwise the
	// flag would be silently dropped on the printUsage / plugin-help branch
	// and users (especially Agents) would see "nothing happened" and assume
	// the capability is broken or unknown.
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	profile := config.NewProfile("test-estimate-cost")
	command := NewCommando(w, profile)

	EstimateCostFlag(ctx.Flags()).SetAssigned(true)
	defer EstimateCostFlag(ctx.Flags()).SetAssigned(false)

	// no args at all
	err := command.main(ctx, []string{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "--estimate-cost requires a product and an API name")

	// product only (forgot API name)
	err = command.main(ctx, []string{"ecs"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "--estimate-cost requires a product and an API name")
}

// helper: build a context with flags and assign --estimate-cost-context values.
func ctxWithPricingContext(values []string) *cli.Context {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	if values != nil {
		f := EstimateCostContextFlag(ctx.Flags())
		f.SetAssigned(true)
		f.SetValues(values)
	}
	return ctx
}

func TestBuildPricingContextMultiValue(t *testing.T) {
	ctx := ctxWithPricingContext([]string{"EstimatedInternetTrafficOutGB=100", "InternetChargeType=PayByTraffic"})
	pc, err := buildPricingContext(ctx)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{
		"EstimatedInternetTrafficOutGB": "100",
		"InternetChargeType":            "PayByTraffic",
	}, pc)
}

func TestBuildPricingContextFirstEqualsSplit(t *testing.T) {
	// value may contain '=' — split on the FIRST '=' only.
	ctx := ctxWithPricingContext([]string{"Expr=a==b"})
	pc, err := buildPricingContext(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "a==b", pc["Expr"])
}

func TestBuildPricingContextEmptyValueAllowed(t *testing.T) {
	ctx := ctxWithPricingContext([]string{"EipAllocationId="})
	pc, err := buildPricingContext(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "", pc["EipAllocationId"])
}

func TestBuildPricingContextMalformedNoEquals(t *testing.T) {
	ctx := ctxWithPricingContext([]string{"novalue"})
	_, err := buildPricingContext(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "novalue")
}

func TestBuildPricingContextEmptyKeyRejected(t *testing.T) {
	ctx := ctxWithPricingContext([]string{"=100"})
	_, err := buildPricingContext(ctx)
	assert.NotNil(t, err)
}

func TestBuildPricingContextUnassigned(t *testing.T) {
	ctx := ctxWithPricingContext(nil)
	pc, err := buildPricingContext(ctx)
	assert.Nil(t, err)
	assert.Nil(t, pc)
}

func TestProcessInvokeEstimateCostWithContext(t *testing.T) {
	// --estimate-cost + --estimate-cost-context: exercises the PricingContext
	// nesting in processEstimateCost, then fails on an unresolvable host
	// (proving the quote request — with the nested context — was assembled and
	// sent, not the target API).
	t.Setenv(estimateCostEndpointEnv, "estimate-cost.test.invalid")

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	profile := config.NewProfile("test-estimate-cost")
	profile.Mode = "AK"
	profile.AccessKeyId = "test-ak"
	profile.AccessKeySecret = "test-secret"
	profile.RegionId = "cn-hangzhou"
	command := NewCommando(w, profile)

	EstimateCostFlag(ctx.Flags()).SetAssigned(true)
	cf := EstimateCostContextFlag(ctx.Flags())
	cf.SetAssigned(true)
	cf.SetValues([]string{"EstimatedInternetTrafficOutGB=100", "InternetChargeType=PayByTraffic"})

	err := command.processInvoke(ctx, "ecs", "DescribeRegions", "")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "estimate-cost.test.invalid")
}

func TestProcessInvokeEstimateCostContextMalformed(t *testing.T) {
	// A malformed --estimate-cost-context entry must abort before any network
	// call (buildPricingContext error propagates out of processEstimateCost).
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	profile := config.NewProfile("test-estimate-cost")
	profile.Mode = "AK"
	profile.AccessKeyId = "test-ak"
	profile.AccessKeySecret = "test-secret"
	profile.RegionId = "cn-hangzhou"
	command := NewCommando(w, profile)

	EstimateCostFlag(ctx.Flags()).SetAssigned(true)
	cf := EstimateCostContextFlag(ctx.Flags())
	cf.SetAssigned(true)
	cf.SetValues([]string{"novalue"})

	err := command.processInvoke(ctx, "ecs", "DescribeRegions", "")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "novalue")
}

func TestEstimateCostBusinessErrorFailure(t *testing.T) {
	// price.success == false must fail the process even though the HTTP call
	// succeeded (previously business failures exited 0 and scripts
	// gating on $? mistook them for successful estimates).
	out := `{"price":{"success":false,"errorCode":"InvalidParameter","errorMessage":"COMMODITY.INVALID_COMPONENT","upstreamRequestId":"req-1"},"requestId":"r"}`
	err := estimateCostBusinessError(out)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "InvalidParameter")
	assert.Contains(t, err.Error(), "COMMODITY.INVALID_COMPONENT")
	assert.Contains(t, err.Error(), "req-1")
}

func TestEstimateCostBusinessErrorSuccessAndTolerantShapes(t *testing.T) {
	// success:true, missing price node, bare-DTO shape and non-JSON output must
	// all keep exit code 0 — only explicit business failure is an error.
	assert.Nil(t, estimateCostBusinessError(`{"price":{"success":true},"requestId":"r"}`))
	assert.Nil(t, estimateCostBusinessError(`{"requestId":"r"}`))
	assert.Nil(t, estimateCostBusinessError(`not json`))
	// bare DTO without the gateway "price" envelope
	assert.NotNil(t, estimateCostBusinessError(`{"success":false,"errorCode":"E"}`))
	assert.Nil(t, estimateCostBusinessError(`{"success":true}`))
}

func TestEstimateCostBusinessErrorWithoutDetails(t *testing.T) {
	err := estimateCostBusinessError(`{"price":{"success":false}}`)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cost estimation failed")
}

func newOpenapiEstimateCostContext(api *meta.Api) *OpenapiContext {
	return &OpenapiContext{
		HttpContext: &HttpContext{
			profile: &config.Profile{RegionId: "cn-hangzhou"},
			product: &meta.Product{Code: "Sls", Version: "2020-12-30"},
			openapiRequest: &openapiutil.OpenApiRequest{
				Query:   map[string]*string{},
				Headers: map[string]*string{},
				HostMap: map[string]*string{},
			},
		},
		method: "POST",
		path:   "/logstores",
		api:    api,
	}
}

func TestBuildEstimateCostParametersFromOpenapi(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, w)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	api := &meta.Api{
		Name:    "CreateLogStore",
		Product: &meta.Product{Version: "2020-12-30"},
		Parameters: []meta.Parameter{
			{Name: "project", Position: "Host"},
			{Name: "logstoreName", Position: "Path"},
		},
	}

	// Path/Host params are recovered from typed flags via api metadata;
	// query comes from the prepared request; JSON body merges on top.
	unknown := cli.NewFlagSet()
	f, _ := unknown.AddByName("project")
	f.SetAssigned(true)
	f.SetValue("my-project")
	f2, _ := unknown.AddByName("logstoreName")
	f2.SetAssigned(true)
	f2.SetValue("my-store")
	ctx.SetUnknownFlags(unknown)

	oc := newOpenapiEstimateCostContext(api)
	shardCount := "2"
	oc.openapiRequest.Query["shardCount"] = &shardCount
	oc.openapiRequest.SetBody([]byte(`{"ttl":30,"hot_ttl":7}`))

	parameters, err := buildEstimateCostParametersFromOpenapi(ctx, oc)
	assert.NoError(t, err)
	assert.Equal(t, "2", parameters["shardCount"])
	assert.Equal(t, "my-project", parameters["project"])
	assert.Equal(t, "my-store", parameters["logstoreName"])
	assert.Equal(t, float64(30), parameters["ttl"])
	assert.Equal(t, float64(7), parameters["hot_ttl"])
	// RegionId defaulted from the profile so mapping expressions can rely on it.
	assert.Equal(t, "cn-hangzhou", parameters["RegionId"])
}

func TestBuildEstimateCostParametersFromOpenapiBodyVariants(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, w)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.SetUnknownFlags(cli.NewFlagSet())

	api := &meta.Api{Name: "CreateLogStore", Product: &meta.Product{Version: "2020-12-30"}}

	// Per-flag body params arrive as an already-parsed map.
	oc := newOpenapiEstimateCostContext(api)
	oc.openapiRequest.Body = map[string]interface{}{"ttl": 30}
	parameters, err := buildEstimateCostParametersFromOpenapi(ctx, oc)
	assert.NoError(t, err)
	assert.Equal(t, 30, parameters["ttl"])

	// Raw --body string form.
	oc = newOpenapiEstimateCostContext(api)
	oc.openapiRequest.Body = `{"ttl":15}`
	parameters, err = buildEstimateCostParametersFromOpenapi(ctx, oc)
	assert.NoError(t, err)
	assert.Equal(t, float64(15), parameters["ttl"])

	// Non-object body must be rejected, mirroring the RPC/ROA path contract.
	oc = newOpenapiEstimateCostContext(api)
	oc.openapiRequest.SetBody([]byte(`[1,2,3]`))
	_, err = buildEstimateCostParametersFromOpenapi(ctx, oc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JSON object")

	// Binary (non-JSON-decodable) bodies are rejected too.
	oc = newOpenapiEstimateCostContext(api)
	oc.openapiRequest.SetBody([]byte{0x28, 0xb5, 0x2f, 0xfd})
	_, err = buildEstimateCostParametersFromOpenapi(ctx, oc)
	assert.Error(t, err)
}

func TestProcessApiInvokeEstimateCost(t *testing.T) {
	// Same contract as TestProcessInvokeEstimateCostFlag but for the openapi
	// invoke path (SLS): the flow must reach the estimate-cost client (DNS
	// failure on the sentinel host proves interception) and the target API
	// hook must never fire.
	t.Setenv(estimateCostEndpointEnv, "estimate-cost.test.invalid")

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.SetUnknownFlags(cli.NewFlagSet())

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(w, profile)

	EstimateCostFlag(ctx.Flags()).SetAssigned(true)
	defer EstimateCostFlag(ctx.Flags()).SetAssigned(false)

	originCallHook := hookHttpContextCall
	defer func() { hookHttpContextCall = originCallHook }()
	targetInvoked := false
	hookHttpContextCall = func(fn func() error) func() error {
		return func() error {
			targetInvoked = true
			return nil
		}
	}

	product := &meta.Product{Code: "sls"}
	api := &meta.Api{
		Name:    "CreateLogStore",
		Product: &meta.Product{Version: "2020-12-30"},
	}
	err := command.processApiInvoke(ctx, product, api, "POST", "/logstores")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "estimate-cost.test.invalid")
	assert.False(t, targetInvoked, "--estimate-cost must not invoke the target API")
}

func TestProcessEstimateCostOpenapiNoApiName(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, w)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	command := NewCommando(w, config.Profile{Mode: "AK", AccessKeyId: "ak", AccessKeySecret: "sk", RegionId: "cn-hangzhou"})
	oc := newOpenapiEstimateCostContext(&meta.Api{})
	err := command.processEstimateCostOpenapi(ctx, oc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot resolve the api name")
}

func TestBuildEstimateCostParametersFromFlags(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, w)
	cmd := &cli.Command{EnableUnknownFlag: true}
	AddFlags(cmd.Flags())
	config.AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	unknown := cli.NewFlagSet()
	f1, _ := unknown.AddByName("WorkspaceId")
	f1.SetAssigned(true)
	f1.SetValue("ws-123")
	ctx.SetUnknownFlags(unknown)

	BodyFlag(ctx.Flags()).SetAssigned(true)
	BodyFlag(ctx.Flags()).SetValue(`{"TrainingSpec":{"Instances":1},"Priority":2}`)

	profile := config.NewProfile("p")
	profile.RegionId = "cn-hangzhou"

	parameters, err := buildEstimateCostParametersFromFlags(ctx, &profile)
	assert.NoError(t, err)
	assert.Equal(t, "ws-123", parameters["WorkspaceId"])
	assert.Equal(t, float64(2), parameters["Priority"])
	assert.NotNil(t, parameters["TrainingSpec"])
	assert.Equal(t, "cn-hangzhou", parameters["RegionId"])

	// --region beats the profile region as RegionId fallback.
	rf := config.RegionFlag(ctx.Flags())
	rf.SetAssigned(true)
	rf.SetValue("cn-beijing")
	parameters, err = buildEstimateCostParametersFromFlags(ctx, &profile)
	assert.NoError(t, err)
	assert.Equal(t, "cn-beijing", parameters["RegionId"])

	// --RegionId is a registered root flag (never an unknown flag) and must
	// win over --region and the profile region.
	rif := config.RegionIdFlag(ctx.Flags())
	rif.SetAssigned(true)
	rif.SetValue("cn-shenzhen")
	parameters, err = buildEstimateCostParametersFromFlags(ctx, &profile)
	assert.NoError(t, err)
	assert.Equal(t, "cn-shenzhen", parameters["RegionId"])

	// Non-object --body is rejected with the shared JSON-object contract.
	BodyFlag(ctx.Flags()).SetValue(`[1,2]`)
	_, err = buildEstimateCostParametersFromFlags(ctx, &profile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JSON object")
}

func TestProcessEstimateCostByTriple(t *testing.T) {
	// Sentinel-endpoint contract: reaching the quote client (DNS failure on
	// the sentinel host) proves the metadata-less path routes to pricing and
	// never tries to resolve/invoke the target api.
	t.Setenv(estimateCostEndpointEnv, "estimate-cost.test.invalid")

	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, w)
	cmd := &cli.Command{EnableUnknownFlag: true}
	AddFlags(cmd.Flags())
	config.AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.SetUnknownFlags(cli.NewFlagSet())

	profile := config.NewProfile("test-triple")
	profile.Mode = "AK"
	profile.AccessKeyId = "test-ak"
	profile.AccessKeySecret = "test-secret"
	profile.RegionId = "cn-hangzhou"
	command := NewCommando(w, profile)

	product := &meta.Product{Code: "pai-dlc", Version: "2020-12-03"}
	err := command.processEstimateCostByTriple(ctx, product, "2022-01-12", "CreateTrainingJob")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "estimate-cost.test.invalid")

	// Non-PascalCase api names are rejected locally with a usage tip.
	err = command.processEstimateCostByTriple(ctx, product, "2022-01-12", "not-an-api")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires an OpenAPI name")
}

func TestMainEstimateCostUnknownApiRoutesToTriple(t *testing.T) {
	// End-to-end through Commando.main: a restful product whose local
	// metadata lacks the api (Tablestore has no api definitions at all)
	// must fall through to the by-triple quote, not InvalidApiError.
	t.Setenv(estimateCostEndpointEnv, "estimate-cost.test.invalid")

	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, w)
	cmd := &cli.Command{EnableUnknownFlag: true}
	AddFlags(cmd.Flags())
	config.AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.SetUnknownFlags(cli.NewFlagSet())

	profile := config.NewProfile("test-triple-main")
	profile.Mode = "AK"
	profile.AccessKeyId = "test-ak"
	profile.AccessKeySecret = "test-secret"
	profile.RegionId = "cn-hangzhou"
	command := NewCommando(w, profile)

	// main() discards the injected profile and reloads one via
	// LoadProfileWithContext, which reads the HOST config file by default —
	// absent on CI runners ("region can't be empty" before the code under
	// test runs). Point --config-path at a self-contained temp config so the
	// test is hermetic on any machine.
	configPath := filepath.Join(t.TempDir(), "config.json")
	assert.NoError(t, os.WriteFile(configPath, []byte(`{"current":"test-triple-main","profiles":[{"name":"test-triple-main","mode":"AK","access_key_id":"test-ak","access_key_secret":"test-secret","region_id":"cn-hangzhou"}]}`), 0o600))
	configPathFlag := config.ConfigurePathFlag(ctx.Flags())
	configPathFlag.SetAssigned(true)
	configPathFlag.SetValue(configPath)

	EstimateCostFlag(ctx.Flags()).SetAssigned(true)
	defer EstimateCostFlag(ctx.Flags()).SetAssigned(false)

	err := command.main(ctx, []string{"tablestore", "CreateVCUInstance"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "estimate-cost.test.invalid")
	assert.NotContains(t, err.Error(), "not a valid api")
}

func TestBuildEstimateCostParametersFromFlagsBodyAndRegionFallbacks(t *testing.T) {
	newCtx := func() *cli.Context {
		w := new(bytes.Buffer)
		ctx := cli.NewCommandContext(w, w)
		cmd := &cli.Command{EnableUnknownFlag: true}
		AddFlags(cmd.Flags())
		config.AddFlags(cmd.Flags())
		ctx.EnterCommand(cmd)
		ctx.SetUnknownFlags(cli.NewFlagSet())
		return ctx
	}
	p := config.NewProfile("p")
	p.RegionId = "cn-shanghai"
	profile := &p

	// --body JSON object merges on top of unknown-flag parameters.
	ctx := newCtx()
	bodyFlag := BodyFlag(ctx.Flags())
	bodyFlag.SetAssigned(true)
	bodyFlag.SetValue(`{"Period":2}`)
	params, err := buildEstimateCostParametersFromFlags(ctx, profile)
	assert.Nil(t, err)
	assert.Equal(t, float64(2), params["Period"])
	assert.Equal(t, "cn-shanghai", params["RegionId"], "profile region is the last fallback")

	// --body that is not a JSON object fails with the actionable error.
	ctx = newCtx()
	bodyFlag = BodyFlag(ctx.Flags())
	bodyFlag.SetAssigned(true)
	bodyFlag.SetValue(`[1,2]`)
	_, err = buildEstimateCostParametersFromFlags(ctx, profile)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "JSON object")

	// --body-file: valid file merges, missing file errors with the path.
	ctx = newCtx()
	bodyPath := filepath.Join(t.TempDir(), "body.json")
	assert.NoError(t, os.WriteFile(bodyPath, []byte(`{"AutoRenew":true}`), 0o600))
	fileFlag := BodyFileFlag(ctx.Flags())
	fileFlag.SetAssigned(true)
	fileFlag.SetValue(bodyPath)
	params, err = buildEstimateCostParametersFromFlags(ctx, profile)
	assert.Nil(t, err)
	assert.Equal(t, true, params["AutoRenew"])

	ctx = newCtx()
	fileFlag = BodyFileFlag(ctx.Flags())
	fileFlag.SetAssigned(true)
	fileFlag.SetValue(filepath.Join(t.TempDir(), "no-such.json"))
	_, err = buildEstimateCostParametersFromFlags(ctx, profile)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "--body-file")

	// --RegionId (registered root flag) wins over --region and the profile.
	ctx = newCtx()
	regionIdFlag := config.RegionIdFlag(ctx.Flags())
	regionIdFlag.SetAssigned(true)
	regionIdFlag.SetValue("cn-beijing")
	params, err = buildEstimateCostParametersFromFlags(ctx, profile)
	assert.Nil(t, err)
	assert.Equal(t, "cn-beijing", params["RegionId"])

	ctx = newCtx()
	regionFlag := config.RegionFlag(ctx.Flags())
	regionFlag.SetAssigned(true)
	regionFlag.SetValue("cn-shenzhen")
	params, err = buildEstimateCostParametersFromFlags(ctx, profile)
	assert.Nil(t, err)
	assert.Equal(t, "cn-shenzhen", params["RegionId"])
}

func TestMergeEstimateCostBodyForms(t *testing.T) {
	// Already-parsed map form.
	params := map[string]interface{}{}
	assert.Nil(t, mergeEstimateCostBody(params, map[string]interface{}{"K": "v"}))
	assert.Equal(t, "v", params["K"])

	// Raw []byte and string forms decode as JSON objects.
	assert.Nil(t, mergeEstimateCostBody(params, []byte(`{"A":1}`)))
	assert.Equal(t, float64(1), params["A"])
	assert.Nil(t, mergeEstimateCostBody(params, `{"B":2}`))
	assert.Equal(t, float64(2), params["B"])

	// Anything else is rejected with the actionable tip.
	err := mergeEstimateCostBody(params, 42)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "cannot read the request body")
}

func TestEstimateCostBusinessErrorShapes(t *testing.T) {
	// Bare DTO shape (success at top level, no "price" envelope) must still
	// fail the process — tolerance for a future envelope change.
	err := estimateCostBusinessError(`{"success":false,"errorCode":"X","errorMessage":"m","upstreamRequestId":"r"}`)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "X")
	assert.Contains(t, err.Error(), "upstreamRequestId: r")

	// Unparseable / non-JSON output is not the exit-code contract's business:
	// transport and server errors surface earlier, so this returns nil.
	assert.Nil(t, estimateCostBusinessError("not-json"))
	// Missing success field means success.
	assert.Nil(t, estimateCostBusinessError(`{"price":{}}`))
}

func TestFilterSupportedPricingApisEdgeShapes(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, w)
	cmd := NewListSupportedPricingApisCommand()
	ctx.EnterCommand(cmd)
	productFlag := ctx.Flags().Get(PricingProductFlagName)
	productFlag.SetAssigned(true)
	productFlag.SetValue("Ecs")

	// Non-string popCode is skipped via the stringField guard, not a panic.
	out, err := filterSupportedPricingApis(ctx, `{"supportedApis":[{"popCode":123},{"popCode":"Ecs","popVersion":"2014-05-26"}],"requestId":"r"}`)
	assert.Nil(t, err)
	assert.Contains(t, out, "2014-05-26")
	assert.NotContains(t, out, "123")

	// Unexpected response shape fails with a clear error instead of emitting
	// a silently unfiltered list.
	_, err = filterSupportedPricingApis(ctx, "not-json")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unexpected response shape")
}
