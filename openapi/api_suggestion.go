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
	"sort"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

// API action verbs are deliberately excluded from token-overlap scoring.
// Otherwise almost every Describe*/Query*/Get* command in a large product
// would look related and the fallback would recreate the guessing loop this
// matcher is meant to stop.
var apiSuggestionActionTokens = map[string]struct{}{
	"add": {}, "allocate": {}, "apply": {}, "batch": {}, "cancel": {},
	"check": {}, "create": {}, "delete": {}, "describe": {}, "disable": {},
	"enable": {}, "execute": {}, "fetch": {}, "find": {}, "get": {},
	"list": {}, "modify": {}, "put": {}, "query": {}, "release": {},
	"remove": {}, "renew": {}, "run": {}, "search": {}, "set": {},
	"start": {}, "stop": {}, "submit": {}, "update": {}, "validate": {},
}

// Generic qualifiers carry little resource identity and can otherwise create
// false positives such as QueryCouponDetails -> QueryAccountDetails.
var apiSuggestionGenericTokens = map[string]struct{}{
	"available": {}, "detail": {}, "info": {}, "information": {},
}

type apiSuggestionScore struct {
	name           string
	overlap        int
	overlapWeight  int
	inputMissing   int
	candidateExtra int
	distance       int
}

// apiTokenSuggestions ranks same-style API names by meaningful token overlap.
// It is intentionally a fallback after edit-distance and prefix matching: a
// close typo remains more precise than a shared resource word.
func apiTokenSuggestions(input string, candidates []string, limit int) []string {
	if limit <= 0 {
		limit = cli.DefaultSuggestLimit
	}
	inputTokens := meaningfulAPITokenSet(input)
	if len(inputTokens) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(candidates))
	scores := make([]apiSuggestionScore, 0)
	inputCompact := compactAPITokens(input)
	for _, candidate := range sameStyleCandidates(input, candidates) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		key := strings.ToLower(candidate)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		candidateTokens := meaningfulAPITokenSet(candidate)
		overlap := 0
		overlapWeight := 0
		for token := range inputTokens {
			if _, exists := candidateTokens[token]; exists {
				overlap++
				overlapWeight += len([]rune(token))
			}
		}
		// A single meaningful token is enough when it has at least three
		// characters (Coupon, Vpc, ECS, ...). Action verbs never reach here.
		if overlap == 0 || overlapWeight < 3 {
			continue
		}
		scores = append(scores, apiSuggestionScore{
			name:           candidate,
			overlap:        overlap,
			overlapWeight:  overlapWeight,
			inputMissing:   len(inputTokens) - overlap,
			candidateExtra: len(candidateTokens) - overlap,
			distance:       cli.CalculateStringDistance(inputCompact, compactAPITokens(candidate)),
		})
	}

	sort.SliceStable(scores, func(i, j int) bool {
		left, right := scores[i], scores[j]
		if left.overlap != right.overlap {
			return left.overlap > right.overlap
		}
		if left.overlapWeight != right.overlapWeight {
			return left.overlapWeight > right.overlapWeight
		}
		if left.inputMissing != right.inputMissing {
			return left.inputMissing < right.inputMissing
		}
		if left.candidateExtra != right.candidateExtra {
			return left.candidateExtra < right.candidateExtra
		}
		if left.distance != right.distance {
			return left.distance < right.distance
		}
		return strings.ToLower(left.name) < strings.ToLower(right.name)
	})

	if len(scores) == 0 {
		return nil
	}
	if len(scores) > limit {
		scores = scores[:limit]
	}
	result := make([]string, len(scores))
	for index := range scores {
		result[index] = scores[index].name
	}
	return result
}

func meaningfulAPITokenSet(value string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, token := range splitHelpSearchTokens(value) {
		token = normalizeAPISuggestionToken(token)
		if token == "" {
			continue
		}
		if ignoredAPISuggestionToken(token) {
			continue
		}
		result[token] = struct{}{}
	}
	return result
}

func ignoredAPISuggestionToken(token string) bool {
	if _, action := apiSuggestionActionTokens[token]; action {
		return true
	}
	_, generic := apiSuggestionGenericTokens[token]
	return generic
}

func compactAPITokens(value string) string {
	tokens := splitHelpSearchTokens(value)
	for index := range tokens {
		tokens[index] = normalizeAPISuggestionToken(tokens[index])
	}
	return strings.Join(tokens, "")
}

// normalizeAPISuggestionToken performs a deliberately small amount of English
// singularisation. Its purpose is to make Coupon and Coupons (or Instance and
// Instances) comparable, not to act as a general-purpose stemmer.
func normalizeAPISuggestionToken(token string) string {
	token = strings.ToLower(strings.TrimSpace(token))
	if len(token) <= 3 {
		return token
	}
	switch {
	case strings.HasSuffix(token, "ies") && len(token) > 4:
		return strings.TrimSuffix(token, "ies") + "y"
	case strings.HasSuffix(token, "sses"), strings.HasSuffix(token, "xes"),
		strings.HasSuffix(token, "zes"), strings.HasSuffix(token, "ches"),
		strings.HasSuffix(token, "shes"):
		return strings.TrimSuffix(token, "es")
	case strings.HasSuffix(token, "s") && !strings.HasSuffix(token, "ss") &&
		!strings.HasSuffix(token, "us") && !strings.HasSuffix(token, "is"):
		return strings.TrimSuffix(token, "s")
	default:
		return token
	}
}

// apiRecoverySearchKeyword selects a real resource token from the failed name.
// Tokens that occur in candidate APIs are preferred; rarer matches are more
// precise for --help-search than broad words such as Instance or Resource.
func apiRecoverySearchKeyword(input string, candidates []string, style string) string {
	tokens := make([]string, 0)
	seen := make(map[string]struct{})
	for _, token := range splitHelpSearchTokens(input) {
		token = normalizeAPISuggestionToken(token)
		if token == "" {
			continue
		}
		if ignoredAPISuggestionToken(token) {
			continue
		}
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	if len(tokens) == 0 {
		return ""
	}

	occurrences := make(map[string]int, len(tokens))
	for _, candidate := range sameStyleCandidates(input, candidates) {
		candidateTokens := meaningfulAPITokenSet(candidate)
		for _, token := range tokens {
			if _, exists := candidateTokens[token]; exists {
				occurrences[token]++
			}
		}
	}

	best := tokens[0]
	bestOccurrences := occurrences[best]
	for _, token := range tokens[1:] {
		count := occurrences[token]
		switch {
		case bestOccurrences == 0 && count > 0:
			best, bestOccurrences = token, count
		case count > 0 && bestOccurrences > 0 && count < bestOccurrences:
			best, bestOccurrences = token, count
		case count == bestOccurrences && len([]rune(token)) > len([]rune(best)):
			best = token
		}
	}
	return formatSearchToken(best, style)
}
