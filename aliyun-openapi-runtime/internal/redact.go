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

// Package internal provides shared implementation details for the runtime.
package internal

import (
	"encoding/json"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
)

const (
	envBearerTokenHeaderKey = "ALIBABA_CLOUD_BEARER_TOKEN_HEADER_KEY"
	outputLimit             = 1000
)

var (
	mu     sync.RWMutex
	fields = defaultFields()
)

func defaultFields() map[string]bool {
	return map[string]bool{
		"access_key": true, "access_key_id": true, "access_key_secret": true,
		"accesskey": true, "accesskeyid": true, "accesskeysecret": true,
		"ak": true, "sk": true,
		"secret": true, "secret_key": true, "secretkey": true,
		"password": true, "passwd": true, "pwd": true,
		"token": true, "auth": true, "authorization": true, "bearer": true,
		"api_key": true, "apikey": true,
		"id_token": true, "idtoken": true, "buc_id_token": true,
		"access_token": true, "accesstoken": true,
		"refresh_token": true, "refreshtoken": true,
		"sts_token": true, "ststoken": true,
		"security_token": true, "securitytoken": true,
		"x-acs-accesskey-id":     true,
		"x-acs-buc-bearer-token": true,
		"x-acs-bearer-token":     true,
		"x-acs-security-token":   true,
		"x-acs-signature":        true,
		"x-acs-credential":       true,
		"cookie":                 true,
		"set-cookie":             true,
		"credential":             true,
		"credentials":            true,
		"private_key":            true,
		"privatekey":             true,
	}
}

func Add(field string) {
	mu.Lock()
	defer mu.Unlock()
	fields[strings.ToLower(field)] = true
}

func SetFields(names []string) {
	mu.Lock()
	defer mu.Unlock()
	fields = make(map[string]bool, len(names))
	for _, name := range names {
		fields[strings.ToLower(name)] = true
	}
}

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

// MaskValue retains the first four bytes as a fingerprint.
func MaskValue(value string) string {
	if len(value) == 0 {
		return value
	}
	if len(value) <= 4 {
		return "***"
	}
	return value[:4] + "***"
}

// Form/RPC serializers flatten objects and arrays into keys such as
// user.password.1 and user.1.password. Match whole path components only.
func isSensitivePath(field string) bool {
	if IsSensitive(field) {
		return true
	}
	for _, part := range strings.FieldsFunc(field, func(r rune) bool { return r == '.' || r == '[' || r == ']' }) {
		if IsSensitive(part) {
			return true
		}
	}
	return false
}

func MaskKV(key, value string) string {
	if isSensitivePath(key) {
		return MaskValue(value)
	}
	return maskEmbeddedJSON(value)
}

func MaskBody(body string) string {
	return truncate(MaskBodyFull(body))
}

// MaskBodyFull redacts parseable JSON or form content without truncation.
// Unparseable content and unchanged form fields retain their original spelling.
func MaskBodyFull(body string) string {
	if json.Valid([]byte(body)) {
		return maskEmbeddedJSON(body)
	}
	if _, err := url.ParseQuery(body); err != nil {
		return body
	}
	parts := strings.Split(body, "&")
	for i, part := range parts {
		key, value, found := strings.Cut(part, "=")
		if !found {
			continue
		}
		name, _ := url.QueryUnescape(key)
		decoded, _ := url.QueryUnescape(value)
		if masked := MaskKV(name, decoded); masked != decoded {
			parts[i] = key + "=" + masked
		}
	}
	return strings.Join(parts, "&")
}

// ParamStyle=json puts serialized JSON inside a form field's string value.
// Retain its string type, and preserve the original spelling when unchanged.
func maskEmbeddedJSON(value string) string {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, `"`) {
		return value
	}
	var data any
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	if !json.Valid([]byte(value)) || decoder.Decode(&data) != nil {
		return value
	}
	masked := maskJSON(data)
	if reflect.DeepEqual(data, masked) {
		return value
	}
	b, err := json.Marshal(masked)
	if err != nil {
		return value
	}
	return string(b)
}

func MaskAny(data any) any {
	return maskJSON(data)
}

func maskJSON(data any) any {
	switch value := data.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			if isSensitivePath(key) {
				if text, ok := item.(string); ok {
					out[key] = MaskValue(text)
				} else {
					out[key] = "***"
				}
				continue
			}
			out[key] = maskJSON(item)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = maskJSON(item)
		}
		return out
	case string:
		return maskEmbeddedJSON(value)
	default:
		return data
	}
}

func truncate(value string) string {
	if len(value) <= outputLimit {
		return value
	}
	return value[:outputLimit] + "... (truncated)"
}
