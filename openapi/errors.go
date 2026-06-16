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
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

// return when use unknown product
type InvalidProductError struct {
	Code    string
	library *Library
}

func (e *InvalidProductError) Error() string {
	return fmt.Sprintf("'%s' is not a valid command or product. See `aliyun help`.", strings.ToLower(e.Code))
}

func (e *InvalidProductError) GetSuggestions() []string {
	sr := cli.NewSuggester(strings.ToLower(e.Code), 2)
	for _, p := range e.library.GetProducts() {
		sr.Apply(strings.ToLower(p.Code))
	}
	return sr.GetResults()
}

// return when use unknown api
type InvalidApiError struct {
	Name    string
	product *meta.Product
}

func (e *InvalidApiError) Error() string {
	return fmt.Sprintf("'%s' is not a valid api. See `aliyun help %s`.", e.Name, e.product.GetLowerCode())
}

func (e *InvalidApiError) GetSuggestions() []string {
	sr := cli.NewSuggester(e.Name, 2)
	for _, s := range e.product.ApiNames {
		sr.Apply(s)
	}
	return sr.GetResults()
}

// return when use unknown parameter
type InvalidParameterError struct {
	Name  string
	api   *meta.Api
	flags *cli.FlagSet
}

func (e *InvalidParameterError) Error() string {
	return fmt.Sprintf("'--%s' is not a valid parameter or flag. See `aliyun help %s %s`.",
		e.Name, e.api.Product.GetLowerCode(), e.api.Name)
}

func (e *InvalidParameterError) GetSuggestions() []string {
	sr := cli.NewSuggester(e.Name, 2)
	for _, p := range e.api.Parameters {
		sr.Apply(p.Name)
	}
	for _, f := range e.flags.Flags() {
		sr.Apply(f.Name)
	}
	return sr.GetResults()
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
	msg := fmt.Sprintf("'%s' is not a valid built-in product or external product plugin. See `aliyun help`.", e.Code)
	if e.Hint != "" {
		msg += "\n" + e.Hint
	}
	return msg
}

func (e *InvalidProductOrPluginError) GetSuggestions() []string {
	sr := cli.NewSuggester(strings.ToLower(e.Code), 2)
	for _, p := range e.plugins {
		sr.Apply(strings.ToLower(p.ProductCode))
	}
	if e.library != nil {
		for _, p := range e.library.GetProducts() {
			sr.Apply(strings.ToLower(p.Code))
		}
	}
	return removeDuplicates(sr.GetResults())
}

type InvalidUnifiedApiError struct {
	Name    string
	product *meta.Product
	lPlugin plugin.LocalPlugin
}

func (e *InvalidUnifiedApiError) Error() string {
	return fmt.Sprintf("'%s' is not a valid api. See `aliyun help %s`.", e.Name, e.product.GetLowerCode())
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

// InvalidRestfulPathError is returned when aliyun <product> <METHOD> <path> cannot be resolved.
// If the path exists for other HTTP methods, GetSuggestions lists them.
type InvalidRestfulPathError struct {
	Method  string
	Path    string
	Product *meta.Product
	matches []meta.Api
}

func newInvalidRestfulPathError(library *Library, product *meta.Product, method, path string) error {
	matches := library.FindApisByPath(product.Code, product.Version, path)
	if len(matches) == 0 {
		return cli.NewErrorWithTip(fmt.Errorf("can not find api by path %s", path),
			"Please confirm if the API path exists")
	}
	return &InvalidRestfulPathError{
		Method:  method,
		Path:    path,
		Product: product,
		matches: matches,
	}
}

func (e *InvalidRestfulPathError) Error() string {
	return fmt.Sprintf("can not find api by path %s with method %s", e.Path, strings.ToUpper(e.Method))
}

func (e *InvalidRestfulPathError) GetSuggestions() []string {
	suggestions := make([]string, 0, len(e.matches))
	for _, api := range e.matches {
		suggestions = append(suggestions, fmt.Sprintf("%s %s (%s)",
			strings.ToUpper(api.Method), api.PathPattern, api.Name))
	}
	return suggestions
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
