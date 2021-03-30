package ehpc

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

// ListSecurityGroups invokes the ehpc.ListSecurityGroups API synchronously
func (client *Client) ListSecurityGroups(request *ListSecurityGroupsRequest) (response *ListSecurityGroupsResponse, err error) {
	response = CreateListSecurityGroupsResponse()
	err = client.DoAction(request, response)
	return
}

// ListSecurityGroupsWithChan invokes the ehpc.ListSecurityGroups API asynchronously
func (client *Client) ListSecurityGroupsWithChan(request *ListSecurityGroupsRequest) (<-chan *ListSecurityGroupsResponse, <-chan error) {
	responseChan := make(chan *ListSecurityGroupsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ListSecurityGroups(request)
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

// ListSecurityGroupsWithCallback invokes the ehpc.ListSecurityGroups API asynchronously
func (client *Client) ListSecurityGroupsWithCallback(request *ListSecurityGroupsRequest, callback func(response *ListSecurityGroupsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ListSecurityGroupsResponse
		var err error
		defer close(result)
		response, err = client.ListSecurityGroups(request)
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

// ListSecurityGroupsRequest is the request struct for api ListSecurityGroups
type ListSecurityGroupsRequest struct {
	*requests.RpcRequest
	ClusterId string `position:"Query" name:"ClusterId"`
}

// ListSecurityGroupsResponse is the response struct for api ListSecurityGroups
type ListSecurityGroupsResponse struct {
	*responses.BaseResponse
	RequestId      string         `json:"RequestId" xml:"RequestId"`
	TotalCount     int            `json:"TotalCount" xml:"TotalCount"`
	SecurityGroups SecurityGroups `json:"SecurityGroups" xml:"SecurityGroups"`
}

// CreateListSecurityGroupsRequest creates a request to invoke ListSecurityGroups API
func CreateListSecurityGroupsRequest() (request *ListSecurityGroupsRequest) {
	request = &ListSecurityGroupsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("EHPC", "2018-04-12", "ListSecurityGroups", "", "")
	request.Method = requests.GET
	return
}

// CreateListSecurityGroupsResponse creates a response to parse from ListSecurityGroups response
func CreateListSecurityGroupsResponse() (response *ListSecurityGroupsResponse) {
	response = &ListSecurityGroupsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
