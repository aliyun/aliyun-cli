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
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
)

func TestRpcInvoker_Prepare(t *testing.T) {
	a := &RpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
		api: &canonicalmeta.API{
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
	assert.Equal(t, "'--body' is not a valid parameter or flag. See `aliyun help  ecs`.", err.Error())

	a.api.Parameters = []canonicalmeta.Parameter{
		{
			Name: "body", RawName: "body",
			Location: "host",
		},
		{
			Name: "secure", RawName: "secure",
		},
	}
	ctx.UnknownFlags().Add(NewSecureFlag())
	err = a.Prepare(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, "unknown parameter position; body is Host", err.Error())

	a.api.Parameters = []canonicalmeta.Parameter{
		{
			Name: "body", RawName: "body",
			Location: "query",
			Required: true,
		},
		{
			Name: "secure", RawName: "secure",
			Location: "query",
			Required: true,
		},
		{
			Name: "RegionId", RawName: "RegionId",
			Required: true,
		},
		{
			Name: "Action", RawName: "Action",
			Required: true,
		},
	}
	err = a.Prepare(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, "required parameters not assigned: \n  --body\n  --secure\n  --RegionId", err.Error())

	a.api.Parameters = []canonicalmeta.Parameter{
		{
			Name: "body", RawName: "body",
			Location: "body",
		},
		{
			Name: "secure", RawName: "secure",
			Location: "body",
		},
	}
	err = a.Prepare(ctx)
	assert.Nil(t, err)

	a.api.Parameters = []canonicalmeta.Parameter{
		{
			Name: "body", RawName: "body",
			Location: "host",
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

func TestRpcInvoker_PrepareFormDataSubParameter(t *testing.T) {
	a := &RpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
		api: &canonicalmeta.API{
			Name:     "CreateEr",
			Protocol: "https",
			Method:   "POST",
			Parameters: []canonicalmeta.Parameter{
				{
					Name:       "Tag",
					RawName:    "Tag",
					Type:       "array",
					Location:   "form",
					ParamStyle: "repeatList",
					ElementFields: []canonicalmeta.Field{
						{Name: "Key", RawName: "Key", Type: "string"},
					},
				},
			},
		},
	}
	w := new(bufio.Writer)
	stderr := new(bufio.Writer)
	ctx := cli.NewCommandContext(w, stderr)
	ctx.SetUnknownFlags(cli.NewFlagSet())
	flag, err := ctx.UnknownFlags().AddByName("Tag.1.Key")
	assert.NoError(t, err)
	flag.SetAssigned(true)
	flag.SetValue("tag-key")

	err = a.Prepare(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "tag-key", a.request.FormParams["Tag.1.Key"])
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
