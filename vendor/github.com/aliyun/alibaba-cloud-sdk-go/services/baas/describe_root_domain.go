package baas

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

// DescribeRootDomain invokes the baas.DescribeRootDomain API synchronously
// api document: https://help.aliyun.com/api/baas/describerootdomain.html
func (client *Client) DescribeRootDomain(request *DescribeRootDomainRequest) (response *DescribeRootDomainResponse, err error) {
	response = CreateDescribeRootDomainResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeRootDomainWithChan invokes the baas.DescribeRootDomain API asynchronously
// api document: https://help.aliyun.com/api/baas/describerootdomain.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeRootDomainWithChan(request *DescribeRootDomainRequest) (<-chan *DescribeRootDomainResponse, <-chan error) {
	responseChan := make(chan *DescribeRootDomainResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeRootDomain(request)
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

// DescribeRootDomainWithCallback invokes the baas.DescribeRootDomain API asynchronously
// api document: https://help.aliyun.com/api/baas/describerootdomain.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeRootDomainWithCallback(request *DescribeRootDomainRequest, callback func(response *DescribeRootDomainResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeRootDomainResponse
		var err error
		defer close(result)
		response, err = client.DescribeRootDomain(request)
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

// DescribeRootDomainRequest is the request struct for api DescribeRootDomain
type DescribeRootDomainRequest struct {
	*requests.RpcRequest
}

// DescribeRootDomainResponse is the response struct for api DescribeRootDomain
type DescribeRootDomainResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	ErrorCode int    `json:"ErrorCode" xml:"ErrorCode"`
	Result    string `json:"Result" xml:"Result"`
}

// CreateDescribeRootDomainRequest creates a request to invoke DescribeRootDomain API
func CreateDescribeRootDomainRequest() (request *DescribeRootDomainRequest) {
	request = &DescribeRootDomainRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Baas", "2018-12-21", "DescribeRootDomain", "baas", "openAPI")
	return
}

// CreateDescribeRootDomainResponse creates a response to parse from DescribeRootDomain response
func CreateDescribeRootDomainResponse() (response *DescribeRootDomainResponse) {
	response = &DescribeRootDomainResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
