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

// Package jsonl names the indexed-JSONL metadata plugin layout.
// Shared index/data random-access lives in source/format/indexed; this package
// only exposes the LayoutName used in MetadataDescriptor.Layout.
package jsonl

// LayoutName identifies the indexed-JSONL physical layout in MetadataDescriptor.Layout.
const LayoutName = "jsonl"
