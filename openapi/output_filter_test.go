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
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"

	"bufio"
	"testing"
)

func TestNewTableOutputFilter(t *testing.T) {
	w := new(bufio.Writer)
	stderr := new(bufio.Writer)
	ctx := cli.NewCommandContext(w, stderr)
	outflag := NewOutputFlag()
	ctx.Flags().Add(outflag)
	out := GetOutputFilter(ctx)
	assert.Nil(t, out)

	OutputFlag(ctx.Flags()).SetAssigned(true)
	out = GetOutputFilter(ctx)
	assert.NotNil(t, out)

	tableout, ok := out.(*TableOutputFilter)
	assert.True(t, ok)
	assert.NotNil(t, tableout)

	content := `test`
	str, err := tableout.FilterOutput(content)
	assert.Equal(t, "{\"RootFilter\":[test]}", str)
	assert.NotNil(t, err)
	assert.Equal(t, "unmarshal output failed invalid character 'e' in literal true (expecting 'r')", err.Error())

	content = `{"path":"/User"}`
	OutputFlag(tableout.ctx.Flags()).Fields = []cli.Field{
		{
			Key: "rows",
		},
		{
			Key: "cols",
		},
	}
	_, err = tableout.FilterOutput(content)
	assert.NotNil(t, `{"path":"/User"}`, err)
	assert.Equal(t, "you need to assign col=col1,col2,... with --output", err.Error())

	content = `{"path":"/User"}`
	OutputFlag(tableout.ctx.Flags()).SetAssigned(true)
	OutputFlag(tableout.ctx.Flags()).Fields[0].SetAssigned(true)
	OutputFlag(tableout.ctx.Flags()).Fields[1].SetAssigned(true)
	_, err = tableout.FilterOutput(content)
	assert.NotNil(t, `{"path":"/User"}`, err)
	assert.Equal(t, "jmespath: 'RootFilter[0].' failed SyntaxError: Expected identifier, lbracket, or lbrace", err.Error())
}

func TestTableOutputFilter_FormatTable(t *testing.T) {
	w := new(bufio.Writer)
	stderr := new(bufio.Writer)
	ctx := cli.NewCommandContext(w, stderr)
	ctx.Flags().Add(NewOutputFlag())
	out := NewTableOutputFilter(ctx)
	tableout, ok := out.(*TableOutputFilter)
	assert.True(t, ok)
	assert.NotNil(t, tableout)

	rowpath := "/User"
	colNames := []string{"name", "type", "api"}
	str, err := tableout.FormatTable(rowpath, colNames, "")
	assert.Equal(t, "", str)
	assert.NotNil(t, err)
	assert.Equal(t, "jmespath: '/User' failed SyntaxError: Unknown char: '/'", err.Error())

	rowpath = "User"
	v := map[string]interface{}{
		"User": "test",
	}
	str, err = tableout.FormatTable(rowpath, colNames, v)
	assert.Equal(t, "", str)
	assert.NotNil(t, err)
	assert.Equal(t, "jmespath: 'User' failed Need Array Expr", err.Error())

	v = map[string]interface{}{
		"User": []interface{}{"test", "test"},
	}
	str, err = tableout.FormatTable(rowpath, colNames, v)
	assert.Equal(t, "name  | type  | api\n----  | ----  | ---\n<nil> | <nil> | <nil>\n<nil> | <nil> | <nil>\n", str)
	assert.Nil(t, err)

	ctx.Flags().Get("output").Fields[2].SetAssigned(true)
	ctx.Flags().Get("output").Fields[2].SetValue("true")
	str, err = tableout.FormatTable(rowpath, colNames, v)
	assert.Equal(t, "Num | name  | type  | api\n--- | ----  | ----  | ---\n0   | <nil> | <nil> | <nil>\n1   | <nil> | <nil> | <nil>\n", str)
	assert.Nil(t, err)

	// test array format
	v = map[string]interface{}{
		"User": []interface{}{
			[]string{"test", "test2"},
			[]string{"test3", "test4"},
		},
	}
	colNames = []string{"name:0", "type:1"}
	str, err = tableout.FormatTable(rowpath, colNames, v)
	assert.Equal(t, "Num | name  | type\n--- | ----  | ----\n0   | test  | test2\n1   | test3 | test4\n", str)
	assert.Nil(t, err)
	// test array format
	colNames = []string{"name", "type:1"}
	str, err = tableout.FormatTable(rowpath, colNames, v)
	assert.NotNil(t, err)
	assert.Equal(t, "colNames: name must be string:number format, like 'name:0', 0 is the array index", err.Error())
}

func TestUnquoteString(t *testing.T) {
	str := UnquoteString(`"nicai"`)
	assert.Equal(t, "nicai", str)
}
