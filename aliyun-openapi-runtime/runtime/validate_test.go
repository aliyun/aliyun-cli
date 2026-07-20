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

package runtime

import (
	"errors"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func roaAPI() *meta.API {
	return &meta.API{
		Name:    "ListLayerVersions",
		Version: "2023-03-30",
		Method:  "GET",
		Style:   meta.StyleRESTful,
		URL:     "/2023-03-30/layers/{layerName}/versions",
		Parameters: []meta.Parameter{
			{Name: "layer_name", RawName: "layerName", Type: meta.TypeString, Position: meta.PosPath, Required: true, Options: []string{"--layer-name"}},
			{Name: "start_version", RawName: "startVersion", Type: meta.TypeString, Position: meta.PosQuery},
			{Name: "limit", RawName: "limit", Type: meta.TypeInteger, Position: meta.PosQuery},
		},
	}
}

func TestValidateRequiredMissing(t *testing.T) {
	err := ValidateRequired(roaAPI(), map[string]any{})
	var mre *MissingRequiredError
	if !errors.As(err, &mre) {
		t.Fatalf("expected MissingRequiredError, got %v", err)
	}
	if len(mre.Flags) != 1 || mre.Flags[0] != "--layer-name" {
		t.Fatalf("missing flags = %v", mre.Flags)
	}
}

func TestValidateRequiredPresent(t *testing.T) {
	err := ValidateRequired(roaAPI(), map[string]any{"layerName": "my-layer"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRequiredEmptyStringCountsMissing(t *testing.T) {
	err := ValidateRequired(roaAPI(), map[string]any{"layerName": ""})
	if err == nil {
		t.Fatal("empty string should count as missing")
	}
}

// TestAssembleROAPathSubstitution confirms that once the required path
// param is present, the placeholder is substituted (the flip side of
// the missing-param bug that produced "Illegal Path Character").
func TestAssembleROAPathSubstitution(t *testing.T) {
	ec := &ExecContext{
		API:      roaAPI(),
		Endpoint: "fcv3.cn-hangzhou.aliyuncs.com",
		Args: map[string]any{
			"layerName": "my-layer",
			"limit":     "10",
		},
	}
	req, err := Assemble(ec)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if req.Pathname != "/2023-03-30/layers/my-layer/versions" {
		t.Fatalf("pathname = %q", req.Pathname)
	}
	if req.Query["limit"] != "10" {
		t.Fatalf("query limit = %q", req.Query["limit"])
	}
}
