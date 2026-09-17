package openapi

import (
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

// Product abbreviations are more likely to be partial names than typos.
// Return the strongest nonempty tier: prefix, substring, then edit distance.
// Sort and deduplicate before truncating so repository order cannot affect
// either the displayed suggestions or the recovery command.
func productSuggestions(input string, candidates []string) []string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return nil
	}
	normalized := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(candidate)))
	}
	candidates = stableStrings(normalized)
	var prefixes, substrings []string
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, input) {
			prefixes = append(prefixes, candidate)
		} else if strings.Contains(candidate, input) {
			substrings = append(substrings, candidate)
		}
	}
	if len(prefixes) > 0 {
		return prefixes[:min(len(prefixes), cli.DefaultSuggestLimit)]
	}
	if len(substrings) > 0 {
		return substrings[:min(len(substrings), cli.DefaultSuggestLimit)]
	}
	return closeSuggestions(input, candidates, false)
}
