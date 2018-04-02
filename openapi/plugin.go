package openapi

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"net/http"
)

type Plugin interface {
	BeforeInvoke(client *sdk.Client, request *requests.CommonRequest)
	AfterInvoke(client *sdk.Client, response *http.Response, err error)
}