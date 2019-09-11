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
	"io"
	"strings"
)

type title string
type row []string
type render interface {
	flush(table *Table) string
}

// Table Table structure
type Table struct {
	BorderStyle
	w     io.Writer
	width int
	// Text and border spacing
	padding      int
	parent       *Table
	subTable     []*Table
	cells        []render
	title        *title
	maxCellWidth map[int]int
}

// The marker selects the boundary position
const (
	TopFlag uint = 1 << iota
	BottomFlag
	LeftFlag
	RightFlag
	TopLeftFlag
	TopRightFlag
	BottomLeftFlag
	BottomRightFlag
)

// BorderStyle Describe the separators above and below
type BorderStyle struct {
	BorderTop,
	BorderBottom,
	BorderLeft,
	BorderRight,
	BorderTopLeft,
	BorderTopRight,
	BorderBottomLeft,
	BorderBottomRight byte
}

// DefaultBorderStyle Default table boundary separator
var DefaultBorderStyle = &BorderStyle{
	BorderTop:         '-',
	BorderBottom:      '-',
	BorderLeft:        '|',
	BorderRight:       '|',
	BorderTopLeft:     '+',
	BorderTopRight:    '+',
	BorderBottomLeft:  '+',
	BorderBottomRight: '+',
}

// NewTable Returns the default Table style
func NewTable(w io.Writer) *Table {
	table := new(Table)
	table.BorderStyle = *DefaultBorderStyle
	table.padding = 2
	table.w = w
	table.width = 2
	table.maxCellWidth = make(map[int]int)
	return table
}

// SetBorder Set the separator for the specified location
func (table *Table) SetBorder(position uint, borderChar byte) *Table {
	switch {
	case position&TopFlag != 0:
		table.BorderTop = borderChar
		fallthrough
	case position&BottomFlag != 0:
		table.BorderBottom = borderChar
		fallthrough
	case position&LeftFlag != 0:
		table.BorderLeft = borderChar
		fallthrough
	case position&RightFlag != 0:
		table.BorderRight = borderChar
		fallthrough
	case position&TopLeftFlag != 0:
		table.BorderTopLeft = borderChar
		fallthrough
	case position&TopRightFlag != 0:
		table.BorderTopRight = borderChar
		fallthrough
	case position&BottomLeftFlag != 0:
		table.BorderBottomLeft = borderChar
		fallthrough
	case position&BottomRightFlag != 0:
		table.BorderBottomRight = borderChar
	}
	return table
}

// ParentTable Return parent table
func (table *Table) ParentTable() *Table {
	return table.parent
}

// AddTitle Add title, centered display
func (table *Table) AddTitle(data string) *Table {
	if strings.ContainsAny(data, "\t\n\r") {
		panic("Illegal title")
	}
	str := title(data)
	table.title = &str
	width := separatorLen(data) + 6
	table.regularLen(width)
	return table

}

// AddRow Increase the line, split the column with \t, \n split the line
func (table *Table) AddRow(data string) *Table {
	var width int
	data = strings.TrimSuffix(data, "\n")
	rows := strings.Split(data, "\n")
	for _, rowData := range rows {
		width = 2
		if strings.Count(rowData, "\t")+1 > len(table.maxCellWidth) {
			width += strings.Count(rowData, "\t")
		} else {
			width += len(table.maxCellWidth) - 1
		}
		cols := strings.Split(rowData, "\t")
		for i, v := range cols {
			length := table.padding*2 + separatorLen(v)
			if length > table.maxCellWidth[i] {
				table.maxCellWidth[i] = length
			}
			width += table.maxCellWidth[i]
		}
		table.regularLen(width)
		table.cells = append(table.cells, row(cols))
	}
	return table
}

// AddNewTable Add a new form to the form and return to the new form
func (table *Table) AddNewTable(w io.Writer) *Table {
	subTable := new(Table)
	subTable.BorderStyle = table.BorderStyle
	subTable.padding = table.padding
	subTable.w = w
	subTable.width = table.width - 2
	subTable.maxCellWidth = make(map[int]int)
	subTable.parent = table
	table.subTable = append(table.subTable, subTable)
	return subTable
}

// AddTable Add an existing table and return to the parent table
func (table *Table) AddTable(subTable *Table) *Table {
	table.subTable = append(table.subTable, subTable)
	subTable.parent = table
	if subTable.width < table.width {
		subTable.regularLen(table.width - 2)
	} else {
		table.regularLen(subTable.width + 2)
	}

	return table
}

func (table *Table) Flush() {
	var (
		count     int
		data      string
		tableTemp = table
	)
	for tableTemp.parent != nil {
		count++
		tableTemp = tableTemp.parent
	}
	parentSpaces := strings.Repeat("|", count)
	separator := parentSpaces + table.flushSeparator() + parentSpaces + "\n"
	io.WriteString(table.w, separator)
	if table.title != nil {
		data = parentSpaces + table.title.flush(table) + parentSpaces + "\n"
		io.WriteString(table.w, data)
		io.WriteString(table.w, separator)
	}
	for _, v := range table.cells {
		data = parentSpaces + v.flush(table) + parentSpaces + "\n"
		io.WriteString(table.w, data)
	}
	for _, v := range table.subTable {
		v.Flush()
	}
	if table.subTable == nil {
		io.WriteString(table.w, separator)
	}

}

// IsEmptyCell Determine if the table contains any cells, not including return true
func (table *Table) IsEmptyCell() bool {
	if len(table.cells) != 0 {
		return false
	}
	return true
}

// IsEmptySub Determine if the table contains any child tables, does not contain return true
func (table *Table) IsEmptySub() bool {
	if len(table.subTable) != 0 {
		return false
	}
	return true
}

// Remove Delete subtable
func (table *Table) Remove(subT *Table) {
	for i, v := range table.subTable {
		if v == subT {
			table.subTable = append(table.subTable[:i], table.subTable[i+1:]...)
		}
	}
}

func (t title) flush(table *Table) string {
	var data []rune
	data = append(data, '|')
	preOffset := (table.width - 2 - separatorLen(string(t))) / 2
	preSpaces := strings.Repeat(" ", preOffset)
	data = append(data, []rune(preSpaces)...)
	data = append(data, []rune(string(t))...)
	sufOffset := table.width - preOffset - separatorLen(string(t)) - 2
	sufSpaces := strings.Repeat(" ", sufOffset)
	data = append(data, []rune(sufSpaces)...)
	data = append(data, '|')
	return string(data)
}

func (table *Table) flushSeparator() string {
	var data []rune
	data = append(data, rune(table.BorderTopLeft))
	data = append(data, []rune(strings.Repeat(string(table.BorderTop), table.width-2))...)
	data = append(data, rune(table.BorderTopRight))
	return string(data)
}

func (r row) flush(table *Table) string {
	var data []rune
	data = append(data, '|')
	tempWidth := table.width - 1 - len(table.maxCellWidth)
	for _, v := range table.maxCellWidth {
		tempWidth -= v
	}
	displacement := tempWidth / len(table.maxCellWidth)
	offset := tempWidth % len(table.maxCellWidth)
	for i := range table.maxCellWidth {
		table.maxCellWidth[i] += displacement
	}
	for i := 1; i <= offset; i++ {
		table.maxCellWidth[len(table.maxCellWidth)-i]++
	}
	preSpaces := strings.Repeat(" ", table.padding)
	for i := 0; i < len(table.maxCellWidth); i++ {
		var (
			column    string
			sufSpaces string
		)
		if i > len(r)-1 {
			column = ""
			sufSpaces = strings.Repeat(" ", table.maxCellWidth[i]-table.padding)
		} else {
			column = r[i]
			sufSpaces = strings.Repeat(" ", table.maxCellWidth[i]-separatorLen(column)-table.padding)
		}
		data = append(data, []rune(preSpaces)...)
		data = append(data, []rune(column)...)
		data = append(data, []rune(sufSpaces)...)
		data = append(data, '|')
	}
	return string(data)
}
func (table *Table) regularLen(length int) {
	if length > table.width {
		table.width = length
		if table.ParentTable() != nil {
			table.ParentTable().regularLen(length + 2)
		}
		for _, v := range table.subTable {
			v.regularLen(length - 2)
		}
	}
}
func separatorLen(w string) (length int) {
	for _, v := range w {
		if len(string(v)) == 1 {
			length++
		} else {
			length += 2
		}
	}
	return length
}
