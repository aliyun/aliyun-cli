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
	"net/url"
	"strings"
)

// encodePath preserves the path encoding contract used by generated Go plugins through aliyun-cli-runtime/http.EncodePath.
// The query suffix, when present, is intentionally left untouched for compatibility.
func encodePath(path string) string {
	var query string
	if idx := strings.Index(path, "?"); idx >= 0 {
		query = path[idx:]
		path = path[:idx]
	}

	parts := strings.Split(path, "/")
	for i, part := range parts {
		encoded := url.QueryEscape(part)
		encoded = strings.ReplaceAll(encoded, "+", "%20")
		encoded = strings.ReplaceAll(encoded, "*", "%2A")
		encoded = strings.ReplaceAll(encoded, "%7E", "~")
		parts[i] = encoded
	}
	return strings.Join(parts, "/") + query
}
