package opensearch

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

// CreateSortScript invokes the opensearch.CreateSortScript API synchronously
func (client *Client) CreateSortScript(request *CreateSortScriptRequest) (response *CreateSortScriptResponse, err error) {
	response = CreateCreateSortScriptResponse()
	err = client.DoAction(request, response)
	return
}

// CreateSortScriptWithChan invokes the opensearch.CreateSortScript API asynchronously
func (client *Client) CreateSortScriptWithChan(request *CreateSortScriptRequest) (<-chan *CreateSortScriptResponse, <-chan error) {
	responseChan := make(chan *CreateSortScriptResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CreateSortScript(request)
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

// CreateSortScriptWithCallback invokes the opensearch.CreateSortScript API asynchronously
func (client *Client) CreateSortScriptWithCallback(request *CreateSortScriptRequest, callback func(response *CreateSortScriptResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CreateSortScriptResponse
		var err error
		defer close(result)
		response, err = client.CreateSortScript(request)
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

// CreateSortScriptRequest is the request struct for api CreateSortScript
type CreateSortScriptRequest struct {
	*requests.RoaRequest
	AppVersionId     string `position:"Path" name:"appVersionId"`
	AppGroupIdentity string `position:"Path" name:"appGroupIdentity"`
}

// CreateSortScriptResponse is the response struct for api CreateSortScript
type CreateSortScriptResponse struct {
	*responses.BaseResponse
	RequestId string `json:"requestId" xml:"requestId"`
}

// CreateCreateSortScriptRequest creates a request to invoke CreateSortScript API
func CreateCreateSortScriptRequest() (request *CreateSortScriptRequest) {
	request = &CreateSortScriptRequest{
		RoaRequest: &requests.RoaRequest{},
	}
	request.InitWithApiInfo("OpenSearch", "2017-12-25", "CreateSortScript", "/v4/openapi/app-groups/[appGroupIdentity]/apps/[appVersionId]/sort-scripts", "opensearch", "openAPI")
	request.Method = requests.POST
	return
}

// CreateCreateSortScriptResponse creates a response to parse from CreateSortScript response
func CreateCreateSortScriptResponse() (response *CreateSortScriptResponse) {
	response = &CreateSortScriptResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
