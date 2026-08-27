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

package openapiruntime

import (
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
)

func TestNewLoaderBuildsConfiguredLayers(t *testing.T) {
	loader := NewLoader(Options{
		BaselineFS:     fstest.MapFS{},
		BundledBy:      "test-cli",
		UserPluginsDir: t.TempDir(),
		OverrideDir:    t.TempDir(),
	})
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
	if err := loader.EnsureProduct("missing"); err == nil {
		t.Fatal("EnsureProduct(missing) unexpectedly succeeded")
	}
}

func TestNewEngineUsesRuntimeAssembly(t *testing.T) {
	engine := NewEngine(Options{
		ExternalFlags: []argparser.ExternalFlagSpec{{Name: "--profile", Mode: argparser.ExternalFlagRequired}},
	}, nil)
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}
	if engine.HasProduct("missing") {
		t.Fatal("HasProduct(missing) = true")
	}
}
