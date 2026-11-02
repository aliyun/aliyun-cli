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
	"os"
	"testing"

	slsgateway "github.com/alibabacloud-go/alibabacloud-gateway-sls/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOpenapiClientEnablesSignatureV4(t *testing.T) {
	require.NoError(t, os.Unsetenv(config.EnvSignatureVersion))
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	profile := &config.Profile{
		Mode:             config.AK,
		AccessKeyId:      "id",
		AccessKeySecret:  "secret",
		RegionId:         "cn-hangzhou",
		SignatureVersion: config.SignatureVersionV4,
	}
	client, err := GetOpenapiClient(profile, ctx, &meta.Product{Code: "Ecs", GlobalEndpoint: "ecs.cn-hangzhou.aliyuncs.com"})
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, config.SignatureVersionV4, tea.StringValue(client.SignatureVersion))
	assert.Equal(t, "Ecs", tea.StringValue(client.ProductId))
	assert.NotNil(t, client.Spi)
}

func TestGetOpenapiClientSignatureV4FromEnv(t *testing.T) {
	t.Setenv(config.EnvSignatureVersion, "v4")
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	profile := &config.Profile{
		Mode:            config.AK,
		AccessKeyId:     "id",
		AccessKeySecret: "secret",
		RegionId:        "cn-hangzhou",
	}
	client, err := GetOpenapiClient(profile, ctx, &meta.Product{Code: "Ecs", GlobalEndpoint: "ecs.cn-hangzhou.aliyuncs.com"})
	require.NoError(t, err)
	assert.Equal(t, config.SignatureVersionV4, tea.StringValue(client.SignatureVersion))
	assert.NotNil(t, client.Spi)
}

func TestGetOpenapiClientSignatureV4FromCliSignVersionFlag(t *testing.T) {
	require.NoError(t, os.Unsetenv(config.EnvSignatureVersion))
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	f := config.CliSignVersionFlag(ctx.Flags())
	require.NotNil(t, f)
	f.SetAssigned(true)
	f.SetValue("v4")
	profile := &config.Profile{
		Mode:            config.AK,
		AccessKeyId:     "id",
		AccessKeySecret: "secret",
		RegionId:        "cn-hangzhou",
	}
	client, err := GetOpenapiClient(profile, ctx, &meta.Product{Code: "Ecs", GlobalEndpoint: "ecs.cn-hangzhou.aliyuncs.com"})
	require.NoError(t, err)
	assert.Equal(t, config.SignatureVersionV4, tea.StringValue(client.SignatureVersion))
	assert.Equal(t, "Ecs", tea.StringValue(client.ProductId))
	assert.NotNil(t, client.Spi)
}

func TestProcessInvokeRejectsSignatureV4ForPascalCase(t *testing.T) {
	require.NoError(t, os.Unsetenv(config.EnvSignatureVersion))
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	config.AddFlags(ctx.Flags())
	c := &Commando{profile: config.Profile{
		Mode:             config.AK,
		AccessKeyId:      "id",
		AccessKeySecret:  "secret",
		RegionId:         "cn-hangzhou",
		SignatureVersion: config.SignatureVersionV4,
	}}
	err := c.processInvoke(ctx, "ecs", "DescribeInstances", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature version v4 is not supported for PascalCase")
	assert.NotContains(t, err.Error(), "init client")
}

func TestProcessInvokeRejectsSignatureV4FromCliSignVersionFlag(t *testing.T) {
	require.NoError(t, os.Unsetenv(config.EnvSignatureVersion))
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	config.AddFlags(ctx.Flags())
	f := config.CliSignVersionFlag(ctx.Flags())
	require.NotNil(t, f)
	f.SetAssigned(true)
	f.SetValue("v4")
	c := &Commando{profile: config.Profile{
		Mode:            config.AK,
		AccessKeyId:     "id",
		AccessKeySecret: "secret",
		RegionId:        "cn-hangzhou",
	}}
	err := c.processInvoke(ctx, "ecs", "DescribeInstances", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature version v4 is not supported for PascalCase")
}

func TestGetOpenapiClientSlsUsesSignatureV4(t *testing.T) {
	require.NoError(t, os.Unsetenv(config.EnvSignatureVersion))
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	profile := &config.Profile{
		Mode:             config.AK,
		AccessKeyId:      "id",
		AccessKeySecret:  "secret",
		RegionId:         "cn-hangzhou",
		SignatureVersion: config.SignatureVersionV4,
	}
	client, err := GetOpenapiClient(profile, ctx, &meta.Product{Code: "Sls"})
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, config.SignatureVersionV4, tea.StringValue(client.SignatureVersion))
	assert.Equal(t, "cn-hangzhou.log.aliyuncs.com", tea.StringValue(client.Endpoint))
	// SLS keeps its own gateway SPI; V4 is selected via SignatureVersion only.
	_, ok := client.Spi.(*slsgateway.Client)
	assert.True(t, ok, "SLS must keep slsgateway.Client under signature V4")
	assert.Empty(t, tea.StringValue(client.ProductId))
}

func TestGetOpenapiClientEmptySignatureUsesDefault(t *testing.T) {
	t.Setenv(config.EnvSignatureVersion, "")
	require.NoError(t, os.Unsetenv(config.EnvSignatureVersion))
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	profile := &config.Profile{
		Mode:            config.AK,
		AccessKeyId:     "id",
		AccessKeySecret: "secret",
		RegionId:        "cn-hangzhou",
	}
	client, err := GetOpenapiClient(profile, ctx, &meta.Product{Code: "Ecs", GlobalEndpoint: "ecs.cn-hangzhou.aliyuncs.com"})
	require.NoError(t, err)
	assert.Empty(t, tea.StringValue(client.SignatureVersion))
	assert.Empty(t, tea.StringValue(client.SignatureAlgorithm))
	assert.Nil(t, client.Spi)
}

func TestGetOpenapiClientEmptyFlagClearsProfileV4(t *testing.T) {
	require.NoError(t, os.Unsetenv(config.EnvSignatureVersion))
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	f := config.CliSignVersionFlag(ctx.Flags())
	require.NotNil(t, f)
	f.SetAssigned(true)
	f.SetValue("")
	profile := &config.Profile{
		Mode:             config.AK,
		AccessKeyId:      "id",
		AccessKeySecret:  "secret",
		RegionId:         "cn-hangzhou",
		SignatureVersion: config.SignatureVersionV4,
	}
	client, err := GetOpenapiClient(profile, ctx, &meta.Product{Code: "Ecs", GlobalEndpoint: "ecs.cn-hangzhou.aliyuncs.com"})
	require.NoError(t, err)
	assert.Empty(t, tea.StringValue(client.SignatureVersion))
	assert.Nil(t, client.Spi)
}

func TestGetOpenapiClientSignatureVersionUnsupported(t *testing.T) {
	require.NoError(t, os.Unsetenv(config.EnvSignatureVersion))
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	profile := &config.Profile{
		Mode:             config.AK,
		AccessKeyId:      "id",
		AccessKeySecret:  "secret",
		RegionId:         "cn-hangzhou",
		SignatureVersion: "v3",
	}
	_, err := GetOpenapiClient(profile, ctx, &meta.Product{Code: "Ecs", GlobalEndpoint: "ecs.cn-hangzhou.aliyuncs.com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")
}

func TestGetOpenapiClientSignatureV4EmptyProductCodeFallsBackToLower(t *testing.T) {
	t.Setenv(config.EnvSignatureVersion, "v4")
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	profile := &config.Profile{
		Mode:            config.AK,
		AccessKeyId:     "id",
		AccessKeySecret: "secret",
		RegionId:        "cn-hangzhou",
		Endpoint:        "ecs.cn-hangzhou.aliyuncs.com",
	}
	client, err := GetOpenapiClient(profile, ctx, &meta.Product{Code: ""})
	require.NoError(t, err)
	assert.Equal(t, config.SignatureVersionV4, tea.StringValue(client.SignatureVersion))
	assert.Equal(t, "", tea.StringValue(client.ProductId))
	assert.NotNil(t, client.Spi)
}

func TestProcessInvokeRejectsUnsupportedSignatureVersion(t *testing.T) {
	require.NoError(t, os.Unsetenv(config.EnvSignatureVersion))
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	config.AddFlags(ctx.Flags())
	c := &Commando{profile: config.Profile{
		Mode:             config.AK,
		AccessKeyId:      "id",
		AccessKeySecret:  "secret",
		RegionId:         "cn-hangzhou",
		SignatureVersion: "v3",
	}}
	err := c.processInvoke(ctx, "ecs", "DescribeInstances", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")
}
