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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/format"
	"github.com/aliyun/aliyun-openapi-runtime/jsonl"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/pbmeta"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/storage"
)

// dirSource serves user/override metadata plugins.
// Only packaged layouts are supported: manifest.json plus an indexed metadata blob (JSONL or Protobuf).
type dirSource struct {
	root  string
	kind  Kind
	store *storage.DirStorage
}

type pluginVolume struct {
	name     string
	code     string
	manifest *schema.PluginManifest
	metadata *schema.MetadataDescriptor
}

func NewUserPluginSource(root string) Source {
	return &dirSource{root: root, kind: KindUser, store: storage.NewDirStorage(root)}
}

func NewOverrideSource(root string) Source {
	return &dirSource{root: root, kind: KindOverride, store: storage.NewDirStorage(root)}
}

func (s *dirSource) Kind() Kind { return s.kind }

func (s *dirSource) LoadProduct(code string) (*meta.Product, *Provenance, error) {
	pv, err := s.resolve(code)
	if err != nil {
		return nil, nil, err
	}
	vol, err := s.store.Open(pv.name)
	if err != nil {
		return nil, nil, normalizeOpenError(err)
	}
	defer vol.Close()

	jsonlIndex, err := openMetadataIndex(vol, pv.metadata)
	if err != nil {
		return nil, nil, err
	}
	versions := append([]string(nil), pv.manifest.APIVersions.Supported...)
	if len(versions) == 0 {
		versions = versionsFromIndex(jsonlIndex)
	}
	sort.Strings(versions)
	defaultVersion := pv.manifest.APIVersions.Default
	if defaultVersion == "" && len(versions) > 0 {
		defaultVersion = versions[len(versions)-1]
	}
	product := &meta.Product{
		Code: pv.code, Versions: versions, DefaultVersion: defaultVersion,
		MinCliVersion: pv.manifest.MinCliVersion,
		Name:          meta.Description{ZH: pv.manifest.ProductName["zh"], EN: pv.manifest.ProductName["en"]},
		Description:   meta.Description{EN: pv.manifest.Description},
		Endpoints:     jsonlIndex.Product.Endpoints(),
	}
	return product, s.provenance(pv, defaultVersion), nil
}

func (s *dirSource) LoadIndex(code, version string) (*meta.APIIndex, error) {
	pv, err := s.resolve(code)
	if err != nil {
		return nil, err
	}
	vol, err := s.store.Open(pv.name)
	if err != nil {
		return nil, normalizeOpenError(err)
	}
	defer vol.Close()
	idx, err := loadMetadataAPIIndex(vol, pv.metadata, code, version)
	if errors.Is(err, storage.ErrEntryNotFound) {
		return nil, ErrNotFound
	}
	return idx, err
}

func (s *dirSource) LoadAPI(code, version, name string) (*meta.API, error) {
	pv, err := s.resolve(code)
	if err != nil {
		return nil, err
	}
	vol, err := s.store.Open(pv.name)
	if err != nil {
		return nil, normalizeOpenError(err)
	}
	defer vol.Close()

	var api *meta.API
	if isProtobufMetadata(pv.metadata) {
		reader, openErr := pbmeta.Open(vol, pv.metadata.Index, pv.metadata.Data)
		if openErr != nil {
			return nil, openErr
		}
		api, err = reader.ReadAPI(version, name)
		if errors.Is(err, storage.ErrEntryNotFound) {
			return nil, ErrNotFound
		}
		if err == nil {
			api.Endpoints = reader.ProductEndpoints()
		}
	} else {
		reader, openErr := jsonl.Open(vol, pv.metadata.Index, pv.metadata.Data)
		if openErr != nil {
			return nil, openErr
		}
		record, readErr := reader.ReadAPI(version, name)
		if errors.Is(readErr, storage.ErrEntryNotFound) {
			return nil, ErrNotFound
		}
		if readErr != nil {
			return nil, readErr
		}
		api, err = format.DecodeAPIJSON(record, version+"/"+name)
		if err == nil && (api.Name != name || api.Version != version) {
			return nil, fmt.Errorf("JSONL record identity mismatch: index=%s/%s data=%s/%s", version, name, api.Version, api.Name)
		}
		if err == nil {
			api.Endpoints = reader.ProductEndpoints()
		}
	}
	if err != nil {
		if storage.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if api.ProductCode == "" {
		api.ProductCode = code
	}
	return api, nil
}

func (s *dirSource) resolve(code string) (*pluginVolume, error) {
	if s.root == "" || code == "" {
		return nil, ErrNotFound
	}
	// Installed metadata plugins have exactly one canonical directory name.
	// Never probe aliases or enumerate unrelated plugin directories.
	name := "aliyun-cli-" + code
	pv, ok, err := s.inspect(name)
	if errors.Is(err, storage.ErrVolumeNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if ok && pv.code == code {
		return pv, nil
	}
	return nil, ErrNotFound
}

func (s *dirSource) inspect(name string) (*pluginVolume, bool, error) {
	vol, err := s.store.Open(name)
	if err != nil {
		return nil, false, err
	}
	defer vol.Close()

	raw, err := vol.ReadAll("manifest.json")
	if err != nil {
		if storage.IsNotExist(err) {
			// No manifest → not a packaged meta plugin (scattered *.json is unsupported).
			return nil, false, nil
		}
		return nil, false, err
	}

	var manifest schema.PluginManifest
	if err := format.DecodePluginManifestJSON(raw, &manifest); err != nil {
		return nil, false, err
	}
	pv := &pluginVolume{name: name, code: name, manifest: &manifest}
	if manifest.ProductCode != "" {
		pv.code = strings.ToLower(manifest.ProductCode)
	} else if manifest.Command != "" {
		pv.code = strings.ToLower(manifest.Command)
	} else {
		pv.code = strings.TrimPrefix(strings.ToLower(name), "aliyun-cli-")
	}
	if manifest.Type == schema.DistributionGo || (manifest.Type == "" && manifest.Bin.Path != "" && manifest.Metadata == nil) {
		return nil, false, nil
	}
	if manifest.Metadata != nil {
		if err := validateMetadataDescriptor(manifest.Metadata); err != nil {
			return nil, false, err
		}
		copy := *manifest.Metadata
		if copy.Index == "" {
			copy.Index = schema.MetadataIndexFile
		}
		if copy.Data == "" {
			copy.Data = schema.MetadataDataFile
		}
		pv.metadata = &copy
	} else if _, statErr := vol.Stat(schema.MetadataIndexFile); statErr == nil {
		pv.metadata = defaultMetadataDescriptor()
	}
	if pv.metadata == nil {
		// Manifest present but no indexed metadata blob → unsupported layout.
		return nil, false, nil
	}
	return pv, true, nil
}

func validateMetadataDescriptor(d *schema.MetadataDescriptor) error {
	if d.Schema != jsonl.SchemaName || d.SchemaVersion != jsonl.SchemaVersion || d.LayoutVersion != jsonl.LayoutVersion {
		return fmt.Errorf("unsupported metadata descriptor format=%q schema=%q schemaVersion=%d layout=%q layoutVersion=%d", d.Format, d.Schema, d.SchemaVersion, d.Layout, d.LayoutVersion)
	}
	if !isJSONLMetadata(d) && !isProtobufMetadata(d) {
		return fmt.Errorf("unsupported metadata descriptor format=%q schema=%q schemaVersion=%d layout=%q layoutVersion=%d", d.Format, d.Schema, d.SchemaVersion, d.Layout, d.LayoutVersion)
	}
	return nil
}

func isJSONLMetadata(d *schema.MetadataDescriptor) bool {
	return d != nil && d.Format == "json" && d.Layout == jsonl.LayoutName
}

func isProtobufMetadata(d *schema.MetadataDescriptor) bool {
	return d != nil && d.Format == "protobuf" && d.Layout == pbmeta.LayoutName
}

func openMetadataIndex(vol storage.Volume, d *schema.MetadataDescriptor) (jsonl.Index, error) {
	if isProtobufMetadata(d) {
		reader, err := pbmeta.Open(vol, d.Index, d.Data)
		if err != nil {
			return jsonl.Index{}, err
		}
		return reader.Index(), nil
	}
	reader, err := jsonl.Open(vol, d.Index, d.Data)
	if err != nil {
		return jsonl.Index{}, err
	}
	return reader.Index(), nil
}

func loadMetadataAPIIndex(vol storage.Volume, d *schema.MetadataDescriptor, product, version string) (*meta.APIIndex, error) {
	if isProtobufMetadata(d) {
		reader, err := pbmeta.Open(vol, d.Index, d.Data)
		if err != nil {
			return nil, err
		}
		return reader.APIIndex(product, version)
	}
	reader, err := jsonl.Open(vol, d.Index, d.Data)
	if err != nil {
		return nil, err
	}
	return reader.APIIndex(product, version)
}

func defaultMetadataDescriptor() *schema.MetadataDescriptor {
	return &schema.MetadataDescriptor{Format: "json", Schema: jsonl.SchemaName, SchemaVersion: jsonl.SchemaVersion, Layout: jsonl.LayoutName, LayoutVersion: jsonl.LayoutVersion, Index: schema.MetadataIndexFile, Data: schema.MetadataDataFile}
}

func versionsFromIndex(idx jsonl.Index) []string {
	seen := map[string]struct{}{}
	for _, rec := range idx.APIs {
		seen[rec.APIVersion] = struct{}{}
	}
	versions := make([]string, 0, len(seen))
	for version := range seen {
		versions = append(versions, version)
	}
	return versions
}

func normalizeOpenError(err error) error {
	if errors.Is(err, storage.ErrVolumeNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *dirSource) provenance(pv *pluginVolume, version string) *Provenance {
	origin := filepath.Join(s.root, pv.name)
	p := &Provenance{Kind: s.kind, Version: version, Origin: origin}
	if info, err := os.Stat(origin); err == nil {
		p.InstalledAt = info.ModTime()
	}
	return p
}
