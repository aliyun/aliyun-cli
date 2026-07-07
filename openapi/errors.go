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
	msg := fmt.Sprintf("'%s' is not a valid built-in product or external product plugin. See `aliyun --help`.", e.Code)
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

const apiNamingRulesHintKebabSuffix = "plugin command of external product plugins"

func apiNamingRulesHint(pluginInstalled bool, pluginName string, includeMethodPath bool) string {
	kebabLine := apiNamingRulesHintKebabSuffix
	if !pluginInstalled && pluginName != "" {
		kebabLine = fmt.Sprintf("%s (run aliyun plugin install --name %s first)", apiNamingRulesHintKebabSuffix, pluginName)
	}
	msg := fmt.Sprintf(`Naming rules in Aliyun CLI:
  · CamelCase     → OpenAPI API name of built-in products
  · kebab-case    → %s`, kebabLine)
	if includeMethodPath {
		msg += "\n  · METHOD + path → REST shortcut for built-in restful products"
	}
	return msg
}

// GetSuggestions scans built-in ApiNames and installed plugin CmdNames (three-way scan subset).
type InvalidApiOrCmdNotFoundError struct {
	Name         string
	product      *meta.Product
	pluginName   string
	localPlugin  *plugin.LocalPlugin
	builtinNames map[string]struct{}
	pluginCmds   map[string]struct{}
}

func newApiOrCmdNotFoundError(
	product *meta.Product,
	name string,
	localPlugin *plugin.LocalPlugin,
	pluginName string,
) *InvalidApiOrCmdNotFoundError {
	e := &InvalidApiOrCmdNotFoundError{
		Name:         name,
		product:      product,
		pluginName:   pluginName,
		localPlugin:  localPlugin,
		builtinNames: make(map[string]struct{}),
		pluginCmds:   make(map[string]struct{}),
	}
	for _, apiName := range product.ApiNames {
		e.builtinNames[apiName] = struct{}{}
	}
	if localPlugin != nil {
		for _, cmd := range localPlugin.CmdNames {
			e.pluginCmds[cmd] = struct{}{}
		}
	}
	return e
}

func (e *InvalidApiOrCmdNotFoundError) Error() string {
	msg := fmt.Sprintf("'%s' is not a valid api/command for product '%s'.",
		e.Name, e.product.GetLowerCode())
	pluginInstalled := e.localPlugin != nil
	msg += "\n\n" + apiNamingRulesHint(pluginInstalled, e.pluginName, true)
	if !pluginInstalled && e.pluginName != "" {
		msg += fmt.Sprintf(
			"\n\nExternal product plugins must be installed before kebab-case commands work:\n"+
				"  aliyun plugin install --name %s\n",
			e.pluginName)
	}
	msg += fmt.Sprintf("\nTo see parameters of a plugin command:\n  aliyun %s <kebab-cmd> --help",
		e.product.GetLowerCode())
	return msg
}

func (e *InvalidApiOrCmdNotFoundError) GetSuggestions() []string {
	sr := cli.NewSuggester(e.Name, 2)
	lowerProduct := e.product.GetLowerCode()

	for apiName := range e.builtinNames {
		sr.UnifyApply(apiName)
	}
	for cmd := range e.pluginCmds {
		sr.UnifyApply(cmd)
	}

	seen := make(map[string]struct{})
	suggestions := make([]string, 0)
	for _, candidate := range sr.GetResults() {
		if _, ok := e.builtinNames[candidate]; ok {
			line := fmt.Sprintf("aliyun %s %s  [built-in OpenAPI]", lowerProduct, candidate)
			if _, dup := seen[line]; dup {
				continue
			}
			seen[line] = struct{}{}
			suggestions = append(suggestions, line)
			continue
		}
		if _, ok := e.pluginCmds[candidate]; ok {
			line := fmt.Sprintf("aliyun %s %s  [plugin]", lowerProduct, candidate)
			if _, dup := seen[line]; dup {
				continue
			}
			seen[line] = struct{}{}
			suggestions = append(suggestions, line)
		}
	}
	return suggestions
}

func pluginCmdMatches(apiName string, lp *plugin.LocalPlugin) bool {
	if lp == nil {
		return false
	}
	for _, cmd := range lp.CmdNames {
		if cmd == apiName {
			return true
		}
	}
	return false
}

func pluginCmdsMatchingApiName(apiName string, lp *plugin.LocalPlugin) []string {
	if lp == nil {
		return nil
	}
	sr := cli.NewSuggester(apiName, 2)
	for _, cmd := range lp.CmdNames {
		sr.UnifyApply(cmd)
	}
	return sr.GetResults()
}

// InvalidRestfulPathError is returned when aliyun <product> <METHOD> <path> cannot be resolved.
// If the path exists for other HTTP methods, GetSuggestions lists them.
type InvalidRestfulPathError struct {
	Method      string
	Path        string
	Product     *meta.Product
	matches     []meta.Api
	pathMissing bool
	pluginName  string
	localPlugin *plugin.LocalPlugin
}

func newInvalidRestfulPathError(
	product *meta.Product,
	method, path string,
	matches []meta.Api,
	pluginName string,
	localPlugin *plugin.LocalPlugin,
) *InvalidRestfulPathError {
	return &InvalidRestfulPathError{
		Method:      method,
		Path:        path,
		Product:     product,
		matches:     matches,
		pathMissing: len(matches) == 0,
		pluginName:  pluginName,
		localPlugin: localPlugin,
	}
}

func restfulInvokeHint(productCode string, pluginInstalled bool, pluginName string, includeMethodPath bool) string {
	lowerProduct := strings.ToLower(productCode)
	var msg string
	if includeMethodPath {
		msg = fmt.Sprintf(`

Invocation options for restful product '%s':
  · ApiName form  : aliyun %s <ApiName> --parameter1 value1 ...
  · METHOD + path : aliyun %s [GET|PUT|POST|DELETE] <path> ...
  · kebab-case    : aliyun %s <kebab-cmd> --parameter1 value1 ... from external product plugins`, lowerProduct, lowerProduct, lowerProduct, lowerProduct)
	} else {
		msg = fmt.Sprintf(`

Invocation options for restful product '%s':
  · ApiName form  : aliyun %s <ApiName> --parameter1 value1 ...
  · kebab-case    : aliyun %s <kebab-cmd> --parameter1 value1 ... from external product plugins`, lowerProduct, lowerProduct, lowerProduct)
	}
	if !pluginInstalled && pluginName != "" {
		msg += fmt.Sprintf(" (run aliyun plugin install --name %s first)", pluginName)
	}
	return msg
}

func (e *InvalidRestfulPathError) Error() string {
	lowerProduct := e.Product.GetLowerCode()
	var msg string
	if e.pathMissing {
		msg = fmt.Sprintf("can not find api by path %s", e.Path)
		msg += fmt.Sprintf("\nUse `aliyun %s --help` to confirm the correct ApiName and METHOD+path for this product.", lowerProduct)
	} else {
		msg = fmt.Sprintf("can not find api by path %s with method %s",
			e.Path, strings.ToUpper(e.Method))
	}
	pluginInstalled := e.localPlugin != nil
	msg += restfulInvokeHint(lowerProduct, pluginInstalled, e.pluginName, true)
	return msg
}

func (e *InvalidRestfulPathError) GetSuggestions() []string {
	lowerProduct := e.Product.GetLowerCode()
	seen := make(map[string]struct{})
	suggestions := make([]string, 0)

	add := func(line string) {
		if _, dup := seen[line]; dup {
			return
		}
		seen[line] = struct{}{}
		suggestions = append(suggestions, line)
	}

	for _, api := range e.matches {
		add(fmt.Sprintf("aliyun %s %s  [built-in OpenAPI ApiName]", lowerProduct, api.Name))
		add(fmt.Sprintf("aliyun %s %s %s [built-in RESTful Style for %s]",
			lowerProduct, strings.ToUpper(api.Method), api.PathPattern, api.Name))
		for _, cmd := range pluginCmdsMatchingApiName(api.Name, e.localPlugin) {
			add(fmt.Sprintf("aliyun %s %s  [product plugin command]", lowerProduct, cmd))
		}
	}
	return suggestions
}

// RestfulBroadPathError is returned when aliyun <product> <METHOD> / matches an API
// but root path "/" is too broad for OpenAPI invocation.
type RestfulBroadPathError struct {
	Method      string
	Path        string
	Product     *meta.Product
	api         meta.Api
	pluginName  string
	localPlugin *plugin.LocalPlugin
}

func newRestfulBroadPathError(
	product *meta.Product,
	method, path string,
	api meta.Api,
	pluginName string,
	localPlugin *plugin.LocalPlugin,
) *RestfulBroadPathError {
	return &RestfulBroadPathError{
		Method:      method,
		Path:        path,
		Product:     product,
		api:         api,
		pluginName:  pluginName,
		localPlugin: localPlugin,
	}
}

func (e *RestfulBroadPathError) Error() string {
	lowerProduct := e.Product.GetLowerCode()
	msg := fmt.Sprintf("path %q is too broad for METHOD+path invocation with %s.",
		e.Path, strings.ToUpper(e.Method))
	msg += fmt.Sprintf("\nUse a specific ApiName instead of the root path %q.", e.Path)
	msg += fmt.Sprintf("\nUse `aliyun %s --help` to confirm the correct ApiName for this product.", lowerProduct)
	pluginInstalled := e.localPlugin != nil
	msg += restfulInvokeHint(lowerProduct, pluginInstalled, e.pluginName, false)
	return msg
}

func (e *RestfulBroadPathError) GetSuggestions() []string {
	lowerProduct := e.Product.GetLowerCode()
	suggestions := make([]string, 0, 3)

	add := func(line string) {
		for _, existing := range suggestions {
			if existing == line {
				return
			}
		}
		suggestions = append(suggestions, line)
	}

	add(fmt.Sprintf("aliyun %s %s  [built-in OpenAPI ApiName]", lowerProduct, e.api.Name))
	if e.api.PathPattern != "" && e.api.PathPattern != "/" {
		add(fmt.Sprintf("aliyun %s %s %s [built-in RESTful Style for %s]",
			lowerProduct, strings.ToUpper(e.api.Method), e.api.PathPattern, e.api.Name))
	}
	for _, cmd := range pluginCmdsMatchingApiName(e.api.Name, e.localPlugin) {
		add(fmt.Sprintf("aliyun %s %s  [product plugin command]", lowerProduct, cmd))
	}
	return suggestions
}

// RpcMethodPathError is returned when aliyun <rpc-product> <METHOD> <path> is used.
type RpcMethodPathError struct {
	Method      string
	Path        string
	Product     *meta.Product
	pluginName  string
	localPlugin *plugin.LocalPlugin
}

func newRpcMethodPathError(
	product *meta.Product,
	method, path string,
	pluginName string,
	localPlugin *plugin.LocalPlugin,
) *RpcMethodPathError {
	return &RpcMethodPathError{
		Method:      method,
		Path:        path,
		Product:     product,
		pluginName:  pluginName,
		localPlugin: localPlugin,
	}
}

func (e *RpcMethodPathError) Error() string {
	lowerProduct := e.Product.GetLowerCode()
	msg := fmt.Sprintf("'%s' is an RPC product and does not accept METHOD + path form (got %s %s).",
		lowerProduct, strings.ToUpper(e.Method), e.Path)
	msg += fmt.Sprintf("\nUse `aliyun %s <ApiName>` instead. See `aliyun %s --help` for available ApiNames.",
		lowerProduct, lowerProduct)
	pluginInstalled := e.localPlugin != nil
	msg += "\n\n" + apiNamingRulesHint(pluginInstalled, e.pluginName, false)
	return msg
}

func (e *RpcMethodPathError) GetSuggestions() []string {
	return []string{fmt.Sprintf("aliyun %s --help", e.Product.GetLowerCode())}
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
