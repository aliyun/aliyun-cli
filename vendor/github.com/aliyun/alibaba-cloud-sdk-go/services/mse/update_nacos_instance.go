package mse

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

// UpdateNacosInstance invokes the mse.UpdateNacosInstance API synchronously
func (client *Client) UpdateNacosInstance(request *UpdateNacosInstanceRequest) (response *UpdateNacosInstanceResponse, err error) {
	response = CreateUpdateNacosInstanceResponse()
	err = client.DoAction(request, response)
	return
}

// UpdateNacosInstanceWithChan invokes the mse.UpdateNacosInstance API asynchronously
func (client *Client) UpdateNacosInstanceWithChan(request *UpdateNacosInstanceRequest) (<-chan *UpdateNacosInstanceResponse, <-chan error) {
	responseChan := make(chan *UpdateNacosInstanceResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.UpdateNacosInstance(request)
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

// UpdateNacosInstanceWithCallback invokes the mse.UpdateNacosInstance API asynchronously
func (client *Client) UpdateNacosInstanceWithCallback(request *UpdateNacosInstanceRequest, callback func(response *UpdateNacosInstanceResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *UpdateNacosInstanceResponse
		var err error
		defer close(result)
		response, err = client.UpdateNacosInstance(request)
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

// UpdateNacosInstanceRequest is the request struct for api UpdateNacosInstance
type UpdateNacosInstanceRequest struct {
	*requests.RpcRequest
	Metadata    string           `position:"Body" name:"Metadata"`
	ClusterName string           `position:"Query" name:"ClusterName"`
	Ip          string           `position:"Query" name:"Ip"`
	Ephemeral   requests.Boolean `position:"Query" name:"Ephemeral"`
	Weight      string           `position:"Query" name:"Weight"`
	GroupName   string           `position:"Query" name:"GroupName"`
	Enabled     requests.Boolean `position:"Query" name:"Enabled"`
	InstanceId  string           `position:"Query" name:"InstanceId"`
	NamespaceId string           `position:"Query" name:"NamespaceId"`
	Port        requests.Integer `position:"Query" name:"Port"`
	ServiceName string           `position:"Query" name:"ServiceName"`
}

// UpdateNacosInstanceResponse is the response struct for api UpdateNacosInstance
type UpdateNacosInstanceResponse struct {
	*responses.BaseResponse
	Message        string `json:"Message" xml:"Message"`
	RequestId      string `json:"RequestId" xml:"RequestId"`
	HttpStatusCode int    `json:"HttpStatusCode" xml:"HttpStatusCode"`
	Data           string `json:"Data" xml:"Data"`
	Code           int    `json:"Code" xml:"Code"`
	Success        bool   `json:"Success" xml:"Success"`
}

// CreateUpdateNacosInstanceRequest creates a request to invoke UpdateNacosInstance API
func CreateUpdateNacosInstanceRequest() (request *UpdateNacosInstanceRequest) {
	request = &UpdateNacosInstanceRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("mse", "2019-05-31", "UpdateNacosInstance", "", "")
	request.Method = requests.POST
	return
}

// CreateUpdateNacosInstanceResponse creates a response to parse from UpdateNacosInstance response
func CreateUpdateNacosInstanceResponse() (response *UpdateNacosInstanceResponse) {
	response = &UpdateNacosInstanceResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
