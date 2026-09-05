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

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
)

type RpcInvoker struct {
	*BasicInvoker
	api *canonicalmeta.API
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
		param := api.FindLegacyParameter(f.Name)
		if param == nil {
			return NewInvalidParameterErrorFromCanonical(f.Name, api, a.productCode(), ctx.Flags())
		}

		if param.LegacyPosition() == "Query" {
			request.QueryParams[f.Name], _ = f.GetValue()
		} else if param.LegacyPosition() == "Body" || param.LegacyPosition() == "FormData" {
			request.FormParams[f.Name], _ = f.GetValue()
		} else if param.LegacyPosition() == "Domain" {
			continue
		} else {
			return fmt.Errorf("unknown parameter position; %s is %s", param.LegacyName(), param.LegacyPosition())
		}
	}
	// check api support Body
	bodyParam := api.FindLegacyParameter("body")
	if bodyParam != nil && bodyParam.LegacyPosition() == "Body" {
		if v, ok := BodyFlag(ctx.Flags()).GetValue(); ok {
			a.request.SetContent([]byte(v))
		}
	}

	applyCallContextRPC(a.productCode(), request.QueryParams)

	err := a.api.CheckLegacyRequiredParameters(func(s string) bool {
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
		return cli.NewErrorWithTip(newLegacyMissingRequiredError(err),
			"use `aliyun %s %s --help` to get more information",
			strings.ToLower(a.productCode()), api.Name)
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
