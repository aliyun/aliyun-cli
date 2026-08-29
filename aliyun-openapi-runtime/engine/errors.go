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

// UnknownCommandError identifies a command that does not exist in the
// product's command index, so the host can render agent guidance (did you
// mean, recovery) instead of an opaque text error.
type UnknownCommandError struct {
	Product string
	Command string
}

func (e *UnknownCommandError) Error() string {
	return fmt.Sprintf("unknown command %q for product %q; try `aliyun %s` to list commands",
		e.Command, e.Product, e.Product)
}

func (*UnknownCommandError) AIRecoveryEligible() {}

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

// InvalidHeaderError preserves the input and expected format without exposing
// either field in recovery commands.
type InvalidHeaderError struct {
	Input          string
	ExpectedFormat string
	Err            error
}

func (e *InvalidHeaderError) Error() string { return localErrorText(e.Err, "invalid header") }

func (e *InvalidHeaderError) Unwrap() error { return e.Err }

func (*InvalidHeaderError) AIRecoveryEligible() {}

// InvalidBodyFileError identifies a local body file read failure.
type InvalidBodyFileError struct {
	Path string
	Err  error
}

func (e *InvalidBodyFileError) Error() string { return localErrorText(e.Err, "invalid body file") }

func (e *InvalidBodyFileError) Unwrap() error { return e.Err }

func (*InvalidBodyFileError) AIRecoveryEligible() {}

func localErrorText(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	return err.Error()
}
