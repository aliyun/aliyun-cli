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

// Package runtime is the L5 execution layer of aliyun-openapi-runtime. It
// turns a resolved (meta.API, argument values) tuple into a real
// HTTP request against Alibaba Cloud and materialises the response.
//
// The layer is split into two concerns:
//
//   - Assemble: pure, side-effect-free translation of meta.API +
//     args into an AssembledRequest (wire params, no network). This is
//     what --dry-run surfaces and what the unit tests pin down.
//   - Execute:  Assemble + credential/endpoint resolution + the actual
//     darabonba-openapi CallApi, decoding the response with UseNumber
//     to preserve int64 / large-number precision.
package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
	credentialsv2 "github.com/aliyun/credentials-go/credentials"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

// Executor runs one API call end-to-end.
type Executor interface {
	Execute(ec *ExecContext) (*Response, error)
}

// SSEExecutor is an optional capability implemented by executors that can
// consume Server-Sent Events without buffering the HTTP response. Each event
// is delivered as one normalized JSON object.
type SSEExecutor interface {
	ExecuteSSE(ec *ExecContext, yield func(SSEEvent)) error
}

// ExecContext bundles everything the Executor needs to make a call.
// It is populated by the L6 Builder just before invocation.
type ExecContext struct {
	API     *meta.API
	Product *meta.Product

	// Args is the parsed argument set keyed by parameter snake_case
	// Name. Values are produced by the argparser: string / json.Number
	// / bool for scalars, []any for arrays, map[string]any for
	// objects and maps.
	Args map[string]any

	// Wire-time knobs.
	Region         string
	Endpoint       string // if non-empty, overrides Endpoints.Resolve
	Version        string // if non-empty, overrides API.Version
	UseVPC         bool
	ReadTimeout    time.Duration
	ConnectTimeout time.Duration
	RetryCount     int
	Credential     credentialsv2.Credential

	// ExtraQuery is merged into the assembled request's Query after
	// schema args are serialized. Used by --pager to inject NextToken /
	// PageNumber on subsequent pages without mutating Args.
	ExtraQuery map[string]string

	// ExtraHeaders are merged into the assembled request Headers after
	// schema params (from --header Name=Value).
	ExtraHeaders map[string]string

	// RawBody, when set, replaces the schema-derived body (from --body
	// / --body-file). JSON text stays raw; only formData needs to decode a
	// JSON object before the SDK performs form serialization. No metadata
	// field conversion is applied.
	RawBody any

	// ForceHTTPS / ForceHTTP override AssembledRequest.Protocol
	// (--secure / --insecure).
	ForceHTTPS bool
	ForceHTTP  bool

	// SkipSecureVerify / CLIVersion / UserAgent come from Host.Settings.
	SkipSecureVerify bool
	CLIVersion       string
	UserAgent        string // host suffixes only; composed with base in send

	// DryRun assembles the request and returns it without sending.
	DryRun bool
}

// AssembledRequest is the fully-resolved wire request, independent of
// any HTTP client. It is returned verbatim on --dry-run and is the
// unit-testable output of Assemble.
type AssembledRequest struct {
	Action   string            `json:"action"`
	Version  string            `json:"version"`
	Method   string            `json:"method"`
	Protocol string            `json:"protocol"`
	Style    string            `json:"style"`
	Pathname string            `json:"pathname,omitempty"`
	Query    map[string]string `json:"query,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     any               `json:"body,omitempty"`
	// ReqBodyType is the request body encoding: "json" (default) or
	// "formData". Empty and all non-formData metadata values are treated as
	// "json" to match aliyun-cli-runtime.
	ReqBodyType string `json:"req_body_type,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	Region      string `json:"region,omitempty"`
}

// Response is what Execute returns.
type Response struct {
	StatusCode int
	Headers    map[string][]string
	Raw        []byte
	// Parsed carries the response decoded with UseNumber() so int64
	// IDs survive as json.Number rather than being routed through
	// float64. Populated on structured output.
	Parsed any
	// Assembled is set on dry-run so callers can inspect / print the
	// request that would have been sent.
	Assembled *AssembledRequest
}

// ErrUnimplemented is retained for callers that still probe for it
// during the transition. The DefaultExecutor no longer returns it.
var ErrUnimplemented = errors.New("runtime: executor not yet implemented")

// DefaultExecutor is the production Executor. It handles RPC and,
// minimally, RESTful (ROA) styles.
type DefaultExecutor struct{}

// NewExecutor returns the default executor. It is stateless and safe
// to reuse across goroutines.
func NewExecutor() *DefaultExecutor { return &DefaultExecutor{} }

// Execute assembles the request and, unless DryRun, sends it. On a
// dry run it returns the AssembledRequest untouched; how it is
// rendered (human dump vs one-line meta JSON) is the caller's
// concern.
func (e *DefaultExecutor) Execute(ec *ExecContext) (*Response, error) {
	req, err := Assemble(ec)
	if err != nil {
		return nil, err
	}
	if ec.DryRun {
		return &Response{StatusCode: 0, Assembled: req}, nil
	}
	return send(ec, req)
}

func (e *DefaultExecutor) ExecuteSSE(ec *ExecContext, yield func(SSEEvent)) error {
	req, err := Assemble(ec)
	if err != nil {
		return err
	}
	if ec.DryRun {
		return errors.New("runtime: SSE execution cannot run in dry-run mode")
	}
	return sendSSE(ec, req, yield)
}

// Assemble is the pure translation from meta + args to wire request.
func Assemble(ec *ExecContext) (*AssembledRequest, error) {
	if ec == nil || ec.API == nil {
		return nil, errors.New("runtime: nil API in ExecContext")
	}
	api := ec.API
	style := string(api.Style)
	if style == "" {
		style = string(meta.StyleRPC)
	}

	req := &AssembledRequest{
		Action:   api.Name,
		Version:  api.Version,
		Method:   defaultMethod(api.Method, style),
		Protocol: defaultProtocol(api.Protocol),
		Style:    style,
		Pathname: api.URL,
		Query:    map[string]string{},
		Headers:  map[string]string{},
		Region:   ec.Region,
		// aliyun-cli-runtime defaults every operation to json and only generated
		// formData APIs call SetReqBodyType("formData").
		ReqBodyType: resolveReqBodyType(api),
	}
	if v := ec.Version; v != "" {
		req.Version = v
	}

	req.Endpoint = ec.Endpoint
	if req.Endpoint == "" {
		req.Endpoint = api.Endpoints.Resolve(ec.Region, ec.UseVPC)
	}

	isRPC := strings.EqualFold(style, string(meta.StyleRPC))
	var bodyParts map[string]any // JSON body (RPC and ROA; plugin SetContent parity)
	var formParts map[string]any // formData body (RPC and ROA; plugin SetReqBodyType parity)
	var directBody any           // a schema parameter named "body" is the whole JSON body
	var directBodySet bool
	var wildcardPath string
	var wildcardPathSet bool

	for name, val := range ec.Args {
		p := api.FindParameter(name)
		if p == nil {
			// Unknown args should have been rejected by the parser;
			// tolerate by treating as a raw query key.
			req.Query[name] = fmt.Sprintf("%v", val)
			continue
		}
		// Args are keyed by RawName; wire name must come from metadata.
		if p.RawName == "" {
			return nil, fmt.Errorf("parameter %q is missing raw_name in metadata", p.Name)
		}
		wire := p.RawName
		// JSON null has a meaningful representation only in a JSON body.
		// Query/header/path/formData encodings have no portable null token;
		// omitting it also matches the generated Go plugin's nil guard.
		if val == nil && p.Position != meta.PosBody {
			continue
		}

		switch p.Position {
		case meta.PosPath:
			if api.HasWildcardPath && p.IsWildcard {
				wildcardPath = scalarString(val)
				wildcardPathSet = true
			}

		case meta.PosHeader:
			req.Headers[wire] = scalarString(val)

		case meta.PosFormData:
			// Form parameters go into the request body for RPC and ROA. Serialization is selected only by operation.req_body_type.
			if formParts == nil {
				formParts = map[string]any{}
			}
			formParts[wire] = val

		case meta.PosBody:
			if isDirectBodyParameter(p) {
				// A top-level parameter whose schema identity is "body" is the complete request body, not a property named "body".
				// This is how generator direct-body APIs (notably type=any) are defined.
				directBody = val
				directBodySet = true
			} else {
				// Match generated plugins: named body params always go into the
				// JSON request body (default ReqBodyType=json), for RPC and ROA.
				if bodyParts == nil {
					bodyParts = map[string]any{}
				}
				bodyParts[wire] = val
			}

		default: // query / host
			kv, err := serializeQuery(wire, val, isRPC, p.ParamStyle)
			if err != nil {
				return nil, err
			}
			mergeFlat(req.Query, kv)
		}
	}

	if wildcardPathSet {
		req.Pathname = wildcardPath
	} else if req.Pathname != "" {
		// Match generated plugins: a RawName in the URL may also be body-bound.
		var err error
		req.Pathname, err = substitutePathTemplateArgs(req.Pathname, api, ec.Args)
		if err != nil {
			return nil, err
		}
	}
	if req.Pathname != "" {
		// Keep this after placeholder substitution to match generated Go plugins.
		// encodePath encodes only the pathname and preserves an inline query suffix.
		// Do not pre-escape individual path parameters while this final encoding is in place: an escaped value such as "a%2Fb" would be encoded again as "a%252Fb".
		req.Pathname = encodePath(req.Pathname)
	}

	switch {
	case directBodySet:
		if directBody == nil {
			// Keep an explicit Any `null` distinct from an absent body. A
			// RawMessage marshals and is sent as the JSON literal null.
			req.Body = json.RawMessage("null")
		} else {
			req.Body = directBody
		}
	case formParts != nil:
		for key, value := range bodyParts {
			formParts[key] = value
		}
		req.Body = formParts
	case bodyParts != nil:
		req.Body = bodyParts
	}

	for k, v := range ec.ExtraQuery {
		req.Query[k] = v
	}
	for k, v := range ec.ExtraHeaders {
		req.Headers[k] = v
	}
	if ec.RawBody != nil {
		body, err := normalizeRawBody(ec.RawBody, req.ReqBodyType)
		if err != nil {
			return nil, err
		}
		req.Body = body
	}
	if req.ReqBodyType == "formData" {
		if req.Body == nil {
			// Generated Go plugins call SetContent with the form builder's empty map even when the user supplied no optional form arguments.
			req.Body = map[string]any{}
		}
		req.Headers["content-type"] = "application/x-www-form-urlencoded"
	}
	switch {
	case ec.ForceHTTP:
		req.Protocol = "HTTP"
	case ec.ForceHTTPS:
		req.Protocol = "HTTPS"
	}
	return req, nil
}

// substitutePathTemplateArgs replaces {RawName} and [RawName] placeholders from parsed API arguments, regardless of parameter position.
// Composite values are rejected because path placeholders must resolve to scalars.
func substitutePathTemplateArgs(path string, api *meta.API, args map[string]any) (string, error) {
	if path == "" || api == nil || len(args) == 0 {
		return path, nil
	}

	for i := range api.Parameters {
		rawName := api.Parameters[i].RawName
		if rawName == "" {
			continue
		}
		bracePlaceholder := "{" + rawName + "}"
		bracketPlaceholder := "[" + rawName + "]"
		if !strings.Contains(path, bracePlaceholder) && !strings.Contains(path, bracketPlaceholder) {
			continue
		}
		value, ok := args[rawName]
		if !ok {
			continue
		}
		formatted, ok := scalarPathTemplateValue(value)
		if !ok {
			if value == nil {
				continue
			}
			return "", fmt.Errorf("path placeholder %q requires a scalar value, got %T", rawName, value)
		}
		path = strings.ReplaceAll(path, bracePlaceholder, formatted)
		path = strings.ReplaceAll(path, bracketPlaceholder, formatted)
	}
	return path, nil
}

func scalarPathTemplateValue(value any) (string, bool) {
	switch value.(type) {
	case nil, []any, map[string]any:
		return "", false
	default:
		return scalarString(value), true
	}
}

// resolveReqBodyType matches aliyun-cli-runtime: json is the operation default, and formData is the only metadata value that changes the SDK serializer.
// Until every metadata artifact exports req_body_type, formData parameters are used as a compatibility fallback only when the operation field is absent.
func resolveReqBodyType(api *meta.API) string {
	if api == nil {
		return "json"
	}
	if value := strings.TrimSpace(api.ReqBodyType); value != "" {
		if strings.EqualFold(value, "formData") {
			return "formData"
		}
		return "json"
	}
	for i := range api.Parameters {
		if api.Parameters[i].Position == meta.PosFormData {
			return "formData"
		}
	}
	return "json"
}

// normalizeRawBody preserves --body as a schema escape hatch while adapting its representation to the serializer selected for the API.
// In particular, darabonba's formData path expects a map, not a Go string. The legacy runtime accepts a JSON object for this purpose and then lets the SDK form-encode it.
func normalizeRawBody(raw any, reqBodyType string) (any, error) {
	switch strings.ToLower(strings.TrimSpace(reqBodyType)) {
	case "formdata":
		switch body := raw.(type) {
		case map[string]any:
			return body, nil
		case string:
			return decodeRawFormBody([]byte(body))
		case []byte:
			return decodeRawFormBody(body)
		case json.RawMessage:
			return decodeRawFormBody([]byte(body))
		default:
			return nil, fmt.Errorf("raw body for formData must be a JSON object, got %T", raw)
		}
	default: // json (and the SDK's default)
		return raw, nil
	}
}

func decodeRawFormBody(data []byte) (map[string]any, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var body map[string]any
	if err := dec.Decode(&body); err != nil {
		return nil, fmt.Errorf("raw body for formData must be a JSON object: %w", err)
	}
	if body == nil {
		return nil, errors.New("raw body for formData must be a JSON object, got null")
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("raw body for formData must contain exactly one JSON object")
	}
	return body, nil
}

func setOpenAPIRequestBody(req *openapiutil.OpenApiRequest, body any) {
	if req == nil {
		return
	}
	switch value := body.(type) {
	case string:
		req.SetBody([]byte(value))
	case []byte:
		req.SetBody(value)
	case map[string]any:
		req.Body = value
	case []any:
		req.Body = value
	case []map[string]any:
		req.Body = value
	default:
		req.Body = value
	}
}

func send(ec *ExecContext, req *AssembledRequest) (*Response, error) {
	call, err := prepareCall(ec, req)
	if err != nil {
		return nil, err
	}
	LogRequest(req)
	raw, err := call.client.CallApi(call.params, call.request, call.runtime)
	if err != nil {
		return nil, fmt.Errorf("runtime: call failed: %w", dara.TeaSDKError(err))
	}

	resp := buildResponse(raw)
	LogResponse(resp)
	return resp, nil
}

type preparedCall struct {
	client  *openapiClient.Client
	params  *openapiClient.Params
	request *openapiutil.OpenApiRequest
	runtime *dara.RuntimeOptions
}

func prepareCall(ec *ExecContext, req *AssembledRequest) (*preparedCall, error) {
	if ec.Credential == nil {
		return nil, errors.New("runtime: no credential resolved; run `aliyun configure` or pass --dry-run")
	}
	if req.Endpoint == "" {
		return nil, fmt.Errorf("runtime: endpoint not resolved for product %q region %q; pass --endpoint", ec.API.ProductCode, ec.Region)
	}

	conf := &openapiClient.Config{Credential: ec.Credential}
	if ec.Region != "" {
		conf.RegionId = tea.String(ec.Region)
	}
	conf.Endpoint = tea.String(stripScheme(req.Endpoint))
	// Always stamp engine + CLI version (plugin-parity BuildBaseUserAgent),
	// then append host suffixes (--user-agent / AI mode).
	conf.UserAgent = tea.String(ComposeUserAgent(ec.CLIVersion, ec.UserAgent))
	if ec.ReadTimeout > 0 {
		conf.ReadTimeout = tea.Int(int(ec.ReadTimeout / time.Millisecond))
	}
	if ec.ConnectTimeout > 0 {
		conf.ConnectTimeout = tea.Int(int(ec.ConnectTimeout / time.Millisecond))
	}
	if ec.RetryCount > 0 {
		conf.RetryOptions = &dara.RetryOptions{
			Retryable: true,
			RetryCondition: []*dara.RetryCondition{
				{MaxAttempts: ec.RetryCount},
			},
		}
	}

	client, err := openapiClient.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("runtime: create client: %w", err)
	}
	client.DisableSDKError = tea.Bool(true)

	params := &openapiClient.Params{
		Action:      tea.String(req.Action),
		Version:     tea.String(req.Version),
		Protocol:    tea.String(req.Protocol),
		Method:      tea.String(req.Method),
		AuthType:    tea.String("AK"),
		Style:       tea.String(darabonbaStyle(req.Style)),
		Pathname:    tea.String(orSlash(req.Pathname)),
		ReqBodyType: tea.String(reqBodyType(req.ReqBodyType)),
		// Request the body as a raw string so WE decode it with
		// UseNumber(); letting the SDK parse to map[string]interface{}
		// would route large integers through float64 and lose
		// precision before we ever see them.
		BodyType: tea.String("string"),
	}

	oaReq := &openapiutil.OpenApiRequest{
		Query:   toStrPtrMap(req.Query),
		Headers: toStrPtrMap(req.Headers),
		HostMap: map[string]*string{},
	}
	if req.Body != nil {
		setOpenAPIRequestBody(oaReq, req.Body)
	}

	runtimeOpts := &dara.RuntimeOptions{}
	if ec.SkipSecureVerify {
		runtimeOpts.IgnoreSSL = tea.Bool(true)
	}
	return &preparedCall{
		client:  client,
		params:  params,
		request: oaReq,
		runtime: runtimeOpts,
	}, nil
}

// buildResponse extracts status/headers/body from darabonba's response
// map and decodes the body with UseNumber so large integers keep full
// precision in Parsed. The body is requested as a raw string (see
// params.BodyType in send).
func buildResponse(raw map[string]any) *Response {
	resp := &Response{Headers: map[string][]string{}}
	resp.StatusCode = asInt(raw["statusCode"])
	if hs, ok := raw["headers"].(map[string]*string); ok {
		for k, v := range hs {
			if v != nil {
				resp.Headers[k] = []string{*v}
			}
		}
	}

	// Normalise the body to raw bytes regardless of the SDK's
	// representation (string / []byte / already-parsed value).
	switch b := raw["body"].(type) {
	case string:
		resp.Raw = []byte(b)
	case []byte:
		resp.Raw = b
	case nil:
		// no body
	default:
		if m, err := json.Marshal(b); err == nil {
			resp.Raw = m
		}
	}

	if len(resp.Raw) > 0 {
		dec := json.NewDecoder(bytes.NewReader(resp.Raw))
		dec.UseNumber()
		var parsed any
		if dec.Decode(&parsed) == nil {
			resp.Parsed = parsed
		}
	}
	return resp
}

// asInt coerces the SDK's statusCode (int / *int / json.Number /
// float64) into an int.
func asInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case *int:
		if t != nil {
			return *t
		}
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return int(n)
		}
	}
	return 0
}

// ============================================================================
// small helpers
// ============================================================================

func defaultMethod(m, style string) string {
	if m != "" {
		return strings.ToUpper(m)
	}
	if strings.EqualFold(style, string(meta.StyleRPC)) {
		return "POST"
	}
	return "GET"
}

func defaultProtocol(p string) string {
	if p == "" {
		return "HTTPS"
	}
	return p
}

func orSlash(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

func reqBodyType(t string) string {
	if strings.EqualFold(strings.TrimSpace(t), "formData") {
		return "formData"
	}
	return "json"
}

func isDirectBodyParameter(p *meta.Parameter) bool {
	if p == nil || p.Position != meta.PosBody {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(p.Name), "body") ||
		!strings.EqualFold(strings.TrimSpace(p.RawName), "body") {
		return false
	}
	switch p.Type {
	case meta.TypeAny, meta.TypeArray, meta.TypeString:
		return true
	default:
		return false
	}
}

// darabonbaStyle translates the engine's meta.APIStyle vocabulary into
// the two values the darabonba-openapi client actually recognises:
// "RPC" and "ROA".
func darabonbaStyle(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "ROA", "RESTFUL":
		return "ROA"
	default:
		return "RPC"
	}
}

func stripScheme(ep string) string {
	ep = strings.TrimPrefix(strings.TrimPrefix(ep, "https://"), "http://")
	return strings.TrimSuffix(ep, "/")
}

func toStrPtrMap(m map[string]string) map[string]*string {
	out := make(map[string]*string, len(m))
	for k, v := range m {
		out[k] = tea.String(v)
	}
	return out
}

func mergeFlat(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

func scalarString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}
