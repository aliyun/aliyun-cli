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
package cli

import (
	"strings"
	"unicode"
)

const DefaultSuggestDistance = 2

// DefaultSuggestLimit caps how many suggestions are listed when the prefix
// fallback triggers. Large products match hundreds of commands for short
// prefixes, so the list is truncated and the caller prints an overflow hint.
const DefaultSuggestLimit = 5

func CalculateStringDistance(source string, target string) int {
	return DistanceForStrings([]rune(source), []rune(target), DefaultOptions)
}

// error with suggestions
type SuggestibleError interface {
	GetSuggestions() []string
}

func PrintSuggestions(ctx *Context, lang string, ss []string) {
	if len(ss) > 0 {
		Noticef(ctx.Stdout(), "\nDid you mean:\n")
		for _, s := range ss {
			Noticef(ctx.Stdout(), "  %s\n", s)
		}
	}
}

// helper class for Suggester
type Suggester struct {
	suggestFor string
	distance   int
	results    []string
}

func NewSuggester(v string, distance int) *Suggester {
	return &Suggester{
		suggestFor: v,
		distance:   distance,
	}
}

func (a *Suggester) Apply(s string) {
	d := CalculateStringDistance(a.suggestFor, s)
	if d <= a.distance {
		if d < a.distance {
			a.distance = d
			a.results = make([]string, 0)
		}
		a.results = append(a.results, s)
	}
}

func stripNonAlphanumeric(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, r)
		}
	}
	return string(result)
}

func (a *Suggester) UnifyApply(s string) {
	cleanedSuggestFor := stripNonAlphanumeric(a.suggestFor)
	cleanedS := stripNonAlphanumeric(s)
	d := CalculateStringDistance(strings.ToLower(cleanedSuggestFor), strings.ToLower(cleanedS))
	if d <= a.distance {
		if d < a.distance {
			a.distance = d
			a.results = make([]string, 0)
		}
		a.results = append(a.results, s)
	}
}

func (a *Suggester) GetResults() []string {
	return a.results
}

// PrefixSuggestions returns up to limit candidates that begin with input,
// compared after stripping non-alphanumeric characters and lowercasing, plus
// the total number of matches before truncation. It is the fallback for
// edit-distance suggestion: a partial name such as "Get" or "get-caller" is
// a prefix rather than a typo, so distance-based matching cannot reach it.
func PrefixSuggestions(input string, candidates []string, limit int) (results []string, total int) {
	if limit <= 0 {
		limit = DefaultSuggestLimit
	}
	needle := strings.ToLower(stripNonAlphanumeric(input))
	if len(needle) < 2 {
		return nil, 0
	}
	matches := make([]string, 0)
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(stripNonAlphanumeric(candidate)), needle) {
			matches = append(matches, candidate)
		}
	}
	if len(matches) == 0 {
		return nil, 0
	}
	if len(matches) > limit {
		return matches[:limit], len(matches)
	}
	return matches, len(matches)
}
