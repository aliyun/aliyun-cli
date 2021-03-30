package outboundbot

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

// ModifyDialogueFlow invokes the outboundbot.ModifyDialogueFlow API synchronously
func (client *Client) ModifyDialogueFlow(request *ModifyDialogueFlowRequest) (response *ModifyDialogueFlowResponse, err error) {
	response = CreateModifyDialogueFlowResponse()
	err = client.DoAction(request, response)
	return
}

// ModifyDialogueFlowWithChan invokes the outboundbot.ModifyDialogueFlow API asynchronously
func (client *Client) ModifyDialogueFlowWithChan(request *ModifyDialogueFlowRequest) (<-chan *ModifyDialogueFlowResponse, <-chan error) {
	responseChan := make(chan *ModifyDialogueFlowResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ModifyDialogueFlow(request)
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

// ModifyDialogueFlowWithCallback invokes the outboundbot.ModifyDialogueFlow API asynchronously
func (client *Client) ModifyDialogueFlowWithCallback(request *ModifyDialogueFlowRequest, callback func(response *ModifyDialogueFlowResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ModifyDialogueFlowResponse
		var err error
		defer close(result)
		response, err = client.ModifyDialogueFlow(request)
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

// ModifyDialogueFlowRequest is the request struct for api ModifyDialogueFlow
type ModifyDialogueFlowRequest struct {
	*requests.RpcRequest
	IsDrafted              requests.Boolean `position:"Query" name:"IsDrafted"`
	ScriptId               string           `position:"Query" name:"ScriptId"`
	InstanceId             string           `position:"Query" name:"InstanceId"`
	DialogueFlowDefinition string           `position:"Query" name:"DialogueFlowDefinition"`
	DialogueFlowId         string           `position:"Query" name:"DialogueFlowId"`
}

// ModifyDialogueFlowResponse is the response struct for api ModifyDialogueFlow
type ModifyDialogueFlowResponse struct {
	*responses.BaseResponse
	Code                   string `json:"Code" xml:"Code"`
	DialogueFlowDefinition string `json:"DialogueFlowDefinition" xml:"DialogueFlowDefinition"`
	DialogueFlowId         string `json:"DialogueFlowId" xml:"DialogueFlowId"`
	HttpStatusCode         int    `json:"HttpStatusCode" xml:"HttpStatusCode"`
	Message                string `json:"Message" xml:"Message"`
	RequestId              string `json:"RequestId" xml:"RequestId"`
	Success                bool   `json:"Success" xml:"Success"`
}

// CreateModifyDialogueFlowRequest creates a request to invoke ModifyDialogueFlow API
func CreateModifyDialogueFlowRequest() (request *ModifyDialogueFlowRequest) {
	request = &ModifyDialogueFlowRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("OutboundBot", "2019-12-26", "ModifyDialogueFlow", "outboundbot", "openAPI")
	request.Method = requests.POST
	return
}

// CreateModifyDialogueFlowResponse creates a response to parse from ModifyDialogueFlow response
func CreateModifyDialogueFlowResponse() (response *ModifyDialogueFlowResponse) {
	response = &ModifyDialogueFlowResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
