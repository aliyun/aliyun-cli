package openanalytics_open

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

// RemoveEndPoint invokes the openanalytics_open.RemoveEndPoint API synchronously
func (client *Client) RemoveEndPoint(request *RemoveEndPointRequest) (response *RemoveEndPointResponse, err error) {
	response = CreateRemoveEndPointResponse()
	err = client.DoAction(request, response)
	return
}

// RemoveEndPointWithChan invokes the openanalytics_open.RemoveEndPoint API asynchronously
func (client *Client) RemoveEndPointWithChan(request *RemoveEndPointRequest) (<-chan *RemoveEndPointResponse, <-chan error) {
	responseChan := make(chan *RemoveEndPointResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.RemoveEndPoint(request)
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

// RemoveEndPointWithCallback invokes the openanalytics_open.RemoveEndPoint API asynchronously
func (client *Client) RemoveEndPointWithCallback(request *RemoveEndPointRequest, callback func(response *RemoveEndPointResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *RemoveEndPointResponse
		var err error
		defer close(result)
		response, err = client.RemoveEndPoint(request)
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

// RemoveEndPointRequest is the request struct for api RemoveEndPoint
type RemoveEndPointRequest struct {
	*requests.RpcRequest
	EndPointID string `position:"Body" name:"EndPointID"`
}

// RemoveEndPointResponse is the response struct for api RemoveEndPoint
type RemoveEndPointResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	RegionId  string `json:"RegionId" xml:"RegionId"`
}

// CreateRemoveEndPointRequest creates a request to invoke RemoveEndPoint API
func CreateRemoveEndPointRequest() (request *RemoveEndPointRequest) {
	request = &RemoveEndPointRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("openanalytics-open", "2018-06-19", "RemoveEndPoint", "openanalytics", "openAPI")
	request.Method = requests.POST
	return
}

// CreateRemoveEndPointResponse creates a response to parse from RemoveEndPoint response
func CreateRemoveEndPointResponse() (response *RemoveEndPointResponse) {
	response = &RemoveEndPointResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
