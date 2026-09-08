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
	"regexp"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
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

// MaskBodyFull redacts text without truncation. Only supported format declarations
// enter text redaction; unknown declarations are omitted even if the body is JSON.
func MaskBodyFull(body string, formats ...string) string {
	if body == "" {
		return ""
	}
	form := false
	for _, format := range formats {
		format = strings.ToLower(strings.TrimSpace(strings.SplitN(format, ";", 2)[0]))
		switch format {
		case "", "raw", "json", "application/json", "text/json":
		case "form", "formdata", "application/x-www-form-urlencoded":
			form = true
		default:
			if !strings.Contains(format, "/") || !strings.HasSuffix(format, "+json") {
				return fmt.Sprintf("[body omitted: %d bytes]", len(body))
			}
		}
	}
	if !utf8.ValidString(body) || strings.ContainsFunc(body, func(r rune) bool {
		return unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t'
	}) {
		return fmt.Sprintf("[body omitted: %d bytes]", len(body))
	}
	var data any
	if json.Valid([]byte(body)) {
		decoder := json.NewDecoder(strings.NewReader(body))
		decoder.UseNumber()
		if err := decoder.Decode(&data); err == nil {
			if masked, err := json.Marshal(maskJSON(data)); err == nil {
				return string(masked)
			}
		}
	}
	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "<") {
		return fmt.Sprintf("[body omitted: %d bytes]", len(body))
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return redactedText(body)
	}
	if form || (strings.Contains(body, "=") && !strings.ContainsAny(body, ",;\"'")) {
		if _, err := url.ParseQuery(body); err != nil {
			return redactedText(body)
		}
		parts := strings.Split(body, "&")
		isForm := true
		for i, part := range parts {
			key, value, found := strings.Cut(part, "=")
			name, _ := url.QueryUnescape(key)
			if strings.ContainsAny(name, " \t:\"'") {
				isForm = false
			}
			if found && isSensitivePath(name) {
				decoded, _ := url.QueryUnescape(value)
				parts[i] = key + "=" + url.QueryEscape(MaskValue(decoded))
			} else {
				decoded, _ := url.QueryUnescape(value)
				if masked := maskEmbeddedJSON(decoded); found && masked != decoded {
					parts[i] = key + "=" + url.QueryEscape(masked)
				} else {
					parts[i] = maskTextFields(part)
				}
			}
		}
		if isForm {
			return strings.Join(parts, "&")
		}
	}
	return maskTextFields(body)
}

func redactedText(body string) string {
	return fmt.Sprintf("[body redacted: %d bytes]", len(body))
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
		return redactedText(value)
	}
	masked := maskJSON(data)
	if reflect.DeepEqual(data, masked) {
		return value
	}
	b, err := json.Marshal(masked)
	if err != nil {
		return redactedText(value)
	}
	return string(b)
}

// ponytail: raw text supports named key/value pairs, not secrets in arbitrary
// prose; add a format parser when another structured text format is supported.
var textField = regexp.MustCompile(`([\w.\[\]-]+|"[\w.\[\]-]+"|'[\w.\[\]-]+')([ \t]*[:=][ \t]*)`)
var textValue = regexp.MustCompile(`^("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\r\n&,;]+)`)

func maskTextFields(body string) string {
	var out strings.Builder
	last := 0
	for _, match := range textField.FindAllStringSubmatchIndex(body, -1) {
		if match[0] < last || !isSensitivePath(strings.Trim(body[match[2]:match[3]], `"'`)) {
			continue
		}
		value := textValue.FindString(body[match[1]:])
		if value == "" {
			if strings.TrimSpace(body[match[1]:]) != "" && strings.ContainsAny(body[match[1]:match[1]+1], "\r\n") {
				return redactedText(body)
			}
			continue
		}
		if (value[0] == '"' || value[0] == '\'') && (len(value) == 1 || value[len(value)-1] != value[0]) {
			return redactedText(body)
		}
		out.WriteString(body[last:match[1]])
		out.WriteString("***")
		last = match[1] + len(value)
	}
	out.WriteString(body[last:])
	return out.String()
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
