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
