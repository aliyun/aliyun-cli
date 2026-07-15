package openapi

import (
	"encoding/json"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
)

type CliDryRunOutput struct {
	Style      string            `json:"style"`
	Endpoint   string            `json:"endpoint"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	Query      map[string]string `json:"query,omitempty"`
	Body       string            `json:"body,omitempty"`
	BodyFormat string            `json:"bodyFormat,omitempty"`

	// ROA specific
	PathPattern string            `json:"pathPattern,omitempty"`
	Pathname    string            `json:"pathname,omitempty"`
	PathParams  map[string]string `json:"pathParams,omitempty"`

	// RPC specific
	Action  string `json:"action,omitempty"`
	Version string `json:"version,omitempty"`
}

var sensitiveHeaderKeys = []string{
	"authorization",
	"x-acs-accesskey-id",
	"x-acs-security-token",
	"x-acs-signature",
}

func sanitizeHeaders(headers map[string]string) map[string]string {
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		if isSensitiveHeader(k) {
			result[k] = maskValue(v)
		} else {
			result[k] = v
		}
	}
	return result
}

func isSensitiveHeader(key string) bool {
	lower := strings.ToLower(key)
	for _, s := range sensitiveHeaderKeys {
		if lower == s {
			return true
		}
	}
	return false
}

func maskValue(v string) string {
	if len(v) <= 4 {
		return "***"
	}
	return v[:4] + "***"
}

func buildCliDryRunFromInvoker(inv Invoker) *CliDryRunOutput {
	req := inv.getRequest()

	out := &CliDryRunOutput{
		Endpoint: req.Domain,
		Method:   req.Method,
		Headers:  sanitizeHeaders(req.Headers),
	}

	if len(req.QueryParams) > 0 {
		out.Query = copyMap(req.QueryParams)
	}

	if len(req.Content) > 0 {
		out.Body = string(req.Content)
		out.BodyFormat = "raw"
	} else if len(req.FormParams) > 0 {
		out.Query = mergeInto(out.Query, nil)
		formJSON, _ := json.Marshal(req.FormParams)
		out.Body = string(formJSON)
		out.BodyFormat = "form"
	}

	if req.PathPattern != "" {
		out.Style = "ROA"
		out.PathPattern = req.PathPattern
		out.Pathname = buildActualPath(req.PathPattern, req.PathParams)
		if len(req.PathParams) > 0 {
			out.PathParams = copyMap(req.PathParams)
		}
	} else {
		out.Style = "RPC"
		out.Action = req.ApiName
		out.Version = req.Version
	}

	return out
}

func buildCliDryRunFromOpenapi(oc *OpenapiContext) *CliDryRunOutput {
	out := &CliDryRunOutput{
		Style:  "ROA",
		Method: oc.method,
	}

	if oc.openapiRequest != nil && oc.openapiRequest.EndpointOverride != nil {
		out.Endpoint = tea.StringValue(oc.openapiRequest.EndpointOverride)
	}
	if out.Endpoint == "" && oc.profile != nil && oc.profile.Endpoint != "" {
		out.Endpoint = oc.profile.Endpoint
	}
	if out.Endpoint == "" && oc.product != nil && strings.ToLower(oc.product.Code) == "sls" {
		if oc.profile != nil && oc.profile.RegionId != "" {
			out.Endpoint = oc.profile.RegionId + ".log.aliyuncs.com"
		}
	}

	if oc.openapiRequest != nil && oc.openapiRequest.Headers != nil {
		h := make(map[string]string, len(oc.openapiRequest.Headers))
		for k, v := range oc.openapiRequest.Headers {
			if v != nil {
				h[k] = *v
			}
		}
		out.Headers = sanitizeHeaders(h)
	}

	if oc.openapiRequest != nil && oc.openapiRequest.Query != nil {
		q := make(map[string]string)
		for k, v := range oc.openapiRequest.Query {
			if v != nil {
				q[k] = *v
			}
		}
		if len(q) > 0 {
			out.Query = q
		}
	}

	if oc.openapiRequest != nil && oc.openapiRequest.Body != nil {
		bodyJSON, err := json.Marshal(oc.openapiRequest.Body)
		if err == nil && string(bodyJSON) != "null" {
			out.Body = string(bodyJSON)
			out.BodyFormat = "json"
		}
	}

	if oc.openapiParams != nil && oc.openapiParams.Pathname != nil {
		out.Pathname = tea.StringValue(oc.openapiParams.Pathname)
	}
	out.PathPattern = oc.path
	if oc.api != nil {
		out.Action = oc.api.Name
		out.Version = oc.api.Product.Version
	}

	if oc.openapiRequest != nil && oc.openapiRequest.Body != nil {
		if stream, ok := oc.openapiRequest.Body.([]byte); ok {
			out.Body = string(stream)
			if oc.openapiParams != nil && oc.openapiParams.ReqBodyType != nil {
				out.BodyFormat = tea.StringValue(oc.openapiParams.ReqBodyType)
			} else {
				out.BodyFormat = "binary"
			}
		}
	}

	return out
}

func marshalCliDryRunOutput(out *CliDryRunOutput) (string, error) {
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func processCliDryRun(ctx *cli.Context, inv Invoker) error {
	out := buildCliDryRunFromInvoker(inv)
	s, err := marshalCliDryRunOutput(out)
	if err != nil {
		return err
	}
	cli.Println(ctx.Stdout(), s)
	return nil
}

func processCliDryRunOpenapi(ctx *cli.Context, oc *OpenapiContext) error {
	out := buildCliDryRunFromOpenapi(oc)
	s, err := marshalCliDryRunOutput(out)
	if err != nil {
		return err
	}
	cli.Println(ctx.Stdout(), s)
	return nil
}

func buildActualPath(pattern string, params map[string]string) string {
	result := pattern
	for k, v := range params {
		placeholder := "[" + k + "]"
		result = strings.ReplaceAll(result, placeholder, v)
	}
	return result
}

func copyMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func mergeInto(dst, src map[string]string) map[string]string {
	if dst == nil && src == nil {
		return nil
	}
	if dst == nil {
		dst = make(map[string]string)
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
