package engine

import (
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

type importanceVersionSource struct {
	multiVersionSource
	base float64
}

func (s importanceVersionSource) LoadAPIIndex(code, version string) (*meta.APIIndex, error) {
	index, err := s.multiVersionSource.LoadAPIIndex(code, version)
	if err != nil {
		return nil, err
	}
	score := s.base
	if version == "v1" {
		score = 0
	}
	entry := index.Entries["RunThing"]
	entry.ImportanceScore = &score
	index.Entries["RunThing"] = entry
	return index, nil
}

func TestProductCommandImportanceKeepsSelectedOwnerAndVersion(t *testing.T) {
	engine := NewEngine(func() (loader.Loader, error) {
		return loader.New(importanceVersionSource{base: .3}, importanceVersionSource{base: .9}), nil
	}, nil)
	if got := engine.ProductCommandImportance("demo", nil); !reflect.DeepEqual(got, map[string]float64{"run-thing": .3}) {
		t.Fatal(got)
	}
	if got := engine.ProductCommandImportance("demo", []string{"--api-version", "v1"}); !reflect.DeepEqual(got, map[string]float64{"run-thing": 0}) {
		t.Fatal(got)
	}
	if got := engine.ProductCommandImportance("demo", []string{"--api-version=v3"}); got != nil {
		t.Fatalf("invalid version borrowed default: %v", got)
	}
	if got := engine.ProductCommandImportance("missing", nil); got != nil {
		t.Fatal(got)
	}
}
