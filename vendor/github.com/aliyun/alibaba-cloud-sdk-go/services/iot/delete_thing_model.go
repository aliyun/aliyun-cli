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

// DeleteThingModel invokes the iot.DeleteThingModel API synchronously
func (client *Client) DeleteThingModel(request *DeleteThingModelRequest) (response *DeleteThingModelResponse, err error) {
	response = CreateDeleteThingModelResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteThingModelWithChan invokes the iot.DeleteThingModel API asynchronously
func (client *Client) DeleteThingModelWithChan(request *DeleteThingModelRequest) (<-chan *DeleteThingModelResponse, <-chan error) {
	responseChan := make(chan *DeleteThingModelResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteThingModel(request)
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

// DeleteThingModelWithCallback invokes the iot.DeleteThingModel API asynchronously
func (client *Client) DeleteThingModelWithCallback(request *DeleteThingModelRequest, callback func(response *DeleteThingModelResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteThingModelResponse
		var err error
		defer close(result)
		response, err = client.DeleteThingModel(request)
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

// DeleteThingModelRequest is the request struct for api DeleteThingModel
type DeleteThingModelRequest struct {
	*requests.RpcRequest
	RealTenantId       string           `position:"Query" name:"RealTenantId"`
	IsClearAllFunction requests.Boolean `position:"Query" name:"IsClearAllFunction"`
	RealTripartiteKey  string           `position:"Query" name:"RealTripartiteKey"`
	ResourceGroupId    string           `position:"Query" name:"ResourceGroupId"`
	PropertyIdentifier *[]string        `position:"Query" name:"PropertyIdentifier"  type:"Repeated"`
	IotInstanceId      string           `position:"Query" name:"IotInstanceId"`
	ServiceIdentifier  *[]string        `position:"Query" name:"ServiceIdentifier"  type:"Repeated"`
	ProductKey         string           `position:"Query" name:"ProductKey"`
	ApiProduct         string           `position:"Body" name:"ApiProduct"`
	ApiRevision        string           `position:"Body" name:"ApiRevision"`
	EventIdentifier    *[]string        `position:"Query" name:"EventIdentifier"  type:"Repeated"`
	FunctionBlockId    string           `position:"Query" name:"FunctionBlockId"`
}

// DeleteThingModelResponse is the response struct for api DeleteThingModel
type DeleteThingModelResponse struct {
	*responses.BaseResponse
	RequestId    string `json:"RequestId" xml:"RequestId"`
	Success      bool   `json:"Success" xml:"Success"`
	Code         string `json:"Code" xml:"Code"`
	ErrorMessage string `json:"ErrorMessage" xml:"ErrorMessage"`
}

// CreateDeleteThingModelRequest creates a request to invoke DeleteThingModel API
func CreateDeleteThingModelRequest() (request *DeleteThingModelRequest) {
	request = &DeleteThingModelRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Iot", "2018-01-20", "DeleteThingModel", "iot", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDeleteThingModelResponse creates a response to parse from DeleteThingModel response
func CreateDeleteThingModelResponse() (response *DeleteThingModelResponse) {
	response = &DeleteThingModelResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
