package iot

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

// CopyThingModel invokes the iot.CopyThingModel API synchronously
func (client *Client) CopyThingModel(request *CopyThingModelRequest) (response *CopyThingModelResponse, err error) {
	response = CreateCopyThingModelResponse()
	err = client.DoAction(request, response)
	return
}

// CopyThingModelWithChan invokes the iot.CopyThingModel API asynchronously
func (client *Client) CopyThingModelWithChan(request *CopyThingModelRequest) (<-chan *CopyThingModelResponse, <-chan error) {
	responseChan := make(chan *CopyThingModelResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CopyThingModel(request)
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

// CopyThingModelWithCallback invokes the iot.CopyThingModel API asynchronously
func (client *Client) CopyThingModelWithCallback(request *CopyThingModelRequest, callback func(response *CopyThingModelResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CopyThingModelResponse
		var err error
		defer close(result)
		response, err = client.CopyThingModel(request)
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

// CopyThingModelRequest is the request struct for api CopyThingModel
type CopyThingModelRequest struct {
	*requests.RpcRequest
	RealTenantId       string `position:"Query" name:"RealTenantId"`
	TargetProductKey   string `position:"Query" name:"TargetProductKey"`
	RealTripartiteKey  string `position:"Query" name:"RealTripartiteKey"`
	ResourceGroupId    string `position:"Query" name:"ResourceGroupId"`
	IotInstanceId      string `position:"Query" name:"IotInstanceId"`
	SourceModelVersion string `position:"Query" name:"SourceModelVersion"`
	SourceProductKey   string `position:"Query" name:"SourceProductKey"`
	ApiProduct         string `position:"Body" name:"ApiProduct"`
	ApiRevision        string `position:"Body" name:"ApiRevision"`
}

// CopyThingModelResponse is the response struct for api CopyThingModel
type CopyThingModelResponse struct {
	*responses.BaseResponse
	RequestId    string `json:"RequestId" xml:"RequestId"`
	Success      bool   `json:"Success" xml:"Success"`
	Code         string `json:"Code" xml:"Code"`
	ErrorMessage string `json:"ErrorMessage" xml:"ErrorMessage"`
}

// CreateCopyThingModelRequest creates a request to invoke CopyThingModel API
func CreateCopyThingModelRequest() (request *CopyThingModelRequest) {
	request = &CopyThingModelRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Iot", "2018-01-20", "CopyThingModel", "iot", "openAPI")
	request.Method = requests.POST
	return
}

// CreateCopyThingModelResponse creates a response to parse from CopyThingModel response
func CreateCopyThingModelResponse() (response *CopyThingModelResponse) {
	response = &CopyThingModelResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
