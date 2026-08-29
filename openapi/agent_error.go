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
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	runtime "github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/alibabacloud-go/tea/tea"
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
		return parameterSearchAgentError(err, unknownFlag.Error(), suggestions, unknownFlag.Flag, context, validate)
	}

	var missing *runtime.MissingRequiredError
	if errors.As(err, &missing) {
		return missingRequiredAgentError(err, missingRequiredAgentMessage(missing), missing.Flags, context, validate)
	}

	var legacyDocRequired *LegacyDocRequiredError
	if errors.As(err, &legacyDocRequired) {
		return missingRequiredAgentError(err, legacyDocRequired.Error(), legacyDocRequired.Flags, context, validate)
	}

	var legacyMissingRequired *LegacyMissingRequiredError
	if errors.As(err, &legacyMissingRequired) {
		return missingRequiredAgentError(err, legacyMissingRequired.Error(), legacyMissingRequiredFlagNames(legacyMissingRequired), context, validate)
	}

	var runtimeConstraint *runtime.ConstraintViolationError
	if errors.As(err, &runtimeConstraint) {
		return constraintViolationAgentError(err, runtimeConstraintFacts(runtimeConstraint), context, validate)
	}

	var legacyConstraint *ConstraintViolationError
	if errors.As(err, &legacyConstraint) {
		return constraintViolationAgentError(err, legacyConstraintFacts(legacyConstraint), context, validate)
	}

	var invalidParameter *InvalidParameterError
	if errors.As(err, &invalidParameter) {
		parameterContext := context.withProductAPI(invalidParameter.ProductCode, invalidParameter.ApiName)
		suggestions := invalidParameter.AgentSuggestions()
		return parameterSearchAgentError(err, invalidParameter.AgentMessage(), suggestions,
			invalidParameter.Name, parameterContext, validate)
	}

	var invalidAPI *InvalidApiError
	if errors.As(err, &invalidAPI) {
		apiContext := context
		if invalidAPI.product != nil {
			apiContext = context.withProductAPI(invalidAPI.product.GetLowerCode(), invalidAPI.Name)
		}
		return unknownAPIAgentError(err, invalidAPI.AgentMessage(), apiCandidateFormsForStyle(invalidAPI.AgentSuggestions(), apiContext.style), apiContext, validate)
	}

	var invalidUnifiedAPI *InvalidUnifiedApiError
	if errors.As(err, &invalidUnifiedAPI) {
		apiContext := context
		if invalidUnifiedAPI.product != nil {
			apiContext = context.withProductAPI(invalidUnifiedAPI.product.GetLowerCode(), invalidUnifiedAPI.Name)
		}
		return unknownAPIAgentError(err, invalidUnifiedAPI.AgentMessage(), apiCandidateFormsForStyle(invalidUnifiedAPI.AgentSuggestions(), apiContext.style), apiContext, validate)
	}

	var invalidBaseline *InvalidBaselineCommandError
	if errors.As(err, &invalidBaseline) {
		apiContext := context.withProductAPI(invalidBaseline.Product, invalidBaseline.Command)
		message := fmt.Sprintf("%q is not a valid api.", invalidBaseline.Command)
		return unknownAPIAgentError(err, message, nil, apiContext, validate)
	}

	var unknownCommand *engine.UnknownCommandError
	if errors.As(err, &unknownCommand) {
		apiContext := context.withProductAPI(unknownCommand.Product, unknownCommand.Command)
		message := fmt.Sprintf("%q is not a valid api.", unknownCommand.Command)
		return unknownAPIAgentError(err, message, nil, apiContext, validate)
	}

	var invalidProduct *InvalidProductError
	if errors.As(err, &invalidProduct) {
		return unknownProductAgentError(err, invalidProduct.AgentMessage(), invalidProduct.AgentSuggestions(), invalidProduct.Code, validate)
	}

	var invalidProductOrPlugin *InvalidProductOrPluginError
	if errors.As(err, &invalidProductOrPlugin) {
		return unknownProductAgentError(err, invalidProductOrPlugin.AgentMessage(), invalidProductOrPlugin.AgentSuggestions(), invalidProductOrPlugin.Code, validate)
	}

	var invalidFlag *cli.InvalidFlagError
	if errors.As(err, &invalidFlag) {
		return unknownHostFlagAgentError(err, invalidFlag.AgentMessage(), invalidFlag.AgentSuggestions(),
			invalidFlag.Flag, invalidFlag.AgentHelpCommand(), validate)
	}

	var invalidCommand *cli.InvalidCommandError
	if errors.As(err, &invalidCommand) {
		return unknownCommandAgentError(err, invalidCommand.Error(), invalidCommand.GetSuggestions(), invalidCommand.Name, context, validate)
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

	var invalidHelpOptions *cli.HelpOptionError
	if errors.As(err, &invalidHelpOptions) {
		return helpOptionAgentError(err, invalidHelpOptions)
	}

	var invalidOptions *engine.InvalidOptionCombinationError
	if errors.As(err, &invalidOptions) {
		return optionCombinationAgentError(err, invalidOptions.Error(), invalidOptions.Options)
	}

	var legacyInvalidOptions *InvalidOptionCombinationError
	if errors.As(err, &legacyInvalidOptions) {
		return optionCombinationAgentError(err, legacyInvalidOptions.Error(), legacyInvalidOptions.Options)
	}

	var invalidHeader *engine.InvalidHeaderError
	if errors.As(err, &invalidHeader) {
		return fixedParameterHelpAgentError(err, "invalid --header value: expected Name=Value", "inspect_header_usage", "header",
			"Inspect header usage and pass each header as Name=Value.", context)
	}

	var legacyInvalidHeader *InvalidHeaderError
	if errors.As(err, &legacyInvalidHeader) {
		return fixedParameterHelpAgentError(err, "invalid --header value: expected Name=Value", "inspect_header_usage", "header",
			"Inspect header usage and pass each header as Name=Value.", context)
	}

	var invalidBodyFile *engine.InvalidBodyFileError
	if errors.As(err, &invalidBodyFile) {
		return fixedParameterHelpAgentError(err, "unable to read --body-file", "fix_body_file", "body-file",
			"Check that --body-file points to a readable file.", context)
	}

	var legacyInvalidBodyFile *InvalidBodyFileError
	if errors.As(err, &legacyInvalidBodyFile) {
		return fixedParameterHelpAgentError(err, "unable to read --body-file", "fix_body_file", "body-file",
			"Check that --body-file points to a readable file.", context)
	}

	var externalReject *argparser.ExternalFlagRejectError
	if errors.As(err, &externalReject) {
		return externalFlagRejectAgentError(err, externalReject, context)
	}

	// Client-side endpoint/region resolution failures carry no server error
	// code, so they are handled before the remote server-error branch (which
	// bails on ErrorWithTip-wrapped errors).
	var endpointErr *meta.InvalidEndpointError
	if errors.As(err, &endpointErr) {
		return endpointAgentError(err, endpointErr, context)
	}

	if normalized := normalizeServerAgentError(err, context); normalized != nil {
		return normalized
	}

	// This is intentionally an allowlist. Credentials, network
	// failures, plugins, postprocessing, safety-policy, Machine Help, corrupt
	// metadata, broad UsageError, and untyped errors all retain their original
	// rendering and identity.
	return err
}

// normalizeServerAgentError wraps remote server errors from either runtime
// (kebab dara/Tea errors and legacy old-SDK ServerError) into one envelope so
// the message prefix and exit code no longer differ by command style. Errors
// already carrying a CLI tip (estimate-cost guidance) keep their original
// rendering, and network/credential failures are not SDK errors and never
// reach this branch.
func normalizeServerAgentError(err error, context recoveryContext) error {
	var withTip cli.ErrorWithTip
	if errors.As(err, &withTip) {
		return nil
	}
	var teaErr *tea.SDKError
	if errors.As(err, &teaErr) {
		message, messageRequestID := cleanTeaServerMessage(tea.StringValue(teaErr.Message))
		requestID := teaRequestID(teaErr)
		if requestID == "" {
			requestID = messageRequestID
		}
		facts := serverErrorFacts{
			code:       tea.StringValue(teaErr.Code),
			message:    message,
			requestID:  requestID,
			statusCode: tea.IntValue(teaErr.StatusCode),
		}
		if facts.code == "" && facts.message == "" {
			return nil
		}
		return serverAgentError(err, facts, context)
	}
	var serverErr *sdkerrors.ServerError
	if errors.As(err, &serverErr) {
		facts := serverErrorFacts{
			code:       serverErr.ErrorCode(),
			message:    serverErr.Message(),
			requestID:  serverErr.RequestId(),
			statusCode: serverErr.HttpStatus(),
		}
		if facts.code == "" && facts.message == "" {
			return nil
		}
		return serverAgentError(err, facts, context)
	}
	return nil
}

// teaRequestID best-effort extracts a RequestId from the Tea error's Data JSON
// payload; the kebab/Tea SDK does not expose it as a dedicated field.
func teaRequestID(err *tea.SDKError) string {
	data := tea.StringValue(err.Data)
	if strings.TrimSpace(data) == "" {
		return ""
	}
	var payload map[string]any
	if json.Unmarshal([]byte(data), &payload) != nil {
		return ""
	}
	if requestID, ok := payload["RequestId"].(string); ok {
		return requestID
	}
	if requestID, ok := payload["requestId"].(string); ok {
		return requestID
	}
	return ""
}

// darabonba-openapi builds Tea server-error messages as
// "code: <status>, <message> request id: <id>" (client.go), duplicating the
// structured status_code/request_id envelope fields. These patterns strip that
// envelope so the message matches the clean legacy old-SDK form.
var (
	teaServerErrorPrefixPattern    = regexp.MustCompile(`^code:\s*\d+,\s*`)
	teaServerErrorRequestIDPattern = regexp.MustCompile(`\s*request id:\s*(\S+)\s*$`)
)

// cleanTeaServerMessage strips the status prefix and trailing request id from
// a Tea server-error message, returning the cleaned message and the extracted
// request id (empty when absent).
func cleanTeaServerMessage(message string) (string, string) {
	message = teaServerErrorPrefixPattern.ReplaceAllString(message, "")
	requestID := ""
	if match := teaServerErrorRequestIDPattern.FindStringSubmatch(message); match != nil {
		requestID = match[1]
		message = teaServerErrorRequestIDPattern.ReplaceAllString(message, "")
	}
	return strings.TrimSpace(message), requestID
}

type serverErrorFacts struct {
	code       string
	message    string
	requestID  string
	statusCode int
}

// serverAgentError renders a remote server error with its facts as structured
// envelope fields and routes recovery to the OpenAPI Explorer error-code
// diagnostics API when a code is available.
func serverAgentError(cause error, facts serverErrorFacts, context recoveryContext) error {
	message := facts.message
	if strings.TrimSpace(message) == "" {
		message = "server error " + facts.code
	}

	recovery := cli.AgentErrorRecovery{
		Action:  "inspect_action_help",
		Command: context.actionHelpCommand(),
		Hint:    "The server rejected the request; check the error code and message, fix the parameters, or retry if the failure is transient.",
	}
	if facts.code != "" {
		recovery = cli.AgentErrorRecovery{
			Action:  "diagnose_error_code",
			Command: diagnoseErrorCodeCommand(facts, context),
			Hint:    fmt.Sprintf("Look up diagnostic solutions for error code %s.", facts.code),
		}
	}

	agentErr := cli.NewAgentError(cli.AgentErrorEnvelope{
		Message:    nonEmptyMessage(message, cause),
		ErrorCode:  facts.code,
		StatusCode: facts.statusCode,
		RequestId:  facts.requestID,
		Recovery:   recovery,
	}, cause)
	if agentErr == nil {
		return cause
	}
	return agentErr
}

// diagnoseErrorCodeCommand builds the OpenAPI Explorer error-code solutions
// invocation in the same command style as the failed call: PascalCase users
// get the PascalCase entry with raw-name flags, kebab users the kebab entry.
// openapiexplorer has no global endpoint, so a known-good region is pinned.
func diagnoseErrorCodeCommand(facts serverErrorFacts, context recoveryContext) string {
	var parts []string
	if context.style == "pascal" {
		parts = []string{"aliyun", "openapiexplorer", "GetErrorCodeSolutions",
			"--errorCode", shellSingleQuote(facts.code)}
	} else {
		parts = []string{"aliyun", "openapiexplorer", "get-error-code-solutions",
			"--error-code", shellSingleQuote(facts.code)}
	}
	if product := strings.TrimSpace(context.product); product != "" {
		parts = append(parts, "--product", firstRuneUpper(product))
	}
	if context.style == "pascal" {
		parts = append(parts, "--acceptLanguage", diagnoseAcceptLanguage())
	} else {
		parts = append(parts, "--accept-language", diagnoseAcceptLanguage())
	}
	parts = append(parts, "--region", "cn-hangzhou")
	return strings.Join(parts, " ")
}

func diagnoseAcceptLanguage() string {
	if i18n.GetLanguage() == "zh" {
		return "zh-CN"
	}
	return "en-US"
}

func externalFlagRejectAgentError(cause error, reject *argparser.ExternalFlagRejectError, context recoveryContext) error {
	return newLocalAgentError(cause, reject.Message, nil, cli.AgentErrorRecovery{
		Action:  "inspect_action_help",
		Command: context.actionHelpCommand(),
		Hint:    "Use the flag spelling this command style accepts; the action help lists the parameters of this API.",
	})
}

// endpointAgentError renders a client-side endpoint/region resolution failure
// as a JSON envelope with a style-matched diagnostics command that lists the
// product's available endpoints.
func endpointAgentError(cause error, endpointErr *meta.InvalidEndpointError, context recoveryContext) error {
	message := endpointErr.Error()
	if message == "" {
		message = "unknown endpoint for the requested region"
	}
	return newLocalAgentError(cause, message, nil, cli.AgentErrorRecovery{
		Action:  "fix_endpoint_or_region",
		Command: endpointDiagnosticsCommand(context),
		Hint:    "The endpoint for this region could not be resolved. List the product's available endpoints with the command, use a supported region, or pass --endpoint <host> explicitly.",
	})
}

// endpointDiagnosticsCommand builds the OpenAPI Explorer product-endpoints
// invocation in the caller's command style; empty when the product is unknown.
func endpointDiagnosticsCommand(context recoveryContext) string {
	product := strings.TrimSpace(context.product)
	if product == "" {
		return ""
	}
	code := firstRuneUpper(product)
	if context.style == "pascal" {
		return "aliyun openapiexplorer GetProductEndpoints --product " + code + " --region cn-hangzhou"
	}
	return "aliyun openapiexplorer get-product-endpoints --product " + code + " --region cn-hangzhou"
}

func newLocalAgentError(cause error, message string, suggestions []string, recovery cli.AgentErrorRecovery) error {
	agentErr := cli.NewAgentError(cli.AgentErrorEnvelope{
		Message:    nonEmptyMessage(message, cause),
		DidYouMean: stableStrings(suggestions),
		Recovery:   recovery,
	}, cause)
	if agentErr == nil {
		return cause
	}
	return agentErr
}

func unknownProductAgentError(cause error, message string, suggestions []string, invalidName string, validate RecoverySearchValidator) error {
	suggestions = stableStrings(suggestions)
	recovery := cli.AgentErrorRecovery{
		Action:  "inspect_root_help",
		Command: "aliyun --help",
		Hint:    "Inspect the available products.",
	}
	for _, candidate := range orderedSearchCandidates(append(append([]string(nil), suggestions...), invalidName)...) {
		request := RecoverySearchRequest{Keyword: candidate}
		if validate != nil && validate(request) {
			recovery.Action = "search_product"
			recovery.Command = "aliyun --help-search " + candidate
			recovery.Hint = fmt.Sprintf("Search products related to %s.", candidate)
			break
		}
	}
	return newLocalAgentError(cause, message, suggestions, recovery)
}

func unknownCommandAgentError(cause error, message string, suggestions []string, invalidName string,
	context recoveryContext, validate RecoverySearchValidator) error {
	suggestions = stableStrings(suggestions)
	recovery := cli.AgentErrorRecovery{
		Action:  "inspect_parent_help",
		Command: context.parentHelpCommand(invalidName),
		Hint:    "Inspect commands under the current parent.",
	}
	seeds := append(append([]string(nil), suggestions...), invalidName)
	for _, candidate := range orderedSearchCandidates(seeds...) {
		request := context.parentSearchRequest(invalidName, candidate)
		if validate != nil && validate(request) {
			recovery.Action = "search_command"
			recovery.Command = context.parentSearchCommand(invalidName, candidate)
			recovery.Hint = fmt.Sprintf("Search commands under the current parent related to %s.", candidate)
			break
		}
	}
	return newLocalAgentError(cause, message, suggestions, recovery)
}

func unknownHostFlagAgentError(cause error, message string, suggestions []string, invalidName, helpCommand string,
	validate RecoverySearchValidator) error {
	suggestions = stableStrings(suggestions)
	recovery := cli.AgentErrorRecovery{
		Action:  "inspect_command_help",
		Command: suffixHelpCommand(helpCommand),
		Hint:    "Inspect the available flags for this command.",
	}
	seeds := append(append([]string(nil), suggestions...), invalidName)
	for _, candidate := range parameterSearchKeywordCandidates("", seeds...) {
		request := searchRequestForHelpCommand(helpCommand, candidate)
		if validate != nil && validate(request) {
			recovery.Action = "search_parameter"
			recovery.Command = suffixSearchCommand(helpCommand, candidate)
			recovery.Hint = fmt.Sprintf("Search flags for this command related to %s.", candidate)
			break
		}
	}
	return newLocalAgentError(cause, message, suggestions, recovery)
}

func unknownAPIAgentError(cause error, message string, suggestions []string, context recoveryContext, validate RecoverySearchValidator) error {
	recovery := cli.AgentErrorRecovery{
		Action:  "inspect_product_help",
		Command: context.productHelpCommand(),
		Hint:    "Inspect the available APIs for this product.",
	}
	seeds := append(append([]string(nil), suggestions...), context.api)
	for _, keyword := range apiSearchKeywordCandidates(context.style, seeds...) {
		request := context.searchRequest("", "", keyword)
		if validate != nil && validate(request) {
			recovery.Action = "search_api"
			recovery.Command = context.productSearchCommand(keyword)
			recovery.Hint = fmt.Sprintf("Search APIs related to %s.", keyword)
			break
		}
	}
	return newLocalAgentError(cause, message, suggestions, recovery)
}

func parameterSearchAgentError(cause error, message string, suggestions []string, invalidName string,
	context recoveryContext, validate RecoverySearchValidator) error {
	recovery := cli.AgentErrorRecovery{
		Action:  "inspect_action_help",
		Command: context.actionHelpCommand(),
		Hint:    "Inspect the action help and correct the parameter or flag.",
	}
	seeds := append(append([]string(nil), suggestions...), invalidName)
	for _, keyword := range parameterSearchKeywordCandidates(context.style, seeds...) {
		if validate != nil && validate(context.searchRequest("request", context.api, keyword)) {
			recovery.Action = "search_parameter"
			recovery.Command = context.actionSearchCommand(keyword)
			recovery.Hint = fmt.Sprintf("Search request parameters related to %s.", keyword)
			break
		}
	}
	return newLocalAgentError(cause, message, suggestions, recovery)
}

func invalidArgumentAgentError(cause error, message, flag, parameter, fieldPath string,
	context recoveryContext, validate RecoverySearchValidator) error {
	if topLevelFlag := topLevelRecoveryFlag(flag); topLevelFlag != "" && context.hasAction() {
		return newLocalAgentError(cause, message, nil, cli.AgentErrorRecovery{
			Action:  "inspect_parameter_help",
			Command: context.parameterHelpCommand(topLevelFlag),
			Hint:    fmt.Sprintf("Inspect help for --%s and correct its syntax or type.", topLevelFlag),
		})
	}

	keyword := firstNonEmpty(strings.TrimLeft(flag, "-"), parameter, fieldPath)
	recovery := cli.AgentErrorRecovery{
		Action:  "inspect_request_help",
		Command: context.requestHelpCommand(),
		Hint:    "Inspect the complete request help and correct the argument syntax or type.",
	}
	if keyword != "" {
		recovery.Hint = fmt.Sprintf("Inspect request help for %s and correct its syntax or type.", keyword)
		if validate != nil && validate(context.searchRequest("request", context.api, keyword)) {
			recovery.Action = "search_parameter"
			recovery.Command = context.requestSearchCommand(keyword)
		}
	}
	return newLocalAgentError(cause, message, nil, recovery)
}

type constraintFacts struct {
	flag       string
	value      string
	constraint string
	bound      string
	allowed    []string
}

// missingRequiredAgentMessage reports missing parameters with the flags the
// user can actually assign. MissingRequiredError.Error() also appends the
// PascalCase wire paths, which must not surface in kebab-style agent output.
func missingRequiredAgentMessage(err *runtime.MissingRequiredError) string {
	return "missing required parameter(s): " + strings.Join(err.Flags, ", ")
}

// legacyMissingRequiredFlagNames extracts the --Flag tokens from the wrapped
// legacy error text, which formats them one per line.
func legacyMissingRequiredFlagNames(err *LegacyMissingRequiredError) []string {
	matches := missingRequiredFlagPattern.FindAllStringSubmatch(err.Error(), -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		if match[1] != "" {
			names = append(names, "--"+match[1])
		}
	}
	return names
}

var missingRequiredFlagPattern = regexp.MustCompile(`--([A-Za-z0-9_.-]+)`)

// missingRequiredAgentError upgrades the recovery command from the full
// request Help to a targeted parameter search whenever the validator confirms
// the missing flag would be found — the search returns the parameter's schema
// alone instead of the complete request document.
func missingRequiredAgentError(cause error, message string, flags []string,
	context recoveryContext, validate RecoverySearchValidator) error {
	recovery := cli.AgentErrorRecovery{
		Action:  "inspect_request_help",
		Command: context.requestHelpCommand(),
		Hint:    "Inspect the complete request help and provide every required parameter.",
	}
	for _, keyword := range parameterSearchKeywordCandidates(context.style, flags...) {
		if validate != nil && validate(context.searchRequest("request", context.api, keyword)) {
			recovery.Action = "search_parameter"
			recovery.Command = context.actionSearchCommand(keyword)
			recovery.Hint = fmt.Sprintf("Search request parameters related to %s and provide the required value.", keyword)
			break
		}
	}
	return newLocalAgentError(cause, message, nil, recovery)
}

func runtimeConstraintFacts(e *runtime.ConstraintViolationError) constraintFacts {
	return constraintFacts{
		flag:       strings.TrimLeft(e.Flag, "-"),
		value:      e.Actual,
		constraint: e.Constraint,
		bound:      e.Expected,
		allowed:    e.Allowed,
	}
}

func legacyConstraintFacts(e *ConstraintViolationError) constraintFacts {
	bound := ""
	switch e.Constraint {
	case "minimum":
		bound = e.Minimum
	case "maximum":
		bound = e.Maximum
	case "minLength":
		bound = e.MinLength
	case "maxLength":
		bound = e.MaxLength
	case "pattern":
		bound = e.Pattern
	}
	return constraintFacts{
		flag:       strings.TrimLeft(e.Flag, "-"),
		value:      e.Value,
		constraint: e.Constraint,
		bound:      bound,
		allowed:    e.Allowed,
	}
}

func constraintViolationAgentError(cause error, facts constraintFacts, context recoveryContext, validate RecoverySearchValidator) error {
	recovery := cli.AgentErrorRecovery{
		Action:  "inspect_request_help",
		Command: context.requestHelpCommand(),
		Hint:    constraintViolationHint(facts),
	}
	for _, keyword := range parameterSearchKeywordCandidates(context.style, facts.flag) {
		if validate != nil && validate(context.searchRequest("request", context.api, keyword)) {
			recovery.Action = "search_parameter"
			recovery.Command = context.actionSearchCommand(keyword)
			recovery.Hint = fmt.Sprintf("Search request parameters related to %s and use a value that satisfies its constraints.", keyword)
			break
		}
	}
	return newLocalAgentError(cause, constraintViolationMessage(facts), stableStrings(facts.allowed), recovery)
}

func constraintViolationMessage(facts constraintFacts) string {
	switch facts.constraint {
	case "enum":
		return fmt.Sprintf("--%s value %q is not allowed", facts.flag, facts.value)
	case "minimum":
		return fmt.Sprintf("--%s value %q must be greater than or equal to %s", facts.flag, facts.value, facts.bound)
	case "maximum":
		return fmt.Sprintf("--%s value %q must be less than or equal to %s", facts.flag, facts.value, facts.bound)
	case "minLength":
		return fmt.Sprintf("--%s value %q must contain at least %s characters", facts.flag, facts.value, facts.bound)
	case "maxLength":
		return fmt.Sprintf("--%s value %q must contain at most %s characters", facts.flag, facts.value, facts.bound)
	case "pattern":
		return fmt.Sprintf("--%s value %q does not match pattern %q", facts.flag, facts.value, facts.bound)
	default:
		return fmt.Sprintf("--%s value %q violates its schema constraint", facts.flag, facts.value)
	}
}

func constraintViolationHint(facts constraintFacts) string {
	switch facts.constraint {
	case "enum":
		return fmt.Sprintf("Use one of the allowed values for --%s.", facts.flag)
	case "minimum":
		if facts.bound == "" {
			break
		}
		return fmt.Sprintf("Use a value greater than or equal to %s.", facts.bound)
	case "maximum":
		if facts.bound == "" {
			break
		}
		return fmt.Sprintf("Use a value less than or equal to %s.", facts.bound)
	case "minLength":
		if facts.bound == "" {
			break
		}
		return fmt.Sprintf("Use at least %s characters.", facts.bound)
	case "maxLength":
		if facts.bound == "" {
			break
		}
		return fmt.Sprintf("Use at most %s characters.", facts.bound)
	case "pattern":
		return fmt.Sprintf("Adjust the value of --%s to match the documented pattern.", facts.flag)
	}
	return fmt.Sprintf("Adjust the value of --%s to satisfy the documented constraint.", facts.flag)
}

func optionCombinationAgentError(cause error, message string, options []string) error {
	options = stableStrings(options)
	hint := "Remove one of the conflicting options."
	if len(options) > 0 {
		hint = fmt.Sprintf("Remove one of the conflicting options: %s.", strings.Join(options, ", "))
	}
	return newLocalAgentError(cause, message, nil, cli.AgentErrorRecovery{
		Action: "fix_option_combination",
		Hint:   hint,
	})
}

func helpOptionAgentError(cause error, optionErr *cli.HelpOptionError) error {
	if optionErr == nil {
		return cause
	}
	recovery := cli.AgentErrorRecovery{Action: "fix_help_options"}
	switch optionErr.Code {
	case cli.HelpOptionConflict:
		recovery.Action = "fix_option_combination"
		recovery.Hint = fmt.Sprintf("Use only one Help operation; remove either %s or %s.", optionErr.Option, optionErr.ConflictsWith)
	case cli.HelpOptionDuplicate:
		recovery.Action = "fix_option_combination"
		recovery.Hint = fmt.Sprintf("Remove the duplicate %s option.", optionErr.Option)
	case cli.HelpOptionEmptySearch:
		recovery.Hint = "Provide a non-empty query after --help-search."
	case cli.HelpOptionInvalidOutput:
		recovery.Hint = "Use --cli-output json, or remove --cli-output."
	case cli.HelpOptionInvalidSection:
		recovery.Hint = "Use --cli-section request or --cli-section response."
	default:
		recovery.Hint = "Correct the Help options and run the command again."
	}
	return newLocalAgentError(cause, optionErr.Error(), nil, recovery)
}

func fixedParameterHelpAgentError(cause error, message, action, parameter, hint string,
	context recoveryContext) error {
	command := context.actionHelpCommand()
	if context.hasAction() {
		command = context.parameterHelpCommand(parameter)
	}
	return newLocalAgentError(cause, message, nil, cli.AgentErrorRecovery{
		Action:  action,
		Command: command,
		Hint:    hint,
	})
}

func topLevelRecoveryFlag(flag string) string {
	flag = strings.TrimSpace(flag)
	if !strings.HasPrefix(flag, "--") {
		return ""
	}
	name := strings.TrimLeft(flag, "-")
	if name == "" || strings.Contains(name, ".") || !safeCommandToken(name) {
		return ""
	}
	return name
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
	if c.product == "" && safeCommandToken(product) {
		c.product = strings.ToLower(product)
	}
	if c.api == "" && safeCommandToken(api) {
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
		return "aliyun --help"
	}
	return c.kebabHelpEnvPrefix() + "aliyun " + c.product + c.versionSuffix() + " --help"
}

func (c recoveryContext) productSearchCommand(keyword string) string {
	if c.product == "" || !safeCommandToken(keyword) {
		return c.productHelpCommand()
	}
	return c.kebabHelpEnvPrefix() + "aliyun " + c.product + c.versionSuffix() + " --help-search " + keyword
}

// kebabHelpEnvPrefix keeps product-level recovery commands in the command style
// the user already used. Products that also have legacy help render PascalCase
// product help by default, so kebab help must be requested through the env var.
// Action-level help is routed by the command token itself and needs no prefix.
func (c recoveryContext) kebabHelpEnvPrefix() string {
	if c.style == "kebab" {
		return baselineProductHelpEnv + "=true "
	}
	return ""
}

func (c recoveryContext) hasAction() bool {
	return c.product != "" && c.api != ""
}

func (c recoveryContext) actionHelpCommand() string {
	if !c.hasAction() {
		return c.productHelpCommand()
	}
	return "aliyun " + c.product + " " + c.api + c.versionSuffix() + " --help"
}

func (c recoveryContext) actionSearchCommand(keyword string) string {
	if !c.hasAction() || !safeCommandToken(keyword) {
		return c.actionHelpCommand()
	}
	return "aliyun " + c.product + " " + c.api + c.versionSuffix() + " --help-search " + keyword
}

func (c recoveryContext) parameterHelpCommand(parameter string) string {
	parameter = strings.TrimLeft(strings.TrimSpace(parameter), "-")
	if !c.hasAction() || !safeCommandToken(parameter) || strings.Contains(parameter, ".") {
		return c.actionHelpCommand()
	}
	return "aliyun " + c.product + " " + c.api + c.versionSuffix() + " --" + parameter + " --help"
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
	return c.requestHelpCommand() + " --help-search " + keyword
}

func (c recoveryContext) parentHelpCommand(invalidName string) string {
	parts := c.parentCommandParts(invalidName)
	if len(parts) == 0 {
		return "aliyun --help"
	}
	return "aliyun " + strings.Join(parts, " ") + " --help"
}

func (c recoveryContext) parentSearchCommand(invalidName, keyword string) string {
	if !safeCommandToken(keyword) {
		return c.parentHelpCommand(invalidName)
	}
	parts := c.parentCommandParts(invalidName)
	if len(parts) == 0 {
		return "aliyun --help-search " + keyword
	}
	return "aliyun " + strings.Join(parts, " ") + " --help-search " + keyword
}

func (c recoveryContext) parentSearchRequest(invalidName, keyword string) RecoverySearchRequest {
	request := RecoverySearchRequest{Keyword: keyword}
	parts := c.parentCommandParts(invalidName)
	if len(parts) > 0 {
		request.Product = parts[0]
	}
	if len(parts) > 1 {
		request.API = parts[1]
		request.Style = commandStyle(parts[1])
	}
	return request
}

func (c recoveryContext) parentCommandParts(invalidName string) []string {
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
	return parts
}

func (c recoveryContext) versionSuffix() string {
	if c.version == "" || c.versionFlag == "" {
		return ""
	}
	return " " + c.versionFlag + " " + c.version
}

// suffixHelpCommand adapts the trusted built-in command path exposed by
// cli.InvalidFlagError until all callers share the HelpTarget command builder.
func suffixHelpCommand(command string) string {
	parts := helpCommandParts(command)
	if parts == nil {
		return "aliyun --help"
	}
	if len(parts) == 0 {
		return "aliyun --help"
	}
	return "aliyun " + strings.Join(parts, " ") + " --help"
}

func suffixSearchCommand(command, keyword string) string {
	if !safeCommandToken(keyword) {
		return suffixHelpCommand(command)
	}
	parts := helpCommandParts(command)
	if parts == nil || len(parts) == 0 {
		return "aliyun --help-search " + keyword
	}
	return "aliyun " + strings.Join(parts, " ") + " --help-search " + keyword
}

func searchRequestForHelpCommand(command, keyword string) RecoverySearchRequest {
	request := RecoverySearchRequest{Keyword: keyword}
	parts := helpCommandParts(command)
	if len(parts) > 0 {
		request.Product = parts[0]
	}
	if len(parts) > 1 {
		request.API = parts[1]
		request.Style = commandStyle(parts[1])
	}
	return request
}

func helpCommandParts(command string) []string {
	fields := strings.Fields(command)
	if len(fields) == 0 || fields[0] != "aliyun" {
		return nil
	}
	fields = fields[1:]
	if len(fields) > 0 && fields[0] == "help" {
		fields = fields[1:]
	}
	if len(fields) > 0 && fields[len(fields)-1] == "--help" {
		fields = fields[:len(fields)-1]
	}
	for _, field := range fields {
		if !safeCommandToken(field) {
			return nil
		}
	}
	return fields
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

func apiSearchKeywordCandidates(style string, seeds ...string) []string {
	candidates := make([]string, 0, len(seeds)*3)
	for _, seed := range seeds {
		resource := resourceKeyword(seed)
		if resource == "" {
			continue
		}
		if style == "kebab" {
			resource = apiNameToKebab(resource)
		}
		candidates = append(candidates, resource)
		tokens := splitHelpSearchTokens(resource)
		for index := len(tokens) - 1; index >= 0; index-- {
			candidates = append(candidates, formatSearchToken(tokens[index], style))
		}
	}
	return orderedSearchCandidates(candidates...)
}

func parameterSearchKeywordCandidates(style string, seeds ...string) []string {
	candidates := make([]string, 0, len(seeds)*3)
	for _, seed := range seeds {
		seed = strings.TrimLeft(strings.TrimSpace(seed), "-")
		if seed == "" {
			continue
		}
		candidates = append(candidates, seed)
		tokens := splitHelpSearchTokens(seed)
		for index := len(tokens) - 1; index >= 0; index-- {
			candidates = append(candidates, formatSearchToken(tokens[index], style))
		}
	}
	return orderedSearchCandidates(candidates...)
}

func formatSearchToken(token, style string) string {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" || style != "pascal" {
		return token
	}
	runes := []rune(token)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func orderedSearchCandidates(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !safeCommandToken(value) {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
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

func apiCandidateFormsForStyle(candidates []string, style string) []string {
	forms := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		switch style {
		case "kebab":
			if candidate != strings.ToLower(candidate) && !strings.Contains(candidate, "-") {
				candidate = apiNameToKebab(candidate)
			}
		case "pascal":
			if candidate == strings.ToLower(candidate) || strings.Contains(candidate, "-") || strings.Contains(candidate, "_") {
				candidate = kebabToPascal(candidate)
			}
		}
		forms = append(forms, candidate)
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
	suggestions := closeSuggestions(input, candidates, true)
	if len(suggestions) == 0 {
		suggestions = crossStyleFlagSuggestions(input, candidates)
	}
	return suggestions
}

// crossStyleFlagSuggestions recovers the exact candidate when the flag merely
// uses the other command style's casing convention: a PascalCase flag on a
// kebab command, or the reverse. Edit distance cannot bridge that gap
// ("RegionId" vs "region-id"), so the converted form is matched exactly.
func crossStyleFlagSuggestions(input string, candidates []string) []string {
	input = strings.TrimLeft(strings.TrimSpace(input), "-")
	if input == "" {
		return nil
	}
	converted := apiNameToKebab(input)
	if strings.ContainsAny(input, "-_") {
		converted = kebabToPascal(input)
	}
	if converted == "" || converted == input {
		return nil
	}
	for _, candidate := range candidates {
		candidate = strings.TrimLeft(strings.TrimSpace(candidate), "-")
		if candidate == converted {
			return []string{"--" + candidate}
		}
	}
	return nil
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
