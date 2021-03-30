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

// GetCrashSummary invokes the emas_appmonitor.GetCrashSummary API synchronously
func (client *Client) GetCrashSummary(request *GetCrashSummaryRequest) (response *GetCrashSummaryResponse, err error) {
	response = CreateGetCrashSummaryResponse()
	err = client.DoAction(request, response)
	return
}

// GetCrashSummaryWithChan invokes the emas_appmonitor.GetCrashSummary API asynchronously
func (client *Client) GetCrashSummaryWithChan(request *GetCrashSummaryRequest) (<-chan *GetCrashSummaryResponse, <-chan error) {
	responseChan := make(chan *GetCrashSummaryResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetCrashSummary(request)
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

// GetCrashSummaryWithCallback invokes the emas_appmonitor.GetCrashSummary API asynchronously
func (client *Client) GetCrashSummaryWithCallback(request *GetCrashSummaryRequest, callback func(response *GetCrashSummaryResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetCrashSummaryResponse
		var err error
		defer close(result)
		response, err = client.GetCrashSummary(request)
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

// GetCrashSummaryRequest is the request struct for api GetCrashSummary
type GetCrashSummaryRequest struct {
	*requests.RpcRequest
	UniqueAppId string           `position:"Body" name:"UniqueAppId"`
	DateTimeMs  requests.Integer `position:"Body" name:"DateTimeMs"`
	AppVersion  string           `position:"Body" name:"AppVersion"`
}

// GetCrashSummaryResponse is the response struct for api GetCrashSummary
type GetCrashSummaryResponse struct {
	*responses.BaseResponse
	RequestId        string             `json:"RequestId" xml:"RequestId"`
	CrashSummaryList []CrashSummaryItem `json:"CrashSummaryList" xml:"CrashSummaryList"`
}

// CreateGetCrashSummaryRequest creates a request to invoke GetCrashSummary API
func CreateGetCrashSummaryRequest() (request *GetCrashSummaryRequest) {
	request = &GetCrashSummaryRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("emas-appmonitor", "2019-06-11", "GetCrashSummary", "", "")
	request.Method = requests.POST
	return
}

// CreateGetCrashSummaryResponse creates a response to parse from GetCrashSummary response
func CreateGetCrashSummaryResponse() (response *GetCrashSummaryResponse) {
	response = &GetCrashSummaryResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
