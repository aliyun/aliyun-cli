package source

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-openapi-runtime/source/storage"
)

type baselineOpenFailure struct {
	storage.Storage
	err error
}

func (s baselineOpenFailure) Open(string) (storage.Volume, error) { return nil, s.err }

func TestBaselineCatalogFailuresAreStable(t *testing.T) {
	for _, tc := range []struct {
		name     string
		files    fstest.MapFS
		notFound bool
	}{
		{name: "missing catalog", files: fstest.MapFS{}, notFound: true},
		{name: "unreadable catalog", files: fstest.MapFS{"metadatas/products.json/child": {Data: []byte("directory")}}},
		{name: "invalid catalog", files: fstest.MapFS{"metadatas/products.json": {Data: []byte("{")}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewBaselineSource(tc.files, "test")
			if s.Kind() != KindBaseline {
				t.Fatal("incorrect source kind")
			}
			_, _, first := s.LoadProduct("ecs")
			_, _, again := s.LoadProduct("ecs")
			if first == nil || again == nil {
				t.Fatal("catalog failure was accepted")
			}
			if tc.notFound {
				if !errors.Is(first, ErrNotFound) {
					t.Fatalf("missing catalog error=%v", first)
				}
			} else if first != again {
				t.Fatalf("catalog failure was not cached: %v / %v", first, again)
			}
		})
	}
	s := NewBaselineSource(fstest.MapFS{"metadatas/products.json": {Data: []byte(`{"products":[{"code":""},{"code":"Ecs","versions":["v1"],"plugin_default_version":"v1"}]}`)}}, "test")
	product, _, err := s.LoadProduct("ECS")
	if err != nil || product.Code != "ecs" {
		t.Fatalf("catalog lookup=%#v,%v", product, err)
	}
}

func TestBaselineAPILookupNormalizesOnlyMissingEntries(t *testing.T) {
	for _, tc := range []struct {
		name, index, api       string
		absentProduct, missing bool
	}{
		{name: "absent product", absentProduct: true, missing: true},
		{name: "absent version", index: "", api: "", missing: true},
		{name: "malformed definitions", index: "{", api: "{"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := fstest.MapFS{}
			if !tc.absentProduct {
				files["canonical/ecs/other/file"] = &fstest.MapFile{Data: []byte("entry")}
			}
			if tc.index != "" {
				files["canonical/ecs/v1/version.json"] = &fstest.MapFile{Data: []byte(tc.index)}
			}
			if tc.api != "" {
				files["canonical/ecs/v1/Get.json"] = &fstest.MapFile{Data: []byte(tc.api)}
			}
			s := NewBaselineSource(files, "test")
			_, indexErr := s.LoadAPIIndex("ECS", "v1")
			_, apiErr := s.LoadAPI("ECS", "v1", "Get")
			for _, err := range []error{indexErr, apiErr} {
				if err == nil || errors.Is(err, ErrNotFound) != tc.missing {
					t.Fatalf("lookup error=%v, missing=%v", err, tc.missing)
				}
			}
		})
	}
	cause := errors.New("storage unavailable")
	s := NewBaselineSource(fstest.MapFS{}, "test")
	s.canonical = baselineOpenFailure{err: cause}
	if _, err := s.LoadAPIIndex("ecs", "v1"); !errors.Is(err, cause) {
		t.Fatalf("index lost storage error: %v", err)
	}
	if _, err := s.LoadAPI("ecs", "v1", "Get"); !errors.Is(err, cause) {
		t.Fatalf("API lost storage error: %v", err)
	}
}
