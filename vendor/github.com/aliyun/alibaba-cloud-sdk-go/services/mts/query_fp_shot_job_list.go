package mts

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

// QueryFpShotJobList invokes the mts.QueryFpShotJobList API synchronously
func (client *Client) QueryFpShotJobList(request *QueryFpShotJobListRequest) (response *QueryFpShotJobListResponse, err error) {
	response = CreateQueryFpShotJobListResponse()
	err = client.DoAction(request, response)
	return
}

// QueryFpShotJobListWithChan invokes the mts.QueryFpShotJobList API asynchronously
func (client *Client) QueryFpShotJobListWithChan(request *QueryFpShotJobListRequest) (<-chan *QueryFpShotJobListResponse, <-chan error) {
	responseChan := make(chan *QueryFpShotJobListResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.QueryFpShotJobList(request)
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

// QueryFpShotJobListWithCallback invokes the mts.QueryFpShotJobList API asynchronously
func (client *Client) QueryFpShotJobListWithCallback(request *QueryFpShotJobListRequest, callback func(response *QueryFpShotJobListResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *QueryFpShotJobListResponse
		var err error
		defer close(result)
		response, err = client.QueryFpShotJobList(request)
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

// QueryFpShotJobListRequest is the request struct for api QueryFpShotJobList
type QueryFpShotJobListRequest struct {
	*requests.RpcRequest
	ResourceOwnerId            requests.Integer `position:"Query" name:"ResourceOwnerId"`
	NextPageToken              string           `position:"Query" name:"NextPageToken"`
	StartOfJobCreatedTimeRange string           `position:"Query" name:"StartOfJobCreatedTimeRange"`
	State                      string           `position:"Query" name:"State"`
	EndOfJobCreatedTimeRange   string           `position:"Query" name:"EndOfJobCreatedTimeRange"`
	ResourceOwnerAccount       string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount               string           `position:"Query" name:"OwnerAccount"`
	MaximumPageSize            requests.Integer `position:"Query" name:"MaximumPageSize"`
	OwnerId                    requests.Integer `position:"Query" name:"OwnerId"`
	PipelineId                 string           `position:"Query" name:"PipelineId"`
	PrimaryKeyList             string           `position:"Query" name:"PrimaryKeyList"`
	JobIds                     string           `position:"Query" name:"JobIds"`
}

// QueryFpShotJobListResponse is the response struct for api QueryFpShotJobList
type QueryFpShotJobListResponse struct {
	*responses.BaseResponse
	RequestId           string                          `json:"RequestId" xml:"RequestId"`
	NextPageToken       string                          `json:"NextPageToken" xml:"NextPageToken"`
	NonExistIds         NonExistIdsInQueryFpShotJobList `json:"NonExistIds" xml:"NonExistIds"`
	NonExistPrimaryKeys NonExistPrimaryKeys             `json:"NonExistPrimaryKeys" xml:"NonExistPrimaryKeys"`
	FpShotJobList       FpShotJobList                   `json:"FpShotJobList" xml:"FpShotJobList"`
}

// CreateQueryFpShotJobListRequest creates a request to invoke QueryFpShotJobList API
func CreateQueryFpShotJobListRequest() (request *QueryFpShotJobListRequest) {
	request = &QueryFpShotJobListRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Mts", "2014-06-18", "QueryFpShotJobList", "mts", "openAPI")
	request.Method = requests.POST
	return
}

// CreateQueryFpShotJobListResponse creates a response to parse from QueryFpShotJobList response
func CreateQueryFpShotJobListResponse() (response *QueryFpShotJobListResponse) {
	response = &QueryFpShotJobListResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
