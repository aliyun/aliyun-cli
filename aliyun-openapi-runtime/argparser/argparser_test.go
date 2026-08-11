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
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

// schema mirrors acc/CreateImageCache plus a couple of synthetic
// composite params to exercise every branch.
func schema() []meta.Parameter {
	return []meta.Parameter{
		{Name: "region_id", RawName: "RegionId", Type: meta.TypeString, Options: []string{"--biz-region-id"}},
		{Name: "image_cache_name", RawName: "ImageCacheName", Type: meta.TypeString, Options: []string{"--image-cache-name"}},
		{Name: "count", RawName: "Count", Type: meta.TypeInteger, Options: []string{"--count"}},
		{Name: "big_id", RawName: "BigId", Type: meta.TypeLong, Options: []string{"--big-id"}},
		{Name: "threshold", RawName: "Threshold", Type: meta.TypeFloat, Options: []string{"--threshold"}},
		{Name: "enabled", RawName: "Enabled", Type: meta.TypeBoolean, Options: []string{"--enabled"}},
		{
			Name: "images", RawName: "Images", Type: meta.TypeArray, Options: []string{"--images"},
			ItemType: &meta.Parameter{Type: meta.TypeString},
		},
		{
			Name: "tags", RawName: "Tags", Type: meta.TypeArray, Options: []string{"--tags"},
			ItemType: &meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{
				{Name: "key", RawName: "Key", Type: meta.TypeString},
				{Name: "value", RawName: "Value", Type: meta.TypeString},
				{Name: "weight", RawName: "Weight", Type: meta.TypeInteger},
			}},
		},
		{
			Name: "network_config", RawName: "NetworkConfig", Type: meta.TypeObject, Options: []string{"--network-config"},
			Fields: []meta.Parameter{
				{Name: "vswitch_id", RawName: "VSwitchId", Type: meta.TypeString},
				{Name: "security_group_id", RawName: "SecurityGroupId", Type: meta.TypeString},
				{Name: "port", RawName: "Port", Type: meta.TypeInteger},
				{Name: "threshold", RawName: "Threshold", Type: meta.TypeFloat},
				{Name: "enabled", RawName: "Enabled", Type: meta.TypeBoolean},
				{Name: "acc", RawName: "Acc", Type: meta.TypeObject, Fields: []meta.Parameter{
					{Name: "level", RawName: "Level", Type: meta.TypeInteger},
				}},
				{
					Name: "rules", RawName: "Rules", Type: meta.TypeArray,
					ItemType: &meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{
						{Name: "name", RawName: "Name", Type: meta.TypeString},
						{Name: "weight", RawName: "Weight", Type: meta.TypeInteger},
					}},
				},
				{
					Name: "ports", RawName: "Ports", Type: meta.TypeArray,
					ItemType: &meta.Parameter{Type: meta.TypeInteger},
				},
				{Name: "spec", RawName: "Spec", Type: meta.TypeObject, Fields: []meta.Parameter{
					{Name: "cpu", RawName: "Cpu", Type: meta.TypeInteger},
				}},
			},
		},
		{Name: "payload", RawName: "Payload", Type: meta.TypeAny, Options: []string{"--payload"}},
		{
			Name: "labels", RawName: "Labels", Type: meta.TypeMap, Options: []string{"--labels"},
			ValueType: &meta.Parameter{Type: meta.TypeString},
		},
		{
			Name: "scores", RawName: "Scores", Type: meta.TypeMap, Options: []string{"--scores"},
			ValueType: &meta.Parameter{Type: meta.TypeInteger},
		},
		{
			Name: "job_file", RawName: "JobFile", Type: meta.TypeMap, Options: []string{"--job-file"}, ParamStyle: "json",
			ValueType: &meta.Parameter{Type: meta.TypeAny},
		},
	}
}

func mustParse(t *testing.T, args ...string) *Result {
	t.Helper()
	res, err := Parse(schema(), args)
	if err != nil {
		t.Fatalf("Parse(%v): %v", args, err)
	}
	return res
}

func TestScalarByOption(t *testing.T) {
	res := mustParse(t, "--biz-region-id", "cn-hangzhou")
	if res.Args["RegionId"] != "cn-hangzhou" {
		t.Fatalf("region_id = %v", res.Args["RegionId"])
	}
	res = mustParse(t, "--image-cache-name", "cache1")
	if res.Args["ImageCacheName"] != "cache1" {
		t.Fatalf("image_cache_name = %v", res.Args["ImageCacheName"])
	}
}

func TestOnlyOptionsAcceptedAsFlag(t *testing.T) {
	for _, flag := range []string{"RegionId", "region-id"} {
		_, err := Parse(schema(), []string{"--" + flag, "cn-beijing"})
		if err == nil {
			t.Fatalf("expected unknown flag --%s", flag)
		}
		var ufe *UnknownFlagError
		if !errors.As(err, &ufe) || ufe.Flag != flag {
			t.Fatalf("got %v, want UnknownFlagError{Flag:%s}", err, flag)
		}
	}
}

func TestScalarInlineEquals(t *testing.T) {
	res := mustParse(t, "--image-cache-name=with=equals")
	if res.Args["ImageCacheName"] != "with=equals" {
		t.Fatalf("inline value = %v", res.Args["ImageCacheName"])
	}
}

func TestCompositeParameterRejectsInlineEquals(t *testing.T) {
	_, err := Parse(schema(), []string{"--network-config={\"Port\":80}"})
	if err == nil || !strings.Contains(err.Error(), "does not support an inline value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNumericPreservedAsJSONNumber(t *testing.T) {
	res := mustParse(t, "--count", "42", "--big-id", "9007199254740993")
	if got, ok := res.Args["Count"].(json.Number); !ok || got.String() != "42" {
		t.Fatalf("count = %#v", res.Args["Count"])
	}
	// 2^53+1 would lose precision as float64; json.Number keeps it.
	if got, ok := res.Args["BigId"].(json.Number); !ok || got.String() != "9007199254740993" {
		t.Fatalf("big_id = %#v", res.Args["BigId"])
	}
}

func TestFloatCoercedToFloat64(t *testing.T) {
	res := mustParse(t, "--threshold", "5.0")
	got, ok := res.Args["Threshold"].(float64)
	if !ok || got != 5 {
		t.Fatalf("threshold = %#v (%T), want float64(5)", res.Args["Threshold"], res.Args["Threshold"])
	}

	encoded, err := json.Marshal(res.Args["Threshold"])
	if err != nil {
		t.Fatalf("marshal threshold: %v", err)
	}
	if string(encoded) != "5" {
		t.Fatalf("encoded threshold = %s, want 5", encoded)
	}
}

func TestNestedFloatCoercedToFloat64(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  float64
	}{
		{input: "Threshold=15.0", want: 15},
		{input: `{"Threshold":200.0}`, want: 200},
	} {
		res := mustParse(t, "--network-config", tt.input)
		network := res.Args["NetworkConfig"].(map[string]any)
		got, ok := network["Threshold"].(float64)
		if !ok {
			t.Fatalf("input %q: threshold = %#v (%T), want float64", tt.input, network["Threshold"], network["Threshold"])
		}
		if got != tt.want {
			t.Fatalf("input %q: threshold = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFloatSupportsDecimalAndExponent(t *testing.T) {
	for _, tt := range []struct {
		raw  string
		want float64
	}{
		{raw: "5.25", want: 5.25},
		{raw: "5e2", want: 500},
	} {
		res := mustParse(t, "--threshold", tt.raw)
		if got, ok := res.Args["Threshold"].(float64); !ok || got != tt.want {
			t.Fatalf("threshold %q = %#v, want float64(%v)", tt.raw, res.Args["Threshold"], tt.want)
		}
	}
}

func TestFloatRejectsInvalidAndOutOfRangeValues(t *testing.T) {
	for _, raw := range []string{"not-a-number", "1e10000"} {
		if _, err := Parse(schema(), []string{"--threshold", raw}); err == nil {
			t.Fatalf("Parse accepted invalid float %q", raw)
		}
	}
}

// TestBoolean pins the plugin-parity contract: API-level booleans are
// typed Go bools so JSON bodies emit true/false literals rather than
// quoted strings.
func TestBoolean(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"t", true},
		{"yes", true},
		{"y", true},
		{"1", true},
		{"false", false},
		{"f", false},
		{"no", false},
		{"n", false},
		{"0", false},
	}
	for _, tt := range tests {
		res := mustParse(t, "--enabled", tt.input)
		if res.Args["Enabled"] != tt.want {
			t.Errorf("--enabled %s = %#v (want %t)", tt.input, res.Args["Enabled"], tt.want)
		}
	}

	_, err := Parse(schema(), []string{"--enabled", "not-a-bool"})
	if err == nil || !strings.Contains(err.Error(), "invalid boolean value") {
		t.Fatalf("invalid boolean error = %v", err)
	}
}

func TestArrayScalarRepeatedMultiAndLiteralComma(t *testing.T) {
	// Repeated flag form.
	res := mustParse(t, "--images", "a", "--images", "b")
	if !reflect.DeepEqual(res.Args["Images"], []any{"a", "b"}) {
		t.Fatalf("images repeated = %#v", res.Args["Images"])
	}
	// Single flag, multiple tokens.
	res = mustParse(t, "--images", "a", "b", "c")
	if !reflect.DeepEqual(res.Args["Images"], []any{"a", "b", "c"}) {
		t.Fatalf("images multi = %#v", res.Args["Images"])
	}
	// A comma in a non-JSON scalar value is literal string content.
	res = mustParse(t, "--images", "a,b,c")
	if !reflect.DeepEqual(res.Args["Images"], []any{"a,b,c"}) {
		t.Fatalf("images literal comma = %#v", res.Args["Images"])
	}
}

func TestArrayOfArrayJSONForms(t *testing.T) {
	params := []meta.Parameter{{
		Name: "process_nodes", RawName: "ProcessNodes", Type: meta.TypeArray, Options: []string{"--process-nodes"},
		ItemType: &meta.Parameter{Type: meta.TypeArray, ItemType: &meta.Parameter{Type: meta.TypeString}},
	}}
	want := []any{[]any{"a", "b"}, []any{"c"}}

	complete, err := Parse(params, []string{"--process-nodes", `[["a","b"],["c"]]`})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(complete.Args["ProcessNodes"], want) {
		t.Fatalf("complete array = %#v, want %#v", complete.Args["ProcessNodes"], want)
	}

	repeated, err := Parse(params, []string{"--process-nodes", `["a","b"]`, "--process-nodes", `["c"]`})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(repeated.Args["ProcessNodes"], want) {
		t.Fatalf("repeated inner arrays = %#v, want %#v", repeated.Args["ProcessNodes"], want)
	}
}

func TestCompositeMapRequiresCompleteJSON(t *testing.T) {
	params := []meta.Parameter{{
		Name: "partitions", RawName: "Partitions", Type: meta.TypeMap, Options: []string{"--partitions"},
		ValueType: &meta.Parameter{Type: meta.TypeMap, ValueType: &meta.Parameter{Type: meta.TypeString}},
	}}

	parsed, err := Parse(params, []string{"--partitions", `{"p1":{"key":"value"}}`})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"p1": map[string]any{"key": "value"}}
	if !reflect.DeepEqual(parsed.Args["Partitions"], want) {
		t.Fatalf("composite map = %#v, want %#v", parsed.Args["Partitions"], want)
	}

	_, err = Parse(params, []string{"--partitions", `p1={"key":"value"}`})
	if err == nil || !strings.Contains(err.Error(), "require a complete JSON object") {
		t.Fatalf("key=value composite map error = %v", err)
	}
}

func TestObjectMapFieldRequiresCompleteJSON(t *testing.T) {
	params := []meta.Parameter{{
		Name: "config", RawName: "Config", Type: meta.TypeObject, Options: []string{"--config"},
		Fields: []meta.Parameter{{
			Name: "labels", RawName: "Labels", Type: meta.TypeMap,
			ValueType: &meta.Parameter{Type: meta.TypeString},
		}},
	}}

	_, err := Parse(params, []string{"--config", "Labels.env=prod"})
	if err == nil || !strings.Contains(err.Error(), "set the complete map field as JSON") {
		t.Fatalf("map path error = %v", err)
	}

	parsed, err := Parse(params, []string{"--config", `Labels={"env":"prod"}`})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"Labels": map[string]any{"env": "prod"}}
	if !reflect.DeepEqual(parsed.Args["Config"], want) {
		t.Fatalf("map JSON field = %#v, want %#v", parsed.Args["Config"], want)
	}
}

func TestArrayOfObjectRepeatable(t *testing.T) {
	// Sub-field keys are addressed by their RawName verbatim (no
	// kebab/snake conversion), and emitted under the same RawName.
	res := mustParse(t, "--tags", "Key=k1", "Value=v1", "--tags", "Key=k2", "Value=v2")
	want := []any{
		map[string]any{"Key": "k1", "Value": "v1"},
		map[string]any{"Key": "k2", "Value": "v2"},
	}
	if !reflect.DeepEqual(res.Args["Tags"], want) {
		t.Fatalf("tags = %#v", res.Args["Tags"])
	}
}

// TestSubFieldNoFormatConversion pins that nested keys are NOT converted:
// a kebab/snake spelling of a RawName field is unknown and rejected.
func TestSubFieldNoFormatConversion(t *testing.T) {
	_, err := Parse(schema(), []string{"--tags", "key=k1"})
	if err == nil || !strings.Contains(err.Error(), "unknown field: key") {
		t.Fatalf("error = %v, want unknown field: key", err)
	}
}

func TestDeclaredObjectRejectsUnknownFields(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "object key value", args: []string{"--network-config", "Unknown=x"}},
		{name: "object JSON", args: []string{"--network-config", `{"Unknown":"x"}`}},
		{name: "nested object key value", args: []string{"--network-config", "Acc.Unknown=x"}},
		{name: "nested object JSON", args: []string{"--network-config", `{"Acc":{"Unknown":"x"}}`}},
		{name: "object JSON leaf", args: []string{"--network-config", `Spec={"Unknown":1}`}},
		{name: "array object key value", args: []string{"--tags", "Unknown=x"}},
		{name: "array object indexed", args: []string{"--network-config", "Rules[0].Unknown=x"}},
		{name: "array object JSON", args: []string{"--tags", `[{"Unknown":"x"}]`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(schema(), tt.args)
			if err == nil || !strings.Contains(err.Error(), "unknown field: Unknown") {
				t.Fatalf("error = %v, want unknown field: Unknown", err)
			}
		})
	}
}

func TestObjectWithoutFieldsRemainsOpen(t *testing.T) {
	params := []meta.Parameter{{
		Name: "open", RawName: "Open", Type: meta.TypeObject, Options: []string{"--open"},
	}}

	res, err := Parse(params, []string{"--open", "free=value", "nested.key=x"})
	if err != nil {
		t.Fatalf("key=value Parse: %v", err)
	}
	want := map[string]any{"free": "value", "nested": map[string]any{"key": "x"}}
	if !reflect.DeepEqual(res.Args["Open"], want) {
		t.Fatalf("Open = %#v, want %#v", res.Args["Open"], want)
	}

	res, err = Parse(params, []string{"--open", `{"free":1,"nested":{"key":true}}`})
	if err != nil {
		t.Fatalf("JSON Parse: %v", err)
	}
	open := res.Args["Open"].(map[string]any)
	if got, ok := open["free"].(json.Number); !ok || got.String() != "1" {
		t.Fatalf("Open.free = %#v", open["free"])
	}
	if nested, ok := open["nested"].(map[string]any); !ok || nested["key"] != true {
		t.Fatalf("Open.nested = %#v", open["nested"])
	}
}

// TestMissingRawNameErrors: top-level (and nested) parameters without
// RawName in metadata are rejected — Args keys are strictly RawName.
func TestMissingRawNameErrors(t *testing.T) {
	params := []meta.Parameter{
		{Name: "broken", Type: meta.TypeString, Options: []string{"--broken"}},
	}
	_, err := Parse(params, []string{"--broken", "x"})
	if err == nil {
		t.Fatal("expected error for parameter missing raw_name")
	}
	if !strings.Contains(err.Error(), "raw_name") {
		t.Fatalf("error = %v, want mention of raw_name", err)
	}
}

func TestObjectMergedAcrossOccurrences(t *testing.T) {
	res := mustParse(t, "--network-config", "VSwitchId=vsw-1", "--network-config", "SecurityGroupId=sg-1")
	want := map[string]any{"VSwitchId": "vsw-1", "SecurityGroupId": "sg-1"}
	if !reflect.DeepEqual(res.Args["NetworkConfig"], want) {
		t.Fatalf("NetworkConfig = %#v", res.Args["NetworkConfig"])
	}
}

func TestMap(t *testing.T) {
	res := mustParse(t, "--labels", "env=prod", "region=cn")
	want := map[string]any{"env": "prod", "region": "cn"}
	if !reflect.DeepEqual(res.Args["Labels"], want) {
		t.Fatalf("labels = %#v", res.Args["Labels"])
	}
}

// TestNestedFieldTypeCoercion verifies object field values (including
// nested objects) are coerced to the field's declared type, not left
// as raw strings — so JSON-body/ROA APIs get real numbers/booleans.
func TestNestedFieldTypeCoercion(t *testing.T) {
	res := mustParse(t, "--network-config", "Port=8080", "Enabled=true", "Acc.Level=3")
	nc, ok := res.Args["NetworkConfig"].(map[string]any)
	if !ok {
		t.Fatalf("network_config = %#v", res.Args["NetworkConfig"])
	}
	if got, ok := nc["Port"].(json.Number); !ok || got.String() != "8080" {
		t.Fatalf("Port = %#v (want json.Number 8080)", nc["Port"])
	}
	if nc["Enabled"] != true {
		t.Fatalf("Enabled = %#v (want bool true)", nc["Enabled"])
	}
	acc, ok := nc["Acc"].(map[string]any)
	if !ok {
		t.Fatalf("Acc = %#v", nc["Acc"])
	}
	if got, ok := acc["Level"].(json.Number); !ok || got.String() != "3" {
		t.Fatalf("Acc.Level = %#v (want json.Number 3)", acc["Level"])
	}
}

// TestArrayObjectFieldCoercion verifies typed fields inside an
// array<object> element are coerced too.
func TestArrayObjectFieldCoercion(t *testing.T) {
	res := mustParse(t, "--tags", "Key=k1", "Weight=5")
	arr, ok := res.Args["Tags"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("tags = %#v", res.Args["Tags"])
	}
	obj := arr[0].(map[string]any)
	if got, ok := obj["Weight"].(json.Number); !ok || got.String() != "5" {
		t.Fatalf("Weight = %#v (want json.Number 5)", obj["Weight"])
	}
}

// TestMapValueCoercion verifies map values are coerced to ValueType.
func TestMapValueCoercion(t *testing.T) {
	res := mustParse(t, "--scores", "a=1", "b=2")
	scores, ok := res.Args["Scores"].(map[string]any)
	if !ok {
		t.Fatalf("scores = %#v", res.Args["Scores"])
	}
	for k, want := range map[string]string{"a": "1", "b": "2"} {
		if got, ok := scores[k].(json.Number); !ok || got.String() != want {
			t.Fatalf("scores[%q] = %#v (want json.Number %s)", k, scores[k], want)
		}
	}
}

// TestArrayIndexPath covers items[i].key / items[i] inside an object,
// matching the plugin's setNestedValue array-index support.
func TestArrayIndexPath(t *testing.T) {
	res := mustParse(t,
		"--network-config",
		"Rules[0].Name=r0", "Rules[0].Weight=1",
		"Rules[1].Name=r1",
		"Ports[0]=80", "Ports[1]=443",
	)
	nc := res.Args["NetworkConfig"].(map[string]any)
	rules, ok := nc["Rules"].([]any)
	if !ok || len(rules) != 2 {
		t.Fatalf("Rules = %#v", nc["Rules"])
	}
	r0 := rules[0].(map[string]any)
	if r0["Name"] != "r0" {
		t.Fatalf("Rules[0].Name = %#v", r0["Name"])
	}
	if got, _ := r0["Weight"].(json.Number); got.String() != "1" {
		t.Fatalf("Rules[0].Weight = %#v", r0["Weight"])
	}
	if rules[1].(map[string]any)["Name"] != "r1" {
		t.Fatalf("Rules[1] = %#v", rules[1])
	}
	ports, ok := nc["Ports"].([]any)
	if !ok || len(ports) != 2 {
		t.Fatalf("Ports = %#v", nc["Ports"])
	}
	if got, _ := ports[0].(json.Number); got.String() != "80" {
		t.Fatalf("Ports[0] = %#v", ports[0])
	}
}

// TestFieldLevelJSONFallback covers object/array field leaves given a
// JSON literal, and an array-of-object element given a JSON object.
func TestFieldLevelJSONFallback(t *testing.T) {
	res := mustParse(t,
		"--network-config",
		`Spec={"Cpu":4}`,
		`Ports=[1,2,3]`,
		`Rules[0]={"Name":"j0","Weight":9}`,
	)
	nc := res.Args["NetworkConfig"].(map[string]any)

	spec, ok := nc["Spec"].(map[string]any)
	if !ok {
		t.Fatalf("Spec = %#v", nc["Spec"])
	}
	if got, _ := spec["Cpu"].(json.Number); got.String() != "4" {
		t.Fatalf("Spec.Cpu = %#v", spec["Cpu"])
	}
	ports, ok := nc["Ports"].([]any)
	if !ok || len(ports) != 3 {
		t.Fatalf("Ports = %#v", nc["Ports"])
	}
	if got, _ := ports[2].(json.Number); got.String() != "3" {
		t.Fatalf("Ports[2] = %#v", ports[2])
	}
	rules := nc["Rules"].([]any)
	r0 := rules[0].(map[string]any)
	if r0["Name"] != "j0" {
		t.Fatalf("Rules[0] JSON = %#v", r0)
	}
	if got, _ := r0["Weight"].(json.Number); got.String() != "9" {
		t.Fatalf("Rules[0].Weight = %#v", r0["Weight"])
	}
}

// TestAnyTypeSmartParse covers `any` parameters: JSON object/array,
// bool/null literals, numbers (json.Number) and raw string fallback.
func TestAnyTypeSmartParse(t *testing.T) {
	res := mustParse(t, "--payload", `{"a":1}`)
	if m, ok := res.Args["Payload"].(map[string]any); !ok {
		t.Fatalf("payload obj = %#v", res.Args["Payload"])
	} else if got, _ := m["a"].(json.Number); got.String() != "1" {
		t.Fatalf("payload.a = %#v", m["a"])
	}

	res = mustParse(t, "--payload", "true")
	if res.Args["Payload"] != true {
		t.Fatalf("payload bool = %#v", res.Args["Payload"])
	}

	res = mustParse(t, "--payload", "123")
	if got, ok := res.Args["Payload"].(json.Number); !ok || got.String() != "123" {
		t.Fatalf("payload num = %#v", res.Args["Payload"])
	}

	res = mustParse(t, "--payload", "null")
	if got, exists := res.Args["Payload"]; !exists || got != nil {
		t.Fatalf("payload null = %#v (exists=%v)", got, exists)
	}

	res = mustParse(t, "--payload", "hello")
	if res.Args["Payload"] != "hello" {
		t.Fatalf("payload str = %#v", res.Args["Payload"])
	}

	res = mustParse(t, "--payload", `"123"`)
	if got, ok := res.Args["Payload"].(string); !ok || got != "123" {
		t.Fatalf("payload quoted str = %#v", res.Args["Payload"])
	}

	res = mustParse(t, "--payload", `[1,"x"]`)
	if arr, ok := res.Args["Payload"].([]any); !ok || len(arr) != 2 {
		t.Fatalf("payload arr = %#v", res.Args["Payload"])
	}
}

func TestMapAnyKeepsInvalidJSONNumbersAsStrings(t *testing.T) {
	res := mustParse(t,
		"--job-file",
		"sign=000",
		"positive=+1",
		"fraction=.5",
		"trailing=1.",
		"count=12",
		"ratio=0.5",
	)
	jobFile := res.Args["JobFile"].(map[string]any)
	for key, want := range map[string]string{
		"sign": "000", "positive": "+1", "fraction": ".5", "trailing": "1.",
	} {
		if got, ok := jobFile[key].(string); !ok || got != want {
			t.Fatalf("JobFile[%q] = %#v, want string %q", key, jobFile[key], want)
		}
	}
	for key, want := range map[string]string{"count": "12", "ratio": "0.5"} {
		if got, ok := jobFile[key].(json.Number); !ok || got.String() != want {
			t.Fatalf("JobFile[%q] = %#v, want json.Number(%s)", key, jobFile[key], want)
		}
	}
	if _, err := json.Marshal(jobFile); err != nil {
		t.Fatalf("json.Marshal(JobFile) failed: %v", err)
	}
}

// TestFlagLevelJSONObject: an object flag given whole JSON is parsed as
// JSON-first, with CLI field names mapped to wire RawNames.
func TestFlagLevelJSONObject(t *testing.T) {
	res := mustParse(t, "--network-config", `{"VSwitchId":"vsw-9","Port":8080,"Acc":{"Level":2}}`)
	nc, ok := res.Args["NetworkConfig"].(map[string]any)
	if !ok {
		t.Fatalf("network_config = %#v", res.Args["NetworkConfig"])
	}
	if nc["VSwitchId"] != "vsw-9" {
		t.Fatalf("VSwitchId = %#v", nc["VSwitchId"])
	}
	if got, _ := nc["Port"].(json.Number); got.String() != "8080" {
		t.Fatalf("Port = %#v", nc["Port"])
	}
	acc, ok := nc["Acc"].(map[string]any)
	if !ok {
		t.Fatalf("Acc = %#v", nc["Acc"])
	}
	if got, _ := acc["Level"].(json.Number); got.String() != "2" {
		t.Fatalf("Acc.Level = %#v", acc["Level"])
	}
}

// TestFlagLevelJSONUsesDeclaredScalarTypes pins that whole-JSON input follows
// the same implicit scalar conversions as key=value input. The JSON syntax
// must not change the resulting request types.
func TestFlagLevelJSONUsesDeclaredScalarTypes(t *testing.T) {
	res := mustParse(t, "--network-config", `{
		"VSwitchId":123,
		"Port":"8080",
		"Enabled":"yes",
		"Acc":{"Level":"2"},
		"Ports":["80",443]
	}`)
	nc := res.Args["NetworkConfig"].(map[string]any)
	if nc["VSwitchId"] != "123" {
		t.Fatalf("VSwitchId = %#v (want string 123)", nc["VSwitchId"])
	}
	if got, ok := nc["Port"].(json.Number); !ok || got.String() != "8080" {
		t.Fatalf("Port = %#v (want json.Number 8080)", nc["Port"])
	}
	if nc["Enabled"] != true {
		t.Fatalf("Enabled = %#v (want bool true)", nc["Enabled"])
	}
	if got, ok := nc["Acc"].(map[string]any)["Level"].(json.Number); !ok || got.String() != "2" {
		t.Fatalf("Acc.Level = %#v (want json.Number 2)", nc["Acc"].(map[string]any)["Level"])
	}
	ports := nc["Ports"].([]any)
	for i, want := range []string{"80", "443"} {
		if got, ok := ports[i].(json.Number); !ok || got.String() != want {
			t.Fatalf("Ports[%d] = %#v (want json.Number %s)", i, ports[i], want)
		}
	}

	res = mustParse(t, "--labels", `{"numeric":1,"boolean":true,"null":null}`)
	labels := res.Args["Labels"].(map[string]any)
	if labels["numeric"] != "1" || labels["boolean"] != "true" || labels["null"] != "" {
		t.Fatalf("string map conversion = %#v", labels)
	}
}

// TestFlagLevelJSONArrayExpands: a JSON array on an array-of-object flag
// expands into multiple elements; a JSON object becomes one element.
// Repeated occurrences accumulate.
func TestFlagLevelJSONArrayExpands(t *testing.T) {
	res := mustParse(t,
		"--tags", `[{"Key":"k1","Value":"v1"},{"Key":"k2","Weight":7}]`,
		"--tags", `{"Key":"k3"}`,
	)
	arr, ok := res.Args["Tags"].([]any)
	if !ok || len(arr) != 3 {
		t.Fatalf("tags = %#v", res.Args["Tags"])
	}
	e0 := arr[0].(map[string]any)
	if e0["Key"] != "k1" || e0["Value"] != "v1" {
		t.Fatalf("tags[0] = %#v", e0)
	}
	e1 := arr[1].(map[string]any)
	if got, _ := e1["Weight"].(json.Number); got.String() != "7" {
		t.Fatalf("tags[1].Weight = %#v", e1["Weight"])
	}
	if arr[2].(map[string]any)["Key"] != "k3" {
		t.Fatalf("tags[2] = %#v", arr[2])
	}
}

func TestFlagLevelExplicitEmptyJSONArrayIsPreserved(t *testing.T) {
	res := mustParse(t, "--tags", `[]`)
	tags, ok := res.Args["Tags"].([]any)
	if !ok {
		t.Fatalf("tags type = %T, want []any", res.Args["Tags"])
	}
	if tags == nil {
		t.Fatal("tags is a nil slice, want an explicit empty slice")
	}
	if len(tags) != 0 {
		t.Fatalf("len(tags) = %d, want 0", len(tags))
	}
}

// TestFlagLevelJSONScalarArray: a JSON array on a scalar-array flag
// expands too (superset of the plugin's default branch).
func TestFlagLevelJSONScalarArray(t *testing.T) {
	res := mustParse(t, "--images", `["a,b","c"]`)
	if !reflect.DeepEqual(res.Args["Images"], []any{"a,b", "c"}) {
		t.Fatalf("images = %#v", res.Args["Images"])
	}
}

func TestFlagLevelJSONObjectRejectedForScalarArray(t *testing.T) {
	for _, itemType := range []meta.DataType{
		meta.TypeString,
		meta.TypeInteger,
		meta.TypeLong,
		meta.TypeFloat,
		meta.TypeBoolean,
	} {
		t.Run(string(itemType), func(t *testing.T) {
			params := []meta.Parameter{{
				Name: "items", RawName: "Items", Type: meta.TypeArray,
				Options: []string{"--items"}, ItemType: &meta.Parameter{Type: itemType},
			}}
			res, err := Parse(params, []string{"--items", `{}`})
			if err == nil {
				t.Fatalf("Parse accepted scalar-array JSON object: %#v", res.Args["Items"])
			}
			if !strings.Contains(err.Error(), "expected JSON array") {
				t.Fatalf("Parse error = %q, want expected JSON array", err)
			}
		})
	}
}

func TestFlagLevelJSONObjectAcceptedForAnyArray(t *testing.T) {
	params := []meta.Parameter{{
		Name: "items", RawName: "Items", Type: meta.TypeArray,
		Options: []string{"--items"}, ItemType: &meta.Parameter{Type: meta.TypeAny},
	}}
	res, err := Parse(params, []string{"--items", `{"key":"value"}`})
	if err != nil {
		t.Fatalf("Parse any-array JSON object: %v", err)
	}
	want := []any{map[string]any{"key": "value"}}
	if !reflect.DeepEqual(res.Args["Items"], want) {
		t.Fatalf("items = %#v, want %#v", res.Args["Items"], want)
	}
}

func TestFlagLevelJSONObjectStringRemainsExpressibleInScalarArray(t *testing.T) {
	res := mustParse(t, "--images", `["{}"]`)
	if !reflect.DeepEqual(res.Args["Images"], []any{"{}"}) {
		t.Fatalf("images = %#v", res.Args["Images"])
	}
}

// TestFlagLevelJSONMap: a map flag accepts whole JSON, values coerced.
func TestFlagLevelJSONMap(t *testing.T) {
	res := mustParse(t, "--scores", `{"a":1,"b":2}`)
	scores, ok := res.Args["Scores"].(map[string]any)
	if !ok {
		t.Fatalf("scores = %#v", res.Args["Scores"])
	}
	if got, _ := scores["a"].(json.Number); got.String() != "1" {
		t.Fatalf("scores.a = %#v", scores["a"])
	}
}

func TestFlagLevelJSONRequiresCompleteValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "invalid trailing text", value: `{"a":"1"} trailing`},
		{name: "second JSON value", value: `{"a":"1"}{"b":"2"}`},
		{name: "malformed JSON", value: `{"a":"1"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(schema(), []string{"--labels", tt.value})
			if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
				t.Fatalf("error = %v, want invalid JSON", err)
			}
		})
	}

	res := mustParse(t, "--labels", `{"a":"1"}   `)
	if !reflect.DeepEqual(res.Args["Labels"], map[string]any{"a": "1"}) {
		t.Fatalf("labels = %#v", res.Args["Labels"])
	}
}

// TestFlagLevelJSONFallsBackToKV: a non-JSON object flag still parses
// as key=value (JSON-first must not break the classic form).
func TestFlagLevelJSONFallsBackToKV(t *testing.T) {
	res := mustParse(t, "--network-config", "VSwitchId=vsw-1", "Port=70")
	nc := res.Args["NetworkConfig"].(map[string]any)
	if nc["VSwitchId"] != "vsw-1" {
		t.Fatalf("VSwitchId = %#v", nc["VSwitchId"])
	}
	if got, _ := nc["Port"].(json.Number); got.String() != "70" {
		t.Fatalf("Port = %#v", nc["Port"])
	}
}

// TestHelpShorthand verifies -h sets Help (parity with the --help /
// -h contract on Reserved).
func TestHelpShorthand(t *testing.T) {
	res := mustParse(t, "-h")
	if !res.Reserved.Help {
		t.Fatal("expected -h to set Reserved.Help")
	}
	res = mustParse(t, "--image-cache-name", "c1", "-h")
	if !res.Reserved.Help {
		t.Fatal("expected trailing -h to set Reserved.Help")
	}
}

// TestDashPrefixedValue is the whole reason this parser exists: values
// beginning with '-' must be accepted, which the legacy cli.Parser
// cannot do.
func TestDashPrefixedValue(t *testing.T) {
	res := mustParse(t, "--image-cache-name", "-1/-1")
	if res.Args["ImageCacheName"] != "-1/-1" {
		t.Fatalf("dash value = %v", res.Args["ImageCacheName"])
	}
	certificate := "-----BEGIN_CERTIFICATE-----MIIDrzCCApeg-----END_CERTIFICATE-----"
	res = mustParse(t, "--image-cache-name", certificate)
	if res.Args["ImageCacheName"] != certificate {
		t.Fatalf("certificate value = %v", res.Args["ImageCacheName"])
	}
	unregisteredLongToken := "--not-a-registered-flag"
	res = mustParse(t, "--image-cache-name", unregisteredLongToken)
	if res.Args["ImageCacheName"] != unregisteredLongToken {
		t.Fatalf("unregistered long token value = %v", res.Args["ImageCacheName"])
	}
	// Negative number into a numeric field.
	res = mustParse(t, "--count", "-5")
	if got, _ := res.Args["Count"].(json.Number); got.String() != "-5" {
		t.Fatalf("negative count = %#v", res.Args["Count"])
	}
}

func TestReservedFlags(t *testing.T) {
	res := mustParse(t,
		"--region", "cn-hangzhou",
		"--endpoint", "ecs.example.com",
		"--api-version", "2014-05-26",
		"--cli-dry-run",
		"--image-cache-name", "c1",
	)
	if res.Reserved.Region != "cn-hangzhou" {
		t.Fatalf("region = %q", res.Reserved.Region)
	}
	if res.Reserved.Endpoint != "ecs.example.com" {
		t.Fatalf("endpoint = %q", res.Reserved.Endpoint)
	}
	if res.Reserved.Version != "2014-05-26" {
		t.Fatalf("version = %q", res.Reserved.Version)
	}
	if !res.Reserved.DryRun {
		t.Fatal("cli-dry-run not set")
	}
	if res.Reserved.DryRunJSON {
		t.Fatal("cli-dry-run must not set DryRunJSON")
	}
	// The API param still lands.
	if res.Args["ImageCacheName"] != "c1" {
		t.Fatalf("image_cache_name = %v", res.Args["ImageCacheName"])
	}
	// Reserved names never leak into API args.
	if _, ok := res.Args["region"]; ok {
		t.Fatal("reserved --region leaked into API args")
	}
}

func TestReservedFlagsHelpFiltersHiddenEntries(t *testing.T) {
	flags := ReservedFlags()
	names := make([]string, len(flags))
	for i, flag := range flags {
		names[i] = flag.Name
		if flag.DescZH == "" || flag.DescEN == "" {
			t.Fatalf("visible reserved flag %q is missing help text", flag.Name)
		}
	}
	want := []string{
		"cli-dry-run",
		"region",
		"endpoint",
		"api-version",
		"cli-query",
		"log-level",
		"quiet",
		"pager",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("visible reserved flags = %v, want %v", names, want)
	}
}

func TestReservedDryRunVariants(t *testing.T) {
	// --cli-dry-run-json implies DryRun + DryRunJSON.
	res := mustParse(t, "--cli-dry-run-json", "--image-cache-name", "c1")
	if !res.Reserved.DryRun || !res.Reserved.DryRunJSON {
		t.Fatalf("cli-dry-run-json: DryRun=%v DryRunJSON=%v", res.Reserved.DryRun, res.Reserved.DryRunJSON)
	}
}

func TestDryRunIsAPIParamNotReserved(t *testing.T) {
	// --dry-run must NOT be a CLI reserved switch; it belongs to API
	// params (DryRun). Coexist with --cli-dry-run for preflight.
	params := []meta.Parameter{
		{Name: "image_cache_name", RawName: "ImageCacheName", Type: meta.TypeString, Options: []string{"--image-cache-name"}},
		{Name: "dry_run", RawName: "DryRun", Type: meta.TypeBoolean, Options: []string{"--dry-run"}},
		{Name: "encrypted", RawName: "Encrypted", Type: meta.TypeBoolean, Options: []string{"--encrypted"}},
	}
	res, err := Parse(params, []string{
		"--cli-dry-run",
		"--dry-run", "true",
		"--encrypted", "false",
		"--image-cache-name", "c1",
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !res.Reserved.DryRun {
		t.Fatal("--cli-dry-run must set Reserved.DryRun")
	}
	if res.Reserved.DryRunJSON {
		t.Fatal("--cli-dry-run must not set DryRunJSON")
	}
	if got := res.Args["DryRun"]; got != true {
		t.Fatalf("API DryRun = %#v, want true", got)
	}
	if got := res.Args["Encrypted"]; got != false {
		t.Fatalf("Encrypted = %#v, want false", got)
	}
}

func TestReservedPager(t *testing.T) {
	// Bare --pager enables aggregation with empty sub-fields.
	res := mustParse(t, "--pager", "--image-cache-name", "c1")
	if res.Reserved.Pager == nil {
		t.Fatal("bare --pager must set Reserved.Pager")
	}
	if res.Reserved.Pager.Path != "" || res.Reserved.Pager.PageNumber != "" {
		t.Fatalf("bare pager should be empty: %+v", res.Reserved.Pager)
	}

	// --all-pages is an alias.
	res = mustParse(t, "--all-pages", "--image-cache-name", "c1")
	if res.Reserved.Pager == nil {
		t.Fatal("bare --all-pages must set Reserved.Pager")
	}

	res = mustParse(t,
		"--pager", "path=Data.Items[]", "PageSize=MaxResults", "NextToken=NextToken",
		"--image-cache-name", "c1",
	)
	p := res.Reserved.Pager
	if p == nil {
		t.Fatal("expected pager")
	}
	if p.Path != "Data.Items[]" || p.PageSize != "MaxResults" || p.NextToken != "NextToken" {
		t.Fatalf("pager = %+v", p)
	}
	if _, ok := res.Args["pager"]; ok {
		t.Fatal("pager must not leak into API args")
	}
}

func TestReservedWaiter(t *testing.T) {
	res := mustParse(t,
		"--waiter", "expr=Status", "to=Running", "timeout=60", "interval=2",
		"--image-cache-name", "c1",
	)
	w := res.Reserved.Waiter
	if w == nil {
		t.Fatal("expected waiter")
	}
	if w.Expr != "Status" || w.To != "Running" || w.Timeout != 60 || w.Interval != 2 {
		t.Fatalf("waiter = %+v", w)
	}
}

func TestReservedLogLevel(t *testing.T) {
	res := mustParse(t, "--log-level", "DEBUG", "--image-cache-name", "c1")
	if res.Reserved.LogLevel != "DEBUG" {
		t.Fatalf("log-level = %q", res.Reserved.LogLevel)
	}
	if _, ok := res.Args["log_level"]; ok {
		t.Fatal("log-level must not leak into API args")
	}
}

func TestReservedCliQueryAndQuiet(t *testing.T) {
	res := mustParse(t, "--cli-query", "Data.Id", "-q", "--image-cache-name", "c1")
	if res.Reserved.CliQuery != "Data.Id" {
		t.Fatalf("cli-query = %q", res.Reserved.CliQuery)
	}
	if !res.Reserved.Quiet {
		t.Fatal("expected quiet from -q")
	}
}

func TestReservedOutputTable(t *testing.T) {
	_, err := Parse(schema(), []string{"--output", "json", "--image-cache-name", "c1"})
	if err == nil {
		t.Fatal("expected --output json to be rejected (plugin: object form only)")
	}
	res := mustParse(t, "--output", "cols=Id,Name", "rows=Instances", "num=true", "--image-cache-name", "c1")
	if res.Reserved.OutputTable == nil || len(res.Reserved.OutputTable.Cols) != 2 {
		t.Fatalf("table output = %+v", res.Reserved.OutputTable)
	}
	if !res.Reserved.OutputTable.ShowNum || res.Reserved.OutputTable.Rows != "Instances" {
		t.Fatalf("table cfg = %+v", res.Reserved.OutputTable)
	}
	res = mustParse(t, "-o", "cols=Id", "--image-cache-name", "c1")
	if res.Reserved.OutputTable == nil || len(res.Reserved.OutputTable.Cols) != 1 {
		t.Fatalf("-o short alias = %+v", res.Reserved.OutputTable)
	}
}

func TestReservedHeaderBodyEstimate(t *testing.T) {
	res := mustParse(t,
		"--header", "X-A=1", "--header", "X-B=2",
		"--body", `{"k":1}`,
		"--secure",
		"--estimate-cost",
		"--estimate-cost-context", "Traffic=10",
		"--no-stream",
		"--image-cache-name", "c1",
	)
	if len(res.Reserved.Headers) != 2 || res.Reserved.Body == "" || !res.Reserved.BodySet || !res.Reserved.Secure {
		t.Fatalf("reserved = %+v", res.Reserved)
	}
	if !res.Reserved.EstimateCost || len(res.Reserved.EstimateCostContext) != 1 || !res.Reserved.NoStream {
		t.Fatalf("estimate/no-stream = %+v", res.Reserved)
	}
}

func TestParserDoesNotValidateEstimateCostOptionDependency(t *testing.T) {
	res := mustParse(t,
		"--estimate-cost-context", "Traffic=10",
		"--image-cache-name", "c1",
	)
	if len(res.Reserved.EstimateCostContext) != 1 || res.Reserved.EstimateCost {
		t.Fatalf("estimate context = %+v", res.Reserved)
	}
}

func TestReservedEmptyBodyStillEnablesEscapeHatch(t *testing.T) {
	res := mustParse(t, "--body", "", "--image-cache-name", "c1")
	if !res.Reserved.BodySet || res.Reserved.Body != "" {
		t.Fatalf("reserved body = %+v", res.Reserved)
	}
}

func TestUnknownFlag(t *testing.T) {
	_, err := Parse(schema(), []string{"--nope", "x"})
	var ufe *UnknownFlagError
	if !errors.As(err, &ufe) {
		t.Fatalf("expected UnknownFlagError, got %v", err)
	}
	if ufe.Flag != "nope" {
		t.Fatalf("flag = %q", ufe.Flag)
	}
}

func TestUnknownFlagAfterScalarIsNotAbsorbedAsAnotherValue(t *testing.T) {
	params := []meta.Parameter{{
		Name: "email", RawName: "Email", Type: meta.TypeString, Options: []string{"--email"},
	}}
	_, err := Parse(params, []string{"--email", "a0@gmail.com", "--timestamp", "example-string"})
	var ufe *UnknownFlagError
	if !errors.As(err, &ufe) {
		t.Fatalf("expected UnknownFlagError, got %v", err)
	}
	if ufe.Flag != "timestamp" {
		t.Fatalf("flag = %q, want timestamp", ufe.Flag)
	}
}

func TestScalarMissingValueBeforeRegisteredFlag(t *testing.T) {
	_, err := Parse(schema(), []string{"--image-cache-name", "--biz-region-id", "cn-hangzhou"})
	if err == nil || err.Error() != "--image-cache-name expects a value" {
		t.Fatalf("error = %v", err)
	}
}

func TestParameterOptionIsCaseSensitive(t *testing.T) {
	_, err := Parse(schema(), []string{"--Biz-Region-Id", "cn-hangzhou"})
	var ufe *UnknownFlagError
	if !errors.As(err, &ufe) {
		t.Fatalf("expected mixed-case option to be rejected, got %v", err)
	}
	if ufe.Flag != "Biz-Region-Id" {
		t.Fatalf("flag = %q", ufe.Flag)
	}
}

func TestParseWithOptionsConsumesExternalFlags(t *testing.T) {
	opts := ParseOptions{ExternalFlags: []ExternalFlagSpec{
		{Name: "profile", Shorthand: 'p', Mode: ExternalFlagOptional},
		{Name: "read-timeout", Mode: ExternalFlagRequired},
		{Name: "cli-ai-mode", Mode: ExternalFlagNone},
	}}
	res, err := ParseWithOptions(schema(), []string{
		"--profile", "prod",
		"--biz-region-id", "cn-hangzhou",
		"--read-timeout=30",
		"--cli-ai-mode",
	}, opts)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := res.Args["RegionId"]; got != "cn-hangzhou" {
		t.Fatalf("RegionId = %v", got)
	}
}

func TestParseWithOptionsConsumesExternalColonValue(t *testing.T) {
	opts := ParseOptions{ExternalFlags: []ExternalFlagSpec{{
		Name: "read-timeout",
		Mode: ExternalFlagRequired,
	}}}
	res, err := ParseWithOptions(schema(), []string{
		"--read-timeout:30",
		"--biz-region-id", "cn-hangzhou",
	}, opts)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := res.Args["RegionId"]; got != "cn-hangzhou" {
		t.Fatalf("RegionId = %v", got)
	}
}

func TestParseWithOptionsExternalBoundaries(t *testing.T) {
	opts := ParseOptions{ExternalFlags: []ExternalFlagSpec{
		{Name: "profile", Shorthand: 'p', Mode: ExternalFlagOptional},
	}}
	res, err := ParseWithOptions(schema(), []string{
		"--image-cache-name", "demo",
		"-p=prod",
		"--profile",
		"--biz-region-id", "cn-hangzhou",
	}, opts)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := res.Args["ImageCacheName"]; got != "demo" {
		t.Fatalf("ImageCacheName = %v", got)
	}
	if got := res.Args["RegionId"]; got != "cn-hangzhou" {
		t.Fatalf("RegionId = %v", got)
	}
}

func TestParseWithOptionsRejectsExternalFlag(t *testing.T) {
	opts := ParseOptions{ExternalFlags: []ExternalFlagSpec{{
		Name:          "RegionId",
		Mode:          ExternalFlagRequired,
		RejectMessage: "use --region or --biz-region-id",
	}}}
	_, err := ParseWithOptions(schema(), []string{"--RegionId", "cn-hangzhou"}, opts)
	if err == nil || !strings.Contains(err.Error(), "use --region or --biz-region-id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseWithOptionsRequiresExternalValue(t *testing.T) {
	opts := ParseOptions{ExternalFlags: []ExternalFlagSpec{{
		Name: "read-timeout",
		Mode: ExternalFlagRequired,
	}}}
	_, err := ParseWithOptions(schema(), []string{"--read-timeout", "--biz-region-id", "cn-hangzhou"}, opts)
	if err == nil || !strings.Contains(err.Error(), "--read-timeout requires a value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseWithOptionsFlagPriority(t *testing.T) {
	opts := ParseOptions{ExternalFlags: []ExternalFlagSpec{
		// Reserved wins even if a host accidentally declares the same name.
		{Name: "quiet", Mode: ExternalFlagRequired},
		// External wins over an API metadata option.
		{Name: "biz-region-id", Mode: ExternalFlagNone},
	}}
	res, err := ParseWithOptions(schema(), []string{"--quiet", "--biz-region-id"}, opts)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !res.Reserved.Quiet {
		t.Fatal("--quiet was not parsed as an engine-reserved flag")
	}
	if _, ok := res.Args["RegionId"]; ok {
		t.Fatal("external --biz-region-id leaked into API arguments")
	}
}
