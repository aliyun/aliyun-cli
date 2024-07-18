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

const DefaultSuggestDistance = 2

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

func (a *Suggester) GetResults() []string {
	return a.results
}
