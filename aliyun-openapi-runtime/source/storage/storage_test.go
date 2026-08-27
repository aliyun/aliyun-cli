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
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"testing/fstest"
)

func TestDirStorageContract(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "root.json"), []byte("root"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "not-a-volume"), []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	volumeDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(filepath.Join(volumeDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(volumeDir, "a.txt"), []byte("abcdef"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(volumeDir, "nested", "b.txt"), []byte("nested"), 0o600); err != nil {
		t.Fatal(err)
	}

	storage := NewDirStorage(root)
	if storage.Root() != root {
		t.Fatalf("Root() = %q, want %q", storage.Root(), root)
	}
	if got, err := storage.ReadRoot("root.json"); err != nil || string(got) != "root" {
		t.Fatalf("ReadRoot() = %q, %v", got, err)
	}
	for _, entry := range []string{"", "missing.json"} {
		if _, err := storage.ReadRoot(entry); !errors.Is(err, ErrEntryNotFound) {
			t.Fatalf("ReadRoot(%q) error = %v, want ErrEntryNotFound", entry, err)
		}
	}

	volumes, err := storage.List()
	if err != nil || !reflect.DeepEqual(volumes, []string{"demo"}) {
		t.Fatalf("List() = %v, %v", volumes, err)
	}
	for _, name := range []string{"missing", "not-a-volume"} {
		if _, err := storage.Open(name); !errors.Is(err, ErrVolumeNotFound) {
			t.Fatalf("Open(%q) error = %v, want ErrVolumeNotFound", name, err)
		}
	}

	volume, err := storage.Open("demo")
	if err != nil {
		t.Fatal(err)
	}
	defer volume.Close()
	if got, err := volume.ReadAll("a.txt"); err != nil || string(got) != "abcdef" {
		t.Fatalf("ReadAll() = %q, %v", got, err)
	}
	if _, err := volume.ReadAll("missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("ReadAll(missing) error = %v", err)
	}
	if got, err := volume.ReadAt("a.txt", 2, 3); err != nil || string(got) != "cde" {
		t.Fatalf("ReadAt(middle) = %q, %v", got, err)
	}
	if got, err := volume.ReadAt("a.txt", 4, 8); err != nil || string(got) != "ef" {
		t.Fatalf("ReadAt(partial) = %q, %v", got, err)
	}
	if _, err := volume.ReadAt("missing", 0, 1); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("ReadAt(missing) error = %v", err)
	}

	stat, err := volume.Stat("a.txt")
	if err != nil || stat.Size != 6 || stat.IsDir {
		t.Fatalf("Stat(file) = %#v, %v", stat, err)
	}
	dirStat, err := volume.Stat("nested")
	if err != nil || !dirStat.IsDir {
		t.Fatalf("Stat(dir) = %#v, %v", dirStat, err)
	}
	if _, err := volume.Stat("missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("Stat(missing) error = %v", err)
	}

	entries, err := volume.List("")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)
	if want := []string{"a.txt", "nested/b.txt"}; !reflect.DeepEqual(entries, want) {
		t.Fatalf("List(all) = %v, want %v", entries, want)
	}
	if got, err := volume.List("nested"); err != nil || !reflect.DeepEqual(got, []string{"nested/b.txt"}) {
		t.Fatalf("List(prefix) = %v, %v", got, err)
	}
	if err := volume.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
}

func TestDirStorageMissingRoot(t *testing.T) {
	storage := NewDirStorage(filepath.Join(t.TempDir(), "missing"))
	if got, err := storage.List(); err != nil || got != nil {
		t.Fatalf("List(missing root) = %v, %v", got, err)
	}
	if _, err := storage.Open("demo"); !errors.Is(err, ErrVolumeNotFound) {
		t.Fatalf("Open() error = %v, want ErrVolumeNotFound", err)
	}
}

func TestFSStorageContract(t *testing.T) {
	files := fstest.MapFS{
		"bundle/root.json":         {Data: []byte("root")},
		"bundle/demo/a.txt":        {Data: []byte("abcdef")},
		"bundle/demo/nested/b.txt": {Data: []byte("nested")},
		"bundle/not-a-volume":      {Data: []byte("file")},
	}
	storage := NewFSStorage(files, "/bundle/")
	if storage.root != "bundle" {
		t.Fatalf("root = %q, want bundle", storage.root)
	}
	if got, err := storage.ReadRoot("/root.json"); err != nil || string(got) != "root" {
		t.Fatalf("ReadRoot() = %q, %v", got, err)
	}
	for _, entry := range []string{"", "missing.json"} {
		if _, err := storage.ReadRoot(entry); !errors.Is(err, ErrEntryNotFound) {
			t.Fatalf("ReadRoot(%q) error = %v, want ErrEntryNotFound", entry, err)
		}
	}

	volumes, err := storage.List()
	if err != nil || !reflect.DeepEqual(volumes, []string{"demo"}) {
		t.Fatalf("List() = %v, %v", volumes, err)
	}
	for _, name := range []string{"missing", "not-a-volume"} {
		if _, err := storage.Open(name); !errors.Is(err, ErrVolumeNotFound) {
			t.Fatalf("Open(%q) error = %v, want ErrVolumeNotFound", name, err)
		}
	}

	volume, err := storage.Open("demo")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := volume.ReadAll("a.txt"); err != nil || string(got) != "abcdef" {
		t.Fatalf("ReadAll() = %q, %v", got, err)
	}
	if _, err := volume.ReadAll("missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("ReadAll(missing) error = %v", err)
	}
	if _, err := volume.ReadAt("a.txt", 0, 1); !errors.Is(err, ErrReadAtUnsupported) {
		t.Fatalf("ReadAt() error = %v, want ErrReadAtUnsupported", err)
	}
	stat, err := volume.Stat("a.txt")
	if err != nil || stat.Size != 6 || stat.IsDir {
		t.Fatalf("Stat(file) = %#v, %v", stat, err)
	}
	if _, err := volume.Stat("missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("Stat(missing) error = %v", err)
	}
	entries, err := volume.List("")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)
	if want := []string{"a.txt", "nested/b.txt"}; !reflect.DeepEqual(entries, want) {
		t.Fatalf("List(all) = %v, want %v", entries, want)
	}
	if got, err := volume.List("nested"); err != nil || !reflect.DeepEqual(got, []string{"nested/b.txt"}) {
		t.Fatalf("List(prefix) = %v, %v", got, err)
	}
	if err := volume.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
}

func TestFSStorageAtRootAndMissingRoot(t *testing.T) {
	rootStorage := NewFSStorage(fstest.MapFS{"demo/a.txt": {Data: []byte("a")}}, "")
	if got, err := rootStorage.List(); err != nil || !reflect.DeepEqual(got, []string{"demo"}) {
		t.Fatalf("root List() = %v, %v", got, err)
	}

	missingStorage := NewFSStorage(fstest.MapFS{}, "missing")
	if _, err := missingStorage.List(); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing root List() error = %v, want fs.ErrNotExist", err)
	}
}

func TestIsNotExist(t *testing.T) {
	for _, err := range []error{ErrVolumeNotFound, ErrEntryNotFound, fs.ErrNotExist, errors.Join(errors.New("context"), ErrEntryNotFound)} {
		if !IsNotExist(err) {
			t.Fatalf("IsNotExist(%v) = false", err)
		}
	}
	if IsNotExist(errors.New("other")) || IsNotExist(nil) {
		t.Fatal("IsNotExist accepted an unrelated error")
	}
}
