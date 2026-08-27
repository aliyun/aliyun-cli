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

package argparser

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestCoerceDecodedScalarBranches(t *testing.T) {
	tests := []struct {
		name    string
		typ     meta.DataType
		value   any
		want    any
		wantErr bool
	}{
		{"any", meta.TypeAny, map[string]any{"x": true}, map[string]any{"x": true}, false},
		{"nil string", meta.TypeString, nil, "", false},
		{"string", meta.TypeString, "value", "value", false},
		{"string conversion", meta.TypeString, 12, "12", false},
		{"nil bool", meta.TypeBoolean, nil, false, false},
		{"bool", meta.TypeBoolean, true, true, false},
		{"bool string", meta.TypeBoolean, "yes", true, false},
		{"bool number", meta.TypeBoolean, json.Number("0"), false, false},
		{"bad bool", meta.TypeBoolean, []string{"x"}, nil, true},
		{"nil integer", meta.TypeInteger, nil, json.Number("0"), false},
		{"integer number", meta.TypeInteger, json.Number("12"), json.Number("12"), false},
		{"long string", meta.TypeLong, " 13 ", json.Number("13"), false},
		{"bad integer", meta.TypeInteger, true, nil, true},
		{"nil float", meta.TypeFloat, nil, float64(0), false},
		{"float number", meta.TypeFloat, json.Number("1.5"), float64(1.5), false},
		{"float string", meta.TypeFloat, " 2.5 ", float64(2.5), false},
		{"bad float", meta.TypeFloat, true, nil, true},
		{"unknown type", meta.DataType("custom"), 42, 42, false},
	}
	for _, test := range tests {
		got, err := coerceDecodedScalar(test.typ, test.value)
		if (err != nil) != test.wantErr {
			t.Fatalf("%s: error = %v, wantErr %v", test.name, err, test.wantErr)
		}
		if !test.wantErr && !reflect.DeepEqual(got, test.want) {
			t.Fatalf("%s: value = %#v, want %#v", test.name, got, test.want)
		}
	}
}

func TestArgumentParserLowLevelValidation(t *testing.T) {
	if _, err := resolveWire(nil); err == nil {
		t.Fatal("resolveWire(nil) succeeded")
	}
	for _, parameter := range []*meta.Parameter{{}, {Name: "named"}} {
		if _, err := resolveWire(parameter); err == nil {
			t.Fatalf("resolveWire(%#v) succeeded", parameter)
		}
	}
	if got, err := resolveWire(&meta.Parameter{RawName: "Wire"}); err != nil || got != "Wire" {
		t.Fatalf("resolveWire(valid) = %q, %v", got, err)
	}

	for _, rest := range []string{"key", "[1", "[x]", "[-1]"} {
		if _, _, err := splitIndex(rest); err == nil {
			t.Fatalf("splitIndex(%q) succeeded", rest)
		}
	}
	if index, rest, err := splitIndex("[12].Name"); err != nil || index != 12 || rest != "Name" {
		t.Fatalf("splitIndex(valid) = %d, %q, %v", index, rest, err)
	}

	for _, raw := range []string{"value", "{bad}"} {
		if _, err := decodeJSONObject(raw); err == nil {
			t.Fatalf("decodeJSONObject(%q) succeeded", raw)
		}
	}
	object, err := decodeJSONObject(`'{"count":9007199254740993}'`)
	if err != nil || object["count"] != json.Number("9007199254740993") {
		t.Fatalf("decodeJSONObject(valid) = %#v, %v", object, err)
	}
	for _, raw := range []string{"value", "[bad]"} {
		if _, err := decodeJSONArray(raw); err == nil {
			t.Fatalf("decodeJSONArray(%q) succeeded", raw)
		}
	}
	array, err := decodeJSONArray(`"[1,2]"`)
	if err != nil || len(array) != 2 {
		t.Fatalf("decodeJSONArray(valid) = %#v, %v", array, err)
	}
}

func TestNumericAndJSONDetectionEdges(t *testing.T) {
	for _, value := range []string{"0", "-1", "+1", "1.5", "1e3", "1E-3"} {
		if !looksNumeric(value) {
			t.Errorf("looksNumeric(%q) = false", value)
		}
	}
	for _, value := range []string{"", "1.2.3", "1e2e3", "1-2", "abc"} {
		if looksNumeric(value) {
			t.Errorf("looksNumeric(%q) = true", value)
		}
	}
	for _, value := range []string{`{"x":1}`, `[1]`, `"text"`} {
		if !isLikelyJSON(value) {
			t.Errorf("isLikelyJSON(%q) = false", value)
		}
	}
	for _, value := range []string{"", "x", "{x", "true"} {
		if isLikelyJSON(value) {
			t.Errorf("isLikelyJSON(%q) = true", value)
		}
	}
	for _, value := range []string{"0", "-1.5", "1e3"} {
		if !isValidJSONNumberLiteral(value) {
			t.Errorf("isValidJSONNumberLiteral(%q) = false", value)
		}
	}
	for _, value := range []string{"01", "true", `"1"`, "bad"} {
		if isValidJSONNumberLiteral(value) {
			t.Errorf("isValidJSONNumberLiteral(%q) = true", value)
		}
	}
}

func TestParseAnyFallbacks(t *testing.T) {
	tests := map[string]any{
		"": "", " true ": true, "false": false, "null": nil,
		"9007199254740993": json.Number("9007199254740993"),
		`{"x":1}`:          map[string]any{"x": json.Number("1")},
		`[1]`:              []any{json.Number("1")},
		`"text"`:           "text",
		"{bad}":            "{bad}",
	}
	for input, want := range tests {
		if got := parseAny(input); !reflect.DeepEqual(got, want) {
			t.Errorf("parseAny(%q) = %#v, want %#v", input, got, want)
		}
	}
}
