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
	t.Run("default (no hint) keeps legacy single-line wording", func(t *testing.T) {
		err := &InvalidProductOrPluginError{
			Code: "fcc",
		}
		assert.Equal(t, "'fcc' is not a valid built-in product or external product plugin. See `aliyun --help`.", err.Error())
	})

	t.Run("hint is appended on its own line", func(t *testing.T) {
		// Hint exists so callers with extra context (e.g. step-4 of tryDelegatePluginHelp) can explain WHY the user landed on this diagnostic without the explanation leaking into other callers.
		// Default Hint=="" must not change pre-existing output (covered by the subtest above).
		err := &InvalidProductOrPluginError{
			Code: "ecs",
			Hint: "If you meant an OpenAPI built-in call, the form is 'aliyun <product> <APIName>'.",
		}
		assert.Equal(t,
			"'ecs' is not a valid built-in product or external product plugin. See `aliyun --help`.\n"+
				"If you meant an OpenAPI built-in call, the form is 'aliyun <product> <APIName>'.",
			err.Error(),
			"hint must follow the legacy line on its own line — single-line legacy users keep their format")
	})
}

func TestInvalidProductOrPluginError_GetSuggestions(t *testing.T) {
	t.Run("Has close match", func(t *testing.T) {
		err := &InvalidProductOrPluginError{
			Code: "ec",
			plugins: []plugin.PluginInfo{
				{Name: "aliyun-cli-ecs", ProductCode: "ecs"},
				{Name: "aliyun-cli-fc", ProductCode: "fc"},
			},
			library: &Library{
				builtinRepo: &meta.Repository{
					Products: []meta.Product{{Code: "ecs"}},
				},
			},
		}
		suggestions := err.GetSuggestions()
		str := strings.Join(suggestions, ",")
		assert.Contains(t, str, "ecs")
	})

	t.Run("Dedupes plugin and built-in product codes", func(t *testing.T) {
		err := &InvalidProductOrPluginError{
			Code: "ec",
			plugins: []plugin.PluginInfo{
				{Name: "aliyun-cli-ecs", ProductCode: "ecs"},
			},
			library: &Library{
				builtinRepo: &meta.Repository{
					Products: []meta.Product{{Code: "ecs"}},
				},
			},
		}
		suggestions := err.GetSuggestions()
		assert.Equal(t, 1, len(suggestions))
		assert.Equal(t, "ecs", suggestions[0])
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

func TestInvalidApiOrCmdNotFoundError_GetSuggestions(t *testing.T) {
	product := &meta.Product{
		Code:     "ecs",
		ApiNames: []string{"DescribeInstances", "DescribeRegions"},
	}
	lp := &plugin.LocalPlugin{
		CmdNames: []string{"describe-instances", "list-images"},
	}

	t.Run("plugin installed", func(t *testing.T) {
		err := newApiOrCmdNotFoundError(product, "describeinstances", lp, "aliyun-cli-ecs")
		assert.Contains(t, err.Error(), "CamelCase")
		assert.Contains(t, err.Error(), "kebab-case")
		assert.NotContains(t, err.Error(), "aliyun plugin install --name")
		assert.NotContains(t, err.Error(), "must be installed")

		suggestions := err.GetSuggestions()
		joined := strings.Join(suggestions, "\n")
		assert.Contains(t, joined, "DescribeInstances")
		assert.Contains(t, joined, "[built-in OpenAPI]")
	})

	t.Run("plugin not installed", func(t *testing.T) {
		err := newApiOrCmdNotFoundError(product, "describeinstances", nil, "aliyun-cli-ecs")
		assert.Contains(t, err.Error(), "aliyun plugin install --name aliyun-cli-ecs")
		assert.Contains(t, err.Error(), "must be installed")
	})
}

func TestPluginCmdMatches(t *testing.T) {
	lp := &plugin.LocalPlugin{CmdNames: []string{"list-functions"}}
	assert.True(t, pluginCmdMatches("list-functions", lp))
	assert.False(t, pluginCmdMatches("List-Functions", lp))
	assert.False(t, pluginCmdMatches("DescribeRegions", lp))
}

func TestInvalidRestfulPathError(t *testing.T) {
	t.Run("path exists wrong method", func(t *testing.T) {
		repo, err := meta.MockLoadRepository([]meta.Product{{
			Code:     "cs",
			Version:  "2015-12-15",
			ApiNames: []string{"DescribeClusters", "CreateCluster"},
		}})
		assert.NoError(t, err)
		lib := &Library{builtinRepo: repo}
		product, ok := lib.GetProduct("cs")
		assert.True(t, ok)
		matches := lib.FindApisByPath(product.Code, product.Version, "/clusters")

		errObj := newInvalidRestfulPathError(&product, "PUT", "/clusters", matches, "aliyun-cli-cs", nil)
		assert.Contains(t, errObj.Error(), "with method PUT")
		assert.Contains(t, errObj.Error(), "ApiName form")
		assert.Contains(t, errObj.Error(), "METHOD + path")
		assert.Contains(t, errObj.Error(), "aliyun plugin install --name aliyun-cli-cs")

		suggestions := errObj.GetSuggestions()
		joined := strings.Join(suggestions, "\n")
		assert.Contains(t, joined, "aliyun cs GET /clusters [built-in RESTful Style for DescribeClusters]")
		assert.Contains(t, joined, "aliyun cs POST /clusters [built-in RESTful Style for CreateCluster]")
		assert.Contains(t, joined, "[built-in OpenAPI ApiName]")
	})

	t.Run("path exists wrong method with plugin", func(t *testing.T) {
		product := meta.Product{Code: "cs", Version: "2015-12-15"}
		matches := []meta.Api{
			{Name: "DescribeClusters", Method: "GET", PathPattern: "/clusters"},
			{Name: "CreateCluster", Method: "POST", PathPattern: "/clusters"},
		}
		lp := &plugin.LocalPlugin{
			CmdNames: []string{"describe-clusters", "create-cluster", "delete-cluster"},
		}

		errObj := newInvalidRestfulPathError(&product, "PUT", "/clusters", matches, "aliyun-cli-cs", lp)
		suggestions := errObj.GetSuggestions()
		joined := strings.Join(suggestions, "\n")
		assert.Contains(t, joined, "aliyun cs describe-clusters  [product plugin command]")
		assert.Contains(t, joined, "aliyun cs create-cluster  [product plugin command]")
		assert.NotContains(t, joined, "delete-cluster")
	})

	t.Run("path not found", func(t *testing.T) {
		repo, err := meta.MockLoadRepository([]meta.Product{{
			Code:     "cs",
			Version:  "2015-12-15",
			ApiNames: []string{"DescribeClusters"},
		}})
		assert.NoError(t, err)
		lib := &Library{builtinRepo: repo}
		product, ok := lib.GetProduct("cs")
		assert.True(t, ok)

		errObj := newInvalidRestfulPathError(&product, "PUT", "/no-such-path", nil, "", nil)
		assert.Contains(t, errObj.Error(), "can not find api by path /no-such-path")
		assert.Contains(t, errObj.Error(), "Use `aliyun cs --help` to confirm the correct ApiName and METHOD+path for this product.")
		assert.Contains(t, errObj.Error(), "ApiName form")
		assert.Empty(t, errObj.GetSuggestions())
	})
}

func TestRestfulBroadPathError(t *testing.T) {
	product := meta.Product{Code: "sls", Version: "2020-03-20"}
	api := meta.Api{Name: "ListProjects", Method: "GET", PathPattern: "/"}
	lp := &plugin.LocalPlugin{CmdNames: []string{"list-projects"}}

	errObj := newRestfulBroadPathError(&product, "GET", "/", api, "aliyun-cli-sls", lp)
	assert.Contains(t, errObj.Error(), `path "/" is too broad for METHOD+path invocation with GET`)
	assert.Contains(t, errObj.Error(), `Use a specific ApiName instead of the root path "/"`)
	assert.Contains(t, errObj.Error(), "Use `aliyun sls --help` to confirm the correct ApiName for this product.")
	assert.Contains(t, errObj.Error(), "ApiName form")
	assert.NotContains(t, errObj.Error(), "METHOD + path")

	suggestions := errObj.GetSuggestions()
	joined := strings.Join(suggestions, "\n")
	assert.Contains(t, joined, "aliyun sls ListProjects  [built-in OpenAPI ApiName]")
	assert.Contains(t, joined, "aliyun sls list-projects  [product plugin command]")
	assert.NotContains(t, joined, "[built-in RESTful Style")
}

func TestRpcMethodPathError(t *testing.T) {
	product := meta.Product{Code: "ecs", Version: "2014-05-26", ApiStyle: "rpc"}
	errObj := newRpcMethodPathError(&product, "GET", "/instances", "aliyun-cli-ecs", nil)

	assert.Contains(t, errObj.Error(), "'ecs' is an RPC product and does not accept METHOD + path form")
	assert.Contains(t, errObj.Error(), "got GET /instances")
	assert.Contains(t, errObj.Error(), "Use `aliyun ecs <ApiName>` instead")
	assert.Contains(t, errObj.Error(), "aliyun ecs --help")
	assert.Contains(t, errObj.Error(), "CamelCase")
	assert.NotContains(t, errObj.Error(), "REST shortcut")
	assert.Equal(t, []string{"aliyun ecs --help"}, errObj.GetSuggestions())
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
