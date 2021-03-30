package amqp_open

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

// ListDownStreamBindings invokes the amqp_open.ListDownStreamBindings API synchronously
// api document: https://help.aliyun.com/api/amqp-open/listdownstreambindings.html
func (client *Client) ListDownStreamBindings(request *ListDownStreamBindingsRequest) (response *ListDownStreamBindingsResponse, err error) {
	response = CreateListDownStreamBindingsResponse()
	err = client.DoAction(request, response)
	return
}

// ListDownStreamBindingsWithChan invokes the amqp_open.ListDownStreamBindings API asynchronously
// api document: https://help.aliyun.com/api/amqp-open/listdownstreambindings.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) ListDownStreamBindingsWithChan(request *ListDownStreamBindingsRequest) (<-chan *ListDownStreamBindingsResponse, <-chan error) {
	responseChan := make(chan *ListDownStreamBindingsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ListDownStreamBindings(request)
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

// ListDownStreamBindingsWithCallback invokes the amqp_open.ListDownStreamBindings API asynchronously
// api document: https://help.aliyun.com/api/amqp-open/listdownstreambindings.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) ListDownStreamBindingsWithCallback(request *ListDownStreamBindingsRequest, callback func(response *ListDownStreamBindingsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ListDownStreamBindingsResponse
		var err error
		defer close(result)
		response, err = client.ListDownStreamBindings(request)
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

// ListDownStreamBindingsRequest is the request struct for api ListDownStreamBindings
type ListDownStreamBindingsRequest struct {
	*requests.RpcRequest
	ExchangeName string           `position:"Query" name:"ExchangeName"`
	InstanceId   string           `position:"Query" name:"InstanceId"`
	NextToken    string           `position:"Query" name:"NextToken"`
	MaxResults   requests.Integer `position:"Query" name:"MaxResults"`
	VirtualHost  string           `position:"Query" name:"VirtualHost"`
}

// ListDownStreamBindingsResponse is the response struct for api ListDownStreamBindings
type ListDownStreamBindingsResponse struct {
	*responses.BaseResponse
	RequestId string                       `json:"RequestId" xml:"RequestId"`
	Code      int                          `json:"Code" xml:"Code"`
	Message   string                       `json:"Message" xml:"Message"`
	Success   bool                         `json:"Success" xml:"Success"`
	Data      DataInListDownStreamBindings `json:"Data" xml:"Data"`
}

// CreateListDownStreamBindingsRequest creates a request to invoke ListDownStreamBindings API
func CreateListDownStreamBindingsRequest() (request *ListDownStreamBindingsRequest) {
	request = &ListDownStreamBindingsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("amqp-open", "2019-12-12", "ListDownStreamBindings", "onsproxy", "openAPI")
	request.Method = requests.GET
	return
}

// CreateListDownStreamBindingsResponse creates a response to parse from ListDownStreamBindings response
func CreateListDownStreamBindingsResponse() (response *ListDownStreamBindingsResponse) {
	response = &ListDownStreamBindingsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
