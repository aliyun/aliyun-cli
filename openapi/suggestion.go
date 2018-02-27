package openapi

import (
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/aliyun/aliyun-cli/cli"
	"strings"
)

const SuggestDistance = 2

func GetProductSuggestions(library *meta.Library, code string) []string {
	result := make([]string, 0)
	for _, p := range library.Products {
		dist := cli.CalculateStringDistance(strings.ToLower(code), strings.ToLower(p.Code))
		if dist <= SuggestDistance {
			result = append(result, strings.ToLower(p.Code))
		}
	}
	return result
}


