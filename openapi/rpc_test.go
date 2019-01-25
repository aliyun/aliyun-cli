package openapi

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/stretchr/testify/assert"
	"testing"
	"bufio"
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
			Name: "ecs",
			Protocol: "https",
			Method: "GET",
		},
	}
	w := new(bufio.Writer)
	ctx := cli.NewCommandContext(w)

	secureflag := NewSecureFlag()
	secureflag.SetAssigned(true)
	ctx.Flags().Add(secureflag)

	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().Add(NewBodyFlag())
	err := a.Prepare(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, "'--body' is not a valid parameter or flag. See `aliyun help ecs ecs`.", err.Error())

	a.api.Parameters = []meta.Parameter{
		{
			Name: "body",
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
			Name: "body",
			Position: "Query",
			Required:true,
		},
		{
			Name: "secure",
			Position: "Query",
			Required:true,
		},
		{
			Name: "RegionId",
			Required:true,
		},
		{
			Name: "Action",
			Required:true,
		},
	}
	err = a.Prepare(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, "required parameters not assigned: \n  --body\n  --secure\n  --RegionId", err.Error())

	a.api.Parameters = []meta.Parameter{
		{
			Name: "body",
			Position: "Body",
		},
		{
			Name: "secure",
			Position: "Body",
		},
	}
	err = a.Prepare(ctx)
	assert.Nil(t, err)
}

func TestRpcInvoker_Call(t *testing.T) {
	client, err := sdk.NewClientWithAccessKey("regionid", "accesskeyid", "accesskeysecret")
	assert.Nil(t, err)

	a := &RpcInvoker{
		BasicInvoker: &BasicInvoker{
			client: client,
			request: requests.NewCommonRequest(),
		},
	}
	_, err = a.Call()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "[SDK.CanNotResolveEndpoint] Can not resolve endpoint")
}
