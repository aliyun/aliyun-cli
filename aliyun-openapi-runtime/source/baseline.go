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
	"io/fs"
	"strings"
	"sync"

	"github.com/aliyun/aliyun-openapi-runtime/format"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/storage"
)

// BaselineSource is the read-only source over the shared
// aliyun-openapi-meta "canonical" dataset embedded in the CLI binary.
//
// Layout (relative to the injected FS root):
//
//	canonical/<product>/<version>/<APIName>.json   per-API definition
//	canonical/<product>/<version>/version.json     per-version index
//	metadatas/products.json                        product catalog + endpoints
//
// Product identity, version list and endpoints come from the central
// products.json (loaded once); per-version indexes and per-API
// definitions come from the canonical/ tree.
type BaselineSource struct {
	fsys      fs.FS
	canonical storage.Storage // rooted at canonical/
	codec     format.Format
	bundledBy string
	origin    string

	loadOnce sync.Once
	loadErr  error
	products map[string]*productRec // key = lower-case product code
}

type productRec struct {
	product      *meta.Product
	distribution string
}

// NewBaselineSource wraps an fs.FS holding canonical/ and metadatas/.
func NewBaselineSource(embedFS fs.FS, bundledBy string) *BaselineSource {
	return &BaselineSource{
		fsys:      embedFS,
		canonical: storage.NewFSStorage(embedFS, schema.CanonicalRoot),
		codec:     format.NewJSONFormat(),
		bundledBy: bundledBy,
		origin:    "embed:aliyun-openapi-meta",
	}
}

// Kind reports the baseline layer.
func (s *BaselineSource) Kind() Kind { return KindBaseline }

// loadCatalog reads metadatas/products.json once and builds the
// product records (with endpoints). Missing file -> empty catalog.
func (s *BaselineSource) loadCatalog() (map[string]*productRec, error) {
	s.loadOnce.Do(func() {
		s.products = map[string]*productRec{}
		data, err := fs.ReadFile(s.fsys, schema.ProductsFile)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return // empty catalog
			}
			s.loadErr = err
			return
		}
		var idx schema.ProductsIndex
		if err := format.DecodeProductsJSON(data, &idx); err != nil {
			s.loadErr = err
			return
		}
		for i := range idx.Products {
			e := &idx.Products[i]
			code := strings.ToLower(e.Code)
			if code == "" {
				continue
			}
			s.products[code] = &productRec{
				product:      format.ProductEntryToProduct(e, code),
				distribution: e.Distribution,
			}
		}
	})
	return s.products, s.loadErr
}

// excluded reports whether the engine must abstain from a product
// because it ships as a Go plugin (distribution == "go").
func excluded(rec *productRec) bool {
	return rec != nil && rec.distribution == schema.DistributionGo
}

func (s *BaselineSource) LoadProduct(code string) (*meta.Product, *Provenance, error) {
	catalog, err := s.loadCatalog()
	if err != nil {
		return nil, nil, err
	}
	rec := catalog[strings.ToLower(code)]
	if rec == nil || excluded(rec) {
		return nil, nil, ErrNotFound
	}
	return rec.product, s.provenance(rec.product.DefaultVersion), nil
}

func (s *BaselineSource) LoadIndex(code, version string) (*meta.APIIndex, error) {
	vol, err := s.canonical.Open(strings.ToLower(code))
	if err != nil {
		if errors.Is(err, storage.ErrVolumeNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer vol.Close()

	idx, err := s.codec.DecodeIndex(vol, version)
	if err != nil {
		if storage.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if idx.ProductCode == "" {
		idx.ProductCode = strings.ToLower(code)
	}
	return idx, nil
}

func (s *BaselineSource) LoadAPI(code, version, name string) (*meta.API, error) {
	vol, err := s.canonical.Open(strings.ToLower(code))
	if err != nil {
		if errors.Is(err, storage.ErrVolumeNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer vol.Close()

	api, err := s.codec.DecodeAPI(vol, format.APIKey{
		Product: strings.ToLower(code), Version: version, Name: name,
	})
	if err != nil {
		if storage.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if api.ProductCode == "" {
		api.ProductCode = strings.ToLower(code)
	}
	// Inject product-level endpoints (canonical layout keeps them on
	// the product, not the API).
	if catalog, cerr := s.loadCatalog(); cerr == nil {
		if rec := catalog[strings.ToLower(code)]; rec != nil {
			api.Endpoints = rec.product.Endpoints
		}
	}
	return api, nil
}

func (s *BaselineSource) provenance(version string) *Provenance {
	return &Provenance{
		Kind:      KindBaseline,
		Version:   version,
		BundledBy: s.bundledBy,
		Origin:    s.origin,
	}
}
