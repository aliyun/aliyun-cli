package r_kvstore

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

// LogRecords is a nested struct in r_kvstore response
type LogRecords struct {
	Category     string `json:"Category" xml:"Category"`
	Account      string `json:"Account" xml:"Account"`
	Id           int    `json:"Id" xml:"Id"`
	Level        string `json:"Level" xml:"Level"`
	CreateTime   string `json:"CreateTime" xml:"CreateTime"`
	IPAddress    string `json:"IPAddress" xml:"IPAddress"`
	DataBaseName string `json:"DataBaseName" xml:"DataBaseName"`
	ElapsedTime  int64  `json:"ElapsedTime" xml:"ElapsedTime"`
	ConnInfo     string `json:"ConnInfo" xml:"ConnInfo"`
	Command      string `json:"Command" xml:"Command"`
	ExecuteTime  string `json:"ExecuteTime" xml:"ExecuteTime"`
	NodeId       string `json:"NodeId" xml:"NodeId"`
	Content      string `json:"Content" xml:"Content"`
	AccountName  string `json:"AccountName" xml:"AccountName"`
	InstanceId   string `json:"InstanceId" xml:"InstanceId"`
	DBName       string `json:"DBName" xml:"DBName"`
}
