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

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestInitLoggerDebug(t *testing.T) {
	t.Cleanup(ResetLoggerForTest)
	ResetLoggerForTest()

	InitLogger("DEBUG", false)
	if !IsDebugEnabled() {
		t.Fatal("DEBUG should enable debug logs")
	}

	var buf bytes.Buffer
	SetLoggerOutputForTest(&buf)
	Debug("hello %s", "world")
	if !strings.Contains(buf.String(), "hello world") {
		t.Fatalf("expected debug output, got %q", buf.String())
	}
}

func TestInitLoggerDryRunSkips(t *testing.T) {
	t.Cleanup(ResetLoggerForTest)
	ResetLoggerForTest()
	InitLogger("DEBUG", true)
	if IsDebugEnabled() {
		t.Fatal("dry-run must not apply --log-level (plugin parity)")
	}
}

func TestInitLoggerNamedPresets(t *testing.T) {
	t.Cleanup(ResetLoggerForTest)
	cases := []struct {
		name  string
		debug bool
	}{
		{"info", false},
		{"ERROR", false},
		{"debug", true},
		{"verbose", true},
	}
	for _, tc := range cases {
		ResetLoggerForTest()
		InitLogger(tc.name, false)
		if got := IsDebugEnabled(); got != tc.debug {
			t.Fatalf("%s: IsDebugEnabled=%v want %v", tc.name, got, tc.debug)
		}
	}
}

func TestLogRequestResponse(t *testing.T) {
	t.Cleanup(ResetLoggerForTest)
	ResetLoggerForTest()
	InitLogger("DEBUG", false)

	var buf bytes.Buffer
	SetLoggerOutputForTest(&buf)

	LogRequest(&AssembledRequest{
		Method: "POST", Pathname: "/", Version: "v1", Action: "List",
		Protocol: "HTTPS", Style: "RPC", Endpoint: "example.com",
		Query: map[string]string{"PageSize": "10", "password": "secret123"},
		Body:  `{"accessKeySecret":"request-secret","name":"visible"}`,
	})
	LogResponse(&Response{
		StatusCode: 200,
		Raw:        []byte(`{"token":"response-secret","ok":true}`),
	})

	out := buf.String()
	if !strings.Contains(out, "HTTP Request") || !strings.Contains(out, "HTTP Response") {
		t.Fatalf("missing sections:\n%s", out)
	}
	if !strings.Contains(out, "secr***") {
		t.Fatalf("password should be masked:\n%s", out)
	}
	if strings.Contains(out, "secret123") {
		t.Fatalf("raw password leaked:\n%s", out)
	}
	for _, secret := range []string{"request-secret", "response-secret"} {
		if strings.Contains(out, secret) {
			t.Fatalf("raw body secret %q leaked:\n%s", secret, out)
		}
	}
	if !strings.Contains(out, "requ***") || !strings.Contains(out, "resp***") {
		t.Fatalf("request or response body was not masked:\n%s", out)
	}
}

func TestApplyNamedConfigAliases(t *testing.T) {
	t.Cleanup(ResetLoggerForTest)
	tests := []struct {
		name  string
		level LogLevel
	}{
		{"production", LogError}, {"prod", LogError}, {"ERROR", LogError},
		{"development", LogInfo}, {"dev", LogInfo}, {"info", LogInfo},
		{"debug", LogDebug}, {"verbose", LogDebug}, {"DEBUG", LogDebug},
		{"quiet", LogFatal}, {"FATAL", LogFatal},
		{"ci", LogWarn}, {"WARN", LogWarn}, {"warning", LogWarn},
	}
	for _, test := range tests {
		if !applyNamedConfig("  " + test.name + "  ") {
			t.Fatalf("applyNamedConfig(%q) = false", test.name)
		}
		globalLogger.mu.Lock()
		level := globalLogger.level
		globalLogger.mu.Unlock()
		if level != test.level {
			t.Fatalf("applyNamedConfig(%q) level = %v, want %v", test.name, level, test.level)
		}
	}
	if applyNamedConfig("invalid") {
		t.Fatal("applyNamedConfig(invalid) = true")
	}
}

func TestInitLoggerEnvironmentAndInvalidConfig(t *testing.T) {
	t.Cleanup(ResetLoggerForTest)
	ResetLoggerForTest()
	t.Setenv(envLogConfig, "debug")
	InitLogger("ERROR", false)
	if !IsDebugEnabled() {
		t.Fatal("environment config should take precedence")
	}

	var buf bytes.Buffer
	SetLoggerOutputForTest(&buf)
	t.Setenv(envLogConfig, "invalid")
	InitLogger("", false)
	if !strings.Contains(buf.String(), "Invalid log config") {
		t.Fatalf("invalid config warning missing: %q", buf.String())
	}

	t.Setenv(envLogConfig, "")
	before := buf.Len()
	InitLogger("", false)
	if buf.Len() != before {
		t.Fatal("empty log config should be a no-op")
	}
}

func TestLoggerFormattingAndFiltering(t *testing.T) {
	var buf bytes.Buffer
	local := &logger{level: LogInfo, output: &buf, enableTime: true, enableColor: true}
	local.log(LogDebug, "hidden")
	if buf.Len() != 0 {
		t.Fatalf("filtered debug log = %q", buf.String())
	}
	local.log(LogInfo, "hello %s", "world")
	out := buf.String()
	if !strings.Contains(out, "\x1b[32m[INFO ]\x1b[0m hello world") || len(out) < len("2006-01-02 15:04:05") {
		t.Fatalf("formatted log = %q", out)
	}

	buf.Reset()
	local.enableTime = false
	local.enableColor = false
	local.log(LogWarn, "plain")
	if got := buf.String(); got != "[WARN ] plain\n" {
		t.Fatalf("plain log = %q", got)
	}
}

func TestLogArgsMasksAndHandlesMarshalFailure(t *testing.T) {
	t.Cleanup(ResetLoggerForTest)
	ResetLoggerForTest()
	LogArgs(map[string]any{"ignored": true})
	logSection("ignored")

	InitLogger("DEBUG", false)
	var buf bytes.Buffer
	SetLoggerOutputForTest(&buf)
	SetLoggerOutputForTest(nil)
	LogArgs(nil)
	LogArgs(map[string]any{
		"password": "secret-value",
		"token":    123,
		"name":     "visible",
	})
	LogArgs(map[string]any{"bad": make(chan int)})

	out := buf.String()
	if !strings.Contains(out, "Arguments: (empty)") || !strings.Contains(out, "secr***") || strings.Contains(out, "secret-value") {
		t.Fatalf("LogArgs output = %q", out)
	}
	if !strings.Contains(out, `"token":"***"`) || !strings.Contains(out, "error marshaling") {
		t.Fatalf("LogArgs special cases missing: %q", out)
	}
}

func TestLogArgsRedactsNestedValuesWithoutMutation(t *testing.T) {
	t.Cleanup(ResetLoggerForTest)
	ResetLoggerForTest()
	InitLogger("DEBUG", false)
	var out bytes.Buffer
	SetLoggerOutputForTest(&out)
	const secret = "FAKE_QA_ARGUMENT_PASSWORD_20261104"
	quoted, _ := json.Marshal(`{"password":"` + secret + `","name":"visible"}`)
	for _, value := range []any{
		string(quoted) + " trailing",
		`{"password":"` + secret + `","name":"visible"}`,
		`{"password":"` + secret + `",`,
		"\ufeff{\"password\":\"" + secret + "\"}",
		map[string]any{"password": secret, "name": "visible"},
		[]any{map[string]any{"password": secret}},
		json.RawMessage(`{"password":"` + secret + `","password":""}`),
	} {
		out.Reset()
		args := map[string]any{"name": value, "region": "cn-hangzhou"}
		before, _ := json.Marshal(args)
		LogArgs(args)
		after, _ := json.Marshal(args)
		if strings.Contains(out.String(), secret) || !bytes.Equal(before, after) {
			t.Fatalf("unsafe argument log: %s", &out)
		}
		if !strings.Contains(out.String(), "Arguments:") {
			t.Fatal("argument log missing")
		}
	}
	out.Reset()
	LogArgs(map[string]any{"name": "[Important] ordinary text", "password": secret})
	if !strings.Contains(out.String(), "[Important] ordinary text") || !strings.Contains(out.String(), "FAKE***") || strings.Contains(out.String(), secret) {
		t.Fatalf("argument preview regressed: %s", &out)
	}
}

func TestLogRequestOmitsOpaqueBody(t *testing.T) {
	t.Cleanup(ResetLoggerForTest)
	ResetLoggerForTest()
	InitLogger("DEBUG", false)
	var out bytes.Buffer
	SetLoggerOutputForTest(&out)
	const secret = "FAKE_SECRET_20261103"
	quoted, _ := json.Marshal(`{"password":"` + secret + `","name":"visible"}`)
	trailingBody, _ := json.Marshal(map[string]string{"payload": string(quoted) + " trailing", "name": "visible"})
	for _, body := range []any{
		string(trailingBody),
		trailingBody,
		map[string]any{"payload": string(quoted) + " trailing", "name": "visible"},
		"password=" + secret + "&name=%zz",
		"<Password>" + secret + "</Password>",
		"\ufeff<Password>" + secret + "==</Password>",
		[]byte("\ufeff<Password>" + secret + "==</Password>"),
		"\u200b<Password>" + secret + "==</Password>",
		[]byte("\u200b<Password>" + secret + "==</Password>"),
		`{"payload":"\ufeff{\"password\":\"` + secret + `\"}"}`,
		"payload=%EF%BB%BF%7B%22password%22%3A%22" + secret + "%22%7D",
		map[string]any{"payload": "\ufeff{\"password\":\"" + secret + "\"}"},
		`{"password":"` + secret,
		"password: " + secret,
		[]byte(strings.Repeat("x", 1600) + secret),
		[]byte("\x00\xff" + secret),
		json.RawMessage(`{"password":"` + secret + `","password":""}`),
		map[string]any{"config": `{"password":"` + secret},
		map[string]any{"config": []any{`{"password":"` + secret}},
		map[string]any{"config": "\"" + secret},
	} {
		out.Reset()
		var data []byte
		switch value := body.(type) {
		case string:
			data = []byte(value)
		case []byte:
			data = append([]byte(nil), value...)
		default:
			data, _ = json.Marshal(value)
		}
		req := &AssembledRequest{Body: body}
		LogRequest(req)
		want := fmt.Sprintf("<opaque body omitted: %d bytes>", len(data))
		if !strings.Contains(out.String(), want) || strings.Contains(out.String(), secret) {
			t.Fatalf("unsafe request log for %T: %s", body, &out)
		}
		var after []byte
		switch value := req.Body.(type) {
		case string:
			after = []byte(value)
		case []byte:
			after = value
		default:
			after, _ = json.Marshal(value)
		}
		if !bytes.Equal(data, after) {
			t.Fatal("logging changed the request body")
		}
	}
}

func TestLogRequestPreservesSensitiveValuePreview(t *testing.T) {
	t.Cleanup(ResetLoggerForTest)
	ResetLoggerForTest()
	InitLogger("DEBUG", false)
	var out bytes.Buffer
	SetLoggerOutputForTest(&out)
	const secret = "FAKE_SECRET_20261103"
	for _, body := range []any{
		`{"password":"[` + secret + `","name":"visible"}`,
		[]byte(`{"password":"[` + secret + `","name":"visible"}`),
		"password=%5B" + secret + "&name=visible",
		map[string]any{"password": "[" + secret, "name": "visible"},
	} {
		out.Reset()
		before, _ := json.Marshal(body)
		req := &AssembledRequest{Body: body}
		LogRequest(req)
		after, _ := json.Marshal(req.Body)
		if strings.Contains(out.String(), secret) || !strings.Contains(out.String(), "visible") || !strings.Contains(out.String(), "[FAK***") || !bytes.Equal(before, after) {
			t.Fatalf("unsafe or regressed log for %T: %s", body, &out)
		}
	}
}

func TestLogJSONRequestAndResponseBranches(t *testing.T) {
	t.Cleanup(ResetLoggerForTest)
	ResetLoggerForTest()
	InitLogger("DEBUG", false)
	var buf bytes.Buffer
	SetLoggerOutputForTest(&buf)

	logJSON("Object", map[string]any{"name": "value"})
	logJSON("Bad", make(chan int))
	LogRequest(nil)
	LogRequest(&AssembledRequest{
		Method: "POST", Pathname: "/bytes", Headers: map[string]string{"Authorization": "secret-token"},
		Body: []byte(`{"password":"byte-secret"}`),
	})
	LogRequest(&AssembledRequest{Method: "POST", Pathname: "/object", Body: map[string]any{"password": "object-secret"}})

	LogResponse(nil)
	LogResponse(&Response{StatusCode: 204})
	LogResponse(&Response{StatusCode: 200, Headers: map[string][]string{
		"Authorization": {"header-secret"}, "X-Test": {"one", "two"},
	}, Raw: []byte(`{"password":"raw-secret"}`)})
	LogResponse(&Response{StatusCode: 200, Raw: []byte(`{"ok":true}`), Parsed: map[string]any{"token": "parsed-secret"}})
	LogResponse(&Response{StatusCode: 200, Raw: []byte("fallback"), Parsed: make(chan int)})
	logStringMap("Empty", nil)

	out := buf.String()
	for _, want := range []string{"Object:", "failed to marshal", "Body (bytes)", "Body:", "Response Body: (empty)", "<opaque body omitted: 8 bytes>"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	for _, secret := range []string{"secret-token", "byte-secret", "object-secret", "header-secret", "raw-secret", "parsed-secret"} {
		if strings.Contains(out, secret) {
			t.Errorf("secret %q leaked:\n%s", secret, out)
		}
	}
}
