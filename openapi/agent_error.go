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
	"net"
	"sort"
	"strings"
	"unicode"

	"github.com/alibabacloud-go/tea/tea"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
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

func normalizeAgentError(err error, args []string) error {
	if err == nil {
		return nil
	}

	var existing *cli.AgentError
	if errors.As(err, &existing) {
		return existing
	}

	var unknownFlag *argparser.UnknownFlagError
	if errors.As(err, &unknownFlag) {
		return newAgentError(err, cli.UsageErrorCategory, "UNKNOWN_FLAG", unknownFlag.Error(),
			flagSuggestions(unknownFlag.Flag, unknownFlag.Known), false, "", commandHelp(args))
	}

	var missing *runtime.MissingRequiredError
	if errors.As(err, &missing) {
		return newAgentError(err, cli.UsageErrorCategory, "MISSING_REQUIRED_PARAMETER", missing.Error(),
			stableStrings(missing.Flags), false, "", commandHelp(args))
	}

	var runtimeConstraint *runtime.ConstraintViolationError
	if errors.As(err, &runtimeConstraint) {
		return newAgentError(err, cli.UsageErrorCategory, "INVALID_PARAMETER_VALUE", runtimeConstraint.Error(),
			stableStrings(runtimeConstraint.Allowed), false, "", commandHelp(args))
	}

	var legacyConstraint *ConstraintViolationError
	if errors.As(err, &legacyConstraint) {
		return newAgentError(err, cli.UsageErrorCategory, "INVALID_PARAMETER_VALUE", legacyConstraint.Error(),
			stableStrings(legacyConstraint.Allowed), false, "", commandHelp(args))
	}

	var invalidParameter *InvalidParameterError
	if errors.As(err, &invalidParameter) {
		return newAgentError(err, cli.UsageErrorCategory, "UNKNOWN_FLAG", invalidParameter.Error(),
			invalidParameter.AgentSuggestions(), false, "", commandHelp(args))
	}

	var invalidAPI *InvalidApiError
	if errors.As(err, &invalidAPI) {
		return newAgentError(err, cli.UsageErrorCategory, "UNKNOWN_API", invalidAPI.Error(),
			invalidAPI.AgentSuggestions(), false, "", productHelp(args))
	}

	var invalidUnifiedAPI *InvalidUnifiedApiError
	if errors.As(err, &invalidUnifiedAPI) {
		return newAgentError(err, cli.UsageErrorCategory, "UNKNOWN_API", invalidUnifiedAPI.Error(),
			invalidUnifiedAPI.AgentSuggestions(), false, "", productHelp(args))
	}

	var invalidProduct *InvalidProductError
	if errors.As(err, &invalidProduct) {
		return newAgentError(err, cli.UsageErrorCategory, "UNKNOWN_PRODUCT", invalidProduct.Error(),
			invalidProduct.AgentSuggestions(), false, "", "aliyun --help")
	}

	var invalidProductOrPlugin *InvalidProductOrPluginError
	if errors.As(err, &invalidProductOrPlugin) {
		return newAgentError(err, cli.UsageErrorCategory, "UNKNOWN_PRODUCT", invalidProductOrPlugin.Error(),
			invalidProductOrPlugin.AgentSuggestions(), false, "", "aliyun --help")
	}

	var invalidFlag *cli.InvalidFlagError
	if errors.As(err, &invalidFlag) {
		return newAgentError(err, cli.UsageErrorCategory, "UNKNOWN_FLAG", invalidFlag.Error(),
			nil, false, "", commandHelp(args))
	}

	var invalidCommand *cli.InvalidCommandError
	if errors.As(err, &invalidCommand) {
		return newAgentError(err, cli.UsageErrorCategory, "UNKNOWN_COMMAND", invalidCommand.Error(),
			invalidCommand.GetSuggestions(), false, "", commandHelp(args))
	}

	var runtimeCredential *engine.CredentialError
	var configuredCredential *credentialConfigurationError
	if errors.As(err, &runtimeCredential) || errors.As(err, &configuredCredential) {
		return newAgentError(err, cli.AuthenticationErrorCategory, "CREDENTIAL_NOT_CONFIGURED", err.Error(),
			nil, false, "", "aliyun configure")
	}

	var usage *engine.UsageError
	if errors.As(err, &usage) {
		code := strings.TrimSpace(usage.Code)
		if code == "" {
			code = "INVALID_ARGUMENT"
		}
		return newAgentError(err, cli.UsageErrorCategory, code, err.Error(), nil, false, "", commandHelp(args))
	}

	var serverError *sdkerrors.ServerError
	if errors.As(err, &serverError) {
		category, retryable := classifySDKFailure(serverError.HttpStatus(), serverError.ErrorCode())
		code := nonEmptyCode(serverError.ErrorCode(), "SERVICE_ERROR")
		message := nonEmptyMessage(serverError.Message(), err.Error())
		return newAgentError(err, category, code, message, nil, retryable, serverError.RequestId(), "")
	}

	var teaError *tea.SDKError
	if errors.As(err, &teaError) {
		status := tea.IntValue(teaError.StatusCode)
		code := tea.StringValue(teaError.Code)
		category, retryable := classifySDKFailure(status, code)
		return newAgentError(err, category, nonEmptyCode(code, "SERVICE_ERROR"),
			nonEmptyMessage(tea.StringValue(teaError.Message), err.Error()), nil, retryable,
			requestIDFromTeaData(tea.StringValue(teaError.Data)), "")
	}

	var networkError net.Error
	if errors.As(err, &networkError) {
		retryable := networkError.Timeout()
		if temporary, ok := networkError.(interface{ Temporary() bool }); ok {
			retryable = retryable || temporary.Temporary()
		}
		return newAgentError(err, cli.NetworkErrorCategory, "NETWORK_FAILURE", err.Error(), nil, retryable, "", "")
	}

	return newAgentError(err, cli.InternalErrorCategory, "INTERNAL_ERROR", err.Error(), nil, false, "", "")
}

func newAgentError(cause error, category cli.AgentErrorCategory, code, message string, suggestions []string,
	retryable bool, requestID, recoveryCommand string) error {
	return cli.NewAgentError(cli.AgentErrorEnvelope{
		OK:          false,
		Category:    category,
		Code:        code,
		Message:     message,
		Suggestions: suggestions,
		Retryable:   retryable,
		RequestID:   requestID,
		Recovery:    cli.AgentErrorRecovery{Command: recoveryCommand},
	}, cause)
}

func classifySDKFailure(status int, code string) (cli.AgentErrorCategory, bool) {
	normalized := normalizeErrorCode(code)
	switch {
	case status == 401 || isAuthenticationCode(normalized):
		return cli.AuthenticationErrorCategory, false
	case status == 403 || isPermissionCode(normalized):
		return cli.PermissionErrorCategory, false
	case status == 429 || isThrottlingCode(normalized):
		return cli.ThrottlingErrorCategory, true
	case status >= 500 && status <= 599:
		return cli.ServiceErrorCategory, true
	default:
		return cli.ServiceErrorCategory, false
	}
}

func normalizeErrorCode(code string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(code) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isAuthenticationCode(code string) bool {
	if code == "unauthorized" {
		return true
	}
	return hasAnyPrefix(code,
		"invalidaccesskey", "invalidsecuritytoken", "missingsecuritytoken",
		"expiredsecuritytoken", "tokenexpired", "signaturedoesnotmatch",
		"incompletesignature", "authentication", "authfailure", "invalidcredential")
}

func isPermissionCode(code string) bool {
	return hasAnyPrefix(code,
		"forbidden", "accessdenied", "unauthorizedoperation", "permissiondenied",
		"nopermission", "operationdenied")
}

func isThrottlingCode(code string) bool {
	return hasAnyPrefix(code, "throttling", "toomanyrequests", "requestlimitexceeded", "flowcontrol")
}

func hasAnyPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func requestIDFromTeaData(data string) string {
	if strings.TrimSpace(data) == "" {
		return ""
	}
	var decoded interface{}
	if json.Unmarshal([]byte(data), &decoded) != nil {
		return ""
	}
	return findRequestID(decoded)
}

func findRequestID(value interface{}) string {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, item := range typed {
			if strings.EqualFold(key, "requestId") {
				if requestID, ok := item.(string); ok {
					return requestID
				}
			}
		}
		for _, item := range typed {
			if requestID := findRequestID(item); requestID != "" {
				return requestID
			}
		}
	case []interface{}:
		for _, item := range typed {
			if requestID := findRequestID(item); requestID != "" {
				return requestID
			}
		}
	}
	return ""
}

func commandHelp(args []string) string {
	if len(args) >= 2 && strings.TrimSpace(args[0]) != "" && strings.TrimSpace(args[1]) != "" {
		return "aliyun " + args[0] + " " + args[1] + " --help"
	}
	return productHelp(args)
}

func productHelp(args []string) string {
	if len(args) >= 1 && strings.TrimSpace(args[0]) != "" {
		return "aliyun " + args[0] + " --help"
	}
	return "aliyun --help"
}

func nonEmptyCode(code, fallback string) string {
	if strings.TrimSpace(code) == "" {
		return fallback
	}
	return code
}

func nonEmptyMessage(message, fallback string) string {
	if strings.TrimSpace(message) == "" {
		return fallback
	}
	return message
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
		for i := range results {
			results[i] = "--" + strings.TrimLeft(results[i], "-")
		}
	}
	return stableStrings(results)
}

func stableStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
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
