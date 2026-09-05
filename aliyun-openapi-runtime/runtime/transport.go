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

import "strings"

const (
	DefaultThrottlingRetryMaxAttempts = 3
	DefaultThrottlingRetryMaxDelayMS  = int64(60000)
)

// TransportOptions carries non-secret request context resolved by the Host.
// The runtime owns protocol-specific injection while the Host owns config, environment and CLI flag parsing.
type TransportOptions struct {
	// Headers are host-managed headers such as W3C traceparent and baggage.
	// They override schema/global-flag headers with the same exact key.
	Headers map[string]string

	CallContext     CallContextOptions
	ThrottlingRetry ThrottlingRetryOptions
}

// CallContextOptions identifies the caller to the API gateway.
type CallContextOptions struct {
	SourceIP        string
	SecureTransport string
	// SkipProducts extends the runtime defaults (sls and pds).
	SkipProducts []string
}

// ThrottlingRetryOptions controls retries requested by x-acs-retry-after.
// Enabled nil means enabled, matching aliyun-cli-runtime.
type ThrottlingRetryOptions struct {
	Enabled     *bool
	MaxAttempts int
	MaxDelayMS  int64
}

func (o ThrottlingRetryOptions) IsEnabled() bool {
	return o.Enabled == nil || *o.Enabled
}

func (o ThrottlingRetryOptions) EffectiveMaxAttempts() int {
	if o.MaxAttempts > 0 {
		return o.MaxAttempts
	}
	return DefaultThrottlingRetryMaxAttempts
}

func (o ThrottlingRetryOptions) EffectiveMaxDelayMS() int64 {
	if o.MaxDelayMS > 0 {
		return o.MaxDelayMS
	}
	return DefaultThrottlingRetryMaxDelayMS
}

func applyTransportOptions(req *AssembledRequest, productCode string, options TransportOptions) {
	if req == nil {
		return
	}
	for k, v := range options.Headers {
		if !isValidHeaderName(k) {
			Warn("skip invalid host-managed header name %q", k)
			continue
		}
		req.Headers[k] = sanitizeHeaderValue(v)
	}
	applyCallContext(req, productCode, options.CallContext)
}

func applyCallContext(req *AssembledRequest, productCode string, options CallContextOptions) {
	if req == nil || shouldSkipCallContext(productCode, options.SkipProducts) {
		return
	}
	sourceIP := strings.TrimSpace(options.SourceIP)
	secureTransport := strings.TrimSpace(options.SecureTransport)
	if sourceIP == "" && secureTransport == "" {
		return
	}

	if strings.EqualFold(req.Style, "RPC") {
		setIfAbsent(req.Query, "SourceIp", sourceIP)
		setIfAbsent(req.Query, "SecureTransport", secureTransport)
		return
	}
	setIfAbsent(req.Headers, "x-acs-source-ip", sourceIP)
	setIfAbsent(req.Headers, "x-acs-secure-transport", secureTransport)
}

func shouldSkipCallContext(productCode string, extra []string) bool {
	code := strings.ToLower(strings.TrimSpace(productCode))
	if code == "" {
		return false
	}
	if code == "sls" || code == "pds" {
		return true
	}
	for _, product := range extra {
		if strings.EqualFold(strings.TrimSpace(product), code) {
			return true
		}
	}
	return false
}

func setIfAbsent(target map[string]string, key, value string) {
	if target == nil || value == "" {
		return
	}
	if _, exists := target[key]; !exists {
		target[key] = value
	}
}

func isValidHeaderName(name string) bool {
	if name == "" {
		return false
	}
	const tokenSpecials = "!#$%&'*+-.^_`|~"
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case strings.IndexByte(tokenSpecials, c) >= 0:
		default:
			return false
		}
	}
	return true
}

func sanitizeHeaderValue(value string) string {
	if !strings.ContainsAny(value, "\r\n") {
		return value
	}
	return strings.NewReplacer("\r", "", "\n", "").Replace(value)
}
