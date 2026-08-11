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
	"encoding/json"
	"reflect"
	"testing"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"

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

func TestAssemblePathTemplateUsesBodyBoundArgument(t *testing.T) {
	api := &meta.API{
		Name: "CreateTrigger", Version: "2015-12-15", Method: "POST", Style: meta.StyleRESTful,
		URL: "/clusters/{cluster_id}/triggers",
		Parameters: []meta.Parameter{
			{Name: "cluster_id", RawName: "cluster_id", Type: meta.TypeString, Position: meta.PosBody},
			{Name: "project_id", RawName: "project_id", Type: meta.TypeString, Position: meta.PosBody},
		},
	}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{
		"cluster_id": "c5cdf7e3938bc4f8eb0e44b21a80f0000",
		"project_id": "default/test-app",
	}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if got, want := req.Pathname, "/clusters/c5cdf7e3938bc4f8eb0e44b21a80f0000/triggers"; got != want {
		t.Fatalf("Pathname = %q, want %q", got, want)
	}
	wantBody := map[string]any{
		"cluster_id": "c5cdf7e3938bc4f8eb0e44b21a80f0000",
		"project_id": "default/test-app",
	}
	if !reflect.DeepEqual(req.Body, wantBody) {
		t.Fatalf("Body = %#v, want %#v", req.Body, wantBody)
	}
}

func TestSubstitutePathTemplateArgsRequiresExactRawName(t *testing.T) {
	api := &meta.API{Parameters: []meta.Parameter{{
		Name: "cluster_id", RawName: "cluster_id", Type: meta.TypeString, Position: meta.PosBody,
	}}}
	args := map[string]any{"cluster_id": "c-123"}

	for _, template := range []string{"/clusters/{cluster_id}", "/clusters/[cluster_id]"} {
		got, err := substitutePathTemplateArgs(template, api, args)
		if err != nil {
			t.Fatalf("substitutePathTemplateArgs(%q): %v", template, err)
		}
		if want := "/clusters/c-123"; got != want {
			t.Errorf("exact RawName: substitutePathTemplateArgs(%q) = %q, want %q", template, got, want)
		}
	}
	for _, template := range []string{"/clusters/{ClusterId}", "/clusters/{clusterId}"} {
		got, err := substitutePathTemplateArgs(template, api, args)
		if err != nil {
			t.Fatalf("substitutePathTemplateArgs(%q): %v", template, err)
		}
		if got != template {
			t.Errorf("non-RawName placeholder must remain unchanged: got %q, want %q", got, template)
		}
	}
}

func TestSubstitutePathTemplateArgsRejectsCompositeValue(t *testing.T) {
	api := &meta.API{Parameters: []meta.Parameter{{
		Name: "cluster_ids", RawName: "ClusterIds", Type: meta.TypeArray, Position: meta.PosBody,
	}}}
	_, err := substitutePathTemplateArgs(
		"/clusters/{ClusterIds}",
		api,
		map[string]any{"ClusterIds": []any{"c-1", "c-2"}},
	)
	if err == nil {
		t.Fatal("expected composite path value error")
	}
	if want := `path placeholder "ClusterIds" requires a scalar value, got []interface {}`; err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}

func TestExecuteDryRun(t *testing.T) {
	ec := &ExecContext{
		API:    rpcAPI(),
		Region: "cn-beijing",
		DryRun: true,
		Args:   map[string]any{"ImageCacheName": "c1"},
	}
	resp, err := NewExecutor().Execute(ec)
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
	_, err := NewExecutor().Execute(ec)
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

func TestParamStyleJSONPreservesExplicitEmptyArray(t *testing.T) {
	api := &meta.API{
		Name: "CreateHubCluster", Version: "2022-01-01", Method: "POST", Style: meta.StyleRPC, ProductCode: "adcp",
		Parameters: []meta.Parameter{{
			Name: "tag", RawName: "Tag", Type: meta.TypeArray, Position: meta.PosQuery, ParamStyle: "json",
			ItemType: &meta.Parameter{Type: meta.TypeObject},
		}},
	}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"Tag": []any{}}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if got := req.Query["Tag"]; got != `[]` {
		t.Fatalf("Tag query = %q, want []", got)
	}
}

// TestFormDataBody routes formData params to a form body for ROA and RPC
// (plugin SetReqBodyType("formData") + SetContent parity).
func TestFormDataBody(t *testing.T) {
	for _, style := range []meta.APIStyle{meta.StyleRESTful, meta.StyleRPC} {
		api := &meta.API{
			Name: "Op", Version: "v", Method: "POST", Style: style, ProductCode: "p",
			Endpoints:   meta.Endpoints{Global: "p.example.com"},
			ReqBodyType: "formData",
			Parameters:  []meta.Parameter{{Name: "field", RawName: "Field", Type: meta.TypeString, Position: meta.PosFormData}},
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

func TestFormDataJSONStyleSerializesFieldValue(t *testing.T) {
	api := &meta.API{
		Name: "AppendCases", Style: meta.StyleRPC, ReqBodyType: "formData",
		Parameters: []meta.Parameter{{
			Name: "body", RawName: "body", Type: meta.TypeArray, Position: meta.PosFormData, ParamStyle: "json",
			ItemType: &meta.Parameter{Type: meta.TypeObject},
		}},
	}
	cases := []any{map[string]any{"ReferenceId": "01", "PhoneNumber": "1888880000"}}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"body": cases}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	want := map[string]any{"body": `[{"PhoneNumber":"1888880000","ReferenceId":"01"}]`}
	if !reflect.DeepEqual(req.Body, want) {
		t.Fatalf("form body = %#v, want %#v", req.Body, want)
	}
}

func TestFormDataSimpleStyleJoinsScalarArray(t *testing.T) {
	api := &meta.API{
		Name: "SendMessageToGroupUsers", Style: meta.StyleRPC, ReqBodyType: "formData",
		Parameters: []meta.Parameter{{
			Name: "receiver_id_list", RawName: "ReceiverIdList", Type: meta.TypeArray,
			Position: meta.PosFormData, ParamStyle: "simple",
			ItemType: &meta.Parameter{Type: meta.TypeString},
		}},
	}
	req, err := Assemble(&ExecContext{
		API:  api,
		Args: map[string]any{"ReceiverIdList": []any{"user1", "user2"}},
	})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	want := map[string]any{"ReceiverIdList": "user1,user2"}
	if !reflect.DeepEqual(req.Body, want) {
		t.Fatalf("form body = %#v, want %#v", req.Body, want)
	}
}

func TestFormDataFlatAndRepeatListRemainStructured(t *testing.T) {
	for _, paramStyle := range []string{"flat", "repeatList"} {
		t.Run(paramStyle, func(t *testing.T) {
			api := &meta.API{
				Name: "Op", Style: meta.StyleRPC, ReqBodyType: "formData",
				Parameters: []meta.Parameter{{
					Name: "items", RawName: "Items", Type: meta.TypeArray,
					Position: meta.PosFormData, ParamStyle: paramStyle,
				}},
			}
			items := []any{map[string]any{"Key": "value"}}
			req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"Items": items}})
			if err != nil {
				t.Fatalf("Assemble: %v", err)
			}
			want := map[string]any{"Items": items}
			if !reflect.DeepEqual(req.Body, want) {
				t.Fatalf("form body = %#v, want %#v", req.Body, want)
			}
		})
	}
}

func TestFormDataPositionFallsBackWhenOperationMetadataIsMissing(t *testing.T) {
	api := &meta.API{
		Name: "Op", Style: meta.StyleRESTful,
		Parameters: []meta.Parameter{{Name: "field", RawName: "Field", Type: meta.TypeString, Position: meta.PosFormData}},
	}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"Field": "v"}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if req.ReqBodyType != "formData" {
		t.Fatalf("ReqBodyType = %q, want formData", req.ReqBodyType)
	}
	if _, exists := req.Headers["content-type"]; exists {
		t.Fatalf("formData content-type must be left to the SDK: %#v", req.Headers)
	}
}

func TestFormDataWithoutArgumentsStillBuildsLegacyEmptyForm(t *testing.T) {
	api := &meta.API{
		Name: "Op", Style: meta.StyleRESTful,
		Parameters: []meta.Parameter{{Name: "field", RawName: "Field", Type: meta.TypeString, Position: meta.PosFormData}},
	}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if body, ok := req.Body.(map[string]any); !ok || len(body) != 0 {
		t.Fatalf("body = %#v, want empty form map", req.Body)
	}
	if req.ReqBodyType != "formData" || len(req.Headers) != 0 {
		t.Fatalf("form wire metadata = %q, %#v", req.ReqBodyType, req.Headers)
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

func TestDirectMapBodyIsNotWrapped(t *testing.T) {
	api := &meta.API{
		Name: "BindAnalyzer", Version: "v", Method: "POST", Style: meta.StyleRESTful, ProductCode: "p",
		Endpoints: meta.Endpoints{Global: "p.example.com"},
		Parameters: []meta.Parameter{{
			Name: "body", RawName: "body", Type: meta.TypeMap, Position: meta.PosBody,
			Options:   []string{"--biz-body"},
			ValueType: &meta.Parameter{Type: meta.TypeString},
		}},
	}
	want := map[string]any{"name": "kevintest-analyzer"}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"body": want}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if !reflect.DeepEqual(req.Body, want) {
		t.Fatalf("direct map body = %#v, want %#v", req.Body, want)
	}
	if wrapped, ok := req.Body.(map[string]any)["body"]; ok {
		t.Fatalf("direct map body was wrapped as body.body: %#v", wrapped)
	}
}

func TestRawBodyEscapeHatchReplacesSchemaBodyVerbatim(t *testing.T) {
	api := &meta.API{
		Name: "CreateThing", Version: "v", Method: "POST", Style: meta.StyleRESTful, ProductCode: "p",
		Endpoints: meta.Endpoints{Global: "p.example.com"},
		Parameters: []meta.Parameter{{
			Name: "enabled", RawName: "enabled", Type: meta.TypeBoolean, Position: meta.PosBody,
		}},
	}
	raw := `{"enabled":"gateway-specific-value","extra":1}`
	req, err := Assemble(&ExecContext{
		API: api, Args: map[string]any{"enabled": true}, RawBody: raw,
	})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if req.Body != raw {
		t.Fatalf("body = %#v, want raw string %#v", req.Body, raw)
	}
	if req.ReqBodyType != "json" {
		t.Fatalf("ReqBodyType = %q, want json", req.ReqBodyType)
	}
}

func TestRawBodyFormDataJSONIsFormEncoded(t *testing.T) {
	api := &meta.API{
		Name: "SubmitForm", Version: "v", Method: "POST", Style: meta.StyleRESTful, ProductCode: "p",
		Endpoints:   meta.Endpoints{Global: "p.example.com"},
		ReqBodyType: "formData",
		Parameters: []meta.Parameter{
			{Name: "a", RawName: "a", Type: meta.TypeString, Position: meta.PosFormData},
			{Name: "b", RawName: "b", Type: meta.TypeString, Position: meta.PosFormData},
		},
	}
	req, err := Assemble(&ExecContext{
		API: api, Args: map[string]any{}, RawBody: `{"a":"1","b":"two words"}`,
	})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if req.ReqBodyType != "formData" {
		t.Fatalf("ReqBodyType = %q, want formData", req.ReqBodyType)
	}
	body, ok := req.Body.(map[string]any)
	if !ok || body["a"] != "1" || body["b"] != "two words" {
		t.Fatalf("form body = %#v", req.Body)
	}
	encoded := openapiutil.ToForm(body)
	if encoded == nil || *encoded != "a=1&b=two+words" {
		t.Fatalf("encoded form = %#v", encoded)
	}
}

func TestRawBodyFormDataRejectsNonObjectJSON(t *testing.T) {
	api := &meta.API{
		Name: "SubmitForm", Style: meta.StyleRESTful, ReqBodyType: "formData",
		Parameters: []meta.Parameter{{Name: "a", RawName: "a", Type: meta.TypeString, Position: meta.PosFormData}},
	}
	for _, raw := range []string{`["a"]`, `{"a":"1"} trailing`, `null`} {
		if _, err := Assemble(&ExecContext{API: api, RawBody: raw}); err == nil {
			t.Fatalf("RawBody %q: expected formData JSON object error", raw)
		}
	}
}

func TestNonFormBodyMetadataKeepsLegacyJSONExecution(t *testing.T) {
	api := &meta.API{
		Name: "Upload", Style: meta.StyleRESTful,
		ReqBodyType: "byte", ContentType: "application/vnd.example.payload",
		Parameters: []meta.Parameter{{Name: "body", RawName: "body", Type: meta.TypeString, Position: meta.PosBody}},
	}
	req, err := Assemble(&ExecContext{API: api, RawBody: "payload"})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if req.ReqBodyType != "json" {
		t.Fatalf("ReqBodyType = %q, want json", req.ReqBodyType)
	}
	if _, exists := req.Headers["content-type"]; exists {
		t.Fatalf("non-form metadata must not override content-type: %#v", req.Headers)
	}
	if req.Body != "payload" {
		t.Fatalf("body = %#v, want raw string", req.Body)
	}
}

func TestMixedBodyAndFormDataUsesOneFormBody(t *testing.T) {
	api := &meta.API{
		Name: "Submit", Style: meta.StyleRESTful, ReqBodyType: "formData",
		Parameters: []meta.Parameter{
			{Name: "form", RawName: "Form", Type: meta.TypeString, Position: meta.PosFormData},
			{Name: "payload", RawName: "Payload", Type: meta.TypeString, Position: meta.PosBody},
		},
	}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"Form": "a", "Payload": "b"}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	want := map[string]any{"Form": "a", "Payload": "b"}
	if !reflect.DeepEqual(req.Body, want) {
		t.Fatalf("form body = %#v, want %#v", req.Body, want)
	}
	if req.ReqBodyType != "formData" {
		t.Fatalf("ReqBodyType = %q, want formData", req.ReqBodyType)
	}
	if _, exists := req.Headers["content-type"]; exists {
		t.Fatalf("formData content-type must be left to the SDK: %#v", req.Headers)
	}
}

func TestSetOpenAPIRequestBodyMatchesLegacyTypeBridge(t *testing.T) {
	oaReq := &openapiutil.OpenApiRequest{}
	setOpenAPIRequestBody(oaReq, `{"a":1}`)
	if got, ok := oaReq.Body.([]byte); !ok || string(got) != `{"a":1}` {
		t.Fatalf("string body bridge = %#v", oaReq.Body)
	}

	wantMap := map[string]any{"a": "1"}
	setOpenAPIRequestBody(oaReq, wantMap)
	if !reflect.DeepEqual(oaReq.Body, wantMap) {
		t.Fatalf("map body bridge = %#v", oaReq.Body)
	}
}

func TestAnyNullWireBehavior(t *testing.T) {
	api := &meta.API{
		Name: "UpdateThing", Version: "v", Method: "POST", Style: meta.StyleRESTful, ProductCode: "p",
		ReqBodyType: "formData",
		URL:         "/things/{id}",
		Endpoints:   meta.Endpoints{Global: "p.example.com"},
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
	if len(req.Query) != 0 {
		t.Fatalf("null query must be omitted: %#v", req.Query)
	}
	if len(req.Headers) != 0 {
		t.Fatalf("formData content-type must be left to the SDK: %#v", req.Headers)
	}
	if req.Pathname != "/things/%7Bid%7D" {
		t.Fatalf("null path value must not become <nil>: %q", req.Pathname)
	}
	body, ok := req.Body.(map[string]any)
	if !ok {
		t.Fatalf("body = %#v, want map", req.Body)
	}
	if value, exists := body["Payload"]; !exists || value != nil {
		t.Fatalf("JSON body must preserve explicit null: %#v", body)
	}
	if req.ReqBodyType != "formData" {
		t.Fatalf("ReqBodyType = %q, want formData", req.ReqBodyType)
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

func TestAssembleWildcardPathUsesCompleteParameterValue(t *testing.T) {
	api := &meta.API{
		Name: "GetResources", Version: "2022-08-30", Method: "GET", Style: meta.StyleRESTful,
		URL: "/api/v1/providers/{provider}/products/{product}/resources/*", HasWildcardPath: true,
		Parameters: []meta.Parameter{{
			Name: "request_path", RawName: "requestPath", Type: meta.TypeString,
			Position: meta.PosPath, Required: true, IsWildcard: true,
		}},
	}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{
		"requestPath": "/api/v1/providers/qqq/products/dd/resources/dddd:4",
	}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if got, want := req.Pathname, "/api/v1/providers/qqq/products/dd/resources/dddd%3A4"; got != want {
		t.Fatalf("Pathname = %q, want %q", got, want)
	}
}
