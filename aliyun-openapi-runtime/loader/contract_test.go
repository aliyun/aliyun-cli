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
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

func TestLoaderCommandDiscoveryAndAccessors(t *testing.T) {
	loader := newTwoVersionLoader(t)
	product := loader.LookupProduct("demo")
	if product == nil || product.DefaultVersion != "2020-01-01" || loader.LookupProduct("missing") != nil {
		t.Fatalf("LookupProduct() = %#v", product)
	}
	if provenance := loader.Provenance("demo"); provenance == nil || provenance.Kind != source.KindBaseline {
		t.Fatalf("Provenance(demo) = %#v", provenance)
	}
	if loader.Provenance("missing") != nil {
		t.Fatal("Provenance(missing) should be nil")
	}

	index, err := loader.GetAPIIndex("demo", "2020-01-01")
	if err != nil || index.ResolveCmd("do-thing") != "DoThingV2" {
		t.Fatalf("GetAPIIndex() = %#v, %v", index, err)
	}
	ref, err := loader.ResolveCommand("demo", "do-thing")
	if err != nil || ref.Name != "DoThingV2" {
		t.Fatalf("ResolveCommand() = %#v, %v", ref, err)
	}
	if !loader.CommandExists("demo", "do-thing") || loader.CommandExists("demo", "missing") ||
		loader.CommandExists("", "do-thing") || loader.CommandExists("demo", "") || loader.CommandExists("missing", "do-thing") {
		t.Fatal("CommandExists returned unexpected result")
	}
	if got, want := loader.FindCommandVersions("demo", "do-thing"), []string{"2018-01-01", "2020-01-01"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("FindCommandVersions() = %v, want %v", got, want)
	}
	for _, pair := range [][2]string{{"", "do-thing"}, {"demo", ""}, {"missing", "do-thing"}} {
		if got := loader.FindCommandVersions(pair[0], pair[1]); got != nil {
			t.Fatalf("FindCommandVersions(%q, %q) = %v", pair[0], pair[1], got)
		}
	}
	if got := loader.FindCommandVersions("demo", "missing"); len(got) != 0 {
		t.Fatalf("FindCommandVersions(missing command) = %v", got)
	}

	api, err := loader.GetAPI("demo", "2020-01-01", "DoThingV2")
	if err != nil || api.Name != "DoThingV2" {
		t.Fatalf("GetAPI() = %#v, %v", api, err)
	}
	if _, err := loader.GetAPI("demo", "2020-01-01", "Missing"); !errors.Is(err, source.ErrNotFound) {
		t.Fatalf("GetAPI(missing) error = %v", err)
	}
	if _, err := loader.ResolveCommand("demo", "missing"); !errors.Is(err, ErrCommandNotFound) {
		t.Fatalf("ResolveCommand(missing) error = %v", err)
	}
	for _, pair := range [][2]string{{"", "do-thing"}, {"demo", ""}} {
		if _, err := loader.ResolveCommandVersion(pair[0], pair[1], ""); !errors.Is(err, ErrCommandNotFound) {
			t.Fatalf("ResolveCommandVersion(%q, %q) error = %v", pair[0], pair[1], err)
		}
	}
}

type loaderContractSource struct {
	product    *meta.Product
	productErr error
	index      *meta.APIIndex
	indexErr   error
	api        *meta.API
	apiErr     error
}

func (s loaderContractSource) Kind() source.Kind { return source.KindUser }

func (s loaderContractSource) LoadProduct(string) (*meta.Product, *source.Provenance, error) {
	return s.product, &source.Provenance{Kind: source.KindUser}, s.productErr
}

func (s loaderContractSource) LoadAPIIndex(string, string) (*meta.APIIndex, error) {
	return s.index, s.indexErr
}

func (s loaderContractSource) LoadAPI(string, string, string) (*meta.API, error) {
	return s.api, s.apiErr
}

func TestLoaderErrorBoundaries(t *testing.T) {
	empty := New(nil)
	if err := empty.EnsureProduct(" "); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("EnsureProduct(empty) error = %v", err)
	}
	if err := empty.EnsureProduct("missing"); err == nil || !strings.Contains(err.Error(), "unknown product") {
		t.Fatalf("EnsureProduct(missing) error = %v", err)
	}
	if _, err := empty.ResolveVersion("missing", ""); err == nil {
		t.Fatal("ResolveVersion(unknown) succeeded")
	}
	if _, err := empty.GetAPIIndex("missing", "v1"); err == nil {
		t.Fatal("GetAPIIndex(unknown) succeeded")
	}
	if _, err := empty.GetAPI("missing", "v1", "Run"); err == nil {
		t.Fatal("GetAPI(unknown) succeeded")
	}

	cause := errors.New("source failed")
	failing := New(loaderContractSource{productErr: cause})
	if err := failing.EnsureProduct("demo"); !errors.Is(err, cause) || !strings.Contains(err.Error(), "user source") {
		t.Fatalf("EnsureProduct(source error) = %v", err)
	}

	noDefault := New(loaderContractSource{product: &meta.Product{Code: "demo", Versions: []string{"v1"}}})
	if err := noDefault.EnsureProduct("demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := noDefault.ResolveVersion("demo", ""); err == nil || !strings.Contains(err.Error(), "no default version") {
		t.Fatalf("ResolveVersion(no default) error = %v", err)
	}

	indexCause := errors.New("index failed")
	indexFailure := New(loaderContractSource{
		product: &meta.Product{Code: "demo", Versions: []string{"v1"}, DefaultVersion: "v1"}, indexErr: indexCause,
	})
	if err := indexFailure.EnsureProduct("demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := indexFailure.ResolveCommandVersion("demo", "run", ""); !errors.Is(err, indexCause) {
		t.Fatalf("ResolveCommandVersion(index error) = %v", err)
	}

	apiCause := errors.New("api failed")
	apiFailure := New(loaderContractSource{
		product: &meta.Product{Code: "demo", Versions: []string{"v1"}, DefaultVersion: "v1"}, apiErr: apiCause,
	})
	if err := apiFailure.EnsureProduct("demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := apiFailure.GetAPI("demo", "v1", "Run"); !errors.Is(err, apiCause) {
		t.Fatalf("GetAPI(source error) = %v", err)
	}

	invalidAPI := New(loaderContractSource{
		product: &meta.Product{Code: "demo", Versions: []string{"v1"}, DefaultVersion: "v1"},
		api:     &meta.API{Parameters: []meta.Parameter{{Name: "missing-option"}}},
	})
	if err := invalidAPI.EnsureProduct("demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := invalidAPI.GetAPI("demo", "v1", "Run"); err == nil || !strings.Contains(err.Error(), "metadata") {
		t.Fatalf("GetAPI(invalid metadata) = %v", err)
	}
}
