package outboundbot

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

// DeleteJobGroup invokes the outboundbot.DeleteJobGroup API synchronously
func (client *Client) DeleteJobGroup(request *DeleteJobGroupRequest) (response *DeleteJobGroupResponse, err error) {
	response = CreateDeleteJobGroupResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteJobGroupWithChan invokes the outboundbot.DeleteJobGroup API asynchronously
func (client *Client) DeleteJobGroupWithChan(request *DeleteJobGroupRequest) (<-chan *DeleteJobGroupResponse, <-chan error) {
	responseChan := make(chan *DeleteJobGroupResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteJobGroup(request)
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

// DeleteJobGroupWithCallback invokes the outboundbot.DeleteJobGroup API asynchronously
func (client *Client) DeleteJobGroupWithCallback(request *DeleteJobGroupRequest, callback func(response *DeleteJobGroupResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteJobGroupResponse
		var err error
		defer close(result)
		response, err = client.DeleteJobGroup(request)
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

// DeleteJobGroupRequest is the request struct for api DeleteJobGroup
type DeleteJobGroupRequest struct {
	*requests.RpcRequest
	InstanceId string `position:"Query" name:"InstanceId"`
	JobGroupId string `position:"Query" name:"JobGroupId"`
}

// DeleteJobGroupResponse is the response struct for api DeleteJobGroup
type DeleteJobGroupResponse struct {
	*responses.BaseResponse
	Code           string `json:"Code" xml:"Code"`
	HttpStatusCode int    `json:"HttpStatusCode" xml:"HttpStatusCode"`
	Message        string `json:"Message" xml:"Message"`
	RequestId      string `json:"RequestId" xml:"RequestId"`
	Success        bool   `json:"Success" xml:"Success"`
}

// CreateDeleteJobGroupRequest creates a request to invoke DeleteJobGroup API
func CreateDeleteJobGroupRequest() (request *DeleteJobGroupRequest) {
	request = &DeleteJobGroupRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("OutboundBot", "2019-12-26", "DeleteJobGroup", "outboundbot", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDeleteJobGroupResponse creates a response to parse from DeleteJobGroup response
func CreateDeleteJobGroupResponse() (response *DeleteJobGroupResponse) {
	response = &DeleteJobGroupResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
