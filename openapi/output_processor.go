package openapi

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/jmespath/go-jmespath"
	"os"
	"strings"
	"text/tabwriter"
)

var (
	processors = []func(ctx *cli.Context, response string) (bool, error){
		outputTable,
	}
)

func outputProcessor(ctx *cli.Context, response string) error {

	for _, processor := range processors {
		ok, err := processor(ctx, response)
		if !ok {
			continue
		}
		return err
	}

	fmt.Println(response)
	return nil
}

func outputTable(ctx *cli.Context, response string) (bool, error) {
	rowsFlag := ctx.Flags().Get(OutputTableRowFlag.Name, OutputTableRowFlag.Shorthand)
	colsFlag := ctx.Flags().Get(OutputTableColsFlag.Name, OutputTableColsFlag.Shorthand)

	if (!rowsFlag.IsAssigned()) && (!colsFlag.IsAssigned()) {
		return false, nil
	}

	if !rowsFlag.IsAssigned() {
		return true, fmt.Errorf("Need %s", flagOutputTableRows)
	}

	if !colsFlag.IsAssigned() {
		return true, fmt.Errorf("Need %s", flagOutputTableCols)
	}

	var v interface{}
	err := json.Unmarshal([]byte(response), &v)

	if err != nil {
		return true, err
	}

	expr := rowsFlag.GetValue()
	rowsIntf, err := jmespath.Search(expr, v)

	if err != nil {
		return true, fmt.Errorf("jmespath: '%s' failed %s", expr, err)
	}

	rowsArray, ok := rowsIntf.([]interface{})

	if !ok {
		return true, fmt.Errorf("jmespath: '%s' failed Need Array Expr", expr)
	}

	colNames := strings.Split(colsFlag.GetValue(), ",")

	if len(colNames) == 0 {
		return true, fmt.Errorf("%s field %s error", flagOutputTableCols, colsFlag.GetValue())
	}

	format := strings.Repeat("%v\t", len(colNames)-1) + "%v"

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)

	fmt.Fprintln(w, fmt.Sprintf(format, toIntfArray(colNames)...))
	separator := "-----------------"
	fmt.Fprintln(w, strings.Repeat(separator+"\t", len(colNames)-1)+separator)
	for _, rowIntf := range rowsArray {
		rowArray, ok := rowIntf.([]interface{})
		if !ok {
			continue
		}
		fmt.Fprintln(w, fmt.Sprintf(format, rowArray...))
	}
	w.Flush()
	return true, nil
}

func toIntfArray(stringArray []string) []interface{} {
	intfArray := []interface{}{}

	for _, elem := range stringArray {
		intfArray = append(intfArray, elem)
	}
	return intfArray
}



//func outputTable1(flag *cli.Flag, response string) error {
//
//	var v interface{}
//	err := json.Unmarshal([]byte(response), &v)
//
//	if err != nil {
//		return err
//	}
//
//	targets := strings.Split(flag.GetValue(), ",")
//	cols := columnFinder(targets, v)
//
//	if len(cols) == 0 {
//		return fmt.Errorf("%s field %s error", flagOutputTableRow, flag.GetValue())
//	}
//
//	format := strings.Repeat("%v\t", len(cols[0]) - 1) + "%v"
//
//	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)
//
//	fmt.Fprintln(w, fmt.Sprintf(format, toIntfArray(targets)...))
//	separator := "-----------------"
//	fmt.Fprintln(w, strings.Repeat(separator + "\t", len(cols[0]) - 1) + separator)
//	for _, col := range cols {
//		fmt.Fprintln(w, fmt.Sprintf(format, col...))
//	}
//	w.Flush()
//
//	return nil
//}
//
//func isBaseType(v interface{}) bool {
//	switch v.(type) {
//	case map[string]interface{}:
//		return false
//	case []interface{}:
//		return false
//	}
//
//	return true
//}
//
//func columnFinder(targets []string, v interface{}) [][]interface{} {
//	var outputTable [][]interface{}
//	switch v1 := v.(type) {
//	case map[string]interface{}:
//		row := []interface{}{}
//		for _, target := range targets {
//			colValue, ok := v1[target]
//			if ok && isBaseType(colValue) {
//				row = append(row, colValue)
//			}
//		}
//
//		if len(row) == len(targets) {
//			outputTable = append(outputTable, row)
//			return outputTable
//		}
//
//		for _, intf := range v1 {
//			outputTable = append(outputTable, columnFinder(targets, intf)...)
//		}
//
//	case []interface{}:
//		for _, intf := range v1{
//			outputTable = append(outputTable, columnFinder(targets, intf)...)
//		}
//	}
//
//	return outputTable
//}

