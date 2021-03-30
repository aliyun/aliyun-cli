package csb

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

// DeleteCasService invokes the csb.DeleteCasService API synchronously
// api document: https://help.aliyun.com/api/csb/deletecasservice.html
func (client *Client) DeleteCasService(request *DeleteCasServiceRequest) (response *DeleteCasServiceResponse, err error) {
	response = CreateDeleteCasServiceResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteCasServiceWithChan invokes the csb.DeleteCasService API asynchronously
// api document: https://help.aliyun.com/api/csb/deletecasservice.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteCasServiceWithChan(request *DeleteCasServiceRequest) (<-chan *DeleteCasServiceResponse, <-chan error) {
	responseChan := make(chan *DeleteCasServiceResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteCasService(request)
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

// DeleteCasServiceWithCallback invokes the csb.DeleteCasService API asynchronously
// api document: https://help.aliyun.com/api/csb/deletecasservice.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DeleteCasServiceWithCallback(request *DeleteCasServiceRequest, callback func(response *DeleteCasServiceResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteCasServiceResponse
		var err error
		defer close(result)
		response, err = client.DeleteCasService(request)
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

// DeleteCasServiceRequest is the request struct for api DeleteCasService
type DeleteCasServiceRequest struct {
	*requests.RpcRequest
	LeafOnly     requests.Boolean `position:"Query" name:"LeafOnly"`
	CasCsbName   string           `position:"Query" name:"CasCsbName"`
	SrcUserId    string           `position:"Query" name:"SrcUserId"`
	CasServiceId string           `position:"Query" name:"CasServiceId"`
}

// DeleteCasServiceResponse is the response struct for api DeleteCasService
type DeleteCasServiceResponse struct {
	*responses.BaseResponse
	Code      int    `json:"Code" xml:"Code"`
	Message   string `json:"Message" xml:"Message"`
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateDeleteCasServiceRequest creates a request to invoke DeleteCasService API
func CreateDeleteCasServiceRequest() (request *DeleteCasServiceRequest) {
	request = &DeleteCasServiceRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("CSB", "2017-11-18", "DeleteCasService", "", "")
	request.Method = requests.POST
	return
}

// CreateDeleteCasServiceResponse creates a response to parse from DeleteCasService response
func CreateDeleteCasServiceResponse() (response *DeleteCasServiceResponse) {
	response = &DeleteCasServiceResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
