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

package storage

import (
	"errors"
	"io/fs"
	"path"
	"strings"
)

// FSStorage is a read-only Storage backed by a Go fs.FS. It is used
// primarily for the baseline metadata bundled via //go:embed but
// works with any fs.FS (e.g. testing/fstest.MapFS in unit tests).
//
// The Storage assumes the underlying tree is laid out as
// <root>/<volume>/... where <root> is the directory passed to
// NewFSStorage and each <volume> is one product.
type FSStorage struct {
	fsys fs.FS
	root string // slash-separated, no trailing slash; "" for the FS root
}

// NewFSStorage wraps fsys at the given root directory. root MUST be
// slash-separated (per fs.FS convention) and MUST NOT begin or end
// with a slash; pass "" to use the FS root itself.
func NewFSStorage(fsys fs.FS, root string) *FSStorage {
	return &FSStorage{
		fsys: fsys,
		root: strings.Trim(root, "/"),
	}
}

// Open returns the Volume named name. Volume names correspond to
// direct children of the storage root.
func (s *FSStorage) Open(name string) (Volume, error) {
	dir := s.join(name)
	info, err := fs.Stat(s.fsys, dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrVolumeNotFound
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrVolumeNotFound
	}
	return &fsVolume{fsys: s.fsys, base: dir}, nil
}

// List returns every direct child directory of the storage root.
// Non-directory entries are skipped.
func (s *FSStorage) List() ([]string, error) {
	dir := s.root
	if dir == "" {
		dir = "."
	}
	entries, err := fs.ReadDir(s.fsys, dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// ReadRoot reads a file located at the storage root (i.e. relative
// to s.root, not inside any volume). It exists so callers can access
// global metadata files like meta_index/product_index.json without
// synthesising a fake "volume".
func (s *FSStorage) ReadRoot(entry string) ([]byte, error) {
	entry = strings.TrimLeft(entry, "/")
	if entry == "" {
		return nil, ErrEntryNotFound
	}
	data, err := fs.ReadFile(s.fsys, s.join(entry))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrEntryNotFound
	}
	return data, err
}

func (s *FSStorage) join(name string) string {
	if s.root == "" {
		return name
	}
	return path.Join(s.root, name)
}

// ============================================================================
// fsVolume — Volume backed by fs.FS + a base directory
// ============================================================================

type fsVolume struct {
	fsys fs.FS
	base string // slash-separated absolute path within fsys
}

func (v *fsVolume) ReadAll(entry string) ([]byte, error) {
	data, err := fs.ReadFile(v.fsys, path.Join(v.base, entry))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrEntryNotFound
	}
	return data, err
}

func (v *fsVolume) ReadAt(entry string, off, n int64) ([]byte, error) {
	// fs.FS has no ReadAt affordance; simulate it. This is fine for
	// small JSON files; if we ever ship JSONL + offset index we'll
	// need a Volume implementation that actually holds an *os.File
	// or a bytes.Reader.
	data, err := v.ReadAll(entry)
	if err != nil {
		return nil, err
	}
	if off < 0 || off > int64(len(data)) {
		return nil, errors.New("storage: offset out of range")
	}
	end := off + n
	if end > int64(len(data)) {
		end = int64(len(data))
	}
	return data[off:end], nil
}

func (v *fsVolume) Stat(entry string) (Stat, error) {
	info, err := fs.Stat(v.fsys, path.Join(v.base, entry))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Stat{}, ErrEntryNotFound
		}
		return Stat{}, err
	}
	return Stat{
		Size:    info.Size(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}, nil
}

func (v *fsVolume) List(prefix string) ([]string, error) {
	var out []string
	err := fs.WalkDir(v.fsys, v.base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(p, v.base)
		rel = strings.TrimPrefix(rel, "/")
		if prefix == "" || strings.HasPrefix(rel, prefix) {
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (v *fsVolume) Close() error { return nil }
