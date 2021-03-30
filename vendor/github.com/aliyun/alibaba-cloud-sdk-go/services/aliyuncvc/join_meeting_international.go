package aliyuncvc

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

// JoinMeetingInternational invokes the aliyuncvc.JoinMeetingInternational API synchronously
func (client *Client) JoinMeetingInternational(request *JoinMeetingInternationalRequest) (response *JoinMeetingInternationalResponse, err error) {
	response = CreateJoinMeetingInternationalResponse()
	err = client.DoAction(request, response)
	return
}

// JoinMeetingInternationalWithChan invokes the aliyuncvc.JoinMeetingInternational API asynchronously
func (client *Client) JoinMeetingInternationalWithChan(request *JoinMeetingInternationalRequest) (<-chan *JoinMeetingInternationalResponse, <-chan error) {
	responseChan := make(chan *JoinMeetingInternationalResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.JoinMeetingInternational(request)
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

// JoinMeetingInternationalWithCallback invokes the aliyuncvc.JoinMeetingInternational API asynchronously
func (client *Client) JoinMeetingInternationalWithCallback(request *JoinMeetingInternationalRequest, callback func(response *JoinMeetingInternationalResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *JoinMeetingInternationalResponse
		var err error
		defer close(result)
		response, err = client.JoinMeetingInternational(request)
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

// JoinMeetingInternationalRequest is the request struct for api JoinMeetingInternational
type JoinMeetingInternationalRequest struct {
	*requests.RpcRequest
	UserId      string `position:"Body" name:"UserId"`
	Password    string `position:"Body" name:"Password"`
	MeetingCode string `position:"Body" name:"MeetingCode"`
}

// JoinMeetingInternationalResponse is the response struct for api JoinMeetingInternational
type JoinMeetingInternationalResponse struct {
	*responses.BaseResponse
	ErrorCode   int         `json:"ErrorCode" xml:"ErrorCode"`
	Success     bool        `json:"Success" xml:"Success"`
	RequestId   string      `json:"RequestId" xml:"RequestId"`
	Message     string      `json:"Message" xml:"Message"`
	MeetingInfo MeetingInfo `json:"MeetingInfo" xml:"MeetingInfo"`
}

// CreateJoinMeetingInternationalRequest creates a request to invoke JoinMeetingInternational API
func CreateJoinMeetingInternationalRequest() (request *JoinMeetingInternationalRequest) {
	request = &JoinMeetingInternationalRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("aliyuncvc", "2019-10-30", "JoinMeetingInternational", "aliyuncvc", "openAPI")
	request.Method = requests.POST
	return
}

// CreateJoinMeetingInternationalResponse creates a response to parse from JoinMeetingInternational response
func CreateJoinMeetingInternationalResponse() (response *JoinMeetingInternationalResponse) {
	response = &JoinMeetingInternationalResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
