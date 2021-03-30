package opensearch

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

// ListInterventionDictionaryNerResults invokes the opensearch.ListInterventionDictionaryNerResults API synchronously
func (client *Client) ListInterventionDictionaryNerResults(request *ListInterventionDictionaryNerResultsRequest) (response *ListInterventionDictionaryNerResultsResponse, err error) {
	response = CreateListInterventionDictionaryNerResultsResponse()
	err = client.DoAction(request, response)
	return
}

// ListInterventionDictionaryNerResultsWithChan invokes the opensearch.ListInterventionDictionaryNerResults API asynchronously
func (client *Client) ListInterventionDictionaryNerResultsWithChan(request *ListInterventionDictionaryNerResultsRequest) (<-chan *ListInterventionDictionaryNerResultsResponse, <-chan error) {
	responseChan := make(chan *ListInterventionDictionaryNerResultsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ListInterventionDictionaryNerResults(request)
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

// ListInterventionDictionaryNerResultsWithCallback invokes the opensearch.ListInterventionDictionaryNerResults API asynchronously
func (client *Client) ListInterventionDictionaryNerResultsWithCallback(request *ListInterventionDictionaryNerResultsRequest, callback func(response *ListInterventionDictionaryNerResultsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ListInterventionDictionaryNerResultsResponse
		var err error
		defer close(result)
		response, err = client.ListInterventionDictionaryNerResults(request)
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

// ListInterventionDictionaryNerResultsRequest is the request struct for api ListInterventionDictionaryNerResults
type ListInterventionDictionaryNerResultsRequest struct {
	*requests.RoaRequest
	Query string `position:"Query" name:"query"`
	Name  string `position:"Path" name:"name"`
}

// ListInterventionDictionaryNerResultsResponse is the response struct for api ListInterventionDictionaryNerResults
type ListInterventionDictionaryNerResultsResponse struct {
	*responses.BaseResponse
	RequestId string    `json:"requestId" xml:"requestId"`
	Result    []NerItem `json:"result" xml:"result"`
}

// CreateListInterventionDictionaryNerResultsRequest creates a request to invoke ListInterventionDictionaryNerResults API
func CreateListInterventionDictionaryNerResultsRequest() (request *ListInterventionDictionaryNerResultsRequest) {
	request = &ListInterventionDictionaryNerResultsRequest{
		RoaRequest: &requests.RoaRequest{},
	}
	request.InitWithApiInfo("OpenSearch", "2017-12-25", "ListInterventionDictionaryNerResults", "/v4/openapi/intervention-dictionaries/[name]/ner-results", "opensearch", "openAPI")
	request.Method = requests.GET
	return
}

// CreateListInterventionDictionaryNerResultsResponse creates a response to parse from ListInterventionDictionaryNerResults response
func CreateListInterventionDictionaryNerResultsResponse() (response *ListInterventionDictionaryNerResultsResponse) {
	response = &ListInterventionDictionaryNerResultsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
