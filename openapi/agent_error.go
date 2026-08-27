// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"sort"
	"strings"
	"unicode"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	runtime "github.com/aliyun/aliyun-openapi-runtime/runtime"
)

type credentialConfigurationError struct {
	Err error
}

func (e *credentialConfigurationError) Error() string { return e.Err.Error() }

func (e *credentialConfigurationError) Unwrap() error { return e.Err }

// RecoverySearchRequest contains only metadata identities and keywords. It
// deliberately carries no user parameter, header, body, or credential value.
type RecoverySearchRequest struct {
	Product string
	API     string
	Version string
	Section string
	Style   string
	Keyword string
}

// RecoverySearchValidator confirms that a provider supports the proposed
// search and that the search has at least one real result. Nil means the
// current provider/path cannot validate search and forces ordinary Help.
type RecoverySearchValidator func(RecoverySearchRequest) bool

func normalizeAgentError(err error, args []string) error {
	return normalizeAgentErrorWithSearch(err, args, nil)
}

func normalizeAgentErrorWithSearch(err error, args []string, validate RecoverySearchValidator) error {
	if err == nil {
		return nil
	}

	var existing *cli.AgentError
	if errors.As(err, &existing) {
		return existing
	}

	context := newRecoveryContext(args)

	var unknownFlag *argparser.UnknownFlagError
	if errors.As(err, &unknownFlag) {
		suggestions := flagSuggestions(unknownFlag.Flag, unknownFlag.Known)
		return parameterSearchAgentError(err, unknownFlag.Error(), suggestions, strings.TrimLeft(firstString(suggestions), "-"), context, validate)
	}

	var missing *runtime.MissingRequiredError
	if errors.As(err, &missing) {
		return newLocalAgentError(err, missing.Error(), nil, cli.AgentErrorRecovery{
			Action:  "inspect_request_help",
			Command: context.requestHelpCommand(),
			Hint:    "Inspect the complete request help and provide every required parameter.",
		})
	}

	var legacyDocRequired *LegacyDocRequiredError
	if errors.As(err, &legacyDocRequired) {
		return newLocalAgentError(err, legacyDocRequired.Error(), nil, cli.AgentErrorRecovery{
			Action:  "inspect_request_help",
			Command: context.requestHelpCommand(),
			Hint:    "Inspect the complete request help and provide every required parameter.",
		})
	}

	var legacyMissingRequired *LegacyMissingRequiredError
	if errors.As(err, &legacyMissingRequired) {
		return newLocalAgentError(err, legacyMissingRequired.Error(), nil, cli.AgentErrorRecovery{
			Action:  "inspect_request_help",
			Command: context.requestHelpCommand(),
			Hint:    "Inspect the complete request help and provide every required parameter.",
		})
	}

	var invalidParameter *InvalidParameterError
	if errors.As(err, &invalidParameter) {
		parameterContext := context.withProductAPI(invalidParameter.ProductCode, invalidParameter.ApiName)
		suggestions := invalidParameter.AgentSuggestions()
		return parameterSearchAgentError(err, invalidParameter.AgentMessage(), suggestions,
			strings.TrimLeft(firstString(suggestions), "-"), parameterContext, validate)
	}

	var invalidAPI *InvalidApiError
	if errors.As(err, &invalidAPI) {
		apiContext := context
		if invalidAPI.product != nil {
			apiContext = context.withProductAPI(invalidAPI.product.GetLowerCode(), invalidAPI.Name)
		}
		return unknownAPIAgentError(err, invalidAPI.AgentMessage(), apiCandidateForms(invalidAPI.AgentSuggestions()), apiContext, validate)
	}

	var invalidUnifiedAPI *InvalidUnifiedApiError
	if errors.As(err, &invalidUnifiedAPI) {
		apiContext := context
		if invalidUnifiedAPI.product != nil {
			apiContext = context.withProductAPI(invalidUnifiedAPI.product.GetLowerCode(), invalidUnifiedAPI.Name)
		}
		return unknownAPIAgentError(err, invalidUnifiedAPI.AgentMessage(), apiCandidateForms(invalidUnifiedAPI.AgentSuggestions()), apiContext, validate)
	}

	var invalidBaseline *InvalidBaselineCommandError
	if errors.As(err, &invalidBaseline) {
		apiContext := context.withProductAPI(invalidBaseline.Product, invalidBaseline.Command)
		message := fmt.Sprintf("%q is not a valid api.", invalidBaseline.Command)
		return unknownAPIAgentError(err, message, nil, apiContext, validate)
	}

	var invalidProduct *InvalidProductError
	if errors.As(err, &invalidProduct) {
		return unknownProductAgentError(err, invalidProduct.AgentMessage(), invalidProduct.AgentSuggestions(), validate)
	}

	var invalidProductOrPlugin *InvalidProductOrPluginError
	if errors.As(err, &invalidProductOrPlugin) {
		return unknownProductAgentError(err, invalidProductOrPlugin.AgentMessage(), invalidProductOrPlugin.AgentSuggestions(), validate)
	}

	var invalidFlag *cli.InvalidFlagError
	if errors.As(err, &invalidFlag) {
		return newLocalAgentError(err, invalidFlag.AgentMessage(), invalidFlag.AgentSuggestions(), cli.AgentErrorRecovery{
			Action:  "inspect_command_help",
			Command: invalidFlag.AgentHelpCommand(),
			Hint:    "Inspect the available flags for this command.",
		})
	}

	var invalidCommand *cli.InvalidCommandError
	if errors.As(err, &invalidCommand) {
		return newLocalAgentError(err, invalidCommand.Error(), stableStrings(invalidCommand.GetSuggestions()), cli.AgentErrorRecovery{
			Action:  "search_command",
			Command: context.parentHelpCommand(invalidCommand.Name),
			Hint:    "Inspect commands under the current parent.",
		})
	}

	var invalidArgument *argparser.InvalidArgumentError
	if errors.As(err, &invalidArgument) {
		return invalidArgumentAgentError(err, invalidArgument.Error(), invalidArgument.Flag,
			invalidArgument.Parameter, invalidArgument.FieldPath, context, validate)
	}

	var legacyInvalidArgument *InvalidArgumentError
	if errors.As(err, &legacyInvalidArgument) {
		return invalidArgumentAgentError(err, legacyInvalidArgument.Error(), legacyInvalidArgument.Flag,
			legacyInvalidArgument.Parameter, legacyInvalidArgument.FieldPath, context, validate)
	}

	var invalidOptions *engine.InvalidOptionCombinationError
	if errors.As(err, &invalidOptions) {
		return optionCombinationAgentError(err, invalidOptions.Error(), invalidOptions.Options, context)
	}

	var legacyInvalidOptions *InvalidOptionCombinationError
	if errors.As(err, &legacyInvalidOptions) {
		return optionCombinationAgentError(err, legacyInvalidOptions.Error(), legacyInvalidOptions.Options, context)
	}

	var invalidHeader *engine.InvalidHeaderError
	if errors.As(err, &invalidHeader) {
		return fixedSearchAgentError(err, invalidHeader.Error(), "inspect_header_usage", "header",
			"Inspect header usage and pass each header as Name=Value.", context, validate)
	}

	var legacyInvalidHeader *InvalidHeaderError
	if errors.As(err, &legacyInvalidHeader) {
		return fixedSearchAgentError(err, legacyInvalidHeader.Error(), "inspect_header_usage", "header",
			"Inspect header usage and pass each header as Name=Value.", context, validate)
	}

	var invalidBodyFile *engine.InvalidBodyFileError
	if errors.As(err, &invalidBodyFile) {
		return fixedSearchAgentError(err, invalidBodyFile.Error(), "fix_body_file", "body-file",
			"Check that --body-file points to a readable file.", context, validate)
	}

	var legacyInvalidBodyFile *InvalidBodyFileError
	if errors.As(err, &legacyInvalidBodyFile) {
		return fixedSearchAgentError(err, legacyInvalidBodyFile.Error(), "fix_body_file", "body-file",
			"Check that --body-file points to a readable file.", context, validate)
	}

	// This is intentionally an allowlist. Canonical constraints, credentials,
	// SDK/Tea/server/network failures, plugins, postprocessing, safety-policy,
	// Machine Help, corrupt metadata, broad UsageError, and untyped errors all
	// retain their original rendering and identity.
	return err
}

func newLocalAgentError(cause error, message string, suggestions []string, recovery cli.AgentErrorRecovery) error {
	return cli.NewAgentError(cli.AgentErrorEnvelope{
		Message:    nonEmptyMessage(message, cause),
		DidYouMean: stableStrings(suggestions),
		Recovery:   recovery,
	}, cause)
}

func unknownProductAgentError(cause error, message string, suggestions []string, validate RecoverySearchValidator) error {
	suggestions = stableStrings(suggestions)
	candidate := firstString(suggestions)
	recovery := cli.AgentErrorRecovery{
		Action:  "search_product",
		Command: "aliyun help",
		Hint:    "Inspect the available products.",
	}
	if candidate != "" {
		recovery.Hint = fmt.Sprintf("Search products related to %s.", candidate)
		request := RecoverySearchRequest{Keyword: candidate}
		if validate != nil && validate(request) && safeCommandToken(candidate) {
			recovery.Command = "aliyun help --cli-search " + candidate
		}
	}
	return newLocalAgentError(cause, message, suggestions, recovery)
}

func unknownAPIAgentError(cause error, message string, suggestions []string, context recoveryContext, validate RecoverySearchValidator) error {
	keyword := resourceKeyword(firstString(suggestions))
	recovery := cli.AgentErrorRecovery{
		Action:  "search_api",
		Command: context.productHelpCommand(),
		Hint:    "Inspect the available APIs for this product.",
	}
	if keyword != "" {
		recovery.Hint = fmt.Sprintf("Search APIs related to %s.", keyword)
		request := context.searchRequest("", "", keyword)
		if validate != nil && validate(request) {
			recovery.Command = context.productSearchCommand(keyword)
		}
	}
	return newLocalAgentError(cause, message, suggestions, recovery)
}

func parameterSearchAgentError(cause error, message string, suggestions []string, keyword string,
	context recoveryContext, validate RecoverySearchValidator) error {
	keyword = strings.TrimLeft(strings.TrimSpace(keyword), "-")
	recovery := cli.AgentErrorRecovery{
		Action:  "search_parameter",
		Command: context.requestHelpCommand(),
		Hint:    "Inspect the complete request help and correct the parameter or flag.",
	}
	if keyword != "" {
		recovery.Hint = fmt.Sprintf("Search request parameters related to %s.", keyword)
		if validate != nil && validate(context.searchRequest("request", context.api, keyword)) {
			recovery.Command = context.requestSearchCommand(keyword)
		}
	}
	return newLocalAgentError(cause, message, suggestions, recovery)
}

func invalidArgumentAgentError(cause error, message, flag, parameter, fieldPath string,
	context recoveryContext, validate RecoverySearchValidator) error {
	keyword := firstNonEmpty(strings.TrimLeft(flag, "-"), parameter, fieldPath)
	recovery := cli.AgentErrorRecovery{
		Action:  "inspect_request_help",
		Command: context.requestHelpCommand(),
		Hint:    "Inspect the complete request help and correct the argument syntax or type.",
	}
	if keyword != "" {
		recovery.Hint = fmt.Sprintf("Inspect request help for %s and correct its syntax or type.", keyword)
		if validate != nil && validate(context.searchRequest("request", context.api, keyword)) {
			recovery.Command = context.requestSearchCommand(keyword)
		}
	}
	return newLocalAgentError(cause, message, nil, recovery)
}

func optionCombinationAgentError(cause error, message string, options []string, context recoveryContext) error {
	options = stableStrings(options)
	hint := "Remove one of the conflicting options."
	if len(options) > 0 {
		hint = fmt.Sprintf("Remove one of the conflicting options: %s.", strings.Join(options, ", "))
	}
	return newLocalAgentError(cause, message, nil, cli.AgentErrorRecovery{
		Action:  "fix_option_combination",
		Command: context.requestHelpCommand(),
		Hint:    hint,
	})
}

func fixedSearchAgentError(cause error, message, action, keyword, hint string,
	context recoveryContext, validate RecoverySearchValidator) error {
	command := context.requestHelpCommand()
	if validate != nil && validate(context.searchRequest("request", context.api, keyword)) {
		command = context.requestSearchCommand(keyword)
	}
	return newLocalAgentError(cause, message, nil, cli.AgentErrorRecovery{
		Action:  action,
		Command: command,
		Hint:    hint,
	})
}

type recoveryContext struct {
	args        []string
	product     string
	api         string
	version     string
	versionFlag string
	style       string
}

func newRecoveryContext(args []string) recoveryContext {
	context := recoveryContext{args: append([]string(nil), args...)}
	if len(args) > 0 && safeCommandToken(args[0]) {
		context.product = args[0]
	}
	if len(args) > 1 && safeCommandToken(args[1]) {
		context.api = args[1]
		context.style = commandStyle(args[1])
	}
	context.versionFlag, context.version = explicitVersion(args)
	return context
}

func (c recoveryContext) withProductAPI(product, api string) recoveryContext {
	if safeCommandToken(product) {
		c.product = strings.ToLower(product)
	}
	if safeCommandToken(api) {
		c.api = api
		c.style = commandStyle(api)
	}
	return c
}

func (c recoveryContext) searchRequest(section, api, keyword string) RecoverySearchRequest {
	return RecoverySearchRequest{
		Product: c.product,
		API:     api,
		Version: c.version,
		Section: section,
		Style:   c.style,
		Keyword: keyword,
	}
}

func (c recoveryContext) productHelpCommand() string {
	if c.product == "" {
		return "aliyun help"
	}
	return "aliyun help " + c.product + c.versionSuffix()
}

func (c recoveryContext) productSearchCommand(keyword string) string {
	if c.product == "" || !safeCommandToken(keyword) {
		return c.productHelpCommand()
	}
	return "aliyun help " + c.product + c.versionSuffix() + " --cli-search " + keyword
}

func (c recoveryContext) requestHelpCommand() string {
	if c.product == "" || c.api == "" {
		return c.productHelpCommand()
	}
	return "aliyun help " + c.product + " " + c.api + c.versionSuffix() + " --cli-section request"
}

func (c recoveryContext) requestSearchCommand(keyword string) string {
	if !safeCommandToken(keyword) {
		return c.requestHelpCommand()
	}
	return c.requestHelpCommand() + " --cli-search " + keyword
}

func (c recoveryContext) parentHelpCommand(invalidName string) string {
	limit := len(c.args)
	if limit > 0 && c.args[limit-1] == invalidName {
		limit--
	}
	parts := make([]string, 0, limit)
	for _, value := range c.args[:limit] {
		if strings.HasPrefix(value, "-") {
			break
		}
		if !safeCommandToken(value) {
			break
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return "aliyun help"
	}
	return "aliyun help " + strings.Join(parts, " ")
}

func (c recoveryContext) versionSuffix() string {
	if c.version == "" || c.versionFlag == "" {
		return ""
	}
	return " " + c.versionFlag + " " + c.version
}

func explicitVersion(args []string) (string, string) {
	for index, arg := range args {
		for _, flag := range []string{"--version", "--api-version"} {
			if arg == flag {
				if index+1 < len(args) && safeCommandToken(args[index+1]) {
					return flag, args[index+1]
				}
				return "", ""
			}
			if strings.HasPrefix(arg, flag+"=") {
				value := strings.TrimPrefix(arg, flag+"=")
				if safeCommandToken(value) {
					return flag, value
				}
				return "", ""
			}
		}
	}
	return "", ""
}

func safeCommandToken(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func commandStyle(api string) string {
	if api != "" && (api == strings.ToLower(api) || strings.Contains(api, "-")) {
		return "kebab"
	}
	if api != "" {
		return "pascal"
	}
	return ""
}

func resourceKeyword(candidate string) string {
	candidate = strings.TrimLeft(strings.TrimSpace(candidate), "-")
	if candidate == "" {
		return ""
	}
	if strings.Contains(candidate, "-") {
		candidate = kebabToPascal(candidate)
	}
	for _, prefix := range []string{
		"Describe", "Create", "Delete", "Update", "Modify", "Remove", "Search", "Disable", "Enable",
		"List", "Query", "Start", "Stop", "Get", "Put", "Set", "Add",
	} {
		if strings.HasPrefix(candidate, prefix) && len(candidate) > len(prefix) {
			return candidate[len(prefix):]
		}
	}
	return candidate
}

func kebabToPascal(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == '-' || r == '_' })
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		builder.WriteRune(unicode.ToUpper(runes[0]))
		builder.WriteString(string(runes[1:]))
	}
	return builder.String()
}

func apiCandidateForms(candidates []string) []string {
	forms := make([]string, 0, len(candidates)*2)
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		forms = append(forms, candidate)
		if candidate != strings.ToLower(candidate) && !strings.Contains(candidate, "-") {
			forms = append(forms, apiNameToKebab(candidate))
		}
	}
	return stableStrings(forms)
}

func nonEmptyMessage(message string, cause error) string {
	if strings.TrimSpace(message) != "" {
		return message
	}
	if cause != nil && strings.TrimSpace(cause.Error()) != "" {
		return cause.Error()
	}
	return "CLI request is invalid."
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func flagSuggestions(input string, candidates []string) []string {
	return closeSuggestions(input, candidates, true)
}

func apiSuggestions(input string, candidates []string) []string {
	return closeSuggestions(input, candidates, false)
}

func closeSuggestions(input string, candidates []string, flags bool) []string {
	input = strings.TrimLeft(input, "-")
	suggester := cli.NewSuggester(input, cli.DefaultSuggestDistance)
	for _, candidate := range candidates {
		candidate = strings.TrimLeft(strings.TrimSpace(candidate), "-")
		if candidate != "" {
			suggester.UnifyApply(candidate)
		}
	}
	results := suggester.GetResults()
	if flags {
		for index := range results {
			results[index] = "--" + strings.TrimLeft(results[index], "-")
		}
	}
	return stableStrings(results)
}

func stableStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
