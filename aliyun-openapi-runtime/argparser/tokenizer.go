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

import "strings"

// splitLongFlag inspects one token syntactically.
// A potential long flag starts with "--" and may carry an inline value via "=".
// Whether it is actually a flag is decided separately from the registered API, runtime, and host flags.
//
// Both aliyun-cli-runtime and aliyun-openapi-runtime support "=" and spaces for long scalar flags.
// Composite API parameters support only the space form.
// The "=" form is retained for compatibility with the legacy runtime, not recommand or used in docs
func splitLongFlag(tok string) (name, value string, hasInline, isFlag bool) {
	if !strings.HasPrefix(tok, "--") || tok == "--" {
		return "", "", false, false
	}
	body := tok[2:]
	if k, v, ok := strings.Cut(body, "="); ok {
		return k, v, true, true
	}
	return body, "", false, true
}

// isFlagToken reports whether tok terminates a value run.
// Only registered API, runtime, and host flags do;
// an unregistered "--..." token remains a value.
func isFlagToken(tok string, externalFlags *externalFlagIndex, apiParams *paramIndex) bool {
	if _, _, _, _, ok := reservedFlags.match(tok); ok {
		return true
	}
	if _, _, _, ok := externalFlags.match(tok); ok {
		return true
	}
	name, _, _, potentialFlag := splitLongFlag(tok)
	if !potentialFlag {
		return false
	}
	return apiParams != nil && apiParams.lookup(name) != nil
}

// takeValues consumes consecutive non-flag tokens starting at i and returns them plus the new index.
// Used for flags that may take multiple value tokens in one occurrence.
func takeValues(args []string, i int, externalFlags *externalFlagIndex, apiParams *paramIndex) ([]string, int) {
	var out []string
	for i < len(args) {
		if isFlagToken(args[i], externalFlags, apiParams) {
			break
		}
		out = append(out, args[i])
		i++
	}
	return out, i
}

// takeOneValue consumes exactly one value token. It returns "" if the next token is a registered flag.
func takeOneValue(args []string, i int, externalFlags *externalFlagIndex, apiParams *paramIndex) (string, int) {
	if i < len(args) {
		if !isFlagToken(args[i], externalFlags, apiParams) {
			return args[i], i + 1
		}
	}
	return "", i
}
