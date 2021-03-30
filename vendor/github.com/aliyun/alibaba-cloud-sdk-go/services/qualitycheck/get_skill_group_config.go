package qualitycheck

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

// GetSkillGroupConfig invokes the qualitycheck.GetSkillGroupConfig API synchronously
func (client *Client) GetSkillGroupConfig(request *GetSkillGroupConfigRequest) (response *GetSkillGroupConfigResponse, err error) {
	response = CreateGetSkillGroupConfigResponse()
	err = client.DoAction(request, response)
	return
}

// GetSkillGroupConfigWithChan invokes the qualitycheck.GetSkillGroupConfig API asynchronously
func (client *Client) GetSkillGroupConfigWithChan(request *GetSkillGroupConfigRequest) (<-chan *GetSkillGroupConfigResponse, <-chan error) {
	responseChan := make(chan *GetSkillGroupConfigResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetSkillGroupConfig(request)
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

// GetSkillGroupConfigWithCallback invokes the qualitycheck.GetSkillGroupConfig API asynchronously
func (client *Client) GetSkillGroupConfigWithCallback(request *GetSkillGroupConfigRequest, callback func(response *GetSkillGroupConfigResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetSkillGroupConfigResponse
		var err error
		defer close(result)
		response, err = client.GetSkillGroupConfig(request)
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

// GetSkillGroupConfigRequest is the request struct for api GetSkillGroupConfig
type GetSkillGroupConfigRequest struct {
	*requests.RpcRequest
	ResourceOwnerId requests.Integer `position:"Query" name:"ResourceOwnerId"`
	JsonStr         string           `position:"Query" name:"JsonStr"`
}

// GetSkillGroupConfigResponse is the response struct for api GetSkillGroupConfig
type GetSkillGroupConfigResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	Code      string `json:"Code" xml:"Code"`
	Message   string `json:"Message" xml:"Message"`
	Data      Data   `json:"Data" xml:"Data"`
}

// CreateGetSkillGroupConfigRequest creates a request to invoke GetSkillGroupConfig API
func CreateGetSkillGroupConfigRequest() (request *GetSkillGroupConfigRequest) {
	request = &GetSkillGroupConfigRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Qualitycheck", "2019-01-15", "GetSkillGroupConfig", "Qualitycheck", "openAPI")
	request.Method = requests.POST
	return
}

// CreateGetSkillGroupConfigResponse creates a response to parse from GetSkillGroupConfig response
func CreateGetSkillGroupConfigResponse() (response *GetSkillGroupConfigResponse) {
	response = &GetSkillGroupConfigResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
