package rtc

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

// DeleteChannel invokes the rtc.DeleteChannel API synchronously
func (client *Client) DeleteChannel(request *DeleteChannelRequest) (response *DeleteChannelResponse, err error) {
	response = CreateDeleteChannelResponse()
	err = client.DoAction(request, response)
	return
}

// DeleteChannelWithChan invokes the rtc.DeleteChannel API asynchronously
func (client *Client) DeleteChannelWithChan(request *DeleteChannelRequest) (<-chan *DeleteChannelResponse, <-chan error) {
	responseChan := make(chan *DeleteChannelResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DeleteChannel(request)
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

// DeleteChannelWithCallback invokes the rtc.DeleteChannel API asynchronously
func (client *Client) DeleteChannelWithCallback(request *DeleteChannelRequest, callback func(response *DeleteChannelResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DeleteChannelResponse
		var err error
		defer close(result)
		response, err = client.DeleteChannel(request)
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

// DeleteChannelRequest is the request struct for api DeleteChannel
type DeleteChannelRequest struct {
	*requests.RpcRequest
	ShowLog   string           `position:"Query" name:"ShowLog"`
	OwnerId   requests.Integer `position:"Query" name:"OwnerId"`
	AppId     string           `position:"Query" name:"AppId"`
	ChannelId string           `position:"Query" name:"ChannelId"`
}

// DeleteChannelResponse is the response struct for api DeleteChannel
type DeleteChannelResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateDeleteChannelRequest creates a request to invoke DeleteChannel API
func CreateDeleteChannelRequest() (request *DeleteChannelRequest) {
	request = &DeleteChannelRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("rtc", "2018-01-11", "DeleteChannel", "rtc", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDeleteChannelResponse creates a response to parse from DeleteChannel response
func CreateDeleteChannelResponse() (response *DeleteChannelResponse) {
	response = &DeleteChannelResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
