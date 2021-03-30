package green

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

// DescribeOssIncrementOverview invokes the green.DescribeOssIncrementOverview API synchronously
func (client *Client) DescribeOssIncrementOverview(request *DescribeOssIncrementOverviewRequest) (response *DescribeOssIncrementOverviewResponse, err error) {
	response = CreateDescribeOssIncrementOverviewResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeOssIncrementOverviewWithChan invokes the green.DescribeOssIncrementOverview API asynchronously
func (client *Client) DescribeOssIncrementOverviewWithChan(request *DescribeOssIncrementOverviewRequest) (<-chan *DescribeOssIncrementOverviewResponse, <-chan error) {
	responseChan := make(chan *DescribeOssIncrementOverviewResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeOssIncrementOverview(request)
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

// DescribeOssIncrementOverviewWithCallback invokes the green.DescribeOssIncrementOverview API asynchronously
func (client *Client) DescribeOssIncrementOverviewWithCallback(request *DescribeOssIncrementOverviewRequest, callback func(response *DescribeOssIncrementOverviewResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeOssIncrementOverviewResponse
		var err error
		defer close(result)
		response, err = client.DescribeOssIncrementOverview(request)
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

// DescribeOssIncrementOverviewRequest is the request struct for api DescribeOssIncrementOverview
type DescribeOssIncrementOverviewRequest struct {
	*requests.RpcRequest
	SourceIp string `position:"Query" name:"SourceIp"`
	Lang     string `position:"Query" name:"Lang"`
}

// DescribeOssIncrementOverviewResponse is the response struct for api DescribeOssIncrementOverview
type DescribeOssIncrementOverviewResponse struct {
	*responses.BaseResponse
	RequestId                  string `json:"RequestId" xml:"RequestId"`
	ImageCount                 int    `json:"ImageCount" xml:"ImageCount"`
	VideoCount                 int    `json:"VideoCount" xml:"VideoCount"`
	VideoFrameCount            int    `json:"VideoFrameCount" xml:"VideoFrameCount"`
	PornUnhandleCount          int    `json:"PornUnhandleCount" xml:"PornUnhandleCount"`
	TerrorismUnhandleCount     int    `json:"TerrorismUnhandleCount" xml:"TerrorismUnhandleCount"`
	AdUnhandleCount            int    `json:"AdUnhandleCount" xml:"AdUnhandleCount"`
	LiveUnhandleCount          int    `json:"LiveUnhandleCount" xml:"LiveUnhandleCount"`
	VoiceAntispamUnhandleCount int    `json:"VoiceAntispamUnhandleCount" xml:"VoiceAntispamUnhandleCount"`
	AudioCount                 int    `json:"AudioCount" xml:"AudioCount"`
}

// CreateDescribeOssIncrementOverviewRequest creates a request to invoke DescribeOssIncrementOverview API
func CreateDescribeOssIncrementOverviewRequest() (request *DescribeOssIncrementOverviewRequest) {
	request = &DescribeOssIncrementOverviewRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Green", "2017-08-23", "DescribeOssIncrementOverview", "green", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDescribeOssIncrementOverviewResponse creates a response to parse from DescribeOssIncrementOverview response
func CreateDescribeOssIncrementOverviewResponse() (response *DescribeOssIncrementOverviewResponse) {
	response = &DescribeOssIncrementOverviewResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
