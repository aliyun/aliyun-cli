package dataworks_public

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

// EntityDto is a nested struct in dataworks_public response
type EntityDto struct {
	Id                int64  `json:"Id" xml:"Id"`
	ProjectName       string `json:"ProjectName" xml:"ProjectName"`
	TableName         string `json:"TableName" xml:"TableName"`
	EnvType           string `json:"EnvType" xml:"EnvType"`
	MatchExpression   string `json:"MatchExpression" xml:"MatchExpression"`
	EntityLevel       int    `json:"EntityLevel" xml:"EntityLevel"`
	OnDuty            string `json:"OnDuty" xml:"OnDuty"`
	ModifyUser        string `json:"ModifyUser" xml:"ModifyUser"`
	CreateTime        int64  `json:"CreateTime" xml:"CreateTime"`
	ModifyTime        int64  `json:"ModifyTime" xml:"ModifyTime"`
	Sql               int    `json:"Sql" xml:"Sql"`
	Task              int    `json:"Task" xml:"Task"`
	Followers         string `json:"Followers" xml:"Followers"`
	HasRelativeNode   bool   `json:"HasRelativeNode" xml:"HasRelativeNode"`
	RelativeNode      string `json:"RelativeNode" xml:"RelativeNode"`
	OnDutyAccountName string `json:"OnDutyAccountName" xml:"OnDutyAccountName"`
}
