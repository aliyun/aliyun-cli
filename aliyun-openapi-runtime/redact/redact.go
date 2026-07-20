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

// Package redact hides secrets in diagnostic output (dry-run dumps,
// request/response logging). It is a focused port of the sensitive
// field handling in aliyun-cli-runtime/http, kept engine-agnostic so
// both the dry-run renderer and any future request logger can share
// one registry.
//
// Matching is EXACT on the lowercased key (not substring), so dashed
// header names must be enumerated explicitly — see defaultFields.
package redact

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

// envBearerTokenHeaderKey lets deployments mark an additional header
// as sensitive without a code change, mirroring the plugin engine.
const envBearerTokenHeaderKey = "ALIBABA_CLOUD_BEARER_TOKEN_HEADER_KEY"

var (
	mu     sync.RWMutex
	fields = defaultFields()
)

// defaultFields is the built-in sensitive key set. Keys are lowercase;
// dashed header variants are spelled out because matching is exact.
func defaultFields() map[string]bool {
	return map[string]bool{
		// Access keys
		"access_key": true, "access_key_id": true, "access_key_secret": true,
		"accesskey": true, "accesskeyid": true, "accesskeysecret": true,
		"ak": true, "sk": true,

		// Secret keys
		"secret": true, "secret_key": true, "secretkey": true,

		// Passwords
		"password": true, "passwd": true, "pwd": true,

		// Tokens and authorization
		"token": true, "auth": true, "authorization": true, "bearer": true,
		"api_key": true, "apikey": true,
		"id_token": true, "idtoken": true, "buc_id_token": true,
		"access_token": true, "accesstoken": true,
		"refresh_token": true, "refreshtoken": true,
		"sts_token": true, "ststoken": true,
		"security_token": true, "securitytoken": true,

		// HTTP header sensitive keys (exact, lowercased)
		"x-acs-buc-bearer-token": true,
		"x-acs-bearer-token":     true,
		"x-acs-security-token":   true,
		"x-acs-signature":        true,
		"x-acs-credential":       true,
		"cookie":                 true,
		"set-cookie":             true,

		// Other sensitive data
		"credential": true, "credentials": true,
		"private_key": true, "privatekey": true,
	}
}

// Add registers an extra sensitive field (lowercased). Safe for
// concurrent use.
func Add(field string) {
	mu.Lock()
	defer mu.Unlock()
	fields[strings.ToLower(field)] = true
}

// SetFields replaces the entire registry with the given names.
func SetFields(names []string) {
	mu.Lock()
	defer mu.Unlock()
	fields = make(map[string]bool, len(names))
	for _, n := range names {
		fields[strings.ToLower(n)] = true
	}
}

// IsSensitive reports whether field is a known secret key (exact,
// case-insensitive match), also honouring the env-configured header.
func IsSensitive(field string) bool {
	key := strings.ToLower(field)
	mu.RLock()
	hit := fields[key]
	mu.RUnlock()
	if hit {
		return true
	}
	if env := strings.TrimSpace(os.Getenv(envBearerTokenHeaderKey)); env != "" {
		return key == strings.ToLower(env)
	}
	return false
}

// MaskValue partially hides a secret: it keeps the first 4 characters
// as a fingerprint and replaces the rest with "***". Values of 4 chars
// or fewer are fully masked.
func MaskValue(value string) string {
	if len(value) == 0 {
		return value
	}
	if len(value) <= 4 {
		return "***"
	}
	return value[:4] + "***"
}

// MaskKV returns value masked when key is sensitive, otherwise the
// value unchanged. It is the convenience form used by key/value dumps.
func MaskKV(key, value string) string {
	if IsSensitive(key) {
		return MaskValue(value)
	}
	return value
}

// MaskBody masks secret fields inside a JSON body (recursively) and
// truncates very long bodies. Non-JSON bodies are returned unchanged
// (but still truncated).
func MaskBody(body string) string {
	const limit = 1000
	if len(body) > limit {
		body = body[:limit] + "... (truncated)"
	}
	var data any
	if err := json.Unmarshal([]byte(body), &data); err == nil {
		if b, err := json.Marshal(maskJSON(data)); err == nil {
			return string(b)
		}
	}
	return body
}

// MaskAny masks a decoded JSON-like value in place-ish (returns a
// masked copy). Useful when the caller already holds structured data.
func MaskAny(data any) any {
	return maskJSON(data)
}

func maskJSON(data any) any {
	switch v := data.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			if IsSensitive(k) {
				if s, ok := val.(string); ok {
					out[k] = MaskValue(s)
				} else {
					out[k] = "***"
				}
				continue
			}
			out[k] = maskJSON(val)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = maskJSON(val)
		}
		return out
	default:
		return data
	}
}
