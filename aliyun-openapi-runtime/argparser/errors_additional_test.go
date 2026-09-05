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

package argparser

import (
	"errors"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestInvalidArgumentErrorNilCauseAndRecoveryMarker(t *testing.T) {
	var nilError *InvalidArgumentError
	if nilError.Error() != "invalid argument" || nilError.Unwrap() != nil {
		t.Fatalf("nil InvalidArgumentError = %q, unwrap %v", nilError.Error(), nilError.Unwrap())
	}
	withoutCause := &InvalidArgumentError{}
	if withoutCause.Error() != "invalid argument" {
		t.Fatalf("empty InvalidArgumentError = %q", withoutCause.Error())
	}
	cause := errors.New("bad integer")
	withCause := &InvalidArgumentError{Err: cause}
	if withCause.Error() != cause.Error() || !errors.Is(withCause, cause) {
		t.Fatalf("InvalidArgumentError did not preserve cause: %v", withCause)
	}
	withCause.AIRecoveryEligible()
}

func TestExternalFlagIndexRejectsInvalidDefinitionsAndModes(t *testing.T) {
	tests := []struct {
		name  string
		specs []ExternalFlagSpec
		want  string
	}{
		{name: "empty name", specs: []ExternalFlagSpec{{Name: "  "}}, want: "name is empty"},
		{name: "duplicate name", specs: []ExternalFlagSpec{{Name: "profile"}, {Name: "profile"}}, want: "--profile duplicated"},
		{name: "duplicate shorthand", specs: []ExternalFlagSpec{{Name: "profile", Shorthand: 'p'}, {Name: "project", Shorthand: 'p'}}, want: "-p duplicated"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newExternalFlagIndex(tc.specs)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("newExternalFlagIndex() error = %v, want containing %q", err, tc.want)
			}
		})
	}

	var nilIndex *externalFlagIndex
	if _, _, _, ok := nilIndex.match("--profile"); ok {
		t.Fatal("nil external flag index matched a flag")
	}
	if _, err := consumeExternalFlag(nil, 0, nil, nil, ExternalFlagSpec{Name: "quiet", Mode: ExternalFlagNone}, "true", true); err == nil || !strings.Contains(err.Error(), "does not accept a value") {
		t.Fatalf("value on no-value flag error = %v", err)
	}
	if _, err := consumeExternalFlag(nil, 0, nil, nil, ExternalFlagSpec{Name: "future", Mode: ExternalFlagMode(99)}, "", false); err == nil || !strings.Contains(err.Error(), "unsupported mode 99") {
		t.Fatalf("unsupported external mode error = %v", err)
	}
	rejected := &ExternalFlagRejectError{Message: "not supported"}
	if rejected.Error() != "not supported" {
		t.Fatalf("ExternalFlagRejectError = %q", rejected.Error())
	}
	rejected.AIRecoveryEligible()
}

func TestParamIndexIgnoresEmptyAndDuplicateAliases(t *testing.T) {
	params := []meta.Parameter{
		{Name: "first", Options: []string{"", "--name"}},
		{Name: "second", Options: []string{"--name"}},
	}
	index := newParamIndex(params)
	if got := index.lookup("name"); got == nil || got.Name != "first" {
		t.Fatalf("duplicate alias resolved to %#v", got)
	}
	unknown := &UnknownFlagError{Flag: "missing"}
	if unknown.Error() != "unknown flag --missing" {
		t.Fatalf("UnknownFlagError = %q", unknown.Error())
	}
	unknown.AIRecoveryEligible()
}
