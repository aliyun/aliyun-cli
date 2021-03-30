package dataworks_public

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

// GetDataServiceGroup invokes the dataworks_public.GetDataServiceGroup API synchronously
func (client *Client) GetDataServiceGroup(request *GetDataServiceGroupRequest) (response *GetDataServiceGroupResponse, err error) {
	response = CreateGetDataServiceGroupResponse()
	err = client.DoAction(request, response)
	return
}

// GetDataServiceGroupWithChan invokes the dataworks_public.GetDataServiceGroup API asynchronously
func (client *Client) GetDataServiceGroupWithChan(request *GetDataServiceGroupRequest) (<-chan *GetDataServiceGroupResponse, <-chan error) {
	responseChan := make(chan *GetDataServiceGroupResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetDataServiceGroup(request)
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

// GetDataServiceGroupWithCallback invokes the dataworks_public.GetDataServiceGroup API asynchronously
func (client *Client) GetDataServiceGroupWithCallback(request *GetDataServiceGroupRequest, callback func(response *GetDataServiceGroupResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetDataServiceGroupResponse
		var err error
		defer close(result)
		response, err = client.GetDataServiceGroup(request)
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

// GetDataServiceGroupRequest is the request struct for api GetDataServiceGroup
type GetDataServiceGroupRequest struct {
	*requests.RpcRequest
	GroupId   string           `position:"Body" name:"GroupId"`
	TenantId  requests.Integer `position:"Body" name:"TenantId"`
	ProjectId requests.Integer `position:"Body" name:"ProjectId"`
}

// GetDataServiceGroupResponse is the response struct for api GetDataServiceGroup
type GetDataServiceGroupResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Group     Group  `json:"Group" xml:"Group"`
}

// CreateGetDataServiceGroupRequest creates a request to invoke GetDataServiceGroup API
func CreateGetDataServiceGroupRequest() (request *GetDataServiceGroupRequest) {
	request = &GetDataServiceGroupRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("dataworks-public", "2020-05-18", "GetDataServiceGroup", "", "")
	request.Method = requests.POST
	return
}

// CreateGetDataServiceGroupResponse creates a response to parse from GetDataServiceGroup response
func CreateGetDataServiceGroupResponse() (response *GetDataServiceGroupResponse) {
	response = &GetDataServiceGroupResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
