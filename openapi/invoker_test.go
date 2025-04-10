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
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

func TestBasicInvoker_Init(t *testing.T) {
	cp := &config.Profile{
		Mode:            config.AuthenticateMode("AK"),
		AccessKeyId:     "akid",
		AccessKeySecret: "aksecret",
	}
	invoker := NewBasicInvoker(cp)
	client := invoker.getClient()
	assert.Nil(t, client)

	req := invoker.getRequest()
	assert.Nil(t, req)

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)

	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)
	ctx.Flags().Add(config.NewRegionIdFlag())

	endpointflag := NewEndpointFlag()
	endpointflag.SetAssigned(true)
	endpointflag.SetValue("ecs.cn-hangzhou.aliyuncs")
	ctx.Flags().Add(endpointflag)

	versionflag := NewVersionFlag()
	versionflag.SetAssigned(true)
	versionflag.SetValue("v1.0")
	ctx.Flags().Add(versionflag)

	headerflag := NewHeaderFlag()
	headerflag.SetValues([]string{"Accept=xml", "Accept=json", "Content-Type=json", "testfail"})
	ctx.Flags().Add(headerflag)

	ctx.Flags().Add(config.NewSkipSecureVerify())

	product := &meta.Product{}

	invoker.profile.Mode = config.AuthenticateMode("StsToken")
	err := invoker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "invaild flag --header `testfail` use `--header HeaderName=Value`", err.Error())

	regionflag.SetAssigned(false)
	endpointflag.SetAssigned(false)
	versionflag.SetAssigned(false)
	headerflag.SetValues([]string{})
	err = invoker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "missing version for product ", err.Error())

	invoker.profile.Mode = config.AuthenticateMode("StsToken")
	invoker.profile.StsToken = "ststoken"
	product.Version = "v1.0"
	err = invoker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "missing region for product ", err.Error())

	invoker.profile.RegionId = "cn-hangzhou"
	err = invoker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "unknown endpoint for /cn-hangzhou! failed unknown endpoint for region cn-hangzhou\n  you need to add --endpoint xxx.aliyuncs.com", err.Error())

	endpointflag.SetAssigned(true)
	err = invoker.Init(ctx, product)
	assert.Nil(t, err)
}
