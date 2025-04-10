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
package openapi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
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

	var rowPath string
	if v, ok := OutputFlag(a.ctx.Flags()).GetFieldValue("rows"); ok {
		rowPath = "RootFilter[0]." + v
	} else {
		rowPath = "RootFilter"
	}

	var colNames []string
	if v, ok := OutputFlag(a.ctx.Flags()).GetFieldValue("cols"); ok {
		v = UnquoteString(v)
		colNames = strings.Split(v, ",")
	} else {
		return s, fmt.Errorf("you need to assign col=col1,col2,... with --output")
	}

	return a.FormatTable(rowPath, colNames, v)
}

func isArrayOrSlice(value interface{}) bool {
	v := reflect.ValueOf(value)
	return v.Kind() == reflect.Array || v.Kind() == reflect.Slice
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

	// delete date type is struct
	// 1 = object, 2 = array
	dataType := 1
	if len(rowsArray) > 0 {
		_, ok := rowsArray[0].(map[string]interface{})
		if !ok {
			// check if it is an array
			if isArrayOrSlice(rowsArray[0]) {
				dataType = 2
			}
		}
	}

	colNamesArray := make([]string, 0)
	colIndexArray := make([]int, 0)

	if dataType == 2 {
		// all colNames must be string:number format
		for _, colName := range colNames {
			// Num ignore
			if colName == "Num" {
				colNamesArray = append(colNamesArray, colName)
				continue
			}
			if !strings.Contains(colName, ":") {
				return "", fmt.Errorf("colNames: %s must be string:number format, like 'name:0', 0 is the array index", colName)
			}
			// split colName to name and number, must be two parts
			parts := strings.Split(colName, ":")
			if len(parts) != 2 {
				return "", fmt.Errorf("colNames: %s must be string:number format, like 'name:0', 0 is the array index", colName)
			}
			// check if number is a number, use regex match
			if !isNumber(parts[1]) {
				return "", fmt.Errorf("colNames: %s must be string:number format, like 'name:0', 0 is the array index", colName)
			}
			colNamesArray = append(colNamesArray, parts[0])
			num, err := strconv.Atoi(parts[1])
			if err != nil {
				return "", fmt.Errorf("colNames: %s must be string:number format, like 'name:0', 0 is the array index", colName)
			}
			colIndexArray = append(colIndexArray, num)
		}
	}

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	format := strings.Repeat("%v\t ", len(colNames)-1) + "%v"
	w := tabwriter.NewWriter(writer, 0, 0, 1, ' ', tabwriter.Debug)
	if dataType == 1 {
		fmt.Fprintln(w, fmt.Sprintf(format, toIntfArray(colNames)...))

		separator := ""
		for i, colName := range colNames {
			separator = separator + strings.Repeat("-", len(colName))
			if i < len(colNames)-1 {
				separator = separator + "\t "
			}
		}

		fmt.Fprintln(w, separator)
	} else {
		fmt.Fprintln(w, fmt.Sprintf(format, toIntfArray(colNamesArray)...))

		separator := ""
		for i, colNameArray := range colNamesArray {
			separator = separator + strings.Repeat("-", len(colNameArray))
			if i < len(colNamesArray)-1 {
				separator = separator + "\t "
			}
		}

		fmt.Fprintln(w, separator)
	}

	for i, row := range rowsArray {
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
		if dataType == 1 {
			for _, colName := range colNames[index:] {
				v, _ := jmespath.Search(colName, row)
				s = fmt.Sprintf("%v", v)
				r = append(r, s)
			}
		} else {
			for _, colIndex := range colIndexArray {
				v, _ := jmespath.Search(fmt.Sprintf("[%d]", colIndex), row)
				s = fmt.Sprintf("%v", v)
				r = append(r, s)
			}
		}
		fmt.Fprintln(w, fmt.Sprintf(format, toIntfArray(r)...))
	}
	w.Flush()
	writer.Flush()
	return buf.String(), nil
}

func isNumber(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true

}

func toIntfArray(stringArray []string) []interface{} {
	intfArray := []interface{}{}

	for _, elem := range stringArray {
		intfArray = append(intfArray, elem)
	}
	return intfArray
}

func UnquoteString(s string) string {
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") && len(s) >= 2 {
		return s[1 : len(s)-1]
	}
	return s
}
