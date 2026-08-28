package canonicalmeta

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPITitleContractFallsBackWithoutInventingMetadata(t *testing.T) {
	var api API
	require.NoError(t, json.Unmarshal([]byte(`{
		"name":"DescribeThings",
		"title_en":"Describe things",
		"title_zh":"查询对象",
		"description_en":"Returns the complete thing collection.",
		"description_zh":"返回完整对象集合。"
	}`), &api))

	assert.Equal(t, "Describe things", api.Title("en"))
	assert.Equal(t, "查询对象", api.Title("zh"))
	assert.Equal(t, "Describe things", api.TitleOrDescription("en"))

	withoutTitle := API{DescriptionEn: "Description only", DescriptionZh: "只有描述"}
	assert.Empty(t, withoutTitle.Title("en"))
	assert.Equal(t, "Description only", withoutTitle.TitleOrDescription("en"))
	assert.Equal(t, "只有描述", withoutTitle.TitleOrDescription("zh"))
}

func TestVersionAPIEntryRetainsOptionalTitleAndDescriptionFallback(t *testing.T) {
	entry := VersionAPIEntry{TitleEn: "Short title", DescriptionEn: "Long description"}
	assert.Equal(t, "Short title", entry.TitleOrDescription("en"))

	entry.TitleEn = ""
	assert.Equal(t, "Long description", entry.TitleOrDescription("en"))
	assert.Empty(t, entry.TitleEn, "fallback must not fabricate title metadata")
}

func TestResponseSectionPreservesAllResponsesAndOnlyReachableComponents(t *testing.T) {
	api := &API{
		Responses: rawJSON(`{
			"200":{"schema":{"$ref":"#/components/schemas/Result"}},
			"400":{"schema":{"$ref":"#/components/schemas/Error"}}
		}`),
		Components: rawJSON(`{"schemas":{
			"Result":{"properties":{"item":{"$ref":"#/components/schemas/Item"}}},
			"Item":{"properties":{"parent":{"$ref":"#/components/schemas/Result"}}},
			"Error":{"properties":{"code":{"type":"string"}}},
			"Unused":{"type":"object"}
		}}`),
	}

	document, err := api.ResponseSection()
	require.NoError(t, err)
	require.JSONEq(t, string(api.Responses), string(document.Responses))
	assert.ElementsMatch(t, []string{"Result", "Item", "Error"}, rawMessageMapKeys(document.Components))
	assert.NotContains(t, document.Components, "Unused")
	assert.Contains(t, document.Warnings, `cyclic schema reference "#/components/schemas/Result" was preserved`)
}

func rawMessageMapKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
