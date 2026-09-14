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
	"fmt"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
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
	assert.Equal(t, `"ecs" is not a valid command or product. See `+"`aliyun help`"+`.`, str)
	assert.Equal(t, `"EC'S" is not a valid command or product. See `+"`aliyun help`"+`.`, (&InvalidProductError{Code: "EC'S"}).Error())
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
	assert.Equal(t, `"describeregion" is not a valid api. Search matching APIs with `+"`ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true aliyun ecs --help-search describeregion`"+`.`, str)
	assert.Equal(t, `"describe'region" is not a valid api. Search matching APIs with `+"`ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true aliyun ecs --help-search region`"+`.`, (&InvalidApiError{
		Name:    "describe'region",
		product: &meta.Product{Code: "ecs"},
	}).Error())
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
		Name:           "ak",
		ProductCode:    "ecs",
		ApiName:        "describeregion",
		ParameterNames: []string{},
	}
	str := err.Error()
	assert.Equal(t, `"--ak" is not a valid parameter or flag. See `+"`aliyun help ecs describeregion`"+`.`, str)
	assert.Equal(t, `"--a'k" is not a valid parameter or flag. See `+"`aliyun help ecs describeregion`"+`.`, (&InvalidParameterError{
		Name:        "a'k",
		ProductCode: "ecs",
		ApiName:     "describeregion",
	}).Error())
}

func TestInvalidParameterError_GetSuggestions(t *testing.T) {
	flags := cli.NewFlagSet()
	AddFlags(flags)
	err := &InvalidParameterError{
		Name:           "secure",
		ProductCode:    "ecs",
		ApiName:        "describeregion",
		ParameterNames: []string{"test"},
		flags:          flags,
	}
	arrstr := err.GetSuggestions()
	str := strings.Join(arrstr, ",")
	assert.Contains(t, str, "secure")
}

func TestNewInvalidParameterErrorFromCanonical_SuggestionsIncludeNearestParameterExample(t *testing.T) {
	flags := cli.NewFlagSet()
	api := &canonicalmeta.API{
		Name: "DescribeInstances",
		Parameters: []canonicalmeta.Parameter{
			{
				RawName:  "InstanceId",
				Location: "query",
				Example:  "i-bp1234567890",
			},
			{
				RawName:  "ImageId",
				Location: "query",
				Example:  "m-bp1234567890",
			},
		},
	}

	err := NewInvalidParameterErrorFromCanonical("InstancId", api, "ecs", flags)

	assert.Equal(t, []string{"InstanceId (example: i-bp1234567890)"}, err.GetSuggestions())
}

func TestInvalidProductOrPluginError_Error(t *testing.T) {
	t.Run("default (no hint) keeps legacy single-line wording", func(t *testing.T) {
		err := &InvalidProductOrPluginError{
			Code: "fcc",
		}
		assert.Equal(t, `"fcc" is not a valid product. See `+"`aliyun help`"+`.`, err.Error())
	})

	t.Run("hint is appended on its own line", func(t *testing.T) {
		// Hint exists so callers with extra context (e.g. step-4 of tryDelegatePluginHelp) can explain WHY the user landed on this diagnostic without the explanation leaking into other callers.
		// Default Hint=="" must not change pre-existing output (covered by the subtest above).
		err := &InvalidProductOrPluginError{
			Code: "ecs",
			Hint: "If you meant an OpenAPI built-in call, the form is 'aliyun <product> <APIName>'.",
		}
		assert.Equal(t,
			`"ecs" is not a valid product. See `+"`aliyun help`"+`.`+"\n"+
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
	assert.Equal(t, `"describreregions" is not a valid api. Search matching APIs with `+"`ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true aliyun ecs --help-search describreregion`"+`.`, err.Error())
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

func TestInvalidApiError_GetSuggestions_PrefixFallback(t *testing.T) {
	err := &InvalidApiError{
		Name: "Get",
		product: &meta.Product{
			Code:     "sts",
			ApiNames: []string{"AssumeRole", "AssumeRoleWithOIDC", "AssumeRoleWithSAML", "GetCallerIdentity"},
		},
	}
	results := err.GetSuggestions()
	assert.Equal(t, []string{"GetCallerIdentity"}, results)
}

func TestInvalidApiError_GetSuggestions_PrefixFallbackWithOverflow(t *testing.T) {
	apiNames := []string{
		"DescribeA", "DescribeB", "DescribeC", "DescribeD", "DescribeE", "DescribeF", "DescribeG",
	}
	err := &InvalidApiError{
		Name:    "Des",
		product: &meta.Product{Code: "ecs", ApiNames: apiNames},
	}
	results := err.GetSuggestions()
	assert.Equal(t, 6, len(results))
	assert.Equal(t, apiNames[:5], results[:5])
	assert.Equal(t, "... and 2 more, run `aliyun ecs --help-search Des`", results[5])
}

func TestInvalidApiError_GetSuggestions_TypoStillPrefersEditDistance(t *testing.T) {
	err := &InvalidApiError{
		Name: "GetCallerIdentit",
		product: &meta.Product{
			Code:     "sts",
			ApiNames: []string{"AssumeRole", "GetCallerIdentity"},
		},
	}
	results := err.GetSuggestions()
	assert.Equal(t, []string{"GetCallerIdentity"}, results)
}

func TestInvalidApiError_AgentSuggestions_PrefixFallback(t *testing.T) {
	err := &InvalidApiError{
		Name: "Get",
		product: &meta.Product{
			Code:     "sts",
			ApiNames: []string{"AssumeRole", "GetCallerIdentity"},
		},
	}
	assert.Equal(t, []string{"GetCallerIdentity"}, err.AgentSuggestions())
}

func TestInvalidApiError_CouponTokenSuggestionsInEveryMode(t *testing.T) {
	apiNames := []string{"QueryAccountBalance", "QueryAccountDetails", "QueryCashCoupons", "QueryOrders"}
	for _, input := range []string{"DescribeCoupons", "QueryAvailableCoupons", "QueryCouponDetails"} {
		t.Run(input, func(t *testing.T) {
			err := &InvalidApiError{
				Name:    input,
				product: &meta.Product{Code: "bssopenapi", ApiNames: apiNames},
			}
			assert.Equal(t, []string{"QueryCashCoupons"}, err.GetSuggestions())
			assert.Equal(t, []string{"QueryCashCoupons"}, err.AgentSuggestions())
			assert.Contains(t, err.Error(), "`aliyun bssopenapi --help-search Coupon`")
		})
	}
}

func TestInvalidAPIWithoutTokenMatchStillUsesTargetedSearch(t *testing.T) {
	err := &InvalidApiError{
		Name:    "QueryVoucherBalance",
		product: &meta.Product{Code: "bssopenapi", ApiNames: []string{"QueryCashCoupons"}},
	}
	assert.Nil(t, err.GetSuggestions())
	assert.Contains(t, err.Error(), "`aliyun bssopenapi --help-search Voucher`")
}

func TestInvalidBaselineCommandError_CouponTokenSuggestion(t *testing.T) {
	err := &InvalidBaselineCommandError{
		Product:    "bssopenapi",
		Command:    "query-coupon-details",
		Candidates: []string{"query-account-balance", "query-cash-coupons", "query-orders"},
		Err:        assert.AnError,
	}
	assert.Equal(t, []string{"query-cash-coupons"}, err.GetSuggestions())
	assert.Equal(t, []string{"query-cash-coupons"}, err.AgentSuggestions())
	assert.Contains(t, err.Error(), "`ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true aliyun bssopenapi --help-search coupon`")
}

func TestSameStyleCandidates(t *testing.T) {
	candidates := []string{"GetCallerIdentity", "get-caller-identity", "AssumeRole"}

	pascal := sameStyleCandidates("Get", candidates)
	assert.Equal(t, []string{"GetCallerIdentity", "AssumeRole"}, pascal)

	kebab := sameStyleCandidates("get-caller", candidates)
	assert.Equal(t, []string{"get-caller-identity"}, kebab)
}

func TestInvalidBaselineCommandError_Suggestions(t *testing.T) {
	candidates := []string{"describe-instances", "describe-instance-attribute", "run-instances"}

	t.Run("edit distance wins for typos", func(t *testing.T) {
		err := &InvalidBaselineCommandError{
			Product:    "Ecs",
			Command:    "describe-instancez",
			Candidates: candidates,
			Err:        nil,
		}
		results := err.GetSuggestions()
		assert.Contains(t, results, "describe-instances")
		assert.Equal(t, []string{"describe-instances"}, err.AgentSuggestions())
	})

	t.Run("prefix fallback for partial commands", func(t *testing.T) {
		err := &InvalidBaselineCommandError{
			Product:    "Ecs",
			Command:    "describe-i",
			Candidates: candidates,
			Err:        nil,
		}
		results := err.GetSuggestions()
		assert.Contains(t, results, "describe-instances")
		assert.Contains(t, results, "describe-instance-attribute")
		assert.NotContains(t, results, "run-instances")

		agentResults := err.AgentSuggestions()
		assert.Contains(t, agentResults, "describe-instances")
		assert.Contains(t, agentResults, "describe-instance-attribute")
		assert.NotContains(t, agentResults, "run-instances")
	})

	t.Run("prefix fallback overflow hint", func(t *testing.T) {
		many := []string{"describe-a", "describe-b", "describe-c", "describe-d", "describe-e", "describe-f"}
		err := &InvalidBaselineCommandError{
			Product:    "Ecs",
			Command:    "desc",
			Candidates: many,
			Err:        nil,
		}
		results := err.GetSuggestions()
		assert.Equal(t, 6, len(results))
		assert.Equal(t, "... and 1 more, run `ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true aliyun ecs --help-search desc`", results[5])
	})

	t.Run("no candidates yields nothing", func(t *testing.T) {
		err := &InvalidBaselineCommandError{
			Product: "Ecs",
			Command: "describe-instancez",
			Err:     nil,
		}
		assert.Nil(t, err.GetSuggestions())
		assert.Nil(t, err.AgentSuggestions())
	})
}

func newProductSuggestionLibrary(products ...meta.Product) *Library {
	return &Library{builtinRepo: &meta.Repository{Products: products}}
}

func TestInvalidProductError_ErrorPreservesInputCase(t *testing.T) {
	err := &InvalidProductError{Code: "DescribeRegions"}
	assert.Contains(t, err.Error(), `"DescribeRegions" is not a valid command or product`)
	assert.Contains(t, err.AgentMessage(), `"DescribeRegions" is not a valid command or product`)
}

func TestInvalidProductError_GetSuggestions_ProductPrefixParity(t *testing.T) {
	// "openapiex" is a prefix of "openapiexplorer" (edit distance 5): the
	// human path must surface it through the same prefix tier AI mode uses.
	err := &InvalidProductError{
		Code:    "openapiex",
		library: newProductSuggestionLibrary(meta.Product{Code: "ecs"}, meta.Product{Code: "openapiexplorer"}),
	}
	assert.Equal(t, []string{"openapiexplorer"}, err.GetSuggestions())
}

func TestInvalidProductError_SuggestionsFromAPIReverseLookup(t *testing.T) {
	library := newProductSuggestionLibrary(
		meta.Product{Code: "ecs", ApiNames: []string{"DescribeRegions", "DescribeInstances"}},
		meta.Product{Code: "ecd", ApiNames: []string{"DescribeRegions"}},
		meta.Product{Code: "sts", ApiNames: []string{"GetCallerIdentity"}},
		meta.Product{Code: "oss", ApiNames: []string{"PutBucket"}},
	)

	t.Run("pascal case input suggests runnable full commands", func(t *testing.T) {
		err := &InvalidProductError{Code: "DescribeRegions", library: library}
		assert.Equal(t, []string{"aliyun ecd DescribeRegions", "aliyun ecs DescribeRegions"}, err.GetSuggestions())
	})

	t.Run("kebab case input keeps the kebab style", func(t *testing.T) {
		err := &InvalidProductError{Code: "describe-regions", library: library}
		assert.Equal(t, []string{"aliyun ecd describe-regions", "aliyun ecs describe-regions"}, err.GetSuggestions())
	})

	t.Run("camel case api name resolves its owning product", func(t *testing.T) {
		err := &InvalidProductError{Code: "getCallerIdentity", library: library}
		assert.Equal(t, []string{"aliyun sts GetCallerIdentity"}, err.GetSuggestions())
	})

	t.Run("singular api name matches its plural candidate", func(t *testing.T) {
		err := &InvalidProductError{Code: "DescribeRegion", library: library}
		assert.Equal(t, []string{"aliyun ecd DescribeRegions", "aliyun ecs DescribeRegions"}, err.GetSuggestions())
	})

	t.Run("unknown garbage yields nothing", func(t *testing.T) {
		err := &InvalidProductError{Code: "zzzznotexist", library: library}
		assert.Nil(t, err.GetSuggestions())
	})

	t.Run("results are capped at the default suggest limit", func(t *testing.T) {
		products := make([]meta.Product, 0, cli.DefaultSuggestLimit+1)
		for i := 0; i < cli.DefaultSuggestLimit+1; i++ {
			products = append(products, meta.Product{Code: fmt.Sprintf("prod%d", i), ApiNames: []string{"DescribeRegions"}})
		}
		err := &InvalidProductError{Code: "DescribeRegions", library: newProductSuggestionLibrary(products...)}
		assert.Len(t, err.GetSuggestions(), cli.DefaultSuggestLimit)
	})
}

func TestInvalidProductError_HumanAndAISuggestionsMatch(t *testing.T) {
	library := newProductSuggestionLibrary(
		meta.Product{Code: "ecs", ApiNames: []string{"DescribeRegions", "DescribeInstances"}},
		meta.Product{Code: "ahas-openapi", ApiNames: []string{"GetToken"}},
	)
	for _, code := range []string{"openapiex", "Ecsx", "ahas-openap", "DescribeRegions", "describe-regions", "getCallerIdentity"} {
		err := &InvalidProductError{Code: code, library: library}
		assert.Equal(t, err.GetSuggestions(), err.AgentSuggestions(), code)
	}
}

func TestInvalidApiError_OverflowHintOmittedWithoutRecoveryCommand(t *testing.T) {
	// An all-verb input such as "Describe" yields no usable search keyword, so
	// apiRecoveryCommand returns "". The overflow hint must be dropped rather
	// than rendered as "... and N more, run ``".
	err := &InvalidApiError{
		Name: "Describe",
		product: &meta.Product{Code: "ecs", ApiNames: []string{
			"DescribeAccessPoints", "DescribeAccountAttributes", "DescribeActivations",
			"DescribeAddresses", "DescribeAdviserCapacity", "DescribeAggregateCompliancePacks",
		}},
	}
	results := err.GetSuggestions()
	assert.Len(t, results, cli.DefaultSuggestLimit)
	for _, r := range results {
		assert.NotContains(t, r, "run ``")
	}
}
