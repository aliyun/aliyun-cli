package ehpc

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

// ApplyNodes invokes the ehpc.ApplyNodes API synchronously
func (client *Client) ApplyNodes(request *ApplyNodesRequest) (response *ApplyNodesResponse, err error) {
	response = CreateApplyNodesResponse()
	err = client.DoAction(request, response)
	return
}

// ApplyNodesWithChan invokes the ehpc.ApplyNodes API asynchronously
func (client *Client) ApplyNodesWithChan(request *ApplyNodesRequest) (<-chan *ApplyNodesResponse, <-chan error) {
	responseChan := make(chan *ApplyNodesResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ApplyNodes(request)
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

// ApplyNodesWithCallback invokes the ehpc.ApplyNodes API asynchronously
func (client *Client) ApplyNodesWithCallback(request *ApplyNodesRequest, callback func(response *ApplyNodesResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ApplyNodesResponse
		var err error
		defer close(result)
		response, err = client.ApplyNodes(request)
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

// ApplyNodesRequest is the request struct for api ApplyNodes
type ApplyNodesRequest struct {
	*requests.RpcRequest
	ImageId                       string                         `position:"Query" name:"ImageId"`
	Memory                        requests.Integer               `position:"Query" name:"Memory"`
	SystemDiskLevel               string                         `position:"Query" name:"SystemDiskLevel"`
	AllocatePublicAddress         requests.Boolean               `position:"Query" name:"AllocatePublicAddress"`
	InternetMaxBandWidthOut       requests.Integer               `position:"Query" name:"InternetMaxBandWidthOut"`
	ResourceAmountType            string                         `position:"Query" name:"ResourceAmountType"`
	SystemDiskType                string                         `position:"Query" name:"SystemDiskType"`
	Cores                         requests.Integer               `position:"Query" name:"Cores"`
	SystemDiskSize                requests.Integer               `position:"Query" name:"SystemDiskSize"`
	ZoneInfos                     *[]ApplyNodesZoneInfos         `position:"Query" name:"ZoneInfos"  type:"Repeated"`
	HostNamePrefix                string                         `position:"Query" name:"HostNamePrefix"`
	ComputeSpotPriceLimit         requests.Float                 `position:"Query" name:"ComputeSpotPriceLimit"`
	ClusterId                     string                         `position:"Query" name:"ClusterId"`
	ComputeSpotStrategy           string                         `position:"Query" name:"ComputeSpotStrategy"`
	HostNameSuffix                string                         `position:"Query" name:"HostNameSuffix"`
	PriorityStrategy              string                         `position:"Query" name:"PriorityStrategy"`
	InstanceFamilyLevel           string                         `position:"Query" name:"InstanceFamilyLevel"`
	InternetChargeType            string                         `position:"Query" name:"InternetChargeType"`
	InstanceTypeModel             *[]ApplyNodesInstanceTypeModel `position:"Query" name:"InstanceTypeModel"  type:"Repeated"`
	InternetMaxBandWidthIn        requests.Integer               `position:"Query" name:"InternetMaxBandWidthIn"`
	TargetCapacity                requests.Integer               `position:"Query" name:"TargetCapacity"`
	StrictSatisfiedTargetCapacity requests.Boolean               `position:"Query" name:"StrictSatisfiedTargetCapacity"`
}

// ApplyNodesZoneInfos is a repeated param struct in ApplyNodesRequest
type ApplyNodesZoneInfos struct {
	VSwitchId string `name:"VSwitchId"`
	ZoneId    string `name:"ZoneId"`
}

// ApplyNodesInstanceTypeModel is a repeated param struct in ApplyNodesRequest
type ApplyNodesInstanceTypeModel struct {
	MaxPrice      string `name:"MaxPrice"`
	TargetImageId string `name:"TargetImageId"`
	InstanceType  string `name:"InstanceType"`
}

// ApplyNodesResponse is the response struct for api ApplyNodes
type ApplyNodesResponse struct {
	*responses.BaseResponse
	RequestId       string                  `json:"RequestId" xml:"RequestId"`
	Detail          string                  `json:"Detail" xml:"Detail"`
	SatisfiedAmount int                     `json:"SatisfiedAmount" xml:"SatisfiedAmount"`
	TaskId          string                  `json:"TaskId" xml:"TaskId"`
	InstanceIds     InstanceIdsInApplyNodes `json:"InstanceIds" xml:"InstanceIds"`
}

// CreateApplyNodesRequest creates a request to invoke ApplyNodes API
func CreateApplyNodesRequest() (request *ApplyNodesRequest) {
	request = &ApplyNodesRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("EHPC", "2018-04-12", "ApplyNodes", "", "")
	request.Method = requests.GET
	return
}

// CreateApplyNodesResponse creates a response to parse from ApplyNodes response
func CreateApplyNodesResponse() (response *ApplyNodesResponse) {
	response = &ApplyNodesResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
