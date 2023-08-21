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
	"github.com/stretchr/testify/assert"

	"testing"
)

var RepositoryTest = &Repository{}

func TestLoadRepository(t *testing.T) {
	repository := LoadRepository()
	assert.NotNil(t, repository)
	assert.Contains(t, repository.Names, "Ecs")
}

func TestGetApi(t *testing.T) {
	repository := LoadRepository()
	assert.NotNil(t, repository)
	_, ok := repository.GetApi("invalid_product", "", "")
	assert.False(t, ok)

	_, ok = repository.GetApi("ros", "", "")
	assert.False(t, ok)

	api, ok := repository.GetApi("Ecs", "2014-05-26", "DescribeRegions")
	assert.True(t, ok)
	assert.NotNil(t, api)
	assert.Equal(t, "DescribeRegions", api.Name)
}

func TestGetStyle(t *testing.T) {
	repository := LoadRepository()
	style, ok := repository.GetStyle("aegis", "2016-11-11")
	assert.True(t, ok)
	assert.Equal(t, "RPC", style)
	_, ok = repository.GetStyle("invalid_product", "2016-11-11")
	assert.False(t, ok)
}
