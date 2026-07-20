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

// Package jsonl implements the indexed JSONL physical layout used by metadata
// plugins. The index is loaded eagerly; full API definitions stay in one data file and are fetched lazily with storage.Volume.ReadAt.
package jsonl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/storage"
)

const (
	SchemaName    = "aliyun-openapi-meta"
	SchemaVersion = 1
	LayoutName    = "jsonl"
	LayoutVersion = 1
)

// Index is the small, eagerly loaded routing table for metadata.jsonl.
type Index struct {
	Schema        string   `json:"schema"`
	SchemaVersion int      `json:"schemaVersion"`
	LayoutVersion int      `json:"layoutVersion"`
	DataFile      string   `json:"dataFile"`
	DataSize      int64    `json:"dataSize"`
	DataSHA256    string   `json:"dataSha256"`
	Product       Product  `json:"product,omitempty"`
	APIs          []Record `json:"apis"`
}

// Product holds metadata shared by every API record in the package. Keeping
// endpoint maps here avoids repeating the same regional table on every JSONL
// line and mirrors the baseline product catalog model.
type Product struct {
	GlobalEndpoint       string            `json:"globalEndpoint,omitempty"`
	RegionalEndpoints    map[string]string `json:"regionalEndpoints,omitempty"`
	RegionalVPCEndpoints map[string]string `json:"regionalVpcEndpoints,omitempty"`
	LocationServiceCode  string            `json:"locationServiceCode,omitempty"`
}

func (p Product) Endpoints() meta.Endpoints {
	return meta.Endpoints{
		Global: p.GlobalEndpoint,
		Public: p.RegionalEndpoints,
		VPC:    p.RegionalVPCEndpoints,
	}
}

// Record identifies one JSON line and carries enough presentation metadata to build command stubs without reading the full record.
type Record struct {
	APIVersion    string `json:"apiVersion"`
	APIName       string `json:"apiName"`
	CommandName   string `json:"commandName"`
	CmdFullName   string `json:"cmdFullName,omitempty"`
	DescriptionZH string `json:"descriptionZh,omitempty"`
	DescriptionEN string `json:"descriptionEn,omitempty"`
	Deprecated    bool   `json:"deprecated,omitempty"`
	Offset        int64  `json:"offset"`
	Length        int64  `json:"length"`
}

// Reader is an immutable random-access view over a validated index and data file. It does not own the Volume; callers close the Volume.
type Reader struct {
	vol      storage.Volume
	index    Index
	dataFile string
	records  map[string]Record
}

// Open loads and validates indexFile. dataFile may be empty, in which case the
// index's dataFile field (or the standard filename) is used.
func Open(vol storage.Volume, indexFile, dataFile string) (*Reader, error) {
	if indexFile == "" {
		indexFile = schema.MetadataIndexFile
	}
	if err := validateEntryName(indexFile); err != nil {
		return nil, err
	}
	raw, err := vol.ReadAll(indexFile)
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("decode JSONL index %s: %w", indexFile, err)
	}
	if dataFile == "" {
		dataFile = idx.DataFile
	}
	if dataFile == "" {
		dataFile = schema.MetadataDataFile
	}
	if err := validateEntryName(dataFile); err != nil {
		return nil, err
	}
	if idx.DataFile != "" && idx.DataFile != dataFile {
		return nil, fmt.Errorf("JSONL index dataFile %q does not match manifest data %q", idx.DataFile, dataFile)
	}
	stat, err := vol.Stat(dataFile)
	if err != nil {
		return nil, err
	}
	if stat.IsDir {
		return nil, fmt.Errorf("JSONL data %s is a directory", dataFile)
	}
	if err := validateIndex(&idx, stat.Size); err != nil {
		return nil, err
	}
	records := make(map[string]Record, len(idx.APIs))
	for _, rec := range idx.APIs {
		records[recordKey(rec.APIVersion, rec.APIName)] = rec
	}
	return &Reader{vol: vol, index: idx, dataFile: dataFile, records: records}, nil
}

// VerifyChecksum performs the intentionally non-lazy, install-time integrity
// check. Runtime startup should normally call only Open so the large data file
// is not read in full on every CLI invocation.
func (r *Reader) VerifyChecksum() error {
	if r.index.DataSHA256 == "" {
		return errors.New("JSONL index is missing dataSha256")
	}
	data, err := r.vol.ReadAll(r.dataFile)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	want := strings.TrimPrefix(strings.ToLower(r.index.DataSHA256), "sha256:")
	if _, err := hex.DecodeString(want); err != nil || len(want) != sha256.Size*2 {
		return fmt.Errorf("invalid JSONL dataSha256 %q", r.index.DataSHA256)
	}
	got := hex.EncodeToString(digest[:])
	if got != want {
		return fmt.Errorf("JSONL data checksum mismatch: index=%s actual=sha256:%s", r.index.DataSHA256, got)
	}
	return nil
}

// Index returns a copy of the parsed index header and records.
func (r *Reader) Index() Index {
	idx := r.index
	idx.APIs = append([]Record(nil), r.index.APIs...)
	return idx
}

// ProductEndpoints returns the product-level endpoint table shared by all API
// records. Reader and its index are immutable after Open.
func (r *Reader) ProductEndpoints() meta.Endpoints {
	return r.index.Product.Endpoints()
}

// ReadAPI returns exactly one JSON document, excluding the JSONL newline.
func (r *Reader) ReadAPI(version, name string) ([]byte, error) {
	rec, ok := r.records[recordKey(version, name)]
	if !ok {
		return nil, storage.ErrEntryNotFound
	}
	data, err := r.vol.ReadAt(r.dataFile, rec.Offset, rec.Length)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != rec.Length {
		return nil, fmt.Errorf("short JSONL record %s/%s: got %d bytes, want %d", version, name, len(data), rec.Length)
	}
	return data, nil
}

// APIIndex projects index records for one API version into the runtime model.
func (r *Reader) APIIndex(product, version string) (*meta.APIIndex, error) {
	idx := &meta.APIIndex{ProductCode: product, Version: version, Entries: map[string]meta.APIIndexEntry{}}
	for _, rec := range r.index.APIs {
		if rec.APIVersion != version {
			continue
		}
		idx.Entries[rec.APIName] = meta.APIIndexEntry{
			APIName: rec.APIName, CmdName: rec.CommandName, CmdFullName: rec.CmdFullName,
			Description: meta.Description{ZH: rec.DescriptionZH, EN: rec.DescriptionEN}, Deprecated: rec.Deprecated,
		}
	}
	if len(idx.Entries) == 0 {
		return nil, storage.ErrEntryNotFound
	}
	idx.BuildCmdIndex()
	return idx, nil
}

func validateIndex(idx *Index, actualSize int64) error {
	if idx.Schema != SchemaName || idx.SchemaVersion != SchemaVersion || idx.LayoutVersion != LayoutVersion {
		return fmt.Errorf("unsupported JSONL contract schema=%q schemaVersion=%d layoutVersion=%d", idx.Schema, idx.SchemaVersion, idx.LayoutVersion)
	}
	if idx.DataSize != actualSize {
		return fmt.Errorf("JSONL data size mismatch: index=%d actual=%d", idx.DataSize, actualSize)
	}
	records := append([]Record(nil), idx.APIs...)
	sort.Slice(records, func(i, j int) bool { return records[i].Offset < records[j].Offset })
	seen := make(map[string]struct{}, len(records))
	var end int64
	for i, rec := range records {
		if rec.APIVersion == "" || rec.APIName == "" || rec.CommandName == "" {
			return errors.New("JSONL index record is missing apiVersion, apiName, or commandName")
		}
		if rec.Offset < 0 || rec.Length <= 0 || rec.Offset > actualSize || rec.Length > actualSize-rec.Offset {
			return fmt.Errorf("JSONL record %s/%s has invalid byte range offset=%d length=%d", rec.APIVersion, rec.APIName, rec.Offset, rec.Length)
		}
		if i > 0 && rec.Offset < end {
			return fmt.Errorf("JSONL record %s/%s overlaps the previous record", rec.APIVersion, rec.APIName)
		}
		end = rec.Offset + rec.Length
		key := recordKey(rec.APIVersion, rec.APIName)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate JSONL record %s/%s", rec.APIVersion, rec.APIName)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func recordKey(version, name string) string { return version + "\x00" + name }

func validateEntryName(name string) error {
	clean := path.Clean(name)
	if name == "" || path.IsAbs(name) || filepath.IsAbs(name) || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(name, `\`) {
		return fmt.Errorf("unsafe JSONL entry path %q", name)
	}
	return nil
}
