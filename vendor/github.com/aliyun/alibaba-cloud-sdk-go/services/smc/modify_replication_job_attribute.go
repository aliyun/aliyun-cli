package smc

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

// ModifyReplicationJobAttribute invokes the smc.ModifyReplicationJobAttribute API synchronously
func (client *Client) ModifyReplicationJobAttribute(request *ModifyReplicationJobAttributeRequest) (response *ModifyReplicationJobAttributeResponse, err error) {
	response = CreateModifyReplicationJobAttributeResponse()
	err = client.DoAction(request, response)
	return
}

// ModifyReplicationJobAttributeWithChan invokes the smc.ModifyReplicationJobAttribute API asynchronously
func (client *Client) ModifyReplicationJobAttributeWithChan(request *ModifyReplicationJobAttributeRequest) (<-chan *ModifyReplicationJobAttributeResponse, <-chan error) {
	responseChan := make(chan *ModifyReplicationJobAttributeResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ModifyReplicationJobAttribute(request)
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

// ModifyReplicationJobAttributeWithCallback invokes the smc.ModifyReplicationJobAttribute API asynchronously
func (client *Client) ModifyReplicationJobAttributeWithCallback(request *ModifyReplicationJobAttributeRequest, callback func(response *ModifyReplicationJobAttributeResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ModifyReplicationJobAttributeResponse
		var err error
		defer close(result)
		response, err = client.ModifyReplicationJobAttribute(request)
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

// ModifyReplicationJobAttributeRequest is the request struct for api ModifyReplicationJobAttribute
type ModifyReplicationJobAttributeRequest struct {
	*requests.RpcRequest
	TargetType             string                                         `position:"Query" name:"TargetType"`
	Description            string                                         `position:"Query" name:"Description"`
	Frequency              requests.Integer                               `position:"Query" name:"Frequency"`
	JobId                  string                                         `position:"Query" name:"JobId"`
	ImageName              string                                         `position:"Query" name:"ImageName"`
	SystemDiskSize         requests.Integer                               `position:"Query" name:"SystemDiskSize"`
	InstanceType           string                                         `position:"Query" name:"InstanceType"`
	ContainerRepository    string                                         `position:"Query" name:"ContainerRepository"`
	ContainerTag           string                                         `position:"Query" name:"ContainerTag"`
	ContainerNamespace     string                                         `position:"Query" name:"ContainerNamespace"`
	LaunchTemplateId       string                                         `position:"Query" name:"LaunchTemplateId"`
	ResourceOwnerAccount   string                                         `position:"Query" name:"ResourceOwnerAccount"`
	SystemDiskPart         *[]ModifyReplicationJobAttributeSystemDiskPart `position:"Query" name:"SystemDiskPart"  type:"Repeated"`
	ValidTime              string                                         `position:"Query" name:"ValidTime"`
	OwnerId                requests.Integer                               `position:"Query" name:"OwnerId"`
	DataDisk               *[]ModifyReplicationJobAttributeDataDisk       `position:"Query" name:"DataDisk"  type:"Repeated"`
	LaunchTemplateVersion  string                                         `position:"Query" name:"LaunchTemplateVersion"`
	ScheduledStartTime     string                                         `position:"Query" name:"ScheduledStartTime"`
	InstanceId             string                                         `position:"Query" name:"InstanceId"`
	InstanceRamRole        string                                         `position:"Query" name:"InstanceRamRole"`
	Name                   string                                         `position:"Query" name:"Name"`
	MaxNumberOfImageToKeep requests.Integer                               `position:"Query" name:"MaxNumberOfImageToKeep"`
}

// ModifyReplicationJobAttributeSystemDiskPart is a repeated param struct in ModifyReplicationJobAttributeRequest
type ModifyReplicationJobAttributeSystemDiskPart struct {
	SizeBytes string `name:"SizeBytes"`
	Block     string `name:"Block"`
	Device    string `name:"Device"`
}

// ModifyReplicationJobAttributeDataDisk is a repeated param struct in ModifyReplicationJobAttributeRequest
type ModifyReplicationJobAttributeDataDisk struct {
	Size  string                               `name:"Size"`
	Part  *[]ModifyReplicationJobAttributePart `name:"Part" type:"Repeated"`
	Index string                               `name:"Index"`
}

// ModifyReplicationJobAttributePart is a repeated param struct in ModifyReplicationJobAttributeRequest
type ModifyReplicationJobAttributePart struct {
	SizeBytes string `name:"SizeBytes"`
	Block     string `name:"Block"`
	Device    string `name:"Device"`
}

// ModifyReplicationJobAttributeResponse is the response struct for api ModifyReplicationJobAttribute
type ModifyReplicationJobAttributeResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateModifyReplicationJobAttributeRequest creates a request to invoke ModifyReplicationJobAttribute API
func CreateModifyReplicationJobAttributeRequest() (request *ModifyReplicationJobAttributeRequest) {
	request = &ModifyReplicationJobAttributeRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("smc", "2019-06-01", "ModifyReplicationJobAttribute", "smc", "openAPI")
	request.Method = requests.POST
	return
}

// CreateModifyReplicationJobAttributeResponse creates a response to parse from ModifyReplicationJobAttribute response
func CreateModifyReplicationJobAttributeResponse() (response *ModifyReplicationJobAttributeResponse) {
	response = &ModifyReplicationJobAttributeResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
