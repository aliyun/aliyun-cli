package sae

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

// RestartApplication invokes the sae.RestartApplication API synchronously
func (client *Client) RestartApplication(request *RestartApplicationRequest) (response *RestartApplicationResponse, err error) {
	response = CreateRestartApplicationResponse()
	err = client.DoAction(request, response)
	return
}

// RestartApplicationWithChan invokes the sae.RestartApplication API asynchronously
func (client *Client) RestartApplicationWithChan(request *RestartApplicationRequest) (<-chan *RestartApplicationResponse, <-chan error) {
	responseChan := make(chan *RestartApplicationResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.RestartApplication(request)
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

// RestartApplicationWithCallback invokes the sae.RestartApplication API asynchronously
func (client *Client) RestartApplicationWithCallback(request *RestartApplicationRequest, callback func(response *RestartApplicationResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *RestartApplicationResponse
		var err error
		defer close(result)
		response, err = client.RestartApplication(request)
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

// RestartApplicationRequest is the request struct for api RestartApplication
type RestartApplicationRequest struct {
	*requests.RoaRequest
	MinReadyInstances requests.Integer `position:"Query" name:"MinReadyInstances"`
	AppId             string           `position:"Query" name:"AppId"`
}

// RestartApplicationResponse is the response struct for api RestartApplication
type RestartApplicationResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Code      string `json:"Code" xml:"Code"`
	Success   bool   `json:"Success" xml:"Success"`
	ErrorCode string `json:"ErrorCode" xml:"ErrorCode"`
	Message   string `json:"Message" xml:"Message"`
	TraceId   string `json:"TraceId" xml:"TraceId"`
	Data      Data   `json:"Data" xml:"Data"`
}

// CreateRestartApplicationRequest creates a request to invoke RestartApplication API
func CreateRestartApplicationRequest() (request *RestartApplicationRequest) {
	request = &RestartApplicationRequest{
		RoaRequest: &requests.RoaRequest{},
	}
	request.InitWithApiInfo("sae", "2019-05-06", "RestartApplication", "/pop/v1/sam/app/restartApplication", "serverless", "openAPI")
	request.Method = requests.PUT
	return
}

// CreateRestartApplicationResponse creates a response to parse from RestartApplication response
func CreateRestartApplicationResponse() (response *RestartApplicationResponse) {
	response = &RestartApplicationResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
