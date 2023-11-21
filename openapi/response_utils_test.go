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
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestDetectArrayPath(t *testing.T) {
	str := detectArrayPath("test")
	assert.Equal(t, "", str)

	array := map[string]interface{}{
		"test": map[string]interface{}{
			"utils": []interface{}{
				"inter",
			},
		},
	}
	str = detectArrayPath(array)
	assert.Equal(t, "test.utils[]", str)
}

func TestEvaluateExpr(t *testing.T) {
	body := []byte("test")
	str, err := evaluateExpr(body, "")
	assert.NotNil(t, err)
	assert.Equal(t, "unmarshal failed invalid character 'e' in literal true (expecting 'r')", err.Error())
	assert.Equal(t, "", str)

	body = []byte(`"test"`)
	str, err = evaluateExpr(body, "")
	assert.NotNil(t, err)
	assert.Equal(t, "jmes search failed SyntaxError: Incomplete expression", err.Error())
	assert.Equal(t, "", str)

	body = []byte(`{
           "PageNumber": 5,
	       "TotalCount": 37,
	       "PageSize": 5,
	       "RegionId": "cn-beijing",
	       "Images": {
				"Image": [{
					"ImageId": "win2016_64_dtc_1607_en-us_40G_alibase_20170915.vhd"
		         }]
           }
    }`)
	str, err = evaluateExpr(body, "RegionId")
	assert.Nil(t, err)
	assert.Equal(t, "cn-beijing", str)

	str, err = evaluateExpr(body, "Images.Image[]")
	assert.NotNil(t, err)
	assert.Equal(t, "object [map[ImageId:win2016_64_dtc_1607_en-us_40G_alibase_20170915.vhd]] isn't string", err.Error())
	assert.Equal(t, "", str)
}
