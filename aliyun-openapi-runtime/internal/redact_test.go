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

func TestMaskBodyContentDetection(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"empty", "", ""},
		{"form", "name=visible&Password=example-secret", "name=visible&Password=exam***"},
		{"encoded key and repeated fields", "%50assword=example-secret&Password=another-secret&name=%E4%B8%AD", "%50assword=exam***&Password=anot***&name=%E4%B8%AD"},
		{"form empty and short secrets", "password=&token=x&name=alice", "password=&token=***&name=alice"},
		{"form literal newline", "password=example-secret\ncontinued&name=visible", ""},
		{"raw lines", "name: visible\npassword: example-secret", ""},
		{"raw assignments", "name=visible, password=example-secret", ""},
		{"raw space separated fields", "name=visible password=example-secret", ""},
		{"plain text", "ordinary text 正文末尾", ""},
		{"JSON", `{"name":"visible","password":"example-secret"}`, `{"name":"visible","password":"exam***"}`},
		{"unchanged JSON", `{ "name": "visible" }`, `{ "name": "visible" }`},
		{"original stars", "password=*abcSECRET&note=***", "password=*abc***&note=***"},
		{"encoded stars", "password=%2AabcSECRET&note=%2a%2A%2a", "password=*abc***&note=%2a%2A%2a"},
		{"escaped delimiters", "password=a%26%3D%2BSECRET&name=alice", "password=a&=+***&name=alice"},
		{"encoded and literal spaces", "note=hello%20world&note=hello+world&password=example-secret", "note=hello%20world&note=hello+world&password=exam***"},
		{"Unicode inside form value", "name=vis\u200bible&password=example-secret", "name=vis\u200bible&password=exam***"},
		{"encoded Unicode inside form value", "name=vis%E2%80%8Bible&password=example-secret", "name=vis%E2%80%8Bible&password=exam***"},
		{"embedded JSON without encoding", `config={"note":"***","password":"*abcSECRET"}`, `config={"note":"***","password":"*abc***"}`},
		{"encoded embedded JSON", "config=" + url.QueryEscape(`{"note":"***","password":"*abcSECRET"}`), `config={"note":"***","password":"*abc***"}`},
		{"embedded JSON array", "config=" + url.QueryEscape(`[{"note":"*","token":123}]`), `config=[{"note":"*","token":"***"}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.want == "" && tc.body != "" {
				tc.want = fmt.Sprintf("<opaque body omitted: %d bytes>", len(tc.body))
			}
			if got := MaskBodyFull(tc.body); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMaskBodyFormOmitsUnparseableValues(t *testing.T) {
	for _, body := range []string{
		`ext_params={key=example-string}`,
		`ext_params={password=secret*value}`,
		`ext_params=%7bpassword%3Dsecret%2avalue%7d`,
		`ext_params={"password":"secret*value"`,
		`ext_params="password=secret*value`,
	} {
		const sensitive = "&user.password.1=example-secret"
		want := fmt.Sprintf("<opaque body omitted: %d bytes>", len(body+sensitive))
		if got := MaskBodyFull(body + sensitive); got != want {
			t.Errorf("unparseable field output = %q, want %q", got, want)
		}
	}
	for _, body := range []string{
		`{password=secret*value}`,
		`{"password":"secret*value"`,
		`password=example-secret&name=%zz`,
		`password=example-secret;name=alice`,
	} {
		want := fmt.Sprintf("<opaque body omitted: %d bytes>", len(body))
		if got := MaskBodyFull(body); got != want {
			t.Errorf("unparseable form output = %q, want %q", got, want)
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
	broken := `{"password":"FAKE_SECRET_123`
	body = []byte("config=" + url.QueryEscape(broken) + "&name=alice")
	if got, want := MaskBodyFull(string(body)), fmt.Sprintf("<opaque body omitted: %d bytes>", len(body)); got != want {
		t.Fatalf("malformed JSON value output = %q, want %q", got, want)
	}
}

func TestMaskBodyOmitsUnparseableContent(t *testing.T) {
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
		want := fmt.Sprintf("<opaque body omitted: %d bytes>", len(body))
		if got := MaskBodyFull(body); got != want {
			t.Errorf("got %q, want %q", got, want)
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

func TestMaskBodyOmitsOpaqueContent(t *testing.T) {
	const secret = "FAKE_SECRET_20261103"
	for name, body := range map[string]string{
		"duplicate nested value":    `{"x":"[` + secret + `","x":""}`,
		"duplicate sensitive key":   `{"password":"` + secret + `","password":""}`,
		"escaped duplicate key":     `{"password":"` + secret + `","pass\u0077ord":""}`,
		"duplicate inside array":    `[{"password":"` + secret + `","password":""}]`,
		"duplicate in form JSON":    "config=" + url.QueryEscape(`{"password":"`+secret+`","password":""}`),
		"xml comment":               "<!--password=" + secret + "-->",
		"invalid form":              "password=" + secret + "&name=%zz",
		"xml":                       "<Password>" + secret + "</Password>",
		"BOM XML":                   "\ufeff<Password>" + secret + "==</Password>",
		"whitespace before BOM XML": " \t\ufeff<Password>" + secret + "==</Password>",
		"BOM JSON":                  "\ufeff{\"password\":\"" + secret + "\"}",
		"BOM form":                  "\ufeffpassword=" + secret,
		"broken JSON":               `{"password":"` + secret,
		"raw":                       "password: " + secret,
		"long tail":                 strings.Repeat("x", 1600) + secret,
		"binary":                    "\x00\xff" + secret,
		"invalid UTF8 JSON":         "{\"name\":\"\xff" + secret + "\"}",
		"invalid UTF8 form":         "name=%FF" + secret,
		"empty key":                 "=" + secret,
		"decoded empty key":         "%00=" + secret + "&name=%zz",
		"missing equals":            "name=visible&" + secret,
		"trailing separator":        "password=" + secret + "&",
		"unencoded whitespace":      "name=visible password=" + secret,
		"control character":         "name=visible\x00&password=" + secret,
		"scalar JSON":               `"` + secret + `"`,
		"number JSON":               "42",
		"bool JSON":                 "true",
		"null JSON":                 "null",
		"broken nested JSON":        "config=" + url.QueryEscape(`{"password":"`+secret),
		"broken nested quoted JSON": "config=" + url.QueryEscape(`"`+secret),
		"nested object":             `{"config":"{\"password\":\"` + secret + `"}`,
		"nested array":              `[{"config":"{\"password\":\"` + secret + `"}]`,
		"damaged JSON with equals":  `{"name":"visible","config":"password=` + secret,
	} {
		t.Run(name, func(t *testing.T) {
			want := fmt.Sprintf("<opaque body omitted: %d bytes>", len(body))
			for _, mask := range []func(string) string{MaskBodyFull, MaskBody} {
				if got := mask(body); got != want {
					t.Fatalf("opaque body output = %q, want %q", got, want)
				}
			}
		})
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

func TestMaskBodyPreservesOrdinaryScalarText(t *testing.T) {
	for _, text := range []string{
		"[Important] ordinary text", "[输入文档内容]", "{placeholder}", "${user}",
		`"quoted" ordinary text`, "[2026 report]", "[name]", "{YourContent}",
		"[base64 encode content]", "{base64}", "{resourceId}", "{regionId}", "{appName}",
		`""，"hive"等`, `"标题示例“`, `"消息内容示例“`, "\u200b00000001\n\n",
		"\"0\" ：parent folder is root\n\"4b648f6d11344f258df876cbdc24dc1f\" ： folderId",
	} {
		body, _ := json.Marshal(map[string]string{"message": text, "name": "visible"})
		if got := MaskBodyFull(string(body)); got != string(body) {
			t.Errorf("ordinary JSON scalar changed: got %q, want %q", got, body)
		}
		form := "message=" + url.QueryEscape(text) + "&name=visible"
		if got := MaskBodyFull(form); got != form {
			t.Errorf("ordinary form scalar changed: got %q, want %q", got, form)
		}
	}
}

func TestMaskBodyQuotedDocumentTrailingText(t *testing.T) {
	const secret = "FAKE_QA_TRAILING_PASSWORD_20261104"
	for _, nested := range []string{
		`{"password":"` + secret + `","name":"visible"}`,
		`[{"password":"` + secret + `"}]`,
		`{"password":"` + secret + `",`,
		"\ufeff{\"password\":\"" + secret + "\"}",
		"\u200b{\"password\":\"" + secret + "\"}",
	} {
		for depth := 1; depth <= 3; depth++ {
			quoted, _ := json.Marshal(nested)
			nested = string(quoted)
			for _, suffix := range []string{" trailing", "junk", "\n{}", "\x00"} {
				value := nested + suffix
				t.Run(fmt.Sprintf("%d/%q", depth, value), func(t *testing.T) {
					if !looksLikeJSONDocument(value) || canRedactJSON(value) {
						t.Fatal("incomplete serialized document was treated as ordinary text")
					}
					want := fmt.Sprintf("<opaque body omitted: %d bytes>", len(value))
					if got := MaskKV("payload", value); got != want {
						t.Fatalf("unsafe nested preview: got %q, want %q", got, want)
					}
					body, _ := json.Marshal(map[string]string{"payload": value, "name": "visible"})
					array, _ := json.Marshal([]string{value})
					for _, input := range []string{string(body), string(array), "payload=" + url.QueryEscape(value) + "&name=visible"} {
						for _, mask := range []func(string) string{MaskBodyFull, MaskBody} {
							want := fmt.Sprintf("<opaque body omitted: %d bytes>", len(input))
							if got := mask(input); got != want || strings.Contains(got, secret) {
								t.Fatalf("unsafe body preview: got %q, want %q", got, want)
							}
						}
					}
				})
			}
		}
	}
	quoted, _ := json.Marshal(`{"password":"` + secret + `","name":"visible"}`)
	body, _ := json.Marshal(map[string]string{"payload": string(quoted), "name": "visible"})
	if got := MaskBodyFull(string(body)); strings.Contains(got, secret) || !strings.Contains(got, "FAKE***") || !strings.Contains(got, "visible") || strings.Contains(got, "opaque body omitted") {
		t.Fatalf("complete nested document preview regressed: %q", got)
	}
}

func TestLooksLikeJSONDocument(t *testing.T) {
	for _, value := range []string{"", "visible", "[Important] text", "[2026 report]", "{YourContent}", `"quoted" ordinary text`} {
		if looksLikeJSONDocument(value) {
			t.Errorf("ordinary scalar classified as JSON: %q", value)
		}
	}
	for _, value := range []string{"{", "[", "{}", "[]", "[1,2]", "{password=secret}", `{"password":`, `[{"password":`, `["secret`, `[1,`, `[true,`, `[null,`, `[1`, `"secret`, `"visible"`} {
		if !looksLikeJSONDocument(value) {
			t.Errorf("serialized document not recognized: %q", value)
		}
	}
}

func TestMaskEmbeddedJSONFailsClosed(t *testing.T) {
	const secret = "FAKE_QA_ARGUMENT_PASSWORD_20261104"
	for _, value := range []string{
		`{"password":"` + secret + `",`,
		`[{"password":"` + secret + `"}`,
		`[1,{"password":"` + secret + `"}`,
		`[true,{"password":"` + secret + `"}`,
		`[null,{"password":"` + secret + `"}`,
		`{password:` + secret + `}`,
		`[password=` + secret + `]`,
		"\ufeff{\"password\":\"" + secret + "\"}",
		"\u200b<Password>" + secret + "</Password>",
		`{"password":"` + secret + `","password":""}`,
	} {
		want := fmt.Sprintf("<opaque body omitted: %d bytes>", len(value))
		if got := MaskKV("name", value); got != want {
			t.Errorf("unsafe scalar output: got %q, want %q", got, want)
		}
	}
}

func TestMaskBodyUnicodePrefixes(t *testing.T) {
	const secret = "FAKE_SECRET_20261103"
	for _, prefix := range []string{"\ufeff", "\u200b", "\u200c", "\u200d", "\u2060", "\u202e", " \t\ufeff"} {
		nested := prefix + `{"password":"` + secret + `"}`
		object, _ := json.Marshal(map[string]string{"payload": nested})
		array, _ := json.Marshal([]string{nested})
		quoted, _ := json.Marshal(nested)
		for _, body := range []string{
			prefix + "<Password>" + secret + "==</Password>",
			prefix + "raw=" + secret,
			url.QueryEscape(prefix+"raw") + "=" + secret,
			string(object), string(array),
			"payload=" + url.QueryEscape(nested),
			"payload=" + url.QueryEscape(string(quoted)),
		} {
			t.Run(fmt.Sprintf("%q/%q", prefix, body), func(t *testing.T) {
				want := fmt.Sprintf("<opaque body omitted: %d bytes>", len(body))
				for _, mask := range []func(string) string{MaskBodyFull, MaskBody} {
					if got := mask(body); got != want {
						t.Fatalf("got %q, want %q", got, want)
					}
				}
			})
		}
	}
}

func TestMaskBodySensitiveValuesNeedNotBeJSON(t *testing.T) {
	const secret = "FAKE_SECRET_20261103"
	for _, value := range []string{"[" + secret, "{" + secret, `"` + secret, "\ufeff" + secret, "\u200b" + secret} {
		for _, key := range []string{"password", "user.password.1", "user[password]"} {
			object, _ := json.Marshal(map[string]string{key: value, "name": "visible"})
			want, _ := json.Marshal(map[string]string{key: MaskValue(value), "name": "visible"})
			body := key + "=" + url.QueryEscape(value) + "&name=visible"
			for _, tc := range []struct{ body, want string }{
				{string(object), string(want)},
				{body, key + "=" + MaskValue(value) + "&name=visible"},
			} {
				t.Run(tc.body, func(t *testing.T) {
					if got := MaskBodyFull(tc.body); got != tc.want || strings.Contains(got, secret) {
						t.Fatalf("got %q, want %q", got, tc.want)
					}
				})
			}
		}
	}
	for _, tc := range []struct{ body, want string }{
		{`{"credentials":{"items":["[FAKE_SECRET_20261103"]},"name":"visible"}`, `{"credentials":"***","name":"visible"}`},
		{`[{"password":"[FAKE_SECRET_20261103","name":"visible"}]`, `[{"name":"visible","password":"[FAK***"}]`},
		{`{"payload":"{\"password\":\"[FAKE_SECRET_20261103\",\"name\":\"visible\"}"}`, `{"payload":"{\"name\":\"visible\",\"password\":\"[FAK***\"}"}`},
	} {
		if got := MaskBodyFull(tc.body); got != tc.want {
			t.Fatalf("got %q, want %q", got, tc.want)
		}
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
