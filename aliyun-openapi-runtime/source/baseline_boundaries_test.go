package source

import (
	"encoding/json"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

type baselineReadErrorFS struct {
	fs.FS
	path  string
	err   error
	reads int
}

func (f *baselineReadErrorFS) Open(name string) (fs.File, error) {
	if name == f.path {
		f.reads++
		return nil, &fs.PathError{Op: "open", Path: name, Err: f.err}
	}
	return f.FS.Open(name)
}

func TestBaselineCatalogSkipsEmptyCodesAndPreservesProvenance(t *testing.T) {
	source := NewBaselineSource(fstest.MapFS{
		"metadatas/products.json": {Data: []byte(`{"products":[{"code":""},{"code":"Ecs","plugin_default_version":"v1","versions":["v1"]}]}`)},
	}, "bundle-test")
	if source.Kind() != KindBaseline {
		t.Fatalf("Kind() = %v", source.Kind())
	}
	product, provenance, err := source.LoadProduct("ECS")
	if err != nil || product == nil || provenance == nil {
		t.Fatalf("LoadProduct() = %#v, %#v, %v", product, provenance, err)
	}
	if product.Code != "ecs" || product.DefaultVersion != "v1" {
		t.Fatalf("product = %#v", product)
	}
	if provenance.Kind != KindBaseline || provenance.BundledBy != "bundle-test" ||
		provenance.APIVersion != "v1" || provenance.Origin != "embed:aliyun-openapi-meta" {
		t.Fatalf("provenance = %#v", provenance)
	}
	for _, code := range []string{"", "missing"} {
		product, provenance, err = source.LoadProduct(code)
		if !errors.Is(err, ErrNotFound) || product != nil || provenance != nil {
			t.Fatalf("LoadProduct(%q) = %#v, %#v, %v", code, product, provenance, err)
		}
	}
}

func TestBaselineCatalogReadFailuresAreCached(t *testing.T) {
	for _, cause := range []error{fs.ErrNotExist, fs.ErrPermission} {
		t.Run(cause.Error(), func(t *testing.T) {
			fsys := &baselineReadErrorFS{FS: fstest.MapFS{}, path: "metadatas/products.json", err: cause}
			source := NewBaselineSource(fsys, "test")
			for attempt := 0; attempt < 2; attempt++ {
				product, provenance, err := source.LoadProduct("ecs")
				want := cause
				if errors.Is(cause, fs.ErrNotExist) {
					want = ErrNotFound
				}
				if !errors.Is(err, want) || product != nil || provenance != nil {
					t.Fatalf("LoadProduct() = %#v, %#v, %v; want %v", product, provenance, err, want)
				}
			}
			if fsys.reads != 1 {
				t.Fatalf("catalog reads = %d, want 1", fsys.reads)
			}
		})
	}
}

func TestBaselineMalformedCatalogDoesNotBlockIndependentAPIMetadata(t *testing.T) {
	source := NewBaselineSource(fstest.MapFS{
		"metadatas/products.json":       {Data: []byte(`{"products":5}`)},
		"canonical/ecs/v1/Run.json":     {Data: []byte(apiJSON("Run", "run", "v1"))},
		"canonical/ecs/v1/version.json": {Data: []byte(versionJSON("Run", "run", "v1"))},
	}, "test")
	product, provenance, err := source.LoadProduct("ecs")
	var shapeErr *json.UnmarshalTypeError
	if !errors.As(err, &shapeErr) || product != nil || provenance != nil {
		t.Fatalf("LoadProduct() = %#v, %#v, %v; want catalog shape error", product, provenance, err)
	}
	api, err := source.LoadAPI("ECS", "v1", "Run")
	if err != nil || api == nil || api.ProductCode != "ecs" || api.Name != "Run" || len(api.Endpoints.Public) != 0 {
		t.Fatalf("LoadAPI() = %#v, %v", api, err)
	}
	index, err := source.LoadAPIIndex("ECS", "v1")
	if err != nil || index == nil || index.ProductCode != "ecs" || index.ResolveCmd("run") != "Run" {
		t.Fatalf("LoadAPIIndex() = %#v, %v", index, err)
	}
}

func TestBaselineAPILookupsDistinguishMissingUnreadableAndMalformedEntries(t *testing.T) {
	for _, lookup := range []string{"index", "api"} {
		for _, failure := range []string{"missing volume", "missing entry", "unreadable volume", "unreadable entry", "malformed JSON"} {
			t.Run(lookup+"/"+failure, func(t *testing.T) {
				entry := "canonical/ecs/v1/Run.json"
				if lookup == "index" {
					entry = "canonical/ecs/v1/version.json"
				}
				files := fstest.MapFS{
					"canonical/ecs/v1/Run.json":     {Data: []byte(apiJSON("Run", "run", "v1"))},
					"canonical/ecs/v1/version.json": {Data: []byte(versionJSON("Run", "run", "v1"))},
				}
				var fsys fs.FS = files
				switch failure {
				case "missing volume":
					fsys = fstest.MapFS{}
				case "missing entry":
					delete(files, entry)
				case "unreadable volume":
					fsys = &baselineReadErrorFS{FS: files, path: "canonical/ecs", err: fs.ErrPermission}
				case "unreadable entry":
					fsys = &baselineReadErrorFS{FS: files, path: entry, err: fs.ErrPermission}
				case "malformed JSON":
					files[entry].Data = []byte(`{"invalid":]}`)
				}
				source := NewBaselineSource(fsys, "test")
				var err error
				if lookup == "index" {
					index, lookupErr := source.LoadAPIIndex("ECS", "v1")
					err = lookupErr
					if index != nil {
						t.Fatalf("failed lookup returned index %#v", index)
					}
				} else {
					api, lookupErr := source.LoadAPI("ECS", "v1", "Run")
					err = lookupErr
					if api != nil {
						t.Fatalf("failed lookup returned API %#v", api)
					}
				}
				switch failure {
				case "missing volume", "missing entry":
					if !errors.Is(err, ErrNotFound) {
						t.Fatalf("lookup error = %v, want ErrNotFound", err)
					}
				case "unreadable volume", "unreadable entry":
					if !errors.Is(err, fs.ErrPermission) {
						t.Fatalf("lookup error = %v, want permission error", err)
					}
				case "malformed JSON":
					var syntaxErr *json.SyntaxError
					if !errors.As(err, &syntaxErr) {
						t.Fatalf("lookup error = %v, want JSON syntax error", err)
					}
				}
			})
		}
	}
}
