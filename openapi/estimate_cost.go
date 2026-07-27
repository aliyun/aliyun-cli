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
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

// Routes an OpenAPI call to CloudControl GetApiPrice (POST /api/v1/price/quote)
// for an estimate instead of invoking the target API.
//
// Env overrides (match the plugin runtime's naming):
//
//	ALIBABA_CLOUD_PRICING_ENDPOINT — default cloudcontrol.aliyuncs.com
//	ALIBABA_CLOUD_PRICING_HOST     — Host header when endpoint is a CNAME
const (
	estimateCostApiVersion  = "2022-08-30"
	estimateCostQuotePath   = "/api/v1/price/quote"
	estimateCostEndpointEnv = "ALIBABA_CLOUD_PRICING_ENDPOINT"
	estimateCostHostEnv     = "ALIBABA_CLOUD_PRICING_HOST"
	defaultEstimateCostHost = "cloudcontrol.aliyuncs.com"
	estimateCostProductCode = "cloudcontrol"
)

type estimateCostRequest struct {
	PopCode    string                 `json:"popCode"`
	PopVersion string                 `json:"popVersion"`
	ApiName    string                 `json:"apiName"`
	Parameters map[string]interface{} `json:"parameters"`
}

// processEstimateCost handles --estimate-cost for both RPC and ROA(restful)
// invokers. Must be called after invoker.Prepare(ctx) so the CommonRequest
// carries the fully assembled parameters; otherwise required params from
// `--body` JSON and path templating wouldn't be visible to the quote.
func (c *Commando) processEstimateCost(ctx *cli.Context, inv Invoker) error {
	req := inv.getRequest()

	apiName, err := resolveEstimateCostApiName(c.library, inv)
	if err != nil {
		return err
	}

	parameters, err := buildEstimateCostParameters(req)
	if err != nil {
		return err
	}

	// PricingContext (--estimate-cost-context): pricing-only assumptions/state
	// overrides. Nested inside `parameters` as a sibling of the API params —
	// the quote service reads the whole parameters object as `request` and
	// mapping expressions reference request.PricingContext.<key>.
	pricingContext, err := buildPricingContext(ctx)
	if err != nil {
		return err
	}
	if len(pricingContext) > 0 {
		parameters["PricingContext"] = pricingContext
	}

	out, err := invokeEstimateCost(ctx, &c.profile, req.Product, req.Version, apiName, parameters)
	if err != nil {
		return err
	}
	if err := printEstimateCostResult(ctx, out); err != nil {
		return err
	}
	// Business failure inside a 200 response (price.success == false) must fail
	// the process: scripts and agents gate on `$?`, and an exit code of 0 with a
	// failed quote inside the JSON reads as a successful estimate (observed in
	// practice with several products' business failures). The JSON
	// is already printed above, so automation can still read the full details.
	return estimateCostBusinessError(out)
}

// estimateCostBusinessError inspects the raw quote response and returns a
// non-nil error when the quote itself reports failure. A missing/unparseable
// success field is treated as success — the exit-code contract only covers
// explicit business failures, and transport/server errors are surfaced earlier
// by invokeEstimateCost.
func estimateCostBusinessError(out string) error {
	type quote struct {
		Success      *bool  `json:"success"`
		ErrorCode    string `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
		RequestId    string `json:"upstreamRequestId"`
	}
	var body struct {
		// Gateway shape: {"price": {...}, "requestId": "..."}; tolerate the bare
		// DTO shape (success at top level) in case the envelope changes.
		Price *quote `json:"price"`
		quote
	}
	if err := json.Unmarshal([]byte(out), &body); err != nil {
		return nil
	}
	if body.Price == nil {
		body.Price = &body.quote
	}
	if body.Price.Success == nil || *body.Price.Success {
		return nil
	}
	msg := "cost estimation failed"
	if body.Price.ErrorCode != "" {
		msg += ": " + body.Price.ErrorCode
	}
	if body.Price.ErrorMessage != "" {
		msg += ": " + body.Price.ErrorMessage
	}
	if body.Price.RequestId != "" {
		msg += " (upstreamRequestId: " + body.Price.RequestId + ")"
	}
	return cli.NewErrorWithTip(fmt.Errorf("%s", msg),
		"the quote response above has the full details; fix the reported parameter or retry later, the target API was NOT invoked")
}

// PascalCase OpenAPI action name, e.g. CreateTrainingJob. Product-command
// tokens and file-operation subcommands are lowercase, so this cannot match
// them by accident.
var estimateCostApiNameRegexp = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

// processEstimateCostByTriple quotes an api the local metadata cannot
// resolve — an api of a non-default product version (pai-dlc 2022-01-12,
// Selectdb 2022-10-19), or a product with no api definitions at all
// (Tablestore). A real invocation needs metadata to derive method/path and
// a signer for the product's protocol, but a quote needs neither: only the
// triple (popCode/popVersion/apiName) + parameters sent to CloudControl.
// The trade-off is no local api-name validation — a typo comes back from
// the server as PricingNotSupported instead of a local "invalid api" error.
func (c *Commando) processEstimateCostByTriple(ctx *cli.Context, product *meta.Product, popVersion string, apiName string) error {
	if !estimateCostApiNameRegexp.MatchString(apiName) {
		return cli.NewErrorWithTip(
			fmt.Errorf("--estimate-cost requires an OpenAPI name, got `%s`", apiName),
			"use `aliyun <product> <ApiName> ... --estimate-cost`, e.g. `aliyun pai-dlc CreateTrainingJob --version 2022-01-12 ... --estimate-cost`")
	}
	parameters, err := buildEstimateCostParametersFromFlags(ctx, &c.profile)
	if err != nil {
		return err
	}
	pricingContext, err := buildPricingContext(ctx)
	if err != nil {
		return err
	}
	if len(pricingContext) > 0 {
		parameters["PricingContext"] = pricingContext
	}
	out, err := invokeEstimateCost(ctx, &c.profile, product.Code, popVersion, apiName, parameters)
	if err != nil {
		return err
	}
	if err := printEstimateCostResult(ctx, out); err != nil {
		return err
	}
	return estimateCostBusinessError(out)
}

// buildEstimateCostParametersFromFlags collects pricing parameters without
// an invoker: unknown flags are the api parameters (there is no metadata to
// type them against), the `--body` JSON object merges on top, and RegionId
// falls back to --region and then the profile region.
func buildEstimateCostParametersFromFlags(ctx *cli.Context, profile *config.Profile) (map[string]interface{}, error) {
	parameters := make(map[string]interface{})
	if ctx.UnknownFlags() != nil {
		for _, f := range ctx.UnknownFlags().Flags() {
			if !f.IsAssigned() {
				continue
			}
			if values := f.GetValues(); len(values) > 1 {
				parameters[f.Name] = values
				continue
			}
			if v, ok := f.GetValue(); ok && v != "" {
				parameters[f.Name] = v
			}
		}
	}
	if v, ok := BodyFlag(ctx.Flags()).GetValue(); ok && v != "" {
		if err := mergeEstimateCostJSONBody(parameters, []byte(v)); err != nil {
			return nil, err
		}
	}
	if v, ok := BodyFileFlag(ctx.Flags()).GetValue(); ok && v != "" {
		buf, err := os.ReadFile(v)
		if err != nil {
			return nil, fmt.Errorf("--estimate-cost failed to read --body-file %s: %v", v, err)
		}
		if err := mergeEstimateCostJSONBody(parameters, buf); err != nil {
			return nil, err
		}
	}
	if _, ok := parameters["RegionId"]; !ok {
		// --RegionId is a REGISTERED root flag (config.NewRegionIdFlag), so it
		// never lands in the unknown flags — read it explicitly, then fall
		// back to --region and the profile region.
		if regionId, ok := config.RegionIdFlag(ctx.Flags()).GetValue(); ok && regionId != "" {
			parameters["RegionId"] = regionId
		} else if region, ok := config.RegionFlag(ctx.Flags()).GetValue(); ok && region != "" {
			parameters["RegionId"] = region
		} else if profile.RegionId != "" {
			parameters["RegionId"] = profile.RegionId
		}
	}
	return parameters, nil
}

// processEstimateCostOpenapi handles --estimate-cost for the openapi invoke
// path (currently SLS only, see ShouldUseOpenapi; the path is meant to take
// over more products later). Must be called after apiContext.Prepare(ctx) so
// query/body/path parameters are fully assembled; the target API is never
// invoked.
func (c *Commando) processEstimateCostOpenapi(ctx *cli.Context, oc *OpenapiContext) error {
	if oc.api == nil || oc.api.Name == "" {
		return cli.NewErrorWithTip(
			fmt.Errorf("--estimate-cost cannot resolve the api name for this call"),
			"cost estimation needs the api name, please use the `aliyun <product> <ApiName>` form")
	}
	popCode := ""
	popVersion := ""
	if oc.product != nil {
		popCode = oc.product.Code
		popVersion = oc.product.Version
	}
	if oc.api.Product != nil && oc.api.Product.Version != "" {
		popVersion = oc.api.Product.Version
	}

	parameters, err := buildEstimateCostParametersFromOpenapi(ctx, oc)
	if err != nil {
		return err
	}
	pricingContext, err := buildPricingContext(ctx)
	if err != nil {
		return err
	}
	if len(pricingContext) > 0 {
		parameters["PricingContext"] = pricingContext
	}

	out, err := invokeEstimateCost(ctx, &c.profile, popCode, popVersion, oc.api.Name, parameters)
	if err != nil {
		return err
	}
	if err := printEstimateCostResult(ctx, out); err != nil {
		return err
	}
	return estimateCostBusinessError(out)
}

// buildEstimateCostParametersFromOpenapi flattens the prepared openapi-path
// request into one parameter map: assembled query params, path/host parameter
// raw values (Prepare substitutes them into the pathname, so they are
// recovered from the typed flags via api metadata), and the JSON body.
func buildEstimateCostParametersFromOpenapi(ctx *cli.Context, oc *OpenapiContext) (map[string]interface{}, error) {
	parameters := make(map[string]interface{})
	if oc.openapiRequest != nil {
		for k, v := range oc.openapiRequest.Query {
			if k != "" && v != nil && *v != "" {
				parameters[k] = *v
			}
		}
	}
	if oc.api != nil && ctx.UnknownFlags() != nil {
		for _, f := range ctx.UnknownFlags().Flags() {
			param := oc.api.FindParameter(f.Name)
			if param == nil || (param.Position != "Path" && param.Position != "Host") {
				continue
			}
			if value, ok := f.GetValue(); ok && value != "" {
				parameters[f.Name] = value
			}
		}
	}
	if oc.openapiRequest != nil && oc.openapiRequest.Body != nil {
		if err := mergeEstimateCostBody(parameters, oc.openapiRequest.Body); err != nil {
			return nil, err
		}
	}
	if regionId := oc.profile.RegionId; regionId != "" {
		if _, ok := parameters["RegionId"]; !ok {
			parameters["RegionId"] = regionId
		}
	}
	return parameters, nil
}

// mergeEstimateCostBody merges the openapi-path request body into the pricing
// parameter map. The body is either the already-parsed per-flag map or the
// raw `--body` bytes/string, which must decode to a JSON object — same
// contract as the RPC/ROA path in buildEstimateCostParameters.
func mergeEstimateCostBody(parameters map[string]interface{}, rawBody interface{}) error {
	switch body := rawBody.(type) {
	case map[string]interface{}:
		for k, v := range body {
			parameters[k] = v
		}
		return nil
	case []byte:
		return mergeEstimateCostJSONBody(parameters, body)
	case string:
		return mergeEstimateCostJSONBody(parameters, []byte(body))
	default:
		return cli.NewErrorWithTip(
			fmt.Errorf("--estimate-cost cannot read the request body of type %T", rawBody),
			"cost estimation merges the JSON body into pricing parameters, please pass `--body` as a JSON object")
	}
}

func mergeEstimateCostJSONBody(parameters map[string]interface{}, raw []byte) error {
	body := make(map[string]interface{})
	if err := json.Unmarshal(raw, &body); err != nil {
		return cli.NewErrorWithTip(
			fmt.Errorf("--estimate-cost requires the request body to be a JSON object: %v", err),
			"cost estimation merges the JSON body into pricing parameters, please pass `--body` as a JSON object")
	}
	for k, v := range body {
		parameters[k] = v
	}
	return nil
}

// resolveEstimateCostApiName returns the action name of the call being
// estimated. Pricing is keyed by the api triple, so a bare RESTful call
// (method + path, no action name) must be resolvable through metadata.
func resolveEstimateCostApiName(library *Library, inv Invoker) (string, error) {
	req := inv.getRequest()
	if req.ApiName != "" {
		return req.ApiName, nil
	}
	if r, ok := inv.(*RestfulInvoker); ok {
		if r.api != nil && r.api.Name != "" {
			return r.api.Name, nil
		}
		if api, found := meta.HookGetApiByPath(library.GetApiByPath)(req.Product, req.Version, r.method, r.path); found && api.Name != "" {
			return api.Name, nil
		}
		return "", cli.NewErrorWithTip(
			fmt.Errorf("--estimate-cost cannot resolve the api name for `%s %s`", r.method, r.path),
			"cost estimation needs the api name, please use the `aliyun <product> <ApiName>` form")
	}
	return "", fmt.Errorf("--estimate-cost cannot resolve the api name for this call")
}

// buildEstimateCostParameters flattens every parameter slot of the prepared
// request (query / form-or-body / path / JSON body) into one map. Values stay
// strings — CLI semantics; the server normalizes dotted keys like `DataDisk.1.Size`.
func buildEstimateCostParameters(req *requests.CommonRequest) (map[string]interface{}, error) {
	parameters := make(map[string]interface{})
	for k, v := range req.QueryParams {
		if k != "" && v != "" {
			parameters[k] = v
		}
	}
	for k, v := range req.FormParams {
		if k != "" && v != "" {
			parameters[k] = v
		}
	}
	for k, v := range req.PathParams {
		if k != "" && v != "" {
			parameters[k] = v
		}
	}
	if len(req.Content) > 0 {
		body := make(map[string]interface{})
		if err := json.Unmarshal(req.Content, &body); err != nil {
			return nil, cli.NewErrorWithTip(
				fmt.Errorf("--estimate-cost requires the request body to be a JSON object: %v", err),
				"cost estimation merges the JSON body into pricing parameters, please pass `--body` as a JSON object")
		}
		for k, v := range body {
			parameters[k] = v
		}
	}
	if req.RegionId != "" {
		if _, ok := parameters["RegionId"]; !ok {
			parameters["RegionId"] = req.RegionId
		}
	}
	return parameters, nil
}

// buildPricingContext collects `--estimate-cost-context Key=Value` entries into
// a map for nesting under parameters.PricingContext. Repeatable and multi-value
// (`--estimate-cost-context K1=V1 K2=V2`). Split on the FIRST `=` so values may
// contain `=`. Keys are not validated — PricingContext is mapping-defined and
// evolving; the quote service validates. Empty value allowed; empty key rejected.
// Returns (nil, nil) when the flag is absent.
func buildPricingContext(ctx *cli.Context) (map[string]interface{}, error) {
	f := EstimateCostContextFlag(ctx.Flags())
	if f == nil || !f.IsAssigned() {
		return nil, nil
	}
	pc := make(map[string]interface{})
	for _, s := range f.GetValues() {
		k, v, ok := cli.SplitStringWithPrefix(s, "=")
		if !ok || k == "" {
			return nil, cli.NewErrorWithTip(
				fmt.Errorf("invalid --estimate-cost-context `%s`", s),
				"use `--estimate-cost-context Key=Value`, e.g. --estimate-cost-context EstimatedInternetTrafficOutGB=100")
		}
		pc[k] = v
	}
	return pc, nil
}

func estimateCostEndpoint() string {
	if v := os.Getenv(estimateCostEndpointEnv); v != "" {
		return v
	}
	return defaultEstimateCostHost
}

// invokeEstimateCost issues the signed quote request and returns the raw
// response body. Reuses the standard authenticated client so signing and
// credential refresh behave identically to a real OpenAPI call.
func invokeEstimateCost(ctx *cli.Context, profile *config.Profile, popCode string, popVersion string, apiName string, parameters map[string]interface{}) (string, error) {
	client, err := GetClient(profile, ctx)
	if err != nil {
		return "", fmt.Errorf("init estimate-cost client failed: %s", err)
	}

	content, err := json.Marshal(estimateCostRequest{
		PopCode:    popCode,
		PopVersion: popVersion,
		ApiName:    apiName,
		Parameters: parameters,
	})
	if err != nil {
		return "", err
	}

	request := requests.NewCommonRequest()
	request.Product = estimateCostProductCode
	request.Version = estimateCostApiVersion
	request.Method = "POST"
	request.PathPattern = estimateCostQuotePath
	request.Domain = estimateCostEndpoint()
	request.Scheme = "https"
	if host := os.Getenv(estimateCostHostEnv); host != "" {
		request.Headers["Host"] = host
	}
	request.SetContent(content)
	request.SetContentType("application/json")

	resp, err := client.ProcessCommonRequest(request)
	if err != nil {
		return "", translateEstimateCostError(popCode, popVersion, apiName, err)
	}
	return resp.GetHttpContentString(), nil
}

// translateEstimateCostError turns ccapi-side server errors into actionable
// CLI tips. PricingNotSupported is the common "this API isn't billable"
// signal — surface that as a friendly hint rather than a raw error string,
// since users mistake it for a misconfiguration otherwise.
func translateEstimateCostError(popCode string, popVersion string, apiName string, err error) error {
	if serverErr, ok := err.(*sdkerrors.ServerError); ok {
		switch serverErr.ErrorCode() {
		case "PricingNotSupported":
			return cli.NewErrorWithTip(
				fmt.Errorf("no pricing information for %s/%s/%s", popCode, popVersion, apiName),
				"this OpenAPI either incurs no cost or has no pricing mapping registered yet")
		case "InvalidParameter":
			return cli.NewErrorWithTip(err,
				"cost estimation rejected the parameters, please check them against the target API")
		}
	}
	return err
}

// printEstimateCostResult reuses the standard output pipeline so `--quiet`,
// `--cli-query` and `--output` keep working on the estimate JSON.
func printEstimateCostResult(ctx *cli.Context, out string) error {
	if QuietFlag(ctx.Flags()).IsAssigned() {
		return nil
	}
	var err error
	if QueryFlag(ctx.Flags()).IsAssigned() {
		out, err = ApplyQueryFilter(ctx, out)
		if err != nil {
			return err
		}
	}
	if filter := GetOutputFilter(ctx); filter != nil {
		out, err = filter.FilterOutput(out)
		if err != nil {
			return err
		}
	}
	out = sortJSON(out)
	cli.Println(ctx.Stdout(), out)
	return nil
}
