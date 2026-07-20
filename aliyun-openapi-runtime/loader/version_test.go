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

package loader

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-openapi-runtime/source"
)

// twoVersionFS lays out one product ("demo") that exposes the same
// command ("do-thing") in two API versions, mapped to two different
// PascalCase API names. Default version is the newer one.
func twoVersionFS() fstest.MapFS {
	api := func(name, version string) string {
		return `{"name":"` + name + `","cmd_name":"do-thing",` +
			`"operation":{"method":"POST","api_version":"` + version + `","action":"` + name + `","api_style":"RPC"}}`
	}
	ver := func(name, version string) string {
		return `{"version":"` + version + `","style":"RPC","apis":{"` + name + `":{"cmd_name":"do-thing"}}}`
	}
	return fstest.MapFS{
		"metadatas/products.json": &fstest.MapFile{Data: []byte(`{"products":[
			{"code":"Demo","plugin_default_version":"2020-01-01","versions":["2018-01-01","2020-01-01"]}
		]}`)},
		"canonical/demo/2018-01-01/version.json":   &fstest.MapFile{Data: []byte(ver("DoThingV1", "2018-01-01"))},
		"canonical/demo/2018-01-01/DoThingV1.json": &fstest.MapFile{Data: []byte(api("DoThingV1", "2018-01-01"))},
		"canonical/demo/2020-01-01/version.json":   &fstest.MapFile{Data: []byte(ver("DoThingV2", "2020-01-01"))},
		"canonical/demo/2020-01-01/DoThingV2.json": &fstest.MapFile{Data: []byte(api("DoThingV2", "2020-01-01"))},
	}
}

func newTwoVersionLoader(t *testing.T) Loader {
	t.Helper()
	src := source.NewBaselineSource(twoVersionFS(), "test")
	l := New(src)
	if err := l.EnsureProduct(context.Background(), "demo"); err != nil {
		t.Fatalf("ensure product: %v", err)
	}
	return l
}

// TestResolveCommandVersionSelectsRequested is the regression guard for
// the --api-version bug: resolving a command with an explicit older
// version must select that version's API, not the default.
func TestResolveCommandVersionSelectsRequested(t *testing.T) {
	l := newTwoVersionLoader(t)

	// Default (empty version) -> newest.
	def, err := l.ResolveCommandVersion("demo", "do-thing", "")
	if err != nil {
		t.Fatalf("default resolve: %v", err)
	}
	if def.Version != "2020-01-01" || def.Name != "DoThingV2" {
		t.Fatalf("default should be V2, got %+v", def)
	}

	// Explicit default version -> same result as empty (fast path).
	expDef, err := l.ResolveCommandVersion("demo", "do-thing", "2020-01-01")
	if err != nil {
		t.Fatalf("explicit default resolve: %v", err)
	}
	if expDef != def {
		t.Fatalf("explicit default should match empty, got %+v vs %+v", expDef, def)
	}

	// Explicit older version -> that version's API.
	old, err := l.ResolveCommandVersion("demo", "do-thing", "2018-01-01")
	if err != nil {
		t.Fatalf("old resolve: %v", err)
	}
	if old.Version != "2018-01-01" || old.Name != "DoThingV1" {
		t.Fatalf("explicit --api-version ignored, got %+v", old)
	}
}

// TestResolveCommandVersionUnknownVersion rejects versions the product
// does not expose.
func TestResolveCommandVersionUnknownVersion(t *testing.T) {
	l := newTwoVersionLoader(t)
	if _, err := l.ResolveCommandVersion("demo", "do-thing", "1999-01-01"); err == nil {
		t.Fatal("expected error for unknown version")
	}
}

// TestDefaultVersionEnvOverride mirrors the Go plugin runtime: when no
// --api-version is given, ALIBABA_CLOUD_<PRODUCT>_API_VERSION pins the
// default (only when it names an exposed version); an invalid value is
// ignored and falls back to plugin_default_version.
func TestDefaultVersionEnvOverride(t *testing.T) {
	l := newTwoVersionLoader(t)

	// Valid override -> the older version becomes the implicit default.
	t.Setenv("ALIBABA_CLOUD_DEMO_API_VERSION", "2018-01-01")
	ref, err := l.ResolveCommandVersion("demo", "do-thing", "")
	if err != nil {
		t.Fatalf("resolve with env override: %v", err)
	}
	if ref.Version != "2018-01-01" || ref.Name != "DoThingV1" {
		t.Fatalf("env override ignored, got %+v", ref)
	}

	// An explicit --api-version still wins over the env override.
	if got, err := l.ResolveCommandVersion("demo", "do-thing", "2020-01-01"); err != nil || got.Version != "2020-01-01" {
		t.Fatalf("explicit flag should override env, got %+v err=%v", got, err)
	}

	// Invalid override -> ignored, falls back to plugin_default_version.
	t.Setenv("ALIBABA_CLOUD_DEMO_API_VERSION", "1999-09-09")
	back, err := l.ResolveCommandVersion("demo", "do-thing", "")
	if err != nil {
		t.Fatalf("resolve with invalid env: %v", err)
	}
	if back.Version != "2020-01-01" {
		t.Fatalf("invalid env should fall back to default, got %+v", back)
	}
}
