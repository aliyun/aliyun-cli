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

// DatabaseInfo is a nested struct in dms_enterprise response
type DatabaseInfo struct {
	SearchName     string   `json:"SearchName" xml:"SearchName"`
	DbId           int64    `json:"DbId" xml:"DbId"`
	Logic          bool     `json:"Logic" xml:"Logic"`
	EnvType        string   `json:"EnvType" xml:"EnvType"`
	DbType         string   `json:"DbType" xml:"DbType"`
	OwnerNickNames []string `json:"OwnerNickNames" xml:"OwnerNickNames"`
	OwnerIds       []int64  `json:"OwnerIds" xml:"OwnerIds"`
}
