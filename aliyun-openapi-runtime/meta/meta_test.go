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

package meta

import (
	"reflect"
	"sort"
	"testing"
)

func TestAPIFindParameterPrefersRawName(t *testing.T) {
	api := &API{Parameters: []Parameter{
		{Name: "shared", RawName: "First"},
		{Name: "second", RawName: "shared"},
	}}

	if got := api.FindParameter("shared"); got == nil || got.Name != "second" {
		t.Fatalf("FindParameter(raw name) = %#v, want second parameter", got)
	}
	if got := api.FindParameter("First"); got == nil || got.Name != "shared" {
		t.Fatalf("FindParameter(first raw name) = %#v, want first parameter", got)
	}
	if got := api.FindParameter("missing"); got != nil {
		t.Fatalf("FindParameter(missing) = %#v, want nil", got)
	}
}

func TestParameterCompositeAndWalkFields(t *testing.T) {
	for _, typ := range []DataType{TypeObject, TypeArray, TypeMap} {
		if !((Parameter{Type: typ}).IsComposite()) {
			t.Fatalf("IsComposite(%q) = false", typ)
		}
	}
	if (Parameter{Type: TypeString}).IsComposite() {
		t.Fatal("string parameter reported as composite")
	}

	tree := Parameter{Type: TypeObject, Fields: []Parameter{
		{Name: "plain", Type: TypeString},
		{Name: "skipped", Type: TypeObject, Fields: []Parameter{{Name: "hidden", Type: TypeString}}},
		{Name: "array", Type: TypeArray, ItemType: &Parameter{
			Name: "item", Type: TypeObject, Fields: []Parameter{{Name: "item_field", Type: TypeString}},
		}},
		{Name: "map", Type: TypeMap, ValueType: &Parameter{Name: "value", Type: TypeString}},
		{Name: "nil_array", Type: TypeArray},
		{Name: "nil_map", Type: TypeMap},
	}}

	var visited []string
	tree.WalkFields(func(parameter Parameter) bool {
		visited = append(visited, parameter.Name)
		return parameter.Name != "skipped"
	})
	want := []string{"plain", "skipped", "array", "item", "item_field", "map", "value", "nil_array", "nil_map"}
	if !reflect.DeepEqual(visited, want) {
		t.Fatalf("WalkFields() = %v, want %v", visited, want)
	}
}

func TestProductAndAPIIndexHelpers(t *testing.T) {
	product := &Product{Versions: []string{"v1", "v2"}}
	if !product.HasVersion("v2") || product.HasVersion("v3") {
		t.Fatalf("HasVersion returned unexpected result")
	}

	var nilIndex *APIIndex
	if nilIndex.Names() != nil || nilIndex.ResolveCmd("list-things") != "" {
		t.Fatal("nil APIIndex should return zero values")
	}
	nilIndex.BuildCmdIndex()

	index := &APIIndex{Entries: map[string]APIIndexEntry{
		"ListThings": {APIName: "ListThings", CmdName: "list-things"},
		"Hidden":     {APIName: "Hidden"},
	}}
	if got := index.ResolveCmd("list-things"); got != "ListThings" {
		t.Fatalf("ResolveCmd() before BuildCmdIndex = %q", got)
	}
	if got := index.ResolveCmd(""); got != "" {
		t.Fatalf("ResolveCmd(empty) = %q", got)
	}

	index.BuildCmdIndex()
	if got := index.ResolveCmd("list-things"); got != "ListThings" {
		t.Fatalf("ResolveCmd() after BuildCmdIndex = %q", got)
	}
	if got := index.ResolveCmd("missing"); got != "" {
		t.Fatalf("ResolveCmd(missing) = %q", got)
	}

	names := index.Names()
	sort.Strings(names)
	if want := []string{"Hidden", "ListThings"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("Names() = %v, want %v", names, want)
	}
	if (&APIIndex{}).Names() != nil {
		t.Fatal("empty APIIndex Names() should be nil")
	}
}

func TestValueHelpers(t *testing.T) {
	tests := []struct {
		description Description
		lang        string
		want        string
	}{
		{Description{ZH: "中文", EN: "English"}, "zh", "中文"},
		{Description{ZH: "", EN: "English"}, "zh", "English"},
		{Description{ZH: "中文", EN: "English"}, "en", "English"},
		{Description{ZH: "中文", EN: ""}, "en", "中文"},
		{Description{ZH: "中文", EN: "English"}, "unknown", "English"},
	}
	for _, test := range tests {
		if got := test.description.Localized(test.lang); got != test.want {
			t.Fatalf("Localized(%q) = %q, want %q", test.lang, got, test.want)
		}
	}

	endpoints := Endpoints{
		Global: "global.example.com",
		Public: map[string]string{"cn-test": "public.example.com"},
		VPC:    map[string]string{"cn-test": "vpc.example.com"},
	}
	if got := endpoints.Resolve("cn-test", true); got != "vpc.example.com" {
		t.Fatalf("VPC Resolve() = %q", got)
	}
	if got := endpoints.Resolve("cn-test", false); got != "public.example.com" {
		t.Fatalf("public Resolve() = %q", got)
	}
	if got := endpoints.Resolve("unknown", true); got != "global.example.com" {
		t.Fatalf("global Resolve() = %q", got)
	}

	ref := APIRef{Product: "ecs", Version: "2014-05-26", Name: "DescribeInstances"}
	if got := ref.String(); got != "ecs/2014-05-26/DescribeInstances" {
		t.Fatalf("APIRef.String() = %q", got)
	}
	if ref.IsZero() || !(APIRef{}).IsZero() {
		t.Fatal("APIRef.IsZero() returned unexpected result")
	}
}
