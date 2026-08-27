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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	slsgateway "github.com/alibabacloud-go/alibabacloud-gateway-sls/client"
	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	openapiTeaUtils "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	slsUtils "github.com/aliyun/aliyun-cli/v3/sls"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/otel"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/throttlingretry"
	"github.com/aliyun/aliyun-cli/v3/util"
)

func ShouldUseOpenapi(ctx *cli.Context, product *meta.Product) bool {
	// SLS is routed through the darabonba-openapi/v2 channel (v3 signature).
	// Should be applied to all products later.
	switch strings.ToLower(product.Code) {
	case "sls":
		return true
	}
	return false
}

var hookHttpContextCall = func(fn func() error) func() error {
	return fn
}

var hookHttpContextGetResponse = func(fn func() (string, error)) func() (string, error) {
	return fn
}

func GetOpenapiClient(cp *config.Profile, ctx *cli.Context, product *meta.Product) (client *openapiClient.Client, err error) {
	if cp.RegionId == "" {
		err = fmt.Errorf("default RegionId is empty! run `aliyun configure` first")
		return
	}
	credential, err := cp.GetCredential(ctx, nil)
	if err != nil {
		return
	}
	conf := openapiClient.Config{
		Credential: credential,
		RegionId:   tea.String(cp.RegionId),
		// AccessKeyId:     tea.String(cp.AccessKeyId),
		// AccessKeySecret: tea.String(cp.AccessKeySecret),
	}
	if cp.Endpoint != "" {
		conf.Endpoint = tea.String(cp.Endpoint)
	} else if strings.ToLower(product.Code) == "sls" {
		conf.Endpoint = tea.String(cp.RegionId + ".log.aliyuncs.com") // should apply product template
	} else if product.LocationServiceCode == "" {
		// Products without a location service resolve their endpoint from the
		// static regional endpoint table, so a nil sdk client is safe here.
		ep, e := product.GetEndpointWithType(cp.RegionId, nil, cp.EndpointType)
		if e != nil {
			// An explicit --endpoint is applied as a request-time override, so a
			// failed table lookup is only fatal when no override is present.
			// Otherwise surface the resolution error (e.g. region not in the
			// table and no global endpoint) instead of sending a request with no host.
			if _, ok := config.EndpointFlag(ctx.Flags()).GetValue(); !ok {
				return nil, e
			}
		} else {
			conf.Endpoint = tea.String(ep)
		}
	}

	ua := util.GetAliyunCliUserAgent()
	if v, ok := UserAgentFlag(ctx.Flags()).GetValue(); ok {
		ua += " " + util.SanitizeUserAgent(v)
	}
	if suf := aiModeSuffixForContext(ctx); suf != "" {
		ua += " " + suf
	}
	conf.SetUserAgent(ua)

	if cp.ReadTimeout > 0 {
		conf.SetReadTimeout(cp.ReadTimeout * 1000)
	}
	if cp.ConnectTimeout > 0 {
		conf.SetConnectTimeout(cp.ConnectTimeout * 1000)
	}
	client, err = openapiClient.NewClient(&conf)
	if err != nil {
		return
	}
	client.DisableSDKError = tea.Bool(true)
	if strings.ToLower(product.Code) == "sls" {
		client.Spi = &slsgateway.Client{} // host management for sls endpoint
	}
	return client, err
}

func openapiThrottlingRetryConfig(ctx *cli.Context) *throttlingretry.Config {
	cfg, err := throttlingretry.LoadEffective(config.GetConfigDir(ctx))
	if err != nil {
		return throttlingretry.Default()
	}
	return cfg
}

func openapiThrottlingRetryEnabled(cfg *throttlingretry.Config) bool {
	if cfg.Enabled != nil && !*cfg.Enabled {
		return false
	}
	return true
}

func openapiThrottlingRetryMaxAttempts(cfg *throttlingretry.Config) int {
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultThrottlingRetryMaxAttempts
	}
	return maxAttempts
}

func openapiThrottlingRetryMaxDelayMS(cfg *throttlingretry.Config) int64 {
	maxDelay := cfg.MaxDelayMS
	if maxDelay <= 0 {
		maxDelay = defaultThrottlingRetryMaxDelayMS
	}
	return maxDelay
}

func openapiThrottlingRetryDelay(err error, cfg *throttlingretry.Config) (int64, bool) {
	if !openapiThrottlingRetryEnabled(cfg) {
		return 0, false
	}

	var throttlingErr *openapiClient.ThrottlingError
	if !errors.As(err, &throttlingErr) {
		return 0, false
	}

	retryAfter := throttlingErr.GetRetryAfter()
	if retryAfter == nil || *retryAfter < 0 {
		return 0, false
	}

	delayMS := *retryAfter
	if maxDelay := openapiThrottlingRetryMaxDelayMS(cfg); maxDelay > 0 && delayMS > maxDelay {
		delayMS = maxDelay
	}
	return delayMS, true
}

func applyOpenAPIRetryHeaders(headers map[string]*string, retryAttempt int, delayMS int64) {
	if retryAttempt <= 0 || headers == nil {
		return
	}
	headers["x-acs-retry-attempts"] = tea.String(fmt.Sprintf("%d", retryAttempt))
	headers["x-acs-retry-delay"] = tea.String(fmt.Sprintf("%d", delayMS))
}

func GetContentFromApiResponse(response map[string]any) string {
	out := ""
	responseBody := response["body"]
	if responseBody == nil {
		return out
	}
	switch v := responseBody.(type) {
	case string:
		out = v
	case map[string]any, []any:
		jsonData, _ := json.Marshal(v)
		out = string(jsonData)
	case []byte:
		out = string(v)
	default:
		out = fmt.Sprintf("%v", v)
	}
	return out
}

type HttpInvoker interface {
	getRequest() *openapiutil.OpenApiRequest
	Prepare(ctx *cli.Context) error
	Call() error
	GetResponse() (string, error)
}

type HttpContext struct {
	profile         *config.Profile
	openapiClient   *openapiClient.Client
	openapiRequest  *openapiutil.OpenApiRequest
	openapiRuntime  *openapiTeaUtils.RuntimeOptions
	openapiParams   *openapiClient.Params
	openapiResponse map[string]any
	product         *meta.Product

	throttlingRetryConfig *throttlingretry.Config
}

func NewHttpContext(cp *config.Profile) *HttpContext {
	return &HttpContext{profile: cp}
}

func (a *HttpContext) getRequest() *openapiutil.OpenApiRequest {
	return a.openapiRequest
}

func (a *HttpContext) Init(ctx *cli.Context, product *meta.Product) error {
	var err error
	initThrottlingLog(ctx)
	a.product = product
	a.openapiRequest = &openapiutil.OpenApiRequest{
		Query:   map[string]*string{},
		Headers: map[string]*string{},
		HostMap: map[string]*string{},
	}
	a.openapiParams = &openapiClient.Params{}
	a.openapiParams.AuthType = tea.String(a.profile.OpenAPIAuthType())
	a.openapiParams.Style = tea.String("ROA")
	a.openapiParams.ReqBodyType = tea.String("json")
	a.openapiParams.BodyType = tea.String("json")
	a.openapiParams.Protocol = tea.String("HTTPS")

	a.openapiRuntime = &openapiTeaUtils.RuntimeOptions{}
	if config.SkipSecureVerify(ctx.Flags()).IsAssigned() {
		a.openapiRuntime.SetIgnoreSSL(true)
	}
	a.throttlingRetryConfig = openapiThrottlingRetryConfig(ctx)

	if v, ok := config.EndpointFlag(ctx.Flags()).GetValue(); ok {
		a.openapiRequest.EndpointOverride = tea.String(v)
	}
	if ctx.Flags() != nil && HeaderFlag(ctx.Flags()) != nil {
		for _, s := range HeaderFlag(ctx.Flags()).GetValues() {
			if k, v, ok := cli.SplitStringWithPrefix(s, "="); ok {
				a.openapiRequest.Headers[k] = tea.String(v)
			} else {
				return &InvalidHeaderError{
					Input:          s,
					ExpectedFormat: "HeaderName=Value",
					Err:            fmt.Errorf("invalid flag --header `%s` use `--header HeaderName=Value`", s),
				}
			}
		}
	}

	a.openapiClient, err = GetOpenapiClient(a.profile, ctx, product)
	if err != nil {
		return fmt.Errorf("init openapi client failed, %s", err)
	}
	a.profile.InjectBearerTokenHeader(a.openapiRequest.Headers)
	otel.InjectTeaHeaders(a.openapiRequest.Headers)
	// 注：sls 等自建网关产品已在 selfBuiltGatewayProducts 中，下面的调用对它会直接跳过。
	applyCallContextTeaHeaders(product.Code, a.openapiRequest.Headers)

	// a.openapiRequest.Headers["x-acs-region-id"] = tea.String(a.profile.RegionId)
	return nil
}

func (a *HttpContext) Call() error {
	cfg := a.throttlingRetryConfig
	if cfg == nil {
		cfg = throttlingretry.Default()
	}
	retried := false
	retryDelayMS := int64(0)
	maxAttempts := openapiThrottlingRetryMaxAttempts(cfg)
	for retryAttempt := 0; ; retryAttempt++ {
		if retryAttempt > 0 {
			applyOpenAPIRetryHeaders(a.openapiRequest.Headers, retryAttempt, retryDelayMS)
		}
		resp, err := httpContextExecuteFunc(a)
		a.openapiResponse = resp
		if err == nil {
			return nil
		}

		delayMS, ok := openapiThrottlingRetryDelay(err, cfg)
		if !ok || retryAttempt >= maxAttempts {
			if ok && retried {
				printThrottlingRetryExhausted(maxAttempts)
			}
			return dara.TeaSDKError(err)
		}

		printThrottlingRetryNotice(delayMS, retryAttempt+1, maxAttempts)
		retryDelayMS = delayMS
		time.Sleep(time.Duration(delayMS) * time.Millisecond)
		retried = true
	}
}

var httpContextExecuteFunc = func(a *HttpContext) (map[string]interface{}, error) {
	// Products with a self-built gateway (e.g. sls) plug in a Spi that handles
	// host/endpoint management, so they go through Execute. Products without a
	// gateway use CallApi, which signs and sends the request directly
	// with the v3 (ACS3-HMAC-SHA256) algorithm and no Spi dependency.
	if a.openapiClient.Spi != nil {
		return a.openapiClient.Execute(a.openapiParams, a.openapiRequest, a.openapiRuntime)
	}
	return a.openapiClient.CallApi(a.openapiParams, a.openapiRequest, a.openapiRuntime)
}

// openapiCallSSEFunc launches the underlying SSE call in a goroutine. It is a
// package-level variable so tests can substitute a fake event producer.
var openapiCallSSEFunc = func(a *OpenapiContext, yield chan *openapiClient.SSEResponse, yieldErr chan error) {
	go a.openapiClient.CallSSEApi(a.openapiParams, a.openapiRequest, a.openapiRuntime, yield, yieldErr)
}

type OpenapiContext struct {
	*HttpContext
	method string
	path   string
	api    *canonicalmeta.API
}

func (a *OpenapiContext) ProcessPullLogsHeaders(ctx *cli.Context) {
	a.openapiRequest.Headers["Accept-Encoding"] = tea.String("lz4")
	a.openapiRequest.Headers["accept"] = tea.String("application/x-protobuf")
	a.openapiParams.BodyType = tea.String("byte")
}

func (a *OpenapiContext) ProcessHeaders(ctx *cli.Context) error {
	for _, f := range ctx.UnknownFlags().Flags() {
		param := a.api.FindLegacyParameter(f.Name)
		if param == nil {
			return NewInvalidParameterErrorFromCanonical(f.Name, a.api, a.product.GetLowerCode(), ctx.Flags())
		}
		if param.LegacyPosition() != "header" {
			continue
		}
		value, _ := f.GetValue()
		if param.LegacyRequired() && value == "" {
			return fmt.Errorf("required parameter missing; %s is required", param.LegacyName())
		}
		a.openapiRequest.Headers[f.Name] = &value
	}
	if a.product.GetLowerCode() == "sls" && a.api.Name == "PullLogs" {
		a.ProcessPullLogsHeaders(ctx)
	}
	return nil
}

func (a *OpenapiContext) ProcessPutLogsBody(ctx *cli.Context) error {
	var body []byte
	if v, ok := BodyFlag(ctx.Flags()).GetValue(); ok {
		body = []byte(v)
	}

	if v, ok := BodyFileFlag(ctx.Flags()).GetValue(); ok && strings.TrimSpace(v) != "" {
		buf, err := os.ReadFile(v)
		if err != nil {
			return &InvalidBodyFileError{Path: v, Err: fmt.Errorf("--body-file: %w", err)}
		}
		body = buf
	}
	if body == nil {
		return fmt.Errorf("no logs provided, please check the input")
	}
	compressedData, rawSize, err := slsUtils.PreparePutLogsData(body)
	if err != nil {
		return err
	}
	if len(compressedData) > 10*1024*1024 {
		return fmt.Errorf("log group size is too large, exceed 10MB")
	}
	a.openapiRequest.Headers["content-type"] = tea.String("application/x-protobuf")
	a.openapiRequest.Headers["x-log-bodyrawsize"] = tea.String(strconv.Itoa(rawSize))
	a.openapiRequest.Headers["x-log-compresstype"] = tea.String("lz4")
	a.openapiParams.ReqBodyType = tea.String("binary")
	a.openapiRequest.SetBody(compressedData)
	return nil
}

func (a *OpenapiContext) ProcessBody(ctx *cli.Context) error {
	if a.product.GetLowerCode() == "sls" && a.api.Name == "PutLogs" {
		return a.ProcessPutLogsBody(ctx)
	}
	if v, ok := BodyFlag(ctx.Flags()).GetValue(); ok {
		a.openapiRequest.SetBody([]byte(v))
	}

	if v, ok := BodyFileFlag(ctx.Flags()).GetValue(); ok && strings.TrimSpace(v) != "" {
		buf, err := os.ReadFile(v)
		if err != nil {
			return &InvalidBodyFileError{Path: v, Err: fmt.Errorf("--body-file: %w", err)}
		}
		a.openapiRequest.SetBody(buf)
	}

	body := map[string]interface{}{}
	for _, f := range ctx.UnknownFlags().Flags() {
		param := a.api.FindLegacyParameter(f.Name)
		if param == nil {
			return NewInvalidParameterErrorFromCanonical(f.Name, a.api, a.product.GetLowerCode(), ctx.Flags())
		}
		if param.LegacyPosition() != "Body" {
			continue
		}
		value, _ := f.GetValue()
		if param.LegacyRequired() && value == "" {
			return fmt.Errorf("required parameter missing; %s is required", param.LegacyName())
		}
		body[f.Name] = value
	}
	if len(body) > 0 {
		a.openapiRequest.Body = body
		// RPC-style products (e.g. DAS) carry their body-position params as
		// formData: the classic RPC channel sends them as form fields
		// (application/x-www-form-urlencoded), so switch ReqBodyType away from
		// the JSON default set in Init. ROA products keep the JSON body.
		if a.openapiParams != nil && tea.StringValue(a.openapiParams.Style) == "RPC" {
			a.openapiParams.ReqBodyType = tea.String("formData")
		}
	}

	return nil
}

func (a *OpenapiContext) ProcessPath(ctx *cli.Context) error {
	pathParams := make(map[string]string)
	pathname := a.path
	for _, f := range ctx.UnknownFlags().Flags() {
		param := a.api.FindLegacyParameter(f.Name)
		if param == nil {
			return NewInvalidParameterErrorFromCanonical(f.Name, a.api, a.product.GetLowerCode(), ctx.Flags())
		}
		if param.LegacyPosition() != "Path" {
			continue
		}
		value, _ := f.GetValue()
		if param.LegacyRequired() && value == "" {
			return fmt.Errorf("required parameter missing; %s is required", param.LegacyName())
		}
		if param.IsWildcard() {
			pathname = value
			continue
		}
		pathParams[f.Name] = value
	}
	if len(pathParams) > 0 {
		for key, value := range pathParams {
			placeholder := "[" + key + "]"
			pathname = strings.ReplaceAll(pathname, placeholder, value)
		}
	}
	a.openapiParams.Pathname = tea.String(pathname)
	return nil
}

func (a *OpenapiContext) ProcessHost(ctx *cli.Context) error {
	for _, f := range ctx.UnknownFlags().Flags() {
		param := a.api.FindLegacyParameter(f.Name)
		if param == nil {
			return NewInvalidParameterErrorFromCanonical(f.Name, a.api, a.product.GetLowerCode(), ctx.Flags())
		}
		if param.LegacyPosition() != "Host" {
			continue
		}
		value, _ := f.GetValue()
		if param.LegacyRequired() && value == "" {
			return fmt.Errorf("required parameter missing; %s is required", param.LegacyName())
		}
		a.openapiRequest.HostMap[strings.ToLower(f.Name)] = tea.String(value)
	}
	return nil
}

func (a *OpenapiContext) ProcessQuery(ctx *cli.Context) error {
	for _, f := range ctx.UnknownFlags().Flags() {
		param := a.api.FindLegacyParameter(f.Name)
		if param == nil {
			return NewInvalidParameterErrorFromCanonical(f.Name, a.api, a.product.GetLowerCode(), ctx.Flags())
		}
		if param.LegacyPosition() != "Query" {
			continue
		}
		value, _ := f.GetValue()
		if param.LegacyRequired() && value == "" {
			return fmt.Errorf("required parameter missing; %s is required", param.LegacyName())
		}
		a.openapiRequest.Query[f.Name] = &value
	}
	return nil
}

type Processor func(ctx *cli.Context) error

func (a *OpenapiContext) Prepare(ctx *cli.Context) error {
	if a.api == nil {
		return fmt.Errorf("api not found, should not happen")
	}
	oaParams := a.openapiParams
	oaParams.Action = tea.String(a.api.Name)
	oaParams.Version = &a.product.Version
	oaParams.Method = &a.method

	// Style defaults to ROA (set in Init); RPC-style products override it here so
	// the darabonba channel adds the RPC-specific request headers.
	if strings.ToLower(a.product.ApiStyle) == "rpc" {
		oaParams.Style = tea.String("RPC")
	}

	oaParams.Protocol = tea.String(a.api.GetProtocol())
	if _, ok := InsecureFlag(ctx.Flags()).GetValue(); ok {
		oaParams.Protocol = tea.String("http")
	}

	if _, ok := SecureFlag(ctx.Flags()).GetValue(); ok {
		oaParams.Protocol = tea.String("https")
	}
	if ctx.UnknownFlags() == nil {
		return fmt.Errorf("no parameters provided, please check")
	}

	return a.RequestProcessors(ctx)
}

func (a *OpenapiContext) checkRequiredParameters(ctx *cli.Context) error {
	// 收集所有 required 的 Path 和 Host 参数
	requiredPathParams := make(map[string]bool)
	requiredHostParams := make(map[string]bool)

	for _, param := range a.api.LegacyTopLevelParameters() {
		if param.LegacyPosition() == "Host" && param.LegacyRequired() {
			requiredHostParams[param.LegacyName()] = false
		}
		if param.LegacyPosition() == "Path" && param.LegacyRequired() {
			requiredPathParams[param.LegacyName()] = false
		}
	}

	for _, f := range ctx.UnknownFlags().Flags() {
		param := a.api.FindLegacyParameter(f.Name)
		if param == nil {
			continue
		}
		if param.LegacyPosition() == "Host" && param.LegacyRequired() {
			value, _ := f.GetValue()
			if value != "" {
				requiredHostParams[param.LegacyName()] = true
			}
		}
		if param.LegacyPosition() == "Path" && param.LegacyRequired() {
			value, _ := f.GetValue()
			if value != "" {
				requiredPathParams[param.LegacyName()] = true
			}
		}
	}

	allMissing := make([]string, 0)
	for paramName, filled := range requiredHostParams {
		if !filled {
			prefix := "--"
			if len(paramName) == 1 {
				prefix = "-"
			}
			allMissing = append(allMissing, fmt.Sprintf("host parameter %s%s", prefix, paramName))
		}
	}
	for paramName, filled := range requiredPathParams {
		if !filled {
			prefix := "--"
			if len(paramName) == 1 {
				prefix = "-"
			}
			allMissing = append(allMissing, fmt.Sprintf("path parameter %s%s", prefix, paramName))
		}
	}

	if len(allMissing) > 0 {
		sort.Strings(allMissing)
		return fmt.Errorf("required parameters missing: %s", strings.Join(allMissing, ", "))
	}

	return nil
}

func (a *OpenapiContext) RequestProcessors(ctx *cli.Context) error {
	processors := []Processor{
		a.ProcessHeaders,
		a.ProcessBody,
		a.ProcessPath,
		a.ProcessHost,
		a.ProcessQuery,
	}

	for _, p := range processors {
		if err := p(ctx); err != nil {
			return err
		}
	}
	return a.checkRequiredParameters(ctx)
}

func (a *OpenapiContext) CheckResponseForPullLogs(response map[string]any) (string, error) {
	responseBody := response["body"]
	bodyStr, ok := responseBody.(string)
	if !ok {
		return "", fmt.Errorf("invalid response body for pulllogs parsing, please check")
	}

	bodyBytes, err := base64.StdEncoding.DecodeString(bodyStr)
	if err != nil {
		return "", err
	}
	if len(bodyBytes) == 0 {
		return "", nil
	}
	result, err := slsUtils.ProcessPullLogsResponse(bodyBytes)
	if err != nil {
		return "", err
	}
	// extract count and next cursor from headers
	responseHeaders := response["headers"].(map[string]any)
	fmt.Printf("count: %s\n", responseHeaders["x-log-count"])
	fmt.Printf("next_cursor: %s\n", responseHeaders["x-log-cursor"])

	return string(result), nil
}

func (a *OpenapiContext) GetResponse() (string, error) {
	if a.product.GetLowerCode() == "sls" && a.api.Name == "PullLogs" {
		return a.CheckResponseForPullLogs(a.openapiResponse)
	}
	out := GetContentFromApiResponse(a.openapiResponse)

	return out, nil
}

// IsSSE reports whether the API streams Server-Sent Events, detected from the
// protocol candidate string (e.g. "HTTPS|SSE").
func (a *OpenapiContext) IsSSE() bool {
	if a.api == nil {
		return false
	}
	return strings.Contains(strings.ToUpper(a.api.Protocol), "SSE")
}

// CallSSE invokes an SSE API and streams each event's data to w as it arrives.
// It returns once the stream ends or on the first transport/server error.
func (a *OpenapiContext) CallSSE(w io.Writer) error {
	yield := make(chan *openapiClient.SSEResponse)
	// buffered so the producer can report a terminal error without blocking on
	// an unbuffered send while we are still draining events.
	yieldErr := make(chan error, 1)
	openapiCallSSEFunc(a, yield, yieldErr)

	flush := func() {
		if f, ok := w.(interface{ Flush() }); ok {
			f.Flush()
		}
	}

	var callErr error
	// Both channels are closed by the producer when it finishes; select on both
	// so an error sent before close cannot deadlock the drain loop.
	for yield != nil || yieldErr != nil {
		select {
		case resp, ok := <-yield:
			if !ok {
				yield = nil
				continue
			}
			if resp == nil || resp.Event == nil || resp.Event.Data == nil {
				continue
			}
			if _, err := io.WriteString(w, tea.StringValue(resp.Event.Data)+"\n"); err != nil {
				return err
			}
			flush()
		case err, ok := <-yieldErr:
			if !ok {
				yieldErr = nil
				continue
			}
			if err != nil {
				callErr = err
			}
		}
	}
	if callErr != nil {
		return dara.TeaSDKError(callErr)
	}
	return nil
}
