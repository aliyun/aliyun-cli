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
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	credentialsv2 "github.com/aliyun/credentials-go/credentials"
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

var priceHTTPClient = &nethttp.Client{Timeout: 20 * time.Second}

// PriceModeEnabled reports whether --estimate-cost or the host env toggle
// is active.
func PriceModeEnabled(flag bool) bool {
	if flag {
		return true
	}
	v := os.Getenv(priceModeEnv)
	return v == "1" || strings.EqualFold(v, "true")
}

type priceRequest struct {
	PopCode    string                 `json:"popCode"`
	PopVersion string                 `json:"popVersion"`
	ApiName    string                 `json:"apiName"`
	Parameters map[string]interface{} `json:"parameters"`
}

// EstimateCost quotes the assembled call via CloudControl GetApiPrice
// without invoking the target API. pricingContext is optional Key=Value
// pairs from --estimate-cost-context.
func EstimateCost(ec *ExecContext, req *AssembledRequest, pricingContext map[string]string) (string, error) {
	if ec == nil || req == nil {
		return "", fmt.Errorf("estimate-cost: missing request context")
	}
	if ec.Credential == nil {
		return "", fmt.Errorf("price mode requires resolved credentials; run the command without --estimate-cost once to verify configuration")
	}
	pr := buildPriceRequest(ec, req, pricingContext)
	raw, err := postPriceQuoteSigned(ec.Credential, pricingEndpoint(), pr)
	if err != nil {
		return "", err
	}
	return prettyJSON(raw), nil
}

// IsAPIDryRunRequested reports whether the assembled call carries
// OpenAPI DryRun=true (product precheck). Combined with --estimate-cost,
// the engine runs that upstream dry-run first and only quotes on pass —
// matching aliyun-cli-runtime processEstimateCost.
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

func buildPriceRequest(ec *ExecContext, req *AssembledRequest, pricingContext map[string]string) *priceRequest {
	popCode := os.Getenv(pricingPopCodeEnv)
	if popCode == "" && ec.API != nil {
		popCode = ec.API.ProductCode
	}

	params := map[string]string{}
	for k, v := range req.Query {
		params[k] = v
	}
	if m, ok := req.Body.(map[string]any); ok {
		for k, v := range m {
			if s, ok := priceScalarString(v); ok {
				params[k] = s
			}
		}
	}
	if _, ok := params["RegionId"]; !ok && ec.Region != "" {
		params["RegionId"] = ec.Region
	}

	out := make(map[string]interface{}, len(params))
	for k, v := range params {
		out[k] = v
	}
	if len(pricingContext) > 0 {
		ctx := make(map[string]interface{}, len(pricingContext))
		for k, v := range pricingContext {
			ctx[k] = v
		}
		out["PricingContext"] = ctx
	}

	return &priceRequest{
		PopCode:    popCode,
		PopVersion: req.Version,
		ApiName:    req.Action,
		Parameters: out,
	}
}

func priceScalarString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case bool:
		return fmt.Sprintf("%v", t), true
	case float64, int, int64, json.Number:
		return fmt.Sprintf("%v", t), true
	default:
		return "", false
	}
}

func pricingEndpoint() string {
	if v := os.Getenv(pricingEndpointEnv); v != "" {
		return v
	}
	return defaultPricingEndpoint
}

func prettyJSON(raw []byte) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return string(raw)
	}
	return buf.String()
}

func postPriceQuoteSigned(cred credentialsv2.Credential, endpoint string, pr *priceRequest) ([]byte, error) {
	credModel, err := cred.GetCredential()
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}
	accessKeyId := tea.StringValue(credModel.AccessKeyId)
	accessKeySecret := tea.StringValue(credModel.AccessKeySecret)
	securityToken := tea.StringValue(credModel.SecurityToken)

	payload, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s%s", endpoint, pricingQuotePath)
	httpReq, err := nethttp.NewRequest(nethttp.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	contentMD5 := md5Base64(payload)
	contentType := "application/json"
	dateStr := time.Now().UTC().Format(nethttp.TimeFormat)
	nonce := randomHex(16)

	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Content-MD5", contentMD5)
	httpReq.Header.Set("Date", dateStr)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("x-acs-signature-nonce", nonce)
	httpReq.Header.Set("x-acs-signature-method", "HMAC-SHA1")
	httpReq.Header.Set("x-acs-signature-version", "1.0")
	httpReq.Header.Set("x-acs-version", pricingApiVersion)
	if securityToken != "" {
		httpReq.Header.Set("x-acs-security-token", securityToken)
		httpReq.Header.Set("x-acs-accesskey-id", accessKeyId)
	}
	if host := os.Getenv(pricingHostEnv); host != "" {
		httpReq.Host = host
	}

	stringToSign := buildROAStringToSign(httpReq, pricingQuotePath, contentMD5, contentType, dateStr)
	signature := hmacSHA1Base64(accessKeySecret, stringToSign)
	httpReq.Header.Set("Authorization", fmt.Sprintf("acs %s:%s", accessKeyId, signature))

	resp, err := priceHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call pricing service: %w", err)
	}
	defer resp.Body.Close()
	// A short read must not be mistaken for a response: the body is what
	// decides success vs failure below, so a truncated one would produce a
	// misleading error — or, worse, an unrecognised cost-irrelevant answer
	// reported as "cannot be quoted", which is the wrong billing signal.
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read pricing service response: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		// A confirmed cost-irrelevant API arrives as an error status, but it
		// is a successful determination — hand back a result document so the
		// caller prints it and succeeds like any other quote.
		if out, isCostIrrelevant := costIrrelevantFromResponse(pr, raw); isCostIrrelevant {
			return out, nil
		}
		return nil, parsePricingHTTPError(pr, resp.StatusCode, raw)
	}
	return raw, nil
}

func buildROAStringToSign(req *nethttp.Request, pathname, contentMD5, contentType, date string) string {
	var canonHeaders []string
	for k := range req.Header {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "x-acs-") {
			canonHeaders = append(canonHeaders, lk)
		}
	}
	sort.Strings(canonHeaders)
	var headerLines []string
	for _, k := range canonHeaders {
		headerLines = append(headerLines, k+":"+req.Header.Get(k))
	}
	return strings.Join([]string{
		req.Method,
		"application/json",
		contentMD5,
		contentType,
		date,
		strings.Join(headerLines, "\n"),
		pathname,
	}, "\n")
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func md5Base64(data []byte) string {
	h := md5.Sum(data)
	return base64.StdEncoding.EncodeToString(h[:])
}

func hmacSHA1Base64(key, data string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
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

// costIrrelevantQuote is what --estimate-cost prints for an API the pricing
// side has confirmed as free.
//
// It is a normal result, not an error: the user asked what the call costs and
// got a definitive answer — nothing. Failing here would make scripts and
// agents that gate on the exit code abort on exactly the operations that are
// free to run. Shape matches aliyun-cli's built-in --estimate-cost path so
// both front-ends print the same document for the same situation.
type costIrrelevantQuote struct {
	CostIrrelevant bool   `json:"costIrrelevant"`
	PopCode        string `json:"popCode"`
	PopVersion     string `json:"popVersion"`
	ApiName        string `json:"apiName"`
	Message        string `json:"message"`
}

// costIrrelevantMessage is the runtime's own wording for the confirmed-free
// answer. The server's message says the same thing but is localized (Chinese),
// and CLI output is read by an international audience and by scripts, so the
// text is fixed here instead of passed through. Matches aliyun-cli's built-in
// --estimate-cost path verbatim.
const costIrrelevantMessage = "this OpenAPI is confirmed to incur no charge, so there is nothing to quote"

// costIrrelevantFromResponse recognises the confirmed-free answer in a non-2xx
// body and renders it as a result document.
func costIrrelevantFromResponse(pr *priceRequest, raw []byte) ([]byte, bool) {
	code, _, _, _ := pricingErrorFields(raw)
	if code != pricingCodeNotRequired {
		return nil, false
	}
	// Marshalling a fixed struct of one bool and four strings cannot fail.
	out, _ := json.Marshal(costIrrelevantQuote{
		CostIrrelevant: true,
		PopCode:        pr.PopCode,
		PopVersion:     pr.PopVersion,
		ApiName:        pr.ApiName,
		Message:        costIrrelevantMessage,
	})
	return out, true
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

// parsePricingHTTPError turns a 4xx/5xx response body into a readable error.
// PricingNotSupported keeps a friendly hint (the user just needs to know the
// API has no pricing mapping yet); other errors lead with code+message so the
// signal is visible at a glance, then requestId on its own line for easy copy,
// then the troubleshoot URL last so it doesn't compete with the message.
//
// The hint says explicitly that "not supported" does not mean "free": an API
// confirmed to incur no charge reports PricingNotRequired and is handled as a
// successful result before this is reached, and reading the two as the same
// thing is a billing-relevant mistake.
func parsePricingHTTPError(pr *priceRequest, status int, raw []byte) error {
	code, message, requestId, recommend := pricingErrorFields(raw)
	if code == "" && message == "" {
		body := strings.TrimSpace(string(raw))
		if len(body) > 256 {
			body = body[:256] + "…"
		}
		return fmt.Errorf("pricing service returned HTTP %d: %s", status, body)
	}
	if code == pricingCodeNotSupported {
		base := fmt.Sprintf("no pricing information for %s/%s/%s: no pricing mapping is registered for this OpenAPI yet, so it cannot be quoted. This does not mean the call is free — an API confirmed to be free is reported as cost-irrelevant instead",
			pr.PopCode, pr.PopVersion, pr.ApiName)
		if requestId != "" {
			base += "\n  requestId: " + requestId
		}
		return fmt.Errorf("%s", base)
	}
	var b strings.Builder
	switch {
	case code != "" && message != "":
		fmt.Fprintf(&b, "%s — %s", code, message)
	case code != "":
		b.WriteString(code)
	default:
		b.WriteString(message)
	}
	if requestId != "" {
		fmt.Fprintf(&b, "\n  requestId: %s", requestId)
	}
	if recommend != "" {
		fmt.Fprintf(&b, "\n  help: %s", recommend)
	}
	return fmt.Errorf("%s", b.String())
}
