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

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
)

// --- Test O-01 ~ O-03: ShouldUseOpenapiForProfile ---

func TestShouldUseOpenapiForProfile_AnonymousForcesOpenapi(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))

	t.Run("O-01: Anonymous mode forces openapi for non-SLS product", func(t *testing.T) {
		product := &meta.Product{Code: "ECS"}
		profile := &config.Profile{Mode: config.Anonymous}
		assert.True(t, ShouldUseOpenapiForProfile(ctx, product, profile))
	})

	t.Run("O-02: AK mode with non-SLS product does not force openapi", func(t *testing.T) {
		product := &meta.Product{Code: "ECS"}
		profile := &config.Profile{Mode: config.AK}
		assert.False(t, ShouldUseOpenapiForProfile(ctx, product, profile))
	})

	t.Run("O-03: AK mode with SLS product uses openapi", func(t *testing.T) {
		product := &meta.Product{Code: "SLS"}
		profile := &config.Profile{Mode: config.AK}
		assert.True(t, ShouldUseOpenapiForProfile(ctx, product, profile))
	})

	t.Run("nil profile falls back to ShouldUseOpenapi", func(t *testing.T) {
		product := &meta.Product{Code: "ECS"}
		assert.False(t, ShouldUseOpenapiForProfile(ctx, product, nil))
	})

	t.Run("nil profile with SLS still uses openapi", func(t *testing.T) {
		product := &meta.Product{Code: "SLS"}
		assert.True(t, ShouldUseOpenapiForProfile(ctx, product, nil))
	})
}

// --- Test O-04: GetOpenapiClient skips credential for Anonymous ---

func TestGetOpenapiClient_AnonymousSkipsCredential(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	config.AddFlags(ctx.Flags())

	profile := &config.Profile{
		Mode:     config.Anonymous,
		RegionId: "cn-hangzhou",
	}
	product := &meta.Product{Code: "ecs", Version: "2014-05-26"}

	client, err := GetOpenapiClient(profile, ctx, product)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	// Anonymous mode: client is created without credential (credential is nil inside Config)
}

func TestGetOpenapiClient_AnonymousMissingRegion(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	config.AddFlags(ctx.Flags())

	profile := &config.Profile{
		Mode:     config.Anonymous,
		RegionId: "",
	}
	product := &meta.Product{Code: "ecs"}

	_, err := GetOpenapiClient(profile, ctx, product)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RegionId is empty")
}

// --- Test O-05: GetClient guard for Anonymous ---

func TestGetClient_AnonymousReturnsError(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	config.AddFlags(ctx.Flags())

	profile := &config.Profile{
		Mode:     config.Anonymous,
		RegionId: "cn-hangzhou",
	}

	client, err := GetClient(profile, ctx)
	assert.Nil(t, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "anonymous mode")
}

// --- Test E-01 ~ E-04: GetOpenapiClient Endpoint Resolution ---

func TestGetOpenapiClient_EndpointResolution(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	config.AddFlags(ctx.Flags())

	t.Run("E-01: User specified endpoint takes priority", func(t *testing.T) {
		profile := &config.Profile{
			Mode:     config.Anonymous,
			RegionId: "cn-hangzhou",
			Endpoint: "custom.endpoint.com",
		}
		product := &meta.Product{Code: "ecs", RegionalEndpoints: map[string]string{"cn-hangzhou": "ecs-cn-hangzhou.aliyuncs.com"}}
		client, err := GetOpenapiClient(profile, ctx, product)
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "custom.endpoint.com", *client.Endpoint)
	})

	t.Run("E-02: SLS product uses region.log.aliyuncs.com", func(t *testing.T) {
		profile := &config.Profile{
			Mode:     config.Anonymous,
			RegionId: "cn-hangzhou",
		}
		product := &meta.Product{Code: "SLS"}
		client, err := GetOpenapiClient(profile, ctx, product)
		assert.NoError(t, err)
		assert.Equal(t, "cn-hangzhou.log.aliyuncs.com", *client.Endpoint)
	})

	t.Run("E-03: RegionalEndpoints lookup", func(t *testing.T) {
		profile := &config.Profile{
			Mode:     config.Anonymous,
			RegionId: "cn-hangzhou",
		}
		product := &meta.Product{
			Code:              "Ecs",
			RegionalEndpoints: map[string]string{"cn-hangzhou": "ecs-cn-hangzhou.aliyuncs.com"},
		}
		client, err := GetOpenapiClient(profile, ctx, product)
		assert.NoError(t, err)
		assert.Equal(t, "ecs-cn-hangzhou.aliyuncs.com", *client.Endpoint)
	})

	t.Run("E-04: GlobalEndpoint fallback", func(t *testing.T) {
		profile := &config.Profile{
			Mode:     config.Anonymous,
			RegionId: "us-west-1",
		}
		product := &meta.Product{
			Code:              "Ecs",
			RegionalEndpoints: map[string]string{"cn-hangzhou": "ecs-cn-hangzhou.aliyuncs.com"},
			GlobalEndpoint:    "ecs.aliyuncs.com",
		}
		client, err := GetOpenapiClient(profile, ctx, product)
		assert.NoError(t, err)
		assert.Equal(t, "ecs.aliyuncs.com", *client.Endpoint)
	})

	t.Run("E-05: No endpoint configured, client.Endpoint is nil", func(t *testing.T) {
		profile := &config.Profile{
			Mode:     config.Anonymous,
			RegionId: "cn-hangzhou",
		}
		product := &meta.Product{
			Code:              "Unknown",
			RegionalEndpoints: map[string]string{},
		}
		client, err := GetOpenapiClient(profile, ctx, product)
		assert.NoError(t, err)
		// No endpoint resolved, client.Endpoint will be nil
		assert.Nil(t, client.Endpoint)
	})
}

// --- Test RPC-01 ~ RPC-05: createHttpContext RPC branch ---

func TestCreateHttpContext_RPC(t *testing.T) {
	// Set up a Commando with Anonymous profile
	profile := config.Profile{
		Mode:     config.Anonymous,
		RegionId: "cn-hangzhou",
	}
	c := NewCommando(new(bytes.Buffer), profile)

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	config.AddFlags(ctx.Flags())
	AddFlags(ctx.Flags())

	t.Run("RPC-01: RPC Anonymous succeeds in createHttpContext", func(t *testing.T) {
		product := &meta.Product{
			Code:              "Ecs",
			Version:           "2014-05-26",
			ApiStyle:          "rpc",
			RegionalEndpoints: map[string]string{"cn-hangzhou": "ecs-cn-hangzhou.aliyuncs.com"},
		}
		api := &meta.Api{Name: "DescribeInstances", Product: product}
		invoker, err := c.createHttpContext(ctx, product, api, "DescribeInstances", "")
		assert.NoError(t, err)
		assert.NotNil(t, invoker)
	})

	t.Run("RPC-02: RPC non-Anonymous is rejected", func(t *testing.T) {
		c2 := NewCommando(new(bytes.Buffer), config.Profile{Mode: config.AK, RegionId: "cn-hangzhou", AccessKeyId: "ak", AccessKeySecret: "sk"})
		product := &meta.Product{
			Code:     "Ecs",
			Version:  "2014-05-26",
			ApiStyle: "rpc",
		}
		api := &meta.Api{Name: "DescribeInstances", Product: product}
		_, err := c2.createHttpContext(ctx, product, api, "DescribeInstances", "")
		assert.Error(t, err)
	})

	t.Run("RPC-03/04/05: Style=RPC, Method=POST, path=/", func(t *testing.T) {
		product := &meta.Product{
			Code:              "Ecs",
			Version:           "2014-05-26",
			ApiStyle:          "rpc",
			RegionalEndpoints: map[string]string{"cn-hangzhou": "ecs-cn-hangzhou.aliyuncs.com"},
		}
		api := &meta.Api{Name: "DescribeInstances", Product: product}
		invoker, err := c.createHttpContext(ctx, product, api, "DescribeInstances", "")
		assert.NoError(t, err)
		oc, ok := invoker.(*OpenapiContext)
		assert.True(t, ok)
		// RPC-03: Style must be RPC
		assert.Equal(t, "RPC", *oc.openapiParams.Style)
		// RPC-04: Method must be POST (not the API name)
		assert.Equal(t, "POST", oc.method)
		// RPC-05: path must be "/"
		assert.Equal(t, "/", oc.path)
	})
}
