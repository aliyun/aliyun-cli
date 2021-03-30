package emas_appmonitor

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

// GetAppStatus invokes the emas_appmonitor.GetAppStatus API synchronously
func (client *Client) GetAppStatus(request *GetAppStatusRequest) (response *GetAppStatusResponse, err error) {
	response = CreateGetAppStatusResponse()
	err = client.DoAction(request, response)
	return
}

// GetAppStatusWithChan invokes the emas_appmonitor.GetAppStatus API asynchronously
func (client *Client) GetAppStatusWithChan(request *GetAppStatusRequest) (<-chan *GetAppStatusResponse, <-chan error) {
	responseChan := make(chan *GetAppStatusResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetAppStatus(request)
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

// GetAppStatusWithCallback invokes the emas_appmonitor.GetAppStatus API asynchronously
func (client *Client) GetAppStatusWithCallback(request *GetAppStatusRequest, callback func(response *GetAppStatusResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetAppStatusResponse
		var err error
		defer close(result)
		response, err = client.GetAppStatus(request)
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

// GetAppStatusRequest is the request struct for api GetAppStatus
type GetAppStatusRequest struct {
	*requests.RpcRequest
	UniqueAppId string `position:"Body" name:"UniqueAppId"`
}

// GetAppStatusResponse is the response struct for api GetAppStatus
type GetAppStatusResponse struct {
	*responses.BaseResponse
	RequestId string    `json:"RequestId" xml:"RequestId"`
	AppStatus AppStatus `json:"AppStatus" xml:"AppStatus"`
}

// CreateGetAppStatusRequest creates a request to invoke GetAppStatus API
func CreateGetAppStatusRequest() (request *GetAppStatusRequest) {
	request = &GetAppStatusRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("emas-appmonitor", "2019-06-11", "GetAppStatus", "", "")
	request.Method = requests.POST
	return
}

// CreateGetAppStatusResponse creates a response to parse from GetAppStatus response
func CreateGetAppStatusResponse() (response *GetAppStatusResponse) {
	response = &GetAppStatusResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
