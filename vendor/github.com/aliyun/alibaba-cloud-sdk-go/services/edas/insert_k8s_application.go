package edas

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

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// InsertK8sApplication invokes the edas.InsertK8sApplication API synchronously
func (client *Client) InsertK8sApplication(request *InsertK8sApplicationRequest) (response *InsertK8sApplicationResponse, err error) {
	response = CreateInsertK8sApplicationResponse()
	err = client.DoAction(request, response)
	return
}

// InsertK8sApplicationWithChan invokes the edas.InsertK8sApplication API asynchronously
func (client *Client) InsertK8sApplicationWithChan(request *InsertK8sApplicationRequest) (<-chan *InsertK8sApplicationResponse, <-chan error) {
	responseChan := make(chan *InsertK8sApplicationResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.InsertK8sApplication(request)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	})
	if err != nil {
		errChan <- err
		close(responseChan)
		close(errChan)
	}
	return responseChan, errChan
}

// InsertK8sApplicationWithCallback invokes the edas.InsertK8sApplication API asynchronously
func (client *Client) InsertK8sApplicationWithCallback(request *InsertK8sApplicationRequest, callback func(response *InsertK8sApplicationResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *InsertK8sApplicationResponse
		var err error
		defer close(result)
		response, err = client.InsertK8sApplication(request)
		callback(response, err)
		result <- 1
	})
	if err != nil {
		defer close(result)
		callback(nil, err)
		result <- 0
	}
	return result
}

// InsertK8sApplicationRequest is the request struct for api InsertK8sApplication
type InsertK8sApplicationRequest struct {
	*requests.RoaRequest
	NasId                  string           `position:"Query" name:"NasId"`
	IntranetSlbId          string           `position:"Query" name:"IntranetSlbId"`
	Envs                   string           `position:"Query" name:"Envs"`
	RequestsMem            requests.Integer `position:"Query" name:"RequestsMem"`
	StorageType            string           `position:"Query" name:"StorageType"`
	AppName                string           `position:"Query" name:"AppName"`
	RequestsmCpu           requests.Integer `position:"Query" name:"RequestsmCpu"`
	DeployAcrossZones      string           `position:"Query" name:"DeployAcrossZones"`
	IntranetSlbPort        requests.Integer `position:"Query" name:"IntranetSlbPort"`
	DeployAcrossNodes      string           `position:"Query" name:"DeployAcrossNodes"`
	PreStop                string           `position:"Query" name:"PreStop"`
	LocalVolume            string           `position:"Query" name:"LocalVolume"`
	UseBodyEncoding        requests.Boolean `position:"Query" name:"UseBodyEncoding"`
	PackageType            string           `position:"Query" name:"PackageType"`
	RuntimeClassName       string           `position:"Query" name:"RuntimeClassName"`
	PostStart              string           `position:"Query" name:"PostStart"`
	RepoId                 string           `position:"Query" name:"RepoId"`
	InternetTargetPort     requests.Integer `position:"Query" name:"InternetTargetPort"`
	WebContainer           string           `position:"Query" name:"WebContainer"`
	EnableAhas             requests.Boolean `position:"Query" name:"EnableAhas"`
	SlsConfigs             string           `position:"Query" name:"SlsConfigs"`
	CommandArgs            string           `position:"Query" name:"CommandArgs"`
	Readiness              string           `position:"Query" name:"Readiness"`
	Liveness               string           `position:"Query" name:"Liveness"`
	InternetSlbPort        requests.Integer `position:"Query" name:"InternetSlbPort"`
	PackageVersion         string           `position:"Query" name:"PackageVersion"`
	Timeout                requests.Integer `position:"Query" name:"Timeout"`
	LimitMem               requests.Integer `position:"Query" name:"LimitMem"`
	LimitmCpu              requests.Integer `position:"Query" name:"LimitmCpu"`
	EdasContainerVersion   string           `position:"Query" name:"EdasContainerVersion"`
	InternetSlbId          string           `position:"Query" name:"InternetSlbId"`
	LogicalRegionId        string           `position:"Query" name:"LogicalRegionId"`
	PackageUrl             string           `position:"Query" name:"PackageUrl"`
	InternetSlbProtocol    string           `position:"Query" name:"InternetSlbProtocol"`
	MountDescs             string           `position:"Query" name:"MountDescs"`
	Replicas               requests.Integer `position:"Query" name:"Replicas"`
	LimitCpu               requests.Integer `position:"Query" name:"LimitCpu"`
	WebContainerConfig     string           `position:"Query" name:"WebContainerConfig"`
	IsMultilingualApp      requests.Boolean `position:"Query" name:"IsMultilingualApp"`
	ClusterId              string           `position:"Query" name:"ClusterId"`
	IntranetTargetPort     requests.Integer `position:"Query" name:"IntranetTargetPort"`
	Command                string           `position:"Query" name:"Command"`
	JDK                    string           `position:"Query" name:"JDK"`
	UriEncoding            string           `position:"Query" name:"UriEncoding"`
	IntranetSlbProtocol    string           `position:"Query" name:"IntranetSlbProtocol"`
	ImageUrl               string           `position:"Query" name:"ImageUrl"`
	PvcMountDescs          string           `position:"Query" name:"PvcMountDescs"`
	Namespace              string           `position:"Query" name:"Namespace"`
	ApplicationDescription string           `position:"Query" name:"ApplicationDescription"`
	RequestsCpu            requests.Integer `position:"Query" name:"RequestsCpu"`
	JavaStartUpConfig      string           `position:"Query" name:"JavaStartUpConfig"`
}

// InsertK8sApplicationResponse is the response struct for api InsertK8sApplication
type InsertK8sApplicationResponse struct {
	*responses.BaseResponse
	Code            int             `json:"Code" xml:"Code"`
	Message         string          `json:"Message" xml:"Message"`
	RequestId       string          `json:"RequestId" xml:"RequestId"`
	ApplicationInfo ApplicationInfo `json:"ApplicationInfo" xml:"ApplicationInfo"`
}

// CreateInsertK8sApplicationRequest creates a request to invoke InsertK8sApplication API
func CreateInsertK8sApplicationRequest() (request *InsertK8sApplicationRequest) {
	request = &InsertK8sApplicationRequest{
		RoaRequest: &requests.RoaRequest{},
	}
	request.InitWithApiInfo("Edas", "2017-08-01", "InsertK8sApplication", "/pop/v5/k8s/acs/create_k8s_app", "Edas", "openAPI")
	request.Method = requests.POST
	return
}

// CreateInsertK8sApplicationResponse creates a response to parse from InsertK8sApplication response
func CreateInsertK8sApplicationResponse() (response *InsertK8sApplicationResponse) {
	response = &InsertK8sApplicationResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
