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
package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// expandJSONArrayParameter tries to parse value as a JSON array/object and
// flatten it into RPC flat-form keys (e.g. Servers.1.ServerId). Returns ok=false
// when value is not JSON structured data, so callers keep the original string.
func expandJSONArrayParameter(name, value string) (map[string]string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, false
	}
	if trimmed[0] != '[' && trimmed[0] != '{' {
		return nil, false
	}

	decoder := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	decoder.UseNumber()
	var parsed interface{}
	if err := decoder.Decode(&parsed); err != nil {
		return nil, false
	}

	out := make(map[string]string)
	flattenRPCValue(name, parsed, out)
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func flattenRPCValue(prefix string, v interface{}, out map[string]string) {
	switch val := v.(type) {
	case []interface{}:
		for i, item := range val {
			flattenRPCValue(fmt.Sprintf("%s.%d", prefix, i+1), item, out)
		}
	case map[string]interface{}:
		for k, item := range val {
			flattenRPCValue(fmt.Sprintf("%s.%s", prefix, k), item, out)
		}
	case nil:
		return
	case json.Number:
		out[prefix] = val.String()
	case string:
		out[prefix] = val
	case bool:
		out[prefix] = fmt.Sprintf("%t", val)
	default:
		out[prefix] = fmt.Sprintf("%v", val)
	}
}
