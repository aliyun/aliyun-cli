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

package format

import (
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestDecodeAPIJSONRejectsFlattenedCompositeShapes(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name: "array element_type",
			payload: `{
				"name":"TestArray",
				"operation":{},
				"parameters":[{"name":"items","raw_name":"Items","type":"array","element_type":"string"}]
			}`,
			want: "parameters[0]: array is missing element",
		},
		{
			name: "map value_type",
			payload: `{
				"name":"TestMap",
				"operation":{},
				"parameters":[{"name":"labels","raw_name":"Labels","type":"map","value_type":"string"}]
			}`,
			want: "parameters[0]: map is missing value",
		},
		{
			name: "nested array inner_element_type",
			payload: `{
				"name":"TestNested",
				"operation":{},
				"parameters":[{"name":"groups","raw_name":"Groups","type":"map","value":{"type":"array","inner_element_type":"string"}}]
			}`,
			want: "parameters[0].value: array is missing element",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeAPIJSON([]byte(test.payload), test.name)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("DecodeAPIJSON() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestDecodeAPIJSONAcceptsRecursiveCompositeShapes(t *testing.T) {
	payload := []byte(`{
		"name":"TestRecursive",
		"operation":{},
		"parameters":[{
			"name":"groups",
			"raw_name":"Groups",
			"type":"map",
			"value":{"type":"array","element":{"type":"string"}}
		}]
	}`)

	api, err := DecodeAPIJSON(payload, "recursive")
	if err != nil {
		t.Fatal(err)
	}
	if len(api.Parameters) != 1 || api.Parameters[0].ValueType == nil ||
		api.Parameters[0].ValueType.ItemType == nil ||
		api.Parameters[0].ValueType.ItemType.Type != meta.TypeString {
		t.Fatalf("unexpected recursive parameter: %#v", api.Parameters)
	}
}
