package csb

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

// Credential is a nested struct in csb response
type Credential struct {
	GmtCreate         int64             `json:"GmtCreate" xml:"GmtCreate"`
	Id                int64             `json:"Id" xml:"Id"`
	Name              string            `json:"Name" xml:"Name"`
	OwnerAttr         string            `json:"OwnerAttr" xml:"OwnerAttr"`
	UserId            string            `json:"UserId" xml:"UserId"`
	CurrentCredential CurrentCredential `json:"CurrentCredential" xml:"CurrentCredential"`
	NewCredential     NewCredential     `json:"NewCredential" xml:"NewCredential"`
}
