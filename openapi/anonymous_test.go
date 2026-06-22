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

package openapi

import (
	"bytes"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
)

// --- Test O-01 ~ O-03: ShouldUseOpenapi ---
// Anonymous mode is only supported through the plugin execution path (execution.go),
// commando.go uses ShouldUseOpenapi (product-based) without profile awareness.

func TestShouldUseOpenapi_ProductRouting(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))

	t.Run("O-01: non-SLS product does not use openapi", func(t *testing.T) {
		product := &meta.Product{Code: "ECS"}
		assert.False(t, ShouldUseOpenapi(ctx, product))
	})

	t.Run("O-02: SLS product uses openapi", func(t *testing.T) {
		product := &meta.Product{Code: "SLS"}
		assert.True(t, ShouldUseOpenapi(ctx, product))
	})

	t.Run("O-03: case insensitive SLS check", func(t *testing.T) {
		product := &meta.Product{Code: "sls"}
		assert.True(t, ShouldUseOpenapi(ctx, product))
	})
}
