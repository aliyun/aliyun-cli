package sgw

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

// DeleteGatewayBlockVolumes invokes the sgw.DeleteGatewayBlockVolumes API synchronously
func (client *Client) DeleteGatewayBlockVolumes(request *DeleteGatewayBlockVolumesRequest) (response *DeleteGatewayBlockVolumesResponse, err error) {
	response = CreateDeleteGatewayBlockVolumesResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteGatewayBlockVolumesWithChan invokes the sgw.DeleteGatewayBlockVolumes API asynchronously
func (client *Client) DeleteGatewayBlockVolumesWithChan(request *DeleteGatewayBlockVolumesRequest) (<-chan *DeleteGatewayBlockVolumesResponse, <-chan error) {
	responseChan := make(chan *DeleteGatewayBlockVolumesResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteGatewayBlockVolumes(request)
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

// DeleteGatewayBlockVolumesWithCallback invokes the sgw.DeleteGatewayBlockVolumes API asynchronously
func (client *Client) DeleteGatewayBlockVolumesWithCallback(request *DeleteGatewayBlockVolumesRequest, callback func(response *DeleteGatewayBlockVolumesResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteGatewayBlockVolumesResponse
		var err error
		defer close(result)
		response, err = client.DeleteGatewayBlockVolumes(request)
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

// DeleteGatewayBlockVolumesRequest is the request struct for api DeleteGatewayBlockVolumes
type DeleteGatewayBlockVolumesRequest struct {
	*requests.RpcRequest
	IsSourceDeletion requests.Boolean `position:"Query" name:"IsSourceDeletion"`
	SecurityToken    string           `position:"Query" name:"SecurityToken"`
	IndexId          string           `position:"Query" name:"IndexId"`
	GatewayId        string           `position:"Query" name:"GatewayId"`
}

// DeleteGatewayBlockVolumesResponse is the response struct for api DeleteGatewayBlockVolumes
type DeleteGatewayBlockVolumesResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	Code      string `json:"Code" xml:"Code"`
	Message   string `json:"Message" xml:"Message"`
	TaskId    string `json:"TaskId" xml:"TaskId"`
}

// CreateDeleteGatewayBlockVolumesRequest creates a request to invoke DeleteGatewayBlockVolumes API
func CreateDeleteGatewayBlockVolumesRequest() (request *DeleteGatewayBlockVolumesRequest) {
	request = &DeleteGatewayBlockVolumesRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("sgw", "2018-05-11", "DeleteGatewayBlockVolumes", "hcs_sgw", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDeleteGatewayBlockVolumesResponse creates a response to parse from DeleteGatewayBlockVolumes response
func CreateDeleteGatewayBlockVolumesResponse() (response *DeleteGatewayBlockVolumesResponse) {
	response = &DeleteGatewayBlockVolumesResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
