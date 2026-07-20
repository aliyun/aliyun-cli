package loader

import (
	"context"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

type targetedSource struct {
	loadProductCalls []string
}

func (s *targetedSource) Kind() source.Kind { return source.KindUser }

func (s *targetedSource) LoadProduct(code string) (*meta.Product, *source.Provenance, error) {
	s.loadProductCalls = append(s.loadProductCalls, code)
	if code != "ecs" {
		return nil, nil, source.ErrNotFound
	}
	return &meta.Product{
		Code: "ecs", Versions: []string{"2014-05-26"}, DefaultVersion: "2014-05-26",
	}, &source.Provenance{Kind: source.KindUser}, nil
}

func (s *targetedSource) LoadIndex(string, string) (*meta.APIIndex, error) {
	return nil, source.ErrNotFound
}

func (s *targetedSource) LoadAPI(string, string, string) (*meta.API, error) {
	return nil, source.ErrNotFound
}

func TestEnsureProductLoadsOnlyRequestedProduct(t *testing.T) {
	src := &targetedSource{}
	ldr := New(src)

	if err := ldr.EnsureProduct(context.Background(), "ECS"); err != nil {
		t.Fatal(err)
	}
	if len(src.loadProductCalls) != 1 || src.loadProductCalls[0] != "ecs" {
		t.Fatalf("LoadProduct calls = %v, want [ecs]", src.loadProductCalls)
	}

	// Ownership is cached for the rest of the process.
	if err := ldr.EnsureProduct(context.Background(), "ecs"); err != nil {
		t.Fatal(err)
	}
	if len(src.loadProductCalls) != 1 {
		t.Fatalf("cached EnsureProduct made extra calls: %v", src.loadProductCalls)
	}
}
