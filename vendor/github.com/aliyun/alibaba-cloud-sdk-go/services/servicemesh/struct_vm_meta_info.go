package servicemesh

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

// VmMetaInfo is a nested struct in servicemesh response
type VmMetaInfo struct {
	RootCertPath     string `json:"RootCertPath" xml:"RootCertPath"`
	RootCertContent  string `json:"RootCertContent" xml:"RootCertContent"`
	KeyPath          string `json:"KeyPath" xml:"KeyPath"`
	KeyContent       string `json:"KeyContent" xml:"KeyContent"`
	CertChainPath    string `json:"CertChainPath" xml:"CertChainPath"`
	CertChainContent string `json:"CertChainContent" xml:"CertChainContent"`
	EnvoyEnvPath     string `json:"EnvoyEnvPath" xml:"EnvoyEnvPath"`
	EnvoyEnvContent  string `json:"EnvoyEnvContent" xml:"EnvoyEnvContent"`
	HostsPath        string `json:"HostsPath" xml:"HostsPath"`
	HostsContent     string `json:"HostsContent" xml:"HostsContent"`
	TokenPath        string `json:"TokenPath" xml:"TokenPath"`
	TokenContent     string `json:"TokenContent" xml:"TokenContent"`
}
