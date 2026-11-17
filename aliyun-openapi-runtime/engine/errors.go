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

package engine

import "fmt"

// UsageError preserves a machine-readable reason while keeping the exact
// existing error text for human-mode callers.
type UsageError struct {
	Code string
	Err  error
}

func (e *UsageError) Error() string { return e.Err.Error() }

func (e *UsageError) Unwrap() error { return e.Err }

func (e *UsageError) LocalErrorCode() string {
	switch e.Code {
	case "MISSING_COMMAND":
		return "INVALID_COMMAND"
	case "UNKNOWN_FLAG":
		return "INVALID_FLAG"
	case "MISSING_REQUIRED_PARAMETER":
		return "MISSING_REQUIRED_PARAMETER"
	case "INVALID_ARGUMENT", "INVALID_PARAMETER_VALUE":
		return "INVALID_ARGUMENT"
	case "INVALID_OPTION_COMBINATION":
		return "INVALID_OPTION_COMBINATION"
	case "INVALID_HEADER":
		return "INVALID_HEADER"
	case "INVALID_BODY_FILE":
		return "INVALID_BODY_FILE"
	default:
		return ""
	}
}

// UnknownCommandError identifies a command that does not exist in the
// product's command index, so the host can render agent guidance (did you
// mean, recovery) instead of an opaque text error.
type UnknownCommandError struct {
	Product    string
	Command    string
	Candidates []string
}

func (e *UnknownCommandError) Error() string {
	return fmt.Sprintf("unknown command %q for product %q; try `aliyun %s` to list commands",
		e.Command, e.Product, e.Product)
}

func (*UnknownCommandError) AIRecoveryEligible()    {}
func (*UnknownCommandError) LocalErrorCode() string { return "INVALID_COMMAND" }

// CredentialError identifies failures that occur before an API call while
// resolving the credential source configured by the embedding host.
type CredentialError struct {
	Err error
}

func (e *CredentialError) Error() string { return e.Err.Error() }

func (e *CredentialError) Unwrap() error { return e.Err }

// InvalidOptionCombinationError identifies conflicting local CLI options.
type InvalidOptionCombinationError struct {
	Options []string
	Err     error
}

func (e *InvalidOptionCombinationError) Error() string {
	return localErrorText(e.Err, "invalid option combination")
}

func (e *InvalidOptionCombinationError) Unwrap() error { return e.Err }

func (*InvalidOptionCombinationError) AIRecoveryEligible() {}
func (*InvalidOptionCombinationError) LocalErrorCode() string {
	return "INVALID_OPTION_COMBINATION"
}

// InvalidHeaderError preserves the input and expected format without exposing
// either field in recovery commands.
type InvalidHeaderError struct {
	Input          string
	ExpectedFormat string
	Err            error
}

func (e *InvalidHeaderError) Error() string { return localErrorText(e.Err, "invalid header") }

func (e *InvalidHeaderError) Unwrap() error { return e.Err }

func (*InvalidHeaderError) AIRecoveryEligible()    {}
func (*InvalidHeaderError) LocalErrorCode() string { return "INVALID_HEADER" }

// InvalidBodyFileError identifies a local body file read failure.
type InvalidBodyFileError struct {
	Path string
	Err  error
}

func (e *InvalidBodyFileError) Error() string { return localErrorText(e.Err, "invalid body file") }

func (e *InvalidBodyFileError) Unwrap() error { return e.Err }

func (*InvalidBodyFileError) AIRecoveryEligible()    {}
func (*InvalidBodyFileError) LocalErrorCode() string { return "INVALID_BODY_FILE" }

// QueryFilterError reports a --cli-query JMESPath failure with one wording in
// both command styles, so the host can render a unified AI envelope.
type QueryFilterError struct {
	Expr string
	Err  error
}

func (e *QueryFilterError) Error() string {
	return fmt.Sprintf("invalid --cli-query %q: %s", e.Expr, localErrorText(e.Err, "invalid JMESPath expression"))
}

func (e *QueryFilterError) Unwrap() error { return e.Err }

func (*QueryFilterError) AIRecoveryEligible()    {}
func (*QueryFilterError) LocalErrorCode() string { return "QUERY_FILTER_ERROR" }

func localErrorText(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	return err.Error()
}
