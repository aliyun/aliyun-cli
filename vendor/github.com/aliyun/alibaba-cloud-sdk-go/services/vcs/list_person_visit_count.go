package vcs

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

// ListPersonVisitCount invokes the vcs.ListPersonVisitCount API synchronously
func (client *Client) ListPersonVisitCount(request *ListPersonVisitCountRequest) (response *ListPersonVisitCountResponse, err error) {
	response = CreateListPersonVisitCountResponse()
	err = client.DoAction(request, response)
	return
}

// ListPersonVisitCountWithChan invokes the vcs.ListPersonVisitCount API asynchronously
func (client *Client) ListPersonVisitCountWithChan(request *ListPersonVisitCountRequest) (<-chan *ListPersonVisitCountResponse, <-chan error) {
	responseChan := make(chan *ListPersonVisitCountResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ListPersonVisitCount(request)
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

// ListPersonVisitCountWithCallback invokes the vcs.ListPersonVisitCount API asynchronously
func (client *Client) ListPersonVisitCountWithCallback(request *ListPersonVisitCountRequest, callback func(response *ListPersonVisitCountResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ListPersonVisitCountResponse
		var err error
		defer close(result)
		response, err = client.ListPersonVisitCount(request)
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

// ListPersonVisitCountRequest is the request struct for api ListPersonVisitCount
type ListPersonVisitCountRequest struct {
	*requests.RpcRequest
	CorpId            string           `position:"Body" name:"CorpId"`
	EndTime           string           `position:"Body" name:"EndTime"`
	CountType         string           `position:"Body" name:"CountType"`
	StartTime         string           `position:"Body" name:"StartTime"`
	PageNumber        requests.Integer `position:"Body" name:"PageNumber"`
	TimeAggregateType string           `position:"Body" name:"TimeAggregateType"`
	MaxVal            requests.Integer `position:"Body" name:"MaxVal"`
	TagCode           string           `position:"Body" name:"TagCode"`
	MinVal            requests.Integer `position:"Body" name:"MinVal"`
	PageSize          requests.Integer `position:"Body" name:"PageSize"`
	AggregateType     string           `position:"Body" name:"AggregateType"`
}

// ListPersonVisitCountResponse is the response struct for api ListPersonVisitCount
type ListPersonVisitCountResponse struct {
	*responses.BaseResponse
	Code       string  `json:"Code" xml:"Code"`
	Message    string  `json:"Message" xml:"Message"`
	PageNo     string  `json:"PageNo" xml:"PageNo"`
	PageSize   string  `json:"PageSize" xml:"PageSize"`
	RequestId  string  `json:"RequestId" xml:"RequestId"`
	Success    string  `json:"Success" xml:"Success"`
	TotalCount string  `json:"TotalCount" xml:"TotalCount"`
	Data       []Datas `json:"Data" xml:"Data"`
}

// CreateListPersonVisitCountRequest creates a request to invoke ListPersonVisitCount API
func CreateListPersonVisitCountRequest() (request *ListPersonVisitCountRequest) {
	request = &ListPersonVisitCountRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Vcs", "2020-05-15", "ListPersonVisitCount", "", "")
	request.Method = requests.POST
	return
}

// CreateListPersonVisitCountResponse creates a response to parse from ListPersonVisitCount response
func CreateListPersonVisitCountResponse() (response *ListPersonVisitCountResponse) {
	response = &ListPersonVisitCountResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
