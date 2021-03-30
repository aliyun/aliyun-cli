package sgw

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

// CheckUpgradeVersion invokes the sgw.CheckUpgradeVersion API synchronously
func (client *Client) CheckUpgradeVersion(request *CheckUpgradeVersionRequest) (response *CheckUpgradeVersionResponse, err error) {
	response = CreateCheckUpgradeVersionResponse()
	err = client.DoAction(request, response)
	return
}

// CheckUpgradeVersionWithChan invokes the sgw.CheckUpgradeVersion API asynchronously
func (client *Client) CheckUpgradeVersionWithChan(request *CheckUpgradeVersionRequest) (<-chan *CheckUpgradeVersionResponse, <-chan error) {
	responseChan := make(chan *CheckUpgradeVersionResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.CheckUpgradeVersion(request)
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

// CheckUpgradeVersionWithCallback invokes the sgw.CheckUpgradeVersion API asynchronously
func (client *Client) CheckUpgradeVersionWithCallback(request *CheckUpgradeVersionRequest, callback func(response *CheckUpgradeVersionResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *CheckUpgradeVersionResponse
		var err error
		defer close(result)
		response, err = client.CheckUpgradeVersion(request)
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

// CheckUpgradeVersionRequest is the request struct for api CheckUpgradeVersion
type CheckUpgradeVersionRequest struct {
	*requests.RpcRequest
	ClientUUID     string `position:"Query" name:"ClientUUID"`
	GatewayVersion string `position:"Query" name:"GatewayVersion"`
	SecurityToken  string `position:"Query" name:"SecurityToken"`
	GatewayId      string `position:"Query" name:"GatewayId"`
}

// CheckUpgradeVersionResponse is the response struct for api CheckUpgradeVersion
type CheckUpgradeVersionResponse struct {
	*responses.BaseResponse
	RequestId     string  `json:"RequestId" xml:"RequestId"`
	Success       bool    `json:"Success" xml:"Success"`
	Code          string  `json:"Code" xml:"Code"`
	Message       string  `json:"Message" xml:"Message"`
	Option        string  `json:"Option" xml:"Option"`
	LatestVersion string  `json:"LatestVersion" xml:"LatestVersion"`
	Patches       Patches `json:"Patches" xml:"Patches"`
}

// CreateCheckUpgradeVersionRequest creates a request to invoke CheckUpgradeVersion API
func CreateCheckUpgradeVersionRequest() (request *CheckUpgradeVersionRequest) {
	request = &CheckUpgradeVersionRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("sgw", "2018-05-11", "CheckUpgradeVersion", "hcs_sgw", "openAPI")
	request.Method = requests.POST
	return
}

// CreateCheckUpgradeVersionResponse creates a response to parse from CheckUpgradeVersion response
func CreateCheckUpgradeVersionResponse() (response *CheckUpgradeVersionResponse) {
	response = &CheckUpgradeVersionResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
