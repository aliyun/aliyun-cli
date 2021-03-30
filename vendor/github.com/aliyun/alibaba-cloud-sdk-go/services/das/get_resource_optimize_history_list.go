package das

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

// GetResourceOptimizeHistoryList invokes the das.GetResourceOptimizeHistoryList API synchronously
func (client *Client) GetResourceOptimizeHistoryList(request *GetResourceOptimizeHistoryListRequest) (response *GetResourceOptimizeHistoryListResponse, err error) {
	response = CreateGetResourceOptimizeHistoryListResponse()
	err = client.DoAction(request, response)
	return
}

// GetResourceOptimizeHistoryListWithChan invokes the das.GetResourceOptimizeHistoryList API asynchronously
func (client *Client) GetResourceOptimizeHistoryListWithChan(request *GetResourceOptimizeHistoryListRequest) (<-chan *GetResourceOptimizeHistoryListResponse, <-chan error) {
	responseChan := make(chan *GetResourceOptimizeHistoryListResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetResourceOptimizeHistoryList(request)
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

// GetResourceOptimizeHistoryListWithCallback invokes the das.GetResourceOptimizeHistoryList API asynchronously
func (client *Client) GetResourceOptimizeHistoryListWithCallback(request *GetResourceOptimizeHistoryListRequest, callback func(response *GetResourceOptimizeHistoryListResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetResourceOptimizeHistoryListResponse
		var err error
		defer close(result)
		response, err = client.GetResourceOptimizeHistoryList(request)
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

// GetResourceOptimizeHistoryListRequest is the request struct for api GetResourceOptimizeHistoryList
type GetResourceOptimizeHistoryListRequest struct {
	*requests.RpcRequest
	Context    string           `position:"Query" name:"__context"`
	Signature  string           `position:"Query" name:"Signature"`
	UserId     string           `position:"Query" name:"UserId"`
	Uid        string           `position:"Query" name:"Uid"`
	InstanceId string           `position:"Query" name:"InstanceId"`
	AccessKey  string           `position:"Query" name:"AccessKey"`
	PageSize   requests.Integer `position:"Query" name:"PageSize"`
	Page       requests.Integer `position:"Query" name:"Page"`
}

// GetResourceOptimizeHistoryListResponse is the response struct for api GetResourceOptimizeHistoryList
type GetResourceOptimizeHistoryListResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Code      string `json:"Code" xml:"Code"`
	Data      string `json:"Data" xml:"Data"`
	Message   string `json:"Message" xml:"Message"`
	Synchro   string `json:"Synchro" xml:"Synchro"`
	Success   string `json:"Success" xml:"Success"`
}

// CreateGetResourceOptimizeHistoryListRequest creates a request to invoke GetResourceOptimizeHistoryList API
func CreateGetResourceOptimizeHistoryListRequest() (request *GetResourceOptimizeHistoryListRequest) {
	request = &GetResourceOptimizeHistoryListRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("DAS", "2020-01-16", "GetResourceOptimizeHistoryList", "das", "openAPI")
	request.Method = requests.POST
	return
}

// CreateGetResourceOptimizeHistoryListResponse creates a response to parse from GetResourceOptimizeHistoryList response
func CreateGetResourceOptimizeHistoryListResponse() (response *GetResourceOptimizeHistoryListResponse) {
	response = &GetResourceOptimizeHistoryListResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
