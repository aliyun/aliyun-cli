package ehpc

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

// GetAccountingReport invokes the ehpc.GetAccountingReport API synchronously
func (client *Client) GetAccountingReport(request *GetAccountingReportRequest) (response *GetAccountingReportResponse, err error) {
	response = CreateGetAccountingReportResponse()
	err = client.DoAction(request, response)
	return
}

// GetAccountingReportWithChan invokes the ehpc.GetAccountingReport API asynchronously
func (client *Client) GetAccountingReportWithChan(request *GetAccountingReportRequest) (<-chan *GetAccountingReportResponse, <-chan error) {
	responseChan := make(chan *GetAccountingReportResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetAccountingReport(request)
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

// GetAccountingReportWithCallback invokes the ehpc.GetAccountingReport API asynchronously
func (client *Client) GetAccountingReportWithCallback(request *GetAccountingReportRequest, callback func(response *GetAccountingReportResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetAccountingReportResponse
		var err error
		defer close(result)
		response, err = client.GetAccountingReport(request)
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

// GetAccountingReportRequest is the request struct for api GetAccountingReport
type GetAccountingReportRequest struct {
	*requests.RpcRequest
	ReportType  string           `position:"Query" name:"ReportType"`
	EndTime     requests.Integer `position:"Query" name:"EndTime"`
	FilterValue string           `position:"Query" name:"FilterValue"`
	Dim         string           `position:"Query" name:"Dim"`
	ClusterId   string           `position:"Query" name:"ClusterId"`
	StartTime   requests.Integer `position:"Query" name:"StartTime"`
	PageNumber  requests.Integer `position:"Query" name:"PageNumber"`
	JobId       string           `position:"Query" name:"JobId"`
	PageSize    requests.Integer `position:"Query" name:"PageSize"`
}

// GetAccountingReportResponse is the response struct for api GetAccountingReport
type GetAccountingReportResponse struct {
	*responses.BaseResponse
	RequestId     string `json:"RequestId" xml:"RequestId"`
	Metrics       string `json:"Metrics" xml:"Metrics"`
	TotalCoreTime int    `json:"TotalCoreTime" xml:"TotalCoreTime"`
	TotalCount    int    `json:"TotalCount" xml:"TotalCount"`
	PageSize      int    `json:"PageSize" xml:"PageSize"`
	PageNumber    int    `json:"PageNumber" xml:"PageNumber"`
	Data          Data   `json:"Data" xml:"Data"`
}

// CreateGetAccountingReportRequest creates a request to invoke GetAccountingReport API
func CreateGetAccountingReportRequest() (request *GetAccountingReportRequest) {
	request = &GetAccountingReportRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("EHPC", "2018-04-12", "GetAccountingReport", "", "")
	request.Method = requests.GET
	return
}

// CreateGetAccountingReportResponse creates a response to parse from GetAccountingReport response
func CreateGetAccountingReportResponse() (response *GetAccountingReportResponse) {
	response = &GetAccountingReportResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
