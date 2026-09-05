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
	"errors"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

// LegacyMissingRequiredError marks required-parameter validation failures from
// the PascalCase OpenAPI path as explicit local CLI errors. The marker lets
// non-AI rendering append the AI-mode hint without guessing from error text.
type LegacyMissingRequiredError struct {
	Err error
}

func (e *LegacyMissingRequiredError) Error() string {
	if e == nil || e.Err == nil {
		return "required parameters are not assigned"
	}
	return e.Err.Error()
}

func (e *LegacyMissingRequiredError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (*LegacyMissingRequiredError) AIRecoveryEligible() {}

func newLegacyMissingRequiredError(err error) error {
	if err == nil {
		return nil
	}
	var existing *LegacyMissingRequiredError
	if errors.As(err, &existing) {
		return err
	}
	return &LegacyMissingRequiredError{Err: err}
}

// return when use unknown product
type InvalidProductError struct {
	Code    string
	library *Library
}

func (e *InvalidProductError) Error() string {
	return fmt.Sprintf("%q is not a valid command or product. See `aliyun help`.", strings.ToLower(e.Code))
}

func (e *InvalidProductError) AgentMessage() string {
	return fmt.Sprintf("%q is not a valid command or product.", strings.ToLower(e.Code))
}

func (*InvalidProductError) AIRecoveryEligible() {}

func (e *InvalidProductError) GetSuggestions() []string {
	sr := cli.NewSuggester(strings.ToLower(e.Code), 2)
	for _, p := range e.library.GetProducts() {
		sr.Apply(strings.ToLower(p.Code))
	}
	return sr.GetResults()
}

func (e *InvalidProductError) AgentSuggestions() []string {
	if e.library == nil {
		return nil
	}
	candidates := make([]string, 0)
	for _, product := range e.library.GetProducts() {
		candidates = append(candidates, strings.ToLower(product.Code))
	}
	return apiSuggestions(strings.ToLower(e.Code), candidates)
}

// return when use unknown api
type InvalidApiError struct {
	Name    string
	product *meta.Product
}

func (e *InvalidApiError) Error() string {
	product := e.product.GetLowerCode()
	if command := apiRecoveryCommand(e.Name, product, e.product.ApiNames); command != "" {
		return fmt.Sprintf("%q is not a valid api. Search matching APIs with `%s`.", e.Name, command)
	}
	return fmt.Sprintf("%q is not a valid api. See `aliyun help %s`.", e.Name, product)
}

func (e *InvalidApiError) AgentMessage() string {
	return fmt.Sprintf("%q is not a valid api.", e.Name)
}

func (*InvalidApiError) AIRecoveryEligible() {}

func (e *InvalidApiError) GetSuggestions() []string {
	return humanAPISuggestions(e.Name, e.product.ApiNames,
		apiRecoveryCommand(e.Name, e.product.GetLowerCode(), e.product.ApiNames))
}

func (e *InvalidApiError) AgentSuggestions() []string {
	if e.product == nil {
		return nil
	}
	return apiSuggestions(e.Name, e.product.ApiNames)
}

// return when use unknown parameter
type InvalidParameterError struct {
	Name              string
	ProductCode       string
	ApiName           string
	ParameterNames    []string
	ParameterExamples map[string]string
	flags             *cli.FlagSet
}

func (e *InvalidParameterError) Error() string {
	return fmt.Sprintf("%q is not a valid parameter or flag. See `aliyun help %s %s`.",
		"--"+e.Name, strings.ToLower(e.ProductCode), e.ApiName)
}

func (e *InvalidParameterError) AgentMessage() string {
	return fmt.Sprintf("%q is not a valid parameter or flag.", "--"+strings.TrimLeft(e.Name, "-"))
}

func (*InvalidParameterError) AIRecoveryEligible() {}

func (e *InvalidParameterError) GetSuggestions() []string {
	sr := cli.NewSuggester(e.Name, 2)
	for _, name := range e.ParameterNames {
		sr.Apply(name)
	}
	if e.flags != nil {
		for _, f := range e.flags.Flags() {
			sr.Apply(f.Name)
		}
	}

	results := sr.GetResults()
	for i, name := range results {
		if example := e.ParameterExamples[name]; example != "" {
			results[i] = fmt.Sprintf("%s (example: %s)", name, example)
		}
	}
	return results
}

func (e *InvalidParameterError) AgentSuggestions() []string {
	candidates := append([]string(nil), e.ParameterNames...)
	if e.flags != nil {
		for _, flag := range e.flags.Flags() {
			candidates = append(candidates, flag.Name)
		}
	}
	return flagSuggestions(e.Name, candidates)
}

// NewInvalidParameterErrorFromCanonical creates error from canonical API
func NewInvalidParameterErrorFromCanonical(name string, api *canonicalmeta.API, productCode string, flags *cli.FlagSet) *InvalidParameterError {
	views := api.LegacyTopLevelParameters()
	params := make([]string, 0, len(views))
	examples := make(map[string]string)
	for _, v := range views {
		pos := v.LegacyPosition()
		if pos == "Domain" || pos == "Header" {
			continue
		}
		name := v.LegacyName()
		params = append(params, name)
		if example := strings.TrimSpace(v.LegacyExample()); example != "" {
			examples[name] = example
		}
	}
	return &InvalidParameterError{
		Name:              name,
		ProductCode:       productCode,
		ApiName:           api.Name,
		ParameterNames:    params,
		ParameterExamples: examples,
		flags:             flags,
	}
}

type InvalidProductOrPluginError struct {
	Code string
	// Hint, when non-empty, is appended to Error() on its own line.
	// Used by callers that have additional context to share
	// — for example tryDelegatePluginHelp's step-4 explains why a 3+ arg lowercase shape was treated as a plugin command,
	// so users who actually meant an OpenAPI built-in call see the right syntax.
	// Default callers leave it empty; behaviour is unchanged.
	Hint    string
	library *Library
	plugins []plugin.PluginInfo
}

func (e *InvalidProductOrPluginError) Error() string {
	msg := fmt.Sprintf("%q is not a valid product. See `aliyun help`.", e.Code)
	if e.Hint != "" {
		msg += "\n" + e.Hint
	}
	return msg
}

func (e *InvalidProductOrPluginError) AgentMessage() string {
	return fmt.Sprintf("%q is not a valid product.", e.Code)
}

func (*InvalidProductOrPluginError) AIRecoveryEligible() {}

func (e *InvalidProductOrPluginError) GetSuggestions() []string {
	sr := cli.NewSuggester(strings.ToLower(e.Code), 2)
	for _, p := range e.plugins {
		sr.Apply(strings.ToLower(p.ProductCode))
	}
	// for _, p := range e.library.GetProducts() {
	// 	sr.Apply(strings.ToLower(p.Code))
	// }
	return sr.GetResults()
}

func (e *InvalidProductOrPluginError) AgentSuggestions() []string {
	candidates := make([]string, 0, len(e.plugins))
	for _, product := range e.plugins {
		candidates = append(candidates, strings.ToLower(product.ProductCode))
	}
	return apiSuggestions(strings.ToLower(e.Code), candidates)
}

type InvalidUnifiedApiError struct {
	Name    string
	product *meta.Product
	lPlugin plugin.LocalPlugin
}

func (e *InvalidUnifiedApiError) Error() string {
	product := e.product.GetLowerCode()
	candidates := append(append([]string(nil), e.product.ApiNames...), e.lPlugin.CmdNames...)
	if command := apiRecoveryCommand(e.Name, product, candidates); command != "" {
		return fmt.Sprintf("%q is not a valid api. Search matching APIs with `%s`.", e.Name, command)
	}
	return fmt.Sprintf("%q is not a valid api. See `aliyun help %s`.", e.Name, product)
}

func (e *InvalidUnifiedApiError) AgentMessage() string {
	return fmt.Sprintf("%q is not a valid api.", e.Name)
}

func (*InvalidUnifiedApiError) AIRecoveryEligible() {}

// InvalidBaselineCommandError adapts the runtime router's existing human text
// into an explicit unknown-API cause without changing that text.
type InvalidBaselineCommandError struct {
	Product string
	Command string
	// Candidates are the kebab command names the runtime engine serves for
	// the product; populated by the creation site so suggestions stay on the
	// kebab path instead of jumping to PascalCase APIs.
	Candidates []string
	Err        error
}

func (e *InvalidBaselineCommandError) Error() string {
	message := explicitLocalErrorText(e.Err, "invalid baseline command")
	if command := apiRecoveryCommand(e.Command, e.Product, e.Candidates); command != "" && !strings.Contains(message, command) {
		return strings.TrimSuffix(message, ".") + fmt.Sprintf(". Search matching APIs with `%s`.", command)
	}
	return message
}

func (e *InvalidBaselineCommandError) Unwrap() error { return e.Err }

func (*InvalidBaselineCommandError) AIRecoveryEligible() {}

func (e *InvalidBaselineCommandError) GetSuggestions() []string {
	return humanAPISuggestions(e.Command, e.Candidates,
		apiRecoveryCommand(e.Command, e.Product, e.Candidates))
}

func (e *InvalidBaselineCommandError) AgentSuggestions() []string {
	return apiSuggestions(e.Command, e.Candidates)
}

type InvalidArgumentError struct {
	Parameter    string
	Flag         string
	FieldPath    string
	ExpectedType string
	Err          error
}

func (e *InvalidArgumentError) Error() string {
	return explicitLocalErrorText(e.Err, "invalid argument")
}

func (e *InvalidArgumentError) Unwrap() error { return e.Err }

func (*InvalidArgumentError) AIRecoveryEligible() {}

type InvalidOptionCombinationError struct {
	Options []string
	Err     error
}

func (e *InvalidOptionCombinationError) Error() string {
	return explicitLocalErrorText(e.Err, "invalid option combination")
}

func (e *InvalidOptionCombinationError) Unwrap() error { return e.Err }

func (*InvalidOptionCombinationError) AIRecoveryEligible() {}

func (*InvalidOptionCombinationError) GetSuggestions() []string { return nil }

type InvalidHeaderError struct {
	Input          string
	ExpectedFormat string
	Err            error
}

func (e *InvalidHeaderError) Error() string { return explicitLocalErrorText(e.Err, "invalid header") }

func (e *InvalidHeaderError) Unwrap() error { return e.Err }

func (*InvalidHeaderError) AIRecoveryEligible() {}

type InvalidBodyFileError struct {
	Path string
	Err  error
}

func (e *InvalidBodyFileError) Error() string {
	return explicitLocalErrorText(e.Err, "invalid body file")
}

func (e *InvalidBodyFileError) Unwrap() error { return e.Err }

func (*InvalidBodyFileError) AIRecoveryEligible() {}

func explicitLocalErrorText(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	return err.Error()
}

func (e *InvalidUnifiedApiError) GetSuggestions() []string {
	candidates := append(append([]string(nil), e.product.ApiNames...), e.lPlugin.CmdNames...)
	return humanAPISuggestions(e.Name, candidates,
		apiRecoveryCommand(e.Name, e.product.GetLowerCode(), candidates))
}

func (e *InvalidUnifiedApiError) AgentSuggestions() []string {
	if e.product == nil {
		return nil
	}
	candidates := append([]string(nil), e.product.ApiNames...)
	candidates = append(candidates, e.lPlugin.CmdNames...)
	return apiSuggestions(e.Name, candidates)
}

func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// sameStyleCandidates keeps only candidates written in the input's command
// style: mixed-case input keeps PascalCase candidates and all-lowercase
// input keeps kebab candidates. Suggestions must not cross styles — a
// suggested command has to run through the same engine as the failed one.
func sameStyleCandidates(input string, candidates []string) []string {
	hasUpper := input != strings.ToLower(input)
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if (candidate != strings.ToLower(candidate)) == hasUpper {
			result = append(result, candidate)
		}
	}
	return result
}

func prefixSuggestionsWithOverflow(input string, candidates []string, helpCommand string) []string {
	results, total := cli.PrefixSuggestions(input, sameStyleCandidates(input, candidates), cli.DefaultSuggestLimit)
	if total > len(results) {
		results = append(results, fmt.Sprintf("... and %d more, run `%s`", total-len(results), helpCommand))
	}
	return results
}

// humanAPISuggestions applies the same precision order in every output mode:
// close typo, partial prefix, then meaningful token overlap. The recovery
// command is included only as an overflow hint; the error text always carries
// it so zero-match cases do not fall back to dumping the whole product Help.
func humanAPISuggestions(input string, candidates []string, recoveryCommand string) []string {
	if results := closeSuggestions(input, sameStyleCandidates(input, candidates), false); len(results) > 0 {
		return results
	}
	if results := prefixSuggestionsWithOverflow(input, candidates, recoveryCommand); len(results) > 0 {
		return results
	}
	return apiTokenSuggestions(input, candidates, cli.DefaultSuggestLimit)
}

func apiRecoveryCommand(input, product string, candidates []string) string {
	style := commandStyle(input)
	keyword := apiRecoverySearchKeyword(input, candidates, style)
	if keyword == "" || !safeCommandToken(keyword) {
		return ""
	}
	if style == "kebab" {
		return baselineSearchHelpCommand(keyword, product)
	}
	return apiSearchHelpCommand(keyword, product)
}

// apiSearchHelpCommand builds the PascalCase-style search hint. --help-search
// returns a compact filtered list, so it costs far fewer tokens than dumping
// the full product help when a prefix matches many APIs.
func apiSearchHelpCommand(input, product string) string {
	if safeCommandToken(input) {
		return fmt.Sprintf("aliyun %s --help-search %s", product, input)
	}
	return fmt.Sprintf("aliyun help %s", product)
}

// baselineSearchHelpCommand builds the kebab-style search hint. Products that
// also have legacy PascalCase help render it by default, so baseline kebab
// help must be requested through the env var prefix.
func baselineSearchHelpCommand(input, product string) string {
	prefix := baselineProductHelpEnv + "=true aliyun " + strings.ToLower(product)
	if safeCommandToken(input) {
		return prefix + " --help-search " + input
	}
	return prefix + " --help"
}
