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

package argparser

// InvalidArgumentError preserves schema context for an API parameter parse
// failure while leaving the historical human-facing error text unchanged.
type InvalidArgumentError struct {
	Parameter    string
	Flag         string
	FieldPath    string
	ExpectedType string
	Err          error
}

func (e *InvalidArgumentError) Error() string {
	if e == nil || e.Err == nil {
		return "invalid argument"
	}
	return e.Err.Error()
}

func (e *InvalidArgumentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (*InvalidArgumentError) AIRecoveryEligible() {}
