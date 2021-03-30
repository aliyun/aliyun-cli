package vcs

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

// UpdateMonitor invokes the vcs.UpdateMonitor API synchronously
func (client *Client) UpdateMonitor(request *UpdateMonitorRequest) (response *UpdateMonitorResponse, err error) {
	response = CreateUpdateMonitorResponse()
	err = client.DoAction(request, response)
	return
}

// UpdateMonitorWithChan invokes the vcs.UpdateMonitor API asynchronously
func (client *Client) UpdateMonitorWithChan(request *UpdateMonitorRequest) (<-chan *UpdateMonitorResponse, <-chan error) {
	responseChan := make(chan *UpdateMonitorResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.UpdateMonitor(request)
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

// UpdateMonitorWithCallback invokes the vcs.UpdateMonitor API asynchronously
func (client *Client) UpdateMonitorWithCallback(request *UpdateMonitorRequest, callback func(response *UpdateMonitorResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *UpdateMonitorResponse
		var err error
		defer close(result)
		response, err = client.UpdateMonitor(request)
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

// UpdateMonitorRequest is the request struct for api UpdateMonitor
type UpdateMonitorRequest struct {
	*requests.RpcRequest
	CorpId               string           `position:"Body" name:"CorpId"`
	Description          string           `position:"Body" name:"Description"`
	RuleName             string           `position:"Body" name:"RuleName"`
	PicOperateType       string           `position:"Body" name:"PicOperateType"`
	AttributeName        string           `position:"Body" name:"AttributeName"`
	AttributeOperateType string           `position:"Body" name:"AttributeOperateType"`
	RuleExpression       string           `position:"Body" name:"RuleExpression"`
	NotifierTimeOut      requests.Integer `position:"Body" name:"NotifierTimeOut"`
	TaskId               string           `position:"Body" name:"TaskId"`
	DeviceOperateType    string           `position:"Body" name:"DeviceOperateType"`
	PicList              string           `position:"Body" name:"PicList"`
	AttributeValueList   string           `position:"Body" name:"AttributeValueList"`
	NotifierAppSecret    string           `position:"Body" name:"NotifierAppSecret"`
	NotifierExtendValues string           `position:"Body" name:"NotifierExtendValues"`
	DeviceList           string           `position:"Body" name:"DeviceList"`
	NotifierUrl          string           `position:"Body" name:"NotifierUrl"`
	NotifierType         string           `position:"Body" name:"NotifierType"`
	AlgorithmVendor      string           `position:"Body" name:"AlgorithmVendor"`
}

// UpdateMonitorResponse is the response struct for api UpdateMonitor
type UpdateMonitorResponse struct {
	*responses.BaseResponse
	Code      string `json:"Code" xml:"Code"`
	Data      string `json:"Data" xml:"Data"`
	Message   string `json:"Message" xml:"Message"`
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateUpdateMonitorRequest creates a request to invoke UpdateMonitor API
func CreateUpdateMonitorRequest() (request *UpdateMonitorRequest) {
	request = &UpdateMonitorRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Vcs", "2020-05-15", "UpdateMonitor", "", "")
	request.Method = requests.POST
	return
}

// CreateUpdateMonitorResponse creates a response to parse from UpdateMonitor response
func CreateUpdateMonitorResponse() (response *UpdateMonitorResponse) {
	response = &UpdateMonitorResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
