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
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

var expectStr = `+------------------------------------------------------+
|                   DescribeReigons                    |
+------------------------------------------------------+
|  RequestId  |  A71F2A35-BA66-496F-8B3F-BC19F4268937  |
|+----------------------------------------------------+|
||                      Regions                       ||
|+----------------------------------------------------+|
||  testcell                   |                      ||
||+--------------------------------------------------+||
|||                      Region                      |||
||+--------------------------------------------------+||
|||  RegionId             |  cn-qingdao              |||
|||  RegionEndpoint       |  ecs.aliyuncs.com        |||
|||  LocalName            |  华北 1                  |||
||+--------------------------------------------------+||
||+--------------------------------------------------+||
||+--------------------------------------------------+||
`
var jsonData = `{
	"RequestId": "A71F2A35-BA66-496F-8B3F-BC19F4268937",
	"Regions": {
			"Region": [
					{
							"RegionId": "cn-qingdao",
							"RegionEndpoint": "ecs.aliyuncs.com",
							"LocalName": "华北 1"
					},
					{
							"RegionId": "cn-beijing",
							"RegionEndpoint": "ecs.aliyuncs.com",
							"LocalName": "华北 2"
					}
			]
	}
}`
var jsonTable = `+------------------------------------------------------+
|                   DescribeRegions                    |
+------------------------------------------------------+
|  RequestId  |  A71F2A35-BA66-496F-8B3F-BC19F4268937  |
|+----------------------------------------------------+|
||                      Regions                       ||
|+----------------------------------------------------+|
||+--------------------------------------------------+||
|||                      Region                      |||
||+--------------------------------------------------+||
|||  LocalName            |  华北 1                  |||
|||  RegionEndpoint       |  ecs.aliyuncs.com        |||
|||  RegionId             |  cn-qingdao              |||
||+--------------------------------------------------+||
||+--------------------------------------------------+||
|||                      Region                      |||
||+--------------------------------------------------+||
|||  LocalName            |  华北 2                  |||
|||  RegionEndpoint       |  ecs.aliyuncs.com        |||
|||  RegionId             |  cn-beijing              |||
||+--------------------------------------------------+||
`

func TestTable(t *testing.T) {

	// test table
	buf := new(bytes.Buffer)
	table := NewTable(buf)
	table2 := NewTable(buf)
	table.
		AddRow("RequestId\tA71F2A35-BA66-496F-8B3F-BC19F4268937").AddTitle("DescribeReigons").
		AddNewTable(buf).AddTitle("Regions").
		AddNewTable(buf).AddTitle("Region").AddRow("RegionId\tcn-qingdao").AddRow("RegionEndpoint\tecs.aliyuncs.com").AddRow("LocalName\t华北 1").ParentTable().AddRow("testcell\t").
		AddTable(table2)
	table.Flush()
	assert.Equal(t, expectStr, buf.String())
	assert.True(t, table2.IsEmptyCell())
	assert.True(t, table2.IsEmptySub())
	table.subTable[0].subTable[0].Remove(table2)
	assert.Equal(t, 0, len(table.subTable[0].subTable[0].subTable))

	// test json to table
	buf.Reset()
	table = NewTable(buf)
	table.AddTitle("DescribeRegions")
	FromJSON([]byte(jsonData), table)
	table.Flush()
	assert.Equal(t, jsonTable, buf.String())
}
