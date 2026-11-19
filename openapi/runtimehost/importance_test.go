package runtimehost

import (
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

func TestProductImportanceWrapperReadsSelectedVersion(t *testing.T) {
	previous := Engine()
	t.Cleanup(func() { engineInst = previous })
	fsys := fstest.MapFS{
		"metadatas/products.json":        {Data: []byte(`{"products":[{"code":"demo","version":"v1","plugin_default_version":"v2","versions":["v1","v2"]}]}`)},
		"canonical/demo/v1/version.json": {Data: []byte(`{"version":"v1","apis":{"RunThing":{"cmd_name":"run-thing"}}}`)},
		"canonical/demo/v2/version.json": {Data: []byte(`{"version":"v2","apis":{"RunThing":{"cmd_name":"run-thing","importance_score":0.8}}}`)},
	}
	engineInst = engine.NewEngine(func() (loader.Loader, error) {
		return loader.New(source.NewBaselineSource(fsys, "test")), nil
	}, nil)
	if got := ProductCommandImportance("demo", nil); !reflect.DeepEqual(got, map[string]float64{"run-thing": .8}) {
		t.Fatalf("default version scores: %v", got)
	}
	if got := ProductCommandImportance("demo", []string{"--api-version", "v1"}); len(got) != 0 {
		t.Fatalf("unscored version borrowed another version: %v", got)
	}
	if ProductCommandImportance("", nil) != nil || ProductCommandImportance("missing", nil) != nil {
		t.Fatal("missing product must have no scores")
	}
}
