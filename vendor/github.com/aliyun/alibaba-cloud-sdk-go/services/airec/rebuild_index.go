package airec

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

// RebuildIndex invokes the airec.RebuildIndex API synchronously
func (client *Client) RebuildIndex(request *RebuildIndexRequest) (response *RebuildIndexResponse, err error) {
	response = CreateRebuildIndexResponse()
	err = client.DoAction(request, response)
	return
}

// RebuildIndexWithChan invokes the airec.RebuildIndex API asynchronously
func (client *Client) RebuildIndexWithChan(request *RebuildIndexRequest) (<-chan *RebuildIndexResponse, <-chan error) {
	responseChan := make(chan *RebuildIndexResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.RebuildIndex(request)
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

// RebuildIndexWithCallback invokes the airec.RebuildIndex API asynchronously
func (client *Client) RebuildIndexWithCallback(request *RebuildIndexRequest, callback func(response *RebuildIndexResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *RebuildIndexResponse
		var err error
		defer close(result)
		response, err = client.RebuildIndex(request)
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

// RebuildIndexRequest is the request struct for api RebuildIndex
type RebuildIndexRequest struct {
	*requests.RoaRequest
	InstanceId  string `position:"Path" name:"instanceId"`
	AlgorithmId string `position:"Path" name:"algorithmId"`
}

// RebuildIndexResponse is the response struct for api RebuildIndex
type RebuildIndexResponse struct {
	*responses.BaseResponse
	RequestId string `json:"requestId" xml:"requestId"`
	Result    string `json:"result" xml:"result"`
}

// CreateRebuildIndexRequest creates a request to invoke RebuildIndex API
func CreateRebuildIndexRequest() (request *RebuildIndexRequest) {
	request = &RebuildIndexRequest{
		RoaRequest: &requests.RoaRequest{},
	}
	request.InitWithApiInfo("Airec", "2020-11-26", "RebuildIndex", "/v2/openapi/instances/[instanceId]/filtering-algorithms/[algorithmId]/actions/rebuild", "airec", "openAPI")
	request.Method = requests.POST
	return
}

// CreateRebuildIndexResponse creates a response to parse from RebuildIndex response
func CreateRebuildIndexResponse() (response *RebuildIndexResponse) {
	response = &RebuildIndexResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
