package imm

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

// GetDRMLicense invokes the imm.GetDRMLicense API synchronously
func (client *Client) GetDRMLicense(request *GetDRMLicenseRequest) (response *GetDRMLicenseResponse, err error) {
	response = CreateGetDRMLicenseResponse()
	err = client.DoAction(request, response)
	return
}

// GetDRMLicenseWithChan invokes the imm.GetDRMLicense API asynchronously
func (client *Client) GetDRMLicenseWithChan(request *GetDRMLicenseRequest) (<-chan *GetDRMLicenseResponse, <-chan error) {
	responseChan := make(chan *GetDRMLicenseResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetDRMLicense(request)
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

// GetDRMLicenseWithCallback invokes the imm.GetDRMLicense API asynchronously
func (client *Client) GetDRMLicenseWithCallback(request *GetDRMLicenseRequest, callback func(response *GetDRMLicenseResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetDRMLicenseResponse
		var err error
		defer close(result)
		response, err = client.GetDRMLicense(request)
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

// GetDRMLicenseRequest is the request struct for api GetDRMLicense
type GetDRMLicenseRequest struct {
	*requests.RpcRequest
	Project    string `position:"Query" name:"Project"`
	DRMType    string `position:"Query" name:"DRMType"`
	DRMLicense string `position:"Query" name:"DRMLicense"`
}

// GetDRMLicenseResponse is the response struct for api GetDRMLicense
type GetDRMLicenseResponse struct {
	*responses.BaseResponse
	RequestId  string `json:"RequestId" xml:"RequestId"`
	DRMData    string `json:"DRMData" xml:"DRMData"`
	DeviceInfo string `json:"DeviceInfo" xml:"DeviceInfo"`
}

// CreateGetDRMLicenseRequest creates a request to invoke GetDRMLicense API
func CreateGetDRMLicenseRequest() (request *GetDRMLicenseRequest) {
	request = &GetDRMLicenseRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("imm", "2017-09-06", "GetDRMLicense", "imm", "openAPI")
	request.Method = requests.POST
	return
}

// CreateGetDRMLicenseResponse creates a response to parse from GetDRMLicense response
func CreateGetDRMLicenseResponse() (response *GetDRMLicenseResponse) {
	response = &GetDRMLicenseResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
