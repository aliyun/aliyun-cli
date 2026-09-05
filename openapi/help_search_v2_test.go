package openapi

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitHelpSearchTokensSeparatesAlphaDigitBoundaries(t *testing.T) {
	assert.Equal(t, []string{"ipv", "6", "gateway", "2", "id"}, splitHelpSearchTokens("IPv6Gateway2ID"))
	assert.Equal(t, []string{"query", "monthly", "bill"}, splitHelpSearchTokens("QueryMonthlyBill"))
	assert.Equal(t, []string{"instance", "id"}, splitHelpSearchTokens("InstanceID"))
}

func TestProjectHelpSearchMatchesRanksAllThenCapsTwenty(t *testing.T) {
	candidates := make([]HelpSearchCandidate, 0, 25)
	for index := 24; index >= 0; index-- {
		candidates = append(candidates, HelpSearchCandidate{Name: fmt.Sprintf("Item%02d", index)})
	}

	projection := ProjectHelpSearchMatches(SearchHelpCandidates(candidates, "item"), false)
	require.Len(t, projection.Matches, helpSearchResultLimit)
	assert.Equal(t, "Item00", projection.Matches[0].Candidate.Name)
	assert.Equal(t, "Item19", projection.Matches[19].Candidate.Name)
	assert.Equal(t, HelpResult{Shown: 20, Total: 25, Truncated: true}, projection.Result)

	validation := ValidateHelpSearch(candidates, "item")
	assert.Equal(t, HelpSearchValidation{Matched: true, MatchCount: 25}, validation)
}

func TestSearchResponseSchemaRanksAllThenProjectsTopTwenty(t *testing.T) {
	properties := ""
	for index := 24; index >= 0; index-- {
		if properties != "" {
			properties += ","
		}
		properties += fmt.Sprintf(`"Item%02d":{"type":"string"}`, index)
	}
	document := HelpResponseSchema{Schema: json.RawMessage(`{"type":"object","properties":{` + properties + `}}`)}

	result, err := SearchResponseSchema(document, "item", false)
	require.NoError(t, err)
	require.Len(t, result.Paths, helpSearchResultLimit)
	assert.Equal(t, "Item00", result.Paths[0])
	assert.Equal(t, "Item19", result.Paths[19])
	assert.Equal(t, HelpResult{Shown: 20, Total: 25, Truncated: true}, result.Result)

	var projected struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(result.Schema, &projected))
	assert.Len(t, projected.Properties, 20)
	assert.NotContains(t, projected.Properties, "Item20")
}
