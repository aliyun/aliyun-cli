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
	"net/url"
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

func TestMaskBodyTextFormats(t *testing.T) {
	const secret = "FAKE_SECRET_123"
	for _, tc := range []struct {
		name, body, want string
		hints            []string
	}{
		{"empty", "", "", []string{"binary"}},
		{"form", "name=visible&Password=" + secret, "name=visible&Password=FAKE***", []string{"formData"}},
		{"encoded form keys", "%50assword=" + secret + "&Password=another-secret&name=%E4%B8%AD", "%50assword=FAKE***&Password=anot***&name=%E4%B8%AD", nil},
		{"form escaped delimiters", "password=a%26%3D%2BSECRET&name=alice", "password=a%26%3D%2B***&name=alice", []string{"formData"}},
		{"form empty and short secrets", "password=&token=x&name=alice", "password=&token=***&name=alice", []string{"formData"}},
		{"raw lines", "name: visible\nPassword: " + secret + "\nregion: cn-hangzhou", "name: visible\nPassword: ***\nregion: cn-hangzhou", []string{"raw"}},
		{"raw assignments", "name=visible, password=" + secret + ", region=cn-hangzhou", "name=visible, password=***, region=cn-hangzhou", []string{"raw"}},
		{"raw space separated fields", "name=visible password=" + secret, "name=visible password=***", []string{"raw"}},
		{"form literal newline", "%50assword=" + secret + "\ncontinued&name=visible", "%50assword=FAKE***&name=visible", []string{"formData"}},
		{"raw quoted values", `name: visible, token: "` + secret + `, with spaces", region: cn-hangzhou`, `name: visible, token: ***, region: cn-hangzhou`, nil},
		{"raw quoted assignment", `name=visible password="` + secret + `&continued-secret"`, `name=visible password=***`, []string{"raw"}},
		{"plain text", "ordinary text 正文末尾", "ordinary text 正文末尾", []string{"raw"}},
		{"JSON without declaration", `{"name":"visible","password":"` + secret + `"}`, `{"name":"visible","password":"FAKE***"}`, nil},
		{"JSON with raw declaration", `{"name":"visible","password":"` + secret + `"}`, `{"name":"visible","password":"FAKE***"}`, []string{"raw"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskBodyFull(tc.body, tc.hints...); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			if got := MaskBody(tc.body); strings.Contains(got, secret) {
				t.Errorf("log body leaked secret: %q", got)
			}
		})
	}
}

func TestMaskBodyFormKeepsOriginalStarsEncoded(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"password prefix", "password=*abcSECRET&note=***", "password=%2Aabc***&note=%2A%2A%2A"},
		{"star-only prefix", "password=****SECRET", "password=%2A%2A%2A%2A***"},
		{"already encoded stars", "password=%2AabcSECRET&note=%2A%2A%2A", "password=%2Aabc***&note=%2A%2A%2A"},
		{"ordinary field", "note=***", "note=%2A%2A%2A"},
		{"embedded JSON", "config=" + url.QueryEscape(`{"note":"***","password":"*abcSECRET"}`), "config=%7B%22note%22%3A%22%2A%2A%2A%22%2C%22password%22%3A%22%2Aabc***%22%7D"},
		{"embedded JSON array", "config=" + url.QueryEscape(`[{"note":"*","token":123}]`), "config=%5B%7B%22note%22%3A%22%2A%22%2C%22token%22%3A%22***%22%7D%5D"},
		{"nested JSON string and marker collision", "config=" + url.QueryEscape(`{"inner":"{\"note\":\"\\u005f_REDACTED__***\",\"password\":\"*abcSECRET\"}"}`), "config=%7B%22inner%22%3A%22%7B%5C%22note%5C%22%3A%5C%22__REDACTED__%2A%2A%2A%5C%22%2C%5C%22password%5C%22%3A%5C%22%2Aabc***%5C%22%7D%22%7D"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskBodyFull(tc.body, "formData"); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMaskBodyNestedForm(t *testing.T) {
	const secret = "FAKE_SECRET_123"
	const nested = `{"name":"alice","password":"FAKE_SECRET_123","id":3460000290000487710}`
	const maskedNested = `{"id":3460000290000487710,"name":"alice","password":"FAKE***"}`
	for _, key := range []string{"user.password", "user.password.1", "user.password.2", "user.1.password", "user[password]", "user[1][password]", "user.%70assword.1"} {
		t.Run(key, func(t *testing.T) {
			body := key + "=" + secret + "&user.name=alice"
			got := MaskBodyFull(body, "formData")
			if want := key + "=FAKE***&user.name=alice"; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
	for _, tc := range []struct{ name, body, want string }{
		{"embedded JSON", "config=" + url.QueryEscape(nested), "config=" + strings.ReplaceAll(url.QueryEscape(maskedNested), "%2A", "*")},
		{"embedded JSON array", "config=" + url.QueryEscape("["+nested+"]"), "config=" + strings.ReplaceAll(url.QueryEscape("["+maskedNested+"]"), "%2A", "*")},
		{"repeated JSON fields", "config=" + url.QueryEscape(nested) + "&config=" + url.QueryEscape(nested), "config=" + strings.ReplaceAll(url.QueryEscape(maskedNested), "%2A", "*") + "&config=" + strings.ReplaceAll(url.QueryEscape(maskedNested), "%2A", "*")},
		{"ordinary encoded value", "name=%E4%B8%AD&note=hello%20world", "name=%E4%B8%AD&note=hello%20world"},
		{"non-sensitive JSON spelling", "config=" + url.QueryEscape(`{ "name": "alice", "id": 1 }`), "config=" + url.QueryEscape(`{ "name": "alice", "id": 1 }`)},
		{"flat form map", `{"user.password.1":"FAKE_SECRET_123","user.name":"alice"}`, `{"user.name":"alice","user.password.1":"FAKE***"}`},
		{"non-sensitive similar names", "user.passwordHint=visible&tokenizer.1=visible", "user.passwordHint=visible&tokenizer.1=visible"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskBodyFull(tc.body, "formData"); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
	// ParamStyle=json stores JSON inside a string in the form map.
	body, _ := json.Marshal(map[string]any{"config": nested})
	got := MaskBodyFull(string(body), "formData")
	var decoded map[string]string
	if err := json.Unmarshal([]byte(got), &decoded); err != nil || decoded["config"] != maskedNested {
		t.Fatalf("JSON string field = %s, error = %v", got, err)
	}
	// A JSON-valued form field can itself contain another JSON string field.
	inner, _ := json.Marshal(map[string]string{"inner": nested})
	body = []byte("config=" + url.QueryEscape(string(inner)))
	form, err := url.ParseQuery(MaskBodyFull(string(body), "formData"))
	if err != nil || json.Unmarshal([]byte(form.Get("config")), &decoded) != nil || decoded["inner"] != maskedNested {
		t.Fatalf("nested JSON string field = %v, error = %v", form, err)
	}
	// Damaged nested JSON must not escape via a non-sensitive outer field.
	broken := `{"password":"FAKE_SECRET_123`
	body = []byte("config=" + url.QueryEscape(broken) + "&name=alice")
	if got := MaskBodyFull(string(body), "formData"); strings.Contains(got, secret) || !strings.Contains(got, "name=alice") {
		t.Fatalf("unsafe malformed JSON value: %s", got)
	}
}

func TestMaskBodyFullOmitsXML(t *testing.T) {
	for _, body := range []string{
		`<Request><Password>FAKE_SECRET_123</Password></Request>`,
		" \n<?xml version=\"1.0\"?><credentials><User>中文</User><Password>FAKE_SECRET_123</Password></credentials>",
		`<Request><Password>FAKE_SECRET_123</Request>`,
	} {
		for _, format := range []string{"", "raw", "application/xml"} {
			want := fmt.Sprintf("[body omitted: %d bytes]", len(body))
			if got := MaskBodyFull(body, format); got != want {
				t.Errorf("MaskBodyFull(%q, %q) = %q, want %q", body, format, got, want)
			}
		}
	}
}

func TestMaskBodyUnsafeFormats(t *testing.T) {
	for _, body := range []string{
		`{"password":"FAKE_SECRET_123`,
		`%50assword=FAKE_SECRET_123&name=%zz`,
		"password: \"FAKE_SECRET_123\ncontinued-secret",
		"password:\n  FAKE_SECRET_123",
	} {
		if got, want := MaskBodyFull(body), fmt.Sprintf("[body redacted: %d bytes]", len(body)); got != want {
			t.Errorf("malformed text: got %q, want %q", got, want)
		}
	}
	for _, tc := range []struct{ body, format string }{
		{"\x00FAKE_SECRET_123", "raw"},
		{"\xffFAKE_SECRET_123", ""},
		{`{"password":"FAKE_SECRET_123"}`, "binary"},
		{"Password=FAKE_SECRET_123", "Application/Octet-Stream; charset=utf-8"},
	} {
		if got, want := MaskBodyFull(tc.body, tc.format), fmt.Sprintf("[body omitted: %d bytes]", len(tc.body)); got != want {
			t.Errorf("binary: got %q, want %q", got, want)
		}
	}
}

func TestMaskBodyFullOmitsUnsupportedFormats(t *testing.T) {
	const body = `{"password":"FAKE_SECRET_123"}`
	for _, format := range []string{
		"byte",
		"binary",
		"application/octet-stream",
		"application/x-protobuf",
		"application/pdf",
		"application/zip",
		"application/cbor",
		"application/msgpack",
		"application/x-custom-format",
		" FutureFormat ; version=2",
		"yaml",
		"text/yaml",
		"text/csv",
		"multipart/form-data; boundary=example",
		"image/png",
		"audio/ogg",
		"video/webm",
		"not-a-media-type+json",
		"text/plain",
		"text/xml",
		"application/xml; charset=utf-8",
		"text",
		"xml",
		"application/example+xml; charset=utf-8",
		"image/svg+xml",
	} {
		t.Run(format, func(t *testing.T) {
			want := fmt.Sprintf("[body omitted: %d bytes]", len(body))
			for _, hints := range [][]string{{format}, {"json", format}, {format, "application/json"}} {
				if got := MaskBodyFull(body, hints...); got != want {
					t.Fatalf("MaskBodyFull(%q) = %q, want %q", hints, got, want)
				}
			}
		})
	}
}

func TestMaskBodyFullAllowsSupportedTextFormats(t *testing.T) {
	const body = `{"password":"FAKE_SECRET_123","name":"visible"}`
	for _, format := range []string{
		"",
		"raw",
		"json",
		"form",
		"formData",
		" Application/JSON ; charset=utf-8",
		"application/json; charset=utf-8",
		"application/problem+json",
		"text/json",
		"application/x-www-form-urlencoded",
	} {
		t.Run(format, func(t *testing.T) {
			got := MaskBodyFull(body, format)
			if got != `{"password":"FAKE***","name":"visible"}` && got != `{"name":"visible","password":"FAKE***"}` {
				t.Fatalf("MaskBodyFull(%q) = %q", format, got)
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
