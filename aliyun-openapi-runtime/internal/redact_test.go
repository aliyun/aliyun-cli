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

func TestMaskBodyContentDetection(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"empty", "", ""},
		{"form", "name=visible&Password=example-secret", "name=visible&Password=exam***"},
		{"encoded key and repeated fields", "%50assword=example-secret&Password=another-secret&name=%E4%B8%AD", "%50assword=exam***&Password=anot***&name=%E4%B8%AD"},
		{"form empty and short secrets", "password=&token=x&name=alice", "password=&token=***&name=alice"},
		{"form literal newline", "password=example-secret\ncontinued&name=visible", "password=exam***&name=visible"},
		{"raw lines", "name: visible\npassword: example-secret", "name: visible\npassword: example-secret"},
		{"raw assignments", "name=visible, password=example-secret", "name=visible, password=example-secret"},
		{"raw space separated fields", "name=visible password=example-secret", "name=visible password=example-secret"},
		{"plain text", "ordinary text 正文末尾", "ordinary text 正文末尾"},
		{"JSON", `{"name":"visible","password":"example-secret"}`, `{"name":"visible","password":"exam***"}`},
		{"unchanged JSON", `{ "name": "visible" }`, `{ "name": "visible" }`},
		{"original stars", "password=*abcSECRET&note=***", "password=*abc***&note=***"},
		{"encoded stars", "password=%2AabcSECRET&note=%2a%2A%2a", "password=*abc***&note=%2a%2A%2a"},
		{"escaped delimiters", "password=a%26%3D%2BSECRET&name=alice", "password=a&=+***&name=alice"},
		{"encoded and literal spaces", "note=hello%20world&note=hello+world&password=example-secret", "note=hello%20world&note=hello+world&password=exam***"},
		{"embedded JSON without encoding", `config={"note":"***","password":"*abcSECRET"}`, `config={"note":"***","password":"*abc***"}`},
		{"encoded embedded JSON", "config=" + url.QueryEscape(`{"note":"***","password":"*abcSECRET"}`), `config={"note":"***","password":"*abc***"}`},
		{"embedded JSON array", "config=" + url.QueryEscape(`[{"note":"*","token":123}]`), `config=[{"note":"*","token":"***"}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskBodyFull(tc.body); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMaskBodyFormPreservesUnparseableValues(t *testing.T) {
	for _, body := range []string{
		`ext_params={key=example-string}`,
		`ext_params={password=secret*value}`,
		`ext_params=%7bpassword%3Dsecret%2avalue%7d`,
		`ext_params={"password":"secret*value"`,
		`ext_params="password=secret*value`,
	} {
		const sensitive = "&user.password.1=example-secret"
		if got := MaskBodyFull(body + sensitive); got != body+"&user.password.1=exam***" {
			t.Errorf("changed unparseable field or missed sibling redaction: %q", got)
		}
	}
	for _, body := range []string{
		`{password=secret*value}`,
		`{"password":"secret*value"`,
		`password=example-secret&name=%zz`,
		`password=example-secret;name=alice`,
	} {
		if got := MaskBodyFull(body); got != body {
			t.Errorf("changed unparseable form: got %q, want %q", got, body)
		}
	}
}

func TestMaskBodyNestedForm(t *testing.T) {
	const secret = "FAKE_SECRET_123"
	const nested = `{"name":"alice","password":"FAKE_SECRET_123","id":3460000290000487710}`
	const maskedNested = `{"id":3460000290000487710,"name":"alice","password":"FAKE***"}`
	for _, key := range []string{"user.password", "user.password.1", "user.password.2", "user.1.password", "user[password]", "user[1][password]", "user.%70assword.1"} {
		t.Run(key, func(t *testing.T) {
			body := key + "=" + secret + "&user.name=alice"
			got := MaskBodyFull(body)
			if want := key + "=FAKE***&user.name=alice"; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
	for _, tc := range []struct{ name, body, want string }{
		{"embedded JSON", "config=" + url.QueryEscape(nested), "config=" + maskedNested},
		{"embedded JSON array", "config=" + url.QueryEscape("["+nested+"]"), "config=" + "[" + maskedNested + "]"},
		{"repeated JSON fields", "config=" + url.QueryEscape(nested) + "&config=" + url.QueryEscape(nested), "config=" + maskedNested + "&config=" + maskedNested},
		{"ordinary encoded value", "name=%E4%B8%AD&note=hello%20world", "name=%E4%B8%AD&note=hello%20world"},
		{"non-sensitive JSON spelling", "config=" + url.QueryEscape(`{ "name": "alice", "id": 1 }`), "config=" + url.QueryEscape(`{ "name": "alice", "id": 1 }`)},
		{"flat form map", `{"user.password.1":"FAKE_SECRET_123","user.name":"alice"}`, `{"user.name":"alice","user.password.1":"FAKE***"}`},
		{"non-sensitive similar names", "user.passwordHint=visible&tokenizer.1=visible", "user.passwordHint=visible&tokenizer.1=visible"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskBodyFull(tc.body); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
	// ParamStyle=json stores JSON inside a string in the form map.
	body, _ := json.Marshal(map[string]any{"config": nested})
	got := MaskBodyFull(string(body))
	var decoded map[string]string
	if err := json.Unmarshal([]byte(got), &decoded); err != nil || decoded["config"] != maskedNested {
		t.Fatalf("JSON string field = %s, error = %v", got, err)
	}
	// A JSON-valued form field can itself contain another JSON string field.
	inner, _ := json.Marshal(map[string]string{"inner": nested})
	body = []byte("config=" + url.QueryEscape(string(inner)))
	form, err := url.ParseQuery(MaskBodyFull(string(body)))
	if err != nil || json.Unmarshal([]byte(form.Get("config")), &decoded) != nil || decoded["inner"] != maskedNested {
		t.Fatalf("nested JSON string field = %v, error = %v", form, err)
	}
	// Preserve damaged nested JSON verbatim so users can diagnose the request.
	broken := `{"password":"FAKE_SECRET_123`
	body = []byte("config=" + url.QueryEscape(broken) + "&name=alice")
	if got := MaskBodyFull(string(body)); got != string(body) {
		t.Fatalf("changed malformed JSON value: %s", got)
	}
}

func TestMaskBodyPreservesUnparseableContent(t *testing.T) {
	for _, body := range []string{
		`<Request><Password>example-secret</Password></Request>`,
		`{"password":"example-secret`,
		`password=example-secret&name=%zz`,
		`password=example-secret;name=alice`,
		"password: \"example-secret\ncontinued-secret",
		"password:\n  example-secret",
		"\x00example-secret",
		"\xffexample-secret",
	} {
		if got := MaskBodyFull(body); got != body {
			t.Errorf("got %q, want unchanged %q", got, body)
		}
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
