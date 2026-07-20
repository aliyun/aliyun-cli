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
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func rpcAPI() *meta.API {
	return &meta.API{
		Name:        "CreateImageCache",
		Version:     "2024-04-02",
		Method:      "POST",
		Style:       meta.StyleRPC,
		Protocol:    "HTTPS",
		ProductCode: "acc",
		Endpoints: meta.Endpoints{
			Global: "acc.cn-hangzhou.aliyuncs.com",
			Public: map[string]string{"cn-beijing": "acc.cn-beijing.aliyuncs.com"},
		},
		Parameters: []meta.Parameter{
			{Name: "region_id", RawName: "RegionId", Type: meta.TypeString, Position: meta.PosQuery},
			{Name: "image_cache_name", RawName: "ImageCacheName", Type: meta.TypeString, Position: meta.PosQuery},
			{
				Name: "images", RawName: "Images", Type: meta.TypeArray, Position: meta.PosQuery,
				ItemType: &meta.Parameter{Type: meta.TypeString},
			},
			{
				Name: "tags", RawName: "Tags", Type: meta.TypeArray, Position: meta.PosQuery,
				ItemType: &meta.Parameter{Type: meta.TypeObject},
			},
		},
	}
}

func TestAssembleRPCScalarAndArray(t *testing.T) {
	ec := &ExecContext{
		API:    rpcAPI(),
		Region: "cn-beijing",
		Args: map[string]any{
			"RegionId":       "cn-beijing",
			"ImageCacheName": "cache1",
			"Images":         []any{"img1"},
		},
	}
	req, err := Assemble(ec)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if req.Action != "CreateImageCache" || req.Version != "2024-04-02" {
		t.Fatalf("identity wrong: %+v", req)
	}
	if req.Method != "POST" || req.Protocol != "HTTPS" || req.Style != "RPC" {
		t.Fatalf("wire wrong: %+v", req)
	}
	// Endpoint resolved from region.
	if req.Endpoint != "acc.cn-beijing.aliyuncs.com" {
		t.Fatalf("endpoint = %q", req.Endpoint)
	}
	want := map[string]string{
		"RegionId":       "cn-beijing",
		"ImageCacheName": "cache1",
		"Images.1":       "img1",
	}
	if !reflect.DeepEqual(req.Query, want) {
		t.Fatalf("query = %#v, want %#v", req.Query, want)
	}
}

func TestAssembleRPCArrayOfObjectFlattening(t *testing.T) {
	ec := &ExecContext{
		API:    rpcAPI(),
		Region: "cn-beijing",
		Args: map[string]any{
			"Tags": []any{
				map[string]any{"Key": "env", "Value": "prod"},
				map[string]any{"Key": "team", "Value": "infra"},
			},
		},
	}
	req, err := Assemble(ec)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	want := map[string]string{
		"Tags.1.Key":   "env",
		"Tags.1.Value": "prod",
		"Tags.2.Key":   "team",
		"Tags.2.Value": "infra",
	}
	if !reflect.DeepEqual(req.Query, want) {
		t.Fatalf("query = %#v, want %#v", req.Query, want)
	}
}

func TestAssembleEndpointOverride(t *testing.T) {
	ec := &ExecContext{
		API:      rpcAPI(),
		Endpoint: "acc.custom.aliyuncs.com",
		Args:     map[string]any{"RegionId": "cn-x"},
	}
	req, err := Assemble(ec)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if req.Endpoint != "acc.custom.aliyuncs.com" {
		t.Fatalf("endpoint override ignored: %q", req.Endpoint)
	}
}

func TestExecuteDryRun(t *testing.T) {
	ec := &ExecContext{
		API:    rpcAPI(),
		Region: "cn-beijing",
		DryRun: true,
		Args:   map[string]any{"ImageCacheName": "c1"},
	}
	resp, err := NewExecutor().Execute(context.Background(), ec)
	if err != nil {
		t.Fatalf("Execute dry-run: %v", err)
	}
	if resp.Assembled == nil {
		t.Fatal("dry-run did not attach AssembledRequest")
	}
	if resp.Assembled.Query["ImageCacheName"] != "c1" {
		t.Fatalf("assembled query = %#v", resp.Assembled.Query)
	}
	if resp.Assembled.Action != "CreateImageCache" {
		t.Fatalf("assembled action = %q", resp.Assembled.Action)
	}
	// Dry-run must not perform a network send; rendering is the
	// caller's job, so Raw stays empty here.
	if len(resp.Raw) != 0 {
		t.Fatalf("dry-run should not pre-render Raw, got %q", resp.Raw)
	}
}

func TestExecuteNoCredentialFails(t *testing.T) {
	ec := &ExecContext{
		API:    rpcAPI(),
		Region: "cn-beijing",
		Args:   map[string]any{"ImageCacheName": "c1"},
	}
	_, err := NewExecutor().Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error without credential")
	}
}

// TestSerializeRPCNumericPrecision guards that json.Number scalars are
// serialized without float64 rounding.
func TestSerializeRPCNumericPrecision(t *testing.T) {
	got := serializeRPC("BigId", json.Number("9007199254740993"))
	if got["BigId"] != "9007199254740993" {
		t.Fatalf("precision lost: %q", got["BigId"])
	}
}

// TestParamStyleJSONAndSimple checks the two non-default query styles.
func TestParamStyleJSONAndSimple(t *testing.T) {
	api := &meta.API{
		Name: "Op", Version: "v", Method: "GET", Style: meta.StyleRESTful, ProductCode: "p",
		Endpoints: meta.Endpoints{Global: "p.example.com"},
		Parameters: []meta.Parameter{
			{Name: "tags", RawName: "Tags", Type: meta.TypeArray, Position: meta.PosQuery, ParamStyle: "json",
				ItemType: &meta.Parameter{Type: meta.TypeObject}},
			{Name: "ids", RawName: "Ids", Type: meta.TypeArray, Position: meta.PosQuery, ParamStyle: "simple",
				ItemType: &meta.Parameter{Type: meta.TypeString}},
		},
	}
	ec := &ExecContext{API: api, Args: map[string]any{
		"Tags": []any{map[string]any{"Key": "k"}},
		"Ids":  []any{"a", "b", "c"},
	}}
	req, err := Assemble(ec)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if req.Query["Tags"] != `[{"Key":"k"}]` {
		t.Errorf("json style = %q", req.Query["Tags"])
	}
	if req.Query["Ids"] != "a,b,c" {
		t.Errorf("simple style = %q", req.Query["Ids"])
	}
}

// TestFormDataBody routes formData params to a form body for ROA and RPC
// (plugin SetReqBodyType("formData") + SetContent parity).
func TestFormDataBody(t *testing.T) {
	for _, style := range []meta.APIStyle{meta.StyleRESTful, meta.StyleRPC} {
		api := &meta.API{
			Name: "Op", Version: "v", Method: "POST", Style: style, ProductCode: "p",
			Endpoints:  meta.Endpoints{Global: "p.example.com"},
			Parameters: []meta.Parameter{{Name: "field", RawName: "Field", Type: meta.TypeString, Position: meta.PosFormData}},
		}
		req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"Field": "v"}})
		if err != nil {
			t.Fatalf("style %s Assemble: %v", style, err)
		}
		if req.ReqBodyType != "formData" {
			t.Fatalf("style %s ReqBodyType = %q, want formData", style, req.ReqBodyType)
		}
		body, _ := req.Body.(map[string]any)
		if body["Field"] != "v" {
			t.Fatalf("style %s form body = %#v", style, req.Body)
		}
		if _, ok := req.Query["Field"]; ok {
			t.Fatalf("style %s: form param must not be folded into Query", style)
		}
	}
}

// TestJSONBody routes body params to a JSON body for ROA and RPC
// (plugin SetContent parity; default ReqBodyType=json).
func TestJSONBody(t *testing.T) {
	for _, style := range []meta.APIStyle{meta.StyleRESTful, meta.StyleRPC} {
		api := &meta.API{
			Name: "Op", Version: "v", Method: "POST", Style: style, ProductCode: "p",
			Endpoints:  meta.Endpoints{Global: "p.example.com"},
			Parameters: []meta.Parameter{{Name: "query", RawName: "query", Type: meta.TypeString, Position: meta.PosBody}},
		}
		req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"query": "hello"}})
		if err != nil {
			t.Fatalf("style %s Assemble: %v", style, err)
		}
		if req.ReqBodyType != "json" {
			t.Fatalf("style %s ReqBodyType = %q, want json", style, req.ReqBodyType)
		}
		body, _ := req.Body.(map[string]any)
		if body["query"] != "hello" {
			t.Fatalf("style %s json body = %#v", style, req.Body)
		}
		if _, ok := req.Query["query"]; ok {
			t.Fatalf("style %s: body param must not be folded into Query", style)
		}
	}
}

func TestDirectAnyBodyIsNotWrapped(t *testing.T) {
	api := &meta.API{
		Name: "UpdateThing", Version: "v", Method: "POST", Style: meta.StyleRESTful, ProductCode: "p",
		Endpoints: meta.Endpoints{Global: "p.example.com"},
		Parameters: []meta.Parameter{{
			Name: "body", RawName: "body", Type: meta.TypeAny, Position: meta.PosBody,
			Options: []string{"--biz-body"},
		}},
	}
	want := map[string]any{
		"name":    "demo",
		"count":   json.Number("9007199254740993"),
		"enabled": true,
	}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"body": want}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if !reflect.DeepEqual(req.Body, want) {
		t.Fatalf("direct body = %#v, want %#v", req.Body, want)
	}
	if req.ReqBodyType != "json" {
		t.Fatalf("ReqBodyType = %q, want json", req.ReqBodyType)
	}
	if wrapped, ok := req.Body.(map[string]any)["body"]; ok {
		t.Fatalf("direct body was wrapped as body.body: %#v", wrapped)
	}
}

func TestAnyNullWireBehavior(t *testing.T) {
	api := &meta.API{
		Name: "UpdateThing", Version: "v", Method: "POST", Style: meta.StyleRESTful, ProductCode: "p",
		URL:       "/things/{id}",
		Endpoints: meta.Endpoints{Global: "p.example.com"},
		Parameters: []meta.Parameter{
			{Name: "query", RawName: "Query", Type: meta.TypeAny, Position: meta.PosQuery},
			{Name: "json_query", RawName: "JsonQuery", Type: meta.TypeAny, Position: meta.PosQuery, ParamStyle: "json"},
			{Name: "header", RawName: "X-Value", Type: meta.TypeAny, Position: meta.PosHeader},
			{Name: "id", RawName: "id", Type: meta.TypeAny, Position: meta.PosPath},
			{Name: "form", RawName: "Form", Type: meta.TypeAny, Position: meta.PosFormData},
			{Name: "payload", RawName: "Payload", Type: meta.TypeAny, Position: meta.PosBody},
		},
	}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{
		"Query": nil, "JsonQuery": nil, "X-Value": nil, "id": nil, "Form": nil, "Payload": nil,
	}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(req.Query) != 0 || len(req.Headers) != 0 {
		t.Fatalf("null query/header must be omitted: query=%#v headers=%#v", req.Query, req.Headers)
	}
	if req.Pathname != "/things/{id}" {
		t.Fatalf("null path value must not become <nil>: %q", req.Pathname)
	}
	body, ok := req.Body.(map[string]any)
	if !ok {
		t.Fatalf("body = %#v, want map", req.Body)
	}
	if value, exists := body["Payload"]; !exists || value != nil {
		t.Fatalf("JSON body must preserve explicit null: %#v", body)
	}
	if req.ReqBodyType != "json" {
		t.Fatalf("ReqBodyType = %q, want json", req.ReqBodyType)
	}
}

func TestDirectAnyNullProducesJSONNullBody(t *testing.T) {
	api := &meta.API{
		Name: "UpdateThing", Version: "v", Method: "POST", Style: meta.StyleRESTful, ProductCode: "p",
		Endpoints:  meta.Endpoints{Global: "p.example.com"},
		Parameters: []meta.Parameter{{Name: "body", RawName: "body", Type: meta.TypeAny, Position: meta.PosBody}},
	}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"body": nil}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	encoded, err := json.Marshal(req.Body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	if string(encoded) != "null" {
		t.Fatalf("direct null body = %s, want null", encoded)
	}
}
