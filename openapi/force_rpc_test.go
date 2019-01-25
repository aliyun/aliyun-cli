package openapi

import (
	"bufio"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestForceRpcInvoker_Prepare(t *testing.T) {
	a := &ForceRpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
		method: "DescribeRegion",
	}
	a.BasicInvoker.request.QueryParams = make(map[string]string)
	w := new(bufio.Writer)
	ctx := cli.NewCommandContext(w)
	cmd := config.NewConfigureCommand()
	cmd.EnableUnknownFlag = true
	ctx.EnterCommand(cmd)

	secureflag := NewSecureFlag()
	secureflag.SetAssigned(true)
	ctx.Flags().Add(secureflag)
	ctx.UnknownFlags().Add(NewSecureFlag())
	err := a.Prepare(ctx)
	assert.Nil(t, err)
}

func TestForceRpcInvoker_Call(t *testing.T) {
	a := &ForceRpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
		method: "DescribeRegion",
	}
	client, err := sdk.NewClientWithAccessKey("regionid", "acesskeyid", "accesskeysecret")
	assert.Nil(t, err)
	a.client = client
	_, err = a.Call()
	assert.NotNil(t, err)
}
