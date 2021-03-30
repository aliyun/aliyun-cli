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

// UpdateGatewayFileShare invokes the sgw.UpdateGatewayFileShare API synchronously
func (client *Client) UpdateGatewayFileShare(request *UpdateGatewayFileShareRequest) (response *UpdateGatewayFileShareResponse, err error) {
	response = CreateUpdateGatewayFileShareResponse()
	err = client.DoAction(request, response)
	return
}

// UpdateGatewayFileShareWithChan invokes the sgw.UpdateGatewayFileShare API asynchronously
func (client *Client) UpdateGatewayFileShareWithChan(request *UpdateGatewayFileShareRequest) (<-chan *UpdateGatewayFileShareResponse, <-chan error) {
	responseChan := make(chan *UpdateGatewayFileShareResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.UpdateGatewayFileShare(request)
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

// UpdateGatewayFileShareWithCallback invokes the sgw.UpdateGatewayFileShare API asynchronously
func (client *Client) UpdateGatewayFileShareWithCallback(request *UpdateGatewayFileShareRequest, callback func(response *UpdateGatewayFileShareResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *UpdateGatewayFileShareResponse
		var err error
		defer close(result)
		response, err = client.UpdateGatewayFileShare(request)
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

// UpdateGatewayFileShareRequest is the request struct for api UpdateGatewayFileShare
type UpdateGatewayFileShareRequest struct {
	*requests.RpcRequest
	ClientSideCmk          string           `position:"Query" name:"ClientSideCmk"`
	InPlace                requests.Boolean `position:"Query" name:"InPlace"`
	Browsable              requests.Boolean `position:"Query" name:"Browsable"`
	ReadWriteUserList      string           `position:"Query" name:"ReadWriteUserList"`
	PollingInterval        requests.Integer `position:"Query" name:"PollingInterval"`
	ReadWriteClientList    string           `position:"Query" name:"ReadWriteClientList"`
	BackendLimit           requests.Integer `position:"Query" name:"BackendLimit"`
	Squash                 string           `position:"Query" name:"Squash"`
	ReadOnlyClientList     string           `position:"Query" name:"ReadOnlyClientList"`
	ServerSideCmk          string           `position:"Query" name:"ServerSideCmk"`
	SecurityToken          string           `position:"Query" name:"SecurityToken"`
	KmsRotatePeriod        requests.Integer `position:"Query" name:"KmsRotatePeriod"`
	RemoteSyncDownload     requests.Boolean `position:"Query" name:"RemoteSyncDownload"`
	ServerSideEncryption   requests.Boolean `position:"Query" name:"ServerSideEncryption"`
	NfsV4Optimization      requests.Boolean `position:"Query" name:"NfsV4Optimization"`
	AccessBasedEnumeration requests.Boolean `position:"Query" name:"AccessBasedEnumeration"`
	GatewayId              string           `position:"Query" name:"GatewayId"`
	IgnoreDelete           requests.Boolean `position:"Query" name:"IgnoreDelete"`
	LagPeriod              requests.Integer `position:"Query" name:"LagPeriod"`
	DirectIO               requests.Boolean `position:"Query" name:"DirectIO"`
	ClientSideEncryption   requests.Boolean `position:"Query" name:"ClientSideEncryption"`
	CacheMode              string           `position:"Query" name:"CacheMode"`
	DownloadLimit          requests.Integer `position:"Query" name:"DownloadLimit"`
	ReadOnlyUserList       string           `position:"Query" name:"ReadOnlyUserList"`
	FastReclaim            requests.Boolean `position:"Query" name:"FastReclaim"`
	WindowsAcl             requests.Boolean `position:"Query" name:"WindowsAcl"`
	Name                   string           `position:"Query" name:"Name"`
	IndexId                string           `position:"Query" name:"IndexId"`
	TransferAcceleration   requests.Boolean `position:"Query" name:"TransferAcceleration"`
	RemoteSync             requests.Boolean `position:"Query" name:"RemoteSync"`
	FrontendLimit          requests.Integer `position:"Query" name:"FrontendLimit"`
}

// UpdateGatewayFileShareResponse is the response struct for api UpdateGatewayFileShare
type UpdateGatewayFileShareResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
	Code      string `json:"Code" xml:"Code"`
	Message   string `json:"Message" xml:"Message"`
	TaskId    string `json:"TaskId" xml:"TaskId"`
}

// CreateUpdateGatewayFileShareRequest creates a request to invoke UpdateGatewayFileShare API
func CreateUpdateGatewayFileShareRequest() (request *UpdateGatewayFileShareRequest) {
	request = &UpdateGatewayFileShareRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("sgw", "2018-05-11", "UpdateGatewayFileShare", "hcs_sgw", "openAPI")
	request.Method = requests.POST
	return
}

// CreateUpdateGatewayFileShareResponse creates a response to parse from UpdateGatewayFileShare response
func CreateUpdateGatewayFileShareResponse() (response *UpdateGatewayFileShareResponse) {
	response = &UpdateGatewayFileShareResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
