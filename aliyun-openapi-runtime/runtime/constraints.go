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
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

// ConstraintViolationError identifies one metadata constraint rejected by a
// parsed argument. Its exported fields are intended for errors.As callers.
type ConstraintViolationError struct {
	Parameter  string
	Flag       string
	Path       string
	Constraint string
	Actual     string
	Expected   string
	Allowed    []string
}

func (e *ConstraintViolationError) Error() string {
	message := fmt.Sprintf(
		"constraint violation: parameter=%q flag=%q path=%q constraint=%q actual=%q",
		e.Parameter, e.Flag, e.Path, e.Constraint, e.Actual,
	)
	if e.Expected != "" {
		message += fmt.Sprintf(" expected=%q", e.Expected)
	}
	if e.Allowed != nil {
		allowed, _ := json.Marshal(e.Allowed)
		message += " allowed=" + string(allowed)
	}
	return message
}

// ValidateConstraints recursively validates parsed arguments against metadata.
// Invalid producer-supplied bounds and patterns are ignored (fail-open).
func ValidateConstraints(api *meta.API, args map[string]any, rawBody bool) error {
	if api == nil {
		return nil
	}
	for i := range api.Parameters {
		parameter := &api.Parameters[i]
		if parameter.Type == meta.TypeAny ||
			rawBody && (parameter.Position == meta.PosBody || parameter.Position == meta.PosFormData) {
			continue
		}
		value, present := args[parameter.RawName]
		if !present {
			continue
		}
		context := constraintContext{
			parameter: parameter.Name,
			flag:      flagLabel(parameter),
			path:      pathName(parameter),
		}
		if err := validateConstraintValue(parameter, value, context); err != nil {
			return err
		}
	}
	return nil
}

type constraintContext struct {
	parameter string
	flag      string
	path      string
}

func validateConstraintValue(parameter *meta.Parameter, value any, context constraintContext) error {
	if parameter == nil || parameter.Type == meta.TypeAny || value == nil {
		return nil
	}

	actual := constraintActual(value)
	for _, allowed := range parameter.Enum {
		if actual == allowed {
			goto enumValid
		}
	}
	if len(parameter.Enum) > 0 {
		return constraintError(context, parameter, "enum", actual, "", parameter.Enum)
	}

enumValid:
	switch parameter.Type {
	case meta.TypeInteger, meta.TypeLong, meta.TypeFloat:
		if err := validateNumericConstraints(parameter, value, actual, context); err != nil {
			return err
		}
	case meta.TypeString:
		if parameter.Pattern != "" {
			expression, err := regexp.Compile(parameter.Pattern)
			if err == nil && !expression.MatchString(actual) {
				return constraintError(context, parameter, "pattern", actual, parameter.Pattern, nil)
			}
		}
	case meta.TypeObject:
		object, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		for i := range parameter.Fields {
			field := &parameter.Fields[i]
			fieldValue, present := object[field.RawName]
			if !present {
				continue
			}
			child := context
			if field.Name != "" {
				child.parameter = field.Name
			}
			child.path = appendObjectPath(context.path, pathName(field))
			if err := validateConstraintValue(field, fieldValue, child); err != nil {
				return err
			}
		}
	case meta.TypeArray:
		array, ok := value.([]any)
		if !ok || parameter.ItemType == nil {
			return nil
		}
		for index, item := range array {
			child := context
			child.path = fmt.Sprintf("%s[%d]", context.path, index)
			if err := validateConstraintValue(parameter.ItemType, item, child); err != nil {
				return err
			}
		}
	case meta.TypeMap:
		values, ok := value.(map[string]any)
		if !ok || parameter.ValueType == nil {
			return nil
		}
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			item := values[key]
			child := context
			child.path = fmt.Sprintf("%s[%q]", context.path, key)
			if err := validateConstraintValue(parameter.ValueType, item, child); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateNumericConstraints(parameter *meta.Parameter, value any, actual string, context constraintContext) error {
	number, ok := constraintRat(value)
	if !ok {
		return nil
	}
	if parameter.Minimum != "" {
		if minimum, valid := new(big.Rat).SetString(parameter.Minimum); valid && number.Cmp(minimum) < 0 {
			return constraintError(context, parameter, "minimum", actual, parameter.Minimum, nil)
		}
	}
	if parameter.Maximum != "" {
		if maximum, valid := new(big.Rat).SetString(parameter.Maximum); valid && number.Cmp(maximum) > 0 {
			return constraintError(context, parameter, "maximum", actual, parameter.Maximum, nil)
		}
	}
	return nil
}

func constraintRat(value any) (*big.Rat, bool) {
	var text string
	switch typed := value.(type) {
	case json.Number:
		text = typed.String()
	case float64:
		text = strconv.FormatFloat(typed, 'g', -1, 64)
	case float32:
		text = strconv.FormatFloat(float64(typed), 'g', -1, 32)
	case int:
		text = strconv.FormatInt(int64(typed), 10)
	case int32:
		text = strconv.FormatInt(int64(typed), 10)
	case int64:
		text = strconv.FormatInt(typed, 10)
	default:
		return nil, false
	}
	number, valid := new(big.Rat).SetString(text)
	return number, valid
}

func constraintActual(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'g', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'g', -1, 32)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprint(value)
	}
}

func constraintError(context constraintContext, parameter *meta.Parameter, constraint, actual, expected string, allowed []string) error {
	return &ConstraintViolationError{
		Parameter:  context.parameter,
		Flag:       context.flag,
		Path:       context.path,
		Constraint: constraint,
		Actual:     actual,
		Expected:   expected,
		Allowed:    append([]string(nil), allowed...),
	}
}

func pathName(parameter *meta.Parameter) string {
	if parameter.RawName != "" {
		return parameter.RawName
	}
	return parameter.Name
}

func appendObjectPath(parent, child string) string {
	if parent == "" {
		return child
	}
	if child == "" {
		return parent
	}
	return parent + "." + child
}
