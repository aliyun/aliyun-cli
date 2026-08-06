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

func TestRpcInvoker_Prepare_ArrayJSONFlattensToFormParams(t *testing.T) {
	a := &RpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
		api: &meta.Api{
			Product: &meta.Product{Code: "alb"},
			Name:    "AddServersToServerGroup",
			Method:  "POST",
			Parameters: []meta.Parameter{
				{Name: "ServerGroupId", Position: "Query", Type: "String", Required: true},
				{Name: "Servers", Position: "Body", Type: "Array", Required: true},
			},
		},
	}
	w := new(bufio.Writer)
	stderr := new(bufio.Writer)
	ctx := cli.NewCommandContext(w, stderr)
	ctx.Flags().Add(NewSecureFlag())
	ctx.Flags().Add(NewInsecureFlag())
	ctx.Flags().Add(NewMethodFlag())
	ctx.SetUnknownFlags(cli.NewFlagSet())

	ctx.UnknownFlags().AddByName("ServerGroupId")
	ctx.UnknownFlags().Get("ServerGroupId").SetAssigned(true)
	ctx.UnknownFlags().Get("ServerGroupId").SetValue("sgp-xxx")
	ctx.UnknownFlags().AddByName("Servers")
	ctx.UnknownFlags().Get("Servers").SetAssigned(true)
	ctx.UnknownFlags().Get("Servers").SetValue(`[{"ServerId":"i-xxx","ServerType":"Ecs","Port":8081,"Weight":100}]`)

	err := a.Prepare(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "sgp-xxx", a.request.QueryParams["ServerGroupId"])
	assert.Equal(t, "i-xxx", a.request.FormParams["Servers.1.ServerId"])
	assert.Equal(t, "Ecs", a.request.FormParams["Servers.1.ServerType"])
	assert.Equal(t, "8081", a.request.FormParams["Servers.1.Port"])
	assert.Equal(t, "100", a.request.FormParams["Servers.1.Weight"])
	_, hasRaw := a.request.FormParams["Servers"]
	assert.False(t, hasRaw, "raw JSON Servers key must not remain after flatten")
}

func TestRpcInvoker_Prepare_ArrayFlatFlagsAccepted(t *testing.T) {
	a := &RpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
		api: &meta.Api{
			Product: &meta.Product{Code: "alb"},
			Name:    "AddServersToServerGroup",
			Method:  "POST",
			Parameters: []meta.Parameter{
				{Name: "ServerGroupId", Position: "Query", Type: "String", Required: true},
				{Name: "Servers", Position: "Body", Type: "Array", Required: true},
			},
		},
	}
	w := new(bufio.Writer)
	stderr := new(bufio.Writer)
	ctx := cli.NewCommandContext(w, stderr)
	ctx.Flags().Add(NewSecureFlag())
	ctx.Flags().Add(NewInsecureFlag())
	ctx.Flags().Add(NewMethodFlag())
	ctx.SetUnknownFlags(cli.NewFlagSet())

	ctx.UnknownFlags().AddByName("ServerGroupId")
	ctx.UnknownFlags().Get("ServerGroupId").SetAssigned(true)
	ctx.UnknownFlags().Get("ServerGroupId").SetValue("sgp-xxx")
	for _, kv := range [][2]string{
		{"Servers.1.ServerId", "i-xxx"},
		{"Servers.1.ServerType", "Ecs"},
		{"Servers.1.Port", "8081"},
		{"Servers.1.Weight", "100"},
	} {
		ctx.UnknownFlags().AddByName(kv[0])
		ctx.UnknownFlags().Get(kv[0]).SetAssigned(true)
		ctx.UnknownFlags().Get(kv[0]).SetValue(kv[1])
	}

	err := a.Prepare(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "i-xxx", a.request.FormParams["Servers.1.ServerId"])
	assert.Equal(t, "Ecs", a.request.FormParams["Servers.1.ServerType"])
	assert.Equal(t, "8081", a.request.FormParams["Servers.1.Port"])
	assert.Equal(t, "100", a.request.FormParams["Servers.1.Weight"])
}

func TestRpcInvoker_Prepare_ArrayJSONFlattensToQueryParams(t *testing.T) {
	a := &RpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
		api: &meta.Api{
			Product: &meta.Product{Code: "demo"},
			Name:    "Demo",
			Method:  "POST",
			Parameters: []meta.Parameter{
				{Name: "Tags", Position: "Query", Type: "Array", Required: true},
			},
		},
	}
	w := new(bufio.Writer)
	stderr := new(bufio.Writer)
	ctx := cli.NewCommandContext(w, stderr)
	ctx.Flags().Add(NewSecureFlag())
	ctx.Flags().Add(NewInsecureFlag())
	ctx.Flags().Add(NewMethodFlag())
	ctx.SetUnknownFlags(cli.NewFlagSet())

	ctx.UnknownFlags().AddByName("Tags")
	ctx.UnknownFlags().Get("Tags").SetAssigned(true)
	ctx.UnknownFlags().Get("Tags").SetValue(`[{"Key":"env","Value":"prod"}]`)

	err := a.Prepare(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "env", a.request.QueryParams["Tags.1.Key"])
	assert.Equal(t, "prod", a.request.QueryParams["Tags.1.Value"])
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
