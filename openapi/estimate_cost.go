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
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	pricingruntime "github.com/aliyun/aliyun-openapi-runtime/runtime"
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

	parameters, err := buildEstimateCostParameters(pricingruntime.PricingParameterInput{
		Query: req.QueryParams, Form: req.FormParams, Path: req.PathParams,
		Body: req.Content, Region: req.RegionId,
	})
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
	return finishEstimateCost(ctx, out)
}

// finishEstimateCost classifies the complete document before output filtering.
func finishEstimateCost(ctx *cli.Context, out string) error {
	resultErr := estimateCostBusinessError(out)
	var business *pricingruntime.PricingBusinessError
	if resultErr != nil && !errors.As(resultErr, &business) {
		return resultErr
	}
	if err := printEstimateCostResult(ctx, out); err != nil {
		return err
	}
	return resultErr
}

func estimateCostBusinessError(out string) error {
	return pricingruntime.PricingResponseError(out)
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
	return finishEstimateCost(ctx, out)
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
	return finishEstimateCost(ctx, out)
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
			param := oc.api.FindLegacyParameter(f.Name)
			if param == nil || (param.LegacyPosition() != "Path" && param.LegacyPosition() != "Host") {
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
	if err := pricingruntime.MergePricingBody(parameters, rawBody); err != nil {
		return cli.NewErrorWithTip(err, "cost estimation merges the JSON body into pricing parameters, please pass `--body` as a JSON object")
	}
	return nil
}

func mergeEstimateCostJSONBody(parameters map[string]interface{}, raw []byte) error {
	return mergeEstimateCostBody(parameters, raw)
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
		lookupVersion := req.Version
		if r.product != nil && r.product.Version != "" {
			lookupVersion = r.product.Version
		}
		if api, found := library.GetApiByPath(req.Product, lookupVersion, r.method, r.path); found && api.Name != "" {
			return api.Name, nil
		}
		return "", cli.NewErrorWithTip(
			fmt.Errorf("--estimate-cost cannot resolve the api name for `%s %s`", r.method, r.path),
			"cost estimation needs the api name, please use the `aliyun <product> <ApiName>` form")
	}
	return "", fmt.Errorf("--estimate-cost cannot resolve the api name for this call")
}

// buildEstimateCostParameters adapts SDK-independent prepared parameter slots.
func buildEstimateCostParameters(input pricingruntime.PricingParameterInput) (map[string]interface{}, error) {
	parameters, err := pricingruntime.BuildPricingParameters(input)
	if err != nil {
		return nil, cli.NewErrorWithTip(err, "cost estimation merges the JSON body into pricing parameters, please pass `--body` as a JSON object")
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
	return pricingruntime.PricingEndpoint()
}

func newPricingClient(ctx *cli.Context, profile *config.Profile) (*pricingruntime.PricingClient, error) {
	// Preserve the existing main-CLI credential modes and region requirement.
	if profile.Mode == config.Anonymous {
		return nil, fmt.Errorf("price mode requires authenticated credentials")
	}
	if profile.Mode == config.BearerToken {
		return nil, config.ErrBearerTokenRequiresPlugin("")
	}
	if profile.RegionId == "" {
		return nil, fmt.Errorf("default RegionId is empty! run `aliyun configure` first")
	}
	credential, err := profile.GetCredential(ctx, nil)
	if err != nil {
		return nil, err
	}
	readTimeout, connectTimeout := 10*time.Second, 5*time.Second
	if profile.ReadTimeout > 0 {
		readTimeout = time.Duration(profile.ReadTimeout) * time.Second
	}
	if profile.ConnectTimeout > 0 {
		connectTimeout = time.Duration(profile.ConnectTimeout) * time.Second
	}
	return pricingruntime.NewPricingClient(pricingruntime.PricingClientOptions{
		Credential: credential, Region: profile.RegionId,
		ReadTimeout: readTimeout, ConnectTimeout: connectTimeout,
		RetryCount:       profile.RetryCount,
		SkipSecureVerify: config.SkipSecureVerify(ctx.Flags()).IsAssigned(),
		UserAgent:        os.Getenv("ALIYUN_USER_AGENT"),
	})
}

// invokeEstimateCost issues only the quote-service request through SDK v2.
func invokeEstimateCost(ctx *cli.Context, profile *config.Profile, popCode string, popVersion string, apiName string, parameters map[string]interface{}) (string, error) {
	client, err := newPricingClient(ctx, profile)
	if err != nil {
		return "", fmt.Errorf("init estimate-cost client failed: %w", err)
	}
	defer client.Close()
	language := profile.Language
	if language == "" {
		language = i18n.GetLanguage()
	}
	telemetry.SetOpenAPIRuntimeVersion(pricingruntime.Version)
	return client.Quote(pricingruntime.PricingRequest{
		Language: language, PopCode: popCode, PopVersion: popVersion,
		ApiName: apiName, Parameters: parameters,
	})
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
	out = formatResponseJSON(ctx, out)
	cli.Println(ctx.Stdout(), out)
	return nil
}
