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
	"time"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiUtil "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

type OpenapiInvoker struct {
	*BasicInvoker
	method  string
	path    string
	api     *meta.Api
	params  *openapiClient.Params
	runtime *openapiUtil.RuntimeOptions
}

func (a *OpenapiInvoker) Prepare(ctx *cli.Context) error {

	a.openapiRequest.Headers["Date"] = tea.String(time.Now().Format(time.RFC1123Z))
	a.openapiRequest.Headers["Content-Type"] = tea.String("application/json")

	a.params.Action = tea.String(a.api.Name)
	a.params.Version = &a.api.Product.Version
	a.params.Method = &a.method
	a.params.AuthType = tea.String("AK")
	a.params.Style = tea.String("ROA")
	a.params.Pathname = &a.path
	a.params.ReqBodyType = tea.String("json")
	a.params.BodyType = tea.String("json")

	if v, ok := BodyFlag(ctx.Flags()).GetValue(); ok {
		a.openapiRequest.SetBody([]byte(v))
	}

	if v, ok := BodyFileFlag(ctx.Flags()).GetValue(); ok {
		buf, _ := os.ReadFile(v)
		a.openapiRequest.SetBody(buf)
	}
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
				continue
			} else {
				return fmt.Errorf("unknown parameter position; %s is %s", param.Name, param.Position)
			}
		}

		a.params.Protocol = tea.String(a.api.GetProtocol())
	}

	if _, ok := InsecureFlag(ctx.Flags()).GetValue(); ok {
		a.params.Protocol = tea.String("http")
	}

	if _, ok := SecureFlag(ctx.Flags()).GetValue(); ok {
		a.params.Protocol = tea.String("https")
	}

	return nil
}

func (a *OpenapiInvoker) Execute() (*map[string]interface{}, error) {
	resp, err := a.openapiClient.Execute(a.params, a.openapiRequest, a.runtime)
	return &resp, err
}

func (a *OpenapiInvoker) Call() (*responses.CommonResponse, error) {
	return nil, nil
}
