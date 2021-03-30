package cloudesl

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

// ActivateApService2 invokes the cloudesl.ActivateApService2 API synchronously
func (client *Client) ActivateApService2(request *ActivateApService2Request) (response *ActivateApService2Response, err error) {
	response = CreateActivateApService2Response()
	err = client.DoAction(request, response)
	return
}

// ActivateApService2WithChan invokes the cloudesl.ActivateApService2 API asynchronously
func (client *Client) ActivateApService2WithChan(request *ActivateApService2Request) (<-chan *ActivateApService2Response, <-chan error) {
	responseChan := make(chan *ActivateApService2Response, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ActivateApService2(request)
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

// ActivateApService2WithCallback invokes the cloudesl.ActivateApService2 API asynchronously
func (client *Client) ActivateApService2WithCallback(request *ActivateApService2Request, callback func(response *ActivateApService2Response, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ActivateApService2Response
		var err error
		defer close(result)
		response, err = client.ActivateApService2(request)
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

// ActivateApService2Request is the request struct for api ActivateApService2
type ActivateApService2Request struct {
	*requests.RpcRequest
	ApMac   string `position:"Query" name:"ApMac"`
	StoreId string `position:"Query" name:"StoreId"`
}

// ActivateApService2Response is the response struct for api ActivateApService2
type ActivateApService2Response struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	Message   string `json:"Message" xml:"Message"`
	ErrorCode string `json:"ErrorCode" xml:"ErrorCode"`
	Code      string `json:"Code" xml:"Code"`
}

// CreateActivateApService2Request creates a request to invoke ActivateApService2 API
func CreateActivateApService2Request() (request *ActivateApService2Request) {
	request = &ActivateApService2Request{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("cloudesl", "2018-08-01", "ActivateApService2", "cloudesl", "openAPI")
	request.Method = requests.POST
	return
}

// CreateActivateApService2Response creates a response to parse from ActivateApService2 response
func CreateActivateApService2Response() (response *ActivateApService2Response) {
	response = &ActivateApService2Response{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
