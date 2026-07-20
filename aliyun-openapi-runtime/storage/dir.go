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
	"os"
	"path/filepath"
)

// DirStorage is a Storage backed by a real filesystem directory. It
// is the storage of choice for user-installed plugins
// (~/.aliyun/plugins/) and for developer overrides pointed to by
// $ALIYUN_CLI_PLUGINS_DIR_OVERRIDE.
type DirStorage struct {
	root string
}

// NewDirStorage returns a DirStorage rooted at absolute path root.
// The directory does not need to exist yet; Storage methods will
// surface fs.ErrNotExist / ErrVolumeNotFound naturally when it is
// missing.
func NewDirStorage(root string) *DirStorage {
	return &DirStorage{root: root}
}

// Root returns the absolute filesystem root of the storage.
func (s *DirStorage) Root() string { return s.root }

// Open opens the volume named name.
func (s *DirStorage) Open(name string) (Volume, error) {
	dir := filepath.Join(s.root, name)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrVolumeNotFound
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrVolumeNotFound
	}
	return &dirVolume{base: dir}, nil
}

// ReadRoot reads a file located directly under the storage root
// (outside any volume). Missing file -> ErrEntryNotFound.
func (s *DirStorage) ReadRoot(entry string) ([]byte, error) {
	if entry == "" {
		return nil, ErrEntryNotFound
	}
	data, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(entry)))
	if err != nil && os.IsNotExist(err) {
		return nil, ErrEntryNotFound
	}
	return data, err
}

// List enumerates every direct child directory of the storage root.
// Missing root -> empty result, not an error.
func (s *DirStorage) List() ([]string, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
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

// ============================================================================
// dirVolume — Volume backed by a local directory
// ============================================================================

type dirVolume struct {
	base string
}

func (v *dirVolume) ReadAll(entry string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(v.base, filepath.FromSlash(entry)))
	if err != nil && os.IsNotExist(err) {
		return nil, ErrEntryNotFound
	}
	return data, err
}

func (v *dirVolume) ReadAt(entry string, off, n int64) ([]byte, error) {
	f, err := os.Open(filepath.Join(v.base, filepath.FromSlash(entry)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrEntryNotFound
		}
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, n)
	m, err := f.ReadAt(buf, off)
	if err != nil && err.Error() != "EOF" {
		// io.EOF at partial read is expected when reading past size.
		if m == 0 {
			return nil, err
		}
	}
	return buf[:m], nil
}

func (v *dirVolume) Stat(entry string) (Stat, error) {
	info, err := os.Stat(filepath.Join(v.base, filepath.FromSlash(entry)))
	if err != nil {
		if os.IsNotExist(err) {
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

func (v *dirVolume) List(prefix string) ([]string, error) {
	var out []string
	prefixNative := filepath.FromSlash(prefix)
	err := filepath.WalkDir(v.base, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(v.base, p)
		if rerr != nil {
			return rerr
		}
		if prefix == "" || hasPrefixNativeOrSlash(rel, prefixNative, prefix) {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return out, nil
}

func (v *dirVolume) Close() error { return nil }

func hasPrefixNativeOrSlash(s, prefixNative, prefixSlash string) bool {
	if len(prefixNative) > 0 && len(s) >= len(prefixNative) && s[:len(prefixNative)] == prefixNative {
		return true
	}
	sSlash := filepath.ToSlash(s)
	return len(prefixSlash) > 0 && len(sSlash) >= len(prefixSlash) && sSlash[:len(prefixSlash)] == prefixSlash
}
