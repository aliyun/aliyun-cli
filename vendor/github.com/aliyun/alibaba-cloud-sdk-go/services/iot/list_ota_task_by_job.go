package iot

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

// ListOTATaskByJob invokes the iot.ListOTATaskByJob API synchronously
func (client *Client) ListOTATaskByJob(request *ListOTATaskByJobRequest) (response *ListOTATaskByJobResponse, err error) {
	response = CreateListOTATaskByJobResponse()
	err = client.DoAction(request, response)
	return
}

// ListOTATaskByJobWithChan invokes the iot.ListOTATaskByJob API asynchronously
func (client *Client) ListOTATaskByJobWithChan(request *ListOTATaskByJobRequest) (<-chan *ListOTATaskByJobResponse, <-chan error) {
	responseChan := make(chan *ListOTATaskByJobResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ListOTATaskByJob(request)
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

// ListOTATaskByJobWithCallback invokes the iot.ListOTATaskByJob API asynchronously
func (client *Client) ListOTATaskByJobWithCallback(request *ListOTATaskByJobRequest, callback func(response *ListOTATaskByJobResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ListOTATaskByJobResponse
		var err error
		defer close(result)
		response, err = client.ListOTATaskByJob(request)
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

// ListOTATaskByJobRequest is the request struct for api ListOTATaskByJob
type ListOTATaskByJobRequest struct {
	*requests.RpcRequest
	JobId         string           `position:"Query" name:"JobId"`
	TaskStatus    string           `position:"Query" name:"TaskStatus"`
	IotInstanceId string           `position:"Query" name:"IotInstanceId"`
	PageSize      requests.Integer `position:"Query" name:"PageSize"`
	DeviceNames   *[]string        `position:"Query" name:"DeviceNames"  type:"Repeated"`
	CurrentPage   requests.Integer `position:"Query" name:"CurrentPage"`
	ApiProduct    string           `position:"Body" name:"ApiProduct"`
	ApiRevision   string           `position:"Body" name:"ApiRevision"`
}

// ListOTATaskByJobResponse is the response struct for api ListOTATaskByJob
type ListOTATaskByJobResponse struct {
	*responses.BaseResponse
	RequestId    string                 `json:"RequestId" xml:"RequestId"`
	Success      bool                   `json:"Success" xml:"Success"`
	Code         string                 `json:"Code" xml:"Code"`
	ErrorMessage string                 `json:"ErrorMessage" xml:"ErrorMessage"`
	Total        int                    `json:"Total" xml:"Total"`
	PageSize     int                    `json:"PageSize" xml:"PageSize"`
	PageCount    int                    `json:"PageCount" xml:"PageCount"`
	CurrentPage  int                    `json:"CurrentPage" xml:"CurrentPage"`
	Data         DataInListOTATaskByJob `json:"Data" xml:"Data"`
}

// CreateListOTATaskByJobRequest creates a request to invoke ListOTATaskByJob API
func CreateListOTATaskByJobRequest() (request *ListOTATaskByJobRequest) {
	request = &ListOTATaskByJobRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Iot", "2018-01-20", "ListOTATaskByJob", "iot", "openAPI")
	request.Method = requests.POST
	return
}

// CreateListOTATaskByJobResponse creates a response to parse from ListOTATaskByJob response
func CreateListOTATaskByJobResponse() (response *ListOTATaskByJobResponse) {
	response = &ListOTATaskByJobResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
