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

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// SetSuccessInstance invokes the dataworks_public.SetSuccessInstance API synchronously
func (client *Client) SetSuccessInstance(request *SetSuccessInstanceRequest) (response *SetSuccessInstanceResponse, err error) {
	response = CreateSetSuccessInstanceResponse()
	err = client.DoAction(request, response)
	return
}

// SetSuccessInstanceWithChan invokes the dataworks_public.SetSuccessInstance API asynchronously
func (client *Client) SetSuccessInstanceWithChan(request *SetSuccessInstanceRequest) (<-chan *SetSuccessInstanceResponse, <-chan error) {
	responseChan := make(chan *SetSuccessInstanceResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.SetSuccessInstance(request)
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

// SetSuccessInstanceWithCallback invokes the dataworks_public.SetSuccessInstance API asynchronously
func (client *Client) SetSuccessInstanceWithCallback(request *SetSuccessInstanceRequest, callback func(response *SetSuccessInstanceResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *SetSuccessInstanceResponse
		var err error
		defer close(result)
		response, err = client.SetSuccessInstance(request)
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

// SetSuccessInstanceRequest is the request struct for api SetSuccessInstance
type SetSuccessInstanceRequest struct {
	*requests.RpcRequest
	ProjectEnv string           `position:"Body" name:"ProjectEnv"`
	InstanceId requests.Integer `position:"Body" name:"InstanceId"`
}

// SetSuccessInstanceResponse is the response struct for api SetSuccessInstance
type SetSuccessInstanceResponse struct {
	*responses.BaseResponse
	ErrorCode      string `json:"ErrorCode" xml:"ErrorCode"`
	ErrorMessage   string `json:"ErrorMessage" xml:"ErrorMessage"`
	HttpStatusCode int    `json:"HttpStatusCode" xml:"HttpStatusCode"`
	RequestId      string `json:"RequestId" xml:"RequestId"`
	Success        bool   `json:"Success" xml:"Success"`
	Data           bool   `json:"Data" xml:"Data"`
}

// CreateSetSuccessInstanceRequest creates a request to invoke SetSuccessInstance API
func CreateSetSuccessInstanceRequest() (request *SetSuccessInstanceRequest) {
	request = &SetSuccessInstanceRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("dataworks-public", "2020-05-18", "SetSuccessInstance", "", "")
	request.Method = requests.POST
	return
}

// CreateSetSuccessInstanceResponse creates a response to parse from SetSuccessInstance response
func CreateSetSuccessInstanceResponse() (response *SetSuccessInstanceResponse) {
	response = &SetSuccessInstanceResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
