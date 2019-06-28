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

package format

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

type tableEncoder struct {
	table  *Table
	v      interface{}
	hasRow bool
}

var funcList []func()
var backupFuncList []func()
var count int

// FromJSONFull Convert from json to table format
func FromJSON(data []byte, table *Table) {
	e := new(tableEncoder)
	e.table = table
	var v interface{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		panic(err)
	}
	e.v = v
	funcList = append(funcList, e.marshal)
	for {
		if len(funcList) == 0 && len(backupFuncList) == 0 {
			break
		}
		switch count % 2 {
		case 0:
			for _, v := range funcList {
				v()
			}
			funcList = []func(){}
		case 1:
			for _, v := range backupFuncList {
				v()
			}
			backupFuncList = []func(){}
		}
		count++
	}
}

func (e *tableEncoder) marshal() {
	e.regularJSONObject()
	deleteEmptyTable(e.table)
}

// deleteEmptyTable Delete empty table from bottom to top
func deleteEmptyTable(table *Table) {
	if table.IsEmptyCell() && table.IsEmptySub() {
		table.ParentTable().Remove(table)
	}
	if table.ParentTable() != nil {
		deleteEmptyTable(table.ParentTable())
	}
}

func (e *tableEncoder) regularJSONObject() {
	switch reflect.ValueOf(e.v).Kind() {
	case reflect.Map:
		vMap := e.v.(map[string]interface{})
		var keySlice []string
		for k := range vMap {
			keySlice = append(keySlice, k)
		}
		sort.Strings(keySlice)
		for _, k := range keySlice {
			v := vMap[k]
			reflectValue := reflect.ValueOf(v)
			if reflectValue.String() == "" {
				e.table.AddRow(fmt.Sprintf("%s\t", k))
			}
			kind := reflectValue.Kind()
			if kind != reflect.Map && kind != reflect.Slice {
				e.table.AddRow(fmt.Sprintf("%s\t%v", k, v))
				e.hasRow = true
				continue
			}
			subTable := e.table.AddNewTable(e.table.w).AddTitle(k)
			subE := new(tableEncoder)
			subE.table = subTable
			subE.v = v
			switch count % 2 {
			case 0:
				backupFuncList = append(backupFuncList, subE.marshal)
			case 1:
				funcList = append(funcList, subE.marshal)
			}
		}
	case reflect.Slice:
		if len(e.v.([]interface{})) == 0 {
			return
		}
		var title string
		if e.table.title != nil {
			title = string(*e.table.title)
		} else {
			title = string(*e.table.ParentTable().title)
		}
		e.table.ParentTable().Remove(e.table)

		e.table = e.table.ParentTable()

		for _, v := range e.v.([]interface{}) {
			kind := reflect.TypeOf(v).Kind()
			if kind != reflect.Map && kind != reflect.Slice {
				e.table.AddRow(fmt.Sprintf("%s\t%v", string(title), v))
				continue
			}
			subTable := e.table.AddNewTable(e.table.w).AddTitle(string(title))
			subE := new(tableEncoder)
			subE.table = subTable
			subE.v = v
			switch count % 2 {
			case 0:
				backupFuncList = append(backupFuncList, subE.marshal)
			case 1:
				funcList = append(funcList, subE.marshal)
			}
		}
	}
}
