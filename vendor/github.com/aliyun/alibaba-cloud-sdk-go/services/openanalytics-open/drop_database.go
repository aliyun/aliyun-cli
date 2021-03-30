package openanalytics_open

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

// DropDatabase invokes the openanalytics_open.DropDatabase API synchronously
func (client *Client) DropDatabase(request *DropDatabaseRequest) (response *DropDatabaseResponse, err error) {
	response = CreateDropDatabaseResponse()
	err = client.DoAction(request, response)
	return
}

// DropDatabaseWithChan invokes the openanalytics_open.DropDatabase API asynchronously
func (client *Client) DropDatabaseWithChan(request *DropDatabaseRequest) (<-chan *DropDatabaseResponse, <-chan error) {
	responseChan := make(chan *DropDatabaseResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DropDatabase(request)
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

// DropDatabaseWithCallback invokes the openanalytics_open.DropDatabase API asynchronously
func (client *Client) DropDatabaseWithCallback(request *DropDatabaseRequest, callback func(response *DropDatabaseResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DropDatabaseResponse
		var err error
		defer close(result)
		response, err = client.DropDatabase(request)
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

// DropDatabaseRequest is the request struct for api DropDatabase
type DropDatabaseRequest struct {
	*requests.RpcRequest
	Cascade requests.Boolean `position:"Query" name:"Cascade"`
	Name    string           `position:"Query" name:"Name"`
}

// DropDatabaseResponse is the response struct for api DropDatabase
type DropDatabaseResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Message   string `json:"Message" xml:"Message"`
	Data      string `json:"Data" xml:"Data"`
	Code      string `json:"Code" xml:"Code"`
	Success   bool   `json:"Success" xml:"Success"`
}

// CreateDropDatabaseRequest creates a request to invoke DropDatabase API
func CreateDropDatabaseRequest() (request *DropDatabaseRequest) {
	request = &DropDatabaseRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("openanalytics-open", "2020-09-28", "DropDatabase", "openanalytics", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDropDatabaseResponse creates a response to parse from DropDatabase response
func CreateDropDatabaseResponse() (response *DropDatabaseResponse) {
	response = &DropDatabaseResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
