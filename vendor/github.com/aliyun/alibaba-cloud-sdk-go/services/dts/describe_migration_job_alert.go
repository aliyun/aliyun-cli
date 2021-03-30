package dts

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

// DescribeMigrationJobAlert invokes the dts.DescribeMigrationJobAlert API synchronously
func (client *Client) DescribeMigrationJobAlert(request *DescribeMigrationJobAlertRequest) (response *DescribeMigrationJobAlertResponse, err error) {
	response = CreateDescribeMigrationJobAlertResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeMigrationJobAlertWithChan invokes the dts.DescribeMigrationJobAlert API asynchronously
func (client *Client) DescribeMigrationJobAlertWithChan(request *DescribeMigrationJobAlertRequest) (<-chan *DescribeMigrationJobAlertResponse, <-chan error) {
	responseChan := make(chan *DescribeMigrationJobAlertResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeMigrationJobAlert(request)
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

// DescribeMigrationJobAlertWithCallback invokes the dts.DescribeMigrationJobAlert API asynchronously
func (client *Client) DescribeMigrationJobAlertWithCallback(request *DescribeMigrationJobAlertRequest, callback func(response *DescribeMigrationJobAlertResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeMigrationJobAlertResponse
		var err error
		defer close(result)
		response, err = client.DescribeMigrationJobAlert(request)
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

// DescribeMigrationJobAlertRequest is the request struct for api DescribeMigrationJobAlert
type DescribeMigrationJobAlertRequest struct {
	*requests.RpcRequest
	ClientToken    string `position:"Query" name:"ClientToken"`
	MigrationJobId string `position:"Query" name:"MigrationJobId"`
	OwnerId        string `position:"Query" name:"OwnerId"`
	AccountId      string `position:"Query" name:"AccountId"`
}

// DescribeMigrationJobAlertResponse is the response struct for api DescribeMigrationJobAlert
type DescribeMigrationJobAlertResponse struct {
	*responses.BaseResponse
	DelayAlertPhone  string `json:"DelayAlertPhone" xml:"DelayAlertPhone"`
	DelayAlertStatus string `json:"DelayAlertStatus" xml:"DelayAlertStatus"`
	DelayOverSeconds string `json:"DelayOverSeconds" xml:"DelayOverSeconds"`
	ErrCode          string `json:"ErrCode" xml:"ErrCode"`
	ErrMessage       string `json:"ErrMessage" xml:"ErrMessage"`
	ErrorAlertPhone  string `json:"ErrorAlertPhone" xml:"ErrorAlertPhone"`
	ErrorAlertStatus string `json:"ErrorAlertStatus" xml:"ErrorAlertStatus"`
	MigrationJobId   string `json:"MigrationJobId" xml:"MigrationJobId"`
	MigrationJobName string `json:"MigrationJobName" xml:"MigrationJobName"`
	RequestId        string `json:"RequestId" xml:"RequestId"`
	Success          string `json:"Success" xml:"Success"`
}

// CreateDescribeMigrationJobAlertRequest creates a request to invoke DescribeMigrationJobAlert API
func CreateDescribeMigrationJobAlertRequest() (request *DescribeMigrationJobAlertRequest) {
	request = &DescribeMigrationJobAlertRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Dts", "2020-01-01", "DescribeMigrationJobAlert", "dts", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDescribeMigrationJobAlertResponse creates a response to parse from DescribeMigrationJobAlert response
func CreateDescribeMigrationJobAlertResponse() (response *DescribeMigrationJobAlertResponse) {
	response = &DescribeMigrationJobAlertResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
