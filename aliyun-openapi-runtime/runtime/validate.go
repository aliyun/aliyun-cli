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

// MissingRequiredError reports one or more required parameters that
// were not supplied. It carries the user-facing flag names so the CLI
// layer can print an actionable message.
type MissingRequiredError struct {
	Flags []string // user-facing option names, e.g. ["--layer-name"]
}

func (e *MissingRequiredError) Error() string {
	return "missing required parameter(s): " + strings.Join(e.Flags, ", ")
}

// ValidateRequired checks that every required top-level parameter of
// api has a non-empty value in args. It returns a *MissingRequiredError
// listing every offender (not just the first) so users can fix them in
// one pass.
//
// This runs before Assemble so failures are caught client-side with a
// clear message rather than surfacing as an opaque server 400 (e.g.
// an unsubstituted {layerName} path placeholder becoming "Illegal Path
// Character").
func ValidateRequired(api *meta.API, args map[string]any) error {
	if api == nil {
		return nil
	}
	var missing []string
	for i := range api.Parameters {
		p := &api.Parameters[i]
		if !p.Required {
			continue
		}
		// Args are keyed by RawName; a required param without RawName
		// can never be satisfied and is reported as missing.
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

// isEmptyValue reports whether v counts as "not provided" for required
// checking: nil, empty string, or empty composite.
func isEmptyValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
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
