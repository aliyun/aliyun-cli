/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
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
		Noticef(ctx.Writer(), "\nDid you mean:\n")
		for _, s := range ss {
			Noticef(ctx.Writer(), "  %s\n", s)
		}
	}
}

//
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
