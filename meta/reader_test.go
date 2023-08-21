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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadJsonFrom(t *testing.T) {
	path := ""
	err := ReadJsonFrom(path, nil)
	assert.NotNil(t, err)
	assert.Equal(t, "read json from  failed", err.Error())

	api := &Api{}
	path = `invalid path`

	err = ReadJsonFrom(path, api)
	assert.NotNil(t, err)
	assert.Equal(t, "read json from invalid path failed", err.Error())

	err = ReadJsonFrom("ecs/DescribeRegions.json", api)
	assert.Equal(t, "DescribeRegions", api.Name)
	assert.Nil(t, err)

	str := ""
	err = ReadJsonFrom("ecs/DescribeRegions.json", &str)
	assert.NotNil(t, err)
	assert.Equal(t, "unmarshal json ecs/DescribeRegions.json failed json: cannot unmarshal object into Go value of type string", err.Error())
}
