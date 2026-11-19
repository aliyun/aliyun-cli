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
	"fmt"
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
// Bodies that cannot be parsed reliably are omitted instead of echoed, so an
// unparseable payload (XML, raw text, damaged JSON) cannot leak its original
// text into previews or logs.
func MaskBodyFull(body string) string {
	if body == "" {
		return ""
	}
	if json.Valid([]byte(body)) {
		if isJSONContainer(body) {
			return maskEmbeddedJSON(body)
		}
		// A JSON scalar (string, number, boolean, null) exposes no fields to
		// redact, so wrapping text in one must not turn into a verbatim echo.
		return omittedBody(body)
	}
	if masked, ok := maskFormBody(body, 0); ok {
		return masked
	}
	return omittedBody(body)
}

// isJSONContainer reports whether a body is a JSON object or array, the only
// shapes whose fields can be redacted one by one. The caller has already
// validated the body as JSON, so it always has a structural first byte.
func isJSONContainer(body string) bool {
	trimmed := structureText(body)
	return trimmed[0] == '{' || trimmed[0] == '['
}

// maxFormNesting caps recursion into brace-wrapped form values such as
// ext_params={key=value&password=secret}.
const maxFormNesting = 4

// maskFormBody redacts a form body only when every segment is trustworthy:
// field names must be serializer-shaped, every segment must carry a value, and
// each value must be maskable without echoing raw text. Any doubt reports the
// body as unsafe so the caller omits it entirely.
func maskFormBody(body string, depth int) (string, bool) {
	if body == "" || !isFormText(body) {
		return "", false
	}
	if _, err := url.ParseQuery(body); err != nil {
		return "", false
	}
	parts := strings.Split(body, "&")
	out := make([]string, len(parts))
	for i, part := range parts {
		key, value, found := strings.Cut(part, "=")
		name, err := url.QueryUnescape(key)
		if err != nil || !isFormFieldName(name) {
			return "", false
		}
		if !found {
			// Serializers always emit key=value. A bare segment is therefore
			// free text that merely looks like a field name, and echoing it can
			// expose whatever the caller put there.
			return "", false
		}
		// url.ParseQuery above already validated every escape in the body.
		decoded, _ := url.QueryUnescape(value)
		masked, ok := maskFormValue(name, decoded, depth)
		if !ok {
			return "", false
		}
		if masked != decoded {
			out[i] = key + "=" + masked
		} else {
			out[i] = part
		}
	}
	return strings.Join(out, "&"), true
}

// maskFormValue redacts one decoded form value. Plain values are kept; JSON is
// masked field by field; brace-wrapped nested form data is masked recursively.
// Damaged JSON, unbalanced braces, JSON string scalars and embedded assignments
// are reported unsafe so the caller omits the whole body instead of echoing it.
func maskFormValue(name, value string, depth int) (string, bool) {
	if isSensitivePath(name) {
		return MaskValue(value), true
	}
	trimmed := structureText(value)
	if trimmed == "" {
		return value, true
	}
	switch trimmed[0] {
	case '{':
		if json.Valid([]byte(trimmed)) {
			return maskEmbeddedJSON(trimmed), true
		}
		if depth >= maxFormNesting || trimmed[len(trimmed)-1] != '}' {
			return "", false
		}
		masked, ok := maskFormBody(trimmed[1:len(trimmed)-1], depth+1)
		if !ok {
			return "", false
		}
		return "{" + masked + "}", true
	case '[':
		if json.Valid([]byte(trimmed)) {
			return maskEmbeddedJSON(trimmed), true
		}
	case '"':
		if json.Valid([]byte(trimmed)) {
			return "", false
		}
	}
	if isBase64Padded(trimmed) {
		return value, true
	}
	if strings.Contains(trimmed, "=") {
		return "", false
	}
	return value, true
}

// structureText trims surrounding space and leading invisible code points.
// strings.TrimSpace keeps BOM and zero-width characters, which would hide a
// serialized structure behind an unremarkable prefix.
func structureText(value string) string {
	return strings.TrimSpace(strings.TrimLeftFunc(strings.TrimSpace(value), isInvisible))
}

func isInvisible(r rune) bool {
	switch r {
	case '\uFEFF', '\u200B', '\u200C', '\u200D', '\u2060':
		return true
	}
	return false
}

// isBase64Padded reports trailing base64 padding, which is not key=value
// structure and must keep previewing like any other plain value.
func isBase64Padded(value string) bool {
	prefix := strings.TrimRight(value, "=")
	if prefix == value || prefix == "" || len(value)-len(prefix) > 2 {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		c := prefix[i]
		switch {
		case 'A' <= c && c <= 'Z', 'a' <= c && c <= 'z', '0' <= c && c <= '9', c == '+', c == '/':
		default:
			return false
		}
	}
	return true
}

// isFormText rejects ASCII control bytes, spaces and DEL. Non-ASCII bytes stay
// allowed because users do pass raw UTF-8 field values, while a real serializer
// percent-encodes whitespace and controls, so their presence means free text.
func isFormText(body string) bool {
	for i := 0; i < len(body); i++ {
		if c := body[i]; c < '!' || c == 0x7f {
			return false
		}
	}
	return true
}

// isFormFieldName accepts the names produced by form/RPC serializers such as
// user.password.1 and user[1][password]; other spellings cannot be redacted.
func isFormFieldName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', '0' <= r && r <= '9':
		case r == '.', r == '-', r == '_', r == '[', r == ']':
		default:
			return false
		}
	}
	return true
}

// omittedBody reports only the size of a body that cannot be safely masked.
func omittedBody(body string) string {
	return fmt.Sprintf("(body omitted: %d bytes of unparseable content)", len(body))
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
