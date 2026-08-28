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
	return fmt.Sprintf("%q is not a valid api. See `aliyun help %s`.", e.Name, e.product.GetLowerCode())
}

func (e *InvalidApiError) AgentMessage() string {
	return fmt.Sprintf("%q is not a valid api.", e.Name)
}

func (*InvalidApiError) AIRecoveryEligible() {}

func (e *InvalidApiError) GetSuggestions() []string {
	sr := cli.NewSuggester(e.Name, 2)
	for _, s := range e.product.ApiNames {
		sr.Apply(s)
	}
	return sr.GetResults()
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
	return fmt.Sprintf("%q is not a valid api. See `aliyun help %s`.", e.Name, e.product.GetLowerCode())
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
	Err     error
}

func (e *InvalidBaselineCommandError) Error() string {
	return explicitLocalErrorText(e.Err, "invalid baseline command")
}

func (e *InvalidBaselineCommandError) Unwrap() error { return e.Err }

func (*InvalidBaselineCommandError) AIRecoveryEligible() {}

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
	sr := cli.NewSuggester(e.Name, 2)
	for _, s := range e.product.ApiNames {
		sr.Apply(s)
	}
	for _, s := range e.lPlugin.CmdNames {
		sr.UnifyApply(s)
	}
	results := removeDuplicates(sr.GetResults())
	return results
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
