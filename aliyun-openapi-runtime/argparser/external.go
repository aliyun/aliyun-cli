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
	"fmt"
	"strings"
)

// ExternalFlagMode describes the value arity of a flag parsed and owned by the embedding host.
// Repeatable external flags are intentionally unsupported until a host has a concrete need and value-boundary contract for them.
type ExternalFlagMode uint8

const (
	ExternalFlagNone ExternalFlagMode = iota
	ExternalFlagOptional
	ExternalFlagRequired
)

// ExternalFlagSpec lets an embedding host declare argv flags that the engine must recognise syntactically but must not expose as API or reserved values.
type ExternalFlagSpec struct {
	Name          string
	Shorthand     rune
	Mode          ExternalFlagMode
	RejectMessage string
}

// ExternalFlagRejectError reports a host flag the engine must refuse outright
// (for example a PascalCase-only global passed to a kebab command). The typed
// carrier lets the host render it through its agent error envelope instead of
// falling back to plain text.
type ExternalFlagRejectError struct {
	Flag    string
	Message string
}

func (e *ExternalFlagRejectError) Error() string {
	return e.Message
}

func (*ExternalFlagRejectError) AIRecoveryEligible() {}

// ParseOptions carries host-specific parser extensions for one invocation.
type ParseOptions struct {
	ExternalFlags []ExternalFlagSpec
}

type externalFlagIndex struct {
	byName      map[string]ExternalFlagSpec
	byShorthand map[rune]ExternalFlagSpec
}

func newExternalFlagIndex(specs []ExternalFlagSpec) (*externalFlagIndex, error) {
	externalFlags := &externalFlagIndex{
		byName:      make(map[string]ExternalFlagSpec, len(specs)),
		byShorthand: make(map[rune]ExternalFlagSpec),
	}
	for _, spec := range specs {
		spec.Name = strings.TrimSpace(spec.Name)
		if spec.Name == "" {
			return nil, fmt.Errorf("external flag name is empty")
		}
		if _, exists := externalFlags.byName[spec.Name]; exists {
			return nil, fmt.Errorf("external flag --%s duplicated", spec.Name)
		}
		externalFlags.byName[spec.Name] = spec
		if spec.Shorthand != 0 {
			if _, exists := externalFlags.byShorthand[spec.Shorthand]; exists {
				return nil, fmt.Errorf("external flag -%c duplicated", spec.Shorthand)
			}
			externalFlags.byShorthand[spec.Shorthand] = spec
		}
	}
	return externalFlags, nil
}

func (externalFlags *externalFlagIndex) match(tok string) (ExternalFlagSpec, string, bool, bool) {
	if externalFlags == nil {
		return ExternalFlagSpec{}, "", false, false
	}
	prefix, inline, hasInline := splitExternalToken(tok)
	if strings.HasPrefix(prefix, "--") && len(prefix) > 2 {
		spec, ok := externalFlags.byName[prefix[2:]]
		return spec, inline, hasInline, ok
	}
	runes := []rune(prefix)
	if len(runes) == 2 && runes[0] == '-' && runes[1] != '-' {
		spec, ok := externalFlags.byShorthand[runes[1]]
		return spec, inline, hasInline, ok
	}
	return ExternalFlagSpec{}, "", false, false
}

func splitExternalToken(tok string) (prefix, value string, hasInline bool) {
	// 只有继承自cli 全局参数的，根据现有规则，支持= : 及空格传参
	if i := strings.IndexAny(tok, "=:"); i >= 0 {
		return tok[:i], tok[i+1:], true
	}
	return tok, "", false
}

func consumeExternalFlag(
	args []string,
	i int,
	externalFlags *externalFlagIndex,
	apiParams *paramIndex,
	spec ExternalFlagSpec,
	inlineVal string,
	hasInline bool,
) (int, error) {
	if spec.RejectMessage != "" {
		return i, &ExternalFlagRejectError{Flag: spec.Name, Message: spec.RejectMessage}
	}
	switch spec.Mode {
	case ExternalFlagNone:
		if hasInline {
			return i, fmt.Errorf("--%s does not accept a value", spec.Name)
		}
	case ExternalFlagOptional:
		if !hasInline || inlineVal == "" {
			_, i = takeOneValue(args, i, externalFlags, apiParams)
		}
	case ExternalFlagRequired:
		if hasInline && inlineVal != "" {
			return i, nil
		}
		before := i
		_, i = takeOneValue(args, i, externalFlags, apiParams)
		if i == before {
			return i, fmt.Errorf("--%s requires a value", spec.Name)
		}
	default:
		return i, fmt.Errorf("external flag --%s has unsupported mode %d", spec.Name, spec.Mode)
	}
	return i, nil
}
