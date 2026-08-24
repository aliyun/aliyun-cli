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

package runtime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

// MissingRequiredError reports one or more required parameters that were not supplied.
type MissingRequiredError struct {
	Flags []string // user-facing option names, e.g. ["--layer-name"]
	Paths []string // wire-side paths; populated by recursive docRequired validation
}

func (e *MissingRequiredError) Error() string {
	if len(e.Paths) == len(e.Flags) && len(e.Paths) > 0 {
		items := make([]string, len(e.Flags))
		for index := range e.Flags {
			items[index] = fmt.Sprintf("%s (%s)", e.Flags[index], e.Paths[index])
		}
		return "missing required parameter(s): " + strings.Join(items, ", ")
	}
	return "missing required parameter(s): " + strings.Join(e.Flags, ", ")
}

// ValidateRequired checks that every required top-level parameter of api was supplied.
// When rawBody is true, body and formData parameters are skipped because --body/--body-file replaces the entire request body.
func ValidateRequired(api *meta.API, args map[string]any, rawBody bool) error {
	if api == nil {
		return nil
	}
	var missing []string
	for i := range api.Parameters {
		p := &api.Parameters[i]
		if !p.Required {
			continue
		}
		if rawBody && (p.Position == meta.PosBody || p.Position == meta.PosFormData) {
			continue
		}
		key := p.RawName
		if key == "" || isEmptyValue(args[key]) {
			missing = append(missing, flagLabel(p))
		}
	}
	if len(missing) > 0 {
		return &MissingRequiredError{Flags: missing}
	}
	return nil
}

// ValidateDocRequired checks AI-oriented docRequired metadata. Unlike Required,
// docRequired may be attached to top-level parameters and named object fields
// nested below objects, array elements, and map values.
func ValidateDocRequired(api *meta.API, args map[string]any, rawBody bool) error {
	if api == nil {
		return nil
	}
	var missing missingRequired
	for i := range api.Parameters {
		parameter := &api.Parameters[i]
		if rawBody && (parameter.Position == meta.PosBody || parameter.Position == meta.PosFormData) {
			continue
		}
		value, present := args[parameter.RawName]
		context := docRequiredContext{
			flag: flagLabel(parameter),
			path: pathName(parameter),
		}
		validateDocRequiredValue(parameter, value, present, context, &missing)
	}
	if len(missing.flags) > 0 {
		return &MissingRequiredError{Flags: missing.flags, Paths: missing.paths}
	}
	return nil
}

type missingRequired struct {
	flags []string
	paths []string
}

type docRequiredContext struct {
	flag string
	path string
}

func validateDocRequiredValue(parameter *meta.Parameter, value any, present bool, context docRequiredContext, missing *missingRequired) {
	if parameter == nil {
		return
	}
	if parameter.DocRequired && (!present || isEmptyValue(value)) {
		missing.flags = append(missing.flags, context.flag)
		missing.paths = append(missing.paths, context.path)
		return
	}
	if !present || value == nil {
		return
	}

	switch parameter.Type {
	case meta.TypeObject:
		object, ok := value.(map[string]any)
		if !ok {
			return
		}
		for i := range parameter.Fields {
			field := &parameter.Fields[i]
			fieldValue, fieldPresent := object[field.RawName]
			child := context
			child.path = appendObjectPath(context.path, pathName(field))
			validateDocRequiredValue(field, fieldValue, fieldPresent, child, missing)
		}
	case meta.TypeArray:
		array, ok := value.([]any)
		if !ok || parameter.ItemType == nil {
			return
		}
		for index, item := range array {
			child := context
			child.path = fmt.Sprintf("%s[%d]", context.path, index)
			validateDocRequiredValue(parameter.ItemType, item, true, child, missing)
		}
	case meta.TypeMap:
		values, ok := value.(map[string]any)
		if !ok || parameter.ValueType == nil {
			return
		}
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			child := context
			child.path = fmt.Sprintf("%s[%q]", context.path, key)
			validateDocRequiredValue(parameter.ValueType, values[key], true, child, missing)
		}
	}
}

// isEmptyValue matches aliyun-cli-runtime's required-argument semantics:
// nil and empty strings count as not provided, while an explicitly supplied empty object or array still satisfies the top-level required parameter.
func isEmptyValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	default:
		return false
	}
}

func flagLabel(p *meta.Parameter) string {
	if len(p.Options) > 0 {
		return p.Options[0]
	}
	return "--" + strings.ReplaceAll(p.Name, "_", "-")
}
