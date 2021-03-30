package linkwan

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

// SubmitKpmEncryptedNodeTupleOrder invokes the linkwan.SubmitKpmEncryptedNodeTupleOrder API synchronously
func (client *Client) SubmitKpmEncryptedNodeTupleOrder(request *SubmitKpmEncryptedNodeTupleOrderRequest) (response *SubmitKpmEncryptedNodeTupleOrderResponse, err error) {
	response = CreateSubmitKpmEncryptedNodeTupleOrderResponse()
	err = client.DoAction(request, response)
	return
}

// SubmitKpmEncryptedNodeTupleOrderWithChan invokes the linkwan.SubmitKpmEncryptedNodeTupleOrder API asynchronously
func (client *Client) SubmitKpmEncryptedNodeTupleOrderWithChan(request *SubmitKpmEncryptedNodeTupleOrderRequest) (<-chan *SubmitKpmEncryptedNodeTupleOrderResponse, <-chan error) {
	responseChan := make(chan *SubmitKpmEncryptedNodeTupleOrderResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.SubmitKpmEncryptedNodeTupleOrder(request)
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

// SubmitKpmEncryptedNodeTupleOrderWithCallback invokes the linkwan.SubmitKpmEncryptedNodeTupleOrder API asynchronously
func (client *Client) SubmitKpmEncryptedNodeTupleOrderWithCallback(request *SubmitKpmEncryptedNodeTupleOrderRequest, callback func(response *SubmitKpmEncryptedNodeTupleOrderResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *SubmitKpmEncryptedNodeTupleOrderResponse
		var err error
		defer close(result)
		response, err = client.SubmitKpmEncryptedNodeTupleOrder(request)
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

// SubmitKpmEncryptedNodeTupleOrderRequest is the request struct for api SubmitKpmEncryptedNodeTupleOrder
type SubmitKpmEncryptedNodeTupleOrderRequest struct {
	*requests.RpcRequest
	LoraVersion   string           `position:"Query" name:"LoraVersion"`
	TupleType     string           `position:"Query" name:"TupleType"`
	ApiProduct    string           `position:"Body" name:"ApiProduct"`
	ApiRevision   string           `position:"Body" name:"ApiRevision"`
	RequiredCount requests.Integer `position:"Query" name:"RequiredCount"`
}

// SubmitKpmEncryptedNodeTupleOrderResponse is the response struct for api SubmitKpmEncryptedNodeTupleOrder
type SubmitKpmEncryptedNodeTupleOrderResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	OrderId   string `json:"OrderId" xml:"OrderId"`
}

// CreateSubmitKpmEncryptedNodeTupleOrderRequest creates a request to invoke SubmitKpmEncryptedNodeTupleOrder API
func CreateSubmitKpmEncryptedNodeTupleOrderRequest() (request *SubmitKpmEncryptedNodeTupleOrderRequest) {
	request = &SubmitKpmEncryptedNodeTupleOrderRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("LinkWAN", "2019-03-01", "SubmitKpmEncryptedNodeTupleOrder", "linkwan", "openAPI")
	request.Method = requests.POST
	return
}

// CreateSubmitKpmEncryptedNodeTupleOrderResponse creates a response to parse from SubmitKpmEncryptedNodeTupleOrder response
func CreateSubmitKpmEncryptedNodeTupleOrderResponse() (response *SubmitKpmEncryptedNodeTupleOrderResponse) {
	response = &SubmitKpmEncryptedNodeTupleOrderResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
