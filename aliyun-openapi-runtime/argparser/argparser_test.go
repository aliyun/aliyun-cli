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

// TestBoolean pins the plugin-parity contract: API-level boolean
// parameters are kept as their verbatim string, not a Go bool, so
// ROA/body payloads emit "true" (quoted) like aliyun-cli-runtime.
func TestBoolean(t *testing.T) {
	res := mustParse(t, "--enabled", "true")
	if res.Args["Enabled"] != "true" {
		t.Fatalf("enabled = %#v (want string \"true\")", res.Args["Enabled"])
	}
}

func TestArrayScalarRepeatedAndMulti(t *testing.T) {
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
	// Comma convenience form.
	res = mustParse(t, "--images", "a,b,c")
	if !reflect.DeepEqual(res.Args["Images"], []any{"a", "b", "c"}) {
		t.Fatalf("images csv = %#v", res.Args["Images"])
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

// TestSubFieldNoFormatConversion pins that nested keys are NOT
// converted: a kebab/snake spelling of a RawName field is treated as an
// unknown (verbatim) key, never mapped to the RawName.
func TestSubFieldNoFormatConversion(t *testing.T) {
	res := mustParse(t, "--tags", "key=k1")
	obj := res.Args["Tags"].([]any)[0].(map[string]any)
	if _, mapped := obj["Key"]; mapped {
		t.Fatalf("lowercase 'key' must not map to RawName 'Key': %#v", obj)
	}
	if obj["key"] != "k1" {
		t.Fatalf("unknown key should pass through verbatim: %#v", obj)
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

func TestNestedDottedKey(t *testing.T) {
	res := mustParse(t, "--network-config", "meta.owner=alice", "meta.team=infra")
	want := map[string]any{"meta": map[string]any{"owner": "alice", "team": "infra"}}
	if !reflect.DeepEqual(res.Args["NetworkConfig"], want) {
		t.Fatalf("nested = %#v", res.Args["NetworkConfig"])
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
	// API-level bool stays a verbatim string (plugin parity).
	if nc["Enabled"] != "true" {
		t.Fatalf("Enabled = %#v (want string \"true\")", nc["Enabled"])
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

// TestFlagLevelJSONScalarArray: a JSON array on a scalar-array flag
// expands too (superset of the plugin's default branch).
func TestFlagLevelJSONScalarArray(t *testing.T) {
	res := mustParse(t, "--images", `["a","b","c"]`)
	if !reflect.DeepEqual(res.Args["Images"], []any{"a", "b", "c"}) {
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

func TestReservedDryRunVariants(t *testing.T) {
	// --cli-dry-run-json implies DryRun + DryRunJSON.
	res := mustParse(t, "--cli-dry-run-json", "--image-cache-name", "c1")
	if !res.Reserved.DryRun || !res.Reserved.DryRunJSON {
		t.Fatalf("cli-dry-run-json: DryRun=%v DryRunJSON=%v", res.Reserved.DryRun, res.Reserved.DryRunJSON)
	}
	// --dry-run is an ergonomic alias of --cli-dry-run (human mode).
	res = mustParse(t, "--dry-run", "--image-cache-name", "c1")
	if !res.Reserved.DryRun || res.Reserved.DryRunJSON {
		t.Fatalf("dry-run alias: DryRun=%v DryRunJSON=%v", res.Reserved.DryRun, res.Reserved.DryRunJSON)
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
	if len(res.Reserved.Headers) != 2 || res.Reserved.Body == "" || !res.Reserved.Secure {
		t.Fatalf("reserved = %+v", res.Reserved)
	}
	if !res.Reserved.EstimateCost || len(res.Reserved.EstimateCostContext) != 1 || !res.Reserved.NoStream {
		t.Fatalf("estimate/no-stream = %+v", res.Reserved)
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
