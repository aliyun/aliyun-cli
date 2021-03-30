package retailcloud

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

// ListAppResourceAllocs invokes the retailcloud.ListAppResourceAllocs API synchronously
func (client *Client) ListAppResourceAllocs(request *ListAppResourceAllocsRequest) (response *ListAppResourceAllocsResponse, err error) {
	response = CreateListAppResourceAllocsResponse()
	err = client.DoAction(request, response)
	return
}

// ListAppResourceAllocsWithChan invokes the retailcloud.ListAppResourceAllocs API asynchronously
func (client *Client) ListAppResourceAllocsWithChan(request *ListAppResourceAllocsRequest) (<-chan *ListAppResourceAllocsResponse, <-chan error) {
	responseChan := make(chan *ListAppResourceAllocsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ListAppResourceAllocs(request)
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

// ListAppResourceAllocsWithCallback invokes the retailcloud.ListAppResourceAllocs API asynchronously
func (client *Client) ListAppResourceAllocsWithCallback(request *ListAppResourceAllocsRequest, callback func(response *ListAppResourceAllocsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ListAppResourceAllocsResponse
		var err error
		defer close(result)
		response, err = client.ListAppResourceAllocs(request)
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

// ListAppResourceAllocsRequest is the request struct for api ListAppResourceAllocs
type ListAppResourceAllocsRequest struct {
	*requests.RpcRequest
	AppId      requests.Integer `position:"Query" name:"AppId"`
	PageSize   requests.Integer `position:"Query" name:"PageSize"`
	AppEnvId   requests.Integer `position:"Query" name:"AppEnvId"`
	ClusterId  string           `position:"Query" name:"ClusterId"`
	PageNumber requests.Integer `position:"Query" name:"PageNumber"`
}

// ListAppResourceAllocsResponse is the response struct for api ListAppResourceAllocs
type ListAppResourceAllocsResponse struct {
	*responses.BaseResponse
	Code       int                            `json:"Code" xml:"Code"`
	ErrorMsg   string                         `json:"ErrorMsg" xml:"ErrorMsg"`
	PageNumber int                            `json:"PageNumber" xml:"PageNumber"`
	PageSize   int                            `json:"PageSize" xml:"PageSize"`
	RequestId  string                         `json:"RequestId" xml:"RequestId"`
	TotalCount int64                          `json:"TotalCount" xml:"TotalCount"`
	Data       []ListAppResourceAllocResponse `json:"Data" xml:"Data"`
}

// CreateListAppResourceAllocsRequest creates a request to invoke ListAppResourceAllocs API
func CreateListAppResourceAllocsRequest() (request *ListAppResourceAllocsRequest) {
	request = &ListAppResourceAllocsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("retailcloud", "2018-03-13", "ListAppResourceAllocs", "retailcloud", "openAPI")
	request.Method = requests.GET
	return
}

// CreateListAppResourceAllocsResponse creates a response to parse from ListAppResourceAllocs response
func CreateListAppResourceAllocsResponse() (response *ListAppResourceAllocsResponse) {
	response = &ListAppResourceAllocsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
