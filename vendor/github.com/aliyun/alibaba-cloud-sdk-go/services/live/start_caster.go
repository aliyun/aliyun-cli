package live

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

// StartCaster invokes the live.StartCaster API synchronously
func (client *Client) StartCaster(request *StartCasterRequest) (response *StartCasterResponse, err error) {
	response = CreateStartCasterResponse()
	err = client.DoAction(request, response)
	return
}

// StartCasterWithChan invokes the live.StartCaster API asynchronously
func (client *Client) StartCasterWithChan(request *StartCasterRequest) (<-chan *StartCasterResponse, <-chan error) {
	responseChan := make(chan *StartCasterResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.StartCaster(request)
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

// StartCasterWithCallback invokes the live.StartCaster API asynchronously
func (client *Client) StartCasterWithCallback(request *StartCasterRequest, callback func(response *StartCasterResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *StartCasterResponse
		var err error
		defer close(result)
		response, err = client.StartCaster(request)
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

// StartCasterRequest is the request struct for api StartCaster
type StartCasterRequest struct {
	*requests.RpcRequest
	CasterId string           `position:"Query" name:"CasterId"`
	OwnerId  requests.Integer `position:"Query" name:"OwnerId"`
}

// StartCasterResponse is the response struct for api StartCaster
type StartCasterResponse struct {
	*responses.BaseResponse
	RequestId     string        `json:"RequestId" xml:"RequestId"`
	PvwSceneInfos PvwSceneInfos `json:"PvwSceneInfos" xml:"PvwSceneInfos"`
	PgmSceneInfos PgmSceneInfos `json:"PgmSceneInfos" xml:"PgmSceneInfos"`
}

// CreateStartCasterRequest creates a request to invoke StartCaster API
func CreateStartCasterRequest() (request *StartCasterRequest) {
	request = &StartCasterRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("live", "2016-11-01", "StartCaster", "live", "openAPI")
	request.Method = requests.POST
	return
}

// CreateStartCasterResponse creates a response to parse from StartCaster response
func CreateStartCasterResponse() (response *StartCasterResponse) {
	response = &StartCasterResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
