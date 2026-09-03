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

package engine

import (
	"errors"
	"strings"
	"testing"
)

type aiRecoveryError interface {
	AIRecoveryEligible()
}

func TestTypedErrorsPreserveCausesAndRecoveryMarker(t *testing.T) {
	cause := errors.New("specific failure")
	tests := []struct {
		name string
		err  interface {
			error
			Unwrap() error
			AIRecoveryEligible()
		}
		want string
	}{
		{name: "option combination", err: &InvalidOptionCombinationError{Err: cause}, want: "specific failure"},
		{name: "header", err: &InvalidHeaderError{Err: cause}, want: "specific failure"},
		{name: "body file", err: &InvalidBodyFileError{Err: cause}, want: "specific failure"},
		{name: "query", err: &QueryFilterError{Expr: "Items[", Err: cause}, want: `invalid --cli-query "Items[": specific failure`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
			if !errors.Is(tc.err, cause) {
				t.Fatalf("errors.Is(%v, cause) = false", tc.err)
			}
			tc.err.AIRecoveryEligible()
		})
	}

	credential := &CredentialError{Err: cause}
	if credential.Error() != cause.Error() || !errors.Is(credential, cause) {
		t.Fatalf("CredentialError did not preserve cause: %v", credential)
	}

	unknown := &UnknownCommandError{Product: "ecs", Command: "missing"}
	if !strings.Contains(unknown.Error(), `unknown command "missing"`) {
		t.Fatalf("UnknownCommandError = %q", unknown.Error())
	}
	var recoverable aiRecoveryError = unknown
	recoverable.AIRecoveryEligible()
}

func TestTypedErrorFallbacks(t *testing.T) {
	option := &InvalidOptionCombinationError{}
	header := &InvalidHeaderError{}
	bodyFile := &InvalidBodyFileError{}
	query := &QueryFilterError{Expr: "Items["}

	if option.Error() != "invalid option combination" {
		t.Fatalf("option fallback = %q", option.Error())
	}
	if header.Error() != "invalid header" {
		t.Fatalf("header fallback = %q", header.Error())
	}
	if bodyFile.Error() != "invalid body file" {
		t.Fatalf("body fallback = %q", bodyFile.Error())
	}
	if !strings.Contains(query.Error(), "invalid JMESPath expression") {
		t.Fatalf("query fallback = %q", query.Error())
	}
}
