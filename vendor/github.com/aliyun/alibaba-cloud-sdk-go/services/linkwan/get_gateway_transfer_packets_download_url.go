package linkwan

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

// GetGatewayTransferPacketsDownloadUrl invokes the linkwan.GetGatewayTransferPacketsDownloadUrl API synchronously
func (client *Client) GetGatewayTransferPacketsDownloadUrl(request *GetGatewayTransferPacketsDownloadUrlRequest) (response *GetGatewayTransferPacketsDownloadUrlResponse, err error) {
	response = CreateGetGatewayTransferPacketsDownloadUrlResponse()
	err = client.DoAction(request, response)
	return
}

// GetGatewayTransferPacketsDownloadUrlWithChan invokes the linkwan.GetGatewayTransferPacketsDownloadUrl API asynchronously
func (client *Client) GetGatewayTransferPacketsDownloadUrlWithChan(request *GetGatewayTransferPacketsDownloadUrlRequest) (<-chan *GetGatewayTransferPacketsDownloadUrlResponse, <-chan error) {
	responseChan := make(chan *GetGatewayTransferPacketsDownloadUrlResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetGatewayTransferPacketsDownloadUrl(request)
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

// GetGatewayTransferPacketsDownloadUrlWithCallback invokes the linkwan.GetGatewayTransferPacketsDownloadUrl API asynchronously
func (client *Client) GetGatewayTransferPacketsDownloadUrlWithCallback(request *GetGatewayTransferPacketsDownloadUrlRequest, callback func(response *GetGatewayTransferPacketsDownloadUrlResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetGatewayTransferPacketsDownloadUrlResponse
		var err error
		defer close(result)
		response, err = client.GetGatewayTransferPacketsDownloadUrl(request)
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

// GetGatewayTransferPacketsDownloadUrlRequest is the request struct for api GetGatewayTransferPacketsDownloadUrl
type GetGatewayTransferPacketsDownloadUrlRequest struct {
	*requests.RpcRequest
	EndMillis     requests.Integer `position:"Query" name:"EndMillis"`
	IotInstanceId string           `position:"Query" name:"IotInstanceId"`
	GwEui         string           `position:"Query" name:"GwEui"`
	Ascending     requests.Boolean `position:"Query" name:"Ascending"`
	DevEui        string           `position:"Query" name:"DevEui"`
	ApiProduct    string           `position:"Body" name:"ApiProduct"`
	ApiRevision   string           `position:"Body" name:"ApiRevision"`
	Category      string           `position:"Query" name:"Category"`
	BeginMillis   requests.Integer `position:"Query" name:"BeginMillis"`
	SortingField  string           `position:"Query" name:"SortingField"`
}

// GetGatewayTransferPacketsDownloadUrlResponse is the response struct for api GetGatewayTransferPacketsDownloadUrl
type GetGatewayTransferPacketsDownloadUrlResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	Data      string `json:"Data" xml:"Data"`
}

// CreateGetGatewayTransferPacketsDownloadUrlRequest creates a request to invoke GetGatewayTransferPacketsDownloadUrl API
func CreateGetGatewayTransferPacketsDownloadUrlRequest() (request *GetGatewayTransferPacketsDownloadUrlRequest) {
	request = &GetGatewayTransferPacketsDownloadUrlRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("LinkWAN", "2019-03-01", "GetGatewayTransferPacketsDownloadUrl", "linkwan", "openAPI")
	request.Method = requests.POST
	return
}

// CreateGetGatewayTransferPacketsDownloadUrlResponse creates a response to parse from GetGatewayTransferPacketsDownloadUrl response
func CreateGetGatewayTransferPacketsDownloadUrlResponse() (response *GetGatewayTransferPacketsDownloadUrlResponse) {
	response = &GetGatewayTransferPacketsDownloadUrlResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
