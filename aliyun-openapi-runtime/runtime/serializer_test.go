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
	"reflect"
	"strings"
	"testing"
)

func TestSerializeQueryStylesAndErrors(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		isRPC   bool
		style   string
		want    map[string]string
		wantErr string
	}{
		{name: "json", value: map[string]any{"b": 2, "a": 1}, style: "json", want: map[string]string{"Param": `{"a":1,"b":2}`}},
		{name: "json marshal error", value: make(chan int), style: "json", wantErr: "marshal Param as json"},
		{name: "simple array", value: []any{"a", json.Number("2"), true}, style: "simple", want: map[string]string{"Param": "a,2,true"}},
		{name: "simple scalar", value: 3, style: "simple", want: map[string]string{"Param": "3"}},
		{name: "flat", value: []any{"a", "b"}, style: "flat", want: map[string]string{"Param.1": "a", "Param.2": "b"}},
		{name: "repeat list", value: map[string]any{"Name": "demo"}, style: "repeatList", want: map[string]string{"Param.Name": "demo"}},
		{name: "rpc default", value: map[string]any{"Name": "demo"}, isRPC: true, want: map[string]string{"Param.Name": "demo"}},
		{name: "rest composite", value: []any{"a", 2}, want: map[string]string{"Param": `["a",2]`}},
		{name: "rest marshal error", value: map[string]any{"bad": make(chan int)}, wantErr: "marshal Param"},
		{name: "rest scalar", value: false, want: map[string]string{"Param": "false"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := serializeQuery("Param", tc.value, tc.isRPC, tc.style)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("serializeQuery() error = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("serializeQuery() error = %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("serializeQuery() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestSerializeFormParameterStylesAndValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		style   string
		want    map[string]any
		wantErr string
	}{
		{name: "simple", value: []any{"a", json.Number("2"), true}, style: "simple", want: map[string]any{"Param": "a,2,true"}},
		{name: "simple empty", value: []any{}, style: "simple", want: map[string]any{}},
		{name: "simple requires array", value: "a", style: "simple", wantErr: "only supports arrays"},
		{name: "simple rejects nil", value: []any{"a", nil}, style: "simple", wantErr: "index 1"},
		{name: "simple rejects nested array", value: []any{[]any{"a"}}, style: "simple", wantErr: "index 0"},
		{name: "simple rejects object", value: []any{map[string]any{"a": 1}}, style: "simple", wantErr: "index 0"},
		{name: "json", value: map[string]any{"a": 1}, style: "json", want: map[string]any{"Param": `{"a":1}`}},
		{name: "json marshal error", value: make(chan int), style: "json", wantErr: "marshal Param as json"},
		{name: "unset", value: []any{"a"}, want: map[string]any{"Param": []any{"a"}}},
		{name: "flat", value: map[string]any{"a": 1}, style: "flat", want: map[string]any{"Param": map[string]any{"a": 1}}},
		{name: "repeat list", value: []any{"a"}, style: "repeatList", want: map[string]any{"Param": []any{"a"}}},
		{name: "unsupported", value: "a", style: "deepObject", wantErr: `unsupported formData style "deepObject"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := serializeFormParameter("Param", tc.value, tc.style)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("serializeFormParameter() error = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("serializeFormParameter() error = %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("serializeFormParameter() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestSerializeRPCAndBasicString(t *testing.T) {
	got := serializeRPC("Root", map[string]any{
		"Ignored": nil,
		"Items": []any{
			map[string]any{"Name": "first"},
			map[string]any{"Name": "second"},
		},
	})
	want := map[string]string{
		"Root.Items.1.Name": "first",
		"Root.Items.2.Name": "second",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("serializeRPC() = %#v, want %#v", got, want)
	}

	tests := []struct {
		value any
		want  string
	}{
		{"text", "text"},
		{json.Number("9007199254740993"), "9007199254740993"},
		{true, "true"},
		{int(12), "12"},
		{int64(13), "13"},
		{float64(1.25), "1.25"},
		{struct{ Name string }{Name: "demo"}, "{demo}"},
	}
	for _, tc := range tests {
		if got := basicString(tc.value); got != tc.want {
			t.Errorf("basicString(%#v) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
