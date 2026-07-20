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

// Package storage abstracts the "give me bytes for a named entry"
// concern. It is the lowest layer of aliyun-openapi-runtime and has no
// notion of products, API versions, or meta shapes; those concepts
// live in ../format and ../source respectively.
//
// Two implementations ship out of the box:
//
//   - EmbedStorage — a read-only Storage over a Go fs.FS (typically
//     the //go:embed FS from the baseline metadata package).
//   - DirStorage  — a read/write Storage over a local directory,
//     used by the user plugin overlay (~/.aliyun/plugins/) and the
//     $ALIYUN_CLI_PLUGINS_DIR_OVERRIDE knob.
//
// Adding a new storage flavour (tar.zst, sqlite, HTTP CDN, ...)
// means implementing the two interfaces below; every layer above
// keeps working unchanged.
package storage

import (
	"errors"
	"io/fs"
	"time"
)

// Storage is a namespace of Volumes. A Volume typically maps to one
// product; the namespace shape (e.g. one product per top-level dir)
// is imposed by the caller, not by Storage itself.
type Storage interface {
	// Open returns the Volume for the given logical name. If no
	// volume exists Open MUST return ErrVolumeNotFound.
	Open(name string) (Volume, error)

	// List enumerates every top-level Volume name in an unspecified
	// order. Callers that need determinism should sort the result.
	List() ([]string, error)

	// ReadRoot reads a file that lives at the storage root, outside
	// any volume (e.g. a global index like meta_index/product_index.json).
	// Implementations that don't expose root-level files MAY return
	// ErrEntryNotFound unconditionally. entry uses slash separators.
	ReadRoot(entry string) ([]byte, error)
}

// Volume is a flat namespace of entries (files) with random-access
// reads. Volumes are cheap to reopen; implementations may treat
// Close as a no-op.
type Volume interface {
	// ReadAll returns the entire content of entry.
	ReadAll(entry string) ([]byte, error)

	// ReadAt returns n bytes of entry starting at offset off. It is
	// intended for range access into large multi-record files
	// (e.g. JSONL + offset index). Implementations that can't
	// support random access MAY simulate it via ReadAll + slice.
	ReadAt(entry string, off, n int64) ([]byte, error)

	// Stat returns metadata for entry.
	Stat(entry string) (Stat, error)

	// List enumerates every entry with the given prefix in an
	// unspecified order. Pass "" for all entries.
	List(prefix string) ([]string, error)

	// Close releases any resources held by the volume. Callers MUST
	// call Close, but multiple calls MUST be safe.
	Close() error
}

// Stat is the minimal file metadata Volume exposes.
type Stat struct {
	Size    int64
	ModTime time.Time
	IsDir   bool
}

// ErrVolumeNotFound is returned by Storage.Open when the requested
// volume does not exist.
var ErrVolumeNotFound = errors.New("storage: volume not found")

// ErrEntryNotFound is returned by Volume operations when the entry
// does not exist. Implementations should return the sentinel, or wrap
// it, so callers can check with errors.Is.
var ErrEntryNotFound = errors.New("storage: entry not found")

// IsNotExist reports whether err (from Storage / Volume) means "the
// named thing does not exist" — covering both sentinels above and
// fs.ErrNotExist raised by embedded implementations.
func IsNotExist(err error) bool {
	return errors.Is(err, ErrVolumeNotFound) ||
		errors.Is(err, ErrEntryNotFound) ||
		errors.Is(err, fs.ErrNotExist)
}
