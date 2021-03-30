package sas

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

// CreateOrUpdateAssetGroup invokes the sas.CreateOrUpdateAssetGroup API synchronously
func (client *Client) CreateOrUpdateAssetGroup(request *CreateOrUpdateAssetGroupRequest) (response *CreateOrUpdateAssetGroupResponse, err error) {
	response = CreateCreateOrUpdateAssetGroupResponse()
	err = client.DoAction(request, response)
	return
}

// CreateOrUpdateAssetGroupWithChan invokes the sas.CreateOrUpdateAssetGroup API asynchronously
func (client *Client) CreateOrUpdateAssetGroupWithChan(request *CreateOrUpdateAssetGroupRequest) (<-chan *CreateOrUpdateAssetGroupResponse, <-chan error) {
	responseChan := make(chan *CreateOrUpdateAssetGroupResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CreateOrUpdateAssetGroup(request)
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

// CreateOrUpdateAssetGroupWithCallback invokes the sas.CreateOrUpdateAssetGroup API asynchronously
func (client *Client) CreateOrUpdateAssetGroupWithCallback(request *CreateOrUpdateAssetGroupRequest, callback func(response *CreateOrUpdateAssetGroupResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CreateOrUpdateAssetGroupResponse
		var err error
		defer close(result)
		response, err = client.CreateOrUpdateAssetGroup(request)
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

// CreateOrUpdateAssetGroupRequest is the request struct for api CreateOrUpdateAssetGroup
type CreateOrUpdateAssetGroupRequest struct {
	*requests.RpcRequest
	GroupId   requests.Integer `position:"Query" name:"GroupId"`
	GroupName string           `position:"Query" name:"GroupName"`
	SourceIp  string           `position:"Query" name:"SourceIp"`
	Uuids     string           `position:"Query" name:"Uuids"`
}

// CreateOrUpdateAssetGroupResponse is the response struct for api CreateOrUpdateAssetGroup
type CreateOrUpdateAssetGroupResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateCreateOrUpdateAssetGroupRequest creates a request to invoke CreateOrUpdateAssetGroup API
func CreateCreateOrUpdateAssetGroupRequest() (request *CreateOrUpdateAssetGroupRequest) {
	request = &CreateOrUpdateAssetGroupRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Sas", "2018-12-03", "CreateOrUpdateAssetGroup", "sas", "openAPI")
	request.Method = requests.POST
	return
}

// CreateCreateOrUpdateAssetGroupResponse creates a response to parse from CreateOrUpdateAssetGroup response
func CreateCreateOrUpdateAssetGroupResponse() (response *CreateOrUpdateAssetGroupResponse) {
	response = &CreateOrUpdateAssetGroupResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
