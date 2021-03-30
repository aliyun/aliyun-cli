package polardbx

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

// CreateDB invokes the polardbx.CreateDB API synchronously
func (client *Client) CreateDB(request *CreateDBRequest) (response *CreateDBResponse, err error) {
	response = CreateCreateDBResponse()
	err = client.DoAction(request, response)
	return
}

// CreateDBWithChan invokes the polardbx.CreateDB API asynchronously
func (client *Client) CreateDBWithChan(request *CreateDBRequest) (<-chan *CreateDBResponse, <-chan error) {
	responseChan := make(chan *CreateDBResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CreateDB(request)
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

// CreateDBWithCallback invokes the polardbx.CreateDB API asynchronously
func (client *Client) CreateDBWithCallback(request *CreateDBRequest, callback func(response *CreateDBResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CreateDBResponse
		var err error
		defer close(result)
		response, err = client.CreateDB(request)
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

// CreateDBRequest is the request struct for api CreateDB
type CreateDBRequest struct {
	*requests.RpcRequest
	DBInstanceName   string `position:"Query" name:"DBInstanceName"`
	Charset          string `position:"Query" name:"Charset"`
	AccountPrivilege string `position:"Query" name:"AccountPrivilege"`
	AccountName      string `position:"Query" name:"AccountName"`
	DbName           string `position:"Query" name:"DbName"`
	DbDescription    string `position:"Query" name:"DbDescription"`
}

// CreateDBResponse is the response struct for api CreateDB
type CreateDBResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	Message   string `json:"Message" xml:"Message"`
}

// CreateCreateDBRequest creates a request to invoke CreateDB API
func CreateCreateDBRequest() (request *CreateDBRequest) {
	request = &CreateDBRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("polardbx", "2020-02-02", "CreateDB", "polardbx", "openAPI")
	request.Method = requests.POST
	return
}

// CreateCreateDBResponse creates a response to parse from CreateDB response
func CreateCreateDBResponse() (response *CreateDBResponse) {
	response = &CreateDBResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
