package dms_enterprise

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

// Result is a nested struct in dms_enterprise response
type Result struct {
	Message         string                   `json:"Message" xml:"Message"`
	SourceTableName string                   `json:"SourceTableName" xml:"SourceTableName"`
	Success         bool                     `json:"Success" xml:"Success"`
	Script          string                   `json:"Script" xml:"Script"`
	RowCount        int64                    `json:"RowCount" xml:"RowCount"`
	TargetTableName string                   `json:"TargetTableName" xml:"TargetTableName"`
	ColumnNames     []string                 `json:"ColumnNames" xml:"ColumnNames"`
	Rows            []map[string]interface{} `json:"Rows" xml:"Rows"`
}
