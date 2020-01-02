// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
	jmespath "github.com/jmespath/go-jmespath"
)

func NewOutputFlag() *cli.Flag {
	return &cli.Flag{
		Name:         OutputFlagName,
		Shorthand:    'o',
		AssignedMode: cli.AssignedRepeatable,
		Short: i18n.T(
			"use `--output cols=Field1,Field2 [rows=jmesPath]` to print output as table",
			"使用 `--output cols=Field1,Field1 [rows=jmesPath]` 使用表格方式打印输出",
		),
		Long: i18n.T(
			"",
			"",
		),
		Fields: []cli.Field{
			{Key: "cols", Repeatable: false, Required: true},
			{Key: "rows", Repeatable: false, Required: false},
			{Key: "num", Repeatable: false, Required: false},
		},
	}
}

type OutputFilter interface {
	FilterOutput(input string) (string, error)
}

func GetOutputFilter(ctx *cli.Context) OutputFilter {
	if !OutputFlag(ctx.Flags()).IsAssigned() {
		return nil
	}
	return NewTableOutputFilter(ctx)
}

type TableOutputFilter struct {
	ctx *cli.Context
}

func NewTableOutputFilter(ctx *cli.Context) OutputFilter {
	return &TableOutputFilter{ctx: ctx}
}

func (a *TableOutputFilter) FilterOutput(s string) (string, error) {
	var v interface{}
	s = fmt.Sprintf("{\"RootFilter\":[%s]}", s)
	decoder := json.NewDecoder(bytes.NewBufferString(s))
	decoder.UseNumber()
	err := decoder.Decode(&v)
	if err != nil {
		return s, fmt.Errorf("unmarshal output failed %s", err)
	}

	rowPath := detectArrayPath(v)
	if v, ok := OutputFlag(a.ctx.Flags()).GetFieldValue("rows"); ok {
		rowPath = "RootFilter[0]." + v
	} else {
		rowPath = "RootFilter"
	}

	var colNames []string
	if v, ok := OutputFlag(a.ctx.Flags()).GetFieldValue("cols"); ok {
		v = cli.UnquoteString(v)
		colNames = strings.Split(v, ",")
	} else {
		return s, fmt.Errorf("you need to assign col=col1,col2,... with --output")
	}

	return a.FormatTable(rowPath, colNames, v)
}

func (a *TableOutputFilter) FormatTable(rowPath string, colNames []string, v interface{}) (string, error) {
	// Add row number
	if v, ok := OutputFlag(a.ctx.Flags()).GetFieldValue("num"); ok {
		if v == "true" {
			colNames = append([]string{"Num"}, colNames...)
		}
	}
	rows, err := jmespath.Search(rowPath, v)

	if err != nil {
		return "", fmt.Errorf("jmespath: '%s' failed %s", rowPath, err)
	}

	rowsArray, ok := rows.([]interface{})
	if !ok {
		return "", fmt.Errorf("jmespath: '%s' failed Need Array Expr", rowPath)
	}

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	format := strings.Repeat("%v\t ", len(colNames)-1) + "%v"
	w := tabwriter.NewWriter(writer, 0, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(w, fmt.Sprintf(format, toIntfArray(colNames)...))

	separator := ""
	for i, colName := range colNames {
		separator = separator + strings.Repeat("-", len(colName))
		if i < len(colNames)-1 {
			separator = separator + "\t "
		}
	}
	fmt.Fprintln(w, separator)

	for i, row := range rowsArray {
		rowIntf, ok := row.(interface{})
		if !ok {
			// fmt.Errorf("parse row to interface failed")
		}
		r := make([]string, 0)
		var s string
		var index int
		if v, ok := OutputFlag(a.ctx.Flags()).GetFieldValue("num"); ok {
			if v == "true" {
				s = fmt.Sprintf("%v", i)
				r = append(r, s)
				index = 1
			}
		}
		for _, colName := range colNames[index:] {
			v, _ := jmespath.Search(colName, rowIntf)
			s = fmt.Sprintf("%v", v)
			r = append(r, s)
		}
		fmt.Fprintln(w, fmt.Sprintf(format, toIntfArray(r)...))
	}
	w.Flush()
	writer.Flush()
	return buf.String(), nil
}

func toIntfArray(stringArray []string) []interface{} {
	intfArray := []interface{}{}

	for _, elem := range stringArray {
		intfArray = append(intfArray, elem)
	}
	return intfArray
}
