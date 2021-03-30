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

// ListSurveys invokes the cloudcallcenter.ListSurveys API synchronously
// api document: https://help.aliyun.com/api/cloudcallcenter/listsurveys.html
func (client *Client) ListSurveys(request *ListSurveysRequest) (response *ListSurveysResponse, err error) {
	response = CreateListSurveysResponse()
	err = client.DoAction(request, response)
	return
}

// ListSurveysWithChan invokes the cloudcallcenter.ListSurveys API asynchronously
// api document: https://help.aliyun.com/api/cloudcallcenter/listsurveys.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) ListSurveysWithChan(request *ListSurveysRequest) (<-chan *ListSurveysResponse, <-chan error) {
	responseChan := make(chan *ListSurveysResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ListSurveys(request)
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

// ListSurveysWithCallback invokes the cloudcallcenter.ListSurveys API asynchronously
// api document: https://help.aliyun.com/api/cloudcallcenter/listsurveys.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) ListSurveysWithCallback(request *ListSurveysRequest, callback func(response *ListSurveysResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ListSurveysResponse
		var err error
		defer close(result)
		response, err = client.ListSurveys(request)
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

// ListSurveysRequest is the request struct for api ListSurveys
type ListSurveysRequest struct {
	*requests.RpcRequest
	InstanceId string `position:"Query" name:"InstanceId"`
	ScenarioId string `position:"Query" name:"ScenarioId"`
}

// ListSurveysResponse is the response struct for api ListSurveys
type ListSurveysResponse struct {
	*responses.BaseResponse
	RequestId      string   `json:"RequestId" xml:"RequestId"`
	Success        bool     `json:"Success" xml:"Success"`
	Code           string   `json:"Code" xml:"Code"`
	Message        string   `json:"Message" xml:"Message"`
	HttpStatusCode int      `json:"HttpStatusCode" xml:"HttpStatusCode"`
	Surveys        []Survey `json:"Surveys" xml:"Surveys"`
}

// CreateListSurveysRequest creates a request to invoke ListSurveys API
func CreateListSurveysRequest() (request *ListSurveysRequest) {
	request = &ListSurveysRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("CloudCallCenter", "2017-07-05", "ListSurveys", "", "")
	request.Method = requests.POST
	return
}

// CreateListSurveysResponse creates a response to parse from ListSurveys response
func CreateListSurveysResponse() (response *ListSurveysResponse) {
	response = &ListSurveysResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
