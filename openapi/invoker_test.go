package openapi

import (
	"bufio"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBasicInvoker_Init(t *testing.T) {
	cp := &config.Profile{
		Mode: config.AuthenticateMode("AK"),
	}
	invooker := NewBasicInvoker(cp)
	client := invooker.getClient()
	assert.Nil(t,client)

	req := invooker.getRequest()
	assert.Nil(t,req)

	w := new(bufio.Writer)
	ctx := cli.NewCommandContext(w)
	config.AddFlags(ctx.Flags())
	AddFlags(ctx.Flags())
	product := &meta.Product{}
	invooker.profile.Mode = config.AuthenticateMode("DEFAULT")
	err := invooker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "init client failed unexcepted certificate mode: DEFAULT", err.Error())

	invooker.profile.Mode = config.AuthenticateMode("StsToken")
	err = invooker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "missing version for product ", err.Error())

	invooker.profile.Mode = config.AuthenticateMode("StsToken")
	product.Version = "v1.0"
	err = invooker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "missing region for product ", err.Error())

	invooker.profile.Mode = config.AuthenticateMode("StsToken")
	invooker.profile.RegionId = "cn-hangzhou"
	err = invooker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "unknown endpoint for /cn-hangzhou! failed unknown endpoint for region cn-hangzhou\n  you need to add --endpoint xxx.aliyuncs.com", err.Error())
}
