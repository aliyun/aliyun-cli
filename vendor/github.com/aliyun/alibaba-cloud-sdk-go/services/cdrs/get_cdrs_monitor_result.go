package cdrs

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

// GetCdrsMonitorResult invokes the cdrs.GetCdrsMonitorResult API synchronously
func (client *Client) GetCdrsMonitorResult(request *GetCdrsMonitorResultRequest) (response *GetCdrsMonitorResultResponse, err error) {
	response = CreateGetCdrsMonitorResultResponse()
	err = client.DoAction(request, response)
	return
}

// GetCdrsMonitorResultWithChan invokes the cdrs.GetCdrsMonitorResult API asynchronously
func (client *Client) GetCdrsMonitorResultWithChan(request *GetCdrsMonitorResultRequest) (<-chan *GetCdrsMonitorResultResponse, <-chan error) {
	responseChan := make(chan *GetCdrsMonitorResultResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetCdrsMonitorResult(request)
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

// GetCdrsMonitorResultWithCallback invokes the cdrs.GetCdrsMonitorResult API asynchronously
func (client *Client) GetCdrsMonitorResultWithCallback(request *GetCdrsMonitorResultRequest, callback func(response *GetCdrsMonitorResultResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetCdrsMonitorResultResponse
		var err error
		defer close(result)
		response, err = client.GetCdrsMonitorResult(request)
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

// GetCdrsMonitorResultRequest is the request struct for api GetCdrsMonitorResult
type GetCdrsMonitorResultRequest struct {
	*requests.RpcRequest
	CorpId          string           `position:"Body" name:"CorpId"`
	EndTime         requests.Integer `position:"Body" name:"EndTime"`
	StartTime       requests.Integer `position:"Body" name:"StartTime"`
	BizId           string           `position:"Body" name:"BizId"`
	AlgorithmVendor string           `position:"Body" name:"AlgorithmVendor"`
	MinRecordId     string           `position:"Body" name:"MinRecordId"`
	TaskId          string           `position:"Body" name:"TaskId"`
}

// GetCdrsMonitorResultResponse is the response struct for api GetCdrsMonitorResult
type GetCdrsMonitorResultResponse struct {
	*responses.BaseResponse
	Code      string                     `json:"Code" xml:"Code"`
	Message   string                     `json:"Message" xml:"Message"`
	RequestId string                     `json:"RequestId" xml:"RequestId"`
	Data      DataInGetCdrsMonitorResult `json:"Data" xml:"Data"`
}

// CreateGetCdrsMonitorResultRequest creates a request to invoke GetCdrsMonitorResult API
func CreateGetCdrsMonitorResultRequest() (request *GetCdrsMonitorResultRequest) {
	request = &GetCdrsMonitorResultRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("CDRS", "2020-11-01", "GetCdrsMonitorResult", "", "")
	request.Method = requests.POST
	return
}

// CreateGetCdrsMonitorResultResponse creates a response to parse from GetCdrsMonitorResult response
func CreateGetCdrsMonitorResultResponse() (response *GetCdrsMonitorResultResponse) {
	response = &GetCdrsMonitorResultResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
