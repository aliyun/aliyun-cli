package live

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

// VerifyLiveDomainOwner invokes the live.VerifyLiveDomainOwner API synchronously
func (client *Client) VerifyLiveDomainOwner(request *VerifyLiveDomainOwnerRequest) (response *VerifyLiveDomainOwnerResponse, err error) {
	response = CreateVerifyLiveDomainOwnerResponse()
	err = client.DoAction(request, response)
	return
}

// VerifyLiveDomainOwnerWithChan invokes the live.VerifyLiveDomainOwner API asynchronously
func (client *Client) VerifyLiveDomainOwnerWithChan(request *VerifyLiveDomainOwnerRequest) (<-chan *VerifyLiveDomainOwnerResponse, <-chan error) {
	responseChan := make(chan *VerifyLiveDomainOwnerResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.VerifyLiveDomainOwner(request)
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

// VerifyLiveDomainOwnerWithCallback invokes the live.VerifyLiveDomainOwner API asynchronously
func (client *Client) VerifyLiveDomainOwnerWithCallback(request *VerifyLiveDomainOwnerRequest, callback func(response *VerifyLiveDomainOwnerResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *VerifyLiveDomainOwnerResponse
		var err error
		defer close(result)
		response, err = client.VerifyLiveDomainOwner(request)
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

// VerifyLiveDomainOwnerRequest is the request struct for api VerifyLiveDomainOwner
type VerifyLiveDomainOwnerRequest struct {
	*requests.RpcRequest
	VerifyType string           `position:"Query" name:"VerifyType"`
	DomainName string           `position:"Query" name:"DomainName"`
	OwnerId    requests.Integer `position:"Query" name:"OwnerId"`
}

// VerifyLiveDomainOwnerResponse is the response struct for api VerifyLiveDomainOwner
type VerifyLiveDomainOwnerResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Content   string `json:"Content" xml:"Content"`
}

// CreateVerifyLiveDomainOwnerRequest creates a request to invoke VerifyLiveDomainOwner API
func CreateVerifyLiveDomainOwnerRequest() (request *VerifyLiveDomainOwnerRequest) {
	request = &VerifyLiveDomainOwnerRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("live", "2016-11-01", "VerifyLiveDomainOwner", "live", "openAPI")
	request.Method = requests.POST
	return
}

// CreateVerifyLiveDomainOwnerResponse creates a response to parse from VerifyLiveDomainOwner response
func CreateVerifyLiveDomainOwnerResponse() (response *VerifyLiveDomainOwnerResponse) {
	response = &VerifyLiveDomainOwnerResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
