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
	"sort"
	"strconv"
	"strings"
)

// serializeRPC flattens a (possibly nested) value into the RPC query
// convention used by Alibaba Cloud OpenAPI:
//
//	scalar          Prefix               = v
//	array<scalar>   Prefix.1, Prefix.2   = v1, v2
//	array<object>   Prefix.1.Key, ...    = ...
//	object / map    Prefix.Key           = ...
//
// It mirrors aliyun-cli-runtime/http.serializeRPCStyle but operates on
// the argparser's concrete value types (string / json.Number / bool /
// []any / map[string]any) instead of reflection over arbitrary types,
// keeping the hot path allocation-light and deterministic.
func serializeRPC(prefix string, value any) map[string]string {
	out := map[string]string{}
	flatten(prefix, value, out)
	return out
}

func flatten(prefix string, value any, out map[string]string) {
	switch v := value.(type) {
	case nil:
		return
	case []any:
		for i, item := range v {
			flatten(fmt.Sprintf("%s.%d", prefix, i+1), item, out)
		}
	case map[string]any:
		// Deterministic key order so dry-run output and tests are
		// stable.
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			flatten(prefix+"."+k, v[k], out)
		}
	default:
		out[prefix] = basicString(v)
	}
}

// serializeQuery renders one parameter into flat query key/values,
// honouring its declared param_style:
//
//	"json"              -> {name: <compact JSON of value>}
//	"simple"            -> {name: "a,b,c"} for scalar arrays
//	"flat"/"repeatList" -> RPC index flatten (name.1, name.1.Key, ...)
//	"" (unset)          -> RPC flatten when the operation is RPC,
//	                       otherwise JSON for composites / raw for scalars
//
// This is the single place that maps the abstract param_style to the
// wire form, shared by query and (RPC) body handling.
func serializeQuery(name string, value any, isRPC bool, paramStyle string) (map[string]string, error) {
	switch paramStyle {
	case "json":
		b, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal %s as json: %w", name, err)
		}
		return map[string]string{name: string(b)}, nil
	case "simple":
		if arr, ok := value.([]any); ok {
			parts := make([]string, 0, len(arr))
			for _, it := range arr {
				parts = append(parts, basicString(it))
			}
			return map[string]string{name: strings.Join(parts, ",")}, nil
		}
		return map[string]string{name: basicString(value)}, nil
	case "flat", "repeatList":
		return serializeRPC(name, value), nil
	}

	// Unset style: RPC flattens everything; RESTful keeps scalars raw
	// and JSON-encodes composites.
	if isRPC {
		return serializeRPC(name, value), nil
	}
	switch value.(type) {
	case map[string]any, []any:
		b, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal %s: %w", name, err)
		}
		return map[string]string{name: string(b)}, nil
	default:
		return map[string]string{name: basicString(value)}, nil
	}
}

func basicString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}
