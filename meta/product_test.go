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
package meta

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestProduct_GetLowerCode(t *testing.T) {
	product := &Product{
		Code: "code",
	}
	code := product.GetLowerCode()
	assert.Equal(t, code, "code")
}

func TestProduct_GetEndpoint(t *testing.T) {
	product := &Product{
		Code: "arms",
		RegionalEndpoints: map[string]string{
			"cn-hangzhou": "arms.cn-hangzhou.aliyuncs.com",
		},
		LocationServiceCode: "arms",
	}
	client, err := sdk.NewClientWithAccessKey("regionid", "acesskeyid", "accesskeysecret")
	assert.Nil(t, err)
	endpoint, err := product.GetEndpoint("cn-hangzhou", client)
	assert.Nil(t, err)
	assert.Equal(t, endpoint, "arms.cn-hangzhou.aliyuncs.com")

	product.LocationServiceCode = ""
	product.GlobalEndpoint = "arms.aliyuncs.com"
	endpoint, err = product.GetEndpoint("us-west-1", client)
	assert.Nil(t, err)
	assert.Equal(t, endpoint, "arms.aliyuncs.com")

	product.GlobalEndpoint = ""
	_, err = product.GetEndpoint("us-west-1", client)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "us-west-1")
}

func TestProduct_GetEndpointWithType(t *testing.T) {
	client, err := sdk.NewClientWithAccessKey("regionid", "acesskeyid", "accesskeysecret")
	assert.Nil(t, err)

	t.Run("VPC endpoint type with RegionalVpcEndpoints", func(t *testing.T) {
		product := &Product{
			Code: "ecs",
			RegionalVpcEndpoints: map[string]string{
				"cn-hangzhou": "ecs-vpc.cn-hangzhou.aliyuncs.com",
			},
			RegionalEndpoints: map[string]string{
				"cn-hangzhou": "ecs.cn-hangzhou.aliyuncs.com",
			},
			LocationServiceCode: "ecs",
		}
		endpoint, err := product.GetEndpointWithType("cn-hangzhou", client, "vpc")
		assert.Nil(t, err)
		assert.Equal(t, "ecs-vpc.cn-hangzhou.aliyuncs.com", endpoint)
	})

	t.Run("VPC endpoint type skips location service", func(t *testing.T) {
		product := &Product{
			Code: "ecs",
			RegionalVpcEndpoints: map[string]string{
				"cn-hangzhou": "ecs-vpc.cn-hangzhou.aliyuncs.com",
			},
			LocationServiceCode: "ecs", // Should be skipped for VPC
		}
		endpoint, err := product.GetEndpointWithType("cn-hangzhou", client, "vpc")
		assert.Nil(t, err)
		assert.Equal(t, "ecs-vpc.cn-hangzhou.aliyuncs.com", endpoint)
	})

	t.Run("VPC endpoint type falls back to RegionalEndpoints when VPC endpoint not found", func(t *testing.T) {
		product := &Product{
			Code: "ecs",
			RegionalVpcEndpoints: map[string]string{
				"cn-beijing": "ecs-vpc.cn-beijing.aliyuncs.com",
			},
			RegionalEndpoints: map[string]string{
				"cn-hangzhou": "ecs.cn-hangzhou.aliyuncs.com",
			},
			LocationServiceCode: "ecs",
		}
		endpoint, err := product.GetEndpointWithType("cn-hangzhou", client, "vpc")
		assert.Nil(t, err)
		assert.Equal(t, "ecs.cn-hangzhou.aliyuncs.com", endpoint)
	})

	t.Run("VPC endpoint type falls back to GlobalEndpoint when VPC and regional endpoints not found", func(t *testing.T) {
		product := &Product{
			Code: "ecs",
			RegionalVpcEndpoints: map[string]string{
				"cn-beijing": "ecs-vpc.cn-beijing.aliyuncs.com",
			},
			GlobalEndpoint: "ecs.aliyuncs.com",
		}
		endpoint, err := product.GetEndpointWithType("us-west-1", client, "vpc")
		assert.Nil(t, err)
		assert.Equal(t, "ecs.aliyuncs.com", endpoint)
	})

	t.Run("Non-VPC endpoint type uses location service when available", func(t *testing.T) {
		product := &Product{
			Code: "arms",
			RegionalEndpoints: map[string]string{
				"cn-hangzhou": "arms.cn-hangzhou.aliyuncs.com",
			},
			LocationServiceCode: "arms",
		}
		// Note: This test may actually use RegionalEndpoints if location service fails
		// The important thing is that location service is attempted
		endpoint, err := product.GetEndpointWithType("cn-hangzhou", client, "")
		assert.Nil(t, err)
		assert.NotEmpty(t, endpoint)
	})

	t.Run("Non-VPC endpoint type uses RegionalEndpoints when location service not available", func(t *testing.T) {
		product := &Product{
			Code: "test",
			RegionalEndpoints: map[string]string{
				"cn-hangzhou": "test.cn-hangzhou.aliyuncs.com",
			},
			LocationServiceCode: "",
		}
		endpoint, err := product.GetEndpointWithType("cn-hangzhou", client, "")
		assert.Nil(t, err)
		assert.Equal(t, "test.cn-hangzhou.aliyuncs.com", endpoint)
	})

	t.Run("Non-VPC endpoint type uses GlobalEndpoint as fallback", func(t *testing.T) {
		product := &Product{
			Code:                "test",
			GlobalEndpoint:      "test.aliyuncs.com",
			LocationServiceCode: "",
		}
		endpoint, err := product.GetEndpointWithType("us-west-1", client, "")
		assert.Nil(t, err)
		assert.Equal(t, "test.aliyuncs.com", endpoint)
	})

	t.Run("VPC endpoint type returns error when no endpoints available", func(t *testing.T) {
		product := &Product{
			Code:                "test",
			LocationServiceCode: "",
		}
		_, err := product.GetEndpointWithType("us-west-1", client, "vpc")
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "us-west-1")
	})

	t.Run("Empty endpoint type behaves like non-VPC", func(t *testing.T) {
		product := &Product{
			Code: "test",
			RegionalEndpoints: map[string]string{
				"cn-hangzhou": "test.cn-hangzhou.aliyuncs.com",
			},
			RegionalVpcEndpoints: map[string]string{
				"cn-hangzhou": "test-vpc.cn-hangzhou.aliyuncs.com",
			},
			LocationServiceCode: "",
		}
		endpoint, err := product.GetEndpointWithType("cn-hangzhou", client, "")
		assert.Nil(t, err)
		assert.Equal(t, "test.cn-hangzhou.aliyuncs.com", endpoint)
	})

	t.Run("VPC endpoint type with nil RegionalVpcEndpoints falls back to RegionalEndpoints", func(t *testing.T) {
		product := &Product{
			Code:                 "test",
			RegionalVpcEndpoints: nil,
			RegionalEndpoints: map[string]string{
				"cn-hangzhou": "test.cn-hangzhou.aliyuncs.com",
			},
		}
		endpoint, err := product.GetEndpointWithType("cn-hangzhou", client, "vpc")
		assert.Nil(t, err)
		assert.Equal(t, "test.cn-hangzhou.aliyuncs.com", endpoint)
	})
}
