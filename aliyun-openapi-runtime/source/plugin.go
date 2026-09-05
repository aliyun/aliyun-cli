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
	"fmt"
	"path/filepath"

	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/indexed"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/jsonl"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/pbmeta"
	"github.com/aliyun/aliyun-openapi-runtime/source/storage"
)

// ValidateMetadataPlugin validates a metadata plugin's declared contract,
// index structure, data bounds, and payload checksum.
func ValidateMetadataPlugin(dir string, descriptor *schema.MetadataDescriptor) error {
	if descriptor == nil {
		return fmt.Errorf("metadata descriptor is required")
	}
	if descriptor.Schema != schema.SchemaName ||
		descriptor.SchemaVersion != schema.SchemaVersion ||
		descriptor.LayoutVersion != schema.LayoutVersion {
		return unsupportedMetadataContract(descriptor)
	}
	isJSONL := descriptor.Format == schema.FormatJSON && descriptor.Layout == jsonl.LayoutName
	isProtobuf := descriptor.Format == schema.FormatProtobuf && descriptor.Layout == pbmeta.LayoutName
	if !isJSONL && !isProtobuf {
		return unsupportedMetadataContract(descriptor)
	}

	store := storage.NewDirStorage(filepath.Dir(dir))
	volume, err := store.Open(filepath.Base(dir))
	if err != nil {
		return fmt.Errorf("open metadata plugin package: %w", err)
	}
	defer volume.Close()

	reader, err := indexed.Open(volume, descriptor.Index, descriptor.Data)
	if err != nil {
		return fmt.Errorf("invalid metadata plugin index: %w", err)
	}
	if err := reader.VerifyChecksum(); err != nil {
		return fmt.Errorf("invalid metadata plugin data: %w", err)
	}
	return nil
}

func unsupportedMetadataContract(descriptor *schema.MetadataDescriptor) error {
	return fmt.Errorf(
		"unsupported metadata contract format=%q schema=%q schemaVersion=%d layout=%q layoutVersion=%d",
		descriptor.Format,
		descriptor.Schema,
		descriptor.SchemaVersion,
		descriptor.Layout,
		descriptor.LayoutVersion,
	)
}
