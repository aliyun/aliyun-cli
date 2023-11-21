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
	"encoding/json"
	"fmt"

	jmespath "github.com/jmespath/go-jmespath"
)

// find array in json object
func detectArrayPath(d interface{}) string {
	m, ok := d.(map[string]interface{})
	if !ok {
		return ""
	}
	for k, v := range m {
		// t.Logf("%v %v\n", k, v)
		if m2, ok := v.(map[string]interface{}); ok {
			for k2, v2 := range m2 {
				if _, ok := v2.([]interface{}); ok {
					return fmt.Sprintf("%s.%s[]", k, k2)
				}
			}
		}
	}
	return ""
}

func evaluateExpr(body []byte, expr string) (string, error) {
	var entity interface{}
	err := json.Unmarshal(body, &entity)
	if err != nil {
		return "", fmt.Errorf("unmarshal failed %s", err)
	}

	obj, err := jmespath.Search(expr, entity)
	if err != nil {
		return "", fmt.Errorf("jmes search failed %s", err)
	}

	if s, ok := obj.(string); ok {
		return s, nil
	} else {
		return "", fmt.Errorf("object %v isn't string", obj)
	}
}
