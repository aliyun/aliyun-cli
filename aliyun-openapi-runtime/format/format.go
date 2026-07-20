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

// Package format decodes on-disk metadata bytes into the in-memory
// Model defined in ../meta.
//
// Contract: every Format implementation MUST produce the same Model
// types (meta.Product / meta.API / meta.APIIndex / meta.Parameter).
// Upper layers depend on the Model, never on wire-specific
// intermediates. This is the seam that lets us swap the disk format
// (JSON <-> Protobuf <-> binary cache) without touching Loader,
// Runtime, or CLI code.
//
// Precision contract (JSON implementations): all number-bearing
// decoders MUST honour encoding/json's UseNumber() so that int64
// resource IDs and other large values are never routed through
// float64.
package format

import (
	"errors"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/storage"
)

type Format interface {
	// DecodeIndex loads the lightweight per-(product, version) index
	// used for command-stub rendering. version is the API version string (e.g. "2014-05-26").
	DecodeIndex(vol storage.Volume, version string) (*meta.APIIndex, error)

	// DecodeAPI loads the full per-API meta identified by key.
	DecodeAPI(vol storage.Volume, key APIKey) (*meta.API, error)

	// DecodeProduct loads the top-level product manifest (versions,
	// default version, description). Implementations that lack an
	// explicit product manifest MAY synthesize one from directory
	// scanning; see JSONFormat for the reference behaviour.
	DecodeProduct(vol storage.Volume, code string) (*meta.Product, error)
}

// APIKey uniquely identifies one API within a Storage.
type APIKey struct {
	Product string // short product code, matches Volume name upstream
	Version string
	Name    string // PascalCase API name (matches file basename)
}

// ErrNotSupported is returned by decoders that do not understand the
// on-disk shape (e.g. legacy files without a required field). Callers
// SHOULD treat ErrNotSupported as a soft failure and either fall
// through to another Format or degrade gracefully.
var ErrNotSupported = errors.New("format: unsupported encoding")
