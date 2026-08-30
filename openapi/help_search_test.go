package openapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchHelpCandidatesNormalizesCaseAndSeparators(t *testing.T) {
	candidates := []HelpSearchCandidate{{
		Kind: "parameter",
		Name: "InstanceId",
	}}

	for _, keyword := range []string{
		"InstanceId",
		"INSTANCE-ID",
		"instance_id",
		"instance id",
	} {
		t.Run(keyword, func(t *testing.T) {
			matches := SearchHelpCandidates(candidates, keyword)
			require.Len(t, matches, 1)
			assert.Equal(t, HelpSearchExactName, matches[0].Rank)
		})
	}
}

func TestSearchHelpCandidatesRanksDeterministically(t *testing.T) {
	candidates := []HelpSearchCandidate{
		{Name: "Unrelated", DescriptionEN: "Use the instance id to identify the resource"},
		{Name: "preinstanceidpost"},
		{Name: "InstanceIdentifier"},
		{Name: "InstanceId", Value: "first exact"},
		{Name: "instance-id", Value: "second exact"},
	}

	matches := SearchHelpCandidates(candidates, "instance_id")
	require.Len(t, matches, 5)
	assert.Equal(t, []string{
		"InstanceId",
		"instance-id",
		"InstanceIdentifier",
		"preinstanceidpost",
		"Unrelated",
	}, helpSearchMatchNames(matches))
	assert.Equal(t, []HelpSearchRank{
		HelpSearchExactName,
		HelpSearchExactName,
		HelpSearchNameTokenPrefix,
		HelpSearchNameContains,
		HelpSearchTextContains,
	}, helpSearchMatchRanks(matches))
}

func TestSearchHelpCandidatesMatchesAliasesAndLocalizedText(t *testing.T) {
	candidates := []HelpSearchCandidate{
		{Name: "ecs", TitleZH: "云服务器", DescriptionEN: "Elastic Compute Service"},
		{Name: "CreateReport", Aliases: []string{"create-report"}, DescriptionZH: "创建报表"},
		{Name: "ReportId", Aliases: []string{"--report-id", "report_id"}},
	}

	tests := []struct {
		keyword string
		name    string
		rank    HelpSearchRank
	}{
		{keyword: "云服务器", name: "ecs", rank: HelpSearchTextContains},
		{keyword: "elastic compute", name: "ecs", rank: HelpSearchTextContains},
		{keyword: "create_report", name: "CreateReport", rank: HelpSearchExactName},
		{keyword: "--REPORT-ID", name: "ReportId", rank: HelpSearchExactName},
	}

	for _, tt := range tests {
		t.Run(tt.keyword, func(t *testing.T) {
			matches := SearchHelpCandidates(candidates, tt.keyword)
			require.NotEmpty(t, matches)
			assert.Equal(t, tt.name, matches[0].Candidate.Name)
			assert.Equal(t, tt.rank, matches[0].Rank)
		})
	}
}

func TestSearchHelpParametersUsesOnlyActiveSetAndGlobals(t *testing.T) {
	input := HelpParameterSearchInput{
		ActiveParameterSet: "kebab",
		ParameterSets: map[string][]HelpSearchCandidate{
			"camel": {{Name: "ReportId", Value: "camel"}},
			"kebab": {{Name: "report-id", Aliases: []string{"--report-id"}, Value: "kebab"}},
		},
		GlobalParameters: []HelpSearchCandidate{
			{Name: "header", Aliases: []string{"--header"}, Value: "global"},
			{Name: "body-file", Aliases: []string{"--body-file"}, Value: "global"},
			{Name: "pager", Aliases: []string{"--pager"}, Value: "global"},
		},
	}

	matches := SearchHelpParameters(input, "report_id")
	require.Len(t, matches, 1)
	assert.Equal(t, "kebab", matches[0].Candidate.Value)

	for _, keyword := range []string{"header", "body_file", "PAGER"} {
		t.Run(keyword, func(t *testing.T) {
			matches := SearchHelpParameters(input, keyword)
			require.Len(t, matches, 1)
			assert.Equal(t, "global", matches[0].Candidate.Value)
		})
	}
}

func TestSearchHelpCandidatesNeverCapsResults(t *testing.T) {
	candidates := make([]HelpSearchCandidate, 25)
	for i := range candidates {
		candidates[i] = HelpSearchCandidate{Name: fmt.Sprintf("DescribeInstance%02d", i)}
	}

	matches := SearchHelpCandidates(candidates, "instance")
	assert.Len(t, matches, 25)
	assert.Equal(t, "DescribeInstance00", matches[0].Candidate.Name)
	assert.Equal(t, "DescribeInstance24", matches[24].Candidate.Name)

	validation := ValidateHelpSearch(candidates, "instance")
	assert.Equal(t, HelpSearchValidation{Matched: true, MatchCount: 25}, validation)
	assert.Equal(t, HelpSearchValidation{}, ValidateHelpSearch(candidates, "does-not-exist"))
}

func TestProjectHelpListingCapsOnlyUnsearchedAIRootAndProductLists(t *testing.T) {
	items20 := integerRange(20)
	items21 := integerRange(21)
	items100 := integerRange(100)

	tests := []struct {
		name        string
		items       []int
		options     HelpListingOptions
		wantShown   int
		wantListing *HelpListingMetadata
	}{
		{
			name:      "exactly twenty is not truncated",
			items:     items20,
			options:   HelpListingOptions{Target: HelpListingRootProducts, AIMode: true},
			wantShown: 20,
		},
		{
			name:      "twenty one root products are capped",
			items:     items21,
			options:   HelpListingOptions{Target: HelpListingRootProducts, AIMode: true},
			wantShown: 20,
			wantListing: &HelpListingMetadata{
				Shown: 20,
				Total: 21,
				Hint:  "Use --help-search <keyword> to narrow the list, or --help-all to show everything.",
			},
		},
		{
			name:      "large API list is capped",
			items:     items100,
			options:   HelpListingOptions{Target: HelpListingProductAPIs, AIMode: true},
			wantShown: 20,
			wantListing: &HelpListingMetadata{
				Shown: 20,
				Total: 100,
				Hint:  "Use --help-search <keyword> to narrow the list, or --help-all to show everything.",
			},
		},
		{
			name:      "search bypasses cap",
			items:     items100,
			options:   HelpListingOptions{Target: HelpListingRootProducts, AIMode: true, Searched: true},
			wantShown: 100,
		},
		{
			name:      "cli all bypasses cap",
			items:     items100,
			options:   HelpListingOptions{Target: HelpListingProductAPIs, AIMode: true, All: true},
			wantShown: 100,
		},
		{
			name:      "non AI list remains complete",
			items:     items100,
			options:   HelpListingOptions{Target: HelpListingRootProducts},
			wantShown: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shown, listing := ProjectHelpListing(tt.items, tt.options)
			assert.Len(t, shown, tt.wantShown)
			assert.Equal(t, tt.wantListing, listing)
		})
	}
}

func TestSearchResponseSchemaPrunesToMatchedRefPath(t *testing.T) {
	document := HelpResponseSchema{
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"RequestId":{"type":"string","description_zh":"请求标识"},
				"Instances":{"type":"object","properties":{
					"Instance":{"type":"array","items":{"$ref":"#/components/schemas/Instance"}}
				}}
			}
		}`),
		Components: map[string]json.RawMessage{
			"Instance": json.RawMessage(`{
				"type":"object",
				"required":["InstanceId","InstanceName"],
				"properties":{
					"InstanceId":{"type":"string","title_zh":"实例标识"},
					"InstanceName":{"type":"string"},
					"Tags":{"type":"array","items":{"type":"object","properties":{
						"Key":{"type":"string"},
						"Value":{"type":"string"}
					}}}
				}
			}`),
			"Unused": json.RawMessage(`{"type":"object","properties":{"Nope":{"type":"string"}}}`),
		},
	}

	result, err := SearchResponseSchema(document, "instance-id", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"Instances.Instance.InstanceId"}, result.Paths)
	assertRawJSONEq(t, `{
		"type":"object",
		"properties":{
			"Instances":{"type":"object","properties":{
				"Instance":{"type":"array","items":{"$ref":"#/components/schemas/Instance"}}
			}}
		}
	}`, result.Schema)
	require.Equal(t, []string{"Instance"}, sortedRawMessageKeys(result.Components))
	assertRawJSONEq(t, `{
		"type":"object",
		"required":["InstanceId"],
		"properties":{"InstanceId":{"type":"string","title_zh":"实例标识"}}
	}`, result.Components["Instance"])

	validation, err := ValidateResponseSchemaSearch(document, "instance_id")
	require.NoError(t, err)
	assert.Equal(t, HelpSearchValidation{Matched: true, MatchCount: 1}, validation)
}

func TestSearchResponseSchemaMergesMultipleMatchesInSchemaOrder(t *testing.T) {
	document := HelpResponseSchema{
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"RequestId":{"type":"string","description_zh":"请求标识"},
				"Result":{"$ref":"#/components/schemas/Result"}
			}
		}`),
		Components: map[string]json.RawMessage{
			"Result": json.RawMessage(`{
				"type":"object",
				"properties":{
					"InstanceId":{"type":"string","title_zh":"实例标识"},
					"Other":{"type":"string"}
				}
			}`),
		},
	}

	result, err := SearchResponseSchema(document, "标识", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"RequestId", "Result.InstanceId"}, result.Paths)
	assertRawJSONEq(t, `{
		"type":"object",
		"properties":{
			"RequestId":{"type":"string","description_zh":"请求标识"},
			"Result":{"$ref":"#/components/schemas/Result"}
		}
	}`, result.Schema)
	assertRawJSONEq(t, `{
		"type":"object",
		"properties":{"InstanceId":{"type":"string","title_zh":"实例标识"}}
	}`, result.Components["Result"])
}

func TestSearchResponseSchemaMatchesFullPathAcrossSeparators(t *testing.T) {
	document := HelpResponseSchema{
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{"Instances":{"type":"object","properties":{
				"Instance":{"type":"object","properties":{
					"InstanceId":{"type":"string"},
					"State":{"type":"string"}
				}}
			}}}
		}`),
	}

	for _, keyword := range []string{
		"Instances.Instance.InstanceId",
		"instances-instance-instance-id",
		"instances_instance_instance_id",
		"instances instance instance id",
	} {
		t.Run(keyword, func(t *testing.T) {
			result, err := SearchResponseSchema(document, keyword, false)
			require.NoError(t, err)
			assert.Equal(t, []string{"Instances.Instance.InstanceId"}, result.Paths)
		})
	}

	result, err := SearchResponseSchema(document, "instances", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"Instances"}, result.Paths)
}

func TestSearchResponseSchemaRanksMatchesByRelevance(t *testing.T) {
	document := HelpResponseSchema{
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"Unrelated":{"type":"string","description_en":"The instance id."},
				"preinstanceidpost":{"type":"string"},
				"InstanceIdentifier":{"type":"string"},
				"InstanceId":{"type":"string"}
			}
		}`),
	}

	result, err := SearchResponseSchema(document, "instance_id", false)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"InstanceId",
		"InstanceIdentifier",
		"preinstanceidpost",
		"Unrelated",
	}, result.Paths)
}

func TestSearchResponseSchemaPrunesUnmatchedCompositionBranches(t *testing.T) {
	for _, keyword := range []string{"allOf", "oneOf", "anyOf"} {
		t.Run(keyword, func(t *testing.T) {
			document := HelpResponseSchema{
				Schema: json.RawMessage(`{
					"type":"object",
					"properties":{"Result":{"` + keyword + `":[
						{"type":"object","properties":{
							"MatchingField":{"type":"string"},
							"UnusedSibling":{"type":"string"}
						}},
						{"type":"object","properties":{"UnusedBranch":{"type":"string"}}}
					]}}
				}`),
			}

			result, err := SearchResponseSchema(document, "matching-field", false)
			require.NoError(t, err)
			assert.Equal(t, []string{"Result.MatchingField"}, result.Paths)
			assertRawJSONEq(t, `{
				"type":"object",
				"properties":{"Result":{"`+keyword+`":[
					{"type":"object","properties":{"MatchingField":{"type":"string"}}}
				]}}
			}`, result.Schema)
		})
	}
}

func TestSearchResponseSchemaMatchesTitleOnReferencedField(t *testing.T) {
	document := HelpResponseSchema{
		Schema: json.RawMessage(`{"type":"object","properties":{"Result":{"$ref":"#/components/schemas/Result"}}}`),
		Components: map[string]json.RawMessage{
			"Result": json.RawMessage(`{
				"type":"object",
				"title_zh":"结果集合",
				"properties":{"Value":{"type":"string"}}
			}`),
		},
	}

	result, err := SearchResponseSchema(document, "结果集合", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"Result"}, result.Paths)
	assertRawJSONEq(t, `{
		"type":"object",
		"properties":{"Result":{"$ref":"#/components/schemas/Result"}}
	}`, result.Schema)
	assertRawJSONEq(t, `{"type":"object","title_zh":"结果集合"}`, result.Components["Result"])
}

func TestSearchResponseSchemaKeepsArrayItemsAndReachableComponents(t *testing.T) {
	document := HelpResponseSchema{
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"Entries":{"type":"array","items":{"$ref":"#/components/schemas/Entry"}},
				"Other":{"type":"string"}
			}
		}`),
		Components: map[string]json.RawMessage{
			"Entry": json.RawMessage(`{
				"type":"object",
				"properties":{
					"Name":{"type":"string"},
					"Nested":{"$ref":"#/components/schemas/Nested"}
				}
			}`),
			"Nested": json.RawMessage(`{"type":"object","properties":{"Value":{"type":"string"}}}`),
			"Unused": json.RawMessage(`{"type":"string"}`),
		},
	}

	result, err := SearchResponseSchema(document, "entries", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"Entries"}, result.Paths)
	assertRawJSONEq(t, `{
		"type":"object",
		"properties":{"Entries":{"type":"array","items":{"$ref":"#/components/schemas/Entry"}}}
	}`, result.Schema)
	assert.Equal(t, []string{"Entry", "Nested"}, sortedRawMessageKeys(result.Components))
	assertRawJSONEq(t, document.Components["Entry"], result.Components["Entry"])
	assertRawJSONEq(t, document.Components["Nested"], result.Components["Nested"])
}

func TestSearchResponseSchemaPrunesNestedComponentClosure(t *testing.T) {
	document := HelpResponseSchema{
		Schema: json.RawMessage(`{"type":"object","properties":{"Result":{"$ref":"#/components/schemas/Page"}}}`),
		Components: map[string]json.RawMessage{
			"Page": json.RawMessage(`{
				"type":"object",
				"properties":{
					"Data":{"$ref":"#/components/schemas/Data"},
					"Ignored":{"type":"string"}
				}
			}`),
			"Data": json.RawMessage(`{
				"type":"object",
				"properties":{
					"DisplayName":{"type":"string"},
					"Count":{"type":"integer"}
				}
			}`),
			"Unused": json.RawMessage(`{"type":"object"}`),
		},
	}

	result, err := SearchResponseSchema(document, "display-name", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"Result.Data.DisplayName"}, result.Paths)
	assert.Equal(t, []string{"Data", "Page"}, sortedRawMessageKeys(result.Components))
	assertRawJSONEq(t, `{
		"type":"object",
		"properties":{"Data":{"$ref":"#/components/schemas/Data"}}
	}`, result.Components["Page"])
	assertRawJSONEq(t, `{
		"type":"object",
		"properties":{"DisplayName":{"type":"string"}}
	}`, result.Components["Data"])
}

func TestSearchResponseSchemaHandlesCyclesMissingRefsAndEmptyMatches(t *testing.T) {
	document := HelpResponseSchema{
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"Node":{"$ref":"#/components/schemas/Node"},
				"Missing":{"$ref":"#/components/schemas/DoesNotExist"}
			}
		}`),
		Components: map[string]json.RawMessage{
			"Node": json.RawMessage(`{
				"type":"object",
				"properties":{
					"Name":{"type":"string"},
					"Child":{"$ref":"#/components/schemas/Node"}
				}
			}`),
		},
	}

	result, err := SearchResponseSchema(document, "name", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"Node.Name"}, result.Paths)
	assert.Equal(t, []string{"Node"}, sortedRawMessageKeys(result.Components))
	assertRawJSONEq(t, `{
		"type":"object",
		"properties":{"Name":{"type":"string"}}
	}`, result.Components["Node"])

	result, err = SearchResponseSchema(document, "missing", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"Missing"}, result.Paths)
	assert.Empty(t, result.Components)
	assertRawJSONEq(t, `{
		"type":"object",
		"properties":{"Missing":{"$ref":"#/components/schemas/DoesNotExist"}}
	}`, result.Schema)

	result, err = SearchResponseSchema(document, "not-present", false)
	require.NoError(t, err)
	assert.Empty(t, result.Paths)
	assert.Nil(t, result.Schema)
	assert.Empty(t, result.Components)
}

func TestSearchResponseSchemaRejectsMalformedJSON(t *testing.T) {
	_, err := SearchResponseSchema(HelpResponseSchema{Schema: json.RawMessage(`{"type":`)}, "field", false)
	require.Error(t, err)
}

func helpSearchMatchNames(matches []HelpSearchMatch) []string {
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		result = append(result, match.Candidate.Name)
	}
	return result
}

func helpSearchMatchRanks(matches []HelpSearchMatch) []HelpSearchRank {
	result := make([]HelpSearchRank, 0, len(matches))
	for _, match := range matches {
		result = append(result, match.Rank)
	}
	return result
}

func integerRange(size int) []int {
	result := make([]int, size)
	for i := range result {
		result[i] = i
	}
	return result
}

func sortedRawMessageKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func assertRawJSONEq(t *testing.T, expected any, actual json.RawMessage) {
	t.Helper()
	var expectedJSON string
	switch value := expected.(type) {
	case string:
		expectedJSON = value
	case json.RawMessage:
		expectedJSON = string(value)
	default:
		require.FailNow(t, "unsupported expected JSON type", "%T", expected)
	}
	require.NotEmpty(t, actual)
	assert.JSONEq(t, expectedJSON, string(actual))
}

func TestProjectHelpSearchMatchesUnlimitedLiftsCap(t *testing.T) {
	matches := make([]HelpSearchMatch, helpSearchResultLimit+5)
	for i := range matches {
		matches[i] = HelpSearchMatch{Candidate: HelpSearchCandidate{Name: fmt.Sprintf("item-%02d", i)}}
	}

	capped := ProjectHelpSearchMatches(matches, false)
	assert.Equal(t, helpSearchResultLimit, capped.Result.Shown)
	assert.Equal(t, len(matches), capped.Result.Total)
	assert.True(t, capped.Result.Truncated)

	unlimited := ProjectHelpSearchMatches(matches, true)
	assert.Equal(t, len(matches), unlimited.Result.Shown)
	assert.Equal(t, len(matches), unlimited.Result.Total)
	assert.False(t, unlimited.Result.Truncated)
}
