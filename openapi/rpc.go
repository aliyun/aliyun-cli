// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package openapi

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/meta"
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

	// if `--secure` assigned, use https
	if _, ok := SecureFlag(ctx.Flags()).GetValue(); ok {
		a.request.Scheme = "https"
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

		if param.Position == "Query" {
			request.QueryParams[f.Name], _ = f.GetValue()
		} else if param.Position == "Body" {
			request.FormParams[f.Name], _ = f.GetValue()
		} else if param.Position == "Domain" {
			continue
		} else {
			return fmt.Errorf("unknown parameter position; %s is %s", param.Name, param.Position)
		}
	}

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
	resp, err := a.client.ProcessCommonRequest(a.request)
	// cli.Printf("Resp: %s", resp.String())
	return resp, err
}

func replaceValueWithFile(f *cli.Flag) {
	value, _ := f.GetValue()
	data, err := ioutil.ReadFile(value)
	if err != nil {
		panic(err)
	}
	f.SetValue(string(data))
}
