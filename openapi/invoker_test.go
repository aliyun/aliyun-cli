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

	endpointflag := config.NewEndpointFlag()
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

func TestBasicInvoker_Init_ProfileEndpoint(t *testing.T) {
	cp := &config.Profile{
		Mode:            config.AuthenticateMode("StsToken"),
		AccessKeyId:     "akid",
		AccessKeySecret: "aksecret",
		StsToken:        "ststoken",
		RegionId:        "cn-hangzhou",
		Endpoint:        "custom.endpoint.aliyuncs.com",
	}
	invoker := NewBasicInvoker(cp)

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)

	ctx.Flags().Add(config.NewRegionFlag())
	ctx.Flags().Add(config.NewRegionIdFlag())

	endpointflag := config.NewEndpointFlag()
	ctx.Flags().Add(endpointflag)

	versionflag := NewVersionFlag()
	ctx.Flags().Add(versionflag)

	ctx.Flags().Add(NewHeaderFlag())
	ctx.Flags().Add(config.NewSkipSecureVerify())

	product := &meta.Product{Version: "v1.0"}

	// When profile.Endpoint is set and no cmd --endpoint flag: should use profile endpoint
	err := invoker.Init(ctx, product)
	assert.Nil(t, err)
	assert.Equal(t, "custom.endpoint.aliyuncs.com", invoker.getRequest().Domain)

	// When both profile.Endpoint and cmd --endpoint flag: cmd flag should win
	endpointflag.SetAssigned(true)
	endpointflag.SetValue("cmd.endpoint.aliyuncs.com")
	invoker2 := NewBasicInvoker(cp)
	err = invoker2.Init(ctx, product)
	assert.Nil(t, err)
	assert.Equal(t, "cmd.endpoint.aliyuncs.com", invoker2.getRequest().Domain)
}

func TestParseCustomUserAgentSegments(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect [][2]string
	}{
		{"empty", "", nil},
		{"key_value", "skill/my-skill", [][2]string{{"skill", "my-skill"}}},
		{"plain_token", "plain-token", [][2]string{{"plain-token", ""}}},
		{"multiple", "skill/foo extra/bar", [][2]string{{"skill", "foo"}, {"extra", "bar"}}},
		{"value_with_slash", "key/val/ue", [][2]string{{"key", "val/ue"}}},
		{"spaces_between", "a/1  b/2  c", [][2]string{{"a", "1"}, {"b", "2"}, {"c", ""}}},
		{"whitespace_only", "  \t  ", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCustomUserAgentSegments(tt.input)
			assert.Equal(t, tt.expect, got)
		})
	}
}
