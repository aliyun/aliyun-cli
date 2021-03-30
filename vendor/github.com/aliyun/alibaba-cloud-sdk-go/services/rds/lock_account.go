package rds

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

// LockAccount invokes the rds.LockAccount API synchronously
func (client *Client) LockAccount(request *LockAccountRequest) (response *LockAccountResponse, err error) {
	response = CreateLockAccountResponse()
	err = client.DoAction(request, response)
	return
}

// LockAccountWithChan invokes the rds.LockAccount API asynchronously
func (client *Client) LockAccountWithChan(request *LockAccountRequest) (<-chan *LockAccountResponse, <-chan error) {
	responseChan := make(chan *LockAccountResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.LockAccount(request)
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

// LockAccountWithCallback invokes the rds.LockAccount API asynchronously
func (client *Client) LockAccountWithCallback(request *LockAccountRequest, callback func(response *LockAccountResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *LockAccountResponse
		var err error
		defer close(result)
		response, err = client.LockAccount(request)
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

// LockAccountRequest is the request struct for api LockAccount
type LockAccountRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
	AccountName          string           `position:"Query" name:"AccountName"`
	DBInstanceId         string           `position:"Query" name:"DBInstanceId"`
}

// LockAccountResponse is the response struct for api LockAccount
type LockAccountResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateLockAccountRequest creates a request to invoke LockAccount API
func CreateLockAccountRequest() (request *LockAccountRequest) {
	request = &LockAccountRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Rds", "2014-08-15", "LockAccount", "rds", "openAPI")
	request.Method = requests.POST
	return
}

// CreateLockAccountResponse creates a response to parse from LockAccount response
func CreateLockAccountResponse() (response *LockAccountResponse) {
	response = &LockAccountResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
