package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"github.com/jmespath/go-jmespath"
	"bytes"
	"bufio"
)

var OutputFlag = &cli.Flag{
	Name: "output",
	AssignedMode: cli.AssignedRepeatable,
	Short: i18n.T(
		"",
		"",
	),
	Fields: []cli.Field {
		{Key: "cols", Repeatable:false, Required:true},
		{Key: "rows", Repeatable:false, Required:false},
	},
}

type OutputFilter interface {
	FilterOutput(input string) (string, error)
}

//var (
//	processors = []func(ctx *cli.Context, response string) (bool, error){
//		outputTable,
//	}
//)

func NewOutputFilter() (OutputFilter, error) {
	if !OutputFlag.IsAssigned() {
		return nil, nil
	}
	return NewTableOutputFilter()
}

type TableOutputFilter struct {
//	rowExpr string
//	colsName []string
}

func NewTableOutputFilter() (OutputFilter, error) {
	return &TableOutputFilter{}, nil
}

func (a *TableOutputFilter) FilterOutput(s string) (string, error) {
	var v interface{}
	err := json.Unmarshal([]byte(s), &v)
	if err != nil {
		return s, fmt.Errorf("unmarshal output failed %s", err)
	}

	rowPath := detectJmesArrayPath(v)
	if v, ok := OutputFlag.GetFieldValue("rows"); ok {
		rowPath = v
	}

	var colNames []string
	if v, ok := OutputFlag.GetFieldValue("cols"); ok {
		v = cli.UnquoteString(v)
		colNames = strings.Split(v, ",")
	} else {
		return s, fmt.Errorf("you need to assign col=col1,col2,... with --output")
	}

	return a.FormatTable(rowPath, colNames, v)
}

func (a *TableOutputFilter) FormatTable(rowPath string, colNames []string, v interface{}) (string, error) {
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
	format := strings.Repeat("%v\t", len(colNames)-1) + "%v"

	w := tabwriter.NewWriter(writer, 0, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(w, fmt.Sprintf(format, toIntfArray(colNames)...))
	separator := "-----------------"
	fmt.Fprintln(w, strings.Repeat(separator+"\t", len(colNames)-1)+separator)


	for _, row := range rowsArray {
		rowIntf, ok := row.(interface{})
		if !ok {
			fmt.Errorf("parse row to interface failed")
		}
		r := make([]string, 0)
		for _, colName := range colNames {
			v, _ := jmespath.Search(colName, rowIntf)
			s := fmt.Sprintf("%v", v)
			r = append(r, s)
		}
		fmt.Fprintln(w, fmt.Sprintf(format, toIntfArray(r)...))
	}
	w.Flush()
	writer.Flush()
	return buf.String(), nil
}





//
//func outputProcessor(ctx *cli.Context, response string) error {
//
//	for _, processor := range processors {
//		ok, err := processor(ctx, response)
//		if !ok {
//			continue
//		}
//		return err
//	}
//
//	fmt.Println(response)
//	return nil
//}
//
//func outputTable(ctx *cli.Context, response string) (bool, error) {
//	rowsFlag := ctx.Flags().Get(OutputTableRowFlag.Name, OutputTableRowFlag.Shorthand)
//	colsFlag := ctx.Flags().Get(OutputTableColsFlag.Name, OutputTableColsFlag.Shorthand)
//
//	if (!rowsFlag.IsAssigned()) && (!colsFlag.IsAssigned()) {
//		return false, nil
//	}
//
//	if !rowsFlag.IsAssigned() {
//		return true, fmt.Errorf("Need %s", flagOutputTableRows)
//	}
//
//	if !colsFlag.IsAssigned() {
//		return true, fmt.Errorf("Need %s", flagOutputTableCols)
//	}
//
//	var v interface{}
//	err := json.Unmarshal([]byte(response), &v)
//
//	if err != nil {
//		return true, err
//	}
//
//	expr := rowsFlag.GetValue()
//	rowsIntf, err := jmespath.Search(expr, v)
//
//	if err != nil {
//		return true, fmt.Errorf("jmespath: '%s' failed %s", expr, err)
//	}
//
//	rowsArray, ok := rowsIntf.([]interface{})
//
//	if !ok {
//		return true, fmt.Errorf("jmespath: '%s' failed Need Array Expr", expr)
//	}
//
//	colNames := strings.Split(colsFlag.GetValue(), ",")
//
//	if len(colNames) == 0 {
//		return true, fmt.Errorf("%s field %s error", flagOutputTableCols, colsFlag.GetValue())
//	}
//
//	format := strings.Repeat("%v\t", len(colNames)-1) + "%v"
//
//	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)
//
//	fmt.Fprintln(w, fmt.Sprintf(format, toIntfArray(colNames)...))
//	separator := "-----------------"
//	fmt.Fprintln(w, strings.Repeat(separator+"\t", len(colNames)-1)+separator)
//	for _, rowIntf := range rowsArray {
//		rowArray, ok := rowIntf.([]interface{})
//		if !ok {
//			continue
//		}
//		fmt.Fprintln(w, fmt.Sprintf(format, rowArray...))
//	}
//	w.Flush()
//	return true, nil
//}
//
func toIntfArray(stringArray []string) []interface{} {
	intfArray := []interface{}{}

	for _, elem := range stringArray {
		intfArray = append(intfArray, elem)
	}
	return intfArray
}
