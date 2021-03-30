package cloudcallcenter

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

// ListCorpIdentifyOssLink invokes the cloudcallcenter.ListCorpIdentifyOssLink API synchronously
// api document: https://help.aliyun.com/api/cloudcallcenter/listcorpidentifyosslink.html
func (client *Client) ListCorpIdentifyOssLink(request *ListCorpIdentifyOssLinkRequest) (response *ListCorpIdentifyOssLinkResponse, err error) {
	response = CreateListCorpIdentifyOssLinkResponse()
	err = client.DoAction(request, response)
	return
}

// ListCorpIdentifyOssLinkWithChan invokes the cloudcallcenter.ListCorpIdentifyOssLink API asynchronously
// api document: https://help.aliyun.com/api/cloudcallcenter/listcorpidentifyosslink.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) ListCorpIdentifyOssLinkWithChan(request *ListCorpIdentifyOssLinkRequest) (<-chan *ListCorpIdentifyOssLinkResponse, <-chan error) {
	responseChan := make(chan *ListCorpIdentifyOssLinkResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ListCorpIdentifyOssLink(request)
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

// ListCorpIdentifyOssLinkWithCallback invokes the cloudcallcenter.ListCorpIdentifyOssLink API asynchronously
// api document: https://help.aliyun.com/api/cloudcallcenter/listcorpidentifyosslink.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) ListCorpIdentifyOssLinkWithCallback(request *ListCorpIdentifyOssLinkRequest, callback func(response *ListCorpIdentifyOssLinkResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ListCorpIdentifyOssLinkResponse
		var err error
		defer close(result)
		response, err = client.ListCorpIdentifyOssLink(request)
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

// ListCorpIdentifyOssLinkRequest is the request struct for api ListCorpIdentifyOssLink
type ListCorpIdentifyOssLinkRequest struct {
	*requests.RpcRequest
	Pics string `position:"Query" name:"Pics"`
}

// ListCorpIdentifyOssLinkResponse is the response struct for api ListCorpIdentifyOssLink
type ListCorpIdentifyOssLinkResponse struct {
	*responses.BaseResponse
	RequestId      string                        `json:"RequestId" xml:"RequestId"`
	Success        bool                          `json:"Success" xml:"Success"`
	Code           string                        `json:"Code" xml:"Code"`
	Message        string                        `json:"Message" xml:"Message"`
	HttpStatusCode int                           `json:"HttpStatusCode" xml:"HttpStatusCode"`
	Data           DataInListCorpIdentifyOssLink `json:"Data" xml:"Data"`
}

// CreateListCorpIdentifyOssLinkRequest creates a request to invoke ListCorpIdentifyOssLink API
func CreateListCorpIdentifyOssLinkRequest() (request *ListCorpIdentifyOssLinkRequest) {
	request = &ListCorpIdentifyOssLinkRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("CloudCallCenter", "2017-07-05", "ListCorpIdentifyOssLink", "", "")
	request.Method = requests.POST
	return
}

// CreateListCorpIdentifyOssLinkResponse creates a response to parse from ListCorpIdentifyOssLink response
func CreateListCorpIdentifyOssLinkResponse() (response *ListCorpIdentifyOssLinkResponse) {
	response = &ListCorpIdentifyOssLinkResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
