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
