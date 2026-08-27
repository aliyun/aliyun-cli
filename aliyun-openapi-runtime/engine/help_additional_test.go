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

package engine

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestHelpEntryValidation(t *testing.T) {
	product := &meta.Product{}
	index := &meta.APIIndex{}
	if err := printProductHelp(nil, product, index, "en", false); err == nil {
		t.Fatal("printProductHelp(nil writer) succeeded")
	}
	if err := printProductHelp(io.Discard, nil, index, "en", false); err == nil {
		t.Fatal("printProductHelp(nil product) succeeded")
	}
	if err := printProductHelp(io.Discard, product, nil, "en", false); err == nil {
		t.Fatal("printProductHelp(nil index) succeeded")
	}
	if err := printAPIVersionsHelp(nil, product, "en"); err == nil {
		t.Fatal("printAPIVersionsHelp(nil writer) succeeded")
	}
	if err := printAPIVersionsHelp(io.Discard, nil, "en"); err == nil {
		t.Fatal("printAPIVersionsHelp(nil product) succeeded")
	}
	if err := printAPIVersions(nil, product, "en"); err == nil {
		t.Fatal("printAPIVersions(nil writer) succeeded")
	}
	if err := printAPIVersions(io.Discard, nil, "en"); err == nil {
		t.Fatal("printAPIVersions(nil product) succeeded")
	}
}

func TestPrintAPIVersionsInBothLanguages(t *testing.T) {
	product := &meta.Product{
		Code: "demo-service", Versions: []string{"2024-01-01", "2020-01-01"}, DefaultVersion: "2024-01-01",
	}
	for _, test := range []struct {
		lang  string
		wants []string
	}{
		{"en", []string{"Description: List supported API versions", "Product: demo-service", "* 2024-01-01 (default)", "ALIBABA_CLOUD_DEMO_SERVICE_API_VERSION"}},
		{"zh", []string{"描述: 列出支持的 API 版本", "产品: demo-service", "* 2024-01-01（默认）", "ALIBABA_CLOUD_DEMO_SERVICE_API_VERSION"}},
	} {
		var buf bytes.Buffer
		if err := printAPIVersionsHelp(&buf, product, test.lang); err != nil {
			t.Fatal(err)
		}
		if err := printAPIVersions(&buf, product, test.lang); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		for _, want := range test.wants {
			if !strings.Contains(out, want) {
				t.Errorf("%s output missing %q:\n%s", test.lang, want, out)
			}
		}
		if strings.Index(out, "2020-01-01") > strings.Index(out, "2024-01-01") {
			t.Errorf("versions were not sorted:\n%s", out)
		}
	}
	if !reflect.DeepEqual(product.Versions, []string{"2024-01-01", "2020-01-01"}) {
		t.Fatalf("printAPIVersions mutated product versions: %v", product.Versions)
	}
}

func TestPrintProductHelpWithoutOptionalText(t *testing.T) {
	var buf bytes.Buffer
	product := &meta.Product{Code: "Demo"}
	index := &meta.APIIndex{Version: "v1", Entries: map[string]meta.APIIndexEntry{"hidden": {}}}
	if err := printProductHelp(&buf, product, index, "zh", false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "产品: demo") || strings.Contains(out, "可用命令:") {
		t.Fatalf("product help = %q", out)
	}
}

func TestCompositeHelpHints(t *testing.T) {
	stringField := meta.Parameter{Name: "name", RawName: "Name", Type: meta.TypeString}
	intField := meta.Parameter{Name: "count", Type: meta.TypeInteger}
	flatObject := &meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{stringField, intField}}
	nestedObject := &meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{
		{Name: "items", Type: meta.TypeArray, ItemType: &meta.Parameter{Type: meta.TypeString}},
	}}

	if !isScalar(&stringField) || isScalar(flatObject) || isFlatObject(nil) || isFlatObject(&meta.Parameter{Type: meta.TypeObject}) || !isFlatObject(flatObject) || isFlatObject(nestedObject) {
		t.Fatal("scalar/flat-object classification returned unexpected values")
	}
	if got := objectFieldNames(nil); got != nil {
		t.Fatalf("objectFieldNames(nil) = %v", got)
	}
	if got := objectFieldNames(flatObject); !reflect.DeepEqual(got, []string{"Name", "count"}) {
		t.Fatalf("objectFieldNames() = %v", got)
	}
	if objectFieldName(&stringField) != "Name" || objectFieldName(&intField) != "count" {
		t.Fatal("objectFieldName returned unexpected value")
	}

	structures := []struct {
		parameter *meta.Parameter
		want      string
	}{
		{nil, ""},
		{&meta.Parameter{Type: meta.TypeArray}, "[]"},
		{&meta.Parameter{Type: meta.TypeArray, ItemType: &meta.Parameter{Type: meta.TypeBoolean}}, "[bool, ...]"},
		{&meta.Parameter{Type: meta.TypeMap}, "map[string]string"},
		{&meta.Parameter{Type: meta.TypeMap, ValueType: flatObject}, "map[string]{Name: string, count: int}"},
		{flatObject, "{Name: string, count: int}"},
	}
	for _, test := range structures {
		if got := describeStructure(test.parameter); got != test.want {
			t.Errorf("describeStructure(%#v) = %q, want %q", test.parameter, got, test.want)
		}
	}

	hints := []struct {
		parameter meta.Parameter
		contains  string
	}{
		{meta.Parameter{Type: meta.TypeString}, ""},
		{meta.Parameter{Type: meta.TypeArray}, "value1 value2 value3"},
		{meta.Parameter{Type: meta.TypeArray, ItemType: &meta.Parameter{Type: meta.TypeString}}, "value1 value2 value3"},
		{meta.Parameter{Type: meta.TypeArray, ItemType: flatObject}, "Name=a"},
		{meta.Parameter{Type: meta.TypeArray, ItemType: nestedObject}, "'value'"},
		{*flatObject, "Name=xxx"},
		{*nestedObject, "'value'"},
		{meta.Parameter{Type: meta.TypeMap}, "key1=value1"},
		{meta.Parameter{Type: meta.TypeMap, ValueType: &meta.Parameter{Type: meta.TypeString}}, "key1=value1"},
		{meta.Parameter{Type: meta.TypeMap, ValueType: flatObject}, "'value'"},
	}
	for _, test := range hints {
		structure, format := describeHint("--config, --cfg", &test.parameter)
		combined := structure + " " + format
		if test.contains != "" && !strings.Contains(combined, test.contains) {
			t.Errorf("describeHint(%s) = %q, want substring %q", test.parameter.Type, combined, test.contains)
		}
		if test.parameter.Type == meta.TypeString && combined != " " {
			t.Errorf("scalar hint = %q", combined)
		}
	}
	if got := primaryOption("--config, --cfg"); got != "--config" {
		t.Fatalf("primaryOption(joined) = %q", got)
	}
	if got := kvExample([]string{"A", "B", "C"}, []string{"x", "y"}); got != "A=x B=y C=x" {
		t.Fatalf("kvExample() = %q", got)
	}
}

func TestHelpTypeNamesWrappingAndWidths(t *testing.T) {
	types := map[meta.DataType]string{
		meta.TypeArray: "list", meta.TypeObject: "object", meta.TypeMap: "map",
		meta.TypeInteger: "int", meta.TypeLong: "int", meta.TypeFloat: "float",
		meta.TypeBoolean: "bool", meta.TypeAny: "string",
	}
	for typ, want := range types {
		if got := typeName(&meta.Parameter{Type: typ}); got != want {
			t.Errorf("typeName(%s) = %q, want %q", typ, got, want)
		}
	}

	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "")
	if getMaxLineLength() != 80 {
		t.Fatal("default max line length should be 80")
	}
	for value, want := range map[string]int{"60": 60, "bad": 80, "0": 80, "-1": 80} {
		t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", value)
		if got := getMaxLineLength(); got != want {
			t.Errorf("max line length %q = %d, want %d", value, got, want)
		}
	}

	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "40")
	var subcommands bytes.Buffer
	printSubCommandWithDescription(&subcommands, "empty", "", 10)
	printSubCommandWithDescription(&subcommands, "long", "This description is deliberately long, with punctuation, so it wraps.", 10)
	if strings.Count(subcommands.String(), "\n") < 3 {
		t.Fatalf("subcommand output did not wrap: %q", subcommands.String())
	}

	var wrapped bytes.Buffer
	tw := tabwriter.NewWriter(&wrapped, 0, 0, 2, ' ', 0)
	printWrappedLine(tw, "--short", "short", 10)
	printWrappedLine(tw, "--long", "A deliberately long sentence, with punctuation, that must wrap over multiple aligned lines.", 10)
	_ = tw.Flush()
	if strings.Count(wrapped.String(), "\n") < 3 {
		t.Fatalf("wrapped output did not wrap: %q", wrapped.String())
	}
	if got := normalizeHelpLine("  multiple   spaces\n and lines "); got != "multiple spaces and lines" {
		t.Fatalf("normalizeHelpLine() = %q", got)
	}
}
