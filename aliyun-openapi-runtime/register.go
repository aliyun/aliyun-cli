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
	"io/fs"

	"github.com/aliyun/aliyun-openapi-runtime/jsoncmd"
	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

// Options collects the wiring knobs of the aliyun-openapi-runtime engine. It
// is deliberately small; anything that changes at runtime (like which
// plugin the user just installed) is discovered from the file system
// by the Source layer instead of being passed in here.
type Options struct {
	// BaselineFS is the embedded baseline metadata tree
	// (metas/ + meta_index/). It is INJECTED by the host so the
	// engine module carries no baseline data of its own. A nil FS
	// simply omits the baseline layer (useful for tests that rely
	// solely on user/override directories).
	BaselineFS fs.FS

	// BundledBy is the CLI version string embedded in every baseline
	// Provenance record, e.g. "aliyun-cli 3.0.234".
	BundledBy string

	// UserPluginsDir is the directory holding user-installed plugins,
	// typically ~/.aliyun/plugins. Empty -> user layer skipped.
	UserPluginsDir string

	// OverrideDir mirrors $ALIYUN_CLI_PLUGINS_DIR_OVERRIDE. Empty ->
	// override layer skipped.
	OverrideDir string
}

// NewLoader builds a Loader from the configured Sources.
//
// Priority (highest first); each product is owned wholly by the first
// Source that claims it:
//
//	OverrideSource   (Options.OverrideDir, if non-empty)
//	UserPluginSource (Options.UserPluginsDir, if non-empty)
//	BaselineSource   (Options.BaselineFS, if non-nil)
//
// The returned Loader starts empty. Call EnsureProduct before target-specific
// access.
func NewLoader(opts Options) loader.Loader {
	var override, user, baselineSrc source.Source
	if opts.OverrideDir != "" {
		override = source.NewOverrideSource(opts.OverrideDir)
	}
	if opts.UserPluginsDir != "" {
		user = source.NewUserPluginSource(opts.UserPluginsDir)
	}
	if opts.BaselineFS != nil {
		baselineSrc = source.NewBaselineSource(opts.BaselineFS, opts.BundledBy)
	}
	return loader.New(override, user, baselineSrc)
}

// NewEngine returns a command engine backed by a lazily-created loader built
// from opts. Normal dispatch resolves only the requested product; it never
// enumerates unrelated products.
//
// executor may be nil to use the default darabonba-backed executor.
func NewEngine(opts Options, executor runtime.Executor) *jsoncmd.Engine {
	return jsoncmd.NewEngine(
		func() (loader.Loader, error) {
			return NewLoader(opts), nil
		},
		executor,
	)
}
