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

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

type RpcInvoker struct {
	*BasicInvoker
	api *meta.Api
}

func (a *RpcInvoker) Prepare(ctx *cli.Context) error {
	// tidy names
	api := a.api
	request := a.request

	// assign api name, scheme method
	request.ApiName = api.Name
	request.Scheme = api.GetProtocol()
	request.Method = api.GetMethod()

	// if `--insecure` assigned, use http
	if _, ok := InsecureFlag(ctx.Flags()).GetValue(); ok {
		a.request.Scheme = "http"
	}

	// if `--secure` assigned, use https
	if _, ok := SecureFlag(ctx.Flags()).GetValue(); ok {
		a.request.Scheme = "https"
	}

	// if '--method' assigned, reset method
	if method, ok := MethodFlag(ctx.Flags()).GetValue(); ok {
		if method == "GET" || method == "POST" {
			a.request.Method = method
		} else {
			return fmt.Errorf("--method value %s is not supported, please set method in {GET|POST}", method)
		}
	}

	// assign parameters
	for _, f := range ctx.UnknownFlags().Flags() {
		if strings.HasSuffix(f.Name, "-FILE") {
			f.Name = strings.TrimSuffix(f.Name, "-FILE")
			replaceValueWithFile(f)
		}
		param := api.FindParameter(f.Name)
		if param == nil {
			return &InvalidParameterError{Name: f.Name, api: api, flags: ctx.Flags()}
		}

		value, _ := f.GetValue()
		// Array params often require RPC flat serialization (Servers.1.ServerId).
		// When the user passes a JSON array/object via --Servers '[...]', expand it.
		if param.Type == "Array" && f.Name == param.Name {
			if flat, ok := expandJSONArrayParameter(param.Name, value); ok {
				assignRPCParams(request, param.Position, flat)
				continue
			}
		}

		if param.Position == "Query" {
			request.QueryParams[f.Name] = value
		} else if param.Position == "Body" || param.Position == "FormData" {
			// new add FormData
			request.FormParams[f.Name] = value
		} else if param.Position == "Domain" {
			continue
		} else {
			return fmt.Errorf("unknown parameter position; %s is %s", param.Name, param.Position)
		}
	}
	// check api support Body
	bodyParam := api.FindParameter("body")
	if bodyParam != nil && bodyParam.Position == "Body" {
		if v, ok := BodyFlag(ctx.Flags()).GetValue(); ok {
			a.request.SetContent([]byte(v))
		}
	}

	applyCallContextRPC(a.productCode(), request.QueryParams)

	err := a.api.CheckRequiredParameters(func(s string) bool {
		switch s {
		case "RegionId":
			return request.RegionId != ""
		case "Action":
			return request.ApiName != ""
		default:
			f := ctx.UnknownFlags().Get(s)
			return f != nil && f.IsAssigned()
		}
	})

	if err != nil {
		return cli.NewErrorWithTip(err,
			"use `aliyun %s %s --help` to get more information",
			api.Product.GetLowerCode(), api.Name)
	}
	return nil
}

func (a *RpcInvoker) Call() (*responses.CommonResponse, error) {
	return a.callWithThrottlingRetry(func() (*responses.CommonResponse, error) {
		return a.client.ProcessCommonRequest(a.request)
	})
}

func replaceValueWithFile(f *cli.Flag) {
	value, _ := f.GetValue()
	data, err := os.ReadFile(value)
	if err != nil {
		panic(err)
	}
	f.SetValue(string(data))
}

func assignRPCParams(request *requests.CommonRequest, position string, params map[string]string) {
	for k, v := range params {
		if position == "Query" {
			request.QueryParams[k] = v
		} else if position == "Body" || position == "FormData" {
			request.FormParams[k] = v
		}
	}
}
