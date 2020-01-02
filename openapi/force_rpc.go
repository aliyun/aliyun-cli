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
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/cli"
)

type ForceRpcInvoker struct {
	*BasicInvoker
	method string
}

func (a *ForceRpcInvoker) Prepare(ctx *cli.Context) error {
	// assign api name
	a.request.ApiName = a.method

	// assign parameters
	for _, f := range ctx.UnknownFlags().Flags() {
		a.request.QueryParams[f.Name], _ = f.GetValue()
	}

	// --secure use https
	if _, ok := SecureFlag(ctx.Flags()).GetValue(); ok {
		a.request.Scheme = "https"
	}
	return nil
}

func (a *ForceRpcInvoker) Call() (*responses.CommonResponse, error) {
	resp, err := a.client.ProcessCommonRequest(a.request)
	return resp, err
}
