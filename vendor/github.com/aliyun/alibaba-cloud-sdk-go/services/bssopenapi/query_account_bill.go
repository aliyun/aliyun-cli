package bssopenapi

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

// QueryAccountBill invokes the bssopenapi.QueryAccountBill API synchronously
func (client *Client) QueryAccountBill(request *QueryAccountBillRequest) (response *QueryAccountBillResponse, err error) {
	response = CreateQueryAccountBillResponse()
	err = client.DoAction(request, response)
	return
}

// QueryAccountBillWithChan invokes the bssopenapi.QueryAccountBill API asynchronously
func (client *Client) QueryAccountBillWithChan(request *QueryAccountBillRequest) (<-chan *QueryAccountBillResponse, <-chan error) {
	responseChan := make(chan *QueryAccountBillResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.QueryAccountBill(request)
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

// QueryAccountBillWithCallback invokes the bssopenapi.QueryAccountBill API asynchronously
func (client *Client) QueryAccountBillWithCallback(request *QueryAccountBillRequest, callback func(response *QueryAccountBillResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *QueryAccountBillResponse
		var err error
		defer close(result)
		response, err = client.QueryAccountBill(request)
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

// QueryAccountBillRequest is the request struct for api QueryAccountBill
type QueryAccountBillRequest struct {
	*requests.RpcRequest
	ProductCode      string           `position:"Query" name:"ProductCode"`
	BillingCycle     string           `position:"Query" name:"BillingCycle"`
	PageNum          requests.Integer `position:"Query" name:"PageNum"`
	OwnerID          requests.Integer `position:"Query" name:"OwnerID"`
	BillOwnerId      requests.Integer `position:"Query" name:"BillOwnerId"`
	BillingDate      string           `position:"Query" name:"BillingDate"`
	IsGroupByProduct requests.Boolean `position:"Query" name:"IsGroupByProduct"`
	Granularity      string           `position:"Query" name:"Granularity"`
	PageSize         requests.Integer `position:"Query" name:"PageSize"`
}

// QueryAccountBillResponse is the response struct for api QueryAccountBill
type QueryAccountBillResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	Code      string `json:"Code" xml:"Code"`
	Message   string `json:"Message" xml:"Message"`
	Data      Data   `json:"Data" xml:"Data"`
}

// CreateQueryAccountBillRequest creates a request to invoke QueryAccountBill API
func CreateQueryAccountBillRequest() (request *QueryAccountBillRequest) {
	request = &QueryAccountBillRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("BssOpenApi", "2017-12-14", "QueryAccountBill", "", "")
	request.Method = requests.POST
	return
}

// CreateQueryAccountBillResponse creates a response to parse from QueryAccountBill response
func CreateQueryAccountBillResponse() (response *QueryAccountBillResponse) {
	response = &QueryAccountBillResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
