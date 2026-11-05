// Package legacyapi adapts the CLI's historical wire and presentation contract
// to the V2 generic OpenAPI client. It is independent of the V1 SDK.
package legacyapi

import (
	"bytes"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Request is the CLI preparation model. Values remain strings, including
// explicitly assigned empty values and already flattened parameter names.
type Request struct {
	Scheme, Method, Domain, Port, RegionId       string
	Version, ApiName, Product                    string
	PathPattern, AcceptFormat                    string
	PathParams, QueryParams, FormParams, Headers map[string]string
	Content                                      []byte
	ReadTimeout, ConnectTimeout                  time.Duration
	signedText                                   string
}

func NewRequest() *Request {
	return &Request{Method: "GET", AcceptFormat: "JSON", PathParams: map[string]string{}, QueryParams: map[string]string{}, FormParams: map[string]string{}, Headers: map[string]string{
		"x-sdk-client": "golang/1.0.0", "x-sdk-invoke-type": "common", "Accept-Encoding": "identity",
	}}
}

func (r *Request) SetContent(v []byte)               { r.Content = v }
func (r *Request) SetContentType(v string)           { r.Headers["Content-Type"] = v }
func (r *Request) SetReadTimeout(v time.Duration)    { r.ReadTimeout = v }
func (r *Request) GetReadTimeout() time.Duration     { return r.ReadTimeout }
func (r *Request) SetConnectTimeout(v time.Duration) { r.ConnectTimeout = v }
func (r *Request) GetConnectTimeout() time.Duration  { return r.ConnectTimeout }
func (r *Request) GetStyle() string {
	if r.PathPattern != "" {
		return "ROA"
	}
	return "RPC"
}
func (r *Request) path() string {
	path := r.PathPattern
	if path == "" {
		return "/"
	}
	for _, key := range sortedKeys(r.PathParams) {
		path = strings.Replace(path, "["+key+"]", r.PathParams[key], 1)
	}
	return path
}

// Resource is the unescaped ROA resource used for legacy signing and previews.
func (r *Request) Resource() string {
	if r.GetStyle() == "RPC" {
		return "/?" + encode(r.QueryParams)
	}
	path := r.path()
	for i, key := range sortedKeys(r.QueryParams) {
		if i == 0 {
			path += "?"
		} else {
			path += "&"
		}
		path += key
		if value := r.QueryParams[key]; value != "" {
			path += "=" + value
		}
	}
	return path
}

func (r *Request) String() string {
	if r.signedText != "" {
		return r.signedText
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s %s %s/1.1\nHost: %s\n", r.Method, r.Resource(), strings.ToUpper(r.Scheme), r.Domain)
	for _, k := range sortedKeys(r.Headers) {
		fmt.Fprintf(&b, "%s: %s\n", k, r.Headers[k])
	}
	b.WriteByte('\n')
	if len(r.Content) > 0 {
		b.Write(r.Content)
		b.WriteByte('\n')
	} else {
		for _, k := range sortedKeys(r.FormParams) {
			fmt.Fprintf(&b, "%s: %s\n", k, r.FormParams[k])
		}
	}
	return b.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func encode(m map[string]string) string {
	values := url.Values{}
	for k, v := range m {
		values.Set(k, v)
	}
	return values.Encode()
}
