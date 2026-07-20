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
	})
	LogResponse(&Response{
		StatusCode: 200,
		Raw:        []byte(`{"ok":true}`),
		Parsed:     map[string]any{"ok": true},
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
}
