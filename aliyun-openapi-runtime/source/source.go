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

// Package source composes Storage + Format into a product-shaped
// view. A Source knows the on-disk layout convention and hides it
// behind a small API surface.
//
// Sources DO NOT cache anything. Every call performs the underlying
// IO + decode work. Product ownership across multiple Sources
// (override / user plugin / baseline) is decided by the Loader.
//
// Concrete Sources:
//
//	OverrideSource   ($ALIYUN_CLI_PLUGINS_DIR_OVERRIDE)
//	UserPluginSource (~/.aliyun/plugins/)
//	BaselineSource   (embed.FS)
package source

import (
	"errors"
	"time"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

// Kind categorises where a Source's data comes from. It is exposed
// on Provenance so downstream code (e.g. `aliyun plugin which`) can
// report which layer served a given command.
type Kind int

// Enumerated Kinds. Order matters for provenance reporting:
// overrides win over user plugins, which win over baseline.
const (
	KindBaseline Kind = iota
	KindUser
	KindOverride
)

// String returns a stable lowercase spelling of the Kind. Callers
// serialising Provenance should use this (never the raw int) to keep
// on-disk provenance.json readable and forward-compatible.
func (k Kind) String() string {
	switch k {
	case KindBaseline:
		return "baseline"
	case KindUser:
		return "user"
	case KindOverride:
		return "override"
	default:
		return "unknown"
	}
}

// Provenance records where a piece of metadata came from. It is
// attached to LoadProduct results by every Source implementation.
type Provenance struct {
	Kind        Kind
	Version     string    // product version, e.g. "1.5.0"
	BundledBy   string    // only meaningful for baseline: e.g. "aliyun-cli 3.0.234"
	InstalledAt time.Time // wall-clock install time for user/override
	Origin      string    // absolute path or embed root, human-readable
}

// Source is the read-only product/API lookup contract.
type Source interface {
	// Kind reports which layer this source represents.
	Kind() Kind

	// LoadProduct returns the product manifest along with a
	// Provenance record. Missing product -> ErrNotFound.
	LoadProduct(code string) (*meta.Product, *Provenance, error)

	// LoadIndex returns the per-(product, version) lightweight
	// index. Missing product/version -> ErrNotFound.
	LoadIndex(code, version string) (*meta.APIIndex, error)

	// LoadAPI returns the full per-API meta. Missing anything ->
	// ErrNotFound. Implementations SHOULD prefer returning the
	// sentinel over wrapping storage-level errors so the Loader
	// can distinguish "not here" from "hard failure".
	LoadAPI(code, version, name string) (*meta.API, error)
}

// ErrNotFound is the sentinel returned when a lookup misses. Callers
// use errors.Is to differentiate; the Loader depends on this to
// know when to try the next Source for product ownership.
var ErrNotFound = errors.New("source: not found")
