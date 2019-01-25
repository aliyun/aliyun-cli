package openapi

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/stretchr/testify/assert"

	"testing"
	"bufio"
)

func TestRestfulInvoker_Prepare(t *testing.T) {
	a := &RestfulInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
	}

	a.BasicInvoker.request.RegionId = "cn-hangzhou"
	a.BasicInvoker.request.Content = []byte("{")
	w := new(bufio.Writer)
	ctx := cli.NewCommandContext(w)

	bodyflag := NewBodyFlag()
	bodyflag.SetAssigned(true)
	ctx.Flags().Add(bodyflag)

	secureflag := NewSecureFlag()
	secureflag.SetAssigned(true)
	ctx.Flags().Add(secureflag)

	bodyfile := NewBodyFileFlag()
	bodyfile.SetAssigned(true)
	ctx.Flags().Add(bodyfile)

	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().Add(NewBodyFlag())
	err := a.Prepare(ctx)
	assert.Nil(t, err)

	BodyFlag(ctx.Flags()).SetAssigned(false)
	BodyFileFlag(ctx.Flags()).SetAssigned(false)
	a.BasicInvoker.request.Content = []byte("{")
	err = a.Prepare(ctx)
	assert.Nil(t, err)

	a.BasicInvoker.request.Headers = map[string]string{}
	a.BasicInvoker.request.Content = []byte("<")
	err = a.Prepare(ctx)
	assert.Nil(t, err)
}

func TestRestfulInvoker_Call(t *testing.T) {
	client, err := sdk.NewClientWithAccessKey("regionid", "accesskeyid", "accesskeysecret")
	assert.Nil(t, err)

	a := &RestfulInvoker{
		BasicInvoker: &BasicInvoker{
			client: client,
			request: requests.NewCommonRequest(),
		},
	}
	_, err = a.Call()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "[SDK.CanNotResolveEndpoint] Can not resolve endpoint")
}
