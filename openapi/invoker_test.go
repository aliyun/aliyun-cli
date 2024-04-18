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

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/meta"
)

func newProfile() *config.Profile {
	return &config.Profile{
		Name:            "default",
		Mode:            "",
		OutputFormat:    "json",
		Language:        "en",
		AccessKeyId:     "",
		AccessKeySecret: "",
		StsToken:        "",
		RamRoleName:     "",

		RamRoleArn:      "",
		RoleSessionName: "",
		PrivateKey:      "",
		KeyPairName:     "",
		ExpiredSeconds:  0,
		Verified:        "",
		RegionId:        "",
		Site:            "",
		ReadTimeout:     0,
		ConnectTimeout:  0,
		RetryCount:      0,
	}
}

// func TestGetClient(t *testing.T) {
// 	actual := newProfile()
// 	buf := new(bytes.Buffer)
// 	buf2 := new(bytes.Buffer)
// 	ctx := cli.NewCommandContext(buf, buf2)
// 	AddFlags(ctx.Flags())
// 	actual.RetryCount = 2

// 	actual.Mode = config.AK
// 	client, err := GetClient(actual, ctx)
// 	assert.Nil(t, client)
// 	assert.NotNil(t, err)

// 	actual.Mode = config.RamRoleArnWithEcs
// 	client, err = GetClient(actual, ctx)
// 	assert.Nil(t, client)
// 	assert.NotNil(t, err)

// 	actual.Mode = config.StsToken
// 	client, err = GetClient(actual, ctx)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, client)

// 	actual.Mode = config.RamRoleArn
// 	client, err = GetClient(actual, ctx)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, client)

// 	actual.Mode = config.EcsRamRole
// 	client, err = GetClient(actual, ctx)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, client)

// 	actual.Mode = config.RsaKeyPair
// 	client, err = GetClient(actual, ctx)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, client)

// 	// config to client
// 	actual.Mode = config.StsToken
// 	actual.ReadTimeout = 2
// 	actual.ConnectTimeout = 2
// 	config.SkipSecureVerify(ctx.Flags()).SetAssigned(true)
// 	client, err = GetClient(actual, ctx)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, client)
// 	assert.Equal(t, 2, client.GetConfig().MaxRetryTime)
// 	assert.Equal(t, float64(2), client.GetReadTimeout().Seconds())
// 	assert.Equal(t, float64(2), client.GetConnectTimeout().Seconds())
// 	assert.True(t, client.GetHTTPSInsecure())
// }

func TestGetClientByAK(t *testing.T) {
	actual := newProfile()
	config := sdk.NewConfig()

	actual.AccessKeyId = "accessKeyId"
	client, err := GetClientByAK(actual, config)
	assert.Nil(t, client)
	assert.EqualError(t, err, "AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first")

	actual.AccessKeySecret = "accessKeySecret"
	client, err = GetClientByAK(actual, config)
	assert.Nil(t, client)
	assert.EqualError(t, err, "default RegionId is empty! run `aliyun configure` first")

	actual.RegionId = "cn-hangzhou"
	client, err = GetClientByAK(actual, config)
	assert.Nil(t, err)
	assert.NotNil(t, client)
}

func TestGetClientWithNoError(t *testing.T) {
	actual := newProfile()
	config := sdk.NewConfig()

	// GetClientBySts
	client, err := GetClientBySts(actual, config)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	// GetClientByRoleArn
	client, err = GetClientByRoleArn(actual, config)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	// GetClientByEcsRamRole
	client, err = GetClientByEcsRamRole(actual, config)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	// GetClientByPrivateKey
	client, err = GetClientByPrivateKey(actual, config)
	assert.Nil(t, err)
	assert.NotNil(t, client)
}

func TestGetClientByRamRoleArnWithEcs(t *testing.T) {
	actual := newProfile()
	config := sdk.NewConfig()
	client, err := GetClientByRamRoleArnWithEcs(actual, config)
	assert.Nil(t, client)
	assert.NotNil(t, err)
}

func TestBasicInvoker_Init(t *testing.T) {
	cp := &config.Profile{
		Mode: config.AuthenticateMode("AK"),
	}
	invooker := NewBasicInvoker(cp)
	client := invooker.getClient()
	assert.Nil(t, client)

	req := invooker.getRequest()
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
	invooker.profile.Mode = config.AuthenticateMode("DEFAULT")
	err := invooker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "init client failed unexcepted certificate mode: DEFAULT", err.Error())

	invooker.profile.Mode = config.AuthenticateMode("StsToken")
	err = invooker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "invaild flag --header `testfail` use `--header HeaderName=Value`", err.Error())

	regionflag.SetAssigned(false)
	endpointflag.SetAssigned(false)
	versionflag.SetAssigned(false)
	headerflag.SetValues([]string{})
	err = invooker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "missing version for product ", err.Error())

	invooker.profile.Mode = config.AuthenticateMode("StsToken")
	product.Version = "v1.0"
	err = invooker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "missing region for product ", err.Error())

	invooker.profile.RegionId = "cn-hangzhou"
	err = invooker.Init(ctx, product)
	assert.NotNil(t, err)
	assert.Equal(t, "unknown endpoint for /cn-hangzhou! failed unknown endpoint for region cn-hangzhou\n  you need to add --endpoint xxx.aliyuncs.com", err.Error())

	endpointflag.SetAssigned(true)
	err = invooker.Init(ctx, product)
	assert.Nil(t, err)
}
