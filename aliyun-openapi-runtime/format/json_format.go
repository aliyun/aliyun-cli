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

package format

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	crschema "github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/storage"
)

// JSONFormat decodes the upstream plugin-generator JSON layout:
//
//	<volume>/<api-version>/<APIName>.json         (per-API CommandDefinition)
//	<volume>/manifest.json                        (optional PluginManifest)

// Precision: every json.Decoder created here uses UseNumber() so int64
// scalars survive into meta.API. Consumers that assign to *int64
// should convert from json.Number via .Int64().
type JSONFormat struct{}

// NewJSONFormat returns a JSONFormat. It is safe to reuse across
// goroutines; the type is stateless.
func NewJSONFormat() *JSONFormat { return &JSONFormat{} }

// ============================================================================
// DecodeIndex
// ============================================================================

// DecodeIndex reads a canonical per-version index
// (canonical/<product>/<version>/version.json) out of the product volume and maps it to meta.APIIndex.
func (JSONFormat) DecodeIndex(vol storage.Volume, version string) (*meta.APIIndex, error) {
	entry := version + "/" + crschema.VersionFileName
	data, err := vol.ReadAll(entry)
	if err != nil {
		return nil, err
	}
	var raw crschema.VersionIndex
	if err := decodeJSON(data, &raw); err != nil {
		return nil, fmt.Errorf("decode version index %s: %w", entry, err)
	}

	idx := &meta.APIIndex{
		Version: version,
		Entries: make(map[string]meta.APIIndexEntry, len(raw.APIs)),
	}
	for name, e := range raw.APIs {
		idx.Entries[name] = meta.APIIndexEntry{
			APIName:     name,
			CmdName:     e.CmdName,
			Description: meta.Description{ZH: e.DescriptionZH, EN: e.DescriptionEN},
			Deprecated:  e.Deprecated,
		}
	}
	// Materialise the cmd-name -> APIName reverse map once, so command
	// resolution against a non-default version is an O(1) lookup.
	idx.BuildCmdIndex()
	return idx, nil
}

// ============================================================================
// DecodeAPI
// ============================================================================

// DecodeAPI reads <version>/<APIName>.json out of the given Volume and maps it to meta.API.
func (JSONFormat) DecodeAPI(vol storage.Volume, key APIKey) (*meta.API, error) {
	entry := key.Version + "/" + key.Name + ".json"
	data, err := vol.ReadAll(entry)
	if err != nil {
		return nil, err
	}
	return DecodeAPIJSON(data, entry)
}

// DecodeAPIJSON maps one complete CommandDefinition JSON record to meta.API.
// Both the legacy per-file layout and JSONL records use this schema mapping.
func DecodeAPIJSON(data []byte, identity string) (*meta.API, error) {
	var def crschema.CommandDefinition
	if err := decodeJSON(data, &def); err != nil {
		return nil, fmt.Errorf("decode api %s: %w", identity, err)
	}
	return DecodeCommandDefinition(&def, identity)
}

// DecodeCommandDefinition validates and maps the format-neutral canonical
// schema. JSON and protobuf decoders converge here so runtime behavior stays
// identical across plugin distribution formats.
func DecodeCommandDefinition(def *crschema.CommandDefinition, identity string) (*meta.API, error) {
	if def.Operation == nil {
		return nil, fmt.Errorf("api %s: missing operation block", identity)
	}
	return schemaToAPI(def), nil
}

func DecodePluginManifestJSON(data []byte, dst *crschema.PluginManifest) error {
	return decodeJSON(data, dst)
}

// ============================================================================
// DecodeProduct
// ============================================================================

// DecodeProduct synthesises a Product by scanning the version
// directories inside vol (each direct child directory that contains at
// least one .json file is one supported version). This is used by the
// directory-backed user-plugin source; the baseline source resolves
// products centrally from metadatas/products.json instead.
func (JSONFormat) DecodeProduct(vol storage.Volume, code string) (*meta.Product, error) {
	entries, err := vol.List("")
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, e := range entries {
		// entries here are slash-separated relative paths; the
		// first segment is the version dir.
		i := indexOfSlash(e)
		if i <= 0 {
			continue
		}
		seen[e[:i]] = struct{}{}
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("product %s: no api-version directories", code)
	}
	versions := make([]string, 0, len(seen))
	for v := range seen {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	return &meta.Product{
		Code:           code,
		Versions:       versions,
		DefaultVersion: chooseDefaultVersion(versions),
	}, nil
}

// chooseDefaultVersion picks the lexicographically-newest version as
// the default. API versions are date-stamped (yyyy-MM-dd) so this
// happens to coincide with "most recent", which is the desired
// behaviour for baseline products lacking an explicit manifest.
func chooseDefaultVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	sorted := append([]string(nil), versions...)
	sort.Strings(sorted)
	return sorted[len(sorted)-1]
}

func indexOfSlash(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

// ============================================================================
// helpers
// ============================================================================

// decodeJSON reads bytes with UseNumber() so numeric scalars keep
// their precision.
// DecodeProductsJSON decodes the central metadatas/products.json catalog into dst.
// Exposed so the Source layer can read the catalog without importing encoding/json directly.
func DecodeProductsJSON(data []byte, dst *crschema.ProductsIndex) error {
	return decodeJSON(data, dst)
}

func decodeJSON(data []byte, dst any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := d.Decode(dst); err != nil {
		return err
	}
	// Guard against trailing garbage.
	if d.More() {
		return errors.New("format: trailing bytes after JSON document")
	}
	return nil
}
