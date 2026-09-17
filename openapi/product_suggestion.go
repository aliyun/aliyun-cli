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

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

// productSuggestions is the single suggestion pipeline for unknown products,
// shared by human and AI rendering so both output modes report the same
// candidates. It matches product codes first (close typo, then prefix) and
// falls back to reverse-looking the input up as an API name, because a
// PascalCase or kebab token at the root position is usually an API whose
// product prefix was omitted.
func productSuggestions(input string, library *Library) []string {
	if library == nil {
		return nil
	}
	products := library.GetProducts()
	candidates := make([]string, 0, len(products))
	for _, product := range products {
		candidates = append(candidates, strings.ToLower(product.Code))
	}
	// Product-code tiers keep the historical lowercased input: product codes
	// carry no case style, and lowercasing preserves today's AI behavior for
	// mixed-case typos such as "Openapiex".
	if suggestions := apiSuggestions(strings.ToLower(input), candidates); len(suggestions) > 0 {
		return suggestions
	}
	return apiNameProductSuggestions(input, products)
}

// apiNameProductSuggestions reverse-looks the failed token up as an API name
// across every product's API list, in the input's own command style, and
// returns runnable full commands such as "aliyun ecs DescribeRegions".
// Matching compares compact forms (case and separator insensitive) in three
// precision passes — compact-equal, prefix, then edit distance — short
// circuiting on the first pass with results. The list is capped at
// DefaultSuggestLimit without an overflow hint, mirroring the product-code
// tiers.
func apiNameProductSuggestions(input string, products []meta.Product) []string {
	if len(products) == 0 || strings.TrimSpace(input) == "" {
		return nil
	}
	kebab := commandStyle(input) == "kebab"
	commands := make([]string, 0, 1024)
	compacts := make([]string, 0, 1024)
	for _, product := range products {
		code := strings.ToLower(product.Code)
		for _, apiName := range product.ApiNames {
			name := apiName
			if kebab {
				name = apiNameToKebab(apiName)
			}
			compact := compactAPITokens(name)
			if compact == "" {
				continue
			}
			commands = append(commands, "aliyun "+code+" "+name)
			compacts = append(compacts, compact)
		}
	}
	needle := compactAPITokens(input)
	if needle == "" || len(commands) == 0 {
		return nil
	}
	for _, match := range []func(candidate, needle string) bool{
		func(candidate, needle string) bool { return candidate == needle },
		func(candidate, needle string) bool {
			return len(needle) >= 2 && strings.HasPrefix(candidate, needle)
		},
		func(candidate, needle string) bool {
			return cli.CalculateStringDistance(needle, candidate) <= cli.DefaultSuggestDistance
		},
	} {
		var results []string
		for index, compact := range compacts {
			if match(compact, needle) {
				results = append(results, commands[index])
			}
		}
		if len(results) > 0 {
			return truncateSuggestions(stableStrings(results))
		}
	}
	return nil
}

func truncateSuggestions(values []string) []string {
	if len(values) > cli.DefaultSuggestLimit {
		return values[:cli.DefaultSuggestLimit]
	}
	return values
}
