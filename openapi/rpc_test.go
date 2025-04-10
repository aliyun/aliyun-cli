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
	"bufio"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
)

func TestRpcInvoker_Prepare(t *testing.T) {
	a := &RpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
		api: &meta.Api{
			Product: &meta.Product{
				Code: "ecs",
			},
			Name:     "ecs",
			Protocol: "https",
			Method:   "GET",
		},
	}
	w := new(bufio.Writer)
	stderr := new(bufio.Writer)
	ctx := cli.NewCommandContext(w, stderr)

	secureflag := NewSecureFlag()
	secureflag.SetAssigned(true)
	ctx.Flags().Add(secureflag)
	ctx.Flags().Add(NewInsecureFlag())
	methodflag := NewMethodFlag()
	methodflag.SetAssigned(true)
	methodflag.SetValue("POST")
	ctx.Flags().Add(methodflag)

	ctx.SetUnknownFlags(cli.NewFlagSet())
	a.Prepare(ctx)
	assert.Equal(t, "POST", a.request.Method)
	ctx.UnknownFlags().Add(NewBodyFlag())
	err := a.Prepare(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, "'--body' is not a valid parameter or flag. See `aliyun help ecs ecs`.", err.Error())

	a.api.Parameters = []meta.Parameter{
		{
			Name:     "body",
			Position: "Domain",
		},
		{
			Name: "secure",
		},
	}
	ctx.UnknownFlags().Add(NewSecureFlag())
	err = a.Prepare(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, "unknown parameter position; secure is ", err.Error())

	a.api.Parameters = []meta.Parameter{
		{
			Name:     "body",
			Position: "Query",
			Required: true,
		},
		{
			Name:     "secure",
			Position: "Query",
			Required: true,
		},
		{
			Name:     "RegionId",
			Required: true,
		},
		{
			Name:     "Action",
			Required: true,
		},
	}
	err = a.Prepare(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, "required parameters not assigned: \n  --body\n  --secure\n  --RegionId", err.Error())

	a.api.Parameters = []meta.Parameter{
		{
			Name:     "body",
			Position: "Body",
		},
		{
			Name:     "secure",
			Position: "Body",
		},
	}
	err = a.Prepare(ctx)
	assert.Nil(t, err)

	a.api.Parameters = []meta.Parameter{
		{
			Name:     "body",
			Position: "Domain",
		},
	}
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("body-FILE")
	defer func() {
		e := recover()
		assert.NotNil(t, e)
	}()
	a.Prepare(ctx)

}

func TestRpcInvoker_Call(t *testing.T) {
	client, err := sdk.NewClientWithAccessKey("regionid", "accesskeyid", "accesskeysecret")
	assert.Nil(t, err)

	a := &RpcInvoker{
		BasicInvoker: &BasicInvoker{
			client:  client,
			request: requests.NewCommonRequest(),
		},
	}
	_, err = a.Call()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "[SDK.CanNotResolveEndpoint] Can not resolve endpoint")
}
