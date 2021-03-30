package ram

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

// SetDefaultPolicyVersion invokes the ram.SetDefaultPolicyVersion API synchronously
// api document: https://help.aliyun.com/api/ram/setdefaultpolicyversion.html
func (client *Client) SetDefaultPolicyVersion(request *SetDefaultPolicyVersionRequest) (response *SetDefaultPolicyVersionResponse, err error) {
	response = CreateSetDefaultPolicyVersionResponse()
	err = client.DoAction(request, response)
	return
}

// SetDefaultPolicyVersionWithChan invokes the ram.SetDefaultPolicyVersion API asynchronously
// api document: https://help.aliyun.com/api/ram/setdefaultpolicyversion.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) SetDefaultPolicyVersionWithChan(request *SetDefaultPolicyVersionRequest) (<-chan *SetDefaultPolicyVersionResponse, <-chan error) {
	responseChan := make(chan *SetDefaultPolicyVersionResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.SetDefaultPolicyVersion(request)
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

// SetDefaultPolicyVersionWithCallback invokes the ram.SetDefaultPolicyVersion API asynchronously
// api document: https://help.aliyun.com/api/ram/setdefaultpolicyversion.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) SetDefaultPolicyVersionWithCallback(request *SetDefaultPolicyVersionRequest, callback func(response *SetDefaultPolicyVersionResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *SetDefaultPolicyVersionResponse
		var err error
		defer close(result)
		response, err = client.SetDefaultPolicyVersion(request)
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

// SetDefaultPolicyVersionRequest is the request struct for api SetDefaultPolicyVersion
type SetDefaultPolicyVersionRequest struct {
	*requests.RpcRequest
	VersionId  string `position:"Query" name:"VersionId"`
	PolicyName string `position:"Query" name:"PolicyName"`
}

// SetDefaultPolicyVersionResponse is the response struct for api SetDefaultPolicyVersion
type SetDefaultPolicyVersionResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateSetDefaultPolicyVersionRequest creates a request to invoke SetDefaultPolicyVersion API
func CreateSetDefaultPolicyVersionRequest() (request *SetDefaultPolicyVersionRequest) {
	request = &SetDefaultPolicyVersionRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ram", "2015-05-01", "SetDefaultPolicyVersion", "Ram", "openAPI")
	return
}

// CreateSetDefaultPolicyVersionResponse creates a response to parse from SetDefaultPolicyVersion response
func CreateSetDefaultPolicyVersionResponse() (response *SetDefaultPolicyVersionResponse) {
	response = &SetDefaultPolicyVersionResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
