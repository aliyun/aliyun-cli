// Copyright 1999-2019 Alibaba Group Holding Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openapi

import (
	"bytes"
	"encoding/json"

	format "github.com/aliyun/aliyun-cli/humanFormat"
)

// Implement the format of outputformat interface

type TableFormat string

type JSONFormat string

// TODO
// type TextFormat string

type OutputFormat interface {
	Format(apiOrMethod, data string) (string, error)
}

// FormatHandler precess output format
func FormatHandler(format string) OutputFormat {
	switch format {
	case "table":
		return TableFormat("table")
	case "json":
		return JSONFormat("json")
	}
	return nil
}

// Format return table format
func (t TableFormat) Format(apiOrMethod, data string) (string, error) {
	buf := new(bytes.Buffer)
	table := format.NewTable(buf).AddTitle(apiOrMethod)
	format.FromJSON([]byte(data), table)
	table.Flush()
	return buf.String(), nil
}

// Format return json format
func (j JSONFormat) Format(apiOrMethod, data string) (string, error) {
	buf := new(bytes.Buffer)
	err := json.Indent(buf, []byte(data), "", "\t")
	return buf.String(), err
}
