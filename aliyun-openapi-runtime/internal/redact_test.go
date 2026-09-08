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

package internal

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestIsSensitiveExactCaseInsensitive(t *testing.T) {
	for _, key := range []string{
		"AccessKeyId", "access_key_secret", "TOKEN",
		"x-acs-accesskey-id", "x-acs-security-token", "Authorization",
	} {
		if !IsSensitive(key) {
			t.Errorf("%q should be sensitive", key)
		}
	}
	for _, key := range []string{"region_id", "image_cache_name", "limit", "tokenizer"} {
		if IsSensitive(key) {
			t.Errorf("%q should not be sensitive", key)
		}
	}
}

func TestMaskValue(t *testing.T) {
	cases := map[string]string{
		"":                "",
		"abcd":            "***",
		"ab":              "***",
		"LTAI5tFakeKeyId": "LTAI***",
	}
	for input, want := range cases {
		if got := MaskValue(input); got != want {
			t.Errorf("MaskValue(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMaskKV(t *testing.T) {
	if got := MaskKV("AccessKeyId", "LTAI5tSecret"); got != "LTAI***" {
		t.Errorf("sensitive value = %q", got)
	}
	if got := MaskKV("region_id", "cn-hangzhou"); got != "cn-hangzhou" {
		t.Errorf("non-sensitive value changed: %q", got)
	}
}

func TestMaskBodyRecursive(t *testing.T) {
	input := `{"regionId":"cn-hangzhou","config":{"accessKeySecret":"topSecretValue"},"tags":[{"password":"hunter2long"}]}`
	output := MaskBody(input)

	var body map[string]any
	if err := json.Unmarshal([]byte(output), &body); err != nil {
		t.Fatalf("masked body is not JSON: %v\n%s", err, output)
	}
	if body["regionId"] != "cn-hangzhou" {
		t.Errorf("non-secret changed: %v", body["regionId"])
	}
	config := body["config"].(map[string]any)
	if config["accessKeySecret"] != "topS***" {
		t.Errorf("nested secret not masked: %v", config["accessKeySecret"])
	}
	tags := body["tags"].([]any)
	if tags[0].(map[string]any)["password"] != "hunt***" {
		t.Errorf("secret in array not masked: %v", tags[0])
	}
}

func TestMaskBodyMasksBeforeTruncating(t *testing.T) {
	secret := "secret-that-must-not-leak"
	input := `{"password":"` + secret + `","padding":"` + strings.Repeat("x", 1200) + `"}`
	output := MaskBody(input)

	if strings.Contains(output, secret) {
		t.Fatalf("secret leaked from long JSON body: %s", output)
	}
	if !strings.HasSuffix(output, "... (truncated)") {
		t.Fatalf("long body was not truncated: %q", output)
	}
}

func TestMaskBodyFullOmitsNonJSON(t *testing.T) {
	const secret = "FAKE_SECRET_123"
	cases := map[string]string{
		"form":        "password=" + secret + "&name=test",
		"xml":         "<Password>" + secret + "</Password>",
		"raw":         "prefix:" + secret,
		"long-tail":   strings.Repeat("x", 1100) + secret,
		"binary":      string(append([]byte{0x00, 0xff, 0x01}, []byte(secret)...)),
		"json-prefix": "payload=" + `{"password":"` + secret + `"}`,
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			want := fmt.Sprintf("[body omitted: %d bytes]", len(input))
			if got := MaskBodyFull(input); got != want {
				t.Fatalf("MaskBodyFull() = %q, want %q", got, want)
			}
		})
	}
}

func TestMaskBodyForDryRunHonorsOpaqueFormat(t *testing.T) {
	const body = `{"password":"FAKE_SECRET_123"}`
	for _, format := range []string{
		"byte",
		"binary",
		"application/octet-stream",
		"application/x-protobuf",
		"application/xml; charset=utf-8",
		"application/example+xml",
		"multipart/form-data; boundary=example",
		"application/x-www-form-urlencoded",
		"text/plain",
		"application/pdf",
		"application/zip",
		"application/cbor",
		"application/msgpack",
		"yaml",
	} {
		t.Run(format, func(t *testing.T) {
			want := fmt.Sprintf("[body omitted: %d bytes]", len(body))
			if got := MaskBodyForDryRun(body, format); got != want {
				t.Fatalf("MaskBodyForDryRun(%q) = %q, want %q", format, got, want)
			}
		})
	}
}

func TestMaskBodyForDryRunAllowsDeclaredJSONAndRawJSON(t *testing.T) {
	const body = `{"password":"FAKE_SECRET_123","name":"visible"}`
	for _, format := range []string{
		"",
		"raw",
		"json",
		"application/json; charset=utf-8",
		"application/problem+json",
	} {
		t.Run(format, func(t *testing.T) {
			got := MaskBodyForDryRun(body, format)
			if got != `{"password":"FAKE***","name":"visible"}` && got != `{"name":"visible","password":"FAKE***"}` {
				t.Fatalf("MaskBodyForDryRun(%q) = %q", format, got)
			}
		})
	}
}

func TestMaskBodyFullPreservesCompleteRedactedJSON(t *testing.T) {
	padding := strings.Repeat("x", 1200)
	input := `{"content":"` + padding + `","password":"FAKE_SECRET_123"}`
	output := MaskBodyFull(input)

	if strings.Contains(output, "FAKE_SECRET_123") {
		t.Fatalf("secret leaked from JSON body: %s", output)
	}
	if !strings.Contains(output, padding) {
		t.Fatal("dry-run JSON body was truncated")
	}
	if !strings.Contains(output, `"password":"FAKE***"`) {
		t.Fatalf("JSON secret was not masked: %s", output)
	}
}

func TestMaskBodyFullEmpty(t *testing.T) {
	if got := MaskBodyFull(""); got != "" {
		t.Fatalf("MaskBodyFull(empty) = %q", got)
	}
}

func TestMaskBodyTopLevelArrayPreservesNumbers(t *testing.T) {
	const secret = "body-secret-value"
	input := `[{"TaskId":3460000290000487710,"password":"` + secret + `"}]`
	output := MaskBody(input)

	if strings.Contains(output, secret) {
		t.Fatalf("secret leaked from JSON array body: %s", output)
	}
	if !strings.Contains(output, `3460000290000487710`) {
		t.Fatalf("large JSON number changed: %s", output)
	}
	var body []any
	decoder := json.NewDecoder(strings.NewReader(output))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		t.Fatalf("masked body is not valid JSON: %v", err)
	}
	item := body[0].(map[string]any)
	if item["password"] != "body***" {
		t.Fatalf("password was not masked: %#v", item)
	}
}

func TestMaskBodyWholeObjectSecret(t *testing.T) {
	output := MaskBody(`{"credentials":{"accessKeySecret":"x"}}`)
	var body map[string]any
	if err := json.Unmarshal([]byte(output), &body); err != nil {
		t.Fatalf("masked body is not JSON: %v", err)
	}
	if body["credentials"] != "***" {
		t.Fatalf("whole-object secret should be masked, got %#v", body["credentials"])
	}
}

func TestAddCustomField(t *testing.T) {
	saved := make([]string, 0)
	mu.RLock()
	for key := range fields {
		saved = append(saved, key)
	}
	mu.RUnlock()
	defer SetFields(saved)

	if IsSensitive("my_custom_secret") {
		t.Fatal("precondition: field should not be sensitive")
	}
	Add("my_custom_secret")
	if !IsSensitive("MY_CUSTOM_SECRET") {
		t.Fatal("Add did not register the field case-insensitively")
	}
}
