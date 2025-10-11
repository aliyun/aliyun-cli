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
	"fmt"
	"os"
	"strings"

	slsgateway "github.com/alibabacloud-go/alibabacloud-gateway-sls/client"
	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	openapiTeaUtils "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

func ShouldUseOpenapi(ctx *cli.Context, product *meta.Product) bool {
	// sls use openapi, should be applied to all products later
	return strings.ToLower(product.Code) == "sls"
}

func GetOpenapiClient(cp *config.Profile, ctx *cli.Context, product *meta.Product) (client *openapiClient.Client, err error) {
	if cp.RegionId == "" {
		err = fmt.Errorf("default RegionId is empty! run `aliyun configure` first")
		return
	}
	conf := openapiClient.Config{
		AccessKeyId:     &cp.AccessKeyId,
		AccessKeySecret: &cp.AccessKeySecret,
	}
	if strings.ToLower(product.Code) == "sls" {
		conf.Endpoint = tea.String(cp.RegionId + ".log.aliyuncs.com") // should apply product template
	}
	conf.RegionId = tea.String(cp.RegionId)

	client, err = openapiClient.NewClient(&conf)
	if err != nil {
		return
	}
	if strings.ToLower(product.Code) == "sls" {
		client.Spi = &slsgateway.Client{} // host management for sls endpoint
	}
	return client, err
}

type ApiInvoker interface {
	getRequest() *openapiutil.OpenApiRequest
	Prepare(ctx *cli.Context) error
	Call() (map[string]any, error)
}

// implementations
// - Waiter
// - Pager
type ApiInvokeHelper interface {
	ApiCallWith(apiInvoker ApiInvoker) (string, error)
}

type ApiContext struct {
	profile        *config.Profile
	openapiClient  *openapiClient.Client
	openapiRequest *openapiutil.OpenApiRequest
	openapiRuntime *openapiTeaUtils.RuntimeOptions
	openapiParams  *openapiClient.Params
	product        *meta.Product
}

func NewApiContext(cp *config.Profile) *ApiContext {
	return &ApiContext{profile: cp}
}

func (a *ApiContext) getRequest() *openapiutil.OpenApiRequest {
	return a.openapiRequest
}

func (a *ApiContext) Init(ctx *cli.Context, product *meta.Product) error {
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

	a.openapiRuntime = &openapiTeaUtils.RuntimeOptions{}

	a.openapiClient, err = GetOpenapiClient(a.profile, ctx, product)
	if err != nil {
		return fmt.Errorf("init openapi client failed %s", err)
	}
	return nil
}

type OpenapiContext struct {
	*ApiContext
	method string
	path   string
	api    *meta.Api
}

func (a *OpenapiContext) Prepare(ctx *cli.Context) error {
	oaParams := a.openapiParams
	oaParams.Action = tea.String(a.api.Name)
	oaParams.Version = &a.api.Product.Version
	oaParams.Method = &a.method

	if v, ok := BodyFlag(ctx.Flags()).GetValue(); ok {
		a.openapiRequest.SetBody([]byte(v))
	}

	if v, ok := BodyFileFlag(ctx.Flags()).GetValue(); ok {
		buf, _ := os.ReadFile(v)
		a.openapiRequest.SetBody(buf)
	}
	pathParams := make(map[string]string)
	// assign parameters
	if a.api == nil {
		for _, f := range ctx.UnknownFlags().Flags() {
			value, _ := f.GetValue()
			a.openapiRequest.Query[f.Name] = &value
		}
	} else {
		for _, f := range ctx.UnknownFlags().Flags() {
			param := a.api.FindParameter(f.Name)
			if param == nil {
				return &InvalidParameterError{Name: f.Name, api: a.api, flags: ctx.Flags()}
			}
			value, _ := f.GetValue()
			if param.Required && value == "" {
				return fmt.Errorf("required parameter missing; %s is required", param.Name)
			}
			if param.Position == "Query" {
				a.openapiRequest.Query[f.Name] = &value
			} else if param.Position == "Body" {
				body := map[string]interface{}{}
				body[f.Name] = value
				a.openapiRequest.Body = body
			} else if param.Position == "Header" {
				a.openapiRequest.Headers[f.Name] = &value
				continue
			} else if param.Position == "Host" {
				a.openapiRequest.HostMap[strings.ToLower(f.Name)] = tea.String(value)
			} else if param.Position == "Domain" {
				continue
			} else if param.Position == "Path" {
				pathParams[strings.ToLower(f.Name)] = value
			} else {
				return fmt.Errorf("unknown parameter position; %s is %s", param.Name, param.Position)
			}
		}

		oaParams.Protocol = tea.String(a.api.GetProtocol())
	}
	pathname := a.path
	if len(pathParams) > 0 {
		// Replace {param} with actual values
		for key, value := range pathParams {
			placeholder := "{" + key + "}"
			pathname = strings.ReplaceAll(pathname, placeholder, value)
		}
	}
	oaParams.Pathname = &pathname

	if _, ok := InsecureFlag(ctx.Flags()).GetValue(); ok {
		oaParams.Protocol = tea.String("http")
	}

	if _, ok := SecureFlag(ctx.Flags()).GetValue(); ok {
		oaParams.Protocol = tea.String("https")
	}

	return nil
}

func (a *OpenapiContext) Call() (map[string]any, error) {
	resp, err := a.openapiClient.Execute(a.openapiParams, a.openapiRequest, a.openapiRuntime)
	return resp, err
}
