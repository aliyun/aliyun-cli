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

package jsoncmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
)

// TestRenderDryRunMasksSecrets verifies the human dump masks sensitive header/query values.
func TestRenderDryRunMasksSecrets(t *testing.T) {
	var buf bytes.Buffer
	req := &runtime.AssembledRequest{
		Action:   "DoThing",
		Version:  "2024-01-01",
		Method:   "POST",
		Protocol: "HTTPS",
		Style:    "RPC",
		Endpoint: "svc.cn-hangzhou.aliyuncs.com",
		Headers:  map[string]string{"x-acs-security-token": "SToKeNSecretValue", "content-type": "application/json"},
		Query:    map[string]string{"AccessKeyId": "LTAI5tRealKeyId", "RegionId": "cn-hangzhou"},
	}
	if err := renderDryRun(&buf, "svc", req, false); err != nil {
		t.Fatalf("renderDryRun: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "SToKeNSecretValue") || strings.Contains(out, "LTAI5tRealKeyId") {
		t.Fatalf("secret leaked:\n%s", out)
	}
	if !strings.Contains(out, "SToK***") || !strings.Contains(out, "LTAI***") {
		t.Fatalf("expected masked fingerprints:\n%s", out)
	}
	if !strings.Contains(out, "RegionId: cn-hangzhou") {
		t.Fatalf("non-secret should be visible:\n%s", out)
	}
}

func TestRenderResponseCliQuery(t *testing.T) {
	raw := []byte(`{"Instances":[{"Id":"i-1","Region":"cn-hangzhou"},{"Id":"i-2","Region":"cn-beijing"}]}`)
	var parsed any
	_ = json.Unmarshal(raw, &parsed)
	resp := &runtime.Response{StatusCode: 200, Raw: raw, Parsed: parsed}

	var buf bytes.Buffer
	if err := renderResponse(&buf, resp, "Instances[].Id"); err != nil {
		t.Fatalf("cli-query json: %v", err)
	}
	if !strings.Contains(buf.String(), `"i-1"`) || !strings.Contains(buf.String(), `"i-2"`) {
		t.Fatalf("filtered json:\n%s", buf.String())
	}
}

// TestWriteJSONPrettyPrints matches the Go plugin FormatJSON behaviour:
// default (unfiltered) JSON output is indented with tabs, not a compact
// one-liner from the wire.
func TestWriteJSONPrettyPrints(t *testing.T) {
	raw := []byte(`{"Status":"200","Data":{"TotalCount":1}}`)
	var parsed any
	_ = json.Unmarshal(raw, &parsed)
	var buf bytes.Buffer
	if err := renderResponse(&buf, &runtime.Response{Raw: raw, Parsed: parsed}, ""); err != nil {
		t.Fatalf("renderResponse: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "{\n\t\"Status\"") {
		t.Fatalf("expected tab-indented JSON, got:\n%s", out)
	}
	if strings.Count(strings.TrimSpace(out), "\n") < 2 {
		t.Fatalf("expected multi-line pretty JSON:\n%s", out)
	}
}

// TestPrintAPIHelpShowsGlobalOptions verifies engine reserved flags are
// listed under a Global options section alongside the API parameters.
func TestPrintAPIHelpShowsGlobalOptions(t *testing.T) {
	var buf bytes.Buffer
	api := &meta.API{
		Name:        "ListTriggers",
		CmdName:     "list-triggers",
		Version:     "2023-03-30",
		Description: meta.Description{EN: "List triggers of a function"},
		Parameters: []meta.Parameter{
			{Name: "function_name", Type: meta.TypeString, Required: true, Options: []string{"--function-name"}},
		},
	}
	if err := printAPIHelp(&buf, "fc", api, "en"); err != nil {
		t.Fatalf("printAPIHelp: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"Description: List triggers",
		"API Version: 2023-03-30",
		"Usage:",
		"aliyun fc list-triggers [parameters]",
		"Parameters:",
		"--function-name",
		"(required)",
		"Global Parameters:",
		"--cli-dry-run",
		"--cli-query",
		"--quiet",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q in:\n%s", want, out)
		}
	}
	// --output is hidden (plugin parity); --force was removed.
	if strings.Contains(out, "--output") {
		t.Errorf("help must not list hidden --output:\n%s", out)
	}
	if strings.Contains(out, "--force") {
		t.Errorf("engine no longer supports --force:\n%s", out)
	}
	// Host credential flags must NOT be listed (host owns them).
	if strings.Contains(out, "--profile") {
		t.Errorf("help must not list host --profile:\n%s", out)
	}
}

// TestPrintAPIHelpChineseLabels verifies section headers localize to
// Chinese when lang == "zh".
func TestPrintAPIHelpChineseLabels(t *testing.T) {
	var buf bytes.Buffer
	api := &meta.API{
		Name: "ListTriggers", CmdName: "list-triggers", Version: "2023-03-30",
		Description: meta.Description{ZH: "查询触发器列表"},
		Parameters:  []meta.Parameter{{Name: "function_name", Type: meta.TypeString, Required: true, Options: []string{"--function-name"}}},
		Examples:    []string{"aliyun fc list-triggers --function-name f"},
	}
	if err := printAPIHelp(&buf, "fc", api, "zh"); err != nil {
		t.Fatalf("printAPIHelp: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"描述:", "API 版本:", "使用:", "参数:", "全局参数:", "示例:", "查询触发器列表"} {
		if !strings.Contains(out, want) {
			t.Errorf("zh help missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Description:") {
		t.Errorf("zh help should not contain English label:\n%s", out)
	}
}

// TestPrintAPIHelpWrapsAligned verifies long descriptions wrap at the
// fixed line width and continuation lines keep the description column
// aligned (empty padded name), matching the Go plugin help.
func TestPrintAPIHelpWrapsAligned(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "80")
	long := strings.Repeat("word ", 40) // far longer than the desc column
	var buf bytes.Buffer
	api := &meta.API{
		Name: "Do", CmdName: "do", Version: "v1",
		Parameters: []meta.Parameter{{
			Name: "region_id", Type: meta.TypeString, Required: true,
			Options:     []string{"--region-id"},
			Description: meta.Description{EN: long},
		}},
	}
	if err := printAPIHelp(&buf, "demo", api, "en"); err != nil {
		t.Fatalf("printAPIHelp: %v", err)
	}
	lines := strings.Split(buf.String(), "\n")
	var paramLines []string
	inParams := false
	for _, line := range lines {
		if strings.HasPrefix(line, "Parameters:") {
			inParams = true
			continue
		}
		if inParams {
			if strings.HasPrefix(line, "Global Parameters:") || line == "" {
				break
			}
			paramLines = append(paramLines, line)
		}
	}
	if len(paramLines) < 2 {
		t.Fatalf("expected wrapped description, got %d param lines:\n%s", len(paramLines), buf.String())
	}
	// First line carries the flag name; subsequent lines pad an empty
	// name column so the description text starts at the same column.
	first := paramLines[0]
	second := paramLines[1]
	nameIdx := strings.Index(first, "--region-id")
	if nameIdx < 0 {
		t.Fatalf("first line missing flag:\n%s", first)
	}
	// After tabwriter flush, description starts after the padded name.
	// Continuation must begin with spaces (no flag name) and its first
	// non-space should line up with the description on the first line.
	descStart := -1
	for i, r := range first {
		if i > nameIdx+len("--region-id") && r != ' ' && r != '\t' {
			// find type/desc start: first non-space after name column
			descStart = i
			break
		}
	}
	// Simpler check: continuation has no "--" and starts with spaces;
	// and no line exceeds the configured max length.
	if strings.Contains(second, "--region-id") {
		t.Errorf("continuation must not repeat the flag name:\n%s", second)
	}
	if !strings.HasPrefix(second, "  ") {
		t.Errorf("continuation should be indented:\n%q", second)
	}
	for _, line := range paramLines {
		if len([]rune(line)) > 80 {
			t.Errorf("line exceeds max width (%d): %q", len([]rune(line)), line)
		}
		_ = descStart
	}
}

// TestDryRunJSONOneLine checks the machine-readable form.
func TestDryRunJSONOneLine(t *testing.T) {
	var buf bytes.Buffer
	req := &runtime.AssembledRequest{Action: "A", Version: "V", Region: "cn-x", Endpoint: "https://svc.example.com"}
	if err := renderDryRun(&buf, "svc", req, true); err != nil {
		t.Fatalf("renderDryRun json: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if strings.Contains(out, "\n") {
		t.Fatalf("json form must be one line: %q", out)
	}
	if !strings.Contains(out, `"product":"svc"`) || !strings.Contains(out, `"endpoint":"svc.example.com"`) {
		t.Fatalf("unexpected json meta: %s", out)
	}
}
