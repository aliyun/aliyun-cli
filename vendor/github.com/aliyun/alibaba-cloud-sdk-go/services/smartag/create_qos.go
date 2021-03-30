package smartag

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

// CreateQos invokes the smartag.CreateQos API synchronously
func (client *Client) CreateQos(request *CreateQosRequest) (response *CreateQosResponse, err error) {
	response = CreateCreateQosResponse()
	err = client.DoAction(request, response)
	return
}

// CreateQosWithChan invokes the smartag.CreateQos API asynchronously
func (client *Client) CreateQosWithChan(request *CreateQosRequest) (<-chan *CreateQosResponse, <-chan error) {
	responseChan := make(chan *CreateQosResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CreateQos(request)
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

// CreateQosWithCallback invokes the smartag.CreateQos API asynchronously
func (client *Client) CreateQosWithCallback(request *CreateQosRequest, callback func(response *CreateQosResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CreateQosResponse
		var err error
		defer close(result)
		response, err = client.CreateQos(request)
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

// CreateQosRequest is the request struct for api CreateQos
type CreateQosRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
	QosName              string           `position:"Query" name:"QosName"`
	QosDescription       string           `position:"Query" name:"QosDescription"`
}

// CreateQosResponse is the response struct for api CreateQos
type CreateQosResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	QosId     string `json:"QosId" xml:"QosId"`
}

// CreateCreateQosRequest creates a request to invoke CreateQos API
func CreateCreateQosRequest() (request *CreateQosRequest) {
	request = &CreateQosRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Smartag", "2018-03-13", "CreateQos", "smartag", "openAPI")
	request.Method = requests.POST
	return
}

// CreateCreateQosResponse creates a response to parse from CreateQos response
func CreateCreateQosResponse() (response *CreateQosResponse) {
	response = &CreateQosResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
