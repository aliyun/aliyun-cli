package ens

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

// PodAbstractInfo is a nested struct in ens response
type PodAbstractInfo struct {
	Name              bool              `json:"Name" xml:"Name"`
	ResourceScope     bool              `json:"ResourceScope" xml:"ResourceScope"`
	ContainerService  bool              `json:"ContainerService" xml:"ContainerService"`
	Namespace         bool              `json:"Namespace" xml:"Namespace"`
	Status            bool              `json:"Status" xml:"Status"`
	ContainerStatuses ContainerStatuses `json:"ContainerStatuses" xml:"ContainerStatuses"`
}
