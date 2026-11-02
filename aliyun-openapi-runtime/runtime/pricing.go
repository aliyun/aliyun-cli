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

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	pricingApiVersion      = "2022-08-30"
	pricingQuotePath       = "/api/v1/price/quote"
	defaultPricingEndpoint = "cloudcontrol.aliyuncs.com"

	priceModeEnv       = "ALIBABA_CLOUD_PRICE_MODE"
	pricingEndpointEnv = "ALIBABA_CLOUD_PRICING_ENDPOINT"
	pricingHostEnv     = "ALIBABA_CLOUD_PRICING_HOST"
	pricingPopCodeEnv  = "ALIBABA_CLOUD_PRICE_POP_CODE"
)

// PriceModeEnabled reports whether --estimate-cost or the host env toggle
// is active.
func PriceModeEnabled(flag bool) bool {
	if flag {
		return true
	}
	v := os.Getenv(priceModeEnv)
	return v == "1" || strings.EqualFold(v, "true")
}

// EstimateCost quotes the assembled call via CloudControl GetApiPrice
// without invoking the target API. pricingContext is optional Key=Value
// pairs from --estimate-cost-context.
func EstimateCost(ec *ExecContext, req *AssembledRequest, pricingContext map[string]string) (string, error) {
	if ec == nil || req == nil {
		return "", fmt.Errorf("estimate-cost: missing request context")
	}
	options := PricingClientOptions{
		Credential: ec.Credential, Region: ec.Region,
		ReadTimeout: ec.ReadTimeout, ConnectTimeout: ec.ConnectTimeout,
		RetryCount: ec.RetryCount, SkipSecureVerify: ec.SkipSecureVerify,
		UserAgent: ComposeUserAgentWithPlugin(ec.CLIVersion, ec.MetadataPluginName, ec.MetadataPluginVersion, ec.UserAgent),
	}
	if options.ReadTimeout <= 0 {
		options.ReadTimeout = 20 * time.Second
	}
	client, err := NewPricingClient(options)
	if err != nil {
		return "", err
	}
	defer client.Close()
	pr, err := buildPriceRequest(ec, req, pricingContext)
	if err != nil {
		return "", err
	}
	raw, err := client.Quote(*pr)
	if err != nil {
		return "", err
	}
	return prettyJSON([]byte(raw)), nil
}

// IsAPIDryRunRequested reports whether the assembled call carries
// OpenAPI DryRun=true (product precheck). Combined with --estimate-cost,
// the engine runs that upstream dry-run first and only quotes on pass —
// preserving the plugin engine's explicit precheck behavior.
func IsAPIDryRunRequested(req *AssembledRequest) bool {
	if req == nil {
		return false
	}
	v, ok := req.Query["DryRun"]
	if !ok {
		return false
	}
	return strings.EqualFold(v, "true") || v == "1"
}

// IsDryRunPassError reports whether err is the product-API "dry-run
// validation passed" signal (HTTP 400 + Code=DryRunOperation).
func IsDryRunPassError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "Code: DryRunOperation") ||
		strings.Contains(s, `"Code":"DryRunOperation"`)
}

// StripAPIDryRun removes the OpenAPI DryRun query param so a subsequent
// price quote does not forward the precheck flag to GetApiPrice.
func StripAPIDryRun(req *AssembledRequest) {
	if req == nil || req.Query == nil {
		return
	}
	delete(req.Query, "DryRun")
}

func buildPriceRequest(ec *ExecContext, req *AssembledRequest, pricingContext map[string]string) (*PricingRequest, error) {
	popCode := os.Getenv(pricingPopCodeEnv)
	if popCode == "" && ec.API != nil {
		popCode = ec.API.ProductCode
	}
	parameters, err := BuildPricingParameters(PricingParameterInput{
		Query: req.Query, Path: req.PathParams, Body: req.Body, Region: ec.Region,
	})
	if err != nil {
		return nil, err
	}
	if len(pricingContext) > 0 {
		context := make(map[string]any, len(pricingContext))
		for key, value := range pricingContext {
			context[key] = value
		}
		parameters["PricingContext"] = context
	}
	return &PricingRequest{
		Language: ec.Language, PopCode: popCode, PopVersion: req.Version,
		ApiName: req.Action, Parameters: parameters,
	}, nil
}

func prettyJSON(raw []byte) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return string(raw)
	}
	return buf.String()
}

// Quote-service error codes the runtime treats specially. They are
// deliberately distinct: "confirmed free" answers the question "what does this
// call cost?", while "not supported" means the question cannot be answered
// yet. Collapsing them leaves users unable to tell a free API from an unmapped
// one — a billing-relevant mistake.
const (
	pricingCodeNotRequired  = "PricingNotRequired"
	pricingCodeNotSupported = "PricingNotSupported"
)

// popErrorEnvelope is the POP gateway error shape (PascalCase).
type popErrorEnvelope struct {
	Code      string `json:"Code"`
	Message   string `json:"Message"`
	RequestId string `json:"RequestId"`
	Recommend string `json:"Recommend"`
}

// priceErrorEnvelope is the ccapi-business error shape (camelCase), seen when
// ALIBABA_CLOUD_PRICING_ENDPOINT points straight at a CloudControl instance
// instead of going through the gateway. Either shape can carry either code, so
// both are understood rather than only the gateway one.
type priceErrorEnvelope struct {
	ErrorCode string `json:"errorCode"`
	ErrorMsg  string `json:"errorMsg"`
	RequestId string `json:"requestId"`
}

// pricingErrorFields extracts the error fields from whichever envelope the
// body carries. An all-empty return means neither shape matched, so the caller
// falls back to reporting the raw body. recommend is gateway-only.
func pricingErrorFields(raw []byte) (code string, message string, requestId string, recommend string) {
	var pe popErrorEnvelope
	if json.Unmarshal(raw, &pe) == nil && pe.Code != "" {
		return pe.Code, pe.Message, pe.RequestId, pe.Recommend
	}
	var env priceErrorEnvelope
	if json.Unmarshal(raw, &env) == nil && (env.ErrorCode != "" || env.ErrorMsg != "") {
		return env.ErrorCode, env.ErrorMsg, env.RequestId, ""
	}
	return "", "", "", ""
}
