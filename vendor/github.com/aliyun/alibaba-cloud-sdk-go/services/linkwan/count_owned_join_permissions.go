package linkwan

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

// CountOwnedJoinPermissions invokes the linkwan.CountOwnedJoinPermissions API synchronously
func (client *Client) CountOwnedJoinPermissions(request *CountOwnedJoinPermissionsRequest) (response *CountOwnedJoinPermissionsResponse, err error) {
	response = CreateCountOwnedJoinPermissionsResponse()
	err = client.DoAction(request, response)
	return
}

// CountOwnedJoinPermissionsWithChan invokes the linkwan.CountOwnedJoinPermissions API asynchronously
func (client *Client) CountOwnedJoinPermissionsWithChan(request *CountOwnedJoinPermissionsRequest) (<-chan *CountOwnedJoinPermissionsResponse, <-chan error) {
	responseChan := make(chan *CountOwnedJoinPermissionsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CountOwnedJoinPermissions(request)
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

// CountOwnedJoinPermissionsWithCallback invokes the linkwan.CountOwnedJoinPermissions API asynchronously
func (client *Client) CountOwnedJoinPermissionsWithCallback(request *CountOwnedJoinPermissionsRequest, callback func(response *CountOwnedJoinPermissionsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CountOwnedJoinPermissionsResponse
		var err error
		defer close(result)
		response, err = client.CountOwnedJoinPermissions(request)
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

// CountOwnedJoinPermissionsRequest is the request struct for api CountOwnedJoinPermissions
type CountOwnedJoinPermissionsRequest struct {
	*requests.RpcRequest
	Enabled                 requests.Boolean `position:"Query" name:"Enabled"`
	FuzzyJoinEui            string           `position:"Query" name:"FuzzyJoinEui"`
	FuzzyJoinPermissionName string           `position:"Query" name:"FuzzyJoinPermissionName"`
	FuzzyRenterAliyunId     string           `position:"Query" name:"FuzzyRenterAliyunId"`
	ApiProduct              string           `position:"Body" name:"ApiProduct"`
	ApiRevision             string           `position:"Body" name:"ApiRevision"`
}

// CountOwnedJoinPermissionsResponse is the response struct for api CountOwnedJoinPermissions
type CountOwnedJoinPermissionsResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	Data      int64  `json:"Data" xml:"Data"`
}

// CreateCountOwnedJoinPermissionsRequest creates a request to invoke CountOwnedJoinPermissions API
func CreateCountOwnedJoinPermissionsRequest() (request *CountOwnedJoinPermissionsRequest) {
	request = &CountOwnedJoinPermissionsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("LinkWAN", "2019-03-01", "CountOwnedJoinPermissions", "linkwan", "openAPI")
	request.Method = requests.POST
	return
}

// CreateCountOwnedJoinPermissionsResponse creates a response to parse from CountOwnedJoinPermissions response
func CreateCountOwnedJoinPermissionsResponse() (response *CountOwnedJoinPermissionsResponse) {
	response = &CountOwnedJoinPermissionsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
