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
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/source/storage"
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

func TestJSONFormatDecodesIndexAndAPIFromVolume(t *testing.T) {
	fsys := fstest.MapFS{
		"demo/v1/version.json": {Data: []byte(`{
			"version":"v1",
			"apis":{"RunThing":{"cmd_name":"run-thing","description_en":"Run a thing","deprecated":true}}
		}`)},
		"demo/v1/RunThing.json": {Data: []byte(`{
			"name":"RunThing","cmd_name":"run-thing",
			"operation":{"action":"RunThing","api_version":"v1","method":"POST","api_style":"RPC"},
			"parameters":[{"name":"count","raw_name":"Count","type":"integer","options":["--count"]}]
		}`)},
	}
	volume, err := storage.NewFSStorage(fsys, "").Open("demo")
	if err != nil {
		t.Fatal(err)
	}
	format := NewJSONFormat()
	index, err := format.DecodeIndex(volume, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if index.Version != "v1" || index.ResolveCmd("run-thing") != "RunThing" || !index.Entries["RunThing"].Deprecated {
		t.Fatalf("DecodeIndex() = %#v", index)
	}
	api, err := format.DecodeAPI(volume, APIKey{Version: "v1", Name: "RunThing"})
	if err != nil {
		t.Fatal(err)
	}
	if api.Name != "RunThing" || api.Version != "v1" || len(api.Parameters) != 1 || api.Parameters[0].Type != meta.TypeInteger {
		t.Fatalf("DecodeAPI() = %#v", api)
	}
	if _, err := format.DecodeIndex(volume, "missing"); !errors.Is(err, storage.ErrEntryNotFound) {
		t.Fatalf("DecodeIndex(missing) error = %v", err)
	}
	if _, err := format.DecodeAPI(volume, APIKey{Version: "v1", Name: "Missing"}); !errors.Is(err, storage.ErrEntryNotFound) {
		t.Fatalf("DecodeAPI(missing) error = %v", err)
	}
}

func TestJSONFormatReportsDecodeAndShapeErrors(t *testing.T) {
	if _, err := DecodeAPIJSON([]byte("{"), "bad"); err == nil || !strings.Contains(err.Error(), "decode api bad") {
		t.Fatalf("DecodeAPIJSON(invalid) error = %v", err)
	}
	if _, err := DecodeCommandDefinition(nil, "nil"); err == nil || !strings.Contains(err.Error(), "nil command definition") {
		t.Fatalf("DecodeCommandDefinition(nil) error = %v", err)
	}
	if _, err := DecodeCommandDefinition(&schema.CommandDefinition{}, "missing"); err == nil || !strings.Contains(err.Error(), "missing operation") {
		t.Fatalf("DecodeCommandDefinition(no operation) error = %v", err)
	}

	tests := []struct {
		name string
		arg  schema.ArgumentDefinition
		want string
	}{
		{"array", schema.ArgumentDefinition{Type: " array "}, "array is missing element"},
		{"map", schema.ArgumentDefinition{Type: "MAP"}, "map is missing value"},
		{"object field", schema.ArgumentDefinition{Type: "object", Fields: []schema.ArgumentDefinition{{Type: "array"}}}, "fields[0]: array is missing element"},
		{"nested object field", schema.ArgumentDefinition{Type: "array", Element: &schema.TypeShape{Type: "object", Fields: []schema.ArgumentDefinition{{Type: "map"}}}}, "element.fields[0]: map is missing value"},
		{"nested map", schema.ArgumentDefinition{Type: "array", Element: &schema.TypeShape{Type: "map"}}, "element: map is missing value"},
	}
	for _, test := range tests {
		definition := &schema.CommandDefinition{Operation: &schema.OperationConfig{}, Parameters: []schema.ArgumentDefinition{test.arg}}
		if _, err := DecodeCommandDefinition(definition, test.name); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("%s: error = %v, want substring %q", test.name, err, test.want)
		}
	}
}

func TestJSONFormatManifestCatalogAndNumberPrecision(t *testing.T) {
	var manifest schema.PluginManifest
	if err := DecodePluginManifestJSON([]byte(`{"name":"aliyun-cli-demo","version":"1.0.0"}`), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "aliyun-cli-demo" || manifest.Version != "1.0.0" {
		t.Fatalf("manifest = %#v", manifest)
	}
	if err := DecodePluginManifestJSON([]byte("bad"), &manifest); err == nil {
		t.Fatal("invalid manifest succeeded")
	}

	var products schema.ProductsIndex
	if err := DecodeProductsJSON([]byte(`{"products":[{"code":"ecs","versions":["v1"]}]}`), &products); err != nil {
		t.Fatal(err)
	}
	if len(products.Products) != 1 || products.Products[0].Code != "ecs" {
		t.Fatalf("products = %#v", products)
	}
	if err := DecodeProductsJSON([]byte("bad"), &products); err == nil {
		t.Fatal("invalid products catalog succeeded")
	}

	var value struct{ ID json.Number }
	if err := decodeJSON([]byte(`{"ID":9007199254740993}`), &value); err != nil {
		t.Fatal(err)
	}
	if value.ID.String() != "9007199254740993" {
		t.Fatalf("decoded ID = %s", value.ID)
	}
}

func TestJSONFormatDecodeProduct(t *testing.T) {
	fsys := fstest.MapFS{
		"demo/2018-01-01/A.json": {Data: []byte("{}")},
		"demo/2020-01-01/B.json": {Data: []byte("{}")},
		"demo/manifest.json":     {Data: []byte("{}")},
	}
	volume, err := storage.NewFSStorage(fsys, "").Open("demo")
	if err != nil {
		t.Fatal(err)
	}
	product, err := (JSONFormat{}).DecodeProduct(volume, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"2018-01-01", "2020-01-01"}; !reflect.DeepEqual(product.Versions, want) || product.DefaultVersion != "2020-01-01" {
		t.Fatalf("DecodeProduct() = %#v", product)
	}

	emptyVolume, err := storage.NewFSStorage(fstest.MapFS{"empty/manifest.json": {Data: []byte("{}")}}, "").Open("empty")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (JSONFormat{}).DecodeProduct(emptyVolume, "empty"); err == nil || !strings.Contains(err.Error(), "no api-version directories") {
		t.Fatalf("DecodeProduct(empty) error = %v", err)
	}
	if chooseDefaultVersion(nil) != "" || chooseDefaultVersion([]string{"v2", "v1"}) != "v2" {
		t.Fatal("chooseDefaultVersion returned unexpected result")
	}
	if indexOfSlash("version/file") != 7 || indexOfSlash("file") != -1 {
		t.Fatal("indexOfSlash returned unexpected result")
	}
}
