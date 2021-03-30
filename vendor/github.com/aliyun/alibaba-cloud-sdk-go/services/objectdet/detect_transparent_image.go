package objectdet

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

// DetectTransparentImage invokes the objectdet.DetectTransparentImage API synchronously
func (client *Client) DetectTransparentImage(request *DetectTransparentImageRequest) (response *DetectTransparentImageResponse, err error) {
	response = CreateDetectTransparentImageResponse()
	err = client.DoAction(request, response)
	return
}

// DetectTransparentImageWithChan invokes the objectdet.DetectTransparentImage API asynchronously
func (client *Client) DetectTransparentImageWithChan(request *DetectTransparentImageRequest) (<-chan *DetectTransparentImageResponse, <-chan error) {
	responseChan := make(chan *DetectTransparentImageResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DetectTransparentImage(request)
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

// DetectTransparentImageWithCallback invokes the objectdet.DetectTransparentImage API asynchronously
func (client *Client) DetectTransparentImageWithCallback(request *DetectTransparentImageRequest, callback func(response *DetectTransparentImageResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DetectTransparentImageResponse
		var err error
		defer close(result)
		response, err = client.DetectTransparentImage(request)
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

// DetectTransparentImageRequest is the request struct for api DetectTransparentImage
type DetectTransparentImageRequest struct {
	*requests.RpcRequest
	ImageURL string `position:"Body" name:"ImageURL"`
}

// DetectTransparentImageResponse is the response struct for api DetectTransparentImage
type DetectTransparentImageResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Data      Data   `json:"Data" xml:"Data"`
}

// CreateDetectTransparentImageRequest creates a request to invoke DetectTransparentImage API
func CreateDetectTransparentImageRequest() (request *DetectTransparentImageRequest) {
	request = &DetectTransparentImageRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("objectdet", "2019-12-30", "DetectTransparentImage", "objectdet", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDetectTransparentImageResponse creates a response to parse from DetectTransparentImage response
func CreateDetectTransparentImageResponse() (response *DetectTransparentImageResponse) {
	response = &DetectTransparentImageResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
