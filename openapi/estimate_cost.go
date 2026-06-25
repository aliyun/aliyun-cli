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
	estimateCostApiVersion   = "2022-08-30"
	estimateCostQuotePath    = "/api/v1/price/quote"
	estimateCostEndpointEnv  = "ALIBABA_CLOUD_PRICING_ENDPOINT"
	estimateCostHostEnv      = "ALIBABA_CLOUD_PRICING_HOST"
	defaultEstimateCostHost  = "cloudcontrol.aliyuncs.com"
	estimateCostProductCode  = "cloudcontrol"
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

	out, err := invokeEstimateCost(ctx, &c.profile, req.Product, req.Version, apiName, parameters)
	if err != nil {
		return err
	}
	return printEstimateCostResult(ctx, out)
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
