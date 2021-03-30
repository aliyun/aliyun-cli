package edas

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

// UpdateK8sIngressRule invokes the edas.UpdateK8sIngressRule API synchronously
func (client *Client) UpdateK8sIngressRule(request *UpdateK8sIngressRuleRequest) (response *UpdateK8sIngressRuleResponse, err error) {
	response = CreateUpdateK8sIngressRuleResponse()
	err = client.DoAction(request, response)
	return
}

// UpdateK8sIngressRuleWithChan invokes the edas.UpdateK8sIngressRule API asynchronously
func (client *Client) UpdateK8sIngressRuleWithChan(request *UpdateK8sIngressRuleRequest) (<-chan *UpdateK8sIngressRuleResponse, <-chan error) {
	responseChan := make(chan *UpdateK8sIngressRuleResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.UpdateK8sIngressRule(request)
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

// UpdateK8sIngressRuleWithCallback invokes the edas.UpdateK8sIngressRule API asynchronously
func (client *Client) UpdateK8sIngressRuleWithCallback(request *UpdateK8sIngressRuleRequest, callback func(response *UpdateK8sIngressRuleResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *UpdateK8sIngressRuleResponse
		var err error
		defer close(result)
		response, err = client.UpdateK8sIngressRule(request)
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

// UpdateK8sIngressRuleRequest is the request struct for api UpdateK8sIngressRule
type UpdateK8sIngressRuleRequest struct {
	*requests.RoaRequest
	Namespace   string `position:"Query" name:"Namespace"`
	Name        string `position:"Query" name:"Name"`
	IngressConf string `position:"Query" name:"IngressConf"`
	Rules       string `position:"Query" name:"Rules"`
	ClusterId   string `position:"Query" name:"ClusterId"`
}

// UpdateK8sIngressRuleResponse is the response struct for api UpdateK8sIngressRule
type UpdateK8sIngressRuleResponse struct {
	*responses.BaseResponse
	Code           int                  `json:"Code" xml:"Code"`
	Message        string               `json:"Message" xml:"Message"`
	ChangeOrderIds []ChangeOrderIdsItem `json:"ChangeOrderIds" xml:"ChangeOrderIds"`
}

// CreateUpdateK8sIngressRuleRequest creates a request to invoke UpdateK8sIngressRule API
func CreateUpdateK8sIngressRuleRequest() (request *UpdateK8sIngressRuleRequest) {
	request = &UpdateK8sIngressRuleRequest{
		RoaRequest: &requests.RoaRequest{},
	}
	request.InitWithApiInfo("Edas", "2017-08-01", "UpdateK8sIngressRule", "/pop/v5/k8s/acs/k8s_ingress", "Edas", "openAPI")
	request.Method = requests.PUT
	return
}

// CreateUpdateK8sIngressRuleResponse creates a response to parse from UpdateK8sIngressRule response
func CreateUpdateK8sIngressRuleResponse() (response *UpdateK8sIngressRuleResponse) {
	response = &UpdateK8sIngressRuleResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
