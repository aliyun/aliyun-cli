package yundun_ds

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

// DescribeDataHubProjects invokes the yundun_ds.DescribeDataHubProjects API synchronously
// api document: https://help.aliyun.com/api/yundun-ds/describedatahubprojects.html
func (client *Client) DescribeDataHubProjects(request *DescribeDataHubProjectsRequest) (response *DescribeDataHubProjectsResponse, err error) {
	response = CreateDescribeDataHubProjectsResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeDataHubProjectsWithChan invokes the yundun_ds.DescribeDataHubProjects API asynchronously
// api document: https://help.aliyun.com/api/yundun-ds/describedatahubprojects.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeDataHubProjectsWithChan(request *DescribeDataHubProjectsRequest) (<-chan *DescribeDataHubProjectsResponse, <-chan error) {
	responseChan := make(chan *DescribeDataHubProjectsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeDataHubProjects(request)
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

// DescribeDataHubProjectsWithCallback invokes the yundun_ds.DescribeDataHubProjects API asynchronously
// api document: https://help.aliyun.com/api/yundun-ds/describedatahubprojects.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeDataHubProjectsWithCallback(request *DescribeDataHubProjectsRequest, callback func(response *DescribeDataHubProjectsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeDataHubProjectsResponse
		var err error
		defer close(result)
		response, err = client.DescribeDataHubProjects(request)
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

// DescribeDataHubProjectsRequest is the request struct for api DescribeDataHubProjects
type DescribeDataHubProjectsRequest struct {
	*requests.RpcRequest
	SourceIp    string           `position:"Query" name:"SourceIp"`
	FeatureType requests.Integer `position:"Query" name:"FeatureType"`
	PageSize    requests.Integer `position:"Query" name:"PageSize"`
	DepartId    requests.Integer `position:"Query" name:"DepartId"`
	CurrentPage requests.Integer `position:"Query" name:"CurrentPage"`
	Lang        string           `position:"Query" name:"Lang"`
	Key         string           `position:"Query" name:"Key"`
	QueryType   requests.Integer `position:"Query" name:"QueryType"`
}

// DescribeDataHubProjectsResponse is the response struct for api DescribeDataHubProjects
type DescribeDataHubProjectsResponse struct {
	*responses.BaseResponse
	RequestId   string    `json:"RequestId" xml:"RequestId"`
	PageSize    int       `json:"PageSize" xml:"PageSize"`
	CurrentPage int       `json:"CurrentPage" xml:"CurrentPage"`
	TotalCount  int       `json:"TotalCount" xml:"TotalCount"`
	Items       []Project `json:"Items" xml:"Items"`
}

// CreateDescribeDataHubProjectsRequest creates a request to invoke DescribeDataHubProjects API
func CreateDescribeDataHubProjectsRequest() (request *DescribeDataHubProjectsRequest) {
	request = &DescribeDataHubProjectsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Yundun-ds", "2019-01-03", "DescribeDataHubProjects", "sddp", "openAPI")
	return
}

// CreateDescribeDataHubProjectsResponse creates a response to parse from DescribeDataHubProjects response
func CreateDescribeDataHubProjectsResponse() (response *DescribeDataHubProjectsResponse) {
	response = &DescribeDataHubProjectsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
