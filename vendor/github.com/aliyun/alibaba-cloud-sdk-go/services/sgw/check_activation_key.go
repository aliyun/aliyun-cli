package sgw

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

// CheckActivationKey invokes the sgw.CheckActivationKey API synchronously
func (client *Client) CheckActivationKey(request *CheckActivationKeyRequest) (response *CheckActivationKeyResponse, err error) {
	response = CreateCheckActivationKeyResponse()
	err = client.DoAction(request, response)
	return
}

// CheckActivationKeyWithChan invokes the sgw.CheckActivationKey API asynchronously
func (client *Client) CheckActivationKeyWithChan(request *CheckActivationKeyRequest) (<-chan *CheckActivationKeyResponse, <-chan error) {
	responseChan := make(chan *CheckActivationKeyResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CheckActivationKey(request)
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

// CheckActivationKeyWithCallback invokes the sgw.CheckActivationKey API asynchronously
func (client *Client) CheckActivationKeyWithCallback(request *CheckActivationKeyRequest, callback func(response *CheckActivationKeyResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CheckActivationKeyResponse
		var err error
		defer close(result)
		response, err = client.CheckActivationKey(request)
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

// CheckActivationKeyRequest is the request struct for api CheckActivationKey
type CheckActivationKeyRequest struct {
	*requests.RpcRequest
	CryptKey      string `position:"Query" name:"CryptKey"`
	Token         string `position:"Query" name:"Token"`
	SecurityToken string `position:"Query" name:"SecurityToken"`
	CryptText     string `position:"Query" name:"CryptText"`
	GatewayId     string `position:"Query" name:"GatewayId"`
}

// CheckActivationKeyResponse is the response struct for api CheckActivationKey
type CheckActivationKeyResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	Code      string `json:"Code" xml:"Code"`
	Message   string `json:"Message" xml:"Message"`
}

// CreateCheckActivationKeyRequest creates a request to invoke CheckActivationKey API
func CreateCheckActivationKeyRequest() (request *CheckActivationKeyRequest) {
	request = &CheckActivationKeyRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("sgw", "2018-05-11", "CheckActivationKey", "hcs_sgw", "openAPI")
	request.Method = requests.POST
	return
}

// CreateCheckActivationKeyResponse creates a response to parse from CheckActivationKey response
func CreateCheckActivationKeyResponse() (response *CheckActivationKeyResponse) {
	response = &CheckActivationKeyResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
