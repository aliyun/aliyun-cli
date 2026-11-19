package openapi

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/openapi/runtimehost"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	runtimemeta "github.com/aliyun/aliyun-openapi-runtime/meta"
)

func candidateImportance(name string, scores map[string]float64) (float64, bool) {
	value, ok := scores[strings.TrimPrefix(name, "--")]
	return value, ok && runtimemeta.NewImportanceScore(&value) != nil
}

func hasCandidateImportance(names []string, scores map[string]float64) bool {
	for _, name := range names {
		if _, ok := candidateImportance(name, scores); ok {
			return true
		}
	}
	return false
}

func hasVaryingImportance(names []string, scores map[string]float64) bool {
	if len(names) < 2 {
		return false
	}
	for _, name := range names[1:] {
		if compareImportance(names[0], name, scores) != 0 {
			return true
		}
	}
	return false
}

func compareImportance(left, right string, scores map[string]float64) int {
	l, lok := candidateImportance(left, scores)
	r, rok := candidateImportance(right, scores)
	if lok != rok {
		if lok {
			return -1
		}
		return 1
	}
	if lok && l != r {
		if l > r {
			return -1
		}
		return 1
	}
	return 0
}

func rankImportance(names []string, scores map[string]float64) []string {
	result := append([]string(nil), names...)
	sort.SliceStable(result, func(i, j int) bool {
		return compareImportance(result[i], result[j], scores) < 0
	})
	return result
}

func rankedAPISuggestions(input string, candidates []string, scores map[string]float64, agent bool, recovery string) ([]string, bool) {
	if len(scores) == 0 {
		return nil, false
	}
	closeCandidates := candidates
	if !agent {
		closeCandidates = sameStyleCandidates(input, candidates)
	}
	if close := closeSuggestions(input, closeCandidates, false); len(close) > 0 {
		return rankImportance(close, scores), hasVaryingImportance(close, scores)
	}
	prefix, total := cli.PrefixSuggestions(input, sameStyleCandidates(input, candidates), len(candidates))
	if len(prefix) > 0 {
		active := hasVaryingImportance(prefix, scores)
		prefix = rankImportance(prefix, scores)
		if len(prefix) > cli.DefaultSuggestLimit {
			prefix = prefix[:cli.DefaultSuggestLimit]
		}
		if !agent && total > len(prefix) {
			prefix = append(prefix, fmt.Sprintf("... and %d more, run `%s`", total-len(prefix), recovery))
		}
		return prefix, active
	}
	return apiTokenSuggestionsRanked(input, candidates, cli.DefaultSuggestLimit, scores)
}

func rankParameterSuggestions(names []string, scores map[string]float64, globals map[string]bool) ([]string, bool) {
	positions, parameters := []int{}, []string{}
	for i, name := range names {
		if globals[strings.TrimPrefix(name, "--")] {
			continue
		}
		positions = append(positions, i)
		parameters = append(parameters, name)
	}
	if !hasCandidateImportance(parameters, scores) {
		return nil, false
	}
	parameters = rankImportance(parameters, scores)
	result := append([]string(nil), names...)
	for i, position := range positions {
		result[position] = parameters[i]
	}
	return result, true
}

func (e *InvalidParameterError) globalSuggestionNames() map[string]bool {
	names := map[string]bool{}
	if e.flags != nil {
		for _, flag := range e.flags.Flags() {
			names[flag.Name] = true
		}
	}
	return names
}

func withImportanceSuggestions(cause error, args []string, result error) error {
	agentErr, ok := result.(*cli.AgentError)
	if !ok {
		return result
	}
	var suggestions []string
	var active, apiNames bool
	var parameter *InvalidParameterError
	var api *InvalidApiError
	var baseline *InvalidBaselineCommandError
	var flag *argparser.UnknownFlagError
	switch {
	case errors.As(cause, &parameter):
		suggestions, active = rankParameterSuggestions(parameter.AgentSuggestions(), parameter.ImportanceScores, parameter.globalSuggestionNames())
	case errors.As(cause, &baseline):
		suggestions, active = rankedAPISuggestions(baseline.Command, baseline.Candidates, baseline.ImportanceScores, true, "")
		apiNames = true
	case errors.As(cause, &api) && api.product != nil:
		suggestions, active = rankedAPISuggestions(api.Name, api.product.ApiNames, api.ImportanceScores, true, "")
		apiNames = true
	case errors.As(cause, &flag):
		suggestions, active = rankParameterSuggestions(flagSuggestions(flag.Flag, flag.Known), flag.ImportanceScores, flag.GlobalNames)
	}
	if !active {
		return result
	}
	if apiNames {
		style := newRecoveryContext(args).style
		forms, seen := []string{}, map[string]bool{}
		for _, suggestion := range suggestions {
			for _, form := range apiCandidateFormsForStyle([]string{suggestion}, style) {
				if !seen[form] {
					forms = append(forms, form)
					seen[form] = true
				}
			}
		}
		suggestions = forms
	}
	envelope := agentErr.Envelope()
	envelope.DidYouMean = suggestions
	return cli.NewAgentError(envelope, cause)
}

func (c *Commando) attachAPIImportance(err error, args []string) {
	var baseline *InvalidBaselineCommandError
	if errors.As(err, &baseline) {
		scores := runtimehost.ProductCommandImportance(baseline.Product, args)
		if len(scores) > 0 {
			baseline.ImportanceScores = scores
		}
		return
	}
	var api *InvalidApiError
	if errors.As(err, &api) && api.product != nil && c.library != nil && c.library.canonicalRepo != nil {
		index, readErr := c.library.canonicalRepo.GetVersionIndex(api.product.Code, api.product.Version)
		if readErr == nil && index != nil {
			scores := map[string]float64{}
			for name, entry := range index.APIs {
				if score := entry.ImportanceScore.Value(); score != nil {
					scores[name] = *score
				}
			}
			if len(scores) > 0 {
				api.ImportanceScores = scores
			}
		}
	}
}
