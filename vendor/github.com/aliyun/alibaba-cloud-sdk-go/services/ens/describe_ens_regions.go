package ens

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

// DescribeEnsRegions invokes the ens.DescribeEnsRegions API synchronously
func (client *Client) DescribeEnsRegions(request *DescribeEnsRegionsRequest) (response *DescribeEnsRegionsResponse, err error) {
	response = CreateDescribeEnsRegionsResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeEnsRegionsWithChan invokes the ens.DescribeEnsRegions API asynchronously
func (client *Client) DescribeEnsRegionsWithChan(request *DescribeEnsRegionsRequest) (<-chan *DescribeEnsRegionsResponse, <-chan error) {
	responseChan := make(chan *DescribeEnsRegionsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeEnsRegions(request)
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

// DescribeEnsRegionsWithCallback invokes the ens.DescribeEnsRegions API asynchronously
func (client *Client) DescribeEnsRegionsWithCallback(request *DescribeEnsRegionsRequest, callback func(response *DescribeEnsRegionsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeEnsRegionsResponse
		var err error
		defer close(result)
		response, err = client.DescribeEnsRegions(request)
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

// DescribeEnsRegionsRequest is the request struct for api DescribeEnsRegions
type DescribeEnsRegionsRequest struct {
	*requests.RpcRequest
	EnsRegionId string `position:"Query" name:"EnsRegionId"`
	Version     string `position:"Query" name:"Version"`
}

// DescribeEnsRegionsResponse is the response struct for api DescribeEnsRegions
type DescribeEnsRegionsResponse struct {
	*responses.BaseResponse
	RequestId  string     `json:"RequestId" xml:"RequestId"`
	Code       int        `json:"Code" xml:"Code"`
	EnsRegions EnsRegions `json:"EnsRegions" xml:"EnsRegions"`
}

// CreateDescribeEnsRegionsRequest creates a request to invoke DescribeEnsRegions API
func CreateDescribeEnsRegionsRequest() (request *DescribeEnsRegionsRequest) {
	request = &DescribeEnsRegionsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ens", "2017-11-10", "DescribeEnsRegions", "ens", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDescribeEnsRegionsResponse creates a response to parse from DescribeEnsRegions response
func CreateDescribeEnsRegionsResponse() (response *DescribeEnsRegionsResponse) {
	response = &DescribeEnsRegionsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
