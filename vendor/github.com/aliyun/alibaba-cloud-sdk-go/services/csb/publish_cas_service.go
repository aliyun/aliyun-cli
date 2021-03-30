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

// PublishCasService invokes the csb.PublishCasService API synchronously
// api document: https://help.aliyun.com/api/csb/publishcasservice.html
func (client *Client) PublishCasService(request *PublishCasServiceRequest) (response *PublishCasServiceResponse, err error) {
	response = CreatePublishCasServiceResponse()
	err = client.DoAction(request, response)
	return
}

// PublishCasServiceWithChan invokes the csb.PublishCasService API asynchronously
// api document: https://help.aliyun.com/api/csb/publishcasservice.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) PublishCasServiceWithChan(request *PublishCasServiceRequest) (<-chan *PublishCasServiceResponse, <-chan error) {
	responseChan := make(chan *PublishCasServiceResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.PublishCasService(request)
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

// PublishCasServiceWithCallback invokes the csb.PublishCasService API asynchronously
// api document: https://help.aliyun.com/api/csb/publishcasservice.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) PublishCasServiceWithCallback(request *PublishCasServiceRequest, callback func(response *PublishCasServiceResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *PublishCasServiceResponse
		var err error
		defer close(result)
		response, err = client.PublishCasService(request)
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

// PublishCasServiceRequest is the request struct for api PublishCasService
type PublishCasServiceRequest struct {
	*requests.RpcRequest
	CasCsbName string `position:"Query" name:"CasCsbName"`
	Data       string `position:"Body" name:"Data"`
}

// PublishCasServiceResponse is the response struct for api PublishCasService
type PublishCasServiceResponse struct {
	*responses.BaseResponse
	Code      int    `json:"Code" xml:"Code"`
	Message   string `json:"Message" xml:"Message"`
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreatePublishCasServiceRequest creates a request to invoke PublishCasService API
func CreatePublishCasServiceRequest() (request *PublishCasServiceRequest) {
	request = &PublishCasServiceRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("CSB", "2017-11-18", "PublishCasService", "", "")
	request.Method = requests.POST
	return
}

// CreatePublishCasServiceResponse creates a response to parse from PublishCasService response
func CreatePublishCasServiceResponse() (response *PublishCasServiceResponse) {
	response = &PublishCasServiceResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
