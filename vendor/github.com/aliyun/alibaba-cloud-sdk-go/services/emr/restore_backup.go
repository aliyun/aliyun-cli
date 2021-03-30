package emr

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

// RestoreBackup invokes the emr.RestoreBackup API synchronously
func (client *Client) RestoreBackup(request *RestoreBackupRequest) (response *RestoreBackupResponse, err error) {
	response = CreateRestoreBackupResponse()
	err = client.DoAction(request, response)
	return
}

// RestoreBackupWithChan invokes the emr.RestoreBackup API asynchronously
func (client *Client) RestoreBackupWithChan(request *RestoreBackupRequest) (<-chan *RestoreBackupResponse, <-chan error) {
	responseChan := make(chan *RestoreBackupResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.RestoreBackup(request)
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

// RestoreBackupWithCallback invokes the emr.RestoreBackup API asynchronously
func (client *Client) RestoreBackupWithCallback(request *RestoreBackupRequest, callback func(response *RestoreBackupResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *RestoreBackupResponse
		var err error
		defer close(result)
		response, err = client.RestoreBackup(request)
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

// RestoreBackupRequest is the request struct for api RestoreBackup
type RestoreBackupRequest struct {
	*requests.RpcRequest
	ResourceOwnerId requests.Integer `position:"Query" name:"ResourceOwnerId"`
	BackupPlanId    string           `position:"Query" name:"BackupPlanId"`
	BackupId        string           `position:"Query" name:"BackupId"`
}

// RestoreBackupResponse is the response struct for api RestoreBackup
type RestoreBackupResponse struct {
	*responses.BaseResponse
	RequestId        string `json:"RequestId" xml:"RequestId"`
	BizId            string `json:"BizId" xml:"BizId"`
	DataSourceId     int64  `json:"DataSourceId" xml:"DataSourceId"`
	TaskType         string `json:"TaskType" xml:"TaskType"`
	TaskStatus       string `json:"TaskStatus" xml:"TaskStatus"`
	StartTime        int64  `json:"StartTime" xml:"StartTime"`
	EndTime          int64  `json:"EndTime" xml:"EndTime"`
	TaskDetail       string `json:"TaskDetail" xml:"TaskDetail"`
	TaskResultDetail string `json:"TaskResultDetail" xml:"TaskResultDetail"`
	TaskProcess      int    `json:"TaskProcess" xml:"TaskProcess"`
	TriggerUser      string `json:"TriggerUser" xml:"TriggerUser"`
	TriggerType      string `json:"TriggerType" xml:"TriggerType"`
	GmtCreate        int64  `json:"GmtCreate" xml:"GmtCreate"`
	GmtModified      int64  `json:"GmtModified" xml:"GmtModified"`
	ClusterBizId     string `json:"ClusterBizId" xml:"ClusterBizId"`
	HostName         string `json:"HostName" xml:"HostName"`
	EcmTaskId        int64  `json:"EcmTaskId" xml:"EcmTaskId"`
}

// CreateRestoreBackupRequest creates a request to invoke RestoreBackup API
func CreateRestoreBackupRequest() (request *RestoreBackupRequest) {
	request = &RestoreBackupRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Emr", "2016-04-08", "RestoreBackup", "emr", "openAPI")
	request.Method = requests.POST
	return
}

// CreateRestoreBackupResponse creates a response to parse from RestoreBackup response
func CreateRestoreBackupResponse() (response *RestoreBackupResponse) {
	response = &RestoreBackupResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
