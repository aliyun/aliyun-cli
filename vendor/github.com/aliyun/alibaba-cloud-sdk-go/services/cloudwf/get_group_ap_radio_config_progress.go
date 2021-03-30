package cloudwf

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

// GetGroupApRadioConfigProgress invokes the cloudwf.GetGroupApRadioConfigProgress API synchronously
// api document: https://help.aliyun.com/api/cloudwf/getgroupapradioconfigprogress.html
func (client *Client) GetGroupApRadioConfigProgress(request *GetGroupApRadioConfigProgressRequest) (response *GetGroupApRadioConfigProgressResponse, err error) {
	response = CreateGetGroupApRadioConfigProgressResponse()
	err = client.DoAction(request, response)
	return
}

// GetGroupApRadioConfigProgressWithChan invokes the cloudwf.GetGroupApRadioConfigProgress API asynchronously
// api document: https://help.aliyun.com/api/cloudwf/getgroupapradioconfigprogress.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) GetGroupApRadioConfigProgressWithChan(request *GetGroupApRadioConfigProgressRequest) (<-chan *GetGroupApRadioConfigProgressResponse, <-chan error) {
	responseChan := make(chan *GetGroupApRadioConfigProgressResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetGroupApRadioConfigProgress(request)
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

// GetGroupApRadioConfigProgressWithCallback invokes the cloudwf.GetGroupApRadioConfigProgress API asynchronously
// api document: https://help.aliyun.com/api/cloudwf/getgroupapradioconfigprogress.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) GetGroupApRadioConfigProgressWithCallback(request *GetGroupApRadioConfigProgressRequest, callback func(response *GetGroupApRadioConfigProgressResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetGroupApRadioConfigProgressResponse
		var err error
		defer close(result)
		response, err = client.GetGroupApRadioConfigProgress(request)
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

// GetGroupApRadioConfigProgressRequest is the request struct for api GetGroupApRadioConfigProgress
type GetGroupApRadioConfigProgressRequest struct {
	*requests.RpcRequest
	Id requests.Integer `position:"Query" name:"Id"`
}

// GetGroupApRadioConfigProgressResponse is the response struct for api GetGroupApRadioConfigProgress
type GetGroupApRadioConfigProgressResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	Message   string `json:"Message" xml:"Message"`
	Data      string `json:"Data" xml:"Data"`
	ErrorCode int    `json:"ErrorCode" xml:"ErrorCode"`
	ErrorMsg  string `json:"ErrorMsg" xml:"ErrorMsg"`
}

// CreateGetGroupApRadioConfigProgressRequest creates a request to invoke GetGroupApRadioConfigProgress API
func CreateGetGroupApRadioConfigProgressRequest() (request *GetGroupApRadioConfigProgressRequest) {
	request = &GetGroupApRadioConfigProgressRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("cloudwf", "2017-03-28", "GetGroupApRadioConfigProgress", "cloudwf", "openAPI")
	return
}

// CreateGetGroupApRadioConfigProgressResponse creates a response to parse from GetGroupApRadioConfigProgress response
func CreateGetGroupApRadioConfigProgressResponse() (response *GetGroupApRadioConfigProgressResponse) {
	response = &GetGroupApRadioConfigProgressResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
