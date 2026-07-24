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

func TestLoadRepository(t *testing.T) {
	repository := LoadRepository()
	assert.NotNil(t, repository)
	assert.Contains(t, repository.Names, "Ecs")
}

func TestMockLoadRepository(t *testing.T) {
	products := []Product{
		{
			Code:     "cs",
			ApiNames: []string{"UpdateUserPermissions"},
		},
	}
	repository, _ := MockLoadRepository(products)
	assert.NotNil(t, repository)
	assert.Contains(t, repository.Names, "cs")
}

func TestMockLoadRepositoryWithProductSet(t *testing.T) {
	products := []Product{
		{
			Code:     "cs",
			ApiNames: []string{"UpdateUserPermissions"},
		},
		{
			Code:     "cs",
			ApiNames: []string{"UpdateUserPermissions"},
		},
	}
	repository, err := MockLoadRepository(products)
	assert.Nil(t, repository)
	assert.Equal(t, "Duplicated Name: cs", err.Error())
}

func TestGetStyle(t *testing.T) {
	repository := LoadRepository()
	style, ok := repository.GetStyle("aegis", "2016-11-11")
	assert.True(t, ok)
	assert.Equal(t, "RPC", style)
	_, ok = repository.GetStyle("invalid_product", "2016-11-11")
	assert.False(t, ok)
}
