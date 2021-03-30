package alidns

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

// DeleteDnsCacheDomain invokes the alidns.DeleteDnsCacheDomain API synchronously
func (client *Client) DeleteDnsCacheDomain(request *DeleteDnsCacheDomainRequest) (response *DeleteDnsCacheDomainResponse, err error) {
	response = CreateDeleteDnsCacheDomainResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteDnsCacheDomainWithChan invokes the alidns.DeleteDnsCacheDomain API asynchronously
func (client *Client) DeleteDnsCacheDomainWithChan(request *DeleteDnsCacheDomainRequest) (<-chan *DeleteDnsCacheDomainResponse, <-chan error) {
	responseChan := make(chan *DeleteDnsCacheDomainResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteDnsCacheDomain(request)
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

// DeleteDnsCacheDomainWithCallback invokes the alidns.DeleteDnsCacheDomain API asynchronously
func (client *Client) DeleteDnsCacheDomainWithCallback(request *DeleteDnsCacheDomainRequest, callback func(response *DeleteDnsCacheDomainResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteDnsCacheDomainResponse
		var err error
		defer close(result)
		response, err = client.DeleteDnsCacheDomain(request)
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

// DeleteDnsCacheDomainRequest is the request struct for api DeleteDnsCacheDomain
type DeleteDnsCacheDomainRequest struct {
	*requests.RpcRequest
	DomainName   string `position:"Query" name:"DomainName"`
	UserClientIp string `position:"Query" name:"UserClientIp"`
	Lang         string `position:"Query" name:"Lang"`
}

// DeleteDnsCacheDomainResponse is the response struct for api DeleteDnsCacheDomain
type DeleteDnsCacheDomainResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateDeleteDnsCacheDomainRequest creates a request to invoke DeleteDnsCacheDomain API
func CreateDeleteDnsCacheDomainRequest() (request *DeleteDnsCacheDomainRequest) {
	request = &DeleteDnsCacheDomainRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Alidns", "2015-01-09", "DeleteDnsCacheDomain", "alidns", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDeleteDnsCacheDomainResponse creates a response to parse from DeleteDnsCacheDomain response
func CreateDeleteDnsCacheDomainResponse() (response *DeleteDnsCacheDomainResponse) {
	response = &DeleteDnsCacheDomainResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
