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

// DescribeRTCAppKey invokes the rtc.DescribeRTCAppKey API synchronously
func (client *Client) DescribeRTCAppKey(request *DescribeRTCAppKeyRequest) (response *DescribeRTCAppKeyResponse, err error) {
	response = CreateDescribeRTCAppKeyResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeRTCAppKeyWithChan invokes the rtc.DescribeRTCAppKey API asynchronously
func (client *Client) DescribeRTCAppKeyWithChan(request *DescribeRTCAppKeyRequest) (<-chan *DescribeRTCAppKeyResponse, <-chan error) {
	responseChan := make(chan *DescribeRTCAppKeyResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeRTCAppKey(request)
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

// DescribeRTCAppKeyWithCallback invokes the rtc.DescribeRTCAppKey API asynchronously
func (client *Client) DescribeRTCAppKeyWithCallback(request *DescribeRTCAppKeyRequest, callback func(response *DescribeRTCAppKeyResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeRTCAppKeyResponse
		var err error
		defer close(result)
		response, err = client.DescribeRTCAppKey(request)
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

// DescribeRTCAppKeyRequest is the request struct for api DescribeRTCAppKey
type DescribeRTCAppKeyRequest struct {
	*requests.RpcRequest
	ShowLog string           `position:"Query" name:"ShowLog"`
	OwnerId requests.Integer `position:"Query" name:"OwnerId"`
	AppId   string           `position:"Query" name:"AppId"`
}

// DescribeRTCAppKeyResponse is the response struct for api DescribeRTCAppKey
type DescribeRTCAppKeyResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	AppKey    string `json:"AppKey" xml:"AppKey"`
}

// CreateDescribeRTCAppKeyRequest creates a request to invoke DescribeRTCAppKey API
func CreateDescribeRTCAppKeyRequest() (request *DescribeRTCAppKeyRequest) {
	request = &DescribeRTCAppKeyRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("rtc", "2018-01-11", "DescribeRTCAppKey", "rtc", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDescribeRTCAppKeyResponse creates a response to parse from DescribeRTCAppKey response
func CreateDescribeRTCAppKeyResponse() (response *DescribeRTCAppKeyResponse) {
	response = &DescribeRTCAppKeyResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
