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
	"reflect"
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
	err := ValidateRequired(roaAPI(), map[string]any{}, false)
	var mre *MissingRequiredError
	if !errors.As(err, &mre) {
		t.Fatalf("expected MissingRequiredError, got %v", err)
	}
	if len(mre.Flags) != 1 || mre.Flags[0] != "--layer-name" {
		t.Fatalf("missing flags = %v", mre.Flags)
	}
}

func TestValidateRequiredPresent(t *testing.T) {
	err := ValidateRequired(roaAPI(), map[string]any{"layerName": "my-layer"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRequiredEmptyStringCountsMissing(t *testing.T) {
	err := ValidateRequired(roaAPI(), map[string]any{"layerName": ""}, false)
	if err == nil {
		t.Fatal("empty string should count as missing")
	}
}

func TestValidateRequiredEmptyCompositeCountsPresent(t *testing.T) {
	api := roaAPI()
	api.Parameters = append(api.Parameters,
		meta.Parameter{
			Name: "payload", RawName: "payload", Type: meta.TypeObject,
			Position: meta.PosBody, Required: true, Options: []string{"--payload"},
		},
		meta.Parameter{
			Name: "items", RawName: "items", Type: meta.TypeArray,
			Position: meta.PosBody, Required: true, Options: []string{"--items"},
		},
	)

	err := ValidateRequired(api, map[string]any{
		"layerName": "my-layer",
		"payload":   map[string]any{},
		"items":     []any{},
	}, false)
	if err != nil {
		t.Fatalf("explicit empty composites should count as present: %v", err)
	}
}

func TestValidateRequiredRawBodySkipsBodyAndFormDataParameters(t *testing.T) {
	api := roaAPI()
	api.Parameters = append(api.Parameters,
		meta.Parameter{
			Name: "payload", RawName: "payload", Type: meta.TypeObject,
			Position: meta.PosBody, Required: true, Options: []string{"--payload"},
		},
		meta.Parameter{
			Name: "document", RawName: "document", Type: meta.TypeString,
			Position: meta.PosFormData, Required: true, Options: []string{"--document"},
		},
	)

	// The raw body satisfies the body as a whole, but cannot satisfy a missing
	// path parameter.
	err := ValidateRequired(api, map[string]any{}, true)
	var mre *MissingRequiredError
	if !errors.As(err, &mre) || len(mre.Flags) != 1 || mre.Flags[0] != "--layer-name" {
		t.Fatalf("missing flags = %v, err = %v", mre, err)
	}

	if err := ValidateRequired(api, map[string]any{"layerName": "my-layer"}, true); err != nil {
		t.Fatalf("raw body should bypass required body and formData parameters: %v", err)
	}

	err = ValidateRequired(api, map[string]any{"layerName": "my-layer"}, false)
	if !errors.As(err, &mre) || len(mre.Flags) != 2 || mre.Flags[0] != "--payload" || mre.Flags[1] != "--document" {
		t.Fatalf("without raw body, missing flags = %v, err = %v", mre, err)
	}
}

func TestValidateDocRequiredRecursesCompositeValues(t *testing.T) {
	api := &meta.API{Parameters: []meta.Parameter{
		{
			Name: "config", RawName: "Config", Type: meta.TypeObject, Options: []string{"--config"},
			Fields: []meta.Parameter{
				{Name: "token", RawName: "Token", Type: meta.TypeString, DocRequired: true},
				{
					Name: "groups", RawName: "Groups", Type: meta.TypeArray,
					ItemType: &meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{{
						Name: "name", RawName: "Name", Type: meta.TypeString, DocRequired: true,
					}}},
				},
				{
					Name: "labels", RawName: "Labels", Type: meta.TypeMap,
					ValueType: &meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{{
						Name: "secret", RawName: "Secret", Type: meta.TypeString, DocRequired: true,
					}}},
				},
			},
		},
		{Name: "region_id", RawName: "RegionId", Type: meta.TypeString, DocRequired: true, Options: []string{"--region-id"}},
	}}
	args := map[string]any{
		"Config": map[string]any{
			"Token": nil,
			"Groups": []any{
				map[string]any{"Name": "primary"},
				map[string]any{},
			},
			"Labels": map[string]any{
				"z": map[string]any{"Secret": "set"},
				"a": map[string]any{},
			},
		},
	}

	err := ValidateDocRequired(api, args, false)
	var missing *MissingRequiredError
	if !errors.As(err, &missing) {
		t.Fatalf("error = %v, want MissingRequiredError", err)
	}
	wantFlags := []string{"--config", "--config", "--config", "--region-id"}
	wantPaths := []string{"Config.Token", "Config.Groups[1].Name", `Config.Labels["a"].Secret`, "RegionId"}
	if !reflect.DeepEqual(missing.Flags, wantFlags) || !reflect.DeepEqual(missing.Paths, wantPaths) {
		t.Fatalf("missing = %#v, want flags %v paths %v", missing, wantFlags, wantPaths)
	}
}

func TestValidateDocRequiredOnlyChecksChildrenOfPresentContainers(t *testing.T) {
	api := &meta.API{Parameters: []meta.Parameter{{
		Name: "config", RawName: "Config", Type: meta.TypeObject,
		Fields: []meta.Parameter{{
			Name: "token", RawName: "Token", Type: meta.TypeString, DocRequired: true,
		}},
	}}}
	if err := ValidateDocRequired(api, map[string]any{}, false); err != nil {
		t.Fatalf("absent optional container should not require its children: %v", err)
	}
	if err := ValidateDocRequired(api, map[string]any{"Config": map[string]any{"Token": ""}}, false); err == nil {
		t.Fatal("empty docRequired object field should count as missing")
	}
}

func TestValidateDocRequiredRawBodySkipsBodyAndFormData(t *testing.T) {
	api := &meta.API{Parameters: []meta.Parameter{
		{Name: "body", RawName: "Body", Type: meta.TypeString, Position: meta.PosBody, DocRequired: true},
		{Name: "form", RawName: "Form", Type: meta.TypeString, Position: meta.PosFormData, DocRequired: true},
		{Name: "query", RawName: "Query", Type: meta.TypeString, Position: meta.PosQuery, DocRequired: true},
	}}
	err := ValidateDocRequired(api, map[string]any{}, true)
	var missing *MissingRequiredError
	if !errors.As(err, &missing) ||
		!reflect.DeepEqual(missing.Paths, []string{"Query"}) {
		t.Fatalf("raw-body docRequired result = %#v, error = %v", missing, err)
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
