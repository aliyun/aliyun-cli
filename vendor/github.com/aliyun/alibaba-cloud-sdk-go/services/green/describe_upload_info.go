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

// DescribeUploadInfo invokes the green.DescribeUploadInfo API synchronously
func (client *Client) DescribeUploadInfo(request *DescribeUploadInfoRequest) (response *DescribeUploadInfoResponse, err error) {
	response = CreateDescribeUploadInfoResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeUploadInfoWithChan invokes the green.DescribeUploadInfo API asynchronously
func (client *Client) DescribeUploadInfoWithChan(request *DescribeUploadInfoRequest) (<-chan *DescribeUploadInfoResponse, <-chan error) {
	responseChan := make(chan *DescribeUploadInfoResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeUploadInfo(request)
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

// DescribeUploadInfoWithCallback invokes the green.DescribeUploadInfo API asynchronously
func (client *Client) DescribeUploadInfoWithCallback(request *DescribeUploadInfoRequest, callback func(response *DescribeUploadInfoResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeUploadInfoResponse
		var err error
		defer close(result)
		response, err = client.DescribeUploadInfo(request)
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

// DescribeUploadInfoRequest is the request struct for api DescribeUploadInfo
type DescribeUploadInfoRequest struct {
	*requests.RpcRequest
	Biz      string `position:"Query" name:"Biz"`
	SourceIp string `position:"Query" name:"SourceIp"`
	Lang     string `position:"Query" name:"Lang"`
}

// DescribeUploadInfoResponse is the response struct for api DescribeUploadInfo
type DescribeUploadInfoResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Accessid  string `json:"Accessid" xml:"Accessid"`
	Policy    string `json:"Policy" xml:"Policy"`
	Signature string `json:"Signature" xml:"Signature"`
	Folder    string `json:"Folder" xml:"Folder"`
	Host      string `json:"Host" xml:"Host"`
	Expire    int    `json:"Expire" xml:"Expire"`
}

// CreateDescribeUploadInfoRequest creates a request to invoke DescribeUploadInfo API
func CreateDescribeUploadInfoRequest() (request *DescribeUploadInfoRequest) {
	request = &DescribeUploadInfoRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Green", "2017-08-23", "DescribeUploadInfo", "green", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDescribeUploadInfoResponse creates a response to parse from DescribeUploadInfo response
func CreateDescribeUploadInfoResponse() (response *DescribeUploadInfoResponse) {
	response = &DescribeUploadInfoResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
