package cloudcallcenter

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

// Category is a nested struct in cloudcallcenter response
type Category struct {
	Name              string `json:"Name" xml:"Name"`
	PreviousSiblingId string `json:"PreviousSiblingId" xml:"PreviousSiblingId"`
	CategoryId        string `json:"CategoryId" xml:"CategoryId"`
	ParentId          string `json:"ParentId" xml:"ParentId"`
	InstanceId        string `json:"InstanceId" xml:"InstanceId"`
	Id                string `json:"Id" xml:"Id"`
	Options           string `json:"Options" xml:"Options"`
	Level             int64  `json:"Level" xml:"Level"`
	ScenarioId        string `json:"ScenarioId" xml:"ScenarioId"`
	NextSiblingId     string `json:"NextSiblingId" xml:"NextSiblingId"`
	Type              int    `json:"Type" xml:"Type"`
}
