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

package engine

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
)

func TestRenderOutputTableValidation(t *testing.T) {
	tests := []struct {
		name    string
		data    any
		raw     []byte
		cfg     *argparser.OutputTableConfig
		wantErr string
	}{
		{name: "nil config", wantErr: "specify columns"},
		{name: "empty columns", cfg: &argparser.OutputTableConfig{}, wantErr: "specify columns"},
		{name: "invalid raw json", raw: []byte("{"), cfg: &argparser.OutputTableConfig{Cols: []string{"Name"}}, wantErr: "unmarshal output failed"},
		{name: "unsupported data", data: make(chan int), cfg: &argparser.OutputTableConfig{Cols: []string{"Name"}}, wantErr: "unsupported type"},
		{name: "invalid rows expression", data: map[string]any{"Items": []any{}}, cfg: &argparser.OutputTableConfig{Cols: []string{"Name"}, Rows: "["}, wantErr: "jmespath"},
		{name: "rows must be array", data: map[string]any{"Item": map[string]any{"Name": "demo"}}, cfg: &argparser.OutputTableConfig{Cols: []string{"Name"}, Rows: "Item"}, wantErr: "need array expression"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := renderOutputTable(&bytes.Buffer{}, tc.data, tc.raw, tc.cfg)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("renderOutputTable() error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestRenderOutputTableRowsNumbersAndCells(t *testing.T) {
	raw := []byte(`{"Items":[{"Name":"first","Count":2,"Ready":true,"Missing":null,"Labels":{"env":"test"}},{"Name":"second","Count":3,"Ready":false}]}`)
	cfg := &argparser.OutputTableConfig{
		Cols:    []string{"Name", "Count", "Ready", "Missing", "Labels", "["},
		Rows:    "Items",
		ShowNum: true,
	}
	var out bytes.Buffer
	if err := renderOutputTable(&out, nil, raw, cfg); err != nil {
		t.Fatalf("renderOutputTable() error = %v", err)
	}
	text := out.String()
	for _, want := range []string{"Num", "Name", "first", "second", "true", `{"env":"test"}`} {
		if !strings.Contains(text, want) {
			t.Fatalf("table output %q does not contain %q", text, want)
		}
	}

	if got := formatTableCell(nil); got != "" {
		t.Fatalf("formatTableCell(nil) = %q", got)
	}
	if got := formatTableCell("text"); got != "text" {
		t.Fatalf("formatTableCell(string) = %q", got)
	}
	if got := formatTableCell(json.Number("9007199254740993")); got != "9007199254740993" {
		t.Fatalf("formatTableCell(number) = %q", got)
	}
	if got := formatTableCell(false); got != "false" {
		t.Fatalf("formatTableCell(bool) = %q", got)
	}
	if got := formatTableCell(make(chan int)); !strings.HasPrefix(got, "0x") {
		t.Fatalf("formatTableCell(channel) = %q", got)
	}
}
