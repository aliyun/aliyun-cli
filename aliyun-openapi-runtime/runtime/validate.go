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
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

// MissingRequiredError reports one or more required parameters that were not supplied.
type MissingRequiredError struct {
	Flags []string // user-facing option names, e.g. ["--layer-name"]
}

func (e *MissingRequiredError) Error() string {
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
