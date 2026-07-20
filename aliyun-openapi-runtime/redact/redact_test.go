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

package redact

import (
	"encoding/json"
	"testing"
)

func TestIsSensitiveExactCaseInsensitive(t *testing.T) {
	for _, k := range []string{"AccessKeyId", "access_key_secret", "TOKEN", "x-acs-security-token", "Authorization"} {
		if !IsSensitive(k) {
			t.Errorf("%q should be sensitive", k)
		}
	}
	for _, k := range []string{"region_id", "image_cache_name", "limit", "tokenizer"} {
		if IsSensitive(k) {
			t.Errorf("%q should NOT be sensitive", k)
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
	for in, want := range cases {
		if got := MaskValue(in); got != want {
			t.Errorf("MaskValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMaskKV(t *testing.T) {
	if got := MaskKV("AccessKeyId", "LTAI5tSecret"); got != "LTAI***" {
		t.Errorf("sensitive kv = %q", got)
	}
	if got := MaskKV("region_id", "cn-hangzhou"); got != "cn-hangzhou" {
		t.Errorf("non-sensitive kv should pass through, got %q", got)
	}
}

func TestMaskBodyRecursive(t *testing.T) {
	// Outer key "config" is deliberately NOT a registered secret, so
	// masking must recurse into it and only hit the leaf secret.
	in := `{"regionId":"cn-hangzhou","config":{"accessKeySecret":"topSecretValue"},"tags":[{"password":"hunter2long"}]}`
	out := MaskBody(in)

	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("masked body not JSON: %v\n%s", err, out)
	}
	if m["regionId"] != "cn-hangzhou" {
		t.Errorf("non-secret changed: %v", m["regionId"])
	}
	cfg := m["config"].(map[string]any)
	if cfg["accessKeySecret"] != "topS***" {
		t.Errorf("nested secret not masked: %v", cfg["accessKeySecret"])
	}
	tags := m["tags"].([]any)
	tag0 := tags[0].(map[string]any)
	if tag0["password"] != "hunt***" {
		t.Errorf("secret in array not masked: %v", tag0["password"])
	}
}

// TestMaskBodyWholeObjectSecret verifies that when a secret key holds a
// non-string (object/array) value, the entire value is replaced with
// "***" rather than recursed into.
func TestMaskBodyWholeObjectSecret(t *testing.T) {
	out := MaskBody(`{"credentials":{"accessKeySecret":"x"}}`)
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if m["credentials"] != "***" {
		t.Fatalf("whole-object secret should be ***, got %#v", m["credentials"])
	}
}

func TestAddCustomField(t *testing.T) {
	// Restore registry after mutating it.
	saved := make([]string, 0)
	mu.RLock()
	for k := range fields {
		saved = append(saved, k)
	}
	mu.RUnlock()
	defer SetFields(saved)

	if IsSensitive("my_custom_secret") {
		t.Fatal("precondition: should not be sensitive yet")
	}
	Add("my_custom_secret")
	if !IsSensitive("MY_CUSTOM_SECRET") {
		t.Fatal("Add did not register the field case-insensitively")
	}
}
