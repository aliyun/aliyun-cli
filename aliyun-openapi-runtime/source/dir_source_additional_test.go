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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/indexed"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/jsonl"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/pbmeta"
	"github.com/aliyun/aliyun-openapi-runtime/source/storage"
)

func writeAdditionalJSONLPlugin(t *testing.T, root, code, indexedName, payloadName string, manifest schema.PluginManifest) string {
	t.Helper()
	pluginDir := filepath.Join(root, "aliyun-cli-"+code)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), manifestData, 0o600); err != nil {
		t.Fatal(err)
	}
	definition := schema.CommandDefinition{
		Name: payloadName, CmdName: "run", Operation: &schema.OperationConfig{
			Action: payloadName, APIVersion: "v1", Method: "POST", APIStyle: "RPC",
		},
	}
	payload, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	data := append(payload, '\n')
	if err := os.WriteFile(filepath.Join(pluginDir, schema.MetadataDataFile), data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	index := indexed.Index{
		Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion, LayoutVersion: schema.LayoutVersion,
		DataFile: schema.MetadataDataFile, DataSize: int64(len(data)), DataSHA256: "sha256:" + hex.EncodeToString(digest[:]),
		APIs: []indexed.Record{{APIVersion: "v1", APIName: indexedName, CommandName: "run", Offset: 0, Length: int64(len(payload))}},
	}
	indexData, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, schema.MetadataIndexFile), indexData, 0o600); err != nil {
		t.Fatal(err)
	}
	return pluginDir
}

func TestDirSourceKindsAndDescriptorHelpers(t *testing.T) {
	user := NewUserPluginSource(t.TempDir())
	override := NewOverrideSource(t.TempDir())
	if user.Kind() != KindUser || override.Kind() != KindOverride {
		t.Fatalf("source kinds = %v, %v", user.Kind(), override.Kind())
	}

	descriptor := defaultMetadataDescriptor()
	if err := validateMetadataDescriptor(descriptor); err != nil {
		t.Fatal(err)
	}
	if !isJSONLMetadata(descriptor) || isProtobufMetadata(descriptor) || isJSONLMetadata(nil) || isProtobufMetadata(nil) {
		t.Fatal("metadata format classification returned unexpected result")
	}
	protobufDescriptor := &schema.MetadataDescriptor{
		Format: schema.FormatProtobuf, Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion,
		Layout: pbmeta.LayoutName, LayoutVersion: schema.LayoutVersion,
	}
	if err := validateMetadataDescriptor(protobufDescriptor); err != nil || !isProtobufMetadata(protobufDescriptor) {
		t.Fatalf("protobuf descriptor = %v", err)
	}
	for _, invalid := range []*schema.MetadataDescriptor{
		{},
		{Format: schema.FormatJSON, Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion, Layout: "unknown", LayoutVersion: schema.LayoutVersion},
	} {
		if err := validateMetadataDescriptor(invalid); err == nil {
			t.Fatalf("validateMetadataDescriptor(%#v) succeeded", invalid)
		}
	}

	versions := versionsFromIndex(indexed.Index{APIs: []indexed.Record{
		{APIVersion: "v2"}, {APIVersion: "v1"}, {APIVersion: "v2"},
	}})
	if len(versions) != 2 {
		t.Fatalf("versionsFromIndex() = %v", versions)
	}
	if !errors.Is(normalizeOpenError(storage.ErrVolumeNotFound), ErrNotFound) {
		t.Fatal("normalizeOpenError did not map missing volume")
	}
	cause := errors.New("other")
	if !errors.Is(normalizeOpenError(cause), cause) {
		t.Fatal("normalizeOpenError changed unrelated error")
	}
	if descriptor.Format != schema.FormatJSON || descriptor.Layout != jsonl.LayoutName || descriptor.Index == "" || descriptor.Data == "" {
		t.Fatalf("default descriptor = %#v", descriptor)
	}
}

func TestDirSourceUsesManifestAndIndexFallbacks(t *testing.T) {
	root := t.TempDir()
	manifest := schema.PluginManifest{
		Name: "aliyun-cli-demo", Version: "1.2.3", Type: "meta", Command: "DEMO",
		Description: "Demo service", ProductName: map[string]string{"en": "Demo", "zh": "演示"},
		MinCliVersion: "3.0.0",
	}
	pluginDir := writeAdditionalJSONLPlugin(t, root, "demo", "Run", "Run", manifest)

	source := NewOverrideSource(root)
	product, provenance, err := source.LoadProduct(" DEMO ")
	if err != nil {
		t.Fatal(err)
	}
	if product.Code != "demo" || product.DefaultVersion != "v1" || !reflect.DeepEqual(product.Versions, []string{"v1"}) ||
		product.Name.EN != "Demo" || product.MinCliVersion != "3.0.0" {
		t.Fatalf("LoadProduct() = %#v", product)
	}
	if provenance.Kind != KindOverride || provenance.PluginName != manifest.Name || provenance.PluginVersion != manifest.Version ||
		provenance.APIVersion != "v1" || provenance.Origin != pluginDir || provenance.InstalledAt.IsZero() {
		t.Fatalf("provenance = %#v", provenance)
	}
	api, err := source.LoadAPI("demo", "v1", "Run")
	if err != nil || api.ProductCode != "demo" {
		t.Fatalf("LoadAPI() = %#v, %v", api, err)
	}
	if _, err := source.LoadAPIIndex("demo", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadAPIIndex(missing) error = %v", err)
	}
	if _, err := source.LoadAPI("demo", "v1", "Missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadAPI(missing) error = %v", err)
	}
}

func TestDirSourceDescriptorDefaultsAndIdentityValidation(t *testing.T) {
	root := t.TempDir()
	descriptor := &schema.MetadataDescriptor{
		Format: schema.FormatJSON, Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion,
		Layout: jsonl.LayoutName, LayoutVersion: schema.LayoutVersion,
	}
	manifest := schema.PluginManifest{Name: "aliyun-cli-demo", Type: "meta", ProductCode: "DEMO", Metadata: descriptor}
	writeAdditionalJSONLPlugin(t, root, "demo", "IndexedName", "PayloadName", manifest)
	source := NewUserPluginSource(root)
	if _, _, err := source.LoadProduct("demo"); err != nil {
		t.Fatal(err)
	}
	_, err := source.LoadAPI("demo", "v1", "IndexedName")
	if err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("LoadAPI(identity mismatch) error = %v", err)
	}
}

func TestDirSourceManifestAndOpenErrors(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "aliyun-cli-bad")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := NewUserPluginSource(root).LoadProduct("bad"); err == nil {
		t.Fatal("malformed manifest succeeded")
	}

	invalidRoot := t.TempDir()
	invalidManifest := schema.PluginManifest{
		Name: "aliyun-cli-invalid", Type: "meta", ProductCode: "invalid",
		Metadata: &schema.MetadataDescriptor{Format: schema.FormatJSON, Schema: "wrong"},
	}
	invalidDir := filepath.Join(invalidRoot, "aliyun-cli-invalid")
	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(invalidManifest)
	if err := os.WriteFile(filepath.Join(invalidDir, "manifest.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := NewUserPluginSource(invalidRoot).LoadProduct("invalid"); err == nil || !strings.Contains(err.Error(), "unsupported metadata descriptor") {
		t.Fatalf("invalid descriptor error = %v", err)
	}

	missingRoot := t.TempDir()
	manifest := schema.PluginManifest{Name: "aliyun-cli-demo", Type: "meta", ProductCode: "demo"}
	validDir := writeAdditionalJSONLPlugin(t, missingRoot, "demo", "Run", "Run", manifest)
	source := NewUserPluginSource(missingRoot)
	if _, _, err := source.LoadProduct("demo"); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(validDir, validDir+"-moved"); err != nil {
		t.Fatal(err)
	}
	if _, err := source.LoadAPI("demo", "v1", "Run"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadAPI(missing volume) error = %v", err)
	}

	emptySource := NewUserPluginSource("")
	if _, _, err := emptySource.LoadProduct("demo"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty-root LoadProduct() error = %v", err)
	}
}
