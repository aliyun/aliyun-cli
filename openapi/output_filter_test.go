package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/stretchr/testify/assert"

	"bufio"
	"testing"
)

func TestNewTableOutputFilter(t *testing.T) {
	w := new(bufio.Writer)
	ctx := cli.NewCommandContext(w)
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
	str, err = tableout.FilterOutput(content)
	assert.NotNil(t, `{"path":"/User"}`, err)
	assert.Equal(t, "you need to assign col=col1,col2,... with --output", err.Error())

	content = `{"path":"/User"}`
	OutputFlag(tableout.ctx.Flags()).SetAssigned(true)
	OutputFlag(tableout.ctx.Flags()).Fields[0].SetAssigned(true)
	OutputFlag(tableout.ctx.Flags()).Fields[1].SetAssigned(true)
	str, err = tableout.FilterOutput(content)
	assert.NotNil(t, `{"path":"/User"}`, err)
	assert.Equal(t, "jmespath: '' failed SyntaxError: Incomplete expression", err.Error())
}

func TestTableOutputFilter_FormatTable(t *testing.T) {
	w := new(bufio.Writer)
	ctx := cli.NewCommandContext(w)
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
}
