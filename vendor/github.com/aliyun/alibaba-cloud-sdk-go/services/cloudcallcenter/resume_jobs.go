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

// ResumeJobs invokes the cloudcallcenter.ResumeJobs API synchronously
// api document: https://help.aliyun.com/api/cloudcallcenter/resumejobs.html
func (client *Client) ResumeJobs(request *ResumeJobsRequest) (response *ResumeJobsResponse, err error) {
	response = CreateResumeJobsResponse()
	err = client.DoAction(request, response)
	return
}

// ResumeJobsWithChan invokes the cloudcallcenter.ResumeJobs API asynchronously
// api document: https://help.aliyun.com/api/cloudcallcenter/resumejobs.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) ResumeJobsWithChan(request *ResumeJobsRequest) (<-chan *ResumeJobsResponse, <-chan error) {
	responseChan := make(chan *ResumeJobsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ResumeJobs(request)
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

// ResumeJobsWithCallback invokes the cloudcallcenter.ResumeJobs API asynchronously
// api document: https://help.aliyun.com/api/cloudcallcenter/resumejobs.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) ResumeJobsWithCallback(request *ResumeJobsRequest, callback func(response *ResumeJobsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ResumeJobsResponse
		var err error
		defer close(result)
		response, err = client.ResumeJobs(request)
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

// ResumeJobsRequest is the request struct for api ResumeJobs
type ResumeJobsRequest struct {
	*requests.RpcRequest
	All            requests.Boolean `position:"Query" name:"All"`
	JobReferenceId *[]string        `position:"Query" name:"JobReferenceId"  type:"Repeated"`
	GroupId        string           `position:"Query" name:"GroupId"`
	JobId          *[]string        `position:"Query" name:"JobId"  type:"Repeated"`
	InstanceId     string           `position:"Query" name:"InstanceId"`
	ScenarioId     string           `position:"Query" name:"ScenarioId"`
}

// ResumeJobsResponse is the response struct for api ResumeJobs
type ResumeJobsResponse struct {
	*responses.BaseResponse
	RequestId      string `json:"RequestId" xml:"RequestId"`
	Success        bool   `json:"Success" xml:"Success"`
	Code           string `json:"Code" xml:"Code"`
	Message        string `json:"Message" xml:"Message"`
	HttpStatusCode int    `json:"HttpStatusCode" xml:"HttpStatusCode"`
}

// CreateResumeJobsRequest creates a request to invoke ResumeJobs API
func CreateResumeJobsRequest() (request *ResumeJobsRequest) {
	request = &ResumeJobsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("CloudCallCenter", "2017-07-05", "ResumeJobs", "", "")
	request.Method = requests.POST
	return
}

// CreateResumeJobsResponse creates a response to parse from ResumeJobs response
func CreateResumeJobsResponse() (response *ResumeJobsResponse) {
	response = &ResumeJobsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
