package ehpc

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

// UpdateClusterVolumes invokes the ehpc.UpdateClusterVolumes API synchronously
func (client *Client) UpdateClusterVolumes(request *UpdateClusterVolumesRequest) (response *UpdateClusterVolumesResponse, err error) {
	response = CreateUpdateClusterVolumesResponse()
	err = client.DoAction(request, response)
	return
}

// UpdateClusterVolumesWithChan invokes the ehpc.UpdateClusterVolumes API asynchronously
func (client *Client) UpdateClusterVolumesWithChan(request *UpdateClusterVolumesRequest) (<-chan *UpdateClusterVolumesResponse, <-chan error) {
	responseChan := make(chan *UpdateClusterVolumesResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.UpdateClusterVolumes(request)
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

// UpdateClusterVolumesWithCallback invokes the ehpc.UpdateClusterVolumes API asynchronously
func (client *Client) UpdateClusterVolumesWithCallback(request *UpdateClusterVolumesRequest, callback func(response *UpdateClusterVolumesResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *UpdateClusterVolumesResponse
		var err error
		defer close(result)
		response, err = client.UpdateClusterVolumes(request)
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

// UpdateClusterVolumesRequest is the request struct for api UpdateClusterVolumes
type UpdateClusterVolumesRequest struct {
	*requests.RpcRequest
	AdditionalVolumes *[]UpdateClusterVolumesAdditionalVolumes `position:"Query" name:"AdditionalVolumes"  type:"Repeated"`
	ClusterId         string                                   `position:"Query" name:"ClusterId"`
}

// UpdateClusterVolumesAdditionalVolumes is a repeated param struct in UpdateClusterVolumesRequest
type UpdateClusterVolumesAdditionalVolumes struct {
	VolumeType       string                       `name:"VolumeType"`
	VolumeProtocol   string                       `name:"VolumeProtocol"`
	LocalDirectory   string                       `name:"LocalDirectory"`
	RemoteDirectory  string                       `name:"RemoteDirectory"`
	Roles            *[]UpdateClusterVolumesRoles `name:"Roles" type:"Repeated"`
	VolumeId         string                       `name:"VolumeId"`
	VolumeMountpoint string                       `name:"VolumeMountpoint"`
	Location         string                       `name:"Location"`
	JobQueue         string                       `name:"JobQueue"`
}

// UpdateClusterVolumesRoles is a repeated param struct in UpdateClusterVolumesRequest
type UpdateClusterVolumesRoles struct {
	Name string `name:"Name"`
}

// UpdateClusterVolumesResponse is the response struct for api UpdateClusterVolumes
type UpdateClusterVolumesResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateUpdateClusterVolumesRequest creates a request to invoke UpdateClusterVolumes API
func CreateUpdateClusterVolumesRequest() (request *UpdateClusterVolumesRequest) {
	request = &UpdateClusterVolumesRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("EHPC", "2018-04-12", "UpdateClusterVolumes", "", "")
	request.Method = requests.GET
	return
}

// CreateUpdateClusterVolumesResponse creates a response to parse from UpdateClusterVolumes response
func CreateUpdateClusterVolumesResponse() (response *UpdateClusterVolumesResponse) {
	response = &UpdateClusterVolumesResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
