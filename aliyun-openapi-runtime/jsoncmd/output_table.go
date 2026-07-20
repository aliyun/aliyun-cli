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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	jmespath "github.com/jmespath/go-jmespath"
)

// renderOutputTable renders --output cols=... [rows=] [num=].
func renderOutputTable(w io.Writer, data any, raw []byte, cfg *argparser.OutputTableConfig) error {
	if cfg == nil || len(cfg.Cols) == 0 {
		return fmt.Errorf("you need to specify columns with --output cols=col1,col2,...")
	}
	if data == nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, &data); err != nil {
			return fmt.Errorf("unmarshal output failed: %w", err)
		}
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// Wrap like the plugin so rows defaults to the root document.
	wrapped := fmt.Sprintf(`{"RootFilter":[%s]}`, string(b))
	var root any
	dec := json.NewDecoder(bytes.NewBufferString(wrapped))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return fmt.Errorf("unmarshal output failed: %w", err)
	}

	rowPath := "RootFilter"
	if cfg.Rows != "" {
		rowPath = "RootFilter[0]." + cfg.Rows
	}
	rows, err := jmespath.Search(rowPath, root)
	if err != nil {
		return fmt.Errorf("jmespath '%s' failed: %w", rowPath, err)
	}
	rowsArray, ok := rows.([]any)
	if !ok {
		return fmt.Errorf("jmespath '%s' failed: need array expression", rowPath)
	}

	colNames := append([]string{}, cfg.Cols...)
	if cfg.ShowNum {
		colNames = append([]string{"Num"}, colNames...)
	}

	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	tw := tabwriter.NewWriter(bw, 0, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(tw, strings.Join(colNames, "\t "))
	sep := make([]string, len(colNames))
	for i, c := range colNames {
		sep[i] = strings.Repeat("-", len(c))
	}
	fmt.Fprintln(tw, strings.Join(sep, "\t "))

	for i, row := range rowsArray {
		cells := make([]string, 0, len(colNames))
		if cfg.ShowNum {
			cells = append(cells, strconv.Itoa(i))
		}
		for _, col := range cfg.Cols {
			val, err := jmespath.Search(col, row)
			if err != nil {
				cells = append(cells, "")
				continue
			}
			cells = append(cells, formatTableCell(val))
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t "))
	}
	_ = tw.Flush()
	_ = bw.Flush()
	fmt.Fprint(w, buf.String())
	return nil
}

func formatTableCell(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	}
}
