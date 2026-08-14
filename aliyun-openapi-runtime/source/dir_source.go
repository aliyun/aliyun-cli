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

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/source/format"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/indexed"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/jsonl"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/pbmeta"
	"github.com/aliyun/aliyun-openapi-runtime/source/storage"
)

// dirSource serves user/override metadata plugins.
// Only packaged layouts are supported: manifest.json plus an indexed metadata blob (JSONL or Protobuf).
type dirSource struct {
	root    string
	kind    Kind
	store   *storage.DirStorage
	plugins map[string]*pluginSnapshot
}

type pluginDescriptor struct {
	name     string
	code     string
	manifest *schema.PluginManifest
	metadata *schema.MetadataDescriptor
}

// pluginSnapshot is the invocation-scoped, validated view of one plugin.
// The Reader's index is immutable; payload reads bind it to a freshly opened
// Volume so no file handle needs to remain open in the cache.
type pluginSnapshot struct {
	plugin *pluginDescriptor
	reader *indexed.Reader
}

func NewUserPluginSource(root string) Source {
	return &dirSource{
		root: root, kind: KindUser, store: storage.NewDirStorage(root),
		plugins: map[string]*pluginSnapshot{},
	}
}

func NewOverrideSource(root string) Source {
	return &dirSource{
		root: root, kind: KindOverride, store: storage.NewDirStorage(root),
		plugins: map[string]*pluginSnapshot{},
	}
}

func (s *dirSource) Kind() Kind { return s.kind }

func (s *dirSource) LoadProduct(code string) (*meta.Product, *Provenance, error) {
	snapshot, err := s.loadPluginSnapshot(code)
	if err != nil {
		return nil, nil, err
	}
	plugin := snapshot.plugin
	metadataIndex := snapshot.reader.Index()
	versions := append([]string(nil), plugin.manifest.APIVersions.Supported...)
	if len(versions) == 0 {
		versions = versionsFromIndex(metadataIndex)
	}
	sort.Strings(versions)
	defaultVersion := plugin.manifest.APIVersions.Default
	if defaultVersion == "" && len(versions) > 0 {
		defaultVersion = versions[len(versions)-1]
	}
	product := &meta.Product{
		Code: plugin.code, Versions: versions, DefaultVersion: defaultVersion,
		MinCliVersion: plugin.manifest.MinCliVersion,
		Name:          meta.Description{ZH: plugin.manifest.ProductName["zh"], EN: plugin.manifest.ProductName["en"]},
		Description:   meta.Description{EN: plugin.manifest.Description},
		Endpoints:     metadataIndex.Product.Endpoints(),
	}
	return product, s.provenance(plugin, defaultVersion), nil
}

func (s *dirSource) LoadAPIIndex(code, version string) (*meta.APIIndex, error) {
	snapshot, err := s.loadPluginSnapshot(code)
	if err != nil {
		return nil, err
	}
	idx, err := snapshot.reader.APIIndex(snapshot.plugin.code, version)
	if errors.Is(err, storage.ErrEntryNotFound) {
		return nil, ErrNotFound
	}
	return idx, err
}

func (s *dirSource) LoadAPI(code, version, name string) (*meta.API, error) {
	snapshot, err := s.loadPluginSnapshot(code)
	if err != nil {
		return nil, err
	}
	plugin := snapshot.plugin
	vol, err := s.store.Open(plugin.name)
	if err != nil {
		return nil, normalizeOpenError(err)
	}
	defer vol.Close()
	reader := snapshot.reader.ForVolume(vol)

	var api *meta.API
	if isProtobufMetadata(plugin.metadata) {
		protobufReader := pbmeta.NewReader(reader)
		api, err = protobufReader.ReadAPI(version, name)
		if errors.Is(err, storage.ErrEntryNotFound) {
			return nil, ErrNotFound
		}
		if err == nil {
			api.Endpoints = protobufReader.ProductEndpoints()
		}
	} else {
		payload, readErr := reader.ReadAPI(version, name)
		if errors.Is(readErr, storage.ErrEntryNotFound) {
			return nil, ErrNotFound
		}
		if readErr != nil {
			return nil, readErr
		}
		api, err = format.DecodeAPIJSON(payload, version+"/"+name)
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
		api.ProductCode = plugin.code
	}
	return api, nil
}

// loadPluginSnapshot resolves, parses and structurally validates one plugin at most once during this Source's lifetime (normally one CLI invocation).
func (s *dirSource) loadPluginSnapshot(code string) (*pluginSnapshot, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if s.plugins == nil {
		s.plugins = map[string]*pluginSnapshot{}
	}
	if snapshot := s.plugins[code]; snapshot != nil {
		return snapshot, nil
	}
	plugin, err := s.resolve(code)
	if err != nil {
		return nil, err
	}
	vol, err := s.store.Open(plugin.name)
	if err != nil {
		return nil, normalizeOpenError(err)
	}
	defer vol.Close()
	reader, err := indexed.Open(vol, plugin.metadata.Index, plugin.metadata.Data)
	if err != nil {
		return nil, err
	}
	snapshot := &pluginSnapshot{plugin: plugin, reader: reader}
	s.plugins[code] = snapshot
	return snapshot, nil
}

func (s *dirSource) resolve(code string) (*pluginDescriptor, error) {
	if s.root == "" || code == "" {
		return nil, ErrNotFound
	}
	// Installed metadata plugins have exactly one canonical directory name.
	name := "aliyun-cli-" + code
	plugin, ok, err := s.inspect(name)
	if errors.Is(err, storage.ErrVolumeNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if ok && plugin.code == code {
		return plugin, nil
	}
	return nil, ErrNotFound
}

func (s *dirSource) inspect(name string) (*pluginDescriptor, bool, error) {
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
	plugin := &pluginDescriptor{name: name, code: name, manifest: &manifest}
	if manifest.ProductCode != "" {
		plugin.code = strings.ToLower(manifest.ProductCode)
	} else if manifest.Command != "" {
		plugin.code = strings.ToLower(manifest.Command)
	} else {
		plugin.code = strings.TrimPrefix(strings.ToLower(name), "aliyun-cli-")
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
		plugin.metadata = &copy
	} else if _, statErr := vol.Stat(schema.MetadataIndexFile); statErr == nil {
		plugin.metadata = defaultMetadataDescriptor()
	}
	if plugin.metadata == nil {
		// Manifest present but no indexed metadata blob → unsupported layout.
		return nil, false, nil
	}
	return plugin, true, nil
}

func validateMetadataDescriptor(d *schema.MetadataDescriptor) error {
	if d.Schema != schema.SchemaName || d.SchemaVersion != schema.SchemaVersion || d.LayoutVersion != schema.LayoutVersion {
		return fmt.Errorf("unsupported metadata descriptor format=%q schema=%q schemaVersion=%d layout=%q layoutVersion=%d", d.Format, d.Schema, d.SchemaVersion, d.Layout, d.LayoutVersion)
	}
	if !isJSONLMetadata(d) && !isProtobufMetadata(d) {
		return fmt.Errorf("unsupported metadata descriptor format=%q schema=%q schemaVersion=%d layout=%q layoutVersion=%d", d.Format, d.Schema, d.SchemaVersion, d.Layout, d.LayoutVersion)
	}
	return nil
}

func isJSONLMetadata(d *schema.MetadataDescriptor) bool {
	return d != nil && d.Format == schema.FormatJSON && d.Layout == jsonl.LayoutName
}

func isProtobufMetadata(d *schema.MetadataDescriptor) bool {
	return d != nil && d.Format == schema.FormatProtobuf && d.Layout == pbmeta.LayoutName
}

func defaultMetadataDescriptor() *schema.MetadataDescriptor {
	return &schema.MetadataDescriptor{
		Format: schema.FormatJSON, Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion,
		Layout: jsonl.LayoutName, LayoutVersion: schema.LayoutVersion,
		Index: schema.MetadataIndexFile, Data: schema.MetadataDataFile,
	}
}

func versionsFromIndex(idx indexed.Index) []string {
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

func (s *dirSource) provenance(plugin *pluginDescriptor, version string) *Provenance {
	origin := filepath.Join(s.root, plugin.name)
	p := &Provenance{Kind: s.kind, Version: version, Origin: origin}
	if info, err := os.Stat(origin); err == nil {
		p.InstalledAt = info.ModTime()
	}
	return p
}
