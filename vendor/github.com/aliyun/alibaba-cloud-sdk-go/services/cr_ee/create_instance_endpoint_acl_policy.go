package cr_ee

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

// CreateInstanceEndpointAclPolicy invokes the cr.CreateInstanceEndpointAclPolicy API synchronously
// api document: https://help.aliyun.com/api/cr/createinstanceendpointaclpolicy.html
func (client *Client) CreateInstanceEndpointAclPolicy(request *CreateInstanceEndpointAclPolicyRequest) (response *CreateInstanceEndpointAclPolicyResponse, err error) {
	response = CreateCreateInstanceEndpointAclPolicyResponse()
	err = client.DoAction(request, response)
	return
}

// CreateInstanceEndpointAclPolicyWithChan invokes the cr.CreateInstanceEndpointAclPolicy API asynchronously
// api document: https://help.aliyun.com/api/cr/createinstanceendpointaclpolicy.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) CreateInstanceEndpointAclPolicyWithChan(request *CreateInstanceEndpointAclPolicyRequest) (<-chan *CreateInstanceEndpointAclPolicyResponse, <-chan error) {
	responseChan := make(chan *CreateInstanceEndpointAclPolicyResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CreateInstanceEndpointAclPolicy(request)
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

// CreateInstanceEndpointAclPolicyWithCallback invokes the cr.CreateInstanceEndpointAclPolicy API asynchronously
// api document: https://help.aliyun.com/api/cr/createinstanceendpointaclpolicy.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) CreateInstanceEndpointAclPolicyWithCallback(request *CreateInstanceEndpointAclPolicyRequest, callback func(response *CreateInstanceEndpointAclPolicyResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CreateInstanceEndpointAclPolicyResponse
		var err error
		defer close(result)
		response, err = client.CreateInstanceEndpointAclPolicy(request)
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

// CreateInstanceEndpointAclPolicyRequest is the request struct for api CreateInstanceEndpointAclPolicy
type CreateInstanceEndpointAclPolicyRequest struct {
	*requests.RpcRequest
	Entry        string `position:"Query" name:"Entry"`
	InstanceId   string `position:"Query" name:"InstanceId"`
	EndpointType string `position:"Query" name:"EndpointType"`
	ModuleName   string `position:"Query" name:"ModuleName"`
	Comment      string `position:"Query" name:"Comment"`
}

// CreateInstanceEndpointAclPolicyResponse is the response struct for api CreateInstanceEndpointAclPolicy
type CreateInstanceEndpointAclPolicyResponse struct {
	*responses.BaseResponse
	CreateInstanceEndpointAclPolicyIsSuccess bool   `json:"IsSuccess" xml:"IsSuccess"`
	Code                                     string `json:"Code" xml:"Code"`
	RequestId                                string `json:"RequestId" xml:"RequestId"`
}

// CreateCreateInstanceEndpointAclPolicyRequest creates a request to invoke CreateInstanceEndpointAclPolicy API
func CreateCreateInstanceEndpointAclPolicyRequest() (request *CreateInstanceEndpointAclPolicyRequest) {
	request = &CreateInstanceEndpointAclPolicyRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("cr", "2018-12-01", "CreateInstanceEndpointAclPolicy", "acr", "openAPI")
	request.Method = requests.POST
	return
}

// CreateCreateInstanceEndpointAclPolicyResponse creates a response to parse from CreateInstanceEndpointAclPolicy response
func CreateCreateInstanceEndpointAclPolicyResponse() (response *CreateInstanceEndpointAclPolicyResponse) {
	response = &CreateInstanceEndpointAclPolicyResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
