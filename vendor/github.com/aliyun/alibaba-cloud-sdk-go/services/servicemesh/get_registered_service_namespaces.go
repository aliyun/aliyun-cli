package servicemesh

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

// GetRegisteredServiceNamespaces invokes the servicemesh.GetRegisteredServiceNamespaces API synchronously
func (client *Client) GetRegisteredServiceNamespaces(request *GetRegisteredServiceNamespacesRequest) (response *GetRegisteredServiceNamespacesResponse, err error) {
	response = CreateGetRegisteredServiceNamespacesResponse()
	err = client.DoAction(request, response)
	return
}

// GetRegisteredServiceNamespacesWithChan invokes the servicemesh.GetRegisteredServiceNamespaces API asynchronously
func (client *Client) GetRegisteredServiceNamespacesWithChan(request *GetRegisteredServiceNamespacesRequest) (<-chan *GetRegisteredServiceNamespacesResponse, <-chan error) {
	responseChan := make(chan *GetRegisteredServiceNamespacesResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetRegisteredServiceNamespaces(request)
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

// GetRegisteredServiceNamespacesWithCallback invokes the servicemesh.GetRegisteredServiceNamespaces API asynchronously
func (client *Client) GetRegisteredServiceNamespacesWithCallback(request *GetRegisteredServiceNamespacesRequest, callback func(response *GetRegisteredServiceNamespacesResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetRegisteredServiceNamespacesResponse
		var err error
		defer close(result)
		response, err = client.GetRegisteredServiceNamespaces(request)
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

// GetRegisteredServiceNamespacesRequest is the request struct for api GetRegisteredServiceNamespaces
type GetRegisteredServiceNamespacesRequest struct {
	*requests.RpcRequest
	ServiceMeshId string `position:"Body" name:"ServiceMeshId"`
}

// GetRegisteredServiceNamespacesResponse is the response struct for api GetRegisteredServiceNamespaces
type GetRegisteredServiceNamespacesResponse struct {
	*responses.BaseResponse
	RequestId  string   `json:"RequestId" xml:"RequestId"`
	Namespaces []string `json:"Namespaces" xml:"Namespaces"`
}

// CreateGetRegisteredServiceNamespacesRequest creates a request to invoke GetRegisteredServiceNamespaces API
func CreateGetRegisteredServiceNamespacesRequest() (request *GetRegisteredServiceNamespacesRequest) {
	request = &GetRegisteredServiceNamespacesRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("servicemesh", "2020-01-11", "GetRegisteredServiceNamespaces", "servicemesh", "openAPI")
	request.Method = requests.POST
	return
}

// CreateGetRegisteredServiceNamespacesResponse creates a response to parse from GetRegisteredServiceNamespaces response
func CreateGetRegisteredServiceNamespacesResponse() (response *GetRegisteredServiceNamespacesResponse) {
	response = &GetRegisteredServiceNamespacesResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
