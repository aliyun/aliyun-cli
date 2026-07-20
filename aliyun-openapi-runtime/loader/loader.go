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

// Package loader is the runtime-facing entry point of aliyun-openapi-runtime.
//
// It layers two concerns on top of one or more source.Source backends:
//
//  1. Product ownership — EnsureProduct picks, for one requested product,
//     the highest-priority Source that knows it (override > user plugin >
//     baseline). That product is then served wholly from that Source;
//     there is no cross-source merge of indexes or APIs.
//  2. Version routing — ResolveVersion() implements the "user
//     omitted --version, pick the default" policy in one place.
//
// Index/API decoding is on demand via GetIndex()/GetAPI() with no
// memoisation (a CLI invocation resolves each item ~once).
package loader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

// Loader is the interface consumed by upper layers. defaultLoader is
// the only implementation shipped; tests may swap in mocks.
type Loader interface {
	// EnsureProduct resolves and caches ownership for exactly one product.
	// Normal CLI dispatch uses this path so it never enumerates unrelated
	// products or plugin directories.
	EnsureProduct(ctx context.Context, code string) error

	// LookupProduct returns the cached Product manifest, or nil if
	// unknown.
	LookupProduct(code string) *meta.Product

	// ResolveVersion resolves an optionally-requested version to
	// the concrete version string used for LoadIndex / GetAPI. Pass
	// "" for the product's default.
	ResolveVersion(product, requested string) (string, error)

	// GetIndex returns the (product, version) index, decoded fresh from
	// the product's owning Source on each call.
	GetIndex(product, version string) (*meta.APIIndex, error)

	// GetAPI returns the fully-decoded API metadata, decoded fresh from
	// the product's owning Source on each call.
	GetAPI(product, version, name string) (*meta.API, error)

	// ResolveCommand maps the user-facing "<product> <cmd>" pair back
	// to the canonical APIRef against the product's DEFAULT version.
	// Unknown commands return ErrCommandNotFound so callers can
	// short-circuit into a helpful "did you mean ..." message.
	ResolveCommand(product, cmdName string) (meta.APIRef, error)

	// ResolveCommandVersion is the version-aware counterpart of
	// ResolveCommand. It resolves the target version (empty -> product
	// default; otherwise validated against the product's `versions`
	// list) and then resolves the command uniformly from that version's
	// index via its cmd-name -> APIName reverse map (built at decode
	// time, O(1)). Default and non-default versions take the exact same
	// path, so (product, cmd, version) always maps to one API through a
	// single, consistent lookup.
	ResolveCommandVersion(product, cmdName, version string) (meta.APIRef, error)

	// CommandExists reports whether the product exposes the command in
	// ANY of its versions. It is the existence check the host router
	// uses to decide whether to route to the engine; unlike a
	// default-version-only table it does not miss commands that live
	// solely in a non-default version.
	CommandExists(product, cmdName string) bool

	// Provenance returns the Provenance record last observed for a product.
	// Returns nil before that product has been ensured.
	Provenance(product string) *source.Provenance
}

// ErrCommandNotFound is returned by ResolveCommand when the requested
// "<product> <cmd>" pair is not in the route table.
var ErrCommandNotFound = errors.New("loader: command not found")

// New returns a fresh Loader backed by the given Sources in priority
// order (highest first). Nil entries are skipped. A product is owned
// entirely by the first Source that LoadProduct-claims it.
func New(layers ...source.Source) Loader {
	filtered := make([]source.Source, 0, len(layers))
	for _, s := range layers {
		if s != nil {
			filtered = append(filtered, s)
		}
	}
	return &defaultLoader{
		layers:   filtered,
		products: map[string]*productEntry{},
	}
}

// ============================================================================
// defaultLoader
// ============================================================================

type productEntry struct {
	product *meta.Product
	prov    *source.Provenance
	src     source.Source // owning layer for this product
}

type defaultLoader struct {
	layers []source.Source // highest priority first

	// mu guards products, which is populated lazily by EnsureProduct.
	mu       sync.RWMutex
	products map[string]*productEntry
}

func (l *defaultLoader) EnsureProduct(ctx context.Context, code string) error {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return fmt.Errorf("product code is empty")
	}

	l.mu.RLock()
	_, ok := l.products[code]
	l.mu.RUnlock()
	if ok {
		return nil
	}

	for _, s := range l.layers {
		if err := ctx.Err(); err != nil {
			return err
		}
		p, prov, err := s.LoadProduct(code)
		if err != nil {
			if errors.Is(err, source.ErrNotFound) {
				continue
			}
			return fmt.Errorf("load product %q from %s source: %w", code, s.Kind(), err)
		}
		l.mu.Lock()
		if _, exists := l.products[code]; !exists {
			l.products[code] = &productEntry{product: p, prov: prov, src: s}
		}
		l.mu.Unlock()
		return nil
	}
	return fmt.Errorf("unknown product %q", code)
}

func (l *defaultLoader) LookupProduct(code string) *meta.Product {
	l.mu.RLock()
	defer l.mu.RUnlock()
	e := l.products[code]
	if e == nil {
		return nil
	}
	return e.product
}

func (l *defaultLoader) Provenance(product string) *source.Provenance {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if e := l.products[product]; e != nil {
		return e.prov
	}
	return nil
}

func (l *defaultLoader) owner(product string) (source.Source, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	e := l.products[product]
	if e == nil || e.src == nil {
		return nil, fmt.Errorf("unknown product %q", product)
	}
	return e.src, nil
}

func (l *defaultLoader) ResolveVersion(product, requested string) (string, error) {
	p := l.LookupProduct(product)
	if p == nil {
		return "", fmt.Errorf("unknown product %q", product)
	}
	if requested == "" {
		if p.DefaultVersion == "" {
			return "", fmt.Errorf("product %q has no default version", product)
		}
		if v := os.Getenv(defaultVersionEnvVar(product)); v != "" && p.HasVersion(v) {
			return v, nil
		}
		return p.DefaultVersion, nil
	}
	if !p.HasVersion(requested) {
		return "", fmt.Errorf("product %q does not expose version %q", product, requested)
	}
	return requested, nil
}

func defaultVersionEnvVar(product string) string {
	prefix := strings.ToUpper(strings.ReplaceAll(product, "-", "_"))
	return "ALIBABA_CLOUD_" + prefix + "_API_VERSION"
}

func (l *defaultLoader) GetIndex(product, version string) (*meta.APIIndex, error) {
	src, err := l.owner(product)
	if err != nil {
		return nil, err
	}
	return src.LoadIndex(product, version)
}

func (l *defaultLoader) ResolveCommand(product, cmdName string) (meta.APIRef, error) {
	return l.ResolveCommandVersion(product, cmdName, "")
}

// CommandExists reports whether the product exposes the command in ANY
// of its registered versions (the default version is always one of
// them), so a command that lives only in a non-default version is still
// recognised — e.g. by the host router when the user will pass
// --api-version. Unknown products short-circuit to false without any
// index I/O.
func (l *defaultLoader) CommandExists(product, cmdName string) bool {
	if product == "" || cmdName == "" {
		return false
	}
	p := l.LookupProduct(product)
	if p == nil {
		return false
	}
	for _, v := range p.Versions {
		if v == "" {
			continue
		}
		if idx, err := l.GetIndex(product, v); err == nil && idx.ResolveCmd(cmdName) != "" {
			return true
		}
	}
	return false
}

func (l *defaultLoader) ResolveCommandVersion(product, cmdName, version string) (meta.APIRef, error) {
	if product == "" || cmdName == "" {
		return meta.APIRef{}, ErrCommandNotFound
	}

	resolved, err := l.ResolveVersion(product, version)
	if err != nil {
		return meta.APIRef{}, err
	}
	idx, err := l.GetIndex(product, resolved)
	if err != nil {
		return meta.APIRef{}, err
	}
	if apiName := idx.ResolveCmd(cmdName); apiName != "" {
		return meta.APIRef{Product: product, Version: resolved, Name: apiName}, nil
	}
	return meta.APIRef{}, fmt.Errorf("%s %s@%s: %w", product, cmdName, resolved, ErrCommandNotFound)
}

func (l *defaultLoader) GetAPI(product, version, name string) (*meta.API, error) {
	src, err := l.owner(product)
	if err != nil {
		return nil, err
	}
	api, err := src.LoadAPI(product, version, name)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
			return nil, fmt.Errorf("api %s/%s/%s: %w", product, version, name, err)
		}
		return nil, err
	}
	return api, nil
}
