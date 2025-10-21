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
	"fmt"
	"os"
	"strconv"
	"strings"

	slsgateway "github.com/alibabacloud-go/alibabacloud-gateway-sls/client"
	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	openapiTeaUtils "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	slsUtils "github.com/aliyun/aliyun-cli/v3/sls"
)

func ShouldUseOpenapi(ctx *cli.Context, product *meta.Product) bool {
	// sls use openapi, should be applied to all products later
	return strings.ToLower(product.Code) == "sls"
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
	if strings.ToLower(product.Code) == "sls" {
		conf.Endpoint = tea.String(cp.RegionId + ".log.aliyuncs.com") // should apply product template
	}

	ua := "Aliyun-CLI/" + cli.GetVersion()
	if vendorEnv, ok := os.LookupEnv("ALIBABA_CLOUD_VENDOR"); ok {
		ua += " vendor/" + vendorEnv
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
	if strings.ToLower(product.Code) == "sls" {
		client.Spi = &slsgateway.Client{} // host management for sls endpoint
	}
	return client, err
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
	case map[string]any:
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
}

func NewHttpContext(cp *config.Profile) *HttpContext {
	return &HttpContext{profile: cp}
}

func (a *HttpContext) getRequest() *openapiutil.OpenApiRequest {
	return a.openapiRequest
}

func (a *HttpContext) Init(ctx *cli.Context, product *meta.Product) error {
	var err error
	a.product = product
	a.openapiRequest = &openapiutil.OpenApiRequest{
		Query:   map[string]*string{},
		Headers: map[string]*string{},
		HostMap: map[string]*string{},
	}
	a.openapiParams = &openapiClient.Params{}
	a.openapiParams.AuthType = tea.String("AK")
	a.openapiParams.Style = tea.String("ROA")
	a.openapiParams.ReqBodyType = tea.String("json")
	a.openapiParams.BodyType = tea.String("json")
	a.openapiParams.Protocol = tea.String("HTTPS")

	a.openapiRuntime = &openapiTeaUtils.RuntimeOptions{}
	if config.SkipSecureVerify(ctx.Flags()).IsAssigned() {
		a.openapiRuntime.SetIgnoreSSL(true)
	}

	if a.profile.RetryCount > 0 {
		// when use --retry-count, enable auto retry
		a.openapiRuntime.SetAutoretry(true)
		a.openapiRuntime.SetMaxAttempts(a.profile.RetryCount)
	}
	if v, ok := EndpointFlag(ctx.Flags()).GetValue(); ok {
		a.openapiRequest.EndpointOverride = tea.String(v)
	}
	if ctx.Flags() != nil && HeaderFlag(ctx.Flags()) != nil {
		for _, s := range HeaderFlag(ctx.Flags()).GetValues() {
			if k, v, ok := cli.SplitStringWithPrefix(s, "="); ok {
				a.openapiRequest.Headers[k] = tea.String(v)
			} else {
				return fmt.Errorf("invaild flag --header `%s` use `--header HeaderName=Value`", s)
			}
		}
	}

	a.openapiClient, err = GetOpenapiClient(a.profile, ctx, product)
	if err != nil {
		return fmt.Errorf("init openapi client failed, %s", err)
	}
	// a.openapiRequest.Headers["x-acs-region-id"] = tea.String(a.profile.RegionId)
	return nil
}

func (a *HttpContext) Call() error {
	resp, err := a.openapiClient.Execute(a.openapiParams, a.openapiRequest, a.openapiRuntime)
	a.openapiResponse = resp
	return err
}

type OpenapiContext struct {
	*HttpContext
	method string
	path   string
	api    *meta.Api
}

func (a *OpenapiContext) ProcessPullLogsHeaders(ctx *cli.Context) {
	a.openapiRequest.Headers["Accept-Encoding"] = tea.String("lz4")
	a.openapiRequest.Headers["accept"] = tea.String("application/x-protobuf")
	a.openapiParams.BodyType = tea.String("byte")
}

func (a *OpenapiContext) ProcessHeaders(ctx *cli.Context) error {
	for _, f := range ctx.UnknownFlags().Flags() {
		param := a.api.FindParameter(f.Name)
		if param == nil {
			return &InvalidParameterError{Name: f.Name, api: a.api, flags: ctx.Flags()}
		}
		if param.Position != "header" {
			continue
		}
		value, _ := f.GetValue()
		if param.Required && value == "" {
			return fmt.Errorf("required parameter missing; %s is required", param.Name)
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

	if v, ok := BodyFileFlag(ctx.Flags()).GetValue(); ok {
		buf, _ := os.ReadFile(v)
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

	if v, ok := BodyFileFlag(ctx.Flags()).GetValue(); ok {
		buf, _ := os.ReadFile(v)
		a.openapiRequest.SetBody(buf)
	}

	for _, f := range ctx.UnknownFlags().Flags() {
		param := a.api.FindParameter(f.Name)
		if param == nil {
			return &InvalidParameterError{Name: f.Name, api: a.api, flags: ctx.Flags()}
		}
		if param.Position != "Body" {
			continue
		}
		value, _ := f.GetValue()
		if param.Required && value == "" {
			return fmt.Errorf("required parameter missing; %s is required", param.Name)
		}
		body := map[string]interface{}{}
		body[f.Name] = value
		a.openapiRequest.Body = body
	}

	return nil
}

func (a *OpenapiContext) ProcessPath(ctx *cli.Context) error {
	pathParams := make(map[string]string)
	for _, f := range ctx.UnknownFlags().Flags() {
		param := a.api.FindParameter(f.Name)
		if param == nil {
			return &InvalidParameterError{Name: f.Name, api: a.api, flags: ctx.Flags()}
		}
		if param.Position != "Path" {
			continue
		}
		value, _ := f.GetValue()
		if param.Required && value == "" {
			return fmt.Errorf("required parameter missing; %s is required", param.Name)
		}
		pathParams[f.Name] = value
	}
	pathname := a.path
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
		param := a.api.FindParameter(f.Name)
		if param == nil {
			return &InvalidParameterError{Name: f.Name, api: a.api, flags: ctx.Flags()}
		}
		if param.Position != "Host" {
			continue
		}
		value, _ := f.GetValue()
		if param.Required && value == "" {
			return fmt.Errorf("required parameter missing; %s is required", param.Name)
		}
		a.openapiRequest.HostMap[strings.ToLower(f.Name)] = tea.String(value)
	}
	return nil
}

func (a *OpenapiContext) ProcessQuery(ctx *cli.Context) error {
	for _, f := range ctx.UnknownFlags().Flags() {
		param := a.api.FindParameter(f.Name)
		if param == nil {
			return &InvalidParameterError{Name: f.Name, api: a.api, flags: ctx.Flags()}
		}
		if param.Position != "Query" {
			continue
		}
		value, _ := f.GetValue()
		if param.Required && value == "" {
			return fmt.Errorf("required parameter missing; %s is required", param.Name)
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
	oaParams.Version = &a.api.Product.Version
	oaParams.Method = &a.method

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
	return nil
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
