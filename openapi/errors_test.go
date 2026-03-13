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
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
)

func TestInvalidProductError_Error(t *testing.T) {
	err := &InvalidProductError{
		Code: "ecs",
	}
	str := err.Error()
	assert.Equal(t, "'ecs' is not a valid command or product. See `aliyun help`.", str)
}

func TestInvalidProductError_GetSuggestions(t *testing.T) {
	err := &InvalidProductError{
		Code: "ecs",
		library: &Library{
			builtinRepo: &meta.Repository{
				Products: []meta.Product{
					{
						Code: "ecs",
					},
				},
			},
		},
	}
	arrstr := err.GetSuggestions()
	str := strings.Join(arrstr, ",")
	assert.Contains(t, str, "ecs")
}

func TestInvalidApiError_Error(t *testing.T) {
	err := &InvalidApiError{
		Name: "describeregion",
		product: &meta.Product{
			Code: "ecs",
		},
	}
	str := err.Error()
	assert.Equal(t, "'describeregion' is not a valid api. See `aliyun help ecs`.", str)
}

func TestInvalidApiError_GetSuggestions(t *testing.T) {
	err := &InvalidApiError{
		Name: "describeregion",
		product: &meta.Product{
			Code:     "ecs",
			ApiNames: []string{"describeregion"},
		},
	}
	arrstr := err.GetSuggestions()
	str := strings.Join(arrstr, ",")
	assert.Contains(t, str, "describeregion")
}

func TestInvalidParameterError_Error(t *testing.T) {
	err := &InvalidParameterError{
		Name: "ak",
		api: &meta.Api{
			Name: "describeregion",
			Product: &meta.Product{
				Code: "ecs",
			},
		},
	}
	str := err.Error()
	assert.Equal(t, "'--ak' is not a valid parameter or flag. See `aliyun help ecs describeregion`.", str)
}

func TestInvalidParameterError_GetSuggestions(t *testing.T) {
	err := &InvalidParameterError{
		Name: "secure",
		api: &meta.Api{
			Name: "describeregion",
			Parameters: []meta.Parameter{
				{
					Name: "test",
				},
			},
		},
		flags: cli.NewFlagSet(),
	}
	AddFlags(err.flags)
	err.GetSuggestions()
	arrstr := err.GetSuggestions()
	str := strings.Join(arrstr, ",")
	assert.Contains(t, str, "secure")
}

func TestInvalidProductOrPluginError_Error(t *testing.T) {
	err := &InvalidProductOrPluginError{
		Code: "fcc",
	}
	assert.Equal(t, "'fcc' is not a valid product. See `aliyun help`.", err.Error())
}

func TestInvalidProductOrPluginError_GetSuggestions(t *testing.T) {
	t.Run("Has close match", func(t *testing.T) {
		err := &InvalidProductOrPluginError{
			Code: "ec",
			plugins: []plugin.PluginInfo{
				{Name: "aliyun-cli-ecs", ProductCode: "ecs"},
				{Name: "aliyun-cli-fc", ProductCode: "fc"},
			},
		}
		suggestions := err.GetSuggestions()
		str := strings.Join(suggestions, ",")
		assert.Contains(t, str, "ecs")
	})

	t.Run("No match", func(t *testing.T) {
		err := &InvalidProductOrPluginError{
			Code: "zzzzzzz",
			plugins: []plugin.PluginInfo{
				{Name: "aliyun-cli-ecs", ProductCode: "ecs"},
			},
		}
		suggestions := err.GetSuggestions()
		assert.Empty(t, suggestions)
	})

	t.Run("Empty plugins", func(t *testing.T) {
		err := &InvalidProductOrPluginError{
			Code:    "ecs",
			plugins: nil,
		}
		suggestions := err.GetSuggestions()
		assert.Empty(t, suggestions)
	})
}

func TestInvalidUnifiedApiError_Error(t *testing.T) {
	err := &InvalidUnifiedApiError{
		Name: "describreregions",
		product: &meta.Product{
			Code: "ecs",
		},
	}
	assert.Equal(t, "'describreregions' is not a valid api. See `aliyun help ecs`.", err.Error())
}

func TestInvalidUnifiedApiError_GetSuggestions(t *testing.T) {
	t.Run("Combines builtin APIs and plugin commands", func(t *testing.T) {
		err := &InvalidUnifiedApiError{
			Name: "describe-region",
			product: &meta.Product{
				Code:     "ecs",
				ApiNames: []string{"DescribeRegions", "DescribeInstances"},
			},
			lPlugin: plugin.LocalPlugin{
				CmdNames: []string{"describe-regions", "list-instances"},
			},
		}
		suggestions := err.GetSuggestions()
		assert.NotEmpty(t, suggestions)
	})

	t.Run("Deduplicates results", func(t *testing.T) {
		err := &InvalidUnifiedApiError{
			Name: "DescribeRegions",
			product: &meta.Product{
				Code:     "ecs",
				ApiNames: []string{"DescribeRegions"},
			},
			lPlugin: plugin.LocalPlugin{
				CmdNames: []string{"DescribeRegions"},
			},
		}
		suggestions := err.GetSuggestions()
		count := 0
		for _, s := range suggestions {
			if s == "DescribeRegions" {
				count++
			}
		}
		assert.LessOrEqual(t, count, 1)
	})

	t.Run("Empty both", func(t *testing.T) {
		err := &InvalidUnifiedApiError{
			Name: "nonexistent",
			product: &meta.Product{
				Code:     "ecs",
				ApiNames: []string{},
			},
			lPlugin: plugin.LocalPlugin{
				CmdNames: []string{},
			},
		}
		suggestions := err.GetSuggestions()
		assert.Empty(t, suggestions)
	})
}

func TestRemoveDuplicates(t *testing.T) {
	t.Run("No duplicates", func(t *testing.T) {
		result := removeDuplicates([]string{"a", "b", "c"})
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("With duplicates", func(t *testing.T) {
		result := removeDuplicates([]string{"a", "b", "a", "c", "b"})
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("All same", func(t *testing.T) {
		result := removeDuplicates([]string{"x", "x", "x"})
		assert.Equal(t, []string{"x"}, result)
	})

	t.Run("Empty", func(t *testing.T) {
		result := removeDuplicates([]string{})
		assert.Empty(t, result)
	})

	t.Run("Nil", func(t *testing.T) {
		result := removeDuplicates(nil)
		assert.Empty(t, result)
	})
}
