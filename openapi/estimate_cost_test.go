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
	"testing"

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

func TestTranslateEstimateCostErrorPassthrough(t *testing.T) {
	// Non-server errors fall through unchanged so callers see the original
	// network/transport error instead of a misleading "pricing rejected" tip.
	plain := assert.AnError
	assert.Equal(t, plain, translateEstimateCostError("Ecs", "2014-05-26", "RunInstances", plain))
}

func TestTranslateEstimateCostErrorPricingNotSupported(t *testing.T) {
	// The common "this API isn't billable" case — must be turned into the
	// friendly hint, not a raw error string with the upstream Code embedded.
	body := `{"RequestId":"req-pns","Code":"PricingNotSupported","Message":"no pricing","HostId":"host"}`
	serverErr := sdkerrors.NewServerError(404, body, "")
	err := translateEstimateCostError("Ecs", "2014-05-26", "DescribeRegions", serverErr)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no pricing information for Ecs/2014-05-26/DescribeRegions")
	tip, _ := err.(cli.ErrorWithTip)
	assert.NotNil(t, tip)
	assert.Contains(t, tip.GetTip(""), "incurs no cost or has no pricing mapping registered yet")
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
