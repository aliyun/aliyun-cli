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

package source

import (
	"errors"
	"testing"
	"testing/fstest"
)

// apiJSON is a minimal canonical per-API definition.
func apiJSON(name, cmd, version string) string {
	return `{"name":"` + name + `","cmd_name":"` + cmd + `",` +
		`"operation":{"method":"POST","api_version":"` + version + `","action":"` + name + `","api_style":"RPC"}}`
}

// versionJSON is a minimal canonical per-version index.
func versionJSON(name, cmd, version string) string {
	return `{"version":"` + version + `","style":"RPC","apis":{"` + name +
		`":{"cmd_name":"` + cmd + `"}}}`
}

// TestGoPluginProductsExcluded verifies that targeted lookup abstains from
// products marked distribution=="go", while meta products remain routable.
func TestGoPluginProductsExcluded(t *testing.T) {
	fsys := fstest.MapFS{
		"metadatas/products.json": &fstest.MapFile{Data: []byte(`{"products":[
			{"code":"Ecs","plugin_default_version":"2014-05-26","versions":["2014-05-26"]},
			{"code":"Fc","plugin_default_version":"2023-03-30","versions":["2023-03-30"],"distribution":"go"}
		]}`)},
		"canonical/ecs/2014-05-26/version.json":           &fstest.MapFile{Data: []byte(versionJSON("DescribeInstances", "describe-instances", "2014-05-26"))},
		"canonical/ecs/2014-05-26/DescribeInstances.json": &fstest.MapFile{Data: []byte(apiJSON("DescribeInstances", "describe-instances", "2014-05-26"))},
		"canonical/fc/2023-03-30/version.json":            &fstest.MapFile{Data: []byte(versionJSON("ListFunctions", "list-functions", "2023-03-30"))},
		"canonical/fc/2023-03-30/ListFunctions.json":      &fstest.MapFile{Data: []byte(apiJSON("ListFunctions", "list-functions", "2023-03-30"))},
	}

	src := NewBaselineSource(fsys, "test")

	if _, _, err := src.LoadProduct("ecs"); err != nil {
		t.Fatalf("meta product must be served: %v", err)
	}
	if _, _, err := src.LoadProduct("fc"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Go plugin product must be excluded, got %v", err)
	}

	// The served product still resolves its commands via the index.
	idx, err := src.LoadIndex("ecs", "2014-05-26")
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if idx.ResolveCmd("describe-instances") != "DescribeInstances" {
		t.Error("ecs describe-instances should resolve")
	}
}

// TestUnmarkedProductIsServed confirms a product with no distribution
// field is served by the engine (an unpopulated catalog behaves as
// before the marker existed).
func TestUnmarkedProductIsServed(t *testing.T) {
	fsys := fstest.MapFS{
		"metadatas/products.json": &fstest.MapFile{Data: []byte(`{"products":[
			{"code":"Fc","plugin_default_version":"2023-03-30","versions":["2023-03-30"]}
		]}`)},
		"canonical/fc/2023-03-30/version.json":       &fstest.MapFile{Data: []byte(versionJSON("ListFunctions", "list-functions", "2023-03-30"))},
		"canonical/fc/2023-03-30/ListFunctions.json": &fstest.MapFile{Data: []byte(apiJSON("ListFunctions", "list-functions", "2023-03-30"))},
	}
	src := NewBaselineSource(fsys, "test")
	if _, _, err := src.LoadProduct("fc"); err != nil {
		t.Fatalf("an unmarked product must remain served: %v", err)
	}
	idx, err := src.LoadIndex("fc", "2023-03-30")
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if idx.ResolveCmd("list-functions") != "ListFunctions" {
		t.Error("an unmarked product's command must resolve")
	}
}

// TestEndpointsInjectedFromCatalog verifies product-level endpoints in
// products.json are attached to the loaded API.
func TestEndpointsInjectedFromCatalog(t *testing.T) {
	fsys := fstest.MapFS{
		"metadatas/products.json": &fstest.MapFile{Data: []byte(`{"products":[
			{"code":"Ecs","plugin_default_version":"2014-05-26","versions":["2014-05-26"],
			 "regional_endpoints":{"cn-hangzhou":"ecs.cn-hangzhou.aliyuncs.com"}}
		]}`)},
		"canonical/ecs/2014-05-26/version.json":           &fstest.MapFile{Data: []byte(versionJSON("DescribeInstances", "describe-instances", "2014-05-26"))},
		"canonical/ecs/2014-05-26/DescribeInstances.json": &fstest.MapFile{Data: []byte(apiJSON("DescribeInstances", "describe-instances", "2014-05-26"))},
	}
	src := NewBaselineSource(fsys, "test")
	api, err := src.LoadAPI("ecs", "2014-05-26", "DescribeInstances")
	if err != nil {
		t.Fatalf("LoadAPI: %v", err)
	}
	if got := api.Endpoints.Public["cn-hangzhou"]; got != "ecs.cn-hangzhou.aliyuncs.com" {
		t.Errorf("endpoint not injected from catalog, got %q", got)
	}
}
